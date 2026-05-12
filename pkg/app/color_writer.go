package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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

func stringValue(value any) string {
	switch v := value.(type) {
	case string:
		return v
	default:
		return ""
	}
}
