import { AnimatedBackground } from "@/components/AnimatedBackground";
import { Nav } from "@/components/Nav";
import { Footer } from "@/components/Footer";
import { APP_VERSION } from "@/lib/version";

const SPECS = [
  { k: "Backend", v: "Go + Rust" },
  { k: "Eggs", v: "10 built-in" },
  { k: "Transport", v: "Signed WS" },
  { k: "License", v: "MIT" },
];

const FEATURES = [
  {
    title: "Real orchestration",
    body: "The Rust daemon runs on each node and dials out to the panel over a signed WebSocket — no inbound ports on your game-server boxes. Every start, stop, and console keystroke drives the real Docker Engine API.",
  },
  {
    title: "A 10-egg catalog, ready to go",
    body: "Paper, Vanilla, Spigot, Forge, Fabric, a BungeeCord proxy, generic Node.js and Python runners, a Rust game server, and a custom-image template — seeded on install, EULA pre-agreed, no config to write.",
  },
  {
    title: "Live everything",
    body: "Server console, CPU / memory / network stats, and admin broadcasts all stream over WebSockets in real time. No polling, no stale numbers.",
  },
  {
    title: "Files & sharing",
    body: "A full file manager on every server — read, write, rename, upload — plus per-server subusers with scoped console / files / power / settings permissions. Hand out exactly the access you mean to.",
  },
  {
    title: "Signed, replay-proof protocol",
    body: "Every panel↔node message after the handshake is HMAC-signed with a timestamp and nonce. Node tokens expire and rotate. TOTP two-factor on every account.",
  },
  {
    title: "One command to update",
    body: "sudo sky-panel-update pulls the latest release, verifies checksums, swaps binaries, and restarts — panel and daemon tracked as independent versions.",
  },
];

export default function Home() {
  return (
    <>
      <AnimatedBackground />
      <Nav />

      <main className="relative z-10 px-6 md:px-12">
        {/* ---------- Hero ---------- */}
        <section className="max-w-4xl mx-auto text-center pt-24 pb-28 md:pt-36 md:pb-36">
          <p className="reveal font-mono text-[11px] tracking-[0.32em] text-signal mb-7 uppercase">
            Self-hosted · Open source · v{APP_VERSION}
          </p>
          <h1 className="reveal font-display text-5xl md:text-[5.5rem] leading-[1.02] mb-2" style={{ animationDelay: "0.06s" }}>
            A game server panel
            <br />
            that gets out of your way.
          </h1>
          <div
            className="reveal mx-auto mb-8 h-px w-40 bg-signal"
            style={{ animation: "sweep 2.4s ease-in-out 0.6s infinite", animationDelay: "0.6s" }}
          />
          <p
            className="reveal text-text-muted text-lg max-w-xl mx-auto mb-10 leading-relaxed"
            style={{ animationDelay: "0.12s" }}
          >
            A lean Go control plane, a Rust node daemon, real Docker orchestration, and live stats over
            WebSockets. Stand it up on a fresh VPS in a single command.
          </p>

          <div className="reveal flex flex-wrap items-center justify-center gap-4" style={{ animationDelay: "0.18s" }}>
            <a
              href="https://github.com/Notbangbang-dev/sky-panel"
              className="rounded-full bg-text text-bg px-6 py-3 text-sm font-medium hover:opacity-90 transition-opacity"
            >
              View on GitHub
            </a>
            <a
              href="/docs"
              className="rounded-full border border-surface-border px-6 py-3 text-sm hover:bg-text/5 transition-colors"
            >
              Read the docs
            </a>
          </div>

          <div
            className="reveal panel ticked mt-14 mx-auto max-w-2xl text-left rounded-2xl p-6 overflow-x-auto"
            style={{ animationDelay: "0.24s" }}
          >
            <p className="font-mono text-[10px] tracking-[0.2em] text-text-muted uppercase mb-3">
              install — panel + web UI, automatic HTTPS
            </p>
            <pre className="font-mono text-sm leading-relaxed whitespace-pre">
              <code>
                <span className="text-signal">$ </span>
                curl -fsSL https://raw.githubusercontent.com/Notbangbang-dev/sky-panel/main/installer/install.sh \{"\n"}
                {"    "}| sudo bash -s -- panel --domain panel.example.com
              </code>
            </pre>
          </div>

          {/* Spec strip */}
          <div className="reveal mt-12 flex flex-wrap items-center justify-center gap-x-10 gap-y-4" style={{ animationDelay: "0.3s" }}>
            {SPECS.map((s) => (
              <div key={s.k} className="text-center">
                <p className="font-mono text-[10px] tracking-[0.18em] text-text-muted uppercase">{s.k}</p>
                <p className="font-display text-2xl">{s.v}</p>
              </div>
            ))}
          </div>
        </section>

        {/* ---------- Features ---------- */}
        <section className="max-w-6xl mx-auto pb-32">
          <div className="flex items-baseline gap-4 mb-10">
            <span className="font-mono text-xs tracking-[0.2em] text-signal">/ FEATURES</span>
            <span className="h-px flex-1 bg-surface-border" />
          </div>
          <div className="grid gap-5 md:grid-cols-2 lg:grid-cols-3">
            {FEATURES.map((f, i) => (
              <div key={f.title} className="panel ticked rounded-2xl p-7 flex flex-col gap-3">
                <span className="font-mono text-[11px] text-text-muted">{String(i + 1).padStart(2, "0")}</span>
                <h3 className="font-display text-2xl leading-tight">{f.title}</h3>
                <p className="text-text-muted text-sm leading-relaxed">{f.body}</p>
              </div>
            ))}
          </div>
        </section>

        {/* ---------- Closing CTA ---------- */}
        <section className="max-w-3xl mx-auto pb-32 text-center">
          <h2 className="font-display text-4xl md:text-5xl mb-5">Run your own.</h2>
          <p className="text-text-muted mb-8 max-w-lg mx-auto">
            Free, open source, and yours to modify. Point it at a VPS and you have a control plane in minutes.
          </p>
          <a
            href="/docs"
            className="inline-block rounded-full bg-text text-bg px-7 py-3 text-sm font-medium hover:opacity-90 transition-opacity"
          >
            Get started
          </a>
        </section>
      </main>

      <Footer />
    </>
  );
}
