// Package load provides utilities to inspect Linux system load averages.
package load

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// DefaultLoadAvgPath is the standard Linux kernel procfs file for load averages.
const DefaultLoadAvgPath = "/proc/loadavg"

// LoadAvg represents the 1, 5, and 15-minute system load averages.
type LoadAvg struct {
	One     float64
	Five    float64
	Fifteen float64
}

// Read reads and parses the system load average from /proc/loadavg.
func Read() (LoadAvg, error) {
	return ReadFile(DefaultLoadAvgPath)
}

// ReadFile reads and parses the load average from a specified file path.
// This allows reading from alternate paths during testing or simulation.
func ReadFile(path string) (LoadAvg, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return LoadAvg{}, fmt.Errorf("reading loadavg file %q: %w", path, err)
	}

	return Parse(string(data))
}

// Parse parses the raw text content of a /proc/loadavg file.
//
// The standard /proc/loadavg format is:
//
//	<1m> <5m> <15m> <runnable/total_threads> <last_pid>
//
// Example: "0.15 0.20 0.18 1/824 12345"
func Parse(content string) (LoadAvg, error) {
	fields := strings.Fields(content)
	if len(fields) < 3 {
		return LoadAvg{}, fmt.Errorf("malformed loadavg content: expected at least 3 fields, got %d", len(fields))
	}

	one, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return LoadAvg{}, fmt.Errorf("parsing 1m load %q: %w", fields[0], err)
	}

	five, err := strconv.ParseFloat(fields[1], 64)
	if err != nil {
		return LoadAvg{}, fmt.Errorf("parsing 5m load %q: %w", fields[1], err)
	}

	fifteen, err := strconv.ParseFloat(fields[2], 64)
	if err != nil {
		return LoadAvg{}, fmt.Errorf("parsing 15m load %q: %w", fields[2], err)
	}

	return LoadAvg{
		One:     one,
		Five:    five,
		Fifteen: fifteen,
	}, nil
}
