# ShowOps monitor plugin — integration contract

**Version:** 1.0  
**Audience:** Engineers working on this plugin, the Go agent (`fpp-agent-monitor`), or ShowOps cloud APIs.

This document is the **frozen boundary** for slice-level work: paths, config keys the plugin and PHP UI rely on, and support correlation. Extend it when you add new integration surface area.

---

## Filesystem layout

| Path | Owner | Purpose |
|------|--------|---------|
| `/home/fpp/media/plugins/showops-agent` | Plugin | Installed plugin tree (`scripts/`, `www/`, `system/`). |
| `/home/fpp/media/config/fpp-monitor-agent.json` | Plugin + agent | JSON config read/written by installer, uninstaller, PHP UI, and agent. |
| `/opt/fpp-monitor-agent/fpp-monitor-agent` | Installer (sudo) | Primary binary when sudo is available. |
| `{plugin}/bin/fpp-monitor-agent` | Installer (no sudo) | Binary location when installing without root. |
| `/opt/fpp-monitor-agent/VERSION` | Installer | Release tag string for support and UI. |
| `/home/fpp/media/logs/fpp-monitor-agent-install.log` | Installer | Install/uninstall log (append). |

Legacy path `/home/fpp/media/plugins/fpp-monitor-agent` may still appear on upgraded systems; uninstall removes artifacts there when present.

---

## Config JSON

The installer writes the **full** default schema on first install and must **never** overwrite an existing file on reinstall.

**Uninstall (default mode)** clears enrollment and pairing fields only via **merge** into the existing object. Keys such as `api_base_url`, `heartbeat_interval_sec`, and `restart_fpp_command` must survive uninstall so operators are not reset to cloud defaults unintentionally.

Pairing-related keys cleared on uninstall (default):

- `enrollment_token`, `device_id`, `device_token`, `device_fingerprint`
- `pairing_requested`, `pairing_request_id`, `pairing_code`, `pairing_expires_at`, `pairing_status`, `unpair_requested`

`PURGE=1` removes the config file entirely. `KEEP_CONFIG=1` skips pairing clears.

---

## Install / uninstall observability

Each `fpp_install.sh` or `fpp_uninstall.sh` run emits a unique line early in the log:

```text
[fpp-monitor-agent] install begin install_run_id=<uuid>
```

Environment variable **`FPP_MONITOR_INSTALL_RUN_ID`** is set to the same value for the duration of the script (override allowed for tests). Field support can ask operators to grep `install_run_id` in `fpp-monitor-agent-install.log` and attach that line to tickets.

This satisfies the **plugin-side correlation** checkpoint from the post–slice-1 risk note: agent ↔ cloud correlation remains the responsibility of the Go binary; the plugin owns install/uninstall session IDs.

---

## Agent binary contract

The plugin downloads release assets from `fpp-agent-monitor` (see README). The tarball must contain:

- `fpp-monitor-agent` (executable)
- Optional `cloudflared` (executable) for remote sessions

Checksum verification uses `checksums.txt` from the same release. **Do not** change asset naming without updating `scripts/fpp_install.sh` and this document.

---

## PHP plugin UI (`www/showops.php`)

The UI reads the same config path and expects JSON objects as produced by the installer. Pairing actions must preserve operator-tuned fields (same merge discipline as uninstall).

---

## References

- Operator-facing install notes: [README.md](../README.md)
- Next-slice risk context: [SHO-253](/SHO/issues/SHO-253)
