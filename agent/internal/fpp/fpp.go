package fpp

import (
  "context"
  "encoding/json"
  "net"
  "net/http"
  "os"
  "strings"
  "syscall"
  "time"
)

type State struct {
  FPPVersion string
  Hostname   string
  Mode       string
  Playing    bool
  Playlist   string
  Sequence   string
  CPUPercent float64
  MemPercent float64
  DiskFreeMB uint64
  IPs        []string
  Raw        map[string]any
}

func Collect(ctx context.Context, client *http.Client) State {
  st := State{
    Raw: map[string]any{},
  }
  st.Hostname, _ = os.Hostname()
  st.FPPVersion = readFPPVersion()

  status := fetchStatus(ctx, client)
  for k, v := range status {
    st.Raw[k] = v
  }

  st.Playing = getBool(status, "status") || getBool(status, "playing")
  st.Mode = getString(status, "mode")
  st.Playlist = getString(status, "playlist")
  st.Sequence = getString(status, "sequence")

  st.CPUPercent = cpuPercent()
  st.MemPercent = memPercent()
  st.DiskFreeMB = diskFreeMB("/home/fpp")
  st.IPs = ipList()

  return st
}

func readFPPVersion() string {
  candidates := []string{
    "/home/fpp/media/config/version",
    "/home/fpp/media/config/fpp-version",
  }
  for _, path := range candidates {
    if b, err := os.ReadFile(path); err == nil {
      return strings.TrimSpace(string(b))
    }
  }
  return ""
}

func fetchStatus(ctx context.Context, client *http.Client) map[string]any {
  req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://localhost/api/fppd/status", nil)
  if err != nil {
    return map[string]any{}
  }
  resp, err := client.Do(req)
  if err != nil {
    return map[string]any{}
  }
  defer resp.Body.Close()

  if resp.StatusCode >= 300 {
    return map[string]any{}
  }

  var data map[string]any
  if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
    return map[string]any{}
  }
  return data
}

func getString(m map[string]any, key string) string {
  if v, ok := m[key]; ok {
    if s, ok := v.(string); ok {
      return s
    }
  }
  return ""
}

func getBool(m map[string]any, key string) bool {
  if v, ok := m[key]; ok {
    switch val := v.(type) {
    case bool:
      return val
    case string:
      return strings.EqualFold(val, "true") || val == "1"
    case float64:
      return val != 0
    }
  }
  return false
}

func ipList() []string {
  ifaces, err := net.Interfaces()
  if err != nil {
    return nil
  }
  ips := []string{}
  for _, iface := range ifaces {
    addrs, err := iface.Addrs()
    if err != nil {
      continue
    }
    for _, addr := range addrs {
      ipNet, ok := addr.(*net.IPNet)
      if !ok || ipNet.IP.IsLoopback() {
        continue
      }
      if ip4 := ipNet.IP.To4(); ip4 != nil {
        ips = append(ips, ip4.String())
      }
    }
  }
  return ips
}

func diskFreeMB(path string) uint64 {
  var stat syscall.Statfs_t
  if err := syscall.Statfs(path, &stat); err != nil {
    return 0
  }
  free := stat.Bavail * uint64(stat.Bsize)
  return free / (1024 * 1024)
}

func memPercent() float64 {
  b, err := os.ReadFile("/proc/meminfo")
  if err != nil {
    return 0
  }
  lines := strings.Split(string(b), "\n")
  var total, available float64
  for _, line := range lines {
    if strings.HasPrefix(line, "MemTotal:") {
      total = parseMemValue(line)
    }
    if strings.HasPrefix(line, "MemAvailable:") {
      available = parseMemValue(line)
    }
  }
  if total == 0 {
    return 0
  }
  used := total - available
  return (used / total) * 100
}

func parseMemValue(line string) float64 {
  fields := strings.Fields(line)
  if len(fields) < 2 {
    return 0
  }
  val, err := parseFloat(fields[1])
  if err != nil {
    return 0
  }
  return val
}

func parseFloat(s string) (float64, error) {
  var f float64
  for _, c := range s {
    if c < '0' || c > '9' {
      break
    }
    f = f*10 + float64(c-'0')
  }
  return f, nil
}

func cpuPercent() float64 {
  total1, idle1 := readCPUStat()
  time.Sleep(150 * time.Millisecond)
  total2, idle2 := readCPUStat()
  if total2 <= total1 {
    return 0
  }
  totalDelta := total2 - total1
  idleDelta := idle2 - idle1
  if totalDelta == 0 {
    return 0
  }
  used := float64(totalDelta-idleDelta) / float64(totalDelta)
  return used * 100
}

func readCPUStat() (uint64, uint64) {
  b, err := os.ReadFile("/proc/stat")
  if err != nil {
    return 0, 0
  }
  lines := strings.Split(string(b), "\n")
  if len(lines) == 0 {
    return 0, 0
  }
  fields := strings.Fields(lines[0])
  if len(fields) < 5 {
    return 0, 0
  }
  var total uint64
  for i := 1; i < len(fields); i++ {
    total += parseUint(fields[i])
  }
  idle := parseUint(fields[4])
  return total, idle
}

func parseUint(s string) uint64 {
  var n uint64
  for _, c := range s {
    if c < '0' || c > '9' {
      break
    }
    n = n*10 + uint64(c-'0')
  }
  return n
}
