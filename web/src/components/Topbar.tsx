import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { motion, useMotionValue, animate } from "framer-motion";
import { useAuthStore } from "../lib/authStore";
import { authApi } from "../lib/endpoints";

export function Topbar({ username, coins }: { username: string; coins: number }) {
  const navigate = useNavigate();
  const refreshToken = useAuthStore((s) => s.refreshToken);
  const clear = useAuthStore((s) => s.clear);

  async function handleLogout() {
    if (refreshToken) authApi.logout(refreshToken).catch(() => {});
    clear();
    navigate("/login");
  }

  return (
    <header className="sp-topbar sp-surface">
      <span className="sp-topbar__readout">
        <span className="sp-topbar__readout-dot" />
        SKY&nbsp;PANEL <span className="sp-topbar__readout-sep">//</span> CONTROL DECK
      </span>
      <div className="sp-topbar__right">
        <CoinTicker value={coins} />
        <div className="sp-topbar__user">
          <span className="sp-topbar__avatar">{username.slice(0, 1).toUpperCase()}</span>
          <span className="sp-mono">{username}</span>
        </div>
        <button className="sp-btn sp-btn--sm" onClick={handleLogout}>
          Sign out
        </button>
      </div>
    </header>
  );
}

function CoinTicker({ value }: { value: number }) {
  const motionValue = useMotionValue(value);
  const [display, setDisplay] = useState(() => value.toLocaleString());

  useEffect(() => {
    // In a hidden/background tab, requestAnimationFrame is paused, so a tween
    // never progresses and would leave the counter frozen on a stale number.
    // Show the exact value immediately in that case; only animate when the tab
    // is actually visible.
    if (typeof document !== "undefined" && document.hidden) {
      motionValue.set(value);
      setDisplay(value.toLocaleString());
      return;
    }
    const controls = animate(motionValue, value, {
      duration: 0.6,
      ease: "easeOut",
      onUpdate: (v) => setDisplay(Math.round(v).toLocaleString()),
      // Snap to the exact target on finish so an interrupted tween can't leave
      // the counter frozen on a stale number.
      onComplete: () => setDisplay(value.toLocaleString()),
    });
    return () => controls.stop();
  }, [value, motionValue]);

  return (
    <motion.div className="sp-coin-ticker" whileHover={{ scale: 1.03 }}>
      <span className="sp-coin-ticker__glyph">◈</span>
      <span className="sp-mono">{display}</span>
    </motion.div>
  );
}
