package main

import (
	"bytes"
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	t.Run("returns 0 on -help", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"-help"}, &stdout, &stderr)
		if code != 0 {
			t.Errorf("exit code = %d, want 0", code)
		}
		if !strings.Contains(stderr.String(), "Usage of syscheck:") {
			t.Errorf("stderr does not contain usage: %q", stderr.String())
		}
	})

	t.Run("returns 3 on invalid configuration", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		// warn > crit is invalid
		code := run([]string{"-mem-warn", "95", "-mem-crit", "80"}, &stdout, &stderr)
		if code != 3 {
			t.Errorf("exit code = %d, want 3", code)
		}
		if !strings.Contains(stderr.String(), "Error:") {
			t.Errorf("stderr does not contain Error: %q", stderr.String())
		}
	})

	t.Run("executes successfully on host", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{}, &stdout, &stderr)
		// On this host, healthy execution will return 0 (or 1 if elevated load/memory)
		if code != 0 && code != 1 && code != 2 {
			t.Errorf("unexpected exit code: %d, stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "Overall Health:") {
			t.Errorf("stdout missing summary: %q", stdout.String())
		}
	})

	t.Run("executes with json output format", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"-format", "json"}, &stdout, &stderr)
		if code != 0 && code != 1 && code != 2 {
			t.Errorf("unexpected exit code: %d, stderr: %s", code, stderr.String())
		}
		var decoded map[string]any
		if err := json.Unmarshal(stdout.Bytes(), &decoded); err != nil {
			t.Fatalf("expected valid JSON output, got error: %v\nOutput: %s", err, stdout.String())
		}
		if _, ok := decoded["overall"]; !ok {
			t.Errorf("missing 'overall' key in JSON: %+v", decoded)
		}
	})

	t.Run("executes with explicit localhost target", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"localhost"}, &stdout, &stderr)
		if code != 0 && code != 1 && code != 2 {
			t.Errorf("unexpected exit code: %d, stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "Target: localhost") {
			t.Errorf("stdout missing Target: localhost: %q", stdout.String())
		}
	})

	t.Run("executes multiple targets", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"localhost", "127.0.0.1"}, &stdout, &stderr)
		if code != 0 && code != 1 && code != 2 {
			t.Errorf("unexpected exit code: %d, stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "Overall Cluster Health:") {
			t.Errorf("stdout missing cluster summary: %q", stdout.String())
		}
	})

	t.Run("executes with procs check reporting critical on missing process", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"-procs", "nonexistent-daemon-99999"}, &stdout, &stderr)
		if code != 2 {
			t.Errorf("exit code = %d, want 2 (Critical) for missing process. Stdout: %s", code, stdout.String())
		}
		if !strings.Contains(stdout.String(), "Process (nonexistent-daemon-99999)") {
			t.Errorf("stdout missing Process check row: %q", stdout.String())
		}
		if !strings.Contains(stdout.String(), "CRITICAL") {
			t.Errorf("stdout missing CRITICAL status: %q", stdout.String())
		}
	})

	t.Run("executes with procs check on systemd/init", func(t *testing.T) {
		var stdout, stderr bytes.Buffer
		code := run([]string{"-procs", "systemd"}, &stdout, &stderr)
		// systemd is PID 1 on this system, so the check should be present in output
		if !strings.Contains(stdout.String(), "Process (systemd)") {
			t.Errorf("stdout missing Process (systemd) check row: %q", stdout.String())
		}
		_ = code
	})

	t.Run("executes with tcp check against live listener", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to listen: %v", err)
		}
		defer ln.Close()

		var stdout, stderr bytes.Buffer
		code := run([]string{"-tcp", ln.Addr().String()}, &stdout, &stderr)
		if code != 0 && code != 1 {
			t.Errorf("unexpected exit code: %d, stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "TCP ("+ln.Addr().String()+")") {
			t.Errorf("stdout missing TCP check: %q", stdout.String())
		}
	})

	t.Run("executes with http check against test server", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer ts.Close()

		var stdout, stderr bytes.Buffer
		code := run([]string{"-http", ts.URL}, &stdout, &stderr)
		if code != 0 && code != 1 {
			t.Errorf("unexpected exit code: %d, stderr: %s", code, stderr.String())
		}
		if !strings.Contains(stdout.String(), "HTTP ("+ts.URL+")") {
			t.Errorf("stdout missing HTTP check: %q", stdout.String())
		}
	})

	t.Run("executes with tcp check reporting critical for closed port", func(t *testing.T) {
		// Use a local port that is unlikely to be listening
		var stdout, stderr bytes.Buffer
		code := run([]string{"-tcp", "127.0.0.1:59999"}, &stdout, &stderr)
		if code != 2 {
			t.Errorf("exit code = %d, want 2 (Critical) for closed port. Stdout: %s", code, stdout.String())
		}
		if !strings.Contains(stdout.String(), "CRITICAL") {
			t.Errorf("stdout missing CRITICAL status: %q", stdout.String())
		}
	})
}
