package structuredlogger

import (
	"bytes"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/flant/shell-operator/pkg/app"
)

func TestStructuredLoggerUsesProvidedLogger(t *testing.T) {
	t.Cleanup(func() {
		app.LogType = "text"
		app.LogLevel = "info"
	})

	app.LogType = "text"

	var buf bytes.Buffer
	logger := app.NewLogger(&buf)
	req := httptest.NewRequest("GET", "/debug", nil)

	entry := (&StructuredLogger{
		Logger:         logger,
		ComponentLabel: "http-server",
	}).NewLogEntry(req).(*StructuredLoggerEntry)
	entry.Write(200, 123, nil, time.Second, nil)

	got := buf.String()
	if strings.HasPrefix(got, "{") {
		t.Fatalf("expected text output, got json: %q", got)
	}
	if !strings.Contains(got, "component=http-server") {
		t.Fatalf("expected component label in output: %q", got)
	}
}
