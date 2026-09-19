package main

import (
	"bytes"
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
}
