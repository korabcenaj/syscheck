// Package host inspects operating system metadata, uptime, and host identity.
package host

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

// Default paths for Linux host metadata.
const (
	DefaultOSReleasePath = "/etc/os-release"
	DefaultUptimePath    = "/proc/uptime"
)

// Info holds host identity, operating system, and uptime statistics.
type Info struct {
	Hostname   string        `json:"hostname"`
	OSName     string        `json:"os_name"`
	OSVersion  string        `json:"os_version"`
	PrettyName string        `json:"pretty_name"`
	Uptime     time.Duration `json:"uptime_seconds"`
}

// FormattedUptime returns a human-readable duration string (e.g. "3d 14h 22m").
func (i Info) FormattedUptime() string {
	totalSeconds := int64(i.Uptime.Seconds())
	if totalSeconds <= 0 {
		return "0s"
	}

	days := totalSeconds / 86400
	hours := (totalSeconds % 86400) / 3600
	minutes := (totalSeconds % 3600) / 60
	seconds := totalSeconds % 60

	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm", days, hours, minutes)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	}
	return fmt.Sprintf("%dm %ds", minutes, seconds)
}

// Read gathers host information from the local Linux system.
func Read() (Info, error) {
	hostname, err := os.Hostname()
	if err != nil {
		hostname = "unknown"
	}

	prettyName, osName, osVersion, err := ReadOSRelease(DefaultOSReleasePath)
	if err != nil {
		prettyName = "Linux (unknown distribution)"
	}

	uptime, err := ReadUptime(DefaultUptimePath)
	if err != nil {
		uptime = 0
	}

	return Info{
		Hostname:   hostname,
		OSName:     osName,
		OSVersion:  osVersion,
		PrettyName: prettyName,
		Uptime:     uptime,
	}, nil
}

// ReadOSRelease reads and parses an os-release file at path.
func ReadOSRelease(path string) (prettyName, name, version string, err error) {
	f, err := os.Open(path)
	if err != nil {
		return "", "", "", fmt.Errorf("opening os-release %q: %w", path, err)
	}
	defer f.Close()

	return ParseOSRelease(f)
}

// ParseOSRelease parses key-value pairs from an os-release stream.
func ParseOSRelease(r io.Reader) (prettyName, name, version string, err error) {
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		key, val, found := strings.Cut(line, "=")
		if !found {
			continue
		}

		key = strings.TrimSpace(key)
		val = strings.Trim(strings.TrimSpace(val), `"'`)

		switch key {
		case "PRETTY_NAME":
			prettyName = val
		case "NAME":
			name = val
		case "VERSION_ID":
			version = val
		}
	}

	if err := scanner.Err(); err != nil {
		return "", "", "", fmt.Errorf("scanning os-release: %w", err)
	}

	if prettyName == "" && name != "" {
		if version != "" {
			prettyName = fmt.Sprintf("%s %s", name, version)
		} else {
			prettyName = name
		}
	}

	return prettyName, name, version, nil
}

// ReadUptime reads and parses system uptime from /proc/uptime at path.
func ReadUptime(path string) (time.Duration, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, fmt.Errorf("reading uptime %q: %w", path, err)
	}

	return ParseUptime(string(data))
}

// ParseUptime parses raw /proc/uptime text into a time.Duration.
// Standard /proc/uptime contains two float numbers: <uptime_seconds> <idle_seconds>.
func ParseUptime(content string) (time.Duration, error) {
	fields := strings.Fields(content)
	if len(fields) == 0 {
		return 0, fmt.Errorf("empty uptime content")
	}

	secs, err := strconv.ParseFloat(fields[0], 64)
	if err != nil {
		return 0, fmt.Errorf("parsing uptime seconds %q: %w", fields[0], err)
	}

	if secs < 0 {
		return 0, fmt.Errorf("uptime cannot be negative: %f", secs)
	}

	return time.Duration(secs * float64(time.Second)), nil
}
