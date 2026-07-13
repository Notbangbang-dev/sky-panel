# Security Policy

Sky Panel is a self-hosted game-server hosting panel. It stores node tokens,
issues JWTs, can provision databases, and orchestrates untrusted user
workloads across multiple machines. Security matters here, and reports are
taken seriously.

## Supported versions

Only the current minor series receives security fixes. Sky Panel ships fast;
please stay on the latest release.

| Version  | Supported          |
| -------- | ------------------ |
| 0.24.x   | :white_check_mark: |
| < 0.24   | :x:                |

## Reporting a vulnerability

**Do not open a public GitHub issue for security vulnerabilities.** A public
issue discloses the problem to everyone running a panel before a fix exists.

Instead, report it privately through GitHub Security Advisories:

> https://github.com/Notbangbang-dev/sky-panel/security/advisories/new

Please include, as far as you can:

- The affected component (panel-api, web, site, or the sky-daemon) and version.
- A description of the issue and its impact (what an attacker can do).
- Reproduction steps or a proof of concept.
- Any suggested remediation.

If the vulnerability lives in the node agent rather than the panel, note that
the Rust daemon is a **separate repository**
([Notbangbang-dev/sky-daemon](https://github.com/Notbangbang-dev/sky-daemon));
you can file the advisory there, or here and we'll route it.

### Response expectations

Sky Panel is a hobby / open-source project maintained on a best-effort basis.
There is no paid support and no formal SLA. That said, security reports jump
the queue: expect an initial acknowledgement within a few days, and a fix or a
plan as fast as is practical. Please give us a reasonable window to ship a fix
before public disclosure, and we'll credit you in the advisory unless you'd
rather stay anonymous.

## Scope

Reports that are in scope and especially valued:

- **Authentication / authorization** in panel-api — JWT issuance and
  validation, TOTP, session/refresh handling, role checks, and any path that
  lets one user act as another or reach admin-only functionality.
- **Node token handling** — leakage, forgery, or misuse of the tokens that
  authenticate the panel to a daemon (or a daemon to the panel), and anything
  that lets an attacker impersonate a node or drive commands to one.
- **Subuser and permission bypass** — a subuser reaching actions or servers
  they weren't granted.
- **Coin economy abuse** — races, negative balances, quota bypass, or any way
  to mint resources (memory/CPU/disk/database slots) without paying for them.
- **File / backup path handling** — path traversal, symlink escapes, or reads
  and writes outside a server's own volume via the file manager or backups.
- **Database provisioning** — injection into generated database/user names,
  cross-tenant database access, or credential leakage.

Out of scope (unless chained into one of the above):

- Vulnerabilities requiring physical access to a node or panel host, or
  requiring pre-existing root/admin on those boxes.
- Missing hardening that the operator is responsible for (see below) —
  e.g. running with the insecure default JWT secrets, or exposing the daemon
  to the public internet.
- Denial of service from an authenticated operator against their own
  infrastructure.
- Reports against unsupported versions.

## Hardening checklist for operators

The panel ships with safe-*by-configuration* defaults, but a few things are
yours to get right. At minimum:

- [ ] **Set `SKY_JWT_ACCESS_SECRET` and `SKY_JWT_REFRESH_SECRET`** to long,
      random, distinct values. The built-in defaults
      (`dev-access-secret-change-me` / `dev-refresh-secret-change-me`) are for
      local development only — leaving them in place lets anyone forge tokens.
- [ ] **Run the panel behind HTTPS** (a reverse proxy such as Caddy, nginx, or
      Traefik with a valid certificate). Never serve the API or the web UI over
      plaintext HTTP on an untrusted network — tokens travel in every request.
- [ ] **Keep each daemon box firewalled.** The daemon should only accept
      connections from the panel, not the public internet. Restrict its port to
      the panel's address at the host firewall and/or your cloud provider's
      security group.
- [ ] **Rotate node tokens** periodically, and immediately if one may have
      leaked (logs, screenshots, a compromised host). Treat a node token like a
      root credential for that box.
- [ ] Keep the OS, Docker, and Sky Panel itself up to date, and back up the
      panel database.
