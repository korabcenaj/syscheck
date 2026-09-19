package memory_test

import (
	"os"
	"strings"
	"testing"

	"syscheck/internal/memory"
)

const sampleMeminfoModern = `
MemTotal:       16384000 kB
MemFree:         2048000 kB
MemAvailable:    8192000 kB
Buffers:          512000 kB
Cached:          4096000 kB
SwapTotal:       4096000 kB
SwapFree:        3072000 kB
`

const sampleMeminfoLegacy = `
MemTotal:        8192000 kB
MemFree:         1024000 kB
Buffers:          512000 kB
Cached:          2048000 kB
SwapTotal:       2048000 kB
SwapFree:        1024000 kB
`

const sampleMeminfoNoSwap = `
MemTotal:       32768000 kB
MemFree:         4096000 kB
MemAvailable:   24576000 kB
SwapTotal:             0 kB
SwapFree:              0 kB
`

func TestParse(t *testing.T) {
	t.Run("parses modern kernel meminfo correctly", func(t *testing.T) {
		r := strings.NewReader(sampleMeminfoModern)
		stats, err := memory.Parse(r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedTotal := uint64(16384000) * 1024
		expectedAvailable := uint64(8192000) * 1024
		expectedFree := uint64(2048000) * 1024
		expectedSwapTotal := uint64(4096000) * 1024
		expectedSwapFree := uint64(3072000) * 1024

		if stats.TotalBytes != expectedTotal {
			t.Errorf("TotalBytes = %d, want %d", stats.TotalBytes, expectedTotal)
		}
		if stats.AvailableBytes != expectedAvailable {
			t.Errorf("AvailableBytes = %d, want %d", stats.AvailableBytes, expectedAvailable)
		}
		if stats.FreeBytes != expectedFree {
			t.Errorf("FreeBytes = %d, want %d", stats.FreeBytes, expectedFree)
		}
		if stats.SwapTotalBytes != expectedSwapTotal {
			t.Errorf("SwapTotalBytes = %d, want %d", stats.SwapTotalBytes, expectedSwapTotal)
		}
		if stats.SwapFreeBytes != expectedSwapFree {
			t.Errorf("SwapFreeBytes = %d, want %d", stats.SwapFreeBytes, expectedSwapFree)
		}

		// Used = Total - Available = 16384000 - 8192000 = 8192000 kB
		expectedUsed := expectedTotal - expectedAvailable
		if stats.UsedBytes() != expectedUsed {
			t.Errorf("UsedBytes() = %d, want %d", stats.UsedBytes(), expectedUsed)
		}

		// UsedPercent = 50.0%
		if pct := stats.UsedPercent(); pct != 50.0 {
			t.Errorf("UsedPercent() = %f, want 50.0", pct)
		}

		// SwapUsed = 4096000 - 3072000 = 1024000 kB (25%)
		expectedSwapUsed := uint64(1024000) * 1024
		if stats.SwapUsedBytes() != expectedSwapUsed {
			t.Errorf("SwapUsedBytes() = %d, want %d", stats.SwapUsedBytes(), expectedSwapUsed)
		}
		if pct := stats.SwapUsedPercent(); pct != 25.0 {
			t.Errorf("SwapUsedPercent() = %f, want 25.0", pct)
		}
	})

	t.Run("calculates fallback available memory for legacy kernels", func(t *testing.T) {
		r := strings.NewReader(sampleMeminfoLegacy)
		stats, err := memory.Parse(r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Available fallback = Free (1024000) + Buffers (512000) + Cached (2048000) = 3584000 kB
		expectedAvailable := uint64(3584000) * 1024
		if stats.AvailableBytes != expectedAvailable {
			t.Errorf("fallback AvailableBytes = %d, want %d", stats.AvailableBytes, expectedAvailable)
		}
	})

	t.Run("handles system with zero swap gracefully", func(t *testing.T) {
		r := strings.NewReader(sampleMeminfoNoSwap)
		stats, err := memory.Parse(r)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if stats.SwapTotalBytes != 0 {
			t.Errorf("SwapTotalBytes = %d, want 0", stats.SwapTotalBytes)
		}
		if stats.SwapUsedBytes() != 0 {
			t.Errorf("SwapUsedBytes() = %d, want 0", stats.SwapUsedBytes())
		}
		if stats.SwapUsedPercent() != 0.0 {
			t.Errorf("SwapUsedPercent() = %f, want 0.0", stats.SwapUsedPercent())
		}
	})

	t.Run("errors on missing MemTotal", func(t *testing.T) {
		input := "SwapTotal: 1024 kB\n"
		_, err := memory.Parse(strings.NewReader(input))
		if err == nil {
			t.Fatal("expected error for missing MemTotal, got nil")
		}
	})

	t.Run("errors on non-numeric value", func(t *testing.T) {
		input := "MemTotal: notanumber kB\n"
		_, err := memory.Parse(strings.NewReader(input))
		if err == nil {
			t.Fatal("expected error for non-numeric field, got nil")
		}
	})
}

func TestRead(t *testing.T) {
	if _, err := os.Stat(memory.DefaultMeminfoPath); os.IsNotExist(err) {
		t.Skipf("%s does not exist on this host, skipping live test", memory.DefaultMeminfoPath)
	}

	stats, err := memory.Read()
	if err != nil {
		t.Fatalf("memory.Read() failed: %v", err)
	}

	if stats.TotalBytes == 0 {
		t.Error("expected non-zero TotalBytes from live /proc/meminfo")
	}
	if stats.AvailableBytes == 0 {
		t.Error("expected non-zero AvailableBytes from live /proc/meminfo")
	}
}
