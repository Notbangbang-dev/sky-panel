import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi, nodesApi, serversApi } from "../lib/endpoints";
import { useTopic } from "../lib/useTopic";
import { useAuthStore } from "../lib/authStore";
import { Console } from "../components/Console";
import { StatusBadge } from "../components/StatusBadge";
import { FilesTab } from "../components/server/FilesTab";
import { SharingTab } from "../components/server/SharingTab";
import { SettingsTab } from "../components/server/SettingsTab";
import { ActivityTab } from "../components/server/ActivityTab";
import { BackupsTab } from "../components/server/BackupsTab";
import { SchedulesTab } from "../components/server/SchedulesTab";
import { ModrinthTab } from "../components/server/ModrinthTab";
import { formatBytes, formatCpu } from "../lib/format";
import { copyText } from "../lib/clipboard";
import type { ContainerHeartbeat } from "../types/api";

interface ConsoleLine {
  server_id: string;
  kind: string;
  message: string;
}

const TABS = ["Console", "Files", "Mods", "Backups", "Schedules", "Activity", "Settings", "Sharing"] as const;
type Tab = (typeof TABS)[number];

export function ServerDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const user = useAuthStore((s) => s.user);

  const { data: server } = useQuery({
    queryKey: ["servers", id],
    queryFn: () => serversApi.get(id!),
    enabled: !!id,
    refetchInterval: (query) => {
      const s = query.state.data?.status;
      return s === "installing" || s === "stopping" ? 3000 : false;
    },
  });

  // The connect address is the owning node's address + the server's port.
  const { data: nodes } = useQuery({ queryKey: ["nodes"], queryFn: nodesApi.list });

  const isAdmin = user?.role === "admin";
  const canManage = !!server && !!user && (server.owner_id === user.id || isAdmin);
  const manageOnly: Tab[] = ["Schedules", "Settings", "Sharing"];
  const visibleTabs = TABS.filter((t) => !manageOnly.includes(t) || canManage);
  const [tab, setTab] = useState<Tab>("Console");

  const [lines, setLines] = useState<string[]>([]);
  const [stats, setStats] = useState<ContainerHeartbeat | null>(null);
  // Rolling window of recent CPU% / memory% samples for the live sparklines.
  const [history, setHistory] = useState<{ cpu: number; mem: number }[]>([]);
  const [copied, setCopied] = useState(false);

  useTopic<ConsoleLine>(id ? `server:${id}:console` : null, (msg) => {
    setLines((prev) => [...prev, msg.message]);
  });
  useTopic<ContainerHeartbeat>(id ? `server:${id}:stats` : null, (msg) => {
    setStats(msg);
    setHistory((prev) => {
      const memPct = msg.mem_limit_bytes > 0 ? (msg.mem_used_bytes / msg.mem_limit_bytes) * 100 : 0;
      const next = [...prev, { cpu: msg.cpu_percent, mem: memPct }];
      return next.length > 60 ? next.slice(next.length - 60) : next;
    });
  });

  const power = useMutation({
    mutationFn: (action: "start" | "stop" | "kill") => serversApi.power(id!, action),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["servers", id] }),
  });

  const restart = useMutation({
    mutationFn: async () => {
      await serversApi.power(id!, "stop").catch(() => {});
      await serversApi.power(id!, "start");
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["servers", id] }),
  });

  const remove = useMutation({
    mutationFn: () => serversApi.remove(id!),
    onSuccess: () => navigate("/servers"),
  });

  const suspend = useMutation({
    mutationFn: (s: boolean) => (s ? adminApi.suspendServer(id!) : adminApi.unsuspendServer(id!)),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["servers", id] }),
  });

  const sendInput = (line: string) => serversApi.consoleInput(id!, line).catch(() => {});

  if (!server) return <p className="sp-mono">loading…</p>;

  const installing = server.status === "installing";
  const stopping = server.status === "stopping";
  const errored = server.status === "errored";
  const running = server.status === "running";
  const busy = installing || stopping;
  // Running per the panel, but no heartbeat has arrived yet — show a pending
  // marker rather than a dead dash so it's clear we're waiting on the node.
  const waitingForStats = running && !stats;

  const node = nodes?.find((n) => n.id === server.node_id);
  const address = node?.address ? `${node.address}:${server.primary_port}` : `:${server.primary_port}`;

  const copyAddress = () => {
    copyText(address).then((ok) => {
      if (ok) {
        setCopied(true);
        setTimeout(() => setCopied(false), 1500);
      }
    });
  };

  const startBlocked = running || busy || (server.suspended && !isAdmin);

  return (
    <div>
      <div className="sp-detail-head">
        <div>
          <p className="sp-kicker">Server</p>
          <div style={{ display: "flex", alignItems: "center", gap: 12, flexWrap: "wrap" }}>
            <h1 className="sp-page-title" style={{ marginBottom: 0 }}>
              {server.name}
            </h1>
            <StatusBadge status={server.status} />
            {server.suspended && (
              <span className="sp-badge" style={{ color: "#ff9b9b", borderColor: "#ff9b9b" }}>
                suspended
              </span>
            )}
          </div>

          <div className="sp-spec-strip" style={{ alignItems: "center" }}>
            <button className="sp-conn" onClick={copyAddress} title="Copy connect address">
              <span className="sp-conn__label">connect</span>
              <span className="sp-conn__addr">{address}</span>
              <span className="sp-conn__copy">{copied ? "copied" : "copy"}</span>
            </button>
            <span className="sp-spec">
              <span className="sp-spec__k">ram</span>
              {formatBytes(server.memory_bytes)}
            </span>
            <span className="sp-spec">
              <span className="sp-spec__k">cpu</span>
              {formatCpu(server.cpu_limit)}
            </span>
            <span className="sp-spec">
              <span className="sp-spec__k">disk</span>
              {server.disk_bytes ? formatBytes(server.disk_bytes) : "—"}
            </span>
            {node && (
              <span className="sp-spec">
                <span className="sp-spec__k">node</span>
                {node.name}
              </span>
            )}
          </div>

          {server.description && (
            <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)", marginTop: 8, maxWidth: 620 }}>
              {server.description}
            </p>
          )}
        </div>

        <div className="sp-ctl-bar">
          <button
            className="sp-btn sp-btn--primary"
            onClick={() => power.mutate("start")}
            disabled={startBlocked || power.isPending}
            title={server.suspended && !isAdmin ? "This server is suspended by an administrator" : undefined}
          >
            ▶ Start
          </button>
          <button
            className="sp-btn"
            onClick={() => restart.mutate()}
            disabled={!running || restart.isPending}
          >
            ↻ Restart
          </button>
          <button className="sp-btn" onClick={() => power.mutate("stop")} disabled={!running}>
            ■ Stop
          </button>
          <button className="sp-btn sp-btn--danger" onClick={() => power.mutate("kill")} disabled={!running}>
            ✕ Kill
          </button>

          <span className="sp-ctl-sep" />

          {isAdmin && (
            <button
              className="sp-btn sp-btn--danger"
              onClick={() => suspend.mutate(!server.suspended)}
              disabled={suspend.isPending}
            >
              {server.suspended ? "Unsuspend" : "Suspend"}
            </button>
          )}
          {canManage && (
            <button
              className="sp-btn"
              onClick={() => navigate(`/servers/${id}/reinstall`)}
              disabled={installing}
            >
              {installing ? "Installing…" : "⟳ Reinstall"}
            </button>
          )}
          <button
            className="sp-btn sp-btn--danger"
            onClick={() => {
              if (window.confirm(`Delete “${server.name}”? This removes the container and frees its port. Files are lost.`)) {
                remove.mutate();
              }
            }}
          >
            🗑 Delete
          </button>
        </div>
      </div>

      {installing && (
        <div className="sp-surface sp-card sp-banner" style={{ marginBottom: 16 }}>
          <span className="sp-spinner" />
          <div>
            <strong>Provisioning…</strong>{" "}
            <span style={{ color: "var(--sp-text-muted)" }}>
              the node is bringing this server online — this page updates on its own. If the image is already cached it's
              a matter of seconds; a cold node pulls it once, then it's fast forever.
            </span>
            {server.status_message && (
              <div className="sp-mono" style={{ fontSize: 12, marginTop: 4, color: "var(--sp-accent)" }}>
                {server.status_message}
              </div>
            )}
          </div>
        </div>
      )}
      {errored && (
        <div className="sp-surface sp-card sp-banner sp-banner--error" style={{ marginBottom: 16 }}>
          <div>
            <strong>Provisioning failed.</strong>{" "}
            <span className="sp-mono" style={{ fontSize: 12 }}>
              {server.status_message || "The node reported an error. Check the node's Docker/logs."}
            </span>
            {canManage && (
              <div style={{ marginTop: 4, color: "var(--sp-text-muted)", fontSize: 12 }}>
                Fix the cause on the node, then use <strong>Reinstall</strong> to retry.
              </div>
            )}
          </div>
        </div>
      )}

      <div className="sp-grid sp-grid--cards" style={{ marginBottom: 16 }}>
        <StatCard
          label="CPU"
          value={stats ? `${stats.cpu_percent.toFixed(1)}%` : waitingForStats ? "···" : "—"}
          pct={stats && server.cpu_limit > 0 ? (stats.cpu_percent / server.cpu_limit) * 100 : stats?.cpu_percent}
          series={history.map((h) => h.cpu)}
        />
        <StatCard
          label="Memory"
          value={stats ? `${(stats.mem_used_bytes / 1024 / 1024).toFixed(0)}MB` : waitingForStats ? "···" : "—"}
          sub={stats ? `of ${formatBytes(stats.mem_limit_bytes || server.memory_bytes)}` : undefined}
          pct={stats && stats.mem_limit_bytes > 0 ? (stats.mem_used_bytes / stats.mem_limit_bytes) * 100 : undefined}
          series={history.map((h) => h.mem)}
        />
        <StatCard label="Net RX" value={stats ? `${(stats.net_rx_bytes / 1024).toFixed(1)}KB` : waitingForStats ? "···" : "—"} />
        <StatCard label="Net TX" value={stats ? `${(stats.net_tx_bytes / 1024).toFixed(1)}KB` : waitingForStats ? "···" : "—"} />
      </div>

      <div style={{ display: "flex", gap: 6, marginBottom: 16, flexWrap: "wrap" }}>
        {visibleTabs.map((t) => (
          <button
            key={t}
            className="sp-btn sp-btn--sm"
            style={t === tab ? { background: "var(--sp-accent)", color: "var(--sp-accent-text)" } : undefined}
            onClick={() => setTab(t)}
          >
            {t}
          </button>
        ))}
      </div>

      {tab === "Console" && (
        <div className="sp-console-frame">
          <div className="sp-console-bar">
            <span className="sp-console-bar__title">
              <span className={"sp-dot" + (running ? " sp-dot--live" : "")} />
              Console {running ? "· live" : `· ${server.status}`}
            </span>
            <button className="sp-btn sp-btn--sm" onClick={() => setLines([])}>
              Clear
            </button>
          </div>
          <div style={{ height: 420, padding: 12 }}>
            <Console lines={lines} onInput={sendInput} />
          </div>
        </div>
      )}
      {tab === "Files" && <FilesTab serverId={id!} />}
      {tab === "Mods" && <ModrinthTab serverId={id!} />}
      {tab === "Backups" && <BackupsTab serverId={id!} />}
      {tab === "Schedules" && canManage && <SchedulesTab serverId={id!} />}
      {tab === "Activity" && <ActivityTab serverId={id!} />}
      {tab === "Settings" && canManage && <SettingsTab server={server} />}
      {tab === "Sharing" && canManage && <SharingTab serverId={id!} />}
    </div>
  );
}

function StatCard({
  label,
  value,
  sub,
  pct,
  series,
}: {
  label: string;
  value: string;
  sub?: string;
  pct?: number;
  series?: number[];
}) {
  const clamped = pct === undefined ? undefined : Math.max(0, Math.min(100, pct));
  return (
    <div className="sp-surface sp-card">
      <p className="sp-stat__label">{label}</p>
      <p className="sp-mono" style={{ fontSize: 26, fontVariantNumeric: "tabular-nums", margin: 0 }}>
        {value}
      </p>
      {sub && (
        <p className="sp-mono" style={{ fontSize: 11, color: "var(--sp-text-muted)", margin: "2px 0 0" }}>
          {sub}
        </p>
      )}
      {series && series.length >= 2 && <Sparkline series={series} warn={clamped !== undefined && clamped >= 90} />}
      {clamped !== undefined && (
        <div className="sp-gauge">
          <div className={"sp-gauge__fill" + (clamped >= 90 ? " sp-gauge__fill--warn" : "")} style={{ width: `${clamped}%` }} />
        </div>
      )}
    </div>
  );
}

// Sparkline renders a compact history line for a metric, auto-scaled to the
// series' own peak so a low-but-varying signal is still readable. Purely an
// SVG polyline + a soft area fill under it — no chart library.
function Sparkline({ series, warn }: { series: number[]; warn?: boolean }) {
  const w = 120;
  const h = 30;
  const max = Math.max(1, ...series);
  const step = series.length > 1 ? w / (series.length - 1) : w;
  const pts = series.map((v, i) => [i * step, h - (v / max) * (h - 2) - 1] as const);
  const line = pts.map(([x, y]) => `${x.toFixed(1)},${y.toFixed(1)}`).join(" ");
  const area = `0,${h} ${line} ${w},${h}`;
  const stroke = warn ? "var(--sp-gauge-warn, #ff9b9b)" : "var(--sp-accent)";
  return (
    <svg
      className="sp-spark"
      viewBox={`0 0 ${w} ${h}`}
      preserveAspectRatio="none"
      style={{ width: "100%", height: 30, marginTop: 10, display: "block" }}
      aria-hidden
    >
      <polygon points={area} fill={stroke} opacity={0.12} />
      <polyline points={line} fill="none" stroke={stroke} strokeWidth={1.5} strokeLinejoin="round" strokeLinecap="round" />
    </svg>
  );
}
