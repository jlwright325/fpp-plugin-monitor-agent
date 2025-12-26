#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$ROOT_DIR/.." && pwd)"
. "$ROOT_DIR/install_common.sh"

PLUGIN_DIR="/home/fpp/media/plugins/showops-agent"
CONFIG_PATH="/home/fpp/media/config/fpp-monitor-agent.json"
INSTALL_DIR="/opt/fpp-monitor-agent"
BIN_LINK="/usr/local/bin/fpp-monitor-agent"
FALLBACK_SCRIPT="$PLUGIN_DIR/system/fpp-monitor-agent.sh"

if ! can_sudo; then
  INSTALL_DIR="$PLUGIN_DIR/bin"
  log "No sudo; using $INSTALL_DIR for binary install"
fi

BIN_PATH="$INSTALL_DIR/fpp-monitor-agent"

DEFAULT_RELEASE_VERSION="v0.1.12"
RELEASE_VERSION="${RELEASE_VERSION:-}"
AGENT_REPO_OWNER="${AGENT_REPO_OWNER:-jlwright325}"
AGENT_REPO_NAME="${AGENT_REPO_NAME:-fpp-agent-monitor}"

resolve_latest_tag() {
  local manifest_url="https://raw.githubusercontent.com/jlwright325/fpp-agent-monitor/main/latest.json"
  local api_url="https://api.github.com/repos/${AGENT_REPO_OWNER}/${AGENT_REPO_NAME}/releases/latest"
  local body=""
  local tmp=""

  tmp="$(mktemp)"
  if download_file "$manifest_url" "$tmp" 1>&2; then
    body="$(cat "$tmp")"
    rm -f "$tmp"
    echo "$body" | sed -n 's/.*"version"[[:space:]]*:[[:space:]]*"\\([^"]*\\)".*/\\1/p' | head -n 1
    return 0
  fi
  rm -f "$tmp"
  log "Failed to resolve latest tag from $manifest_url" >&2

  tmp="$(mktemp)"
  if ! download_file "$api_url" "$tmp" 1>&2; then
    rm -f "$tmp"
    log "Failed to resolve latest tag from $api_url" >&2
    return 1
  fi
  body="$(cat "$tmp")"
  rm -f "$tmp"

  echo "$body" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\\([^"]*\\)".*/\\1/p' | head -n 1
}

RESOLVED_TAG="$RELEASE_VERSION"
if [[ -z "$RESOLVED_TAG" ]]; then
  RESOLVED_TAG="$(resolve_latest_tag || true)"
  if [[ -z "$RESOLVED_TAG" ]]; then
    RESOLVED_TAG="$DEFAULT_RELEASE_VERSION"
    log "Failed to resolve latest tag; falling back to $RESOLVED_TAG"
  else
    log "Resolved latest tag: $RESOLVED_TAG"
  fi
fi

RELEASE_BASE="${RELEASE_BASE:-https://github.com/${AGENT_REPO_OWNER}/${AGENT_REPO_NAME}/releases/download/${RESOLVED_TAG}}"

platform_arch="$($ROOT_DIR/detect_platform.sh)"
asset_name="fpp-monitor-agent-linux-${platform_arch}"
checksums_name="checksums.txt"

ensure_dir "$PLUGIN_DIR"
ensure_dir "$(dirname "$CONFIG_PATH")"

log "Installing for platform: $platform_arch"

if ! is_dry_run; then
  ensure_dir "$INSTALL_DIR"
fi

tmp_dir="$(mktemp -d)"
tmp_bin="$tmp_dir/$asset_name"
tmp_checksums="$tmp_dir/$checksums_name"

log "Downloading release assets from $RELEASE_BASE"
log "Resolved asset URLs: $RELEASE_BASE/$asset_name and $RELEASE_BASE/$checksums_name"
if is_dry_run; then
  log "DRY_RUN: would download $RELEASE_BASE/$asset_name"
  log "DRY_RUN: would download $RELEASE_BASE/$checksums_name"
  log "DRY_RUN: would verify checksum and install $BIN_PATH"
  log "DRY_RUN: would write version file to $INSTALL_DIR/VERSION"
  rm -rf "$tmp_dir"
else
  if ! download_file "$RELEASE_BASE/$asset_name" "$tmp_bin"; then
    log "Failed to download $asset_name"
    rm -rf "$tmp_dir"
    exit 1
  fi
  if ! download_file "$RELEASE_BASE/$checksums_name" "$tmp_checksums"; then
    log "Failed to download $checksums_name"
    rm -rf "$tmp_dir"
    exit 1
  fi

  expected_sha="$(awk "/$asset_name/ {print \$1}" "$tmp_checksums")"
  if [[ -z "$expected_sha" ]]; then
    log "Checksum for $asset_name not found in checksums.txt"
    rm -rf "$tmp_dir"
    exit 1
  fi

  if ! actual_sha="$(sha256_file "$tmp_bin")"; then
    log "Failed to compute sha256 for downloaded binary"
    rm -rf "$tmp_dir"
    exit 1
  fi
  if [[ "$expected_sha" != "$actual_sha" ]]; then
    log "Checksum mismatch for downloaded binary"
    rm -rf "$tmp_dir"
    exit 1
  fi

  log "Installing binary to $BIN_PATH"
  run_cmd_sudo install -m 0755 "$tmp_bin" "$BIN_PATH"
  run_cmd_sudo sh -c "echo \"$RESOLVED_TAG\" > \"$INSTALL_DIR/VERSION\""

  if can_sudo; then
    run_cmd sudo ln -sf "$BIN_PATH" "$BIN_LINK"
  else
    log "No sudo; skipping symlink to $BIN_LINK"
  fi

  rm -rf "$tmp_dir"
fi

if [[ ! -f "$CONFIG_PATH" ]]; then
  log "Writing default config to $CONFIG_PATH"
  if is_dry_run; then
    log "DRY_RUN: would write config template"
  else
    cat <<'JSON' > "$CONFIG_PATH"
{
  "enrollment_token": "",
  "heartbeat_interval_sec": 10,
  "command_poll_interval_sec": 5,
  "reboot_enabled": false,
  "restart_fpp_command": "systemctl restart fppd || systemctl restart fpp || service fppd restart || true"
}
JSON
  fi
else
  log "Config exists; leaving $CONFIG_PATH unchanged"
fi

if is_dry_run; then
  log "DRY_RUN: would ensure $CONFIG_PATH is writable by fpp"
else
  if can_sudo; then
    run_cmd_sudo chown fpp:fpp "$CONFIG_PATH" || true
    run_cmd_sudo chmod 664 "$CONFIG_PATH" || true
  else
    run_cmd chmod 664 "$CONFIG_PATH" || true
  fi
fi

if is_systemd; then
  log "Installing systemd service"
  if can_sudo; then
    run_cmd_sudo install -m 0644 "$REPO_ROOT/system/fpp-monitor-agent.service" /etc/systemd/system/fpp-monitor-agent.service
    run_cmd_sudo systemctl daemon-reload
    run_cmd_sudo systemctl enable fpp-monitor-agent.service
    restart_output="$(run_cmd_capture sudo systemctl restart fpp-monitor-agent.service)"
    restart_code=$?
    if [[ $restart_code -eq 0 ]]; then
      run_cmd_sudo systemctl --no-pager --full status fpp-monitor-agent.service || true
    else
      if [[ -n "$restart_output" ]]; then
        log "Systemd restart failed: $restart_output"
      else
        log "Systemd restart failed with exit code $restart_code"
      fi
      log "Falling back to runner"
      ensure_dir "$PLUGIN_DIR/system"
      run_cmd install -m 0755 "$REPO_ROOT/system/fpp-monitor-agent.sh" "$FALLBACK_SCRIPT"
      run_cmd nohup "$FALLBACK_SCRIPT" >/dev/null 2>&1 &
    fi
  else
    log "Systemd present but no sudo; using fallback runner"
    ensure_dir "$PLUGIN_DIR/system"
    run_cmd install -m 0755 "$REPO_ROOT/system/fpp-monitor-agent.sh" "$FALLBACK_SCRIPT"
    run_cmd nohup "$FALLBACK_SCRIPT" >/dev/null 2>&1 &
  fi
else
  log "Systemd not detected; installing fallback runner"
  ensure_dir "$PLUGIN_DIR/system"
  run_cmd install -m 0755 "$REPO_ROOT/system/fpp-monitor-agent.sh" "$FALLBACK_SCRIPT"
  run_cmd nohup "$FALLBACK_SCRIPT" >/dev/null 2>&1 &
  if have_command crontab; then
    log "Registering fallback runner at boot via crontab"
    if is_dry_run; then
      log "DRY_RUN: would add @reboot $FALLBACK_SCRIPT to crontab"
    else
      (crontab -l 2>/dev/null | grep -v "fpp-monitor-agent.sh" ; echo "@reboot $FALLBACK_SCRIPT") | crontab -
    fi
  else
    log "crontab not available; fallback runner will not be auto-started"
  fi
fi

log "Install complete"
