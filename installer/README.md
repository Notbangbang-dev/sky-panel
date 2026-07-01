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
