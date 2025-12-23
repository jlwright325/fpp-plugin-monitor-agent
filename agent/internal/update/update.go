package update

import (
  "context"
  "crypto/sha256"
  "encoding/hex"
  "encoding/json"
  "fmt"
  "io"
  "net/http"
  "os"
  "path/filepath"
  "runtime"
  "strings"

  "github.com/your-org/fpp-plugin-monitor-agent/agent/internal/client"
  "github.com/your-org/fpp-plugin-monitor-agent/agent/internal/config"
)

type Manifest struct {
  URL    string `json:"url"`
  Sha256 string `json:"sha256"`
}

func UpdateToVersion(ctx context.Context, cfg *config.Config, cl *client.Client, version string) error {
  manifest, err := fetchManifest(ctx, cfg, cl, version)
  if err != nil {
    return err
  }

  if manifest.URL == "" || manifest.Sha256 == "" {
    return fmt.Errorf("manifest missing url or sha256")
  }

  tmpDir := os.TempDir()
  tmpFile := filepath.Join(tmpDir, fmt.Sprintf("fpp-monitor-agent-%s", version))

  if err := downloadFile(ctx, manifest.URL, tmpFile); err != nil {
    return err
  }

  if err := verifySha256(tmpFile, manifest.Sha256); err != nil {
    return err
  }

  return swapBinary(tmpFile)
}

func fetchManifest(ctx context.Context, cfg *config.Config, cl *client.Client, version string) (*Manifest, error) {
  base := strings.TrimRight(cfg.ApiBaseURL, "/")
  if cfg.UpdateBaseURL != "" {
    base = strings.TrimRight(cfg.UpdateBaseURL, "/")
  }
  platform := runtime.GOARCH
  if runtime.GOARCH == "arm" {
    platform = fmt.Sprintf("armv%d", runtime.GOARM)
  }
  url := fmt.Sprintf("%s/v1/agent/releases/manifest?channel=%s&version=%s&platform=%s", base, cfg.UpdateChannel, version, platform)

  resp, body, err := cl.DoJSONWithRetry(ctx, http.MethodGet, url, nil, nil)
  if err != nil {
    return nil, err
  }
  if resp.StatusCode >= 300 {
    return nil, fmt.Errorf("manifest status %d", resp.StatusCode)
  }

  var m Manifest
  if err := json.Unmarshal(body, &m); err != nil {
    return nil, err
  }
  return &m, nil
}

func downloadFile(ctx context.Context, url, dest string) error {
  req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
  if err != nil {
    return err
  }
  resp, err := http.DefaultClient.Do(req)
  if err != nil {
    return err
  }
  defer resp.Body.Close()
  if resp.StatusCode >= 300 {
    return fmt.Errorf("download status %d", resp.StatusCode)
  }

  tmp, err := os.Create(dest)
  if err != nil {
    return err
  }
  defer tmp.Close()

  _, err = io.Copy(tmp, resp.Body)
  return err
}

func verifySha256(path, expected string) error {
  f, err := os.Open(path)
  if err != nil {
    return err
  }
  defer f.Close()

  h := sha256.New()
  if _, err := io.Copy(h, f); err != nil {
    return err
  }
  actual := hex.EncodeToString(h.Sum(nil))
  expected = strings.ToLower(strings.TrimSpace(expected))
  if actual != expected {
    return fmt.Errorf("sha256 mismatch")
  }
  return nil
}

func swapBinary(tmpPath string) error {
  target := os.Args[0]
  dir := filepath.Dir(target)
  tmpTarget := filepath.Join(dir, ".fpp-monitor-agent.new")

  if err := os.Rename(tmpPath, tmpTarget); err != nil {
    return err
  }

  if err := os.Chmod(tmpTarget, 0755); err != nil {
    return err
  }

  return os.Rename(tmpTarget, target)
}
