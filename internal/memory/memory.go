// Package memory provides functionality to inspect Linux memory and swap utilization.
package memory

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// DefaultMeminfoPath is the standard Linux procfs file for memory statistics.
const DefaultMeminfoPath = "/proc/meminfo"

// Stats holds physical memory and swap statistics in bytes.
type Stats struct {
	TotalBytes     uint64
	AvailableBytes uint64
	FreeBytes      uint64
	SwapTotalBytes uint64
	SwapFreeBytes  uint64
}

// UsedBytes returns the amount of physical memory actively in use.
func (s Stats) UsedBytes() uint64 {
	if s.TotalBytes < s.AvailableBytes {
		return 0
	}
	return s.TotalBytes - s.AvailableBytes
}

// UsedPercent returns the percentage of physical memory in use (0.0 to 100.0).
func (s Stats) UsedPercent() float64 {
	if s.TotalBytes == 0 {
		return 0.0
	}
	return (float64(s.UsedBytes()) / float64(s.TotalBytes)) * 100.0
}

// SwapUsedBytes returns the amount of swap space in use.
func (s Stats) SwapUsedBytes() uint64 {
	if s.SwapTotalBytes < s.SwapFreeBytes {
		return 0
	}
	return s.SwapTotalBytes - s.SwapFreeBytes
}

// SwapUsedPercent returns the percentage of swap space in use (0.0 to 100.0).
// If the system has no swap configured (SwapTotal is 0), it returns 0.0.
func (s Stats) SwapUsedPercent() float64 {
	if s.SwapTotalBytes == 0 {
		return 0.0
	}
	return (float64(s.SwapUsedBytes()) / float64(s.SwapTotalBytes)) * 100.0
}

// Read reads and parses memory statistics from the host system's /proc/meminfo.
func Read() (Stats, error) {
	return ReadFile(DefaultMeminfoPath)
}

// ReadFile opens and parses the specified meminfo file path.
func ReadFile(path string) (Stats, error) {
	file, err := os.Open(path)
	if err != nil {
		return Stats{}, fmt.Errorf("opening meminfo file %q: %w", path, err)
	}
	defer file.Close()

	return Parse(file)
}

// Parse reads lines from an io.Reader and extracts memory and swap values.
// Linux /proc/meminfo reports values in "kB" (1024 bytes per unit).
func Parse(r io.Reader) (Stats, error) {
	scanner := bufio.NewScanner(r)

	var (
		memTotal     uint64
		memFree      uint64
		memAvailable uint64
		buffers      uint64
		cached       uint64
		swapTotal    uint64
		swapFree     uint64

		hasMemTotal     bool
		hasMemAvailable bool
	)

	for scanner.Scan() {
		line := scanner.Text()
		key, valStr, found := strings.Cut(line, ":")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		fields := strings.Fields(valStr)
		if len(fields) == 0 {
			continue
		}

		// Values in /proc/meminfo are typically suffixed with "kB"
		valKb, err := strconv.ParseUint(fields[0], 10, 64)
		if err != nil {
			return Stats{}, fmt.Errorf("parsing field %q value %q: %w", key, fields[0], err)
		}

		// Convert kibibytes (kB in Linux procfs) to bytes
		valBytes := valKb * 1024

		switch key {
		case "MemTotal":
			memTotal = valBytes
			hasMemTotal = true
		case "MemFree":
			memFree = valBytes
		case "MemAvailable":
			memAvailable = valBytes
			hasMemAvailable = true
		case "Buffers":
			buffers = valBytes
		case "Cached":
			cached = valBytes
		case "SwapTotal":
			swapTotal = valBytes
		case "SwapFree":
			swapFree = valBytes
		}
	}

	if err := scanner.Err(); err != nil {
		return Stats{}, fmt.Errorf("scanning meminfo: %w", err)
	}

	if !hasMemTotal || memTotal == 0 {
		return Stats{}, fmt.Errorf("invalid meminfo: missing or zero MemTotal")
	}

	// Fallback for older Linux kernels (<3.14) where MemAvailable was not yet introduced
	if !hasMemAvailable {
		memAvailable = memFree + buffers + cached
		if memAvailable > memTotal {
			memAvailable = memTotal
		}
	}

	return Stats{
		TotalBytes:     memTotal,
		AvailableBytes: memAvailable,
		FreeBytes:      memFree,
		SwapTotalBytes: swapTotal,
		SwapFreeBytes:  swapFree,
	}, nil
}
