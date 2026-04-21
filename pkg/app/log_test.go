package app

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewLoggerRespectsLogType(t *testing.T) {
	t.Cleanup(func() {
		LogType = "text"
		LogLevel = "info"
	})

	testCases := []struct {
		name     string
		logType  string
		wantJSON bool
		wantANSI bool
	}{
		{name: "json", logType: "json", wantJSON: true, wantANSI: false},
		{name: "text", logType: "text", wantJSON: false, wantANSI: false},
		{name: "color", logType: "color", wantJSON: false, wantANSI: true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			LogType = tc.logType

			var buf bytes.Buffer
			logger := NewLogger(&buf)
			if tc.logType == "color" {
				logger.Warn("hello world")
			} else {
				logger.Info("hello world")
			}

			got := buf.String()
			if strings.HasPrefix(got, "{") != tc.wantJSON {
				t.Fatalf("json prefix mismatch for %s: %q", tc.logType, got)
			}
			if strings.Contains(got, "\x1b[") != tc.wantANSI {
				t.Fatalf("ansi escape mismatch for %s: %q", tc.logType, got)
			}
			if tc.logType != "json" {
				if strings.Contains(got, "logger=") {
					t.Fatalf("expected logger field to be removed for %s: %q", tc.logType, got)
				}
				if strings.Contains(got, "msg='") {
					t.Fatalf("expected message to be unquoted for %s: %q", tc.logType, got)
				}
			}
		})
	}
}

func TestColorWriterLeavesInfoDefaultAndColorsWarningsAndErrors(t *testing.T) {
	var buf bytes.Buffer
	writer := NewColorWriter(&buf)

	if _, err := writer.Write([]byte(`{"time":"2026-01-01T00:00:00Z","level":"info","msg":"plain"}` + "\n")); err != nil {
		t.Fatalf("write info log: %v", err)
	}

	infoOutput := buf.String()
	if strings.Contains(infoOutput, "\x1b[") {
		t.Fatalf("expected info log without ansi escapes, got %q", infoOutput)
	}
	if !strings.Contains(infoOutput, "INFO plain") {
		t.Fatalf("expected concise info format, got %q", infoOutput)
	}

	buf.Reset()

	if _, err := writer.Write([]byte(`{"time":"2026-01-01T00:00:00Z","level":"warn","msg":"warn"}` + "\n")); err != nil {
		t.Fatalf("write warn log: %v", err)
	}
	if !strings.Contains(buf.String(), "\x1b[") {
		t.Fatalf("expected warn log with ansi escapes, got %q", buf.String())
	}
	if !strings.Contains(buf.String(), "WARNING warn") {
		t.Fatalf("expected warning label in output, got %q", buf.String())
	}

	buf.Reset()

	if _, err := writer.Write([]byte(`{"time":"2026-01-01T00:00:00Z","level":"error","msg":"error"}` + "\n")); err != nil {
		t.Fatalf("write error log: %v", err)
	}
	if !strings.Contains(buf.String(), "\x1b[") {
		t.Fatalf("expected error log with ansi escapes, got %q", buf.String())
	}
}

func TestPrettyWriterFormatsJSONLogCompactly(t *testing.T) {
	var buf bytes.Buffer
	writer := NewPrettyWriter(&buf, false)

	line := `{"level":"info","logger":"shell-operator","msg":"Hook executed successfully","binding":"Monitor clusterconfiguration","event":"kubernetes","hook":"kubesphere/installRunner.py","queue":"main","task":"HookRun","time":"2026-04-21T15:18:49+08:00"}`
	if _, err := writer.Write([]byte(line + "\n")); err != nil {
		t.Fatalf("write pretty log: %v", err)
	}

	got := buf.String()
	if strings.Contains(got, "logger=") {
		t.Fatalf("expected logger field removed, got %q", got)
	}
	if strings.Contains(got, "msg='") {
		t.Fatalf("expected unquoted message, got %q", got)
	}
	if !strings.Contains(got, "2026-04-21T15:18:49+08:00 INFO Hook executed successfully | binding=Monitor clusterconfiguration event=kubernetes hook=kubesphere/installRunner.py queue=main task=HookRun") {
		t.Fatalf("unexpected compact output: %q", got)
	}
}
