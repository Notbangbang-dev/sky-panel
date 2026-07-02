import { useEffect, useMemo, useRef, useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { useMutation, useQuery } from "@tanstack/react-query";
import { motion } from "framer-motion";
import { eggsApi, serversApi } from "../lib/endpoints";
import { ApiError } from "../lib/api";

const PHASES = ["Removing old container", "Pulling image", "Creating container", "Starting server"];

export function ReinstallPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();

  const { data: server } = useQuery({
    queryKey: ["servers", id],
    queryFn: () => serversApi.get(id!),
    enabled: !!id,
    refetchInterval: (query) => (query.state.data?.status === "installing" ? 2000 : false),
  });
  const { data: eggs } = useQuery({ queryKey: ["eggs"], queryFn: eggsApi.list });

  const [started, setStarted] = useState(false);
  const [eggId, setEggId] = useState("");
  const [error, setError] = useState<string | null>(null);
  const startRef = useRef<number>(0);
  const [now, setNow] = useState(Date.now());

  // Default the egg selector to the server's current egg once loaded.
  useEffect(() => {
    if (server && !eggId) setEggId(server.egg_id);
  }, [server, eggId]);

  // If we land on the page while a (re)install is already running, jump
  // straight to the progress view — don't offer a confirm button that would
  // kick off a second concurrent reinstall.
  useEffect(() => {
    if (!started && server?.status === "installing") {
      startRef.current = Date.now();
      setStarted(true);
    }
  }, [server?.status, started]);

  // Tick for the elapsed timer / phase animation while provisioning.
  useEffect(() => {
    if (!started) return;
    const t = setInterval(() => setNow(Date.now()), 250);
    return () => clearInterval(t);
  }, [started]);

  const reinstall = useMutation({
    mutationFn: () => serversApi.reinstall(id!, eggId && eggId !== server?.egg_id ? eggId : undefined),
    onMutate: () => {
      startRef.current = Date.now();
      setError(null);
      setStarted(true);
    },
    onError: (err) => {
      setStarted(false);
      setError(err instanceof ApiError ? err.message : "Failed to start reinstall.");
    },
  });

  const currentEgg = eggs?.find((e) => e.id === server?.egg_id);
  const targetEgg = eggs?.find((e) => e.id === eggId);
  const switching = !!server && eggId !== "" && eggId !== server.egg_id;

  const status = server?.status;
  // Prefer the live phase the node reports (status_message) over a pure timer,
  // so the checklist tracks what's actually happening. Falls back to the timer
  // estimate before the first phase message lands.
  const messagePhase = status === "installing" ? phaseFromMessage(server?.status_message) : -1;
  const phase =
    status === "installing"
      ? messagePhase >= 0
        ? messagePhase
        : Math.min(PHASES.length - 1, Math.floor((now - startRef.current) / 6000))
      : -1;
  const elapsed = started ? Math.max(0, Math.floor((now - startRef.current) / 1000)) : 0;

  // Which view: confirm (not started) → progress (installing) → done.
  const done = started && (status === "running" || status === "errored");
  const failed = done && status === "errored";

  const reactorState = useMemo(() => {
    if (!started) return "idle";
    if (failed) return "error";
    if (status === "running") return "ok";
    return "run";
  }, [started, failed, status]);

  if (!server) return <p className="sp-mono">loading…</p>;

  return (
    <div className="sp-reinstall">
      <div className="sp-reinstall__panel sp-surface sp-card">
        <p className="sp-kicker">Reinstall</p>

        <Reactor state={reactorState} label={server.name} />

        {/* ---- Confirm ---- */}
        {!started && (
          <>
            <h1 className="sp-reinstall__title">Reinstall “{server.name}”</h1>
            <p className="sp-reinstall__lede">
              This rebuilds the container from scratch and runs the egg's install again. Your files, worlds and configs on
              disk are <strong>kept</strong> — but the server will stop briefly while it re-provisions.
            </p>

            <div className="sp-field" style={{ maxWidth: 420, margin: "0 auto 8px" }}>
              <label className="sp-label">Reinstall with</label>
              <select className="sp-select" value={eggId} onChange={(e) => setEggId(e.target.value)}>
                {eggs?.map((egg) => (
                  <option key={egg.id} value={egg.id}>
                    {egg.category ? `${egg.category} — ${egg.name}` : egg.name}
                    {egg.id === server.egg_id ? " (current)" : ""}
                  </option>
                ))}
              </select>
            </div>

            {switching && (
              <p className="sp-reinstall__warn sp-mono">
                ⚠ Switching from {currentEgg?.name ?? "current"} to {targetEgg?.name}. Existing files may not be compatible
                with the new software.
              </p>
            )}

            {error && <p className="sp-error">{error}</p>}

            <div className="sp-reinstall__actions">
              <button className="sp-btn" onClick={() => navigate(`/servers/${id}`)}>
                Cancel
              </button>
              <button className="sp-btn sp-btn--danger sp-btn--lg" onClick={() => reinstall.mutate()} disabled={reinstall.isPending}>
                {reinstall.isPending ? "Starting…" : switching ? `Reinstall as ${targetEgg?.name}` : "Reinstall"}
              </button>
            </div>
          </>
        )}

        {/* ---- Progress ---- */}
        {started && !done && (
          <>
            <h1 className="sp-reinstall__title">Reinstalling…</h1>
            <div className="sp-phases">
              {PHASES.map((p, i) => (
                <div key={p} className={"sp-phase" + (i < phase ? " is-done" : i === phase ? " is-active" : "")}>
                  <span className="sp-phase__mark">{i < phase ? "✓" : i === phase ? "" : ""}</span>
                  <span>{p}</span>
                </div>
              ))}
            </div>
            {server.status_message && (
              <p className="sp-mono" style={{ fontSize: 12, color: "var(--sp-signal, var(--sp-accent))" }}>
                {server.status_message}
              </p>
            )}
            <p className="sp-reinstall__lede sp-mono" style={{ fontSize: 13 }}>
              Elapsed {formatElapsed(elapsed)} · a cold node pulls the image the first time; after that it's cached and
              this is quick. You can leave — it keeps going.
            </p>
          </>
        )}

        {/* ---- Result ---- */}
        {done && !failed && (
          <>
            <h1 className="sp-reinstall__title">Reinstall complete</h1>
            <p className="sp-reinstall__lede">
              “{server.name}” is back online, running {targetEgg?.name ?? currentEgg?.name ?? "its egg"}.
            </p>
            <div className="sp-reinstall__actions">
              <button className="sp-btn sp-btn--primary sp-btn--lg" onClick={() => navigate(`/servers/${id}`)}>
                Back to server
              </button>
            </div>
          </>
        )}
        {failed && (
          <>
            <h1 className="sp-reinstall__title">Reinstall failed</h1>
            <p className="sp-reinstall__warn sp-mono">{server.status_message || "The node reported an error."}</p>
            <div className="sp-reinstall__actions">
              <button className="sp-btn" onClick={() => navigate(`/servers/${id}`)}>
                Back to server
              </button>
              <button className="sp-btn sp-btn--danger sp-btn--lg" onClick={() => reinstall.mutate()}>
                Try again
              </button>
            </div>
          </>
        )}
      </div>
    </div>
  );
}

function formatElapsed(s: number): string {
  const m = Math.floor(s / 60);
  const sec = s % 60;
  return `${m}:${sec.toString().padStart(2, "0")}`;
}

// Map the node's live status message to an index in PHASES so the checklist
// highlights the real step. Returns -1 when there's no message yet.
function phaseFromMessage(msg?: string): number {
  if (!msg) return -1;
  if (/pulling/i.test(msg)) return 1;
  if (/creating/i.test(msg)) return 2;
  if (/starting/i.test(msg)) return 3;
  return 0;
}

function Reactor({ state, label }: { state: "idle" | "run" | "ok" | "error"; label: string }) {
  const centerLabel =
    state === "ok" ? "ONLINE" : state === "error" ? "FAILED" : state === "run" ? "WORKING" : "READY";
  return (
    <div className={"sp-reactor sp-reactor--" + state}>
      <span className="sp-reactor__ring sp-reactor__ring--1" />
      <span className="sp-reactor__ring sp-reactor__ring--2" />
      <span className="sp-reactor__ring sp-reactor__ring--3" />
      <motion.div
        className="sp-reactor__core"
        animate={state === "run" ? { scale: [1, 1.05, 1] } : { scale: 1 }}
        transition={{ duration: 2.4, repeat: Infinity, ease: "easeInOut" }}
      >
        <span className="sp-reactor__glyph">{state === "error" ? "✕" : state === "ok" ? "✓" : "⟳"}</span>
        <span className="sp-mono sp-reactor__status">{centerLabel}</span>
        <span className="sp-reactor__name">{label}</span>
      </motion.div>
    </div>
  );
}
