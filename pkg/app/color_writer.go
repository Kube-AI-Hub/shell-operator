package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/muesli/termenv"
)

type PrettyWriter struct {
	output *termenv.Output
	target io.Writer
	mu     sync.Mutex
	color  bool
	buf    []byte
}

func NewPrettyWriter(target io.Writer, color bool) *PrettyWriter {
	return &PrettyWriter{
		output: termenv.NewOutput(target, termenv.WithProfile(termenv.TrueColor)),
		target: target,
		color:  color,
	}
}

func NewColorWriter(target io.Writer) *PrettyWriter {
	return NewPrettyWriter(target, true)
}

func (w *PrettyWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	total := len(p)

	w.buf = append(w.buf, p...)
	for len(w.buf) > 0 {
		idx := bytes.IndexByte(w.buf, '\n')
		if idx == -1 {
			break
		}

		line := string(w.buf[:idx])
		if _, err := io.WriteString(w.target, w.render(line, true)); err != nil {
			return 0, err
		}

		w.buf = w.buf[idx+1:]
	}

	return total, nil
}

func (w *PrettyWriter) render(line string, appendNewline bool) string {
	if strings.TrimSpace(line) == "" {
		if appendNewline {
			return "\n"
		}
		return line
	}

	formatted := line
	level := ""

	var payload map[string]any
	if err := json.Unmarshal([]byte(line), &payload); err == nil {
		formatted, level = formatPrettyLog(payload)
	}

	if w.color {
		styled := w.output.String(formatted)
		switch level {
		case "WARNING":
			styled = styled.Foreground(termenv.ANSIBrightYellow)
		case "ERROR":
			styled = styled.Foreground(termenv.ANSIBrightRed)
		case "FATAL":
			styled = styled.Foreground(termenv.ANSIRed).Bold()
		}
		formatted = styled.String()
	}

	if appendNewline {
		return formatted + "\n"
	}

	return formatted
}

func formatPrettyLog(payload map[string]any) (string, string) {
	timeValue := stringValue(payload["time"])
	level := prettyLevel(stringValue(payload["level"]))
	message := stringValue(payload["msg"])
	if message == "" {
		message = "-"
	}

	context := formatContext(payload)
	if context != "" {
		return fmt.Sprintf("%s %s %s | %s", timeValue, level, message, context), level
	}

	return fmt.Sprintf("%s %s %s", timeValue, level, message), level
}

func prettyLevel(level string) string {
	switch strings.ToUpper(level) {
	case "WARN":
		return "WARNING"
	case "ERROR":
		return "ERROR"
	case "FATAL":
		return "FATAL"
	case "DEBUG":
		return "DEBUG"
	case "TRACE":
		return "TRACE"
	case "INFO":
		return "INFO"
	default:
		return strings.ToUpper(level)
	}
}

func formatContext(payload map[string]any) string {
	priority := []string{
		"binding",
		"event",
		"hook",
		"queue",
		"task",
		"output",
		"error",
		"component",
		"operator.component",
		"http_method",
		"uri",
		"resp_status",
		"resp_elapsed_ms",
		"address",
		"port",
	}

	rename := map[string]string{
		"operator.component": "component",
		"http_method":        "method",
		"resp_status":        "status",
		"resp_elapsed_ms":    "elapsed_ms",
	}

	used := make(map[string]struct{}, len(priority))
	parts := make([]string, 0, len(priority))

	for _, key := range priority {
		value, ok := payload[key]
		if !ok {
			continue
		}

		rendered, ok := renderField(renameKey(rename, key), value)
		if !ok {
			continue
		}

		parts = append(parts, rendered)
		used[key] = struct{}{}
	}

	keys := make([]string, 0, len(payload))
	for key := range payload {
		if _, ok := used[key]; ok || skipField(key) {
			continue
		}
		keys = append(keys, key)
	}

	sort.Strings(keys)
	for _, key := range keys {
		rendered, ok := renderField(key, payload[key])
		if !ok {
			continue
		}
		parts = append(parts, rendered)
		if len(parts) >= 8 {
			break
		}
	}

	return strings.Join(parts, " ")
}

func renameKey(mapping map[string]string, key string) string {
	if short, ok := mapping[key]; ok {
		return short
	}
	return key
}

func skipField(key string) bool {
	if key == "level" || key == "logger" || key == "msg" || key == "source" || key == "stacktrace" || key == "time" || key == ProxyJsonLogKey {
		return true
	}

	if strings.HasPrefix(key, "hook_") || strings.HasPrefix(key, "hook.") || strings.HasPrefix(key, "hook_event_data") {
		return true
	}

	return false
}

func renderField(key string, value any) (string, bool) {
	if value == nil {
		return "", false
	}

	switch v := value.(type) {
	case string:
		if v == "" {
			return "", false
		}
		return key + "=" + truncate(v), true
	case bool:
		return key + "=" + strconv.FormatBool(v), true
	case float64:
		return key + "=" + strconv.FormatFloat(v, 'f', -1, 64), true
	case int:
		return key + "=" + strconv.Itoa(v), true
	case int64:
		return key + "=" + strconv.FormatInt(v, 10), true
	case uint64:
		return key + "=" + strconv.FormatUint(v, 10), true
	default:
		return "", false
	}
}

func stringValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}

func truncate(value string) string {
	const maxLen = 120
	if len(value) <= maxLen {
		return value
	}
	return value[:maxLen] + "..."
}
