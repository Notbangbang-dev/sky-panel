# Sky Panel

A self-hosted game-server hosting panel that doesn't get in your way. Go control plane, Rust daemon, React/TypeScript frontend, real multi-node Docker orchestration, a 10-egg starter catalog (Minecraft, generic Node.js/Python, and more), live stats, a coin economy with an AFK page, and a full admin console.

**Website:** https://skypanel-app.vercel.app · **Docs:** https://skypanel-app.vercel.app/docs

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
cd panel-api && go build ./... && go vet ./... && go test ./...

# web
cd web && npm install && npm run typecheck && npm test && npm run dev

# site
cd site && npm install && npm run dev
```

> Running panel-api locally? Either set `SKY_JWT_ACCESS_SECRET` / `SKY_JWT_REFRESH_SECRET` to strong random values, or export `SKY_DEV_MODE=1` — as of v0.24.0 the server refuses to start on the built-in default secrets (see below).

## Configuration (panel-api)

All configuration is via environment variables (systemd `EnvironmentFile` in production):

| Variable | Default | Notes |
| --- | --- | --- |
| `SKY_HTTP_ADDR` | `:8080` | Listen address. |
| `SKY_DB_PATH` | `sky-panel.db` | SQLite path (WAL mode is enabled automatically for file DBs). |
| `SKY_JWT_ACCESS_SECRET` | — | **Required in production.** ≥32 random chars (`openssl rand -hex 32`). Boot fails if unset/default/short. |
| `SKY_JWT_REFRESH_SECRET` | — | **Required in production.** As above. |
| `SKY_ACCESS_TTL` | `15m` | Access-token lifetime. |
| `SKY_REFRESH_TTL` | `720h` | Refresh-token lifetime. |
| `SKY_CORS_ORIGIN` | `*` | Pin `Access-Control-Allow-Origin` to your panel's origin. |
| `SKY_DEV_MODE` | `false` | Relaxes the secret-strength check for local dev. **Never set in production.** |

The bundled `installer/install.sh` generates strong secrets, verifies release binaries against published SHA-256 checksums, and configures Caddy with a Content-Security-Policy and HTTPS. `sky-panel-update` updates in place (with `--rollback`), and `uninstall.sh` removes an install cleanly. See [SECURITY.md](SECURITY.md) for the hardening checklist and how to report a vulnerability.

## Status

Early, actively developed. See [CHANGELOG.md](CHANGELOG.md).
