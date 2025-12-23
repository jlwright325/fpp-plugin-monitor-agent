#!/usr/bin/env bash
set -euo pipefail

CONFIG_PATH="${FPP_MONITOR_AGENT_CONFIG:-/home/fpp/media/config/fpp-monitor-agent.json}"
BIN_PATH="${FPP_MONITOR_AGENT_BIN:-/usr/local/bin/fpp-monitor-agent}"

if [[ ! -x "$BIN_PATH" ]]; then
  echo "fpp-monitor-agent binary not found at $BIN_PATH" >&2
  exit 1
fi

# Use logger when available so output goes to syslog.
if command -v logger >/dev/null 2>&1; then
  "$BIN_PATH" 2>&1 | logger -t fpp-monitor-agent
else
  "$BIN_PATH"
fi
