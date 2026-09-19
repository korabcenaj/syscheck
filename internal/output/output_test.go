package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"syscheck/internal/check"
	"syscheck/internal/output"
)

func sampleReport() output.Report {
	return output.Report{
		Timestamp: time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC),
		Hostname:  "test-server-01",
		Overall:   check.StatusOK,
		Checks: []check.Result{
			{
				Name:    "Load Average",
				Status:  check.StatusOK,
				Message: "normal load: 0.50 (1m), 0.60 (5m), 0.70 (15m)",
			},
			{
				Name:    "Memory",
				Status:  check.StatusOK,
				Message: "normal memory usage: 4.00 GiB / 16.00 GiB (25.0% used)",
			},
		},
	}
}

func TestTextFormatter(t *testing.T) {
	formatter := &output.TextFormatter{}
	var buf bytes.Buffer

	report := sampleReport()
	if err := formatter.Format(&buf, report); err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "STATUS") || !strings.Contains(out, "CHECK") || !strings.Contains(out, "MESSAGE") {
		t.Errorf("expected table headers in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Load Average") {
		t.Errorf("expected 'Load Average' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Overall Health: OK") {
		t.Errorf("expected 'Overall Health: OK' in output, got:\n%s", out)
	}
}

func TestJSONFormatter(t *testing.T) {
	formatter := &output.JSONFormatter{}
	var buf bytes.Buffer

	report := sampleReport()
	if err := formatter.Format(&buf, report); err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	var decoded output.Report
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode JSON output: %v\nOutput was:\n%s", err, buf.String())
	}

	if decoded.Hostname != "test-server-01" {
		t.Errorf("Hostname = %q, want %q", decoded.Hostname, "test-server-01")
	}
	if decoded.Overall != check.StatusOK {
		t.Errorf("Overall = %v, want OK", decoded.Overall)
	}
	if len(decoded.Checks) != 2 {
		t.Fatalf("len(Checks) = %d, want 2", len(decoded.Checks))
	}
	if decoded.Checks[0].Name != "Load Average" || decoded.Checks[0].Status != check.StatusOK {
		t.Errorf("decoded.Checks[0] = %+v", decoded.Checks[0])
	}
}

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		format  string
		wantErr bool
	}{
		{"text", false},
		{"", false},
		{"json", false},
		{"yaml", true},
		{"invalid", true},
	}

	for _, tt := range tests {
		t.Run("format="+tt.format, func(t *testing.T) {
			_, err := output.NewFormatter(tt.format)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewFormatter(%q) error = %v, wantErr %v", tt.format, err, tt.wantErr)
			}
		})
	}
}
