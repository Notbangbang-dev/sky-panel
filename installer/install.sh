#!/usr/bin/env bash
# Sky Panel installer.
#
# Usage:
#   curl -fsSL https://raw.githubusercontent.com/Notbangbang-dev/sky-panel/main/installer/install.sh | sudo bash -s -- panel --domain panel.example.com
#   curl -fsSL https://raw.githubusercontent.com/Notbangbang-dev/sky-panel/main/installer/install.sh | sudo bash -s -- node --panel-url wss://panel.example.com/agent/ws --node-token <TOKEN>
#
# Modes:
#   panel   installs panel-api + the web UI behind Caddy (automatic HTTPS)
#   node    installs sky-daemon + Docker on a game-server VPS
#   all     installs both on a single box (fine for a first, single-node setup)
#
# panel-api/web and sky-daemon ship as separate GitHub releases (separate
# repos, separate version cadence), so each is tracked and updated
# independently — see sky-panel-update.
set -euo pipefail

REPO="Notbangbang-dev/sky-panel"
DAEMON_REPO="Notbangbang-dev/sky-daemon"
INSTALL_DIR="/opt/sky-panel"
SERVICE_USER="sky-panel"
VOLUMES_ROOT="/srv/sky-panel/volumes"

MODE="${1:-}"
shift || true

DOMAIN=""
PANEL_URL=""
# The node token is a secret. Prefer sourcing it from the environment
# (SKY_NODE_TOKEN) so it never lands in shell/ps history — this is the
# recommended form on shared hosts. The --node-token CLI flag still works for
# compatibility, and if neither is given in node/all mode we fall back to
# reading it from stdin (again keeping it off the command line).
NODE_TOKEN="${SKY_NODE_TOKEN:-}"

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
  # panel-api refuses to boot with unset/default/short JWT secrets (unless
  # SKY_DEV_MODE=1), so generate a strong one: 64 hex chars (32 bytes of
  # entropy). Prefer openssl; fall back to /dev/urandom, still emitting 64
  # hex characters so the result clears panel-api's >=32-char requirement.
  if command -v openssl >/dev/null 2>&1; then
    openssl rand -hex 32
  else
    head -c 32 /dev/urandom | od -An -tx1 | tr -d ' \n'
  fi
}

latest_release_tag() {
  local repo="$1"
  curl -fsSL "https://api.github.com/repos/${repo}/releases/latest" | grep -m1 '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/'
}

download_release_asset() {
  local repo="$1" tag="$2" asset="$3" dest="$4"
  curl -fsSL "https://github.com/${repo}/releases/download/${tag}/${asset}" -o "$dest"
}

# Verify a downloaded release asset against the release's published
# checksums.txt (same mechanism sky-panel-update uses). The checksum is checked
# BEFORE we chmod +x / extract / run the asset. Releases that predate
# checksums.txt simply won't have one — in that case we warn and continue
# rather than hard-failing an install, but we always verify when it's present.
verify_release_asset() {
  local repo="$1" tag="$2" asset="$3" file="$4"
  local base_url="https://github.com/${repo}/releases/download/${tag}"
  local sums expected
  sums="$(mktemp)"
  if ! curl -fsSL "${base_url}/checksums.txt" -o "$sums" 2>/dev/null; then
    log "WARNING: no checksums.txt for ${repo} ${tag} — skipping integrity check for ${asset} (this release may predate published checksums)"
    rm -f "$sums"
    return 0
  fi
  # checksums.txt records hashes against the release asset's own name
  # (e.g. panel-api-linux-amd64), which usually differs from the local
  # filename we saved it as — rewrite the checksum line to reference the
  # local file so sha256sum -c can find it.
  expected="$(grep " ${asset}\$" "$sums" | awk '{print $1}')"
  rm -f "$sums"
  if [[ -z "$expected" ]]; then
    log "WARNING: checksums.txt for ${repo} ${tag} has no entry for ${asset} — skipping integrity check"
    return 0
  fi
  echo "${expected}  $(basename "$file")" | ( cd "$(dirname "$file")" && sha256sum -c - ) \
    || fail "checksum verification failed for ${asset}"
  log "verified ${asset} against checksums.txt"
}

# install.sh is designed to be curl'd on its own (see the usage comment at
# the top) rather than requiring a full clone, so its sibling files
# (systemd units, sky-panel-update) are fetched straight from the repo at
# install time instead of being expected to sit next to this script.
fetch_repo_file() {
  local path="$1" dest="$2"
  curl -fsSL "https://raw.githubusercontent.com/${REPO}/main/installer/${path}" -o "$dest"
}

install_panel() {
  require_apt
  local arch tag
  arch="$(detect_arch)"
  tag="$(latest_release_tag "$REPO")"
  [[ -n "$tag" ]] || fail "could not determine the latest sky-panel release"

  log "installing panel-api (release ${tag}, linux/${arch})"
  mkdir -p "$INSTALL_DIR/bin" "$INSTALL_DIR/data" "$INSTALL_DIR/web"
  create_service_user

  # Stop before overwriting the binary in place — re-running this against
  # an already-running install (e.g. to fix a bad --domain) can otherwise
  # fail the download's write with the service still holding it open.
  systemctl stop sky-panel 2>/dev/null || true

  download_release_asset "$REPO" "$tag" "panel-api-linux-${arch}" "$INSTALL_DIR/bin/panel-api"
  verify_release_asset "$REPO" "$tag" "panel-api-linux-${arch}" "$INSTALL_DIR/bin/panel-api"
  chmod +x "$INSTALL_DIR/bin/panel-api"

  download_release_asset "$REPO" "$tag" "web-dist.tar.gz" "/tmp/sky-panel-web.tar.gz"
  verify_release_asset "$REPO" "$tag" "web-dist.tar.gz" "/tmp/sky-panel-web.tar.gz"
  tar -xzf /tmp/sky-panel-web.tar.gz -C "$INSTALL_DIR/web"
  rm -f /tmp/sky-panel-web.tar.gz

  if [[ ! -f "$INSTALL_DIR/panel-api.env" ]]; then
    log "generating panel-api.env"
    cat > "$INSTALL_DIR/panel-api.env" <<EOF
SKY_HTTP_ADDR=127.0.0.1:8080
SKY_DB_PATH=${INSTALL_DIR}/data/sky-panel.db
# panel-api refuses to start unless these are set to long, unique random
# secrets (>=32 chars). Generated here as 64 hex chars. Do NOT reuse across
# installs, and rotating them logs everyone out.
SKY_JWT_ACCESS_SECRET=$(random_secret)
SKY_JWT_REFRESH_SECRET=$(random_secret)
# Optional: pin CORS to your panel's exact origin instead of the permissive
# default (recommended once you know your URL), e.g.
# SKY_CORS_ORIGIN=https://panel.example.com
# Optional: relaxes the secret-strength boot checks. For LOCAL DEVELOPMENT
# ONLY — never set this in production.
# SKY_DEV_MODE=1
EOF
    chmod 600 "$INSTALL_DIR/panel-api.env"
  fi

  echo "$tag" > "$INSTALL_DIR/VERSION"
  chown -R "$SERVICE_USER":"$SERVICE_USER" "$INSTALL_DIR"

  fetch_repo_file "systemd/sky-panel.service" /etc/systemd/system/sky-panel.service
  chmod 644 /etc/systemd/system/sky-panel.service
  fetch_repo_file "sky-panel-update" /usr/local/bin/sky-panel-update
  chmod 755 /usr/local/bin/sky-panel-update
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
        # Security headers on the SPA/static responses only. These are inside
        # the fall-through `handle` (not the /api/*, /ws or /agent/ws proxy
        # blocks above), so the CSP can't break API calls or WebSockets.
        # The app uses inline styles (unsafe-inline in style-src) and loads
        # Google Fonts (fonts.googleapis.com stylesheet + fonts.gstatic.com
        # font files); its own bundled JS is same-origin ('self'), and its
        # same-origin API/WS calls are covered by connect-src 'self' + ws/wss.
        header {
            Content-Security-Policy "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline' https://fonts.googleapis.com; img-src 'self' data: https:; media-src 'self' https:; font-src 'self' https://fonts.gstatic.com; connect-src 'self' https: wss: ws:; frame-ancestors 'none'; base-uri 'self'"
            X-Content-Type-Options "nosniff"
            X-Frame-Options "DENY"
            Referrer-Policy "strict-origin-when-cross-origin"
        }
        root * ${INSTALL_DIR}/web
        try_files {path} /index.html
        file_server
    }
}
EOF
  systemctl enable --now caddy
  systemctl reload caddy
}

# Resolve the node token, preferring forms that keep it out of shell/ps
# history: SKY_NODE_TOKEN env (already folded into $NODE_TOKEN above) or
# --node-token, else read it from stdin if we have a terminal to prompt on.
resolve_node_token() {
  if [[ -z "$NODE_TOKEN" && -t 0 ]]; then
    read -rsp "Node token (from the admin console): " NODE_TOKEN
    echo >&2
  fi
}

install_node() {
  require_apt
  [[ -n "$PANEL_URL" ]] || fail "--panel-url is required for node mode"
  resolve_node_token
  [[ -n "$NODE_TOKEN" ]] || fail "node token is required for node mode — set SKY_NODE_TOKEN (preferred on shared hosts), pass --node-token, or run interactively to be prompted (create the node from the admin console first)"

  local arch tag
  arch="$(detect_arch)"
  tag="$(latest_release_tag "$DAEMON_REPO")"
  [[ -n "$tag" ]] || fail "could not determine the latest sky-daemon release"

  install_docker_if_missing

  log "installing sky-daemon (release ${tag}, linux/${arch})"
  mkdir -p "$INSTALL_DIR/bin" "$VOLUMES_ROOT"
  systemctl stop sky-daemon 2>/dev/null || true
  download_release_asset "$DAEMON_REPO" "$tag" "sky-daemon-linux-${arch}" "$INSTALL_DIR/bin/sky-daemon"
  verify_release_asset "$DAEMON_REPO" "$tag" "sky-daemon-linux-${arch}" "$INSTALL_DIR/bin/sky-daemon"
  chmod +x "$INSTALL_DIR/bin/sky-daemon"

  cat > "$INSTALL_DIR/sky-daemon.env" <<EOF
SKY_PANEL_WS_URL=${PANEL_URL}
SKY_NODE_TOKEN=${NODE_TOKEN}
SKY_DOCKER_SOCKET=/var/run/docker.sock
SKY_VOLUMES_ROOT=${VOLUMES_ROOT}
EOF
  chmod 600 "$INSTALL_DIR/sky-daemon.env"

  echo "$tag" > "$INSTALL_DIR/VERSION-daemon"
  fetch_repo_file "systemd/sky-daemon.service" /etc/systemd/system/sky-daemon.service
  chmod 644 /etc/systemd/system/sky-daemon.service
  fetch_repo_file "sky-panel-update" /usr/local/bin/sky-panel-update
  chmod 755 /usr/local/bin/sky-panel-update
  systemctl daemon-reload
  systemctl enable --now sky-daemon

  log "sky-daemon installed and connecting to ${PANEL_URL}"
}

require_root

case "$MODE" in
  panel) install_panel ;;
  node) install_node ;;
  all)
    install_panel
    PANEL_URL="${PANEL_URL:-ws://127.0.0.1:8080/agent/ws}"
    resolve_node_token
    if [[ -z "$NODE_TOKEN" ]]; then
      fail "a node token is required for 'all' mode too — register the first admin account, create a node from the admin console, then re-run with SKY_NODE_TOKEN=... (preferred) or --node-token"
    fi
    install_node
    ;;
  *)
    fail "usage: install.sh <panel|node|all> [--domain ...] [--panel-url ...] [--node-token ...]  (node token may also come from SKY_NODE_TOKEN env or stdin)"
    ;;
esac
