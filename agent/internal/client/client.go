package client

import (
  "bytes"
  "context"
  "encoding/json"
  "io"
  "net/http"
  "time"
)

type Client struct {
  http *http.Client
}

func New(timeout time.Duration) *Client {
  return &Client{http: &http.Client{Timeout: timeout}}
}

func (c *Client) DoJSON(ctx context.Context, method, url string, headers map[string]string, payload any) (*http.Response, []byte, error) {
  var body io.Reader
  if payload != nil {
    buf, err := json.Marshal(payload)
    if err != nil {
      return nil, nil, err
    }
    body = bytes.NewBuffer(buf)
  }

  req, err := http.NewRequestWithContext(ctx, method, url, body)
  if err != nil {
    return nil, nil, err
  }
  if payload != nil {
    req.Header.Set("Content-Type", "application/json")
  }
  for k, v := range headers {
    req.Header.Set(k, v)
  }

  resp, err := c.http.Do(req)
  if err != nil {
    return nil, nil, err
  }
  defer resp.Body.Close()

  b, err := io.ReadAll(resp.Body)
  if err != nil {
    return resp, nil, err
  }
  return resp, b, nil
}

func (c *Client) DoJSONWithRetry(ctx context.Context, method, url string, headers map[string]string, payload any) (*http.Response, []byte, error) {
  var lastErr error
  for i := 0; i < 3; i++ {
    resp, body, err := c.DoJSON(ctx, method, url, headers, payload)
    if err == nil && resp.StatusCode < 500 {
      return resp, body, nil
    }
    lastErr = err
    select {
    case <-ctx.Done():
      return nil, nil, ctx.Err()
    case <-time.After(time.Duration(400*(i+1)) * time.Millisecond):
    }
  }
  return nil, nil, lastErr
}
