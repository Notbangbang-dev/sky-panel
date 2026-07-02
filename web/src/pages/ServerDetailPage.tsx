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
import type { ContainerHeartbeat } from "../types/api";

interface ConsoleLine {
  server_id: string;
  kind: string;
  message: string;
}

const TABS = ["Console", "Files", "Backups", "Activity", "Settings", "Sharing"] as const;
type Tab = (typeof TABS)[number];

export function ServerDetailPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const user = useAuthStore((s) => s.user);

  const { data: server } = useQuery({ queryKey: ["servers", id], queryFn: () => serversApi.get(id!), enabled: !!id });

  const isAdmin = user?.role === "admin";
  const canManage = !!server && !!user && (server.owner_id === user.id || isAdmin);
  const visibleTabs = TABS.filter((t) => (t !== "Sharing" && t !== "Settings") || canManage);
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

  const reinstall = useMutation({
    mutationFn: () => serversApi.reinstall(id!),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["servers", id] }),
  });

  const suspend = useMutation({
    mutationFn: (s: boolean) => (s ? adminApi.suspendServer(id!) : adminApi.unsuspendServer(id!)),
    onSuccess: () => queryClient.invalidateQueries({ queryKey: ["servers", id] }),
  });

  const sendInput = (line: string) => serversApi.consoleInput(id!, line).catch(() => {});

  if (!server) return <p className="sp-mono">loading…</p>;

  return (
    <div>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 18 }}>
        <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
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
        <div style={{ display: "flex", gap: 8 }}>
          <button
            className="sp-btn"
            onClick={() => power.mutate("start")}
            disabled={server.suspended && !isAdmin}
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
              onClick={() => {
                if (window.confirm("Reinstall this server? The container is rebuilt from its egg. Your files are kept.")) {
                  reinstall.mutate();
                }
              }}
              disabled={reinstall.isPending}
            >
              {reinstall.isPending ? "Reinstalling…" : "Reinstall"}
            </button>
          )}
          <button className="sp-btn sp-btn--danger" onClick={() => remove.mutate()}>
            Delete
          </button>
        </div>
      </div>

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
