package main

import (
  "context"
  "net/http"
  "os"
  "os/signal"
  "strings"
  "syscall"
  "time"

  "github.com/your-org/fpp-plugin-monitor-agent/agent/internal/client"
  "github.com/your-org/fpp-plugin-monitor-agent/agent/internal/commands"
  "github.com/your-org/fpp-plugin-monitor-agent/agent/internal/config"
  "github.com/your-org/fpp-plugin-monitor-agent/agent/internal/fpp"
  "github.com/your-org/fpp-plugin-monitor-agent/agent/internal/heartbeat"
  "github.com/your-org/fpp-plugin-monitor-agent/agent/internal/logging"
)

var Version = "dev"

func main() {
  log := logging.New()

  cfg, err := config.Load()
  if err != nil {
    log.Error("config load failed", map[string]any{"error": err.Error()})
  }

  apiClient := client.New(10 * time.Second)
  fppHTTP := &http.Client{Timeout: 2 * time.Second}

  ctx, cancel := context.WithCancel(context.Background())
  defer cancel()

  sigs := make(chan os.Signal, 1)
  signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
  go func() {
    <-sigs
    cancel()
  }()

  hbTicker := time.NewTicker(cfg.HeartbeatInterval())
  cmdTicker := time.NewTicker(cfg.CommandPollInterval())
  defer hbTicker.Stop()
  defer cmdTicker.Stop()

  runner := commands.NewRunner(cfg, apiClient, log, Version)

  log.Info("agent started", map[string]any{"version": Version})

  for {
    select {
    case <-ctx.Done():
      log.Info("agent shutting down", nil)
      return
    case <-hbTicker.C:
      go sendHeartbeat(ctx, cfg, apiClient, fppHTTP, log)
    case <-cmdTicker.C:
      go runner.PollAndExecute(ctx)
    }
  }
}

func sendHeartbeat(ctx context.Context, cfg *config.Config, apiClient *client.Client, fppHTTP *http.Client, log *logging.Logger) {
  if cfg.DeviceID == "" || cfg.DeviceToken == "" {
    log.Info("heartbeat skipped; missing device_id or device_token", nil)
    return
  }

  st := fpp.Collect(ctx, fppHTTP)
  payload := heartbeat.Build(cfg.DeviceID, Version, st)

  url := strings.TrimRight(cfg.ApiBaseURL, "/") + "/v1/ingest/heartbeat"
  headers := map[string]string{"Authorization": "Bearer " + cfg.DeviceToken}

  resp, _, err := apiClient.DoJSONWithRetry(ctx, http.MethodPost, url, headers, payload)
  if err != nil {
    log.Error("heartbeat failed", map[string]any{"error": err.Error()})
    return
  }
  if resp.StatusCode >= 300 {
    log.Error("heartbeat non-200", map[string]any{"status": resp.StatusCode})
    return
  }
  log.Info("heartbeat sent", map[string]any{"status": resp.StatusCode})
}
