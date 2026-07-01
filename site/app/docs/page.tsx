import type { Metadata } from "next";
import { AnimatedBackground } from "@/components/AnimatedBackground";
import { Nav } from "@/components/Nav";
import { Footer } from "@/components/Footer";

export const metadata: Metadata = {
  title: "Docs — Sky Panel",
  description: "How Sky Panel is put together, how to install and update it, and how eggs, file manager and sharing work.",
};

const SECTIONS = [
  { id: "what-is-sky-panel", label: "What is Sky Panel" },
  { id: "architecture", label: "Architecture" },
  { id: "installing", label: "Installing" },
  { id: "updating", label: "Updating" },
  { id: "eggs", label: "Eggs" },
  { id: "files-and-sharing", label: "File manager & sharing" },
  { id: "security", label: "Security" },
];

function Code({ children }: { children: string }) {
  return (
    <pre className="bg-surface border border-surface-border rounded-lg p-4 overflow-x-auto text-xs font-mono text-text my-4">
      <code>{children}</code>
    </pre>
  );
}

function Section({ id, title, children }: { id: string; title: string; children: React.ReactNode }) {
  return (
    <section id={id} className="scroll-mt-24">
      <h2 className="font-display text-2xl mb-4">{title}</h2>
      <div className="space-y-4 text-sm text-text-muted leading-relaxed">{children}</div>
    </section>
  );
}

export default function DocsPage() {
  return (
    <>
      <AnimatedBackground />
      <Nav />

      <main className="relative z-10 px-6 md:px-12 max-w-3xl mx-auto py-20 flex-1">
        <h1 className="font-display text-4xl mb-4">Docs</h1>
        <p className="text-text-muted mb-10">Everything you need to run Sky Panel yourself.</p>

        <nav className="flex flex-wrap gap-x-5 gap-y-2 mb-16 text-xs font-mono">
          {SECTIONS.map((s) => (
            <a key={s.id} href={`#${s.id}`} className="text-text-muted hover:text-text transition-colors">
              {s.label}
            </a>
          ))}
        </nav>

        <div className="space-y-16">
          <Section id="what-is-sky-panel" title="What is Sky Panel">
            <p>
              Sky Panel is a self-hosted game-server hosting panel: a control plane you run on one box, and a
              lightweight daemon you run on every game-server node behind it. It handles Docker container
              lifecycle, live stats, a console, file management, per-server access sharing, and a small coin
              economy — the pieces you&apos;d otherwise stitch together yourself.
            </p>
            <p>It&apos;s split across two repositories with independent release cycles:</p>
            <ul className="list-disc list-inside space-y-1">
              <li>
                <a href="https://github.com/Notbangbang-dev/sky-panel" className="underline">
                  sky-panel
                </a>{" "}
                — the Go control plane (<code>panel-api</code>) and the React web UI (<code>web</code>).
              </li>
              <li>
                <a href="https://github.com/Notbangbang-dev/sky-daemon" className="underline">
                  sky-daemon
                </a>{" "}
                — the Rust daemon that runs on each node, drives Docker, and talks back to the panel over a
                signed WebSocket.
              </li>
            </ul>
          </Section>

          <Section id="architecture" title="Architecture">
            <p>
              <code>panel-api</code> is the single source of truth: users, nodes, eggs, servers, coins, and
              subuser permissions all live in its SQLite database. It never touches Docker directly — instead,
              every node runs <code>sky-daemon</code>, which dials <em>out</em> to the panel over a WebSocket (no
              inbound ports need to be opened on a game-server box) and drives the local Docker Engine API on the
              panel&apos;s behalf.
            </p>
            <p>
              Every message on that connection, after an initial handshake, is signed with HMAC-SHA256 and
              carries a timestamp and a nonce — the panel verifies all three before acting on anything, closing
              the connection on any failure. The full wire format is documented in{" "}
              <a
                href="https://github.com/Notbangbang-dev/sky-panel/blob/main/docs/ARCHITECTURE.md"
                className="underline"
              >
                docs/ARCHITECTURE.md
              </a>{" "}
              if you want the exact bytes.
            </p>
            <p>
              The web app talks to <code>panel-api</code> over plain authenticated HTTP + a WebSocket for
              real-time stats/console/broadcasts — it never talks to a node or to <code>sky-daemon</code>{" "}
              directly.
            </p>
          </Section>

          <Section id="installing" title="Installing">
            <p>
              The installer supports three modes, run from a fresh Ubuntu/Debian box. Each is a single command —
              pipe the script straight into <code>bash</code> so there&apos;s nothing to copy wrong:
            </p>
            <p className="font-mono text-text">panel — the control plane + web UI, behind Caddy for automatic HTTPS</p>
            <Code>{`curl -fsSL https://raw.githubusercontent.com/Notbangbang-dev/sky-panel/main/installer/install.sh | sudo bash -s -- panel --domain panel.example.com`}</Code>
            <p>
              If you don&apos;t have a domain pointed at the box yet, just drop <code>--domain panel.example.com</code>{" "}
              — Caddy serves plain HTTP instead of trying (and failing) to get a certificate for a domain it
              can&apos;t verify.
            </p>
            <p className="font-mono text-text">node — the daemon + Docker, on a game-server box</p>
            <p>Create a node from the admin console first to get its one-time token, then:</p>
            <Code>{`curl -fsSL https://raw.githubusercontent.com/Notbangbang-dev/sky-panel/main/installer/install.sh | sudo bash -s -- node --panel-url wss://panel.example.com/agent/ws --node-token <TOKEN>`}</Code>
            <p className="font-mono text-text">all — both on a single box (fine for a first setup)</p>
            <Code>{`curl -fsSL https://raw.githubusercontent.com/Notbangbang-dev/sky-panel/main/installer/install.sh | sudo bash -s -- all --domain panel.example.com --node-token <TOKEN>`}</Code>
            <p>Register the first account once the panel is up — it automatically becomes an admin.</p>
          </Section>

          <Section id="updating" title="Updating">
            <p>
              Every install places <code>sky-panel-update</code> at <code>/usr/local/bin</code>. Run it any time:
            </p>
            <Code>sudo sky-panel-update</Code>
            <p>
              panel-api/web and sky-daemon are <strong>separate GitHub releases with independent version
              numbers</strong> — the panel doesn&apos;t wait on a daemon release and vice versa. <code>sky-panel-update</code>{" "}
              checks each independently, tracked by two separate version files on disk:
            </p>
            <ul className="list-disc list-inside space-y-1">
              <li>panel-api/web, from the sky-panel repo&apos;s latest release</li>
              <li>sky-daemon, from its own repo&apos;s latest release</li>
            </ul>
            <p>
              For whichever half is out of date, it downloads the new binary, verifies it against the release&apos;s
              published checksums, stops the relevant service, swaps the binary in, and restarts it — then prints
              that release&apos;s changelog entry. A box running only <code>panel</code> or only <code>node</code>{" "}
              simply has nothing to do for the half it doesn&apos;t have installed, and it says so
              (&quot;already up to date&quot;) rather than erroring.
            </p>
          </Section>

          <Section id="eggs" title="Eggs">
            <p>
              An &quot;egg&quot; is a server template: a Docker image, an optional startup command, and a set of
              variables that become environment variables in the container. Sky Panel ships with a starter
              catalog — several Minecraft variants (Paper, Vanilla, Spigot, Forge, Fabric) built on{" "}
              <code>itzg/minecraft-server</code>, a BungeeCord proxy, generic Node.js and Python app runners, a
              Rust (Facepunch) game server, and a blank custom-image template.
            </p>
            <p>
              The Minecraft eggs need no startup command at all — that image downloads and configures everything
              itself from the environment variables you set (or leave at their defaults) when creating a server,
              including agreeing to Mojang&apos;s EULA. Admins can add, edit, or remove eggs — including their
              variables — from the admin console.
            </p>
          </Section>

          <Section id="files-and-sharing" title="File manager & sharing">
            <p>
              Every server has a file manager (list, read, write, rename, delete, create folders) that operates
              directly on that server&apos;s volume on the host — no shell access to the container required. Owners
              can also grant other accounts scoped access to a server: <code>console</code>, <code>files</code>,{" "}
              <code>power</code>, and <code>settings</code> permissions can each be granted independently, so you
              can, for example, let someone manage files without letting them stop the server.
            </p>
          </Section>

          <Section id="security" title="Security">
            <ul className="list-disc list-inside space-y-1">
              <li>Passwords are hashed; two-factor auth (TOTP) is available per account.</li>
              <li>
                Every panel↔daemon message is signed and replay-protected (see{" "}
                <a href="#architecture" className="underline">
                  Architecture
                </a>
                ).
              </li>
              <li>
                Node tokens expire (90 days by default) and can be rotated from the admin console without
                recreating the node.
              </li>
              <li>Registration can be turned off entirely from the admin console once your team is set up.</li>
            </ul>
          </Section>
        </div>
      </main>

      <Footer />
    </>
  );
}
