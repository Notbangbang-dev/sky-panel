#!/usr/bin/env bash
# Sky Panel uninstaller.
#
# Usage:
#   sudo bash uninstall.sh              # remove services + install dir, KEEP data
#   sudo bash uninstall.sh --purge-data # also delete the data dir (irreversible)
#
# Removes whatever this box has installed (panel-api, sky-daemon, or both):
# stops + disables the systemd services, removes their unit files, removes the
# install dir, and removes the sky-panel-update helper. The data directory is
# KEPT by default — deleting user data must be an explicit choice, so it only
# happens with --purge-data. Safe to re-run: every step tolerates things that
# are already gone.
set -euo pipefail

INSTALL_DIR="/opt/sky-panel"
SERVICE_USER="sky-panel"
# The panel keeps its SQLite database under INSTALL_DIR/data (see install.sh:
# SKY_DB_PATH=${INSTALL_DIR}/data/sky-panel.db). That's the only irreplaceable
# state, so it's what --purge-data guards.
DATA_DIR="${INSTALL_DIR}/data"

PURGE_DATA=0
while [[ $# -gt 0 ]]; do
  case "$1" in
    --purge-data) PURGE_DATA=1; shift ;;
    -h|--help)
      echo "usage: uninstall.sh [--purge-data]"
      echo "  (no args)     stop/disable services, remove unit files and ${INSTALL_DIR}, KEEP ${DATA_DIR}"
      echo "  --purge-data  additionally delete ${DATA_DIR} (irreversible)"
      exit 0 ;;
    *) echo "unknown argument: $1" >&2; exit 1 ;;
  esac
done

log() { echo "==> $*"; }
fail() { echo "error: $*" >&2; exit 1; }

[[ "$(id -u)" -eq 0 ]] || fail "must be run as root (try: sudo bash uninstall.sh)"

# Preserve the data dir across the INSTALL_DIR removal when we're keeping it,
# then restore it (it lives inside INSTALL_DIR).
KEEP_DATA_TMP=""

disable_service() {
  local svc="$1" unit="/etc/systemd/system/${svc}.service"
  # Only announce work if there's actually something here; stop/disable are
  # tolerant of an absent unit either way (idempotent).
  if [[ -f "$unit" ]] || systemctl is-active --quiet "$svc" 2>/dev/null; then
    log "stopping and disabling ${svc}"
  fi
  systemctl stop "$svc" 2>/dev/null || true
  systemctl disable "$svc" 2>/dev/null || true
  if [[ -f "$unit" ]]; then
    log "removing ${unit}"
    rm -f "$unit"
  fi
}

disable_service sky-panel
disable_service sky-daemon
systemctl daemon-reload 2>/dev/null || true

# Remove the updater helper.
if [[ -f /usr/local/bin/sky-panel-update ]]; then
  log "removing /usr/local/bin/sky-panel-update"
  rm -f /usr/local/bin/sky-panel-update
fi

if [[ -d "$INSTALL_DIR" ]]; then
  if [[ "$PURGE_DATA" -eq 1 ]]; then
    log "removing ${INSTALL_DIR} (including data — --purge-data given)"
    rm -rf "$INSTALL_DIR"
  else
    # Keep DATA_DIR: stash it aside, wipe the rest of INSTALL_DIR, restore it.
    if [[ -d "$DATA_DIR" ]]; then
      KEEP_DATA_TMP="$(mktemp -d)"
      log "keeping data dir ${DATA_DIR} (use --purge-data to delete it)"
      mv "$DATA_DIR" "${KEEP_DATA_TMP}/data"
    fi
    log "removing ${INSTALL_DIR} (data preserved)"
    rm -rf "$INSTALL_DIR"
    if [[ -n "$KEEP_DATA_TMP" ]]; then
      mkdir -p "$INSTALL_DIR"
      mv "${KEEP_DATA_TMP}/data" "$DATA_DIR"
      rmdir "$KEEP_DATA_TMP" 2>/dev/null || true
      # Leave it owned by the service user in case the panel is reinstalled.
      chown -R "$SERVICE_USER":"$SERVICE_USER" "$INSTALL_DIR" 2>/dev/null || true
      log "data left at ${DATA_DIR}"
    fi
  fi
else
  log "${INSTALL_DIR} not present — nothing to remove"
fi

# We intentionally do NOT remove the SERVICE_USER account or Docker/Caddy —
# those may be shared with other software and are cheap to leave in place.
log "note: the '${SERVICE_USER}' user, Docker, and Caddy were left installed."

if [[ "$PURGE_DATA" -eq 1 ]]; then
  log "uninstall complete (data purged)."
else
  log "uninstall complete. Data kept at ${DATA_DIR} — re-run with --purge-data to delete it."
fi
