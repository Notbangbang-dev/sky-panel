import { useQuery } from "@tanstack/react-query";
import { motion } from "framer-motion";
import { leaderboardApi } from "../lib/endpoints";
import { useAuthStore } from "../lib/authStore";
import type { LeaderboardEntry } from "../types/api";

export function LeaderboardPage() {
  const me = useAuthStore((s) => s.user);
  const { data, isError } = useQuery({
    queryKey: ["leaderboard"],
    queryFn: leaderboardApi.list,
    refetchInterval: 30_000,
  });

  const entries = data ?? [];
  const podium = entries.slice(0, 3);
  const rest = entries.slice(3);

  // Reorder the podium as [2nd, 1st, 3rd] so #1 sits centered and tallest.
  const podiumOrder = [podium[1], podium[0], podium[2]].filter(Boolean) as LeaderboardEntry[];

  return (
    <div>
      <p className="sp-kicker">Coin standings</p>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-end", marginBottom: 24 }}>
        <h1 className="sp-page-title" style={{ marginBottom: 0 }}>
          Leaderboard
        </h1>
        <span className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)" }}>
          top {entries.length} · updates live
        </span>
      </div>

      {isError && <p className="sp-error">failed to load the leaderboard</p>}

      {podium.length > 0 && (
        <div className="sp-podium">
          {podiumOrder.map((entry) => (
            <PodiumPillar key={entry.username} entry={entry} isMe={entry.username === me?.username} />
          ))}
        </div>
      )}

      {rest.length > 0 && (
        <div className="sp-surface sp-card" style={{ marginTop: 22 }}>
          <table className="sp-table">
            <thead>
              <tr>
                <th style={{ width: 60 }}>Rank</th>
                <th>Player</th>
                <th style={{ textAlign: "right" }}>Coins</th>
              </tr>
            </thead>
            <tbody>
              {rest.map((entry) => (
                <tr
                  key={entry.username}
                  style={
                    entry.username === me?.username
                      ? { background: "var(--sp-accent)", color: "var(--sp-accent-text)" }
                      : undefined
                  }
                >
                  <td className="sp-mono">#{entry.rank}</td>
                  <td>{entry.username}</td>
                  <td className="sp-mono" style={{ textAlign: "right", fontVariantNumeric: "tabular-nums" }}>
                    {entry.coins.toLocaleString()} ⧫
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}

      {entries.length === 0 && !isError && (
        <div className="sp-surface sp-card" style={{ textAlign: "center", padding: "40px 20px" }}>
          <p className="sp-mono" style={{ color: "var(--sp-text-muted)", margin: 0 }}>
            No coins earned yet — idle on the AFK page to claim the top spot.
          </p>
        </div>
      )}
    </div>
  );
}

function PodiumPillar({ entry, isMe }: { entry: LeaderboardEntry; isMe: boolean }) {
  const heights: Record<number, number> = { 1: 150, 2: 112, 3: 84 };
  const height = heights[entry.rank] ?? 84;

  return (
    <div className="sp-podium__col">
      <div className="sp-podium__badge" data-rank={entry.rank}>
        {entry.rank}
      </div>
      <div className="sp-podium__name" title={entry.username}>
        {entry.username}
        {isMe && <span className="sp-podium__you"> you</span>}
      </div>
      <div className="sp-podium__coins sp-mono">{entry.coins.toLocaleString()} ⧫</div>
      <motion.div
        className="sp-podium__pillar"
        data-rank={entry.rank}
        initial={{ height: 0, opacity: 0.4 }}
        animate={{ height, opacity: 1 }}
        transition={{ type: "spring", stiffness: 120, damping: 18, delay: (entry.rank - 1) * 0.08 }}
      >
        <span className="sp-podium__pillar-rank">#{entry.rank}</span>
      </motion.div>
    </div>
  );
}
