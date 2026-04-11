# AGENTS.md – fpp-plugin-monitor-agent

## Role & Purpose
This repository contains the Falcon Player plugin and install/uninstall/configuration wrapper for the ShowOps FPP agent.

It owns:
- plugin UI shown inside FPP
- install and uninstall scripts
- default config creation and config handoff
- plugin-side pairing and integration behavior

## Architecture Rules
- this repo wraps the agent; it does not replace the agent runtime
- preserve FPP compatibility and plugin-manager expectations
- keep install/uninstall scripts safe and idempotent
- do not invent backend contracts; consume approved contracts from specs and impact plans
- config shape must stay aligned with the agent contract

## Expected Inputs for Codex
Before changing code, read:
- `.repo-manifest.yaml`
- the relevant feature spec in `showops-specs/product/...`
- the approved impact plan

## Likely Areas
- `scripts/`
- `system/`
- `www/`
- `docs/PLUGIN_API_CONTRACT.md`

## Do / Don't
✅ keep FPP install behavior predictable and scriptable
✅ update contract docs when plugin-agent boundaries change
✅ call out any config-schema changes explicitly
❌ do not move agent runtime logic into the plugin layer
❌ do not break dry-run or uninstall safety guarantees
