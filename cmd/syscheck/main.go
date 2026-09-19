package main

import (
	"fmt"
	"os"

	"syscheck/internal/load"
	"syscheck/internal/memory"
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

	mem, err := memory.Read()
	if err != nil {
		return fmt.Errorf("memory check: %w", err)
	}

	fmt.Printf("System Load: %.2f (1m), %.2f (5m), %.2f (15m)\n",
		avg.One, avg.Five, avg.Fifteen)

	fmt.Printf("Memory:      %s / %s (%.1f%% used)\n",
		formatBytes(mem.UsedBytes()),
		formatBytes(mem.TotalBytes),
		mem.UsedPercent())

	if mem.SwapTotalBytes > 0 {
		fmt.Printf("Swap:        %s / %s (%.1f%% used)\n",
			formatBytes(mem.SwapUsedBytes()),
			formatBytes(mem.SwapTotalBytes),
			mem.SwapUsedPercent())
	} else {
		fmt.Printf("Swap:        none configured\n")
	}

	return nil
}

// formatBytes returns a human-readable string formatted in binary units (KiB, MiB, GiB, TiB).
func formatBytes(b uint64) string {
	const (
		kib = 1024
		mib = 1024 * kib
		gib = 1024 * mib
		tib = 1024 * gib
	)

	switch {
	case b >= tib:
		return fmt.Sprintf("%.2f TiB", float64(b)/float64(tib))
	case b >= gib:
		return fmt.Sprintf("%.2f GiB", float64(b)/float64(gib))
	case b >= mib:
		return fmt.Sprintf("%.2f MiB", float64(b)/float64(mib))
	case b >= kib:
		return fmt.Sprintf("%.2f KiB", float64(b)/float64(kib))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
