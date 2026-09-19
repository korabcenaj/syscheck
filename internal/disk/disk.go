// Package disk provides filesystem disk space and inode monitoring using Linux system calls.
package disk

import (
	"fmt"
	"syscall"
)

// Usage contains disk space and inode usage statistics for a filesystem mount path.
type Usage struct {
	Path           string
	TotalBytes     uint64
	FreeBytes      uint64 // Free blocks available to privileged processes (root)
	AvailableBytes uint64 // Free blocks available to unprivileged processes
	UsedBytes      uint64
	TotalInodes    uint64
	FreeInodes     uint64
	UsedInodes     uint64
}

// UsedPercent calculates the percentage of disk space used (0.0 to 100.0).
func (u Usage) UsedPercent() float64 {
	if u.TotalBytes == 0 {
		return 0.0
	}
	return (float64(u.UsedBytes) / float64(u.TotalBytes)) * 100.0
}

// InodesUsedPercent calculates the percentage of inodes used (0.0 to 100.0).
func (u Usage) InodesUsedPercent() float64 {
	if u.TotalInodes == 0 {
		return 0.0
	}
	return (float64(u.UsedInodes) / float64(u.TotalInodes)) * 100.0
}

// HasInodes returns true if the underlying filesystem reports fixed inode counts.
// Dynamic filesystems (such as Btrfs or ZFS) report 0 total inodes.
func (u Usage) HasInodes() bool {
	return u.TotalInodes > 0
}

// Check queries filesystem statistics for the given mount point or directory path.
func Check(path string) (Usage, error) {
	return checkWith(path, syscall.Statfs)
}

// statfsFunc defines the signature of a function matching syscall.Statfs.
// This allows dependency injection in unit tests without touching the host filesystem.
type statfsFunc func(path string, buf *syscall.Statfs_t) error

// checkWith executes the given statfsFunc against path, returning a Usage struct.
func checkWith(path string, fn statfsFunc) (Usage, error) {
	var stat syscall.Statfs_t
	if err := fn(path, &stat); err != nil {
		return Usage{}, fmt.Errorf("statfs %q: %w", path, err)
	}

	return FromStatfs(path, stat), nil
}

// FromStatfs converts a raw Linux syscall.Statfs_t structure into a strongly-typed Usage struct.
func FromStatfs(path string, stat syscall.Statfs_t) Usage {
	var bsize uint64
	if stat.Bsize > 0 {
		bsize = uint64(stat.Bsize)
	}

	totalBytes := stat.Blocks * bsize
	freeBytes := stat.Bfree * bsize
	availBytes := stat.Bavail * bsize

	var usedBytes uint64
	if totalBytes >= freeBytes {
		usedBytes = totalBytes - freeBytes
	}

	totalInodes := stat.Files
	freeInodes := stat.Ffree

	var usedInodes uint64
	if totalInodes >= freeInodes {
		usedInodes = totalInodes - freeInodes
	}

	return Usage{
		Path:           path,
		TotalBytes:     totalBytes,
		FreeBytes:      freeBytes,
		AvailableBytes: availBytes,
		UsedBytes:      usedBytes,
		TotalInodes:    totalInodes,
		FreeInodes:     freeInodes,
		UsedInodes:     usedInodes,
	}
}
