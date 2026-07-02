# Changelog

All notable changes to Sky Panel are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [0.15.0] - 2026-07-02

### ✨ New Features

- **Servers create in seconds.** Provisioning is now split into three visible phases — **pull image → create container → start** — and the only slow step (the image pull) is taken out of the critical path. When a node connects, the panel **pre-warms every egg's Docker image** in the background; adding or changing an egg's image re-warms it on every connected node. By the time you create a server the image is already cached, so create/start finish in seconds instead of blocking on a multi-minute download.
- **Live provisioning progress.** The install and reinstall screens now show the real phase the node is on (&ldquo;Pulling image…&rdquo;, &ldquo;Creating container…&rdquo;, &ldquo;Starting…&rdquo;) streamed straight from the daemon, instead of a static spinner — the same treatment reinstall gets.
- **Reinstall is fast too.** It reuses the warmed cache and the same phased flow, so rebuilding a container on a warm node is a matter of seconds, tracked live on the reinstall screen. Your volume and files are still preserved.

### 🛠 Under the hood

- New `pull_image` daemon command (idempotent — a no-op when the image is already present) and panel-side image warming on node-connect and egg change.

### 📚 Docs

- The marketing site's **Docs** page was rebuilt with a sticky scroll-spy sidebar, copy-to-clipboard code blocks and callouts, and now covers everything: architecture, install, fast provisioning, eggs, files & sharing, backups & automations, the coin economy, API keys and the security model. The **Changelog** page is now a proper release timeline.

### 🔗 Requires

- **sky-daemon v0.4.0** on your nodes for image warming and live pull progress (`sudo sky-panel-update` on each node). Without it, servers still create — just without the pre-warm speedup.

## [0.14.0] - 2026-07-02

### ✨ New Features

Five more features ported from the cloud-panel playbook:

- **Change your password.** Account → Change password verifies your current password, enforces a minimum length, and — for safety — signs every other session out when you change it. The tab you're on stays logged in.
- **Active sessions.** See every device currently signed in to your account, with the current one clearly marked. Revoke any single session, or **Sign out others** in one click (e.g. after using a shared computer).
- **Personal API keys.** Mint `sky_…` keys under Account → API keys and use them as a `Bearer` token to drive the panel API from scripts and CI. Keys are shown exactly once at creation (and only ever stored hashed), track their last-used time, and can be revoked anytime.
- **Server automations (schedules).** Each server has a **Schedules** tab: have the panel automatically **start / stop / restart / kill / back up** a server — or **run a console command** — on a fixed interval (every 30 minutes up to daily). Pause, resume, or delete any automation. They run on their own, even while you're away.
- **Coins leaderboard.** A new **Leaderboard** page ranks the top coin balances with a podium for the top three (you're highlighted), so idling on the AFK page now comes with some bragging rights.

### 🔗 Requires

- Panel-only release — works with sky-daemon v0.3.0.

## [0.13.0] - 2026-07-02

### ✨ New Features

- **A dedicated Reinstall experience.** Hitting **Reinstall** now opens a full page instead of a plain confirm dialog: a clear rundown of what happens (container rebuilt, files kept, brief downtime), then a live animated "reactor" that tracks provisioning through to **complete** or a clear **failure** (with the node's error message and a one-click **Try again**).
- **Reinstall onto a different egg.** From that page you can switch the server's software — e.g. Paper → Fabric, or a Minecraft server → a Python app — with a warning that existing files may not be compatible. The volume is preserved; only the container/image changes.

### 🔗 Requires

- Panel-only release — works with sky-daemon v0.3.0.

## [0.12.0] - 2026-07-02

### 🛠 Fixes

- **The server page no longer looks stuck on "installing".** It now refreshes itself while a server is provisioning, so it moves to **running** (or **errored**) on its own — no manual reload.
- **Errored servers explain why.** When provisioning fails, the reason from the node (e.g. `dispatch create: node is not connected`, an image-pull error, a Docker error) is captured and shown in a banner on the server page, instead of a bare "errored". Fix the cause and hit **Reinstall** to retry.
- **Reinstall is now async too** and uses the long image-pull timeout, so retrying a failed install actually works instead of timing out.

### 🎨 Improvements

- Server page polish: a cleaner header with a status line and a spec strip (port · RAM · CPU · disk), an animated "provisioning" banner, and a clearer errored banner.

### 🔗 Requires

- Panel-only release — works with sky-daemon v0.3.0.

## [0.11.0] - 2026-07-02

### 🛠 Fixes

- **Creating a server no longer fails with `dispatch create: context deadline exceeded`.** The panel used to wait synchronously (15s, under a 30s request cap) for the node to create and start the container — but a first-time create pulls the egg's Docker image, which can take minutes. Provisioning is now **asynchronous**: creating a server returns immediately with the server shown as **installing**, the node does the work (including the image pull, see sky-daemon v0.3.0) in the background, and the server flips to **running** (or **errored**) when it's done. The servers list auto-refreshes while anything is installing, so it updates on its own.

### 🔗 Requires

- **sky-daemon v0.3.0** on your nodes for the image-pull fix (`sudo sky-panel-update` updates both). Without it, creation still won't fail with a timeout, but a node missing the image will end up **errored** until the image is present.

## [0.10.0] - 2026-07-02

### ✨ New Features

- **Admin → Quotas tab** — one place to control resource limits: the global **default** memory, CPU, and disk every user starts with, plus a **"Disable unlimited CPU"** switch.
- **Turn off unlimited CPU** — with it off, a server can no longer be created or resized with a CPU limit of `0` (which meant "unlimited" and quietly sidestepped the CPU quota). Every server must now reserve CPU from the user's quota, just like memory and disk — so all three are genuinely capped. Admins remain unmetered. The create form requires a CPU value and `/me/quota` reports the setting so the UI adapts.

### 🔗 Requires

- Panel-only release — works with sky-daemon v0.2.0.

## [0.9.0] - 2026-07-01

### 🛠 Fixes

- **The coin balance in the top bar now stays live.** It refreshes on its own (and the moment you refocus the tab), so it reflects AFK earnings, purchases, and admin adjustments without a manual reload. Two underlying bugs are fixed: the panel never re-fetched your balance after login, and the animated counter could freeze on a stale number — it crashed on one code path and, in a background tab, the count-up animation was paused by the browser. It now shows the exact value immediately when the tab is hidden and animates only when visible.

### 🎨 Improvements

- A dramatic pass on the panel's black-and-white "control instrument" look, over the same animated background:
  - A faint CRT **scanline** layer and stronger grain for lit-screen texture.
  - **Instrument panel surfaces** with a lit top-edge bezel and a soft radial sheen.
  - Bigger, more **editorial typography** — larger display-serif page titles and stat numbers, with readout-style kickers.
  - A **glowing** coin ticker and active-nav indicator; a live "system online" status in the sidebar and a "CONTROL DECK" readout in the top bar.
  - A sheen that sweeps across primary buttons on hover, and a staggered reveal as cards load in.

### 🔗 Requires

- Panel-only release — works with sky-daemon v0.2.0.

## [0.8.0] - 2026-07-01

### ✨ New Features

- **Editable economy (admin → Economy tab)** — tune how the AFK page and daily reward pay out without redeploying: coins per AFK tick, the min/max tick window, the daily reward amount, and its cooldown. Changes take effect immediately; each value falls back to a sensible default if unset.
- **Per-user coins & quota in the admin console** — the admin Users tab now shows each user's live resource usage against their limit, and the quota editor pre-fills the current bonus so you can see and adjust a user's RAM, CPU, and disk (on top of the global default) — alongside the existing coin adjustment.
- **Server suspension** — admins can suspend a server from its page: it's stopped, and its owner can't start it, use its console, save settings, or reinstall until an admin unsuspends it. Suspended servers show a clear badge in the list and on the server page. (A classic control from panels like cloudpanel/Pterodactyl.)

### 🛠 Fixes

- A misconfigured AFK tick window (min ≥ max) no longer silently stops all earning — the service falls back to the defaults.
- Suspension can't be bypassed via the settings-save or reinstall paths (both re-provision and would otherwise restart the container).

### 🔗 Requires

- Panel-only release — works with sky-daemon v0.2.0.

## [0.7.0] - 2026-07-01

### ✨ New Features

- **Admin Allocations tab** — a new tab in the admin console to manage a node's port allocations: see every port (free vs. in use, and by which server), add a single port or a whole range at once, and delete free ports. Existing ports are skipped so re-adding a range is safe.
- **Ports out of the box** — every newly registered node is automatically seeded with 50 default port allocations (25565–25614), so you can create a server on it immediately without hand-seeding the database. Add more any time from the Allocations tab.
- **Automatic port publishing** — when a server claims an allocation, the node's daemon publishes that port on the host for both TCP and UDP (bound to all interfaces), so the server is reachable at `node-ip:port` — the same "just works" flow as other panels. (On a cloud host, make sure the port range is open in your firewall / security group.)

### 🛠 Fixes

- Deleting an allocation is now an atomic check-and-delete, closing a race where a port could be claimed by a new server in the instant between the "is it free?" check and the delete.

### 🔗 Requires

- Panel-only release — works with sky-daemon v0.2.0.

## [0.6.0] - 2026-07-01

### ✨ New Features

- **Resource quotas** — every user now has a quota capping the total RAM, CPU, and disk they can allocate across all their servers. Creating or resizing a server is checked against it (admins are unmetered), and a new usage meter on the create form, the servers list, and the store shows exactly how much of each you've used. Defaults are 2 GB RAM / 2 cores / 10 GB disk and are admin-configurable.
- **Per-server disk allocation** — servers now carry a disk allocation (settable on create and in the Settings tab) that counts against your disk quota. (Declared allocation for accounting; on-node enforcement via usage monitoring is planned.)
- **Coin store** — spend the coins you earn on permanent quota upgrades: +512 MB / +1 GB RAM, +50% / +100% CPU, +5 GB / +10 GB disk. Buying one debits your balance (atomically, never below zero) and raises the matching limit immediately.
- **A real AFK page** — the AFK screen is now a proper idle session: a live balance orb with a next-credit progress ring, a session timer, coins earned this session, and your earn rate — plus the daily reward claim.
- **AFK anti-abuse** — only one AFK session earns at a time. Opening the AFK page in a second tab shows "already running in another tab" and earns nothing until the first session goes idle, so you can't multiply coins by stacking tabs.
- **Admin quota control** — the admin user table gained a per-user quota editor to grant bonus RAM/CPU/disk on top of the defaults.

### 🔗 Requires

- No node update needed — this release is panel-only and works with sky-daemon v0.2.0.

## [0.5.0] - 2026-07-01

### ✨ New Features

- **Per-server Settings tab** — rename a server and change its Memory, CPU limit, and automatic-backup interval from a proper form. Saving re-provisions the container in place; your files and world data are preserved.
- **CPU limits** — servers now take a real CPU cap (percent of one core: `0` = unlimited, `100` = one full core, `200` = two cores), enforced by the node via Docker. Available both when creating a server and in the Settings tab.
- **Reinstall server** — a one-click rebuild of the container from its egg, keeping the server's volume (files, worlds, configs) intact. Handy after changing an egg or recovering from a broken image.
- **Backups tab** — take an on-demand snapshot of a server's files (compressed on the node with tar + zstd), then restore or delete any snapshot from a list.
- **Scheduled backups** — set an interval (in hours) in the Settings tab and the panel snapshots the server automatically on that cadence.
- **Per-server Activity log** — every power action, settings change, reinstall, delete, and backup is recorded and shown on a new Activity tab, with timestamps.

### 🛠 Fixes

- The Memory (and other numeric) inputs no longer reject values like `3232` with "please enter a value greater than…". They snap to whole numbers now instead of forcing multiples of 128.
- Minecraft eggs no longer carry a duplicate `Memory` variable next to the panel's own Memory field — the panel injects the JVM heap size from the server's Memory setting directly, so there was no second place to set it.

### 📦 Requires

- **sky-daemon v0.2.0** for the backup features (`Back up now`, restore, delete, and scheduled backups). Update your nodes with `sudo sky-panel-update`. CPU limits, reinstall, settings, and the activity log work with any daemon version.

## [0.4.0] - 2026-07-01

### 🚀 Improvements

- A ground-up visual refresh of the panel UI, still strictly black-and-white and still over the animated node-mesh background — but with a proper "control instrument" design language: layered precision-panel surfaces, registration-mark corner ticks on cards, a monospace spec-sheet voice for labels, an accent bar on the active nav item, refined tables/inputs/buttons with real focus states, and an orchestrated fade-in on every page.
- The marketing site got the same treatment: a stronger hero with an animated sweep line, a spec strip, cohesive corner-ticked feature cards, refreshed copy for the v0.3 feature set, and a new closing call-to-action — all working in both light and dark.

### 🛠 Fixes

- The install commands in the docs and installer README are now single-line (`curl … | sudo bash -s -- …`) so they can't be accidentally pasted as one mangled line — the previous download-then-run two-liner silently broke when the newline was dropped on paste.

## [0.3.1] - 2026-07-01

### 🛠 Fixes

- `sky-panel-update` always failed checksum verification on both panel-api and sky-daemon updates. It downloaded each binary to a local filename (e.g. `panel-api`) that didn't match the name recorded in the release's `checksums.txt` (e.g. `panel-api-linux-amd64`), so `sha256sum -c` could never find the file it was asked to verify. It now looks up the expected hash by the release asset's name and checks it against the actual local filename.

Since `sky-panel-update` doesn't update its own script, this fix only takes effect once you fetch a fresh copy:
```
sudo curl -fsSL https://raw.githubusercontent.com/Notbangbang-dev/sky-panel/main/installer/sky-panel-update -o /usr/local/bin/sky-panel-update
sudo chmod 755 /usr/local/bin/sky-panel-update
sudo sky-panel-update
```

## [0.3.0] - 2026-07-01

### ✨ New Features

- A 10-egg starter catalog ships out of the box: Paper, Vanilla, Spigot, Forge, and Fabric Minecraft servers, a BungeeCord proxy, generic Node.js and Python app runners, a Rust (Facepunch) game server, and a blank custom-image template — all seeded automatically on install.
- Admins can edit an egg's docker image, startup command, and variables after creation (`PUT /api/v1/admin/eggs/{id}`), not just create/delete it.
- The create-server form now shows a real node picker (name/address/online status) instead of asking for a raw node ID, and dynamically renders an input for each of the selected egg's editable variables — pre-filled with its default, so e.g. Minecraft's EULA is agreed to by default without a single manual step.
- A "Nodes" page, visible to every user (not just admins), lists every registered node and whether it's currently connected.
- Admins can turn public registration on or off from the console; the login/register pages respect it automatically.
- The marketing site has a full Docs page (architecture, installing, updating, eggs, file manager/sharing, security) and a light/dark theme toggle.

### 🚀 Improvements

- `docker_image`/`startup` are the only required fields when creating an egg now — a blank startup command is valid for images (like the Minecraft ones) that configure themselves entirely from environment variables.
- Clearer documentation of exactly what `sky-panel-update` checks and does, in both the installer README and the new Docs page.
- Sharper GitHub descriptions and READMEs for both `sky-panel` and `sky-daemon`.

## [0.2.1] - 2026-07-01

### 🛠 Fixes

- The released web build was hardcoding `http://localhost:8080` as the API URL (a build-time default that only ever worked on the machine building it, not a deployed server) — every real deployment of v0.2.0 failed to register/log in with a generic "Something went wrong." The release build now targets same-origin, matching how Caddy already proxies panel-api and the web assets together. Also fixes the same issue for the real-time WebSocket connection, which can't use a relative URL the way a same-origin `fetch()` can.
- `install.sh`'s sibling files (systemd units, `sky-panel-update`) are now fetched from the repo instead of assumed to sit next to a curl-downloaded `install.sh`, matching how it's actually documented to be installed.

If you installed v0.2.0 and hit a "Something went wrong" on register/login, re-run `sudo bash install.sh panel` (after `sudo systemctl stop sky-panel` first, since install.sh doesn't stop the service before replacing its binary) to pick up this fix.

## [0.2.0] - 2026-07-01

### ✨ New Features

- File manager: list/read/write/rename/delete/mkdir on a server's volume, exposed under `/api/v1/servers/{id}/files/*` and a new "Files" tab on the server detail page.
- Subusers: server owners can grant other accounts scoped `console`/`files`/`power`/`settings` access under a new "Sharing" tab; every server-scoped endpoint now authorizes owner-or-admin-or-matching-subuser instead of owner-or-admin only.
- Admin "Rotate token" action per node, backed by a new `POST /api/v1/admin/nodes/{id}/rotate-token` endpoint.

### 🚀 Improvements

- The node daemon is rewritten in **Rust** and moved to its own repository, [Notbangbang-dev/sky-daemon](https://github.com/Notbangbang-dev/sky-daemon), for lower resource usage and a smaller attack surface — see that repo's changelog for what's new there.
- `docs/ARCHITECTURE.md` documenting the signed panel↔daemon wire protocol.

### ⚠ Breaking Changes

- `node-agent/` and `skyperf/` are removed from this monorepo. `install.sh` and `sky-panel-update` now fetch `sky-daemon` from its own releases, independent of panel-api/web's version. Existing node installs: the systemd unit is renamed `sky-node-agent` → `sky-daemon` and its env file `node-agent.env` → `sky-daemon.env`; re-run `install.sh node` to pick up the new binary and unit.

### 🔒 Security

- Panel↔daemon protocol hardened: every message after the initial hello is signed (HMAC-SHA256) and carries a timestamp + nonce, closing a gap where the WebSocket accepted any message from an authenticated node without verifying it hadn't been tampered with or replayed.
- Per-connection rate limiting on the panel side of the agent WebSocket.
- Node tokens now expire (90 days by default) and are checked on every connection attempt; admins can rotate a node's token without recreating the node.

## [0.1.0] - 2026-07-01

### Added

- **panel-api** (Go): JWT + TOTP auth, SQLite storage, users/roles, nodes, eggs, servers with port allocation and startup-command templating, coin ledger with AFK-heartbeat and daily-reward accrual, admin console (users, nodes, eggs, settings, audit log, broadcast), and a WebSocket hub for real-time server stats/console/broadcasts.
- **node-agent** (Go): persistent outbound WebSocket connection to panel-api (no inbound ports required), a `ContainerRuntime` abstraction backed by a hand-rolled Docker Engine API client (create/start/stop/kill/remove/inspect/stats/attach) with a fake implementation for tests.
- **skyperf** (Rust): a small perf-sensitive CLI — `dirsize`, `backup create/restore` (tar+zstd, with a path-traversal guard on restore), and `tail --follow`.
- **web** (React + TypeScript): full panel UI — auth, dashboard, server list/detail with a live xterm.js console and real-time stats, AFK page, wallet, account/2FA, a live theme builder (black/white "Monochrome" default + "Signal" accent preset, fully custom themes persisted to `localStorage`), an ambient animated "server mesh" background, and an admin console.
- **installer/**: `install.sh` (panel/node/all modes, Docker + Caddy automatic HTTPS) and `sky-panel-update` for in-place updates with checksum verification.
- **site** (Next.js): marketing site deployed to Vercel at [skypanel-app.vercel.app](https://skypanel-app.vercel.app), with a live changelog page pulled from this file.
- CI (build/vet/test across all five components), a release workflow cross-compiling linux/amd64+arm64 binaries, and a Discord changelog webhook workflow.
