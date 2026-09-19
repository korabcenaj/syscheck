package check_test

import (
	"errors"
	"testing"

	"github.com/korabcenaj/syscheck/internal/check"
	"github.com/korabcenaj/syscheck/internal/disk"
	"github.com/korabcenaj/syscheck/internal/load"
	"github.com/korabcenaj/syscheck/internal/memory"
)

func TestStatusString(t *testing.T) {
	tests := []struct {
		status check.Status
		want   string
	}{
		{check.StatusOK, "OK"},
		{check.StatusWarning, "WARNING"},
		{check.StatusCritical, "CRITICAL"},
		{check.StatusUnknown, "UNKNOWN"},
		{check.Status(999), "UNKNOWN"},
	}

	for _, tt := range tests {
		if got := tt.status.String(); got != tt.want {
			t.Errorf("Status(%d).String() = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func TestStatusExitCode(t *testing.T) {
	tests := []struct {
		status check.Status
		want   int
	}{
		{check.StatusOK, 0},
		{check.StatusWarning, 1},
		{check.StatusCritical, 2},
		{check.StatusUnknown, 3},
		{check.Status(999), 3},
	}

	for _, tt := range tests {
		if got := tt.status.ExitCode(); got != tt.want {
			t.Errorf("Status(%d).ExitCode() = %d, want %d", tt.status, got, tt.want)
		}
	}
}

func TestStatusJSON(t *testing.T) {
	t.Run("marshals correctly", func(t *testing.T) {
		b, err := check.StatusOK.MarshalJSON()
		if err != nil {
			t.Fatalf("MarshalJSON() err = %v", err)
		}
		if string(b) != `"OK"` {
			t.Errorf("MarshalJSON() = %s, want %s", string(b), `"OK"`)
		}
	})

	t.Run("unmarshals correctly", func(t *testing.T) {
		var s check.Status
		if err := s.UnmarshalJSON([]byte(`"CRITICAL"`)); err != nil {
			t.Fatalf("UnmarshalJSON() err = %v", err)
		}
		if s != check.StatusCritical {
			t.Errorf("UnmarshalJSON() = %v, want CRITICAL", s)
		}
	})
}

func TestOverallStatus(t *testing.T) {
	tests := []struct {
		name    string
		results []check.Result
		want    check.Status
	}{
		{
			name:    "empty slice defaults to OK",
			results: []check.Result{},
			want:    check.StatusOK,
		},
		{
			name: "all OK returns OK",
			results: []check.Result{
				{Status: check.StatusOK},
				{Status: check.StatusOK},
			},
			want: check.StatusOK,
		},
		{
			name: "warning and OK returns Warning",
			results: []check.Result{
				{Status: check.StatusOK},
				{Status: check.StatusWarning},
			},
			want: check.StatusWarning,
		},
		{
			name: "unknown outranks warning",
			results: []check.Result{
				{Status: check.StatusWarning},
				{Status: check.StatusUnknown},
			},
			want: check.StatusUnknown,
		},
		{
			name: "critical outranks all",
			results: []check.Result{
				{Status: check.StatusOK},
				{Status: check.StatusWarning},
				{Status: check.StatusUnknown},
				{Status: check.StatusCritical},
			},
			want: check.StatusCritical,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := check.OverallStatus(tt.results); got != tt.want {
				t.Errorf("OverallStatus() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestLoadChecker(t *testing.T) {
	t.Run("evaluates OK status", func(t *testing.T) {
		c := &check.LoadChecker{
			WarnThreshold: 2.0,
			CritThreshold: 5.0,
			ReadFunc: func() (load.LoadAvg, error) {
				return load.LoadAvg{One: 0.8, Five: 0.9, Fifteen: 0.7}, nil
			},
		}

		res := c.Check()
		if res.Status != check.StatusOK {
			t.Errorf("Status = %v, want OK", res.Status)
		}
	})

	t.Run("evaluates Warning status", func(t *testing.T) {
		c := &check.LoadChecker{
			WarnThreshold: 2.0,
			CritThreshold: 5.0,
			ReadFunc: func() (load.LoadAvg, error) {
				return load.LoadAvg{One: 2.5, Five: 1.5, Fifteen: 1.0}, nil
			},
		}

		res := c.Check()
		if res.Status != check.StatusWarning {
			t.Errorf("Status = %v, want Warning", res.Status)
		}
	})

	t.Run("evaluates Critical status", func(t *testing.T) {
		c := &check.LoadChecker{
			WarnThreshold: 2.0,
			CritThreshold: 5.0,
			ReadFunc: func() (load.LoadAvg, error) {
				return load.LoadAvg{One: 6.5, Five: 5.2, Fifteen: 3.8}, nil
			},
		}

		res := c.Check()
		if res.Status != check.StatusCritical {
			t.Errorf("Status = %v, want Critical", res.Status)
		}
	})

	t.Run("evaluates error as Unknown status", func(t *testing.T) {
		c := &check.LoadChecker{
			ReadFunc: func() (load.LoadAvg, error) {
				return load.LoadAvg{}, errors.New("procfs unavailable")
			},
		}

		res := c.Check()
		if res.Status != check.StatusUnknown {
			t.Errorf("Status = %v, want Unknown", res.Status)
		}
		if res.Err == nil {
			t.Error("expected non-nil Err in result")
		}
	})
}

func TestMemoryChecker(t *testing.T) {
	t.Run("evaluates OK status", func(t *testing.T) {
		c := &check.MemoryChecker{
			WarnPercent: 80.0,
			CritPercent: 90.0,
			ReadFunc: func() (memory.Stats, error) {
				// 50% used
				return memory.Stats{
					TotalBytes:     1000,
					AvailableBytes: 500,
				}, nil
			},
		}

		res := c.Check()
		if res.Status != check.StatusOK {
			t.Errorf("Status = %v, want OK", res.Status)
		}
	})

	t.Run("evaluates Warning status", func(t *testing.T) {
		c := &check.MemoryChecker{
			WarnPercent: 80.0,
			CritPercent: 90.0,
			ReadFunc: func() (memory.Stats, error) {
				// 85% used (150 available out of 1000)
				return memory.Stats{
					TotalBytes:     1000,
					AvailableBytes: 150,
				}, nil
			},
		}

		res := c.Check()
		if res.Status != check.StatusWarning {
			t.Errorf("Status = %v, want Warning", res.Status)
		}
	})

	t.Run("evaluates Critical status", func(t *testing.T) {
		c := &check.MemoryChecker{
			WarnPercent: 80.0,
			CritPercent: 90.0,
			ReadFunc: func() (memory.Stats, error) {
				// 95% used (50 available out of 1000)
				return memory.Stats{
					TotalBytes:     1000,
					AvailableBytes: 50,
				}, nil
			},
		}

		res := c.Check()
		if res.Status != check.StatusCritical {
			t.Errorf("Status = %v, want Critical", res.Status)
		}
	})
}

func TestDiskChecker(t *testing.T) {
	t.Run("evaluates OK on healthy disk", func(t *testing.T) {
		c := &check.DiskChecker{
			Path:        "/",
			WarnPercent: 80.0,
			CritPercent: 90.0,
			CheckFunc: func(path string) (disk.Usage, error) {
				return disk.Usage{
					Path:        path,
					TotalBytes:  1000,
					UsedBytes:   400,
					TotalInodes: 500,
					UsedInodes:  100,
				}, nil
			},
		}

		res := c.Check()
		if res.Status != check.StatusOK {
			t.Errorf("Status = %v, want OK", res.Status)
		}
	})

	t.Run("triggers Warning when space threshold exceeded", func(t *testing.T) {
		c := &check.DiskChecker{
			Path:        "/var",
			WarnPercent: 80.0,
			CritPercent: 90.0,
			CheckFunc: func(path string) (disk.Usage, error) {
				return disk.Usage{
					Path:        path,
					TotalBytes:  1000,
					UsedBytes:   850, // 85% space used
					TotalInodes: 1000,
					UsedInodes:  100,
				}, nil
			},
		}

		res := c.Check()
		if res.Status != check.StatusWarning {
			t.Errorf("Status = %v, want Warning", res.Status)
		}
	})

	t.Run("triggers Critical when inode threshold exceeded", func(t *testing.T) {
		c := &check.DiskChecker{
			Path:        "/data",
			WarnPercent: 80.0,
			CritPercent: 90.0,
			CheckFunc: func(path string) (disk.Usage, error) {
				return disk.Usage{
					Path:        path,
					TotalBytes:  1000,
					UsedBytes:   200, // 20% space used
					TotalInodes: 1000,
					UsedInodes:  950, // 95% inodes used!
				}, nil
			},
		}

		res := c.Check()
		if res.Status != check.StatusCritical {
			t.Errorf("Status = %v, want Critical", res.Status)
		}
	})
}

func TestRunner(t *testing.T) {
	c1 := &check.LoadChecker{
		ReadFunc: func() (load.LoadAvg, error) {
			return load.LoadAvg{One: 0.5}, nil
		},
	}
	c2 := &check.MemoryChecker{
		ReadFunc: func() (memory.Stats, error) {
			return memory.Stats{TotalBytes: 1000, AvailableBytes: 800}, nil
		},
	}

	runner := check.NewRunner(c1)
	runner.Add(c2)

	results := runner.RunAll()
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	if results[0].Name != "Load Average" || results[0].Status != check.StatusOK {
		t.Errorf("results[0] = %+v", results[0])
	}
	if results[1].Name != "Memory" || results[1].Status != check.StatusOK {
		t.Errorf("results[1] = %+v", results[1])
	}
}
