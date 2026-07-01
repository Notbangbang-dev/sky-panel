# Sky Panel

A game-server hosting panel. Go + Rust backend, React/TypeScript frontend, real multi-node Docker orchestration, live stats, a coin economy with an AFK page, and a full admin console.

**Website:** https://skypanel-app.vercel.app

## Stack

- **panel-api** — Go control plane: auth (JWT + TOTP), users/roles, nodes, eggs, servers, coins/ledger, admin, WebSocket hub. SQLite (pure-Go driver, no cgo).
- **node-agent** — Go daemon that runs on each VPS node. Connects outbound to panel-api over an authenticated WebSocket (no inbound ports required on the node) and drives Docker via the Docker Engine API.
- **skyperf** — a small Rust CLI for the genuinely perf-sensitive slice: directory sizing, backup archive create/restore (tar+zstd), and log tailing. Everything else is Go.
- **web** — React + TypeScript (Vite) panel UI. Black/white default theme, a live theme builder, and an animated ambient background.
- **site** — Next.js marketing site (deployed to Vercel).
- **installer** — `install.sh` for a fresh VPS + `sky-panel-update` for in-place updates.

## Repo layout

```
panel-api/     Go control plane
node-agent/    Go node daemon
skyperf/       Rust perf CLI
web/           React panel UI
site/          Next.js marketing site
installer/     install.sh, sky-panel-update, systemd units
```

## Development

```bash
# panel-api
cd panel-api && go build ./... && go test ./...

# node-agent
cd node-agent && go build ./... && go test ./...

# skyperf
cd skyperf && cargo build && cargo test

# web
cd web && npm install && npm run dev

# site
cd site && npm install && npm run dev
```

## Status

Early, actively developed. See [CHANGELOG.md](CHANGELOG.md).
