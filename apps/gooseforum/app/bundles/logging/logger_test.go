package logging

import (
	"bytes"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestParseLogLevel(t *testing.T) {
	cases := []struct {
		name     string
		raw      string
		isDebug  bool
		explicit bool
		want     zapcore.Level
		wantErr  bool
	}{
		{name: "explicit debug", raw: "debug", explicit: true, want: zapcore.DebugLevel},
		{name: "explicit info", raw: "info", explicit: true, want: zapcore.InfoLevel},
		{name: "explicit warn", raw: "warn", explicit: true, want: zapcore.WarnLevel},
		{name: "explicit error", raw: "error", explicit: true, want: zapcore.ErrorLevel},
		{name: "explicit uppercase", raw: "WARN", explicit: true, want: zapcore.WarnLevel},
		{name: "explicit invalid", raw: "verbose", explicit: true, wantErr: true},
		{name: "implicit debug mode", raw: "info", isDebug: true, explicit: false, want: zapcore.DebugLevel},
		{name: "implicit prod mode", raw: "info", isDebug: false, explicit: false, want: zapcore.InfoLevel},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseLogLevel(tc.raw, tc.isDebug, tc.explicit)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseLogLevel(%q) expected error, got level %v", tc.raw, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseLogLevel(%q) unexpected error: %v", tc.raw, err)
			}
			if got != tc.want {
				t.Fatalf("parseLogLevel(%q) = %v, want %v", tc.raw, got, tc.want)
			}
		})
	}
}

// TestWarnLevelCoreDropsInfo verifies that a core constructed with a WarnLevel
// threshold (the error file core) drops info entries but keeps warn and error.
func TestWarnLevelCoreDropsInfo(t *testing.T) {
	var buf bytes.Buffer
	core := newZapCore(zapcore.AddSync(&buf), zapcore.WarnLevel)
	logger := zap.New(core)

	logger.Info("should-be-dropped")
	logger.Warn("warning-kept")
	logger.Error("error-kept")
	_ = logger.Sync()

	out := buf.String()
	if strings.Contains(out, "should-be-dropped") {
		t.Fatal("info entry leaked into warn-level core")
	}
	if !strings.Contains(out, "warning-kept") {
		t.Fatal("warn entry missing from warn-level core")
	}
	if !strings.Contains(out, "error-kept") {
		t.Fatal("error entry missing from warn-level core")
	}
}

// TestConsoleFormatProducesPlainText verifies the console encoder produces
// non-JSON output.
func TestConsoleFormatProducesPlainText(t *testing.T) {
	old := logFormat
	logFormat = "console"
	defer func() { logFormat = old }()

	var buf bytes.Buffer
	core := newZapCore(zapcore.AddSync(&buf), zapcore.InfoLevel)
	logger := zap.New(core)
	logger.Info("console-line")
	_ = logger.Sync()

	out := buf.String()
	if out == "" {
		t.Fatal("console encoder produced no output")
	}
	if strings.HasPrefix(strings.TrimSpace(out), "{") {
		t.Fatalf("console encoder produced JSON: %q", out)
	}
	if !strings.Contains(out, "console-line") {
		t.Fatalf("console output missing message: %q", out)
	}
}

// TestJSONFormatProducesJSON verifies the default encoder emits JSON lines.
func TestJSONFormatProducesJSON(t *testing.T) {
	var buf bytes.Buffer
	core := newZapCore(zapcore.AddSync(&buf), zapcore.InfoLevel)
	logger := zap.New(core)
	logger.Info("json-line")
	_ = logger.Sync()

	out := strings.TrimSpace(buf.String())
	if !strings.HasPrefix(out, "{") || !strings.Contains(out, `"msg":"json-line"`) {
		t.Fatalf("json encoder produced unexpected output: %q", out)
	}
}
