package config

import (
  "encoding/json"
  "os"
  "strconv"
  "strings"
  "time"
)

const DefaultPath = "/home/fpp/media/config/fpp-monitor-agent.json"
const apiBaseURL = "https://api.showops.io"

// Config defines runtime settings for the agent.
type Config struct {
  DeviceID               string   `json:"device_id"`
  DeviceToken            string   `json:"device_token"`
  ApiBaseURL             string   `json:"api_base_url"`
  HeartbeatIntervalSec   int      `json:"heartbeat_interval_sec"`
  CommandPollIntervalSec int      `json:"command_poll_interval_sec"`
  RebootEnabled          bool     `json:"reboot_enabled"`
  RestartFPPCommand      string   `json:"restart_fpp_command"`
  UpdateChannel          string   `json:"update_channel"`
  AllowedCommands        []string `json:"allowed_commands,omitempty"`
  UpdateBaseURL          string   `json:"update_base_url,omitempty"`
}

func Load() (*Config, error) {
  path := os.Getenv("FPP_MONITOR_AGENT_CONFIG")
  if path == "" {
    path = DefaultPath
  }

  cfg := &Config{}
  if b, err := os.ReadFile(path); err == nil {
    _ = json.Unmarshal(b, cfg)
  }

  applyEnvOverrides(cfg)
  applyDefaults(cfg)
  return cfg, nil
}

func applyDefaults(cfg *Config) {
  cfg.ApiBaseURL = apiBaseURL
  if cfg.HeartbeatIntervalSec <= 0 {
    cfg.HeartbeatIntervalSec = 10
  }
  if cfg.CommandPollIntervalSec <= 0 {
    cfg.CommandPollIntervalSec = 5
  }
  if cfg.RestartFPPCommand == "" {
    cfg.RestartFPPCommand = "systemctl restart fppd"
  }
  if cfg.UpdateChannel == "" {
    cfg.UpdateChannel = "stable"
  }
}

func applyEnvOverrides(cfg *Config) {
  setString(&cfg.DeviceID, "FPP_MONITOR_AGENT_DEVICE_ID")
  setString(&cfg.DeviceToken, "FPP_MONITOR_AGENT_DEVICE_TOKEN")
  setString(&cfg.RestartFPPCommand, "FPP_MONITOR_AGENT_RESTART_FPP_COMMAND")
  setString(&cfg.UpdateChannel, "FPP_MONITOR_AGENT_UPDATE_CHANNEL")
  setString(&cfg.UpdateBaseURL, "FPP_MONITOR_AGENT_UPDATE_BASE_URL")

  setInt(&cfg.HeartbeatIntervalSec, "FPP_MONITOR_AGENT_HEARTBEAT_INTERVAL_SEC")
  setInt(&cfg.CommandPollIntervalSec, "FPP_MONITOR_AGENT_COMMAND_POLL_INTERVAL_SEC")

  if v := os.Getenv("FPP_MONITOR_AGENT_REBOOT_ENABLED"); v != "" {
    cfg.RebootEnabled = strings.EqualFold(v, "true") || v == "1"
  }

  if v := os.Getenv("FPP_MONITOR_AGENT_ALLOWED_COMMANDS"); v != "" {
    cfg.AllowedCommands = strings.Split(v, ",")
  }
}

func setString(target *string, key string) {
  if v := os.Getenv(key); v != "" {
    *target = v
  }
}

func setInt(target *int, key string) {
  if v := os.Getenv(key); v != "" {
    if parsed, err := strconv.Atoi(v); err == nil {
      *target = parsed
    }
  }
}

func (c *Config) HeartbeatInterval() time.Duration {
  return time.Duration(c.HeartbeatIntervalSec) * time.Second
}

func (c *Config) CommandPollInterval() time.Duration {
  return time.Duration(c.CommandPollIntervalSec) * time.Second
}
