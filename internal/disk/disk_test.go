package disk_test

import (
	"errors"
	"syscall"
	"testing"

	"github.com/korabcenaj/syscheck/internal/disk"
)

func TestFromStatfs(t *testing.T) {
	t.Run("calculates metrics for typical ext4/xfs filesystem", func(t *testing.T) {
		stat := syscall.Statfs_t{
			Bsize:  4096,
			Blocks: 26214400, // 100 GiB (26214400 * 4096)
			Bfree:  13107200, // 50 GiB free
			Bavail: 11796480, // ~45 GiB available to non-root (5% reserved)
			Files:  6553600,  // ~6.5M total inodes
			Ffree:  5898240,  // ~5.8M free inodes (~10% used)
		}

		usage := disk.FromStatfs("/mnt/data", stat)

		expectedTotalBytes := uint64(26214400) * 4096
		expectedFreeBytes := uint64(13107200) * 4096
		expectedAvailBytes := uint64(11796480) * 4096
		expectedUsedBytes := expectedTotalBytes - expectedFreeBytes

		if usage.Path != "/mnt/data" {
			t.Errorf("Path = %q, want %q", usage.Path, "/mnt/data")
		}
		if usage.TotalBytes != expectedTotalBytes {
			t.Errorf("TotalBytes = %d, want %d", usage.TotalBytes, expectedTotalBytes)
		}
		if usage.FreeBytes != expectedFreeBytes {
			t.Errorf("FreeBytes = %d, want %d", usage.FreeBytes, expectedFreeBytes)
		}
		if usage.AvailableBytes != expectedAvailBytes {
			t.Errorf("AvailableBytes = %d, want %d", usage.AvailableBytes, expectedAvailBytes)
		}
		if usage.UsedBytes != expectedUsedBytes {
			t.Errorf("UsedBytes = %d, want %d", usage.UsedBytes, expectedUsedBytes)
		}

		// UsedPercent should be 50.0%
		if pct := usage.UsedPercent(); pct != 50.0 {
			t.Errorf("UsedPercent() = %f, want 50.0", pct)
		}

		// Inodes: 655360 used out of 6553600 = 10.0%
		if pct := usage.InodesUsedPercent(); pct != 10.0 {
			t.Errorf("InodesUsedPercent() = %f, want 10.0", pct)
		}
	})

	t.Run("handles 100 percent full disk without overflow", func(t *testing.T) {
		stat := syscall.Statfs_t{
			Bsize:  4096,
			Blocks: 1000,
			Bfree:  0,
			Bavail: 0,
			Files:  500,
			Ffree:  0,
		}

		usage := disk.FromStatfs("/full", stat)
		if pct := usage.UsedPercent(); pct != 100.0 {
			t.Errorf("UsedPercent() = %f, want 100.0", pct)
		}
		if pct := usage.InodesUsedPercent(); pct != 100.0 {
			t.Errorf("InodesUsedPercent() = %f, want 100.0", pct)
		}
	})

	t.Run("handles zero blocks and zero inodes safely", func(t *testing.T) {
		stat := syscall.Statfs_t{
			Bsize:  4096,
			Blocks: 0,
			Bfree:  0,
			Bavail: 0,
			Files:  0,
			Ffree:  0,
		}

		usage := disk.FromStatfs("/proc", stat)
		if pct := usage.UsedPercent(); pct != 0.0 {
			t.Errorf("UsedPercent() = %f, want 0.0", pct)
		}
		if pct := usage.InodesUsedPercent(); pct != 0.0 {
			t.Errorf("InodesUsedPercent() = %f, want 0.0", pct)
		}
	})
}

func TestCheck(t *testing.T) {
	t.Run("succeeds on host root directory", func(t *testing.T) {
		usage, err := disk.Check("/")
		if err != nil {
			t.Fatalf("Check(\"/\") failed: %v", err)
		}

		if usage.TotalBytes == 0 {
			t.Error("expected non-zero TotalBytes for /")
		}
		if usage.Path != "/" {
			t.Errorf("Path = %q, want \"/\"", usage.Path)
		}
	})

	t.Run("fails cleanly for non-existent path", func(t *testing.T) {
		_, err := disk.Check("/nonexistent_directory_for_syscheck_test")
		if err == nil {
			t.Fatal("expected error for non-existent path, got nil")
		}
		if !errors.Is(err, syscall.ENOENT) {
			t.Errorf("expected error wrapping syscall.ENOENT, got: %v", err)
		}
	})
}
