# Changelog

All notable changes to Sky Panel are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

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
