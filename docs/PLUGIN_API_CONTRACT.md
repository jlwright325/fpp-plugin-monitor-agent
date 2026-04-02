# FPP ShowOps monitor plugin — integration contract

**Version:** 1.1.0  
**Slice:** #2 — follows monitor-agent plugin slice #1; scaling and deferral notes live in company issue **SHO-253**.  
**Status:** Frozen interfaces below are covered by CI (`scripts/verify_plugin_contract.sh`).

This document is the **versioned boundary** between:

- This repo (FPP plugin: install shell, systemd glue, `www/showops.php` UI), and
- The Go agent binary ([fpp-agent-monitor](https://github.com/jlwright325/fpp-agent-monitor)), which reads/writes the shared JSON config and calls ShowOps HTTP APIs.

Breaking changes to paths, config semantics, or plugin UI actions require **bumping `contractVersion`** in `docs/contract-fingerprints.json` and updating this file.

---

## 1. Frozen filesystem paths

| Symbol | Path | Owner |
|--------|------|--------|
| User config | `/home/fpp/media/config/fpp-monitor-agent.json` | Plugin installer creates parent dir; agent + UI read/write |
| Plugin root | `/home/fpp/media/plugins/showops-agent` | FPP plugin manager + `scripts/fpp_install.sh` |
| Agent binary (privileged install) | `/opt/fpp-monitor-agent/fpp-monitor-agent` | Installer when `sudo` available |
| Fallback wrapper | `{plugin root}/system/fpp-monitor-agent.sh` | Invoked by systemd unit or manual fallback |
| Systemd unit | `fpp-monitor-agent.service` (file under `/etc/systemd/system/` or `/lib/systemd/system/`) | Installer |

---

## 2. Config file (`fpp-monitor-agent.json`)

The **authoritative field list** for operators is in the root [README](../README.md#configuration). The agent binary may persist additional keys; the plugin UI depends on at least:

| Key | Plugin UI usage |
|-----|------------------|
| `device_id` | Enrollment / paired state |
| `last_heartbeat_ts` | Status card (“Last Heartbeat”) |
| `pairing_requested`, `pairing_request_id`, `pairing_code`, `pairing_expires_at`, `pairing_status` | Pairing flow |
| `unpair_requested` | Unpair flow |
| `enrollment_token` | Cleared/updated during pair/unpair transitions |
| `api_base_url` | Cleared during pair/unpair (agent re-resolves) |

**Encoding:** UTF-8 JSON object. Pretty-print is optional.

---

## 3. FPP plugin web surface (`www/showops.php`)

**Entry:** FPP loads the page via `menu.inc` → `showops.php` (path under plugin root).

**POST `action` values** (form field `action`):

| Value | Behavior |
|-------|----------|
| `pair` | Set pairing flags in config; restart agent |
| `unpair` | Request unpair; restart agent |
| `restart` | Restart agent service / fallback runner |
| `tail` | Refresh log snippet in UI |

CI asserts these four actions exist in `www/showops.php`. New actions require a contract bump and a fingerprint update.

---

## 4. Correlation and observability (agent HTTP)

The Go agent should attach a stable correlation identifier on **outbound HTTPS** to ShowOps so support can tie plugin restarts, heartbeats, and command polls to one device/session:

- **Header:** `ShowOps-Correlation-Id`
- **Value:** Prefer existing `device_id` once enrolled; before enroll, use a one-time UUID generated at process start and rotate after successful enrollment if needed.

This repo does not emit the header (PHP UI is local-only); **fpp-agent-monitor** implements it. Plugin contract version bumps do not require agent releases unless paths or config semantics change.

---

## 5. Security boundary (show network)

- **Leaves the LAN:** TLS to ShowOps API (`api_base_url`), GitHub release URLs for binary updates, and optional cloudflared endpoints for remote support — as configured by the agent.
- **Stays on the Pi:** FPP plugin UI (HTTP to local FPP), config file on disk, local logs under `/home/fpp/media/logs/` when the installer writes them.
- **Secrets:** `device_token`, `enrollment_token`, `cloudflared_token` must not be logged by the plugin UI (UI does not print raw tokens).

---

## 6. SLO targets (informative, not CI-enforced)

Aligned with architect checkpoint **SHO-253** — targets for product/ops, not automated gates yet:

| Signal | Target (v1 field ops) |
|--------|-------------------------|
| Heartbeat success → API `2xx` | ≥ 99% over 24h per device (excluding operator maintenance windows) |
| Config write (pair/unpair) → visible in UI on refresh | \< 5 s |
| False “offline” flip (dashboard) | \< 1 per show-night per device under normal WAN |

---

## 7. CI contract verification

`scripts/verify_plugin_contract.sh` reads `docs/contract-fingerprints.json` and fails if frozen paths or POST actions drift. Run locally:

```bash
bash scripts/verify_plugin_contract.sh
```
