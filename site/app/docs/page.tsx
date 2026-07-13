import type { Metadata } from "next";
import { AnimatedBackground } from "@/components/AnimatedBackground";
import { Nav } from "@/components/Nav";
import { Footer } from "@/components/Footer";
import { CodeBlock } from "@/components/docs/CodeBlock";
import { Callout } from "@/components/docs/Callout";
import { DocsSidebar, type DocGroup } from "@/components/docs/DocsSidebar";

export const metadata: Metadata = {
  title: "Docs — Sky Panel",
  description:
    "How Sky Panel is put together: architecture, install, fast image-warmed provisioning, eggs, files, backups, automations, the coin economy, API keys and the security model.",
};

const GROUPS: DocGroup[] = [
  {
    group: "Overview",
    items: [
      { id: "what-is-sky-panel", label: "What is Sky Panel" },
      { id: "architecture", label: "Architecture" },
    ],
  },
  {
    group: "Install & run",
    items: [
      { id: "installing", label: "Installing" },
      { id: "fast-provisioning", label: "Fast provisioning" },
      { id: "updating", label: "Updating" },
    ],
  },
  {
    group: "Features",
    items: [
      { id: "servers", label: "Servers & console" },
      { id: "eggs", label: "Eggs & auto-install" },
      { id: "files-sharing", label: "Files & sharing" },
      { id: "backups-schedules", label: "Backups & automations" },
      { id: "databases", label: "Databases" },
      { id: "economy", label: "Coins, store & board" },
      { id: "api-keys", label: "API keys" },
    ],
  },
  {
    group: "Operations",
    items: [
      { id: "security", label: "Security" },
      { id: "uninstalling", label: "Uninstalling" },
    ],
  },
];

function Section({ id, title, kicker, children }: { id: string; title: string; kicker?: string; children: React.ReactNode }) {
  return (
    <section id={id} className="scroll-mt-24">
      {kicker && <p className="mb-1 font-mono text-[11px] uppercase tracking-[0.2em] text-text-muted/70">{kicker}</p>}
      <h2 className="mb-4 font-display text-[28px] leading-tight">{title}</h2>
      <div className="space-y-4 text-sm leading-relaxed text-text-muted">{children}</div>
    </section>
  );
}

function GH({ path, children }: { path: string; children: React.ReactNode }) {
  return (
    <a href={`https://github.com/Notbangbang-dev/${path}`} className="text-text underline decoration-text-muted/40 underline-offset-2 hover:decoration-signal">
      {children}
    </a>
  );
}

export default function DocsPage() {
  return (
    <>
      <AnimatedBackground />
      <Nav />

      <main className="relative z-10 mx-auto w-full max-w-6xl flex-1 px-6 py-14 md:px-12">
        <header className="reveal mb-12">
          <p className="mb-2 font-mono text-[11px] uppercase tracking-[0.28em] text-text-muted">Documentation</p>
          <h1 className="font-display text-4xl md:text-5xl">Run Sky Panel yourself</h1>
          <p className="mt-3 max-w-2xl text-text-muted">
            A control plane on one box, a lightweight signed daemon on every node. Here&apos;s how it fits together —
            and how a server goes from &ldquo;create&rdquo; to online in seconds.
          </p>
        </header>

        <div className="grid gap-12 lg:grid-cols-[220px_minmax(0,1fr)]">
          <aside className="hidden lg:block">
            <div className="sticky top-16">
              <DocsSidebar groups={GROUPS} />
            </div>
          </aside>

          <div className="min-w-0 space-y-16">
            <Section id="what-is-sky-panel" title="What is Sky Panel" kicker="Overview">
              <p>
                Sky Panel is a self-hosted game-server hosting panel: a control plane you run on one box, and a
                lightweight daemon you run on every game-server node behind it. It handles Docker container lifecycle,
                live stats, a console, file management, backups, timed automations, per-server access sharing, and a
                small coin economy — the pieces you&apos;d otherwise stitch together yourself.
              </p>
              <p>It&apos;s split across two repositories with independent release cycles:</p>
              <ul className="list-inside list-disc space-y-1">
                <li>
                  <GH path="sky-panel">sky-panel</GH> — the Go control plane (<code>panel-api</code>) and the React web
                  UI (<code>web</code>), plus the Next.js marketing site and the installer.
                </li>
                <li>
                  <GH path="sky-daemon">sky-daemon</GH> — the Rust daemon that runs on each node, drives Docker, and
                  talks back to the panel over a signed WebSocket.
                </li>
              </ul>
            </Section>

            <Section id="architecture" title="Architecture" kicker="Overview">
              <p>
                <code>panel-api</code> is the single source of truth: users, nodes, eggs, servers, coins, schedules and
                subuser permissions all live in its SQLite database. It never touches Docker directly — instead, every
                node runs <code>sky-daemon</code>, which dials <em>out</em> to the panel over a WebSocket (no inbound
                ports need opening on a game-server box) and drives the local Docker Engine API on the panel&apos;s
                behalf.
              </p>
              <p>
                Every message on that connection, after the initial handshake, is signed with HMAC-SHA256 and carries a
                timestamp and a nonce — the panel verifies all three before acting on anything, closing the connection
                on any failure. The full wire format lives in{" "}
                <a
                  href="https://github.com/Notbangbang-dev/sky-panel/blob/main/docs/ARCHITECTURE.md"
                  className="text-text underline decoration-text-muted/40 underline-offset-2 hover:decoration-signal"
                >
                  docs/ARCHITECTURE.md
                </a>
                .
              </p>
              <p>
                The web app talks to <code>panel-api</code> over authenticated HTTP plus a WebSocket for real-time
                stats, console and broadcasts — it never talks to a node or to <code>sky-daemon</code> directly.
              </p>
            </Section>

            <Section id="installing" title="Installing" kicker="Install &amp; run">
              <p>
                The installer supports three modes, run from a fresh Ubuntu/Debian box. Each is a single command — pipe
                the script straight into <code>bash</code> so there&apos;s nothing to copy wrong.
              </p>
              <p className="font-mono text-text">panel — the control plane + web UI, behind Caddy for automatic HTTPS</p>
              <CodeBlock
                label="panel"
                code={`curl -fsSL https://raw.githubusercontent.com/Notbangbang-dev/sky-panel/main/installer/install.sh | sudo bash -s -- panel --domain panel.example.com`}
              />
              <Callout title="No domain yet?">
                Drop <code>--domain panel.example.com</code> and Caddy serves plain HTTP instead of failing to get a
                certificate for a domain it can&apos;t verify.
              </Callout>
              <p className="font-mono text-text">node — the daemon + Docker, on a game-server box</p>
              <p>Create a node from the admin console first to get its one-time token, then:</p>
              <CodeBlock
                label="node"
                code={`curl -fsSL https://raw.githubusercontent.com/Notbangbang-dev/sky-panel/main/installer/install.sh | sudo bash -s -- node --panel-url wss://panel.example.com/agent/ws --node-token <TOKEN>`}
              />
              <p className="font-mono text-text">all — both on a single box (fine for a first setup)</p>
              <CodeBlock
                label="all-in-one"
                code={`curl -fsSL https://raw.githubusercontent.com/Notbangbang-dev/sky-panel/main/installer/install.sh | sudo bash -s -- all --domain panel.example.com --node-token <TOKEN>`}
              />
              <p>Register the first account once the panel is up — it automatically becomes an admin.</p>
            </Section>

            <Section id="fast-provisioning" title="Fast provisioning" kicker="Install &amp; run">
              <p>
                Creating a server is a two-part story. The <strong>record</strong> — its row, port allocation and
                settings — is written instantly, so the API returns right away and the server shows up immediately as{" "}
                <code>installing</code>. The <strong>container</strong> is then brought up on the node in the
                background, in three explicit phases you can watch live:
              </p>
              <ol className="list-inside list-decimal space-y-1">
                <li>
                  <strong>Pull image</strong> — ensure the egg&apos;s Docker image is on the node.
                </li>
                <li>
                  <strong>Create container</strong> — near-instant once the image is local.
                </li>
                <li>
                  <strong>Start</strong> — the container boots and its console goes live.
                </li>
              </ol>
              <p>
                The only step that can be slow is the image pull — and Sky Panel makes sure it usually isn&apos;t.{" "}
                <strong>Images are warmed ahead of time:</strong> when a node connects, the panel pre-pulls every
                egg&apos;s image in the background, and whenever an egg&apos;s image is added or changed it&apos;s warmed
                across all connected nodes. So by the time you actually create a server, the image is already cached and
                the whole pull → create → start sequence finishes in <strong>seconds</strong>.
              </p>
              <Callout title="Why this is fast">
                A cold node pulls each image exactly once (in the background, before you need it). Every create after
                that skips the download entirely — Docker&apos;s create call hits the local cache in ~100ms. Live phase
                messages (&ldquo;Pulling image…&rdquo;, &ldquo;Creating container…&rdquo;, &ldquo;Starting…&rdquo;) stream
                straight to the server&apos;s console and the reinstall screen, so you always see exactly where it is.
              </Callout>
              <p>
                <strong>Reinstalling</strong> uses the same path and the same warmed cache: it tears down just the
                container (your volume and files are preserved), then re-runs pull → create → start — so a reinstall on
                a warm node is a matter of seconds, tracked live on a dedicated reinstall screen.
              </p>
            </Section>

            <Section id="updating" title="Updating" kicker="Install &amp; run">
              <p>
                Every install places <code>sky-panel-update</code> at <code>/usr/local/bin</code>. Run it any time:
              </p>
              <CodeBlock label="update" code={`sudo sky-panel-update`} />
              <p>
                panel-api/web and sky-daemon are <strong>separate GitHub releases with independent version numbers</strong>{" "}
                — the panel doesn&apos;t wait on a daemon release and vice versa. <code>sky-panel-update</code> checks
                each independently, tracked by two version files on disk:
              </p>
              <ul className="list-inside list-disc space-y-1">
                <li>panel-api/web, from the sky-panel repo&apos;s latest release</li>
                <li>sky-daemon, from its own repo&apos;s latest release</li>
              </ul>
              <p>
                For whichever half is out of date it downloads the new binary, verifies it against the release&apos;s
                published checksums, stops the service, swaps the binary in, restarts it, and prints that release&apos;s
                changelog entry. A box running only <code>panel</code> or only <code>node</code> simply has nothing to do
                for the half it doesn&apos;t have, and says so (&ldquo;already up to date&rdquo;) rather than erroring.
              </p>
              <Callout tone="warn" title="Getting the fast-provisioning speedups">
                Image warming needs <strong>sky-daemon v0.4.0+</strong> on your nodes — run <code>sudo sky-panel-update</code>{" "}
                on each node so it picks up the <code>pull_image</code> support.
              </Callout>
            </Section>

            <Section id="servers" title="Servers &amp; console" kicker="Features">
              <p>
                Each server has a live console (streamed over the WebSocket), real-time CPU/memory/network stats, and
                start / stop / kill / restart controls. Resource limits — memory, CPU and disk — are enforced per server
                and metered against your quota. A server that fails to provision shows the node&apos;s exact error in a
                banner, with a one-click reinstall to retry once you&apos;ve fixed the cause.
              </p>
            </Section>

            <Section id="eggs" title="Eggs &amp; auto-install" kicker="Features">
              <p>
                An &ldquo;egg&rdquo; is a server template: a Docker image, an optional startup command, and a set of
                variables that become environment variables in the container. Sky Panel ships with a starter catalog —
                several Minecraft variants (Paper, Vanilla, Spigot, Forge, Fabric) built on <code>itzg/minecraft-server</code>
                , a BungeeCord proxy, generic Node.js and Python app runners, a Rust (Facepunch) game server, and a blank
                custom-image template.
              </p>
              <p>
                The Minecraft eggs need no startup command at all — that image downloads and configures everything itself
                from the environment variables you set (or leave at their defaults) when creating a server, including
                agreeing to Mojang&apos;s EULA. Admins can add, edit or remove eggs — and their variables — from the admin
                console; changing an egg&apos;s image re-warms it on every node.
              </p>
            </Section>

            <Section id="files-sharing" title="Files &amp; sharing" kicker="Features">
              <p>
                Every server has a file manager (list, read, write, rename, delete, create folders) that operates
                directly on that server&apos;s volume on the host — no shell access to the container required. Owners can
                also grant other accounts scoped access to a server: <code>console</code>, <code>files</code>,{" "}
                <code>power</code> and <code>settings</code> permissions are each granted independently, so you can let
                someone manage files without letting them stop the server.
              </p>
            </Section>

            <Section id="backups-schedules" title="Backups &amp; automations" kicker="Features">
              <p>
                Back up a server&apos;s files to a compressed archive on the node on demand, on a schedule, or restore an
                earlier one. Beyond backups, every server has an <strong>automations</strong> tab: have the panel
                automatically start, stop, restart, kill or back up the server — or run a console command — on a fixed
                interval, from every 30 minutes up to daily. Automations run on their own, even while you&apos;re away,
                and can be paused, resumed or deleted at any time.
              </p>
            </Section>

            <Section id="databases" title="Databases" kicker="Features">
              <p>
                Give users their own <strong>MariaDB databases</strong>, provisioned on the node that hosts their server.
                Each server has a <strong>Databases</strong> tab: create one, and the panel generates a scoped database,
                user and strong password and shows the full connection details (host, port, username, password and a
                ready-to-paste JDBC URL). Databases count against a per-user quota bought from the store, and are dropped
                automatically when the server is deleted.
              </p>
              <p>
                Databases are off until an operator installs MariaDB on a node and points the daemon at it — see the{" "}
                <a
                  href="https://github.com/Notbangbang-dev/sky-panel/blob/main/docs/DATABASES.md"
                  className="text-text underline decoration-text-muted/40 underline-offset-2 hover:decoration-signal"
                >
                  per-node MariaDB setup guide
                </a>
                .
              </p>
            </Section>

            <Section id="economy" title="Coins, store &amp; leaderboard" kicker="Features">
              <p>
                Sky Panel has an optional coin economy: users earn coins idling on the AFK page and claiming a daily
                reward, then spend them in the store to raise their own memory, CPU and disk quota. A leaderboard ranks
                the top balances, with a podium for the top three. Admins tune the earn rates, the store, and any
                user&apos;s coins and quota from the admin console.
              </p>
            </Section>

            <Section id="api-keys" title="API keys" kicker="Features">
              <p>
                Mint personal API keys (prefixed <code>sky_</code>) under Account → API keys and use them as a{" "}
                <code>Bearer</code> token to drive the panel API from scripts and CI — they authenticate the same
                endpoints as a logged-in session. Keys are shown exactly once at creation, stored only as a hash, track
                their last-used time, and can be revoked anytime.
              </p>
              <CodeBlock
                label="curl"
                code={`curl -H "Authorization: Bearer sky_your_key_here" \\
  https://panel.example.com/api/v1/me`}
              />
            </Section>

            <Section id="security" title="Security" kicker="Operations">
              <ul className="list-inside list-disc space-y-1">
                <li>Passwords are bcrypt-hashed; two-factor auth (TOTP) is available per account.</li>
                <li>Changing your password signs every other session out; you can also revoke sessions individually.</li>
                <li>
                  Every panel↔daemon message is signed and replay-protected (see{" "}
                  <a href="#architecture" className="text-text underline decoration-text-muted/40 underline-offset-2 hover:decoration-signal">
                    Architecture
                  </a>
                  ).
                </li>
                <li>Node tokens expire (90 days by default) and can be rotated from the admin console without recreating the node.</li>
                <li>API keys and refresh tokens are stored only as hashes; raw secrets are shown once.</li>
                <li>Registration can be turned off entirely from the admin console once your team is set up.</li>
              </ul>
            </Section>

            <Section id="uninstalling" title="Uninstalling" kicker="Operations">
              <p>
                Sky Panel installs cleanly and removes cleanly — everything lives under a handful of predictable paths, so
                there&apos;s no uninstaller to run, just a short sequence per box. Remove the same halves you installed:
                the <strong>panel</strong> steps on your control-plane box, the <strong>node</strong> steps on each
                game-server box, or both on an all-in-one setup.
              </p>
              <Callout tone="warn" title="This is irreversible — back up first">
                Two paths hold the only copies of your data: <code>/opt/sky-panel/data/sky-panel.db</code> (every user,
                node, egg and server record) on the panel box, and <code>/srv/sky-panel/volumes</code> (every server&apos;s
                files) on each node. Copy them somewhere safe before you start if there&apos;s any chance you&apos;ll want
                them back.
              </Callout>

              <p className="font-mono text-text">Panel box</p>
              <p>Stop the service, drop its unit, then remove the install directory, the update helper and the service user.</p>
              <CodeBlock
                label="remove the panel"
                code={`# stop and disable the control plane
sudo systemctl disable --now sky-panel

# remove its systemd unit and reload
sudo rm -f /etc/systemd/system/sky-panel.service
sudo systemctl daemon-reload

# remove binaries, web UI, config and the database
sudo rm -rf /opt/sky-panel

# remove the update helper and the service user
sudo rm -f /usr/local/bin/sky-panel-update
sudo userdel sky-panel`}
              />
              <Callout title="Caddy is optional to remove">
                The panel installs Caddy for automatic HTTPS. If nothing else on the box uses it, you can remove it too —
                otherwise just delete the Sky Panel site from its config.
              </Callout>
              <CodeBlock
                label="remove Caddy (only if unused elsewhere)"
                code={`sudo systemctl disable --now caddy
sudo rm -f /etc/caddy/Caddyfile
sudo apt-get remove -y caddy
sudo rm -f /etc/apt/sources.list.d/caddy-stable.list \\
  /usr/share/keyrings/caddy-stable-archive-keyring.gpg`}
              />

              <p className="font-mono text-text">Node box</p>
              <p>
                Stop the daemon, remove the game-server containers it created (each is named <code>sky-&lt;server-id&gt;</code>),
                then remove the daemon and — if you&apos;re sure — the server files.
              </p>
              <CodeBlock
                label="remove a node"
                code={`# stop and disable the daemon
sudo systemctl disable --now sky-daemon
sudo rm -f /etc/systemd/system/sky-daemon.service
sudo systemctl daemon-reload

# stop and remove every server container Sky Panel created
docker ps -aq --filter "name=sky-" | xargs -r docker rm -f

# remove the daemon binary, env and update helper
sudo rm -rf /opt/sky-panel
sudo rm -f /usr/local/bin/sky-panel-update

# remove all server files — irreversible
sudo rm -rf /srv/sky-panel/volumes`}
              />
              <Callout title="Docker stays">
                Sky Panel installs Docker on a node only if it wasn&apos;t already there, and never removes it — other
                things may depend on it. If this box was dedicated to Sky Panel and you want Docker gone too, remove it
                with your distro&apos;s package manager afterwards. Egg images stay cached until you{" "}
                <code>docker image prune</code> them; server data is bind-mounted from{" "}
                <code>/srv/sky-panel/volumes</code>, so there are no leftover named Docker volumes to clean up.
              </Callout>

              <p className="font-mono text-text">All-in-one box</p>
              <p>
                If you installed with <code>all</code>, run the panel steps and the node steps on the same box. The shared{" "}
                <code>/opt/sky-panel</code> directory holds both halves, so a single <code>sudo rm -rf /opt/sky-panel</code>{" "}
                clears everything once both services are stopped and disabled.
              </p>
            </Section>
          </div>
        </div>
      </main>

      <Footer />
    </>
  );
}
