# Contributing to Sky Panel

Thanks for helping build Sky Panel. This repo is the control plane and
frontends; the node agent (**sky-daemon**, Rust) lives in a separate repo:
[Notbangbang-dev/sky-daemon](https://github.com/Notbangbang-dev/sky-daemon).
Changes to the daemon go there.

## Repository layout

| Directory    | What it is                                                        |
| ------------ | ----------------------------------------------------------------- |
| `panel-api/` | Go control plane — auth, users, nodes, eggs, servers, coins, admin |
| `web/`       | React / TypeScript panel frontend                                  |
| `site/`      | Next.js marketing / docs site                                      |
| `installer/` | Install scripts                                                    |
| `docs/`      | Operator documentation                                             |

## Building and testing

Please run the full check for **every component you touch** before opening a
PR. CI runs exactly these; getting them green locally saves a round trip.

### panel-api (Go 1.25)

From `panel-api/`:

```sh
go build ./...
go vet ./...
go test ./...        # CI runs this with -race
```

CI runs the tests with `-race`; run `go test ./... -race` yourself if your
change touches concurrency (the coin ledger, WebSocket hub, node scheduling,
port claims, database provisioning, and so on).

### web (React / TypeScript)

From `web/`:

```sh
npm ci
npm run typecheck
npm test
npm run build
```

### site (Next.js)

From `site/`:

```sh
npm ci
npm run build       # CI also runs `npm run lint`
```

## Branches and pull requests

- Branch off `main`. Use a short, descriptive branch name
  (`feat/multi-port`, `fix/coin-race`).
- Keep a PR focused on one thing. Split unrelated changes.
- **CI must be green** before a PR is merged — no exceptions. Fix the build,
  don't skip the check.
- Fill out the pull request template (tests pass, changelog updated, no
  secrets committed, style followed).
- Never commit secrets, tokens, `.env` files, or the panel database.

## Commit and changelog conventions

Commit messages use conventional-commit prefixes with a version suffix in
parentheses. Match the existing history:

```
feat: multiple ports per server (v0.24.0)
fix: prevent coin balance going negative under concurrent spends (v0.24.0)
```

Common prefixes: `feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`.

User-facing changes go in [`CHANGELOG.md`](CHANGELOG.md), which follows the
[Keep a Changelog](https://keepachangelog.com/en/1.1.0/) format. Add your entry
under the current unreleased version heading, in the appropriate section
(New Features, Fixes, Hardening, Requires, ...). If a change needs a specific
sky-daemon version, call it out under a **Requires** note like the existing
entries do.

## Security

Found a vulnerability? **Do not open a public issue.** See
[SECURITY.md](SECURITY.md) for private reporting via GitHub Security Advisories.
