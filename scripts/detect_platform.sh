#!/usr/bin/env bash
set -euo pipefail

arch_raw="$(uname -m)"
arch="unknown"

case "$arch_raw" in
  armv7l|armv7*) arch="armv7" ;;
  aarch64|arm64) arch="arm64" ;;
  *) arch="$arch_raw" ;;
 esac

echo "$arch"
