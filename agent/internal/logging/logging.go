package logging

import (
  "encoding/json"
  "fmt"
  "os"
  "time"
)

type Logger struct {
  out *os.File
}

type entry struct {
  Level string                 `json:"level"`
  Time  string                 `json:"time"`
  Msg   string                 `json:"msg"`
  Fields map[string]any        `json:"fields,omitempty"`
}

func New() *Logger {
  return &Logger{out: os.Stdout}
}

func (l *Logger) log(level, msg string, fields map[string]any) {
  e := entry{
    Level: level,
    Time:  time.Now().UTC().Format(time.RFC3339),
    Msg:   msg,
    Fields: fields,
  }
  b, err := json.Marshal(e)
  if err != nil {
    fmt.Fprintf(l.out, "{\"level\":\"error\",\"msg\":\"failed to marshal log\",\"err\":%q}\n", err.Error())
    return
  }
  fmt.Fprintln(l.out, string(b))
}

func (l *Logger) Info(msg string, fields map[string]any) {
  l.log("info", msg, fields)
}

func (l *Logger) Error(msg string, fields map[string]any) {
  l.log("error", msg, fields)
}

func Truncate(s string, max int) string {
  if max <= 0 || len(s) <= max {
    return s
  }
  return s[:max]
}
