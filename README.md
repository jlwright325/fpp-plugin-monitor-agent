# ShowOps Agent Plugin

Lightweight outbound-only ShowOps monitoring agent for Falcon Player (FPP). The plugin installs a small Go binary and runs it under systemd when available, otherwise falls back to a background runner.

## Install via FPP

Paste this URL into the FPP Plugin Manager:

`https://raw.githubusercontent.com/jlwright325/fpp-plugin-monitor-agent/main/pluginInfo.json`

## Configure in FPP UI

In FPP, go to **Content Setup → Plugins → ShowOps Configuration**. Set your enrollment token and API base URL, then click **Save + Restart**. Enrollment tokens are one-time use and will be cleared by the agent after enrollment.

Config file path: `/home/fpp/media/config/fpp-monitor-agent.json`

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
