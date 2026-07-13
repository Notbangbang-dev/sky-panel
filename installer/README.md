# Sky Panel installer

Targets Ubuntu/Debian VPS hosts.

Each command is a single line — pipe the installer straight into `bash` so
there's nothing to copy wrong. (Downloading it to a file and running it in a
second step also works, but is easy to fumble into one mangled line.)

## Install the panel (control plane + web UI)

```bash
curl -fsSL https://raw.githubusercontent.com/Notbangbang-dev/sky-panel/main/installer/install.sh | sudo bash -s -- panel --domain panel.example.com
```

Caddy is installed automatically and gets you HTTPS for free. Visit the
domain and register — the first account created becomes admin.

Every release binary (and the web bundle) is verified against the release's
published `checksums.txt` before it's made executable or extracted, so a
corrupted or tampered download is rejected. Releases old enough to predate
published checksums simply skip the check with a warning rather than failing.

Caddy is also configured to send a strict set of security headers
(Content-Security-Policy, `X-Content-Type-Options`, `X-Frame-Options`,
`Referrer-Policy`) on the panel's static/SPA responses. These are scoped so
they never touch the `/api/*`, `/ws`, or `/agent/ws` proxied routes.

Don't have a domain pointed at the box yet? Drop `--domain panel.example.com`
entirely — Caddy serves plain HTTP instead of trying (and failing) to get a
certificate for a domain it can't verify. You can always re-run the
installer later once you do have one; it's safe to run more than once (see
[Updating](#updating)).

## Install a node (game-server host)

From the admin console, create a node to get its one-time node token, then
on that VPS:

```bash
curl -fsSL https://raw.githubusercontent.com/Notbangbang-dev/sky-panel/main/installer/install.sh | sudo bash -s -- node --panel-url wss://panel.example.com/agent/ws --node-token <TOKEN>
```

**Keeping the token off the command line (preferred on shared hosts).** The
node token is a secret, and `--node-token <TOKEN>` leaves it in your shell
history and `ps`. Pass it via the `SKY_NODE_TOKEN` environment variable
instead — `--node-token` still works, but the env form is recommended:

```bash
export SKY_NODE_TOKEN=<TOKEN>
curl -fsSL https://raw.githubusercontent.com/Notbangbang-dev/sky-panel/main/installer/install.sh | sudo -E bash -s -- node --panel-url wss://panel.example.com/agent/ws
```

(`sudo -E` preserves the variable through sudo.) If neither is given and you're
running interactively, the installer prompts for the token without echoing it.

This installs Docker (if missing) and [`sky-daemon`](https://github.com/Notbangbang-dev/sky-daemon)
(a separate Rust binary, released from its own repo), which dials out to the
panel — no inbound ports need to be opened on the node.

## Single-box setup

```bash
curl -fsSL https://raw.githubusercontent.com/Notbangbang-dev/sky-panel/main/installer/install.sh | sudo bash -s -- all --domain panel.example.com --node-token <TOKEN>
```

Run it once without `--node-token` to get the panel up, create a node from
the admin console, then re-run with the token to also install the agent on
the same box.

## Updating

Every install places `sky-panel-update` at `/usr/local/bin`. Run it any time:

```bash
sudo sky-panel-update
```

**What it actually does:** panel-api/web and sky-daemon are separate
GitHub releases with independent version numbers — the panel doesn't wait
on a daemon release and vice versa. `sky-panel-update` checks each half
independently, tracked by two separate version files on disk
(`/opt/sky-panel/VERSION` for panel-api/web, `/opt/sky-panel/VERSION-daemon`
for sky-daemon):

1. For each half that's installed on this box, it asks GitHub for that
   repo's latest release tag and compares it against the recorded version.
2. If they match, it prints `already up to date` for that half and moves on
   — no download, no restart.
3. If they differ, it downloads the new binary (and the web assets, for
   panel-api), verifies it against the release's published
   `checksums.txt`, stops the relevant systemd service, swaps the binary
   in, and restarts it.
4. It prints that release's changelog entry once the swap is done.

A box running only `panel` or only `node` simply has nothing to do for the
half it doesn't have installed — it isn't an error, it's just skipped.

### Rolling back a bad update

Each update keeps the previous binary/web as `.bak` before swapping. If a new
release misbehaves, roll straight back:

```bash
sudo sky-panel-update --rollback
```

This restores the previous `panel-api` / web dir and/or `sky-daemon` from their
`.bak` copies, re-applies file ownership, and restarts the affected service(s).
It prints what it restored.

## Configuration (panel-api.env)

The installer writes `/opt/sky-panel/panel-api.env` on first install and
generates strong `SKY_JWT_ACCESS_SECRET` and `SKY_JWT_REFRESH_SECRET` values
(64 hex chars each). **panel-api refuses to start** if either is unset, left at
a built-in default, or shorter than 32 characters — so keep them set to long,
unique random secrets. Rotating them logs everyone out.

Two optional variables are documented (commented out) in the generated file:

- `SKY_CORS_ORIGIN` — pin CORS to your panel's exact origin
  (e.g. `https://panel.example.com`) instead of the permissive default.
  Recommended once you know your URL.
- `SKY_DEV_MODE=1` — relaxes the secret-strength boot checks. **Local
  development only — never set this in production.**

## Uninstalling

To remove Sky Panel from a box:

```bash
sudo bash uninstall.sh              # stop/disable services, remove the install dir — KEEPS your data
sudo bash uninstall.sh --purge-data # also delete /opt/sky-panel/data (irreversible)
```

It stops and disables the `sky-panel` / `sky-daemon` services, removes their
unit files and the `sky-panel-update` helper, and removes `/opt/sky-panel`. The
data directory (`/opt/sky-panel/data`, your panel database) is **kept by
default** — deleting it requires the explicit `--purge-data` flag. The script is
idempotent (safe to re-run) and leaves the `sky-panel` user, Docker, and Caddy
in place since those may be shared with other software.
