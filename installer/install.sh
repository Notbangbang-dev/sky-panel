#!/usr/bin/env bash
# Sky Panel installer.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Notbangbang-dev/sky-panel/main/installer/install.sh | sudo bash -s -- panel --domain panel.example.com
#   curl -fsSL https://raw.githubusercontent.com/Notbangbang-dev/sky-panel/main/installer/install.sh | sudo bash -s -- node --panel-url wss://panel.example.com/agent/ws --node-token <TOKEN>
#
# Modes:
#   panel   installs panel-api + the web UI behind Caddy (automatic HTTPS)
#   node    installs node-agent + Docker on a game-server VPS
#   all     installs both on a single box (fine for a first, single-node setup)
set -euo pipefail

REPO="Notbangbang-dev/sky-panel"
INSTALL_DIR="/opt/sky-panel"
SERVICE_USER="sky-panel"

MODE="${1:-}"
shift || true

DOMAIN=""
PANEL_URL=""
NODE_TOKEN=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --domain) DOMAIN="$2"; shift 2 ;;
    --panel-url) PANEL_URL="$2"; shift 2 ;;
    --node-token) NODE_TOKEN="$2"; shift 2 ;;
    *) echo "unknown argument: $1" >&2; exit 1 ;;
  esac
done

log() { echo "==> $*"; }
fail() { echo "error: $*" >&2; exit 1; }

require_root() {
  if [[ "$(id -u)" -ne 0 ]]; then
    fail "this installer must be run as root (try: sudo bash install.sh ...)"
  fi
}

detect_arch() {
  case "$(uname -m)" in
    x86_64|amd64) echo "amd64" ;;
    aarch64|arm64) echo "arm64" ;;
    *) fail "unsupported architecture: $(uname -m)" ;;
  esac
}

require_apt() {
  command -v apt-get >/dev/null 2>&1 || fail "this installer currently only supports Debian/Ubuntu (apt-get not found)"
}

install_docker_if_missing() {
  if command -v docker >/dev/null 2>&1; then
    log "Docker already installed, skipping"
    return
  fi
  log "installing Docker"
  curl -fsSL https://get.docker.com | sh
  systemctl enable --now docker
}

create_service_user() {
  if ! id "$SERVICE_USER" >/dev/null 2>&1; then
    log "creating service user $SERVICE_USER"
    useradd --system --no-create-home --shell /usr/sbin/nologin "$SERVICE_USER"
  fi
}

random_secret() {
  head -c 48 /dev/urandom | base64 | tr -dc 'a-zA-Z0-9' | head -c 48
}

latest_release_tag() {
  curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep -m1 '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/'
}

download_release_asset() {
  local tag="$1" asset="$2" dest="$3"
  curl -fsSL "https://github.com/${REPO}/releases/download/${tag}/${asset}" -o "$dest"
}

install_panel() {
  require_apt
  local arch tag
  arch="$(detect_arch)"
  tag="$(latest_release_tag)"
  [[ -n "$tag" ]] || fail "could not determine the latest sky-panel release"

  log "installing panel-api (release ${tag}, linux/${arch})"
  mkdir -p "$INSTALL_DIR/bin" "$INSTALL_DIR/data" "$INSTALL_DIR/web"
  create_service_user

  download_release_asset "$tag" "panel-api-linux-${arch}" "$INSTALL_DIR/bin/panel-api"
  chmod +x "$INSTALL_DIR/bin/panel-api"

  download_release_asset "$tag" "web-dist.tar.gz" "/tmp/sky-panel-web.tar.gz"
  tar -xzf /tmp/sky-panel-web.tar.gz -C "$INSTALL_DIR/web"
  rm -f /tmp/sky-panel-web.tar.gz

  if [[ ! -f "$INSTALL_DIR/panel-api.env" ]]; then
    log "generating panel-api.env"
    cat > "$INSTALL_DIR/panel-api.env" <<EOF
SKY_HTTP_ADDR=127.0.0.1:8080
SKY_DB_PATH=${INSTALL_DIR}/data/sky-panel.db
SKY_JWT_ACCESS_SECRET=$(random_secret)
SKY_JWT_REFRESH_SECRET=$(random_secret)
EOF
  fi

  echo "$tag" > "$INSTALL_DIR/VERSION"
  chown -R "$SERVICE_USER":"$SERVICE_USER" "$INSTALL_DIR"

  install -m 644 "$(dirname "$0")/systemd/sky-panel.service" /etc/systemd/system/sky-panel.service
  install -m 755 "$(dirname "$0")/sky-panel-update" /usr/local/bin/sky-panel-update
  systemctl daemon-reload
  systemctl enable --now sky-panel

  install_caddy_reverse_proxy
  local address="${DOMAIN:-this-server}"
  log "panel-api installed. Visit https://${address} and register the first account (it becomes admin)."
}

install_caddy_reverse_proxy() {
  if ! command -v caddy >/dev/null 2>&1; then
    log "installing Caddy"
    apt-get update -y
    apt-get install -y debian-keyring debian-archive-keyring apt-transport-https curl
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/gpg.key' | gpg --dearmor -o /usr/share/keyrings/caddy-stable-archive-keyring.gpg
    curl -1sLf 'https://dl.cloudsmith.io/public/caddy/stable/debian.deb.txt' | tee /etc/apt/sources.list.d/caddy-stable.list
    apt-get update -y
    apt-get install -y caddy
  fi

  local site="${DOMAIN:-:80}"
  cat > /etc/caddy/Caddyfile <<EOF
${site} {
    handle /api/* {
        reverse_proxy 127.0.0.1:8080
    }
    handle /ws {
        reverse_proxy 127.0.0.1:8080
    }
    handle /agent/ws {
        reverse_proxy 127.0.0.1:8080
    }
    handle {
        root * ${INSTALL_DIR}/web
        try_files {path} /index.html
        file_server
    }
}
EOF
  systemctl enable --now caddy
  systemctl reload caddy
}

install_node() {
  require_apt
  [[ -n "$PANEL_URL" ]] || fail "--panel-url is required for node mode"
  [[ -n "$NODE_TOKEN" ]] || fail "--node-token is required for node mode (create the node from the admin console first)"

  local arch tag
  arch="$(detect_arch)"
  tag="$(latest_release_tag)"
  [[ -n "$tag" ]] || fail "could not determine the latest sky-panel release"

  install_docker_if_missing

  log "installing node-agent (release ${tag}, linux/${arch})"
  mkdir -p "$INSTALL_DIR/bin"
  download_release_asset "$tag" "node-agent-linux-${arch}" "$INSTALL_DIR/bin/node-agent"
  download_release_asset "$tag" "skyperf-linux-${arch}" "$INSTALL_DIR/bin/skyperf"
  chmod +x "$INSTALL_DIR/bin/node-agent" "$INSTALL_DIR/bin/skyperf"

  cat > "$INSTALL_DIR/node-agent.env" <<EOF
SKY_PANEL_WS_URL=${PANEL_URL}
SKY_NODE_TOKEN=${NODE_TOKEN}
SKY_DOCKER_SOCKET=/var/run/docker.sock
EOF
  chmod 600 "$INSTALL_DIR/node-agent.env"

  echo "$tag" > "$INSTALL_DIR/VERSION"
  install -m 644 "$(dirname "$0")/systemd/sky-node-agent.service" /etc/systemd/system/sky-node-agent.service
  install -m 755 "$(dirname "$0")/sky-panel-update" /usr/local/bin/sky-panel-update
  systemctl daemon-reload
  systemctl enable --now sky-node-agent

  log "node-agent installed and connecting to ${PANEL_URL}"
}

require_root

case "$MODE" in
  panel) install_panel ;;
  node) install_node ;;
  all)
    install_panel
    PANEL_URL="${PANEL_URL:-ws://127.0.0.1:8080/agent/ws}"
    if [[ -z "$NODE_TOKEN" ]]; then
      fail "--node-token is required for 'all' mode too (register the first admin account, create a node from the admin console, then re-run with --node-token)"
    fi
    install_node
    ;;
  *)
    fail "usage: install.sh <panel|node|all> [--domain ...] [--panel-url ...] [--node-token ...]"
    ;;
esac
