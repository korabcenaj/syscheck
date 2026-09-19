package main

import (
	"fmt"
	"os"

	"syscheck/internal/check"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// run orchestrates the health check runner and displays formatted results.
func run() error {
	runner := check.NewRunner(
		&check.LoadChecker{
			WarnThreshold: 2.0,
			CritThreshold: 5.0,
		},
		&check.MemoryChecker{
			WarnPercent: 80.0,
			CritPercent: 90.0,
		},
		&check.DiskChecker{
			Path:        "/",
			WarnPercent: 80.0,
			CritPercent: 90.0,
		},
	)

	results := runner.RunAll()

	for _, res := range results {
		fmt.Printf("[%-8s] %-14s %s\n", res.Status, res.Name, res.Message)
	}

	overall := check.OverallStatus(results)
	fmt.Printf("\nOverall Health: %s\n", overall)

	return nil
}
