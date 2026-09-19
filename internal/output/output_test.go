package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/korabcenaj/syscheck/internal/check"
	"github.com/korabcenaj/syscheck/internal/host"
	"github.com/korabcenaj/syscheck/internal/output"
)

func sampleMultiTargetReport() output.Report {
	return output.Report{
		Timestamp: time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC),
		Overall:   check.StatusOK,
		Targets: []output.TargetReport{
			{
				Target: "localhost",
				HostInfo: &host.Info{
					Hostname:   "web-01",
					PrettyName: "Ubuntu 22.04 LTS",
					Uptime:     180000 * time.Second,
				},
				Overall: check.StatusOK,
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
			},
			{
				Target:  "192.168.1.10",
				Overall: check.StatusOK,
				Checks: []check.Result{
					{
						Name:    "Reachability (192.168.1.10:22)",
						Status:  check.StatusOK,
						Message: "host reachable (TCP handshake in 2ms)",
					},
				},
			},
		},
	}
}

func TestTextFormatter(t *testing.T) {
	formatter := &output.TextFormatter{}
	var buf bytes.Buffer

	report := sampleMultiTargetReport()
	if err := formatter.Format(&buf, report); err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, "Target: localhost (Ubuntu 22.04 LTS, uptime: 2d 2h 0m)") {
		t.Errorf("expected localhost target header in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Target: 192.168.1.10") {
		t.Errorf("expected remote target header in output, got:\n%s", out)
	}
	if !strings.Contains(out, "STATUS") || !strings.Contains(out, "CHECK") || !strings.Contains(out, "MESSAGE") {
		t.Errorf("expected table headers in output, got:\n%s", out)
	}
	if !strings.Contains(out, "Overall Cluster Health: OK") {
		t.Errorf("expected 'Overall Cluster Health: OK' in output, got:\n%s", out)
	}
}

func TestJSONFormatter(t *testing.T) {
	formatter := &output.JSONFormatter{}
	var buf bytes.Buffer

	report := sampleMultiTargetReport()
	if err := formatter.Format(&buf, report); err != nil {
		t.Fatalf("Format() returned error: %v", err)
	}

	var decoded output.Report
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode JSON output: %v\nOutput was:\n%s", err, buf.String())
	}

	if decoded.Overall != check.StatusOK {
		t.Errorf("Overall = %v, want OK", decoded.Overall)
	}
	if len(decoded.Targets) != 2 {
		t.Fatalf("len(Targets) = %d, want 2", len(decoded.Targets))
	}
	if decoded.Targets[0].Target != "localhost" || decoded.Targets[0].HostInfo.PrettyName != "Ubuntu 22.04 LTS" {
		t.Errorf("decoded.Targets[0] = %+v", decoded.Targets[0])
	}
	if decoded.Targets[1].Target != "192.168.1.10" {
		t.Errorf("decoded.Targets[1] = %+v", decoded.Targets[1])
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
