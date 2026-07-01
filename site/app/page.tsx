import { AnimatedBackground } from "@/components/AnimatedBackground";
import { Nav } from "@/components/Nav";
import { Footer } from "@/components/Footer";

const FEATURES = [
  {
    title: "Real orchestration",
    body: "Node agents dial out to the panel over an authenticated WebSocket — no inbound ports on your game-server boxes. Every start/stop/console command drives the actual Docker Engine API.",
  },
  {
    title: "Go + Rust",
    body: "A lean Go control plane and a hand-rolled Docker client, with a tiny Rust CLI doing only the genuinely perf-sensitive work: directory sizing, tar+zstd backups, log tailing.",
  },
  {
    title: "Live everything",
    body: "Server console, CPU/memory/network stats, and admin broadcasts all stream over WebSockets in real time. No polling, no stale numbers.",
  },
  {
    title: "A coin economy",
    body: "An AFK page that accrues coins server-side (heartbeat-verified, not client-trusted), daily rewards, and a full ledger — wired into an admin console that can adjust anyone's balance.",
  },
  {
    title: "Themeable, not just dark mode",
    body: "Strict black-and-white by default, with a live theme builder so users can build and save their own — driven entirely by CSS variables, no rebuild required.",
  },
  {
    title: "One command to update",
    body: "`sudo sky-panel-update` pulls the latest release, verifies checksums, and restarts — on the panel box and every node.",
  },
];

export default function Home() {
  return (
    <>
      <AnimatedBackground />
      <Nav />

      <main className="relative z-10 px-6 md:px-12">
        <section className="max-w-4xl mx-auto text-center py-24 md:py-36">
          <p className="font-mono text-xs tracking-[0.25em] text-signal mb-6">SELF-HOSTED · OPEN SOURCE</p>
          <h1 className="font-display text-5xl md:text-7xl leading-[1.05] mb-6">
            A game server panel
            <br />
            that doesn&apos;t get in your way.
          </h1>
          <p className="text-text-muted text-lg max-w-xl mx-auto mb-10">
            Go + Rust backend, real Docker orchestration, live stats over WebSockets, and an admin
            console that actually works. Install it on a VPS in one command.
          </p>
          <div className="flex flex-wrap items-center justify-center gap-4">
            <a
              href="https://github.com/Notbangbang-dev/sky-panel"
              className="rounded-full bg-text text-bg px-6 py-3 text-sm font-medium hover:opacity-90 transition-opacity"
            >
              View on GitHub
            </a>
            <a
              href="https://github.com/Notbangbang-dev/sky-panel/blob/main/installer/README.md"
              className="rounded-full border border-surface-border px-6 py-3 text-sm hover:bg-white/5 transition-colors"
            >
              Install on a VPS
            </a>
          </div>

          <pre className="mt-14 mx-auto max-w-xl text-left rounded-2xl border border-surface-border bg-surface/80 backdrop-blur p-5 font-mono text-sm overflow-x-auto">
            <code>
              curl -fsSL .../installer/install.sh -o install.sh{"\n"}
              sudo bash install.sh panel --domain panel.example.com
            </code>
          </pre>
        </section>

        <section className="max-w-6xl mx-auto pb-28 grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          {FEATURES.map((f) => (
            <div key={f.title} className="rounded-2xl border border-surface-border bg-surface/60 backdrop-blur p-6">
              <h3 className="text-lg mb-2">{f.title}</h3>
              <p className="text-text-muted text-sm leading-relaxed">{f.body}</p>
            </div>
          ))}
        </section>
      </main>

      <Footer />
    </>
  );
}
