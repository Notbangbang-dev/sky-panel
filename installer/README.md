# Sky Panel installer

Targets Ubuntu/Debian VPS hosts.

## Install the panel (control plane + web UI)

```bash
curl -fsSL https://raw.githubusercontent.com/Notbangbang-dev/sky-panel/main/installer/install.sh -o install.sh
sudo bash install.sh panel --domain panel.example.com
```

Caddy is installed automatically and gets you HTTPS for free. Visit the
domain and register — the first account created becomes admin.

## Install a node (game-server host)

From the admin console, create a node to get its one-time node token, then
on that VPS:

```bash
sudo bash install.sh node --panel-url wss://panel.example.com/agent/ws --node-token <TOKEN>
```

This installs Docker (if missing) and `node-agent`, which dials out to the
panel — no inbound ports need to be opened on the node.

## Single-box setup

```bash
sudo bash install.sh all --domain panel.example.com --node-token <TOKEN>
```

Run it once without `--node-token` to get the panel up, create a node from
the admin console, then re-run with the token to also install the agent on
the same box.

## Updating

Every install places `sky-panel-update` at `/usr/local/bin`. Run it any time
to pull the latest release, verify checksums, and restart whichever Sky
Panel services are present on that box:

```bash
sudo sky-panel-update
```
