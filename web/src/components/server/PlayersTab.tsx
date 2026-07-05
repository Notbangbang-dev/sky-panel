import { useEffect } from "react";
import { useMutation, useQuery } from "@tanstack/react-query";
import { serversApi } from "../../lib/endpoints";

// The panel derives the roster from the server's console stream. We nudge it by
// periodically running `list` (so the server prints its current players, which
// the panel parses) while also polling the tracked roster for display.
export function PlayersTab({ serverId, running, canManage }: { serverId: string; running: boolean; canManage: boolean }) {
  const { data } = useQuery({
    queryKey: ["players", serverId],
    queryFn: () => serversApi.players(serverId),
    refetchInterval: running ? 5000 : false,
    enabled: running,
  });

  useEffect(() => {
    if (!running) return;
    const send = () => serversApi.consoleInput(serverId, "list").catch(() => {});
    send();
    const t = setInterval(send, 15000);
    return () => clearInterval(t);
  }, [serverId, running]);

  const command = useMutation({
    mutationFn: (cmd: string) => serversApi.consoleInput(serverId, cmd),
  });

  const players = data?.players ?? [];

  if (!running) {
    return (
      <div className="sp-surface sp-card" style={{ textAlign: "center", padding: "36px 20px" }}>
        <p className="sp-mono" style={{ color: "var(--sp-text-muted)", margin: 0 }}>
          The server is offline — start it to see who's online.
        </p>
      </div>
    );
  }

  return (
    <div>
      <div style={{ display: "flex", alignItems: "baseline", gap: 12, marginBottom: 14, flexWrap: "wrap" }}>
        <h2 className="sp-mono" style={{ fontSize: 20, margin: 0 }}>
          {players.length}
          {data?.max ? ` / ${data.max}` : ""} online
        </h2>
        {data?.version && (
          <span className="sp-badge" title="Server version">
            {data.version}
          </span>
        )}
        <span className="sp-mono" style={{ fontSize: 11, color: "var(--sp-text-muted)" }}>
          live · updates every few seconds
        </span>
      </div>

      {players.length === 0 ? (
        <div className="sp-surface sp-card" style={{ textAlign: "center", padding: "32px 20px" }}>
          <p className="sp-mono" style={{ color: "var(--sp-text-muted)", margin: 0 }}>
            No players online right now.
          </p>
        </div>
      ) : (
        <div style={{ display: "grid", gap: 8 }}>
          {players.map((name) => (
            <div key={name} className="sp-surface" style={{ display: "flex", alignItems: "center", gap: 12, padding: "10px 14px" }}>
              <img
                src={`https://mc-heads.net/avatar/${encodeURIComponent(name)}/28`}
                alt=""
                width={28}
                height={28}
                style={{ borderRadius: 4, flex: "0 0 28px" }}
              />
              <span style={{ flex: 1, fontWeight: 500 }}>{name}</span>
              {canManage && (
                <>
                  <button
                    className="sp-btn sp-btn--sm sp-btn--ghost"
                    disabled={command.isPending}
                    onClick={() => command.mutate(`kick ${name}`)}
                    title="Kick this player"
                  >
                    Kick
                  </button>
                  <button
                    className="sp-btn sp-btn--sm sp-btn--danger"
                    disabled={command.isPending}
                    onClick={() => {
                      if (window.confirm(`Ban ${name} from the server?`)) command.mutate(`ban ${name}`);
                    }}
                    title="Ban this player"
                  >
                    Ban
                  </button>
                </>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
