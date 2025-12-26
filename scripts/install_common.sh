#!/usr/bin/env bash
set -euo pipefail

log() {
  echo "[fpp-monitor-agent] $*"
}

have_command() {
  command -v "$1" >/dev/null 2>&1
}

download_file() {
  local url="$1"
  local dest="$2"
  local status=""

  if have_command curl; then
    status="$(curl -sSL -L -w "%{http_code}" -o "$dest" "$url" || true)"
    if [[ "$status" != "200" ]]; then
      log "Download failed (HTTP $status): $url"
      return 1
    fi
    return 0
  elif have_command wget; then
    status="$(wget --server-response -O "$dest" "$url" 2>&1 | awk '/^  HTTP/{code=$2} END{print code}' || true)"
    if [[ "$status" != "200" ]]; then
      log "Download failed (HTTP $status): $url"
      return 1
    fi
    return 0
  else
    log "Neither curl nor wget found."
    return 1
  fi
}

sha256_file() {
  local file="$1"
  if have_command sha256sum; then
    sha256sum "$file" | awk '{print $1}'
  elif have_command shasum; then
    shasum -a 256 "$file" | awk '{print $1}'
  else
    log "No sha256 tool available."
    return 1
  fi
}

is_systemd() {
  [[ -d /run/systemd/system ]] && have_command systemctl
}

ensure_dir() {
  local dir="$1"
  if [[ ! -d "$dir" ]]; then
    mkdir -p "$dir"
  fi
}

can_sudo() {
  have_command sudo && sudo -n true >/dev/null 2>&1
}

is_dry_run() {
  [[ "${DRY_RUN:-0}" == "1" ]]
}

run_cmd() {
  if is_dry_run; then
    log "DRY_RUN: $*"
    return 0
  fi
  "$@"
}

run_cmd_sudo() {
  if can_sudo; then
    run_cmd sudo "$@"
  else
    run_cmd "$@"
  fi
}
