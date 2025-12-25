package commands

import (
  "context"
  "encoding/json"
  "errors"
  "fmt"
  "os"
  "os/exec"
  "strings"

  "github.com/jlwright325/fpp-plugin-monitor-agent/agent/internal/client"
  "github.com/jlwright325/fpp-plugin-monitor-agent/agent/internal/config"
  "github.com/jlwright325/fpp-plugin-monitor-agent/agent/internal/logging"
  "github.com/jlwright325/fpp-plugin-monitor-agent/agent/internal/update"
)

type Command struct {
  ID     string         `json:"id"`
  Action string         `json:"action"`
  Params map[string]any `json:"params"`
}

type Result struct {
  Status   string `json:"status"`
  Stdout   string `json:"stdout"`
  Stderr   string `json:"stderr"`
  ExitCode int    `json:"exit_code"`
}

type Runner struct {
  cfg     *config.Config
  client  *client.Client
  log     *logging.Logger
  version string
}

func NewRunner(cfg *config.Config, client *client.Client, log *logging.Logger, version string) *Runner {
  return &Runner{cfg: cfg, client: client, log: log, version: version}
}

func (r *Runner) PollAndExecute(ctx context.Context) {
  if r.cfg.DeviceID == "" || r.cfg.DeviceToken == "" {
    r.log.Info("command polling disabled; missing device_id or device_token", nil)
    return
  }

  url := fmt.Sprintf("%s/v1/agent/commands?device_id=%s", strings.TrimRight(r.cfg.ApiBaseURL, "/"), r.cfg.DeviceID)
  headers := map[string]string{"Authorization": "Bearer " + r.cfg.DeviceToken}

  resp, body, err := r.client.DoJSONWithRetry(ctx, "GET", url, headers, nil)
  if err != nil {
    r.log.Error("command poll failed", map[string]any{"error": err.Error()})
    return
  }
  if resp.StatusCode >= 300 {
    r.log.Error("command poll non-200", map[string]any{"status": resp.StatusCode})
    return
  }

  var cmds []Command
  if err := json.Unmarshal(body, &cmds); err != nil {
    r.log.Error("command decode failed", map[string]any{"error": err.Error()})
    return
  }

  for _, cmd := range cmds {
    r.executeOne(ctx, cmd)
  }
}

func (r *Runner) executeOne(ctx context.Context, cmd Command) {
  res, exitAfter := r.runCommand(ctx, cmd)
  r.complete(ctx, cmd.ID, res)
  if exitAfter {
    os.Exit(0)
  }
}

func (r *Runner) complete(ctx context.Context, id string, res Result) {
  url := fmt.Sprintf("%s/v1/agent/commands/%s/complete", strings.TrimRight(r.cfg.ApiBaseURL, "/"), id)
  headers := map[string]string{"Authorization": "Bearer " + r.cfg.DeviceToken}

  _, _, err := r.client.DoJSONWithRetry(ctx, "POST", url, headers, res)
  if err != nil {
    r.log.Error("command completion failed", map[string]any{"error": err.Error()})
  }
}

func (r *Runner) runCommand(ctx context.Context, cmd Command) (Result, bool) {
  switch cmd.Action {
  case "restart_agent":
    return r.restartAgent(ctx)
  case "restart_fpp":
    return r.restartFPP(ctx)
  case "reboot":
    return r.reboot(ctx)
  case "update_to_version":
    return r.updateToVersion(ctx, cmd)
  case "run_allowlisted":
    return r.runAllowlisted(ctx, cmd)
  default:
    return Result{Status: "rejected", Stderr: "unknown action", ExitCode: 1}, false
  }
}

func (r *Runner) restartAgent(ctx context.Context) (Result, bool) {
  if isSystemd() {
    return Result{Status: "ok", ExitCode: 0}, true
  }
  if err := launchFallback(ctx); err != nil {
    return Result{Status: "error", Stderr: logging.Truncate(err.Error(), 2000), ExitCode: 1}, false
  }
  return Result{Status: "ok", ExitCode: 0}, true
}

func (r *Runner) restartFPP(ctx context.Context) (Result, bool) {
  parts := strings.Fields(r.cfg.RestartFPPCommand)
  if len(parts) == 0 {
    return Result{Status: "rejected", Stderr: "restart_fpp_command not set", ExitCode: 1}, false
  }
  cmd := exec.CommandContext(ctx, parts[0], parts[1:]...)
  out, err := cmd.CombinedOutput()
  if err != nil {
    return Result{Status: "error", Stderr: logging.Truncate(string(out), 2000), ExitCode: 1}, false
  }
  return Result{Status: "ok", Stdout: logging.Truncate(string(out), 2000), ExitCode: 0}, false
}

func (r *Runner) reboot(ctx context.Context) (Result, bool) {
  if !r.cfg.RebootEnabled {
    return Result{Status: "rejected", Stderr: "reboot disabled", ExitCode: 1}, false
  }
  cmd := exec.CommandContext(ctx, "reboot")
  out, err := cmd.CombinedOutput()
  if err != nil {
    return Result{Status: "error", Stderr: logging.Truncate(string(out), 2000), ExitCode: 1}, false
  }
  return Result{Status: "ok", Stdout: logging.Truncate(string(out), 2000), ExitCode: 0}, false
}

func (r *Runner) updateToVersion(ctx context.Context, cmd Command) (Result, bool) {
  vRaw, ok := cmd.Params["version"].(string)
  if !ok || vRaw == "" {
    return Result{Status: "rejected", Stderr: "missing version", ExitCode: 1}, false
  }
  if err := update.UpdateToVersion(ctx, r.cfg, r.client, vRaw); err != nil {
    return Result{Status: "error", Stderr: logging.Truncate(err.Error(), 2000), ExitCode: 1}, false
  }
  if !isSystemd() {
    _ = launchFallback(ctx)
  }
  return Result{Status: "ok", ExitCode: 0}, true
}

func (r *Runner) runAllowlisted(ctx context.Context, cmd Command) (Result, bool) {
  raw, ok := cmd.Params["command"].(string)
  if !ok || raw == "" {
    return Result{Status: "rejected", Stderr: "missing command", ExitCode: 1}, false
  }
  if !isAllowed(raw, r.cfg.AllowedCommands) {
    return Result{Status: "rejected", Stderr: "command not allowlisted", ExitCode: 1}, false
  }
  parts := strings.Fields(raw)
  if len(parts) == 0 {
    return Result{Status: "rejected", Stderr: "invalid command", ExitCode: 1}, false
  }

  cmdExec := exec.CommandContext(ctx, parts[0], parts[1:]...)
  out, err := cmdExec.CombinedOutput()
  if err != nil {
    return Result{Status: "error", Stderr: logging.Truncate(string(out), 2000), ExitCode: 1}, false
  }
  return Result{Status: "ok", Stdout: logging.Truncate(string(out), 2000), ExitCode: 0}, false
}

func isAllowed(cmd string, allowed []string) bool {
  if len(allowed) == 0 {
    return false
  }
  for _, a := range allowed {
    if strings.TrimSpace(a) == cmd {
      return true
    }
  }
  return false
}

var ErrCommandDisabled = errors.New("command disabled")

func ExitCode(err error) int {
  if err == nil {
    return 0
  }
  var exitErr *exec.ExitError
  if errors.As(err, &exitErr) {
    return exitErr.ExitCode()
  }
  return 1
}

func isSystemd() bool {
  if _, err := os.Stat("/run/systemd/system"); err == nil {
    if _, err := exec.LookPath("systemctl"); err == nil {
      return true
    }
  }
  return false
}

func launchFallback(ctx context.Context) error {
  script := "/home/fpp/media/plugins/fpp-monitor-agent/system/fpp-monitor-agent.sh"
  cmd := exec.CommandContext(ctx, "sh", "-c", fmt.Sprintf("nohup %s >/dev/null 2>&1 &", script))
  return cmd.Run()
}
