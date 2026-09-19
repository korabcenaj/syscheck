package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/korabcenaj/syscheck/internal/check"
	"github.com/korabcenaj/syscheck/internal/config"
	"github.com/korabcenaj/syscheck/internal/host"
	"github.com/korabcenaj/syscheck/internal/netcheck"
	"github.com/korabcenaj/syscheck/internal/output"
)

func main() {
	exitCode := run(os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(exitCode)
}

// run encapsulates CLI execution, accepting arguments and writers for stdout and stderr.
// It returns an integer exit code corresponding to overall cluster health:
// 0 = OK, 1 = WARNING, 2 = CRITICAL, 3 = UNKNOWN/ERROR.
func run(args []string, stdout, stderr io.Writer) int {
	cfg, err := config.Parse(args, stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return check.StatusUnknown.ExitCode()
	}

	formatter, err := output.NewFormatter(cfg.Format)
	if err != nil {
		fmt.Fprintf(stderr, "Error: %v\n", err)
		return check.StatusUnknown.ExitCode()
	}

	targetReports := make([]output.TargetReport, 0, len(cfg.Targets))
	allResults := make([]check.Result, 0)

	for _, target := range cfg.Targets {
		if isLocalTarget(target) {
			hostInfo, _ := host.Read()

			runner := check.NewRunner(
				&check.LoadChecker{
					WarnThreshold: cfg.LoadWarn,
					CritThreshold: cfg.LoadCrit,
				},
				&check.MemoryChecker{
					WarnPercent: cfg.MemWarn,
					CritPercent: cfg.MemCrit,
				},
				&check.DiskChecker{
					Path:        cfg.DiskPath,
					WarnPercent: cfg.DiskWarn,
					CritPercent: cfg.DiskCrit,
				},
			)

			results := runner.RunAll()
			targetOverall := check.OverallStatus(results)
			allResults = append(allResults, results...)

			targetReports = append(targetReports, output.TargetReport{
				Target:   target,
				HostInfo: &hostInfo,
				Overall:  targetOverall,
				Checks:   results,
			})
		} else {
			// Remote target: probe network reachability on SSH port 22
			runner := check.NewRunner(
				&netcheck.ReachabilityChecker{
					Target:  target,
					Port:    22,
					Timeout: 2 * time.Second,
				},
			)

			results := runner.RunAll()
			targetOverall := check.OverallStatus(results)
			allResults = append(allResults, results...)

			targetReports = append(targetReports, output.TargetReport{
				Target:  target,
				Overall: targetOverall,
				Checks:  results,
			})
		}
	}

	overall := check.OverallStatus(allResults)

	report := output.Report{
		Timestamp: time.Now().UTC(),
		Overall:   overall,
		Targets:   targetReports,
	}

	if err := formatter.Format(stdout, report); err != nil {
		fmt.Fprintf(stderr, "Error formatting report: %v\n", err)
		return check.StatusUnknown.ExitCode()
	}

	return overall.ExitCode()
}

// isLocalTarget returns true if the target represents the current local machine.
func isLocalTarget(target string) bool {
	switch strings.ToLower(target) {
	case "localhost", "127.0.0.1", "::1", "":
		return true
	}

	hostname, err := os.Hostname()
	if err == nil && strings.EqualFold(target, hostname) {
		return true
	}

	return false
}
