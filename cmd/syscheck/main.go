package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"

	"syscheck/internal/check"
	"syscheck/internal/config"
)

func main() {
	exitCode := run(os.Args[1:], os.Stdout, os.Stderr)
	os.Exit(exitCode)
}

// run encapsulates CLI execution, accepting arguments and writers for stdout and stderr.
// It returns an integer exit code corresponding to the overall system health:
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

	for _, res := range results {
		fmt.Fprintf(stdout, "[%-8s] %-14s %s\n", res.Status, res.Name, res.Message)
	}

	overall := check.OverallStatus(results)
	fmt.Fprintf(stdout, "\nOverall Health: %s\n", overall)

	return overall.ExitCode()
}
