// Package config handles command-line flag parsing and validation for syscheck.
package config

import (
	"flag"
	"fmt"
	"io"
	"net"
	"net/url"
	"strings"
)

// Config holds runtime configuration options and threshold limits.
type Config struct {
	LoadWarn float64
	LoadCrit float64

	MemWarn float64
	MemCrit float64

	DiskWarn float64
	DiskCrit float64
	DiskPath    string
	Format      string
	Targets     []string
	Procs       []string
	Services    []string
	TCPTargets  []string
	HTTPTargets []string
}

// DefaultConfig returns production-safe default thresholds.
func DefaultConfig() Config {
	return Config{
		LoadWarn: 2.0,
		LoadCrit: 5.0,

		MemWarn: 80.0,
		MemCrit: 90.0,

		DiskWarn: 80.0,
		DiskCrit: 90.0,
		DiskPath: "/",
		Format:   "text",
		Targets:  []string{"localhost"},
	}
}

// Parse parses CLI arguments from args using an isolated flag.FlagSet.
// The output writer receives usage and flag error text.
func Parse(args []string, output io.Writer) (Config, error) {
	cfg := DefaultConfig()

	fs := flag.NewFlagSet("syscheck", flag.ContinueOnError)
	fs.SetOutput(output)

	fs.Float64Var(&cfg.LoadWarn, "load-warn", cfg.LoadWarn, "1-minute load average warning threshold")
	fs.Float64Var(&cfg.LoadCrit, "load-crit", cfg.LoadCrit, "1-minute load average critical threshold")

	fs.Float64Var(&cfg.MemWarn, "mem-warn", cfg.MemWarn, "Memory percentage warning threshold (0-100)")
	fs.Float64Var(&cfg.MemCrit, "mem-crit", cfg.MemCrit, "Memory percentage critical threshold (0-100)")

	fs.Float64Var(&cfg.DiskWarn, "disk-warn", cfg.DiskWarn, "Disk percentage warning threshold (0-100)")
	fs.Float64Var(&cfg.DiskCrit, "disk-crit", cfg.DiskCrit, "Disk percentage critical threshold (0-100)")

	fs.StringVar(&cfg.DiskPath, "disk-path", cfg.DiskPath, "Filesystem path to monitor for disk space and inodes")
	fs.StringVar(&cfg.Format, "format", cfg.Format, "Output format: 'text' (default) or 'json'")

	var procsFlag, servicesFlag, tcpFlag, httpFlag string
	fs.StringVar(&procsFlag, "procs", "", "Comma-separated list of process names to monitor (e.g. 'sshd,cron')")
	fs.StringVar(&servicesFlag, "services", "", "Comma-separated list of system services to monitor (e.g. 'sshd,docker')")
	fs.StringVar(&tcpFlag, "tcp", "", "Comma-separated list of TCP targets to check in host:port format (e.g. '127.0.0.1:5432,localhost:80')")
	fs.StringVar(&httpFlag, "http", "", "Comma-separated list of HTTP/HTTPS URLs to check (e.g. 'http://localhost:8080/health,https://example.com')")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if procsFlag != "" {
		for _, p := range strings.Split(procsFlag, ",") {
			trimmed := strings.TrimSpace(p)
			if trimmed != "" {
				cfg.Procs = append(cfg.Procs, trimmed)
			}
		}
	}

	if servicesFlag != "" {
		for _, s := range strings.Split(servicesFlag, ",") {
			trimmed := strings.TrimSpace(s)
			if trimmed != "" {
				cfg.Services = append(cfg.Services, trimmed)
			}
		}
	}

	if tcpFlag != "" {
		for _, t := range strings.Split(tcpFlag, ",") {
			trimmed := strings.TrimSpace(t)
			if trimmed != "" {
				cfg.TCPTargets = append(cfg.TCPTargets, trimmed)
			}
		}
	}

	if httpFlag != "" {
		for _, u := range strings.Split(httpFlag, ",") {
			trimmed := strings.TrimSpace(u)
			if trimmed != "" {
				cfg.HTTPTargets = append(cfg.HTTPTargets, trimmed)
			}
		}
	}

	if posArgs := fs.Args(); len(posArgs) > 0 {
		cfg.Targets = posArgs
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// Validate checks that the configuration values are logically sound.
func (c Config) Validate() error {
	// Load thresholds validation
	if c.LoadWarn < 0 {
		return fmt.Errorf("load warning threshold cannot be negative (got %.2f)", c.LoadWarn)
	}
	if c.LoadCrit < 0 {
		return fmt.Errorf("load critical threshold cannot be negative (got %.2f)", c.LoadCrit)
	}
	if c.LoadWarn > 0 && c.LoadCrit > 0 && c.LoadWarn > c.LoadCrit {
		return fmt.Errorf("load warning threshold (%.2f) cannot exceed critical threshold (%.2f)",
			c.LoadWarn, c.LoadCrit)
	}

	// Memory thresholds validation
	if c.MemWarn < 0 || c.MemWarn > 100 {
		return fmt.Errorf("memory warning threshold must be between 0 and 100 (got %.1f)", c.MemWarn)
	}
	if c.MemCrit < 0 || c.MemCrit > 100 {
		return fmt.Errorf("memory critical threshold must be between 0 and 100 (got %.1f)", c.MemCrit)
	}
	if c.MemWarn > 0 && c.MemCrit > 0 && c.MemWarn > c.MemCrit {
		return fmt.Errorf("memory warning threshold (%.1f%%) cannot exceed critical threshold (%.1f%%)",
			c.MemWarn, c.MemCrit)
	}

	// Disk thresholds validation
	if c.DiskWarn < 0 || c.DiskWarn > 100 {
		return fmt.Errorf("disk warning threshold must be between 0 and 100 (got %.1f)", c.DiskWarn)
	}
	if c.DiskCrit < 0 || c.DiskCrit > 100 {
		return fmt.Errorf("disk critical threshold must be between 0 and 100 (got %.1f)", c.DiskCrit)
	}
	if c.DiskWarn > 0 && c.DiskCrit > 0 && c.DiskWarn > c.DiskCrit {
		return fmt.Errorf("disk warning threshold (%.1f%%) cannot exceed critical threshold (%.1f%%)",
			c.DiskWarn, c.DiskCrit)
	}

	// Disk path validation
	if c.DiskPath == "" {
		return fmt.Errorf("disk path cannot be empty")
	}

	// Format validation
	if c.Format != "text" && c.Format != "json" {
		return fmt.Errorf("unsupported output format %q: choose 'text' or 'json'", c.Format)
	}

	// TCP targets validation
	for _, target := range c.TCPTargets {
		if _, _, err := net.SplitHostPort(target); err != nil {
			return fmt.Errorf("invalid tcp target %q: expected host:port format (%w)", target, err)
		}
	}

	// HTTP targets validation
	for _, target := range c.HTTPTargets {
		parsed, err := url.Parse(target)
		if err != nil || parsed.Scheme == "" || parsed.Host == "" {
			return fmt.Errorf("invalid http target %q: valid scheme and host required", target)
		}
		scheme := strings.ToLower(parsed.Scheme)
		if scheme != "http" && scheme != "https" {
			return fmt.Errorf("invalid http target %q: scheme must be http or https (got %q)", target, parsed.Scheme)
		}
	}

	return nil
}
