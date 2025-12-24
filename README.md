# FPP Monitor Agent Plugin

Lightweight outbound-only monitoring agent for Falcon Player (FPP). The plugin installs a small Go binary and runs it under systemd when available, otherwise falls back to a background runner.

## Install via FPP

Paste this URL into the FPP Plugin Manager:

`https://raw.githubusercontent.com/jlwright325/fpp-plugin-monitor-agent/main/pluginInfo.json`

After install, edit `/home/fpp/media/config/fpp-monitor-agent.json` and set `enrollment_token`.
