import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { adminApi, serversApi } from "../lib/endpoints";
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
import { formatBytes, formatCpu } from "../lib/format";
import type { ContainerHeartbeat } from "../types/api";

interface ConsoleLine {
  server_id: string;
  kind: string;
  message: string;
}

const TABS = ["Console", "Files", "Backups", "Schedules", "Activity", "Settings", "Sharing"] as const;
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
    // While provisioning (async — may pull an image), poll so the page moves
    // off "installing" to running/errored on its own instead of looking stuck.
    refetchInterval: (query) => {
      const s = query.state.data?.status;
      return s === "installing" || s === "stopping" ? 3000 : false;
    },
  });

  const isAdmin = user?.role === "admin";
  const canManage = !!server && !!user && (server.owner_id === user.id || isAdmin);
  const manageOnly: Tab[] = ["Schedules", "Settings", "Sharing"];
  const visibleTabs = TABS.filter((t) => !manageOnly.includes(t) || canManage);
  const [tab, setTab] = useState<Tab>("Console");

  const [lines, setLines] = useState<string[]>([]);
  const [stats, setStats] = useState<ContainerHeartbeat | null>(null);

  useTopic<ConsoleLine>(id ? `server:${id}:console` : null, (msg) => {
    setLines((prev) => [...prev, msg.message]);
  });
  useTopic<ContainerHeartbeat>(id ? `server:${id}:stats` : null, (msg) => setStats(msg));

  const power = useMutation({
    mutationFn: (action: "start" | "stop" | "kill") => serversApi.power(id!, action),
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
  const errored = server.status === "errored";

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
          <div className="sp-spec-strip">
            <span className="sp-spec">
              <span className="sp-spec__k">port</span>
              {server.primary_port}
            </span>
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
          </div>
        </div>
        <div style={{ display: "flex", gap: 8, flexWrap: "wrap", justifyContent: "flex-end" }}>
          <button
            className="sp-btn"
            onClick={() => power.mutate("start")}
            disabled={(server.suspended && !isAdmin) || installing}
            title={server.suspended && !isAdmin ? "This server is suspended by an administrator" : undefined}
          >
            Start
          </button>
          <button className="sp-btn" onClick={() => power.mutate("stop")}>
            Stop
          </button>
          <button className="sp-btn sp-btn--danger" onClick={() => power.mutate("kill")}>
            Kill
          </button>
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
              className="sp-btn sp-btn--danger"
              onClick={() => navigate(`/servers/${id}/reinstall`)}
              disabled={installing}
            >
              {installing ? "Installing…" : "Reinstall"}
            </button>
          )}
          <button className="sp-btn sp-btn--danger" onClick={() => remove.mutate()}>
            Delete
          </button>
        </div>
      </div>

      {installing && (
        <div className="sp-surface sp-card sp-banner" style={{ marginBottom: 16 }}>
          <span className="sp-spinner" />
          <div>
            <strong>Provisioning…</strong>{" "}
            <span style={{ color: "var(--sp-text-muted)" }}>
              the node is creating this server. A first launch pulls the Docker image and can take a few minutes — this
              page updates on its own.
            </span>
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
        <StatCard label="CPU" value={stats ? `${stats.cpu_percent.toFixed(1)}%` : "—"} />
        <StatCard label="Memory" value={stats ? `${(stats.mem_used_bytes / 1024 / 1024).toFixed(0)}MB` : "—"} />
        <StatCard label="Net RX" value={stats ? `${(stats.net_rx_bytes / 1024).toFixed(1)}KB` : "—"} />
        <StatCard label="Net TX" value={stats ? `${(stats.net_tx_bytes / 1024).toFixed(1)}KB` : "—"} />
      </div>

      <div style={{ display: "flex", gap: 6, marginBottom: 16 }}>
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
        <div className="sp-surface" style={{ height: 420, padding: 12 }}>
          <Console lines={lines} onInput={sendInput} />
        </div>
      )}
      {tab === "Files" && <FilesTab serverId={id!} />}
      {tab === "Backups" && <BackupsTab serverId={id!} />}
      {tab === "Schedules" && canManage && <SchedulesTab serverId={id!} />}
      {tab === "Activity" && <ActivityTab serverId={id!} />}
      {tab === "Settings" && canManage && <SettingsTab server={server} />}
      {tab === "Sharing" && canManage && <SharingTab serverId={id!} />}
    </div>
  );
}

function StatCard({ label, value }: { label: string; value: string }) {
  return (
    <div className="sp-surface sp-card">
      <p className="sp-stat__label">{label}</p>
      <p className="sp-mono" style={{ fontSize: 26, fontVariantNumeric: "tabular-nums" }}>
        {value}
      </p>
    </div>
  );
}
