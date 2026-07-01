import { useEffect, useRef, useState } from "react";
import { motion } from "framer-motion";
import { coinsApi } from "../lib/endpoints";
import { useAuthStore } from "../lib/authStore";
import { ApiError } from "../lib/api";

const HEARTBEAT_INTERVAL_MS = 25_000;

export function AfkPage() {
  const updateUser = useAuthStore((s) => s.updateUser);
  const user = useAuthStore((s) => s.user);

  const [lastCredited, setLastCredited] = useState<number | null>(null);
  const [dailyMessage, setDailyMessage] = useState<string | null>(null);
  const [dailyLoading, setDailyLoading] = useState(false);

  const heartbeatRef = useRef<ReturnType<typeof setInterval> | null>(null);

  useEffect(() => {
    async function beat() {
      try {
        const result = await coinsApi.heartbeat();
        setLastCredited(result.credited);
        if (user) updateUser({ ...user, coins: result.balance });
      } catch {
        // network hiccup — the next tick will just try again
      }
    }

    beat();
    heartbeatRef.current = setInterval(beat, HEARTBEAT_INTERVAL_MS);
    return () => {
      if (heartbeatRef.current) clearInterval(heartbeatRef.current);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  async function claimDaily() {
    setDailyLoading(true);
    setDailyMessage(null);
    try {
      const result = await coinsApi.claimDaily();
      if (user) updateUser({ ...user, coins: result.balance });
      setDailyMessage(`+${result.credited} coins claimed.`);
    } catch (err) {
      setDailyMessage(err instanceof ApiError ? err.message : "Failed to claim.");
    } finally {
      setDailyLoading(false);
    }
  }

  return (
    <div style={{ display: "flex", flexDirection: "column", alignItems: "center", paddingTop: 40 }}>
      <motion.div
        className="sp-afk-orb"
        animate={{ scale: [1, 1.06, 1] }}
        transition={{ duration: 3.5, repeat: Infinity, ease: "easeInOut" }}
      >
        <span className="sp-mono" style={{ fontSize: 13, color: "var(--sp-text-muted)" }}>
          balance
        </span>
        <span style={{ fontFamily: "var(--sp-font-display)", fontSize: 48 }}>{user?.coins.toLocaleString()}</span>
      </motion.div>

      <p className="sp-mono" style={{ marginTop: 22, color: "var(--sp-text-muted)", fontSize: 13 }}>
        {lastCredited && lastCredited > 0 ? `+${lastCredited} coin credited` : "keep this page open to earn coins"}
      </p>

      <button className="sp-btn sp-btn--primary" style={{ marginTop: 28 }} onClick={claimDaily} disabled={dailyLoading}>
        {dailyLoading ? "Claiming…" : "Claim daily reward"}
      </button>

      {dailyMessage && (
        <p className="sp-mono" style={{ marginTop: 10, fontSize: 13 }}>
          {dailyMessage}
        </p>
      )}
    </div>
  );
}
