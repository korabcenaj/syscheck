package main

import (
	"fmt"
	"os"

	"syscheck/internal/load"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// run encapsulates the main program logic, returning an error instead of
// directly calling os.Exit. This ensures deferred calls execute cleanly
// and makes the function straightforward to test.
func run() error {
	avg, err := load.Read()
	if err != nil {
		return fmt.Errorf("system load check: %w", err)
	}

	fmt.Printf("System Load Average: %.2f (1m), %.2f (5m), %.2f (15m)\n",
		avg.One, avg.Five, avg.Fifteen)

	return nil
}
