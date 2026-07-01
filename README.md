# Sky Panel

A game-server hosting panel. Go control plane, Rust daemon, React/TypeScript frontend, real multi-node Docker orchestration, live stats, a coin economy with an AFK page, and a full admin console.

**Website:** https://skypanel-app.vercel.app

## Stack

- **panel-api** — Go control plane: auth (JWT + TOTP), users/roles, nodes, eggs, servers, subusers, coins/ledger, admin, WebSocket hub. SQLite (pure-Go driver, no cgo).
- **[sky-daemon](https://github.com/Notbangbang-dev/sky-daemon)** — the per-node daemon lives in its own repo now, rewritten in Rust for performance, security, and a smaller footprint. It connects outbound to panel-api over a signed WebSocket protocol (no inbound ports required on the node) and drives Docker via the Docker Engine API. See that repo's README and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) here for the wire protocol.
- **web** — React + TypeScript (Vite) panel UI. Black/white default theme, a live theme builder, a file manager, per-server sharing, and an animated ambient background.
- **site** — Next.js marketing site (deployed to Vercel).
- **installer** — `install.sh` for a fresh VPS + `sky-panel-update` for in-place updates. panel-api/web and sky-daemon ship as separate releases and update independently.

## Repo layout

```
panel-api/     Go control plane
web/           React panel UI
site/          Next.js marketing site
installer/     install.sh, sky-panel-update, systemd units
docs/          architecture notes
```

The node daemon (`sky-daemon`, Rust) lives at [Notbangbang-dev/sky-daemon](https://github.com/Notbangbang-dev/sky-daemon).

## Development

```bash
# panel-api
cd panel-api && go build ./... && go test ./...

# web
cd web && npm install && npm run dev

# site
cd site && npm install && npm run dev
```

## Status

Early, actively developed. See [CHANGELOG.md](CHANGELOG.md).
