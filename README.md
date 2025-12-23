# FPP Monitor Agent Plugin

Lightweight outbound-only monitoring agent for Falcon Player (FPP). The plugin installs a small Go binary and runs it under systemd when available, otherwise falls back to a background runner.

## Install

1. Host this repository somewhere reachable by FPP (GitHub is typical).
2. In FPP, open **Content Setup → Plugins → Add Plugin**, then paste the URL to `pluginInfo.json` from this repo.
3. Complete installation and configure `/home/fpp/media/config/fpp-monitor-agent.json` with your device ID and token.

## How It Works

- The plugin installs the agent binary and writes a default config if missing.
- If systemd is available, it creates `fpp-monitor-agent.service` and starts it.
- If systemd is not available, it launches `system/fpp-monitor-agent.sh` in the background and adds an `@reboot` crontab entry to start on boot.
- The agent sends heartbeats to your platform and polls for commands on a short interval.

## Configuration

Config file: `/home/fpp/media/config/fpp-monitor-agent.json`

```json
{
  "device_id": "",
  "device_token": "",
  "api_base_url": "https://api.your-platform.example",
  "heartbeat_interval_sec": 10,
  "command_poll_interval_sec": 5,
  "reboot_enabled": false,
  "restart_fpp_command": "systemctl restart fppd",
  "update_channel": "stable",
  "update_base_url": "https://api.your-platform.example",
  "allowed_commands": [
    "systemctl restart fppd"
  ]
}
```

Environment overrides are supported:
- `FPP_MONITOR_AGENT_DEVICE_ID`
- `FPP_MONITOR_AGENT_DEVICE_TOKEN`
- `FPP_MONITOR_AGENT_API_BASE_URL`
- `FPP_MONITOR_AGENT_HEARTBEAT_INTERVAL_SEC`
- `FPP_MONITOR_AGENT_COMMAND_POLL_INTERVAL_SEC`
- `FPP_MONITOR_AGENT_REBOOT_ENABLED`
- `FPP_MONITOR_AGENT_RESTART_FPP_COMMAND`
- `FPP_MONITOR_AGENT_UPDATE_CHANNEL`
- `FPP_MONITOR_AGENT_UPDATE_BASE_URL`
- `FPP_MONITOR_AGENT_ALLOWED_COMMANDS` (comma-separated)

## Heartbeat Payload

POST `{API_BASE}/v1/ingest/heartbeat`

```json
{
  "payload_version": 1,
  "sent_at": 1710000000,
  "device": {
    "device_id": "...",
    "hostname": "...",
    "fpp_version": "...",
    "agent_version": "..."
  },
  "state": {
    "playing": true,
    "mode": "...",
    "playlist": "...",
    "sequence": "..."
  },
  "resources": {
    "cpu_percent": 12.3,
    "memory_percent": 42.1,
    "disk_free_mb": 1234
  },
  "extra": {
    "raw": {
      "ips": ["192.168.1.2"],
      "fpp": { "...": "..." }
    }
  }
}
```

## Command Polling

GET `{API_BASE}/v1/agent/commands?device_id=...`

Supported actions:
- `restart_agent`
- `restart_fpp`
- `reboot` (only if `reboot_enabled` is true)
- `update_to_version` (requires a release manifest endpoint)
- `run_allowlisted`

Command completion POST:
`{API_BASE}/v1/agent/commands/:id/complete`

## Self-Update

The agent looks up a manifest at:
`{API_BASE}/v1/agent/releases/manifest?channel={update_channel}&version={version}&platform={goarch}`

Manifest format:

```json
{ "url": "https://.../fpp-monitor-agent-armv7", "sha256": "..." }
```

The agent downloads the artifact, verifies sha256, atomically swaps the binary, then restarts itself.

## Local Dev

```bash
cd agent
GOOS=linux GOARCH=arm GOARM=7 go build -o fpp-monitor-agent
```

Run locally:

```bash
FPP_MONITOR_AGENT_DEVICE_ID=dev \
FPP_MONITOR_AGENT_DEVICE_TOKEN=token \
FPP_MONITOR_AGENT_API_BASE_URL=http://localhost:8080 \
./fpp-monitor-agent
```

## Install/Uninstall Flags

- `DRY_RUN=1` on `fpp_install.sh` logs actions without making changes.
- `PURGE=1` on `fpp_uninstall.sh` removes the config file.

## Release Process

1. Update version references if needed (pluginInfo.json, install script, docs).
2. Tag a release: `git tag vX.Y.Z && git push origin vX.Y.Z`.
3. GitHub Actions builds:
   - `dist/fpp-monitor-agent-linux-armv7`
   - `dist/fpp-monitor-agent-linux-arm64`
   - `dist/checksums.txt`
4. The workflow creates or updates the GitHub Release for the tag and uploads the artifacts.

## Files

- `pluginInfo.json` plugin descriptor
- `fpp_install.sh` install script
- `fpp_uninstall.sh` uninstall script
- `system/` systemd unit + fallback runner
- `agent/` Go agent source
