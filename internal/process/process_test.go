package process_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/korabcenaj/syscheck/internal/process"
)

func TestReadProcesses(t *testing.T) {
	t.Run("scans valid procfs structure", func(t *testing.T) {
		tempProc := t.TempDir()

		// Create mock PID 101: systemd
		pid101 := filepath.Join(tempProc, "101")
		if err := os.Mkdir(pid101, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pid101, "comm"), []byte("systemd\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pid101, "cmdline"), []byte("/sbin/init\x00--system\x00"), 0644); err != nil {
			t.Fatal(err)
		}

		// Create mock PID 202: sshd
		pid202 := filepath.Join(tempProc, "202")
		if err := os.Mkdir(pid202, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pid202, "comm"), []byte("sshd\n"), 0644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(pid202, "cmdline"), []byte("/usr/sbin/sshd\x00-D\x00"), 0644); err != nil {
			t.Fatal(err)
		}

		// Create non-PID directories that should be skipped
		if err := os.Mkdir(filepath.Join(tempProc, "sys"), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(filepath.Join(tempProc, "net"), 0755); err != nil {
			t.Fatal(err)
		}

		// Create a file in proc root that should be skipped
		if err := os.WriteFile(filepath.Join(tempProc, "stat"), []byte("cpu 1 2 3"), 0644); err != nil {
			t.Fatal(err)
		}

		procs, err := process.ReadProcesses(tempProc)
		if err != nil {
			t.Fatalf("unexpected error reading processes: %v", err)
		}

		if len(procs) != 2 {
			t.Fatalf("expected 2 processes, got %d: %+v", len(procs), procs)
		}

		pMap := make(map[int]process.Process)
		for _, p := range procs {
			pMap[p.PID] = p
		}

		p101, ok := pMap[101]
		if !ok {
			t.Errorf("PID 101 missing from results")
		} else {
			if p101.Name != "systemd" {
				t.Errorf("PID 101 Name = %q, want systemd", p101.Name)
			}
			if p101.Cmdline != "/sbin/init --system" {
				t.Errorf("PID 101 Cmdline = %q, want '/sbin/init --system'", p101.Cmdline)
			}
		}

		p202, ok := pMap[202]
		if !ok {
			t.Errorf("PID 202 missing from results")
		} else {
			if p202.Name != "sshd" {
				t.Errorf("PID 202 Name = %q, want sshd", p202.Name)
			}
			if p202.Cmdline != "/usr/sbin/sshd -D" {
				t.Errorf("PID 202 Cmdline = %q, want '/usr/sbin/sshd -D'", p202.Cmdline)
			}
		}
	})

	t.Run("skips PID directory if comm is missing", func(t *testing.T) {
		tempProc := t.TempDir()
		pid999 := filepath.Join(tempProc, "999")
		if err := os.Mkdir(pid999, 0755); err != nil {
			t.Fatal(err)
		}

		procs, err := process.ReadProcesses(tempProc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(procs) != 0 {
			t.Errorf("expected 0 processes, got %d", len(procs))
		}
	})

	t.Run("returns error on non-existent directory", func(t *testing.T) {
		_, err := process.ReadProcesses("/non/existent/proc/path")
		if err == nil {
			t.Errorf("expected error for non-existent directory, got nil")
		}
	})
}

func TestFindByName(t *testing.T) {
	tempProc := t.TempDir()

	// PID 301: postgres
	pid301 := filepath.Join(tempProc, "301")
	_ = os.Mkdir(pid301, 0755)
	_ = os.WriteFile(filepath.Join(pid301, "comm"), []byte("postgres\n"), 0644)
	_ = os.WriteFile(filepath.Join(pid301, "cmdline"), []byte("/usr/bin/postgres\x00-D\x00/var/lib/data\x00"), 0644)

	// PID 302: another postgres worker
	pid302 := filepath.Join(tempProc, "302")
	_ = os.Mkdir(pid302, 0755)
	_ = os.WriteFile(filepath.Join(pid302, "comm"), []byte("postgres\n"), 0644)
	_ = os.WriteFile(filepath.Join(pid302, "cmdline"), []byte("postgres: checkpointer\x00"), 0644)

	// PID 401: worker binary found by cmdline basename
	pid401 := filepath.Join(tempProc, "401")
	_ = os.Mkdir(pid401, 0755)
	_ = os.WriteFile(filepath.Join(pid401, "comm"), []byte("app-runner\n"), 0644)
	_ = os.WriteFile(filepath.Join(pid401, "cmdline"), []byte("/opt/bin/worker\x00--port=8080\x00"), 0644)

	t.Run("finds matching processes by comm name", func(t *testing.T) {
		matched, err := process.FindByName("postgres", tempProc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(matched) != 2 {
			t.Errorf("expected 2 postgres processes, got %d", len(matched))
		}
	})

	t.Run("finds matching process case-insensitively", func(t *testing.T) {
		matched, err := process.FindByName("Postgres", tempProc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(matched) != 2 {
			t.Errorf("expected 2 postgres processes, got %d", len(matched))
		}
	})

	t.Run("finds matching process by cmdline basename", func(t *testing.T) {
		matched, err := process.FindByName("worker", tempProc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(matched) != 1 {
			t.Fatalf("expected 1 worker process, got %d", len(matched))
		}
		if matched[0].PID != 401 {
			t.Errorf("PID = %d, want 401", matched[0].PID)
		}
	})

	t.Run("returns empty slice when process not found", func(t *testing.T) {
		matched, err := process.FindByName("nginx", tempProc)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(matched) != 0 {
			t.Errorf("expected 0 matches, got %d", len(matched))
		}
	})

	t.Run("returns error when name is empty", func(t *testing.T) {
		_, err := process.FindByName("", tempProc)
		if err == nil {
			t.Errorf("expected error on empty process name, got nil")
		}
	})
}

func TestQueryServiceWithRunner(t *testing.T) {
	t.Run("returns active for systemctl active output", func(t *testing.T) {
		runner := func(name string, args ...string) ([]byte, error) {
			return []byte("active\n"), nil
		}

		state, err := process.QueryServiceWithRunner("sshd", runner)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !state.Active {
			t.Errorf("Active = false, want true")
		}
		if state.State != "active" {
			t.Errorf("State = %q, want 'active'", state.State)
		}
		if state.Name != "sshd" {
			t.Errorf("Name = %q, want 'sshd'", state.Name)
		}
	})

	t.Run("returns inactive for systemctl inactive output with non-zero exit", func(t *testing.T) {
		runner := func(name string, args ...string) ([]byte, error) {
			return []byte("inactive\n"), errors.New("exit status 3")
		}

		state, err := process.QueryServiceWithRunner("sshd", runner)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state.Active {
			t.Errorf("Active = true, want false")
		}
		if state.State != "inactive" {
			t.Errorf("State = %q, want 'inactive'", state.State)
		}
	})

	t.Run("returns failed for systemctl failed output", func(t *testing.T) {
		runner := func(name string, args ...string) ([]byte, error) {
			return []byte("failed\n"), errors.New("exit status 3")
		}

		state, err := process.QueryServiceWithRunner("nginx", runner)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if state.Active {
			t.Errorf("Active = true, want false")
		}
		if state.State != "failed" {
			t.Errorf("State = %q, want 'failed'", state.State)
		}
	})

	t.Run("returns error when command fails without recognized state output", func(t *testing.T) {
		runner := func(name string, args ...string) ([]byte, error) {
			return nil, errors.New("executable file not found in $PATH")
		}

		state, err := process.QueryServiceWithRunner("sshd", runner)
		if err == nil {
			t.Errorf("expected error when runner fails completely, got nil")
		}
		if state.Active {
			t.Errorf("Active = true, want false")
		}
	})

	t.Run("returns error when service name is empty", func(t *testing.T) {
		runner := func(name string, args ...string) ([]byte, error) {
			return []byte("active\n"), nil
		}

		_, err := process.QueryServiceWithRunner("  ", runner)
		if err == nil {
			t.Errorf("expected error on empty service name, got nil")
		}
	})
}
