# ShowOps Agent Plugin

Lightweight outbound-only ShowOps monitoring agent for Falcon Player (FPP). The plugin installs a small Go binary (`fpp-monitor-agent`) and runs it under systemd when available, otherwise falls back to a background runner.

---

## Table of Contents

- [Overview](#overview)
- [Requirements](#requirements)
- [Installation](#installation)
- [Configuration](#configuration)
- [Pairing a Device](#pairing-a-device)
- [Remote Sessions](#remote-sessions)
- [Uninstallation](#uninstallation)
- [Troubleshooting](#troubleshooting)
- [Development](#development)

---

## Overview

The plugin provides:

- **Heartbeat reporting** — device sends periodic status updates to the ShowOps API
- **Command polling** — device polls for and executes allowlisted management commands
- **Device pairing** — one-time enrollment flow to associate the device with a ShowOps account
- **Remote sessions** — optional cloudflared-based tunnel for secure remote access

Architecture:

```
FPP Dashboard
    ↓
menu.inc → www/showops.php  (Plugin web UI)
    ↓
/home/fpp/media/config/fpp-monitor-agent.json  (Config file)
    ↓
fpp-monitor-agent (Go binary, managed by systemd or fallback runner)
    ↓
https://api.showops.io  (ShowOps cloud API)
    ↓
cloudflared tunnel (optional — for remote sessions)
```

The Go binary itself lives in the [fpp-agent-monitor](https://github.com/jlwright325/fpp-agent-monitor) repository. This plugin repo handles FPP integration, installation, and the web UI.

---

## Requirements

- Falcon Player (FPP) v9.0 or later
- Linux (ARMv7, ARM64, or x86_64)
- `curl` or `wget` for downloading the agent binary
- `sha256sum` or `shasum` for checksum verification
- `systemd` recommended; fallback to `crontab`/`nohup` if unavailable

---

## Installation

FPP installs the plugin automatically when added via the FPP Plugin Manager. The plugin manager clones this repository into `/home/fpp/media/plugins/showops-agent` and runs `scripts/fpp_install.sh`.

### Manual installation

```bash
cd /home/fpp/media/plugins/showops-agent
bash scripts/fpp_install.sh
```

The installer will:

1. Detect the system architecture (armv7, arm64, x86_64)
2. Download the latest `fpp-monitor-agent` binary from GitHub releases
3. Verify the SHA-256 checksum
4. Install the binary to `/opt/fpp-monitor-agent/fpp-monitor-agent` (or `./bin/` if no sudo)
5. Write a default config to `/home/fpp/media/config/fpp-monitor-agent.json` (skipped if config already exists)
6. Install and start the `fpp-monitor-agent` systemd service (or start via fallback runner)

### Environment variables

| Variable | Default | Description |
|----------|---------|-------------|
| `RELEASE_VERSION` | _(latest)_ | Pin to a specific release tag (e.g. `v0.1.27`) |
| `AGENT_REPO_OWNER` | `jlwright325` | GitHub owner for agent binary releases |
| `AGENT_REPO_NAME` | `fpp-agent-monitor` | GitHub repo name for agent binary releases |
| `RELEASE_BASE` | _(derived)_ | Override full release asset base URL |
| `DRY_RUN` | `0` | Set to `1` to print actions without executing them |

---

## Configuration

The config file is located at `/home/fpp/media/config/fpp-monitor-agent.json`. It is created with defaults on first install and never overwritten by reinstalls.

```json
{
  "api_base_url": "https://api.showops.io",
  "enrollment_token": "",
  "device_id": "",
  "device_token": "",
  "device_fingerprint": "",
  "pairing_requested": false,
  "pairing_request_id": "",
  "pairing_code": "",
  "pairing_expires_at": "",
  "pairing_status": "",
  "unpair_requested": false,
  "cloudflared_token": "",
  "cloudflared_hostname": "",
  "heartbeat_interval_sec": 60,
  "command_poll_interval_sec": 30,
  "reboot_enabled": false,
  "restart_fpp_command": "systemctl restart fppd || systemctl restart fpp || service fppd restart || true"
}
```

| Field | Description |
|-------|-------------|
| `api_base_url` | ShowOps API endpoint. Do not change unless self-hosting. |
| `enrollment_token` | Token used during initial device enrollment. Set automatically during pairing. |
| `device_id` / `device_token` | Set automatically after successful enrollment. |
| `device_fingerprint` | Hardware fingerprint used to identify the device. Set automatically. |
| `pairing_requested` | Set to `true` to initiate a pairing request on next agent cycle. |
| `pairing_code` | Short code displayed during pairing. Read-only — set by agent. |
| `pairing_expires_at` | ISO timestamp when the pairing code expires. Read-only. |
| `pairing_status` | Current pairing state (`pending`, `approved`, `denied`). Read-only. |
| `cloudflared_token` | Tunnel token for remote session support. Set by agent after enrollment. |
| `cloudflared_hostname` | Tunnel hostname. Set by agent. |
| `heartbeat_interval_sec` | How often (seconds) the agent sends a heartbeat to the API. |
| `command_poll_interval_sec` | How often (seconds) the agent polls for queued commands. |
| `reboot_enabled` | Allow ShowOps to trigger a system reboot remotely. Defaults to `false`. |
| `restart_fpp_command` | Command used to restart FPP when requested remotely. |

---

## Pairing a Device

1. In FPP, navigate to **Plugins → ShowOps Configuration**.
2. Click **Request Pairing**.
3. A 6-character pairing code will appear. Enter it in the ShowOps dashboard under **Devices → Add Device**.
4. The page will update automatically once the pairing is approved.
5. The agent will begin sending heartbeats within one polling cycle.

If pairing fails or expires, click **Request Pairing** again to generate a new code.

---

## Remote Sessions

Remote sessions use [cloudflared](https://github.com/cloudflare/cloudflared) to establish an outbound-only tunnel to the ShowOps cloud. No inbound ports need to be opened.

`cloudflared` is bundled in release tarballs starting from `v0.1.27`. If the installed release does not include it, remote sessions will be unavailable until cloudflared is installed manually:

```bash
# ARM64 example
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-arm64 \
  -o /opt/fpp-monitor-agent/cloudflared
chmod +x /opt/fpp-monitor-agent/cloudflared
```

---

## Uninstallation

```bash
bash /home/fpp/media/plugins/showops-agent/scripts/fpp_uninstall.sh
```

To preserve the config file (device pairing state, tokens) during uninstall:

```bash
KEEP_CONFIG=1 bash scripts/fpp_uninstall.sh
```

---

## Troubleshooting

### Check service status

```bash
systemctl status fpp-monitor-agent.service
```

### View agent logs

```bash
# systemd journal
journalctl -u fpp-monitor-agent.service -n 100 --no-pager

# Install log
cat /home/fpp/media/logs/fpp-monitor-agent-install.log
```

### Agent not starting

1. Check that the binary exists: `ls -la /opt/fpp-monitor-agent/fpp-monitor-agent`
2. Check for config errors: `cat /home/fpp/media/config/fpp-monitor-agent.json`
3. Try running manually: `/opt/fpp-monitor-agent/fpp-monitor-agent`

### Pairing code not appearing

- Ensure the device has outbound internet access to `https://api.showops.io`
- Check agent logs for API errors
- Verify `pairing_requested` is `true` in the config file

### Re-install / update

Re-running `fpp_install.sh` will download and install the latest agent binary without overwriting your existing config:

```bash
bash /home/fpp/media/plugins/showops-agent/scripts/fpp_install.sh
```

To install a specific version:

```bash
RELEASE_VERSION=v0.1.27 bash scripts/fpp_install.sh
```

---

## Development

### Running a dry-run install

```bash
DRY_RUN=1 bash scripts/fpp_install.sh
```

### Linting shell scripts

```bash
shellcheck scripts/*.sh system/*.sh
```

### CI

GitHub Actions runs on every push and PR:

- **ShellCheck** — lints all shell scripts in `scripts/` and `system/`
- **Dry-run install** — validates the installer runs without error in dry-run mode
- **JSON validation** — validates `pluginInfo.json`
- **PHP syntax** — `php -l` on `www/showops.php` (FPP plugin UI; catches syntax errors before deploy)

See [`.github/workflows/ci.yml`](.github/workflows/ci.yml). 
