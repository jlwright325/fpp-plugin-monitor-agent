# ShowOps Agent Plugin

Lightweight outbound-only ShowOps monitoring agent for Falcon Player (FPP). The plugin installs a small Go binary and runs it under systemd when available, otherwise falls back to a background runner.

## Install via FPP

Paste this URL into the FPP Plugin Manager:

`https://raw.githubusercontent.com/jlwright325/fpp-plugin-monitor-agent/main/pluginInfo.json`

The installer fetches the agent binary from GitHub Releases:
`https://github.com/jlwright325/fpp-agent-monitor/releases/download/<tag>/`

## Configure in FPP UI

In FPP, go to **Content Setup → Plugins → ShowOps Configuration**. Set your enrollment token, then click **Save + Restart**. Enrollment tokens are one-time use and will be cleared by the agent after enrollment.

Config file path: `/home/fpp/media/config/fpp-monitor-agent.json`

Remote sessions are provisioned automatically when requested from the ShowOps UI.

## API Base URL

The agent defaults to `https://api.showops.io` internally. The plugin does not expose this in the UI or config file. For development only, override via `SHOWOPS_API_BASE_URL` in the agent environment.

## Verify Agent Status

If systemd is available:

```
systemctl status fpp-monitor-agent.service
journalctl -u fpp-monitor-agent.service -n 200 --no-pager
```

Without systemd:

```
ps aux | grep fpp-monitor-agent
```

## Troubleshooting

- Install failures: re-run `/home/fpp/media/plugins/showops-agent/scripts/fpp_install.sh` and check for URL or checksum mismatch messages.
- UI issues: verify `/fpp` loads and check Apache errors in `/home/fpp/media/logs/apache2-base-error.log`.

## Agent Logs

- systemd: `journalctl -u fpp-monitor-agent.service -n 200 --no-pager`
- fallback runner: `/var/log/syslog` or `/var/log/messages` (logger tag: `fpp-monitor-agent`)
