import { useQuery } from "@tanstack/react-query";
import { useParams } from "react-router-dom";
import { publicApi } from "../lib/endpoints";
import { formatBytes } from "../lib/format";

// Unauthenticated, shareable status page for a server whose owner opted in.
export function StatusPage() {
  const { id } = useParams<{ id: string }>();
  const { data, isError, isLoading } = useQuery({
    queryKey: ["public-status", id],
    queryFn: () => publicApi.serverStatus(id!),
    enabled: !!id,
    refetchInterval: 10_000,
    retry: false,
  });

  return (
    <div className="sp-shell" style={{ minHeight: "100vh", display: "grid", placeItems: "center", padding: 24 }}>
      <div className="sp-surface sp-card" style={{ width: "100%", maxWidth: 460 }}>
        <p className="sp-kicker">Server status</p>

        {isLoading && <p className="sp-mono" style={{ color: "var(--sp-text-muted)" }}>loading…</p>}
        {isError && (
          <p className="sp-mono" style={{ color: "var(--sp-text-muted)" }}>
            This status page isn't available (the server may be private or doesn't exist).
          </p>
        )}

        {data && (
          <>
            <div style={{ display: "flex", alignItems: "center", gap: 12, flexWrap: "wrap" }}>
              <h1 className="sp-page-title" style={{ margin: 0 }}>
                {data.name}
              </h1>
              <span
                className="sp-badge"
                style={
                  data.online
                    ? { color: "var(--sp-accent)", borderColor: "var(--sp-accent)" }
                    : { color: "#ff9b9b", borderColor: "#ff9b9b" }
                }
              >
                <span className={"sp-dot" + (data.online ? " sp-dot--live" : "")} /> {data.online ? "Online" : "Offline"}
              </span>
            </div>

            {data.online ? (
              <div style={{ marginTop: 18, display: "grid", gap: 14 }}>
                <div style={{ display: "flex", gap: 24, flexWrap: "wrap" }}>
                  <Stat label="Players" value={`${data.player_count}${data.max_players ? ` / ${data.max_players}` : ""}`} />
                  {data.version && <Stat label="Version" value={data.version} />}
                  <Stat label="CPU" value={`${data.cpu_percent.toFixed(1)}%`} />
                  <Stat
                    label="Memory"
                    value={
                      data.mem_limit_bytes > 0
                        ? `${formatBytes(data.mem_used_bytes)} / ${formatBytes(data.mem_limit_bytes)}`
                        : formatBytes(data.mem_used_bytes)
                    }
                  />
                </div>

                {data.players.length > 0 && (
                  <div>
                    <p className="sp-label">Online now</p>
                    <div style={{ display: "flex", flexWrap: "wrap", gap: 8 }}>
                      {data.players.map((p) => (
                        <span key={p} className="sp-badge" style={{ display: "inline-flex", alignItems: "center", gap: 6 }}>
                          <img src={`https://mc-heads.net/avatar/${encodeURIComponent(p)}/18`} alt="" width={18} height={18} style={{ borderRadius: 3 }} />
                          {p}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            ) : (
              <p className="sp-mono" style={{ marginTop: 16, color: "var(--sp-text-muted)" }}>
                The server is currently offline.
              </p>
            )}

            <p className="sp-mono" style={{ fontSize: 11, color: "var(--sp-text-muted)", marginTop: 20 }}>
              live · refreshes automatically
            </p>
          </>
        )}
      </div>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div>
      <p className="sp-stat__label">{label}</p>
      <p className="sp-mono" style={{ fontSize: 18, margin: 0, fontVariantNumeric: "tabular-nums" }}>
        {value}
      </p>
    </div>
  );
}
