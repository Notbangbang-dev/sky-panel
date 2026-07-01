import { useQuery } from "@tanstack/react-query";
import { Link } from "react-router-dom";
import { serversApi } from "../lib/endpoints";
import { useAuthStore } from "../lib/authStore";
import { StatusBadge } from "../components/StatusBadge";

export function DashboardPage() {
  const user = useAuthStore((s) => s.user);
  const { data: servers, isLoading } = useQuery({ queryKey: ["servers"], queryFn: serversApi.list });

  const running = servers?.filter((s) => s.status === "running").length ?? 0;

  return (
    <div>
      <p className="sp-kicker">Overview</p>
      <h1 className="sp-page-title">Welcome back, {user?.username}.</h1>

      <div className="sp-grid sp-grid--cards" style={{ marginBottom: 24 }}>
        <div className="sp-surface sp-card">
          <p className="sp-stat__label">Servers</p>
          <p className="sp-stat__value">{servers?.length ?? 0}</p>
        </div>
        <div className="sp-surface sp-card">
          <p className="sp-stat__label">Running now</p>
          <p className="sp-stat__value" style={{ color: "var(--sp-accent)" }}>{running}</p>
        </div>
        <div className="sp-surface sp-card">
          <p className="sp-stat__label">Coin balance</p>
          <p className="sp-stat__value">{user?.coins.toLocaleString()}</p>
        </div>
      </div>

      <h2 style={{ fontSize: 18, marginBottom: 12 }}>Your servers</h2>
      {isLoading && <p className="sp-mono">loading…</p>}
      {!isLoading && servers?.length === 0 && (
        <p className="sp-mono" style={{ color: "var(--sp-text-muted)" }}>
          No servers yet.
        </p>
      )}

      <div className="sp-grid sp-grid--cards">
        {servers?.map((server) => (
          <Link key={server.id} to={`/servers/${server.id}`} className="sp-surface sp-card" style={{ textDecoration: "none", color: "inherit" }}>
            <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center", marginBottom: 10 }}>
              <strong>{server.name}</strong>
              <StatusBadge status={server.status} />
            </div>
            <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)" }}>
              port {server.primary_port} · {(server.memory_bytes / 1024 / 1024).toFixed(0)}MB
            </p>
          </Link>
        ))}
      </div>
    </div>
  );
}
