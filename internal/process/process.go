// Package process provides utilities to discover running processes and inspect system service states.
package process

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// DefaultProcPath is the standard Linux procfs mount point.
const DefaultProcPath = "/proc"

// Process represents a discovered operating system process.
type Process struct {
	PID     int    `json:"pid"`
	Name    string `json:"name"`
	Cmdline string `json:"cmdline"`
}

// ReadProcesses scans the provided procfs directory path and extracts running processes.
// If procPath is empty, DefaultProcPath ("/proc") is used.
func ReadProcesses(procPath string) ([]Process, error) {
	if procPath == "" {
		procPath = DefaultProcPath
	}

	entries, err := os.ReadDir(procPath)
	if err != nil {
		return nil, fmt.Errorf("reading procfs directory %q: %w", procPath, err)
	}

	var procs []Process
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		pid, err := strconv.Atoi(entry.Name())
		if err != nil || pid <= 0 {
			continue
		}

		pidDir := filepath.Join(procPath, entry.Name())

		// Read /proc/<pid>/comm for the short process name.
		commPath := filepath.Join(pidDir, "comm")
		commBytes, err := os.ReadFile(commPath)
		if err != nil {
			// Process may have exited between ReadDir and ReadFile.
			continue
		}
		name := strings.TrimSpace(string(commBytes))

		// Read /proc/<pid>/cmdline for full invocation arguments.
		cmdlinePath := filepath.Join(pidDir, "cmdline")
		cmdBytes, _ := os.ReadFile(cmdlinePath)
		cmdline := strings.ReplaceAll(string(cmdBytes), "\x00", " ")
		cmdline = strings.TrimSpace(cmdline)

		procs = append(procs, Process{
			PID:     pid,
			Name:    name,
			Cmdline: cmdline,
		})
	}

	return procs, nil
}

// FindByName searches running processes in procPath for any process matching the given name.
// A process matches if its short name (comm) matches or if the executable basename in cmdline matches.
func FindByName(name string, procPath string) ([]Process, error) {
	if name == "" {
		return nil, errors.New("process name cannot be empty")
	}

	all, err := ReadProcesses(procPath)
	if err != nil {
		return nil, err
	}

	target := strings.ToLower(strings.TrimSpace(name))
	var matched []Process

	for _, p := range all {
		if strings.ToLower(p.Name) == target {
			matched = append(matched, p)
			continue
		}

		if p.Cmdline != "" {
			fields := strings.Fields(p.Cmdline)
			if len(fields) > 0 {
				base := strings.ToLower(filepath.Base(fields[0]))
				if base == target {
					matched = append(matched, p)
				}
			}
		}
	}

	return matched, nil
}

// ServiceState represents the status of an init/systemd service.
type ServiceState struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
	State  string `json:"state"` // e.g. "active", "inactive", "failed", "unknown"
}

// ServiceQueryFunc is a function signature for querying a service state.
type ServiceQueryFunc func(serviceName string) (ServiceState, error)

// CommandRunner defines a function signature for running external system commands.
type CommandRunner func(name string, args ...string) ([]byte, error)

func defaultCommandRunner(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	return cmd.CombinedOutput()
}

// QueryService checks the state of a system service using systemctl.
func QueryService(serviceName string) (ServiceState, error) {
	return QueryServiceWithRunner(serviceName, defaultCommandRunner)
}

// QueryServiceWithRunner checks the state of a service using the provided command runner.
// This allows deterministic unit testing without executing real system binaries.
func QueryServiceWithRunner(serviceName string, runner CommandRunner) (ServiceState, error) {
	if strings.TrimSpace(serviceName) == "" {
		return ServiceState{}, errors.New("service name cannot be empty")
	}

	// Execute: systemctl is-active <serviceName>
	out, err := runner("systemctl", "is-active", serviceName)
	rawState := strings.ToLower(strings.TrimSpace(string(out)))

	// systemctl is-active returns:
	// "active" (exit 0) -> Active: true
	// "inactive", "failed", "activating", "deactivating" (exit non-zero) -> Active: false
	if rawState == "active" {
		return ServiceState{
			Name:   serviceName,
			Active: true,
			State:  rawState,
		}, nil
	}

	if rawState != "" {
		return ServiceState{
			Name:   serviceName,
			Active: false,
			State:  rawState,
		}, nil
	}

	if err != nil {
		return ServiceState{
			Name:   serviceName,
			Active: false,
			State:  "unknown",
		}, fmt.Errorf("executing systemctl for service %q: %w", serviceName, err)
	}

	return ServiceState{
		Name:   serviceName,
		Active: false,
		State:  "unknown",
	}, nil
}
