#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
. "$ROOT_DIR/install_common.sh"

PLUGIN_DIR="/home/fpp/media/plugins/showops-agent"
BIN_LINK="/usr/local/bin/fpp-monitor-agent"
INSTALL_DIR="/opt/fpp-monitor-agent"
BIN_PATH_SYSTEM="$INSTALL_DIR/fpp-monitor-agent"
BIN_PATH_PLUGIN="$PLUGIN_DIR/bin/fpp-monitor-agent"
FALLBACK_SCRIPT="$PLUGIN_DIR/system/fpp-monitor-agent.sh"
CONFIG_PATH="/home/fpp/media/config/fpp-monitor-agent.json"

if is_systemd; then
  if can_sudo; then
    run_cmd_sudo systemctl disable --now fpp-monitor-agent.service || true
    run_cmd_sudo rm -f /etc/systemd/system/fpp-monitor-agent.service
    run_cmd_sudo systemctl daemon-reload || true
  else
    log "Systemd present but no sudo; cannot fully remove service"
  fi
else
  run_cmd pkill -f "fpp-monitor-agent" || true
fi

if can_sudo; then
  run_cmd sudo rm -f "$BIN_LINK"
  run_cmd sudo rm -f "$BIN_PATH_SYSTEM"
  run_cmd sudo rmdir "$INSTALL_DIR" 2>/dev/null || true
else
  log "No sudo; cannot remove $BIN_PATH_SYSTEM or $BIN_LINK"
fi

run_cmd rm -f "$BIN_PATH_PLUGIN" || true
run_cmd rm -f "$FALLBACK_SCRIPT" || true

if have_command crontab; then
  if is_dry_run; then
    log "DRY_RUN: would remove crontab entry for $FALLBACK_SCRIPT"
  else
    crontab -l 2>/dev/null | grep -v "fpp-monitor-agent.sh" | crontab - || true
  fi
fi

if [[ "${PURGE:-0}" == "1" ]]; then
  log "PURGE=1 set; removing config"
  run_cmd rm -f "$CONFIG_PATH"
  log "Uninstall complete (config removed)"
else
  log "Config retained at $CONFIG_PATH"
  log "Uninstall complete (config retained)"
fi
