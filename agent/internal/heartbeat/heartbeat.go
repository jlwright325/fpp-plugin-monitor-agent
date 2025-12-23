package heartbeat

import (
  "time"

  "github.com/your-org/fpp-plugin-monitor-agent/agent/internal/fpp"
)

type Payload struct {
  PayloadVersion int               `json:"payload_version"`
  SentAt         int64             `json:"sent_at"`
  Device         Device            `json:"device"`
  State          State             `json:"state"`
  Resources      Resources         `json:"resources"`
  Extra          map[string]any    `json:"extra"`
}

type Device struct {
  DeviceID     string `json:"device_id"`
  Hostname     string `json:"hostname"`
  FPPVersion   string `json:"fpp_version"`
  AgentVersion string `json:"agent_version"`
}

type State struct {
  Playing  bool   `json:"playing"`
  Mode     string `json:"mode"`
  Playlist string `json:"playlist"`
  Sequence string `json:"sequence"`
}

type Resources struct {
  CPUPercent   float64 `json:"cpu_percent"`
  MemoryPercent float64 `json:"memory_percent"`
  DiskFreeMB   uint64 `json:"disk_free_mb"`
}

func Build(deviceID, agentVersion string, st fpp.State) Payload {
  return Payload{
    PayloadVersion: 1,
    SentAt:         time.Now().Unix(),
    Device: Device{
      DeviceID:     deviceID,
      Hostname:     st.Hostname,
      FPPVersion:   st.FPPVersion,
      AgentVersion: agentVersion,
    },
    State: State{
      Playing:  st.Playing,
      Mode:     st.Mode,
      Playlist: st.Playlist,
      Sequence: st.Sequence,
    },
    Resources: Resources{
      CPUPercent:   st.CPUPercent,
      MemoryPercent: st.MemPercent,
      DiskFreeMB:   st.DiskFreeMB,
    },
    Extra: map[string]any{
      "raw": map[string]any{
        "ips": st.IPs,
        "fpp": st.Raw,
      },
    },
  }
}
