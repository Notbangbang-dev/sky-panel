# Changelog

All notable changes to Sky Panel are documented here. Format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/).

## [Unreleased] — "First Light"

### ✨ Added

- **panel-api** (Go): JWT + TOTP auth, SQLite storage, users/roles, nodes, eggs, servers with port allocation and startup-command templating, coin ledger with AFK-heartbeat and daily-reward accrual, admin console (users, nodes, eggs, settings, audit log, broadcast), and a WebSocket hub for real-time server stats/console/broadcasts.
- **node-agent** (Go): persistent outbound WebSocket connection to panel-api (no inbound ports required), a `ContainerRuntime` abstraction backed by a hand-rolled Docker Engine API client (create/start/stop/kill/remove/inspect/stats/attach) with a fake implementation for tests.
- **skyperf** (Rust): a small perf-sensitive CLI — `dirsize`, `backup create/restore` (tar+zstd, with a path-traversal guard on restore), and `tail --follow`.
- **web** (React + TypeScript): full panel UI — auth, dashboard, server list/detail with a live xterm.js console and real-time stats, AFK page, wallet, account/2FA, a live theme builder (black/white "Monochrome" default + "Signal" accent preset, fully custom themes persisted to `localStorage`), an ambient animated "server mesh" background, and an admin console.
- **installer/**: `install.sh` (panel/node/all modes, Docker + Caddy automatic HTTPS) and `sky-panel-update` for in-place updates with checksum verification.
- **site** (Next.js): marketing site deployed to Vercel at [skypanel-app.vercel.app](https://skypanel-app.vercel.app), with a live changelog page pulled from this file.
- CI (build/vet/test across all five components), a release workflow cross-compiling linux/amd64+arm64 binaries, and a Discord changelog webhook workflow.
