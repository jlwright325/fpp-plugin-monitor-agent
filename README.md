# ShowOps Agent Plugin

Lightweight outbound-only ShowOps monitoring agent for Falcon Player (FPP). The plugin installs a small Go binary and runs it under systemd when available, otherwise falls back to a background runner.

## Install via FPP

Paste this URL into the FPP Plugin Manager:

`https://raw.githubusercontent.com/jlwright325/fpp-plugin-monitor-agent/main/pluginInfo.json`

After install, edit `/home/fpp/media/config/fpp-monitor-agent.json` and set `enrollment_token`.

## Configure in FPP UI

In FPP, go to **Content Setup → Plugins → ShowOps Configuration**. Paste your enrollment token, save, and the agent will restart. Enrollment tokens are one-time use and will be cleared by the agent after enrollment.
