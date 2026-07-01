import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { motion, useMotionValue, useTransform, animate } from "framer-motion";
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
      <div />
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
  const rounded = useTransform(motionValue, (v) => Math.round(v).toLocaleString());
  const [display, setDisplay] = useState(rounded.get());

  useEffect(() => {
    const controls = animate(motionValue, value, { duration: 0.6, ease: "easeOut" });
    const unsubscribe = rounded.on("change", (v) => setDisplay(v));
    return () => {
      controls.stop();
      unsubscribe();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [value]);

  return (
    <motion.div className="sp-coin-ticker" whileHover={{ scale: 1.03 }}>
      <span className="sp-coin-ticker__glyph">◈</span>
      <span className="sp-mono">{display}</span>
    </motion.div>
  );
}
