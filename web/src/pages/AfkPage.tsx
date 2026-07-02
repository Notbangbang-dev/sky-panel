import { useEffect, useMemo, useRef, useState } from "react";
import { motion } from "framer-motion";
import { coinsApi } from "../lib/endpoints";
import { useAuthStore } from "../lib/authStore";
import { ApiError } from "../lib/api";

// Beat a little faster than the server's 20s minimum-credit window so a credit
// lands on almost every tick, while staying well under the 90s lapse cutoff.
const HEARTBEAT_INTERVAL_MS = 25_000;
const CYCLE_SECONDS = HEARTBEAT_INTERVAL_MS / 1000;

type Status = "connecting" | "earning" | "blocked";

// One random token per browser tab. The server only lets one session earn at a
// time, so a second tab gets "running elsewhere" instead of double coins.
function newSessionId(): string {
  if (typeof crypto !== "undefined" && "randomUUID" in crypto) return crypto.randomUUID();
  return `sess-${Date.now()}-${Math.floor(Math.random() * 1e9)}`;
}

function formatDuration(seconds: number): string {
  const s = Math.max(0, Math.floor(seconds));
  const h = Math.floor(s / 3600);
  const m = Math.floor((s % 3600) / 60);
  const sec = s % 60;
  const pad = (n: number) => n.toString().padStart(2, "0");
  return h > 0 ? `${h}:${pad(m)}:${pad(sec)}` : `${pad(m)}:${pad(sec)}`;
}

export function AfkPage() {
  const updateUser = useAuthStore((s) => s.updateUser);
  const user = useAuthStore((s) => s.user);

  const sessionIdRef = useRef<string>(newSessionId());
  const lastBeatAtRef = useRef<number>(Date.now());
  const sessionStartRef = useRef<number | null>(null);

  const [status, setStatus] = useState<Status>("connecting");
  const [sessionEarned, setSessionEarned] = useState(0);
  const [now, setNow] = useState(Date.now());

  const [dailyMessage, setDailyMessage] = useState<string | null>(null);
  const [dailyLoading, setDailyLoading] = useState(false);

  // The heartbeat loop. Kept in a ref-free effect that reads current state via
  // the store; it stops itself once a session is blocked elsewhere.
  useEffect(() => {
    let cancelled = false;
    let timer: ReturnType<typeof setInterval> | null = null;

    async function beat() {
      try {
        const result = await coinsApi.heartbeat(sessionIdRef.current);
        if (cancelled) return;
        lastBeatAtRef.current = Date.now();
        const started = Date.parse(result.session_started_at);
        if (!Number.isNaN(started)) sessionStartRef.current = started;
        setStatus("earning");
        if (result.credited > 0) {
          setSessionEarned((e) => e + result.credited);
          const current = useAuthStore.getState().user;
          if (current) updateUser({ ...current, coins: result.balance });
        }
      } catch (err) {
        if (cancelled) return;
        if (err instanceof ApiError && err.code === "afk_session_active_elsewhere") {
          setStatus("blocked");
          if (timer) clearInterval(timer);
          timer = null;
        }
        // Other errors are transient — the next tick retries.
      }
    }

    beat();
    timer = setInterval(beat, HEARTBEAT_INTERVAL_MS);
    return () => {
      cancelled = true;
      if (timer) clearInterval(timer);
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // 1s UI ticker for the session timer and the next-credit progress ring.
  useEffect(() => {
    const t = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(t);
  }, []);

  const sessionSeconds = sessionStartRef.current ? (now - sessionStartRef.current) / 1000 : 0;
  const sinceBeat = (now - lastBeatAtRef.current) / 1000;
  const ringProgress = status === "earning" ? Math.min(1, sinceBeat / CYCLE_SECONDS) : 0;
  const ratePerHour = useMemo(() => {
    if (sessionSeconds < 1) return 0;
    return Math.round((sessionEarned / sessionSeconds) * 3600);
  }, [sessionEarned, sessionSeconds]);

  async function retry() {
    sessionIdRef.current = newSessionId();
    setStatus("connecting");
    try {
      const result = await coinsApi.heartbeat(sessionIdRef.current);
      lastBeatAtRef.current = Date.now();
      const started = Date.parse(result.session_started_at);
      if (!Number.isNaN(started)) sessionStartRef.current = started;
      setStatus("earning");
      // Re-arm the loop by reloading — simplest reliable way to restart the
      // interval effect after it stopped on a block.
      window.location.reload();
    } catch {
      setStatus("blocked");
    }
  }

  async function claimDaily() {
    setDailyLoading(true);
    setDailyMessage(null);
    try {
      const result = await coinsApi.claimDaily();
      const current = useAuthStore.getState().user;
      if (current) updateUser({ ...current, coins: result.balance });
      setDailyMessage(`+${result.credited} coins claimed.`);
    } catch (err) {
      setDailyMessage(err instanceof ApiError ? err.message : "Failed to claim.");
    } finally {
      setDailyLoading(false);
    }
  }

  const RING = 116;
  const CIRC = 2 * Math.PI * RING;

  return (
    <div style={{ display: "flex", flexDirection: "column", alignItems: "center", paddingTop: 28 }}>
      <p className="sp-kicker">Idle & earn</p>

      <div style={{ position: "relative", width: 240, height: 240, marginTop: 6 }}>
        <svg width={240} height={240} style={{ position: "absolute", inset: 0, transform: "rotate(-90deg)" }}>
          <circle cx={120} cy={120} r={RING} fill="none" stroke="var(--sp-surface-border)" strokeWidth={2} />
          <circle
            cx={120}
            cy={120}
            r={RING}
            fill="none"
            stroke={status === "blocked" ? "#ff9b9b" : "var(--sp-accent)"}
            strokeWidth={2}
            strokeLinecap="round"
            strokeDasharray={CIRC}
            strokeDashoffset={CIRC * (1 - ringProgress)}
            style={{ transition: "stroke-dashoffset 1s linear" }}
            opacity={status === "earning" ? 1 : 0.3}
          />
        </svg>

        <motion.div
          className="sp-afk-orb"
          style={{ width: 240, height: 240 }}
          animate={status === "earning" ? { scale: [1, 1.04, 1] } : { scale: 1 }}
          transition={{ duration: 3.5, repeat: Infinity, ease: "easeInOut" }}
        >
          <span className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)" }}>
            balance
          </span>
          <span style={{ fontFamily: "var(--sp-font-display)", fontSize: 46, lineHeight: 1 }}>
            {user?.coins.toLocaleString()}
          </span>
          <span className="sp-mono" style={{ fontSize: 12, color: "var(--sp-text-muted)" }}>
            ⧫ coins
          </span>
        </motion.div>
      </div>

      <div className="sp-afk-status" style={{ marginTop: 22 }}>
        <span
          className={
            "sp-afk-status__dot" +
            (status === "earning" ? "" : status === "blocked" ? " sp-afk-status__dot--blocked" : " sp-afk-status__dot--idle")
          }
        />
        {status === "earning" && "earning — keep this page open"}
        {status === "connecting" && "connecting…"}
        {status === "blocked" && "AFK is already running in another tab"}
      </div>

      {status === "blocked" ? (
        <button className="sp-btn sp-btn--primary" style={{ marginTop: 20 }} onClick={retry}>
          Earn here instead
        </button>
      ) : (
        <div className="sp-grid sp-grid--cards" style={{ marginTop: 26, width: "100%", maxWidth: 560 }}>
          <SessionStat label="This session" value={`+${sessionEarned}`} />
          <SessionStat label="Session time" value={formatDuration(sessionSeconds)} />
          <SessionStat label="Rate" value={ratePerHour > 0 ? `${ratePerHour}/hr` : "—"} />
        </div>
      )}

      <div style={{ marginTop: 30, textAlign: "center" }}>
        <button className="sp-btn sp-btn--primary" onClick={claimDaily} disabled={dailyLoading}>
          {dailyLoading ? "Claiming…" : "Claim daily reward"}
        </button>
        {dailyMessage && (
          <p className="sp-mono" style={{ marginTop: 10, fontSize: 13 }}>
            {dailyMessage}
          </p>
        )}
      </div>
    </div>
  );
}

function SessionStat({ label, value }: { label: string; value: string }) {
  return (
    <div className="sp-surface sp-card" style={{ textAlign: "center", padding: 16 }}>
      <p className="sp-stat__label">{label}</p>
      <p className="sp-mono" style={{ fontSize: 22, fontVariantNumeric: "tabular-nums", margin: 0 }}>
        {value}
      </p>
    </div>
  );
}
