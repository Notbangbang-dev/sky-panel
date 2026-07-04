import { useQuery } from "@tanstack/react-query";
import { motion } from "framer-motion";
import { achievementsApi } from "../lib/endpoints";
import type { Achievement } from "../types/api";

// A fixed glyph per achievement id, so each milestone reads as its own emblem
// rather than a generic checkmark. Unknown ids fall back to a neutral mark.
const GLYPHS: Record<string, string> = {
  first_server: "▣",
  fleet: "⬢",
  rich: "⧫",
  generous: "❖",
  lucky: "✧",
  secured: "⛊",
};

export function AchievementsPage() {
  const { data, isError } = useQuery({
    queryKey: ["achievements"],
    queryFn: achievementsApi.list,
    refetchInterval: 60_000,
  });

  const list = data ?? [];
  const unlocked = list.filter((a) => a.unlocked).length;
  const total = list.length;
  const pct = total > 0 ? Math.round((unlocked / total) * 100) : 0;

  return (
    <div>
      <p className="sp-kicker">Milestones</p>
      <div style={{ display: "flex", justifyContent: "space-between", alignItems: "flex-end", marginBottom: 24 }}>
        <h1 className="sp-page-title" style={{ marginBottom: 0 }}>
          Achievements
        </h1>
        <span className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)" }}>
          {unlocked} / {total} unlocked · {pct}%
        </span>
      </div>

      {isError && <p className="sp-error">failed to load achievements</p>}

      {total > 0 && (
        <div className="sp-ach-track" aria-hidden>
          <motion.div
            className="sp-ach-track__fill"
            initial={{ width: 0 }}
            animate={{ width: `${pct}%` }}
            transition={{ type: "spring", stiffness: 90, damping: 20 }}
          />
        </div>
      )}

      <div className="sp-ach-grid">
        {list.map((a, i) => (
          <AchievementCard key={a.id} achievement={a} index={i} />
        ))}
      </div>

      {total === 0 && !isError && (
        <div className="sp-surface sp-card" style={{ textAlign: "center", padding: "40px 20px" }}>
          <p className="sp-mono" style={{ color: "var(--sp-text-muted)", margin: 0 }}>
            No achievements to show.
          </p>
        </div>
      )}
    </div>
  );
}

function AchievementCard({ achievement, index }: { achievement: Achievement; index: number }) {
  const glyph = GLYPHS[achievement.id] ?? "✦";
  return (
    <motion.div
      className="sp-ach-card sp-surface"
      data-unlocked={achievement.unlocked ? "1" : undefined}
      initial={{ opacity: 0, y: 12 }}
      animate={{ opacity: 1, y: 0 }}
      transition={{ delay: index * 0.05, type: "spring", stiffness: 140, damping: 20 }}
    >
      <div className="sp-ach-card__emblem">{achievement.unlocked ? glyph : "🔒"}</div>
      <div className="sp-ach-card__body">
        <div className="sp-ach-card__name">{achievement.name}</div>
        <div className="sp-ach-card__desc">{achievement.description}</div>
      </div>
      <div className="sp-ach-card__state sp-mono">{achievement.unlocked ? "UNLOCKED" : "LOCKED"}</div>
    </motion.div>
  );
}
