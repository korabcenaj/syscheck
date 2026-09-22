// Package config handles command-line flag parsing and validation for syscheck.
package config

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
)

// Config holds runtime configuration options and threshold limits.
type Config struct {
	LoadWarn    float64  `json:"load_warn"`
	LoadCrit    float64  `json:"load_crit"`
	MemWarn     float64  `json:"mem_warn"`
	MemCrit     float64  `json:"mem_crit"`
	DiskWarn    float64  `json:"disk_warn"`
	DiskCrit    float64  `json:"disk_crit"`
	DiskPath    string   `json:"disk_path"`
	Format      string   `json:"format"`
	Targets     []string `json:"targets"`
	Procs       []string `json:"procs"`
	Services    []string `json:"services"`
	TCPTargets  []string `json:"tcp"`
	HTTPTargets []string `json:"http"`
	ConfigFile  string   `json:"-"`
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

// fileConfig mirrors Config using pointers to identify explicitly supplied JSON fields.
type fileConfig struct {
	LoadWarn    *float64 `json:"load_warn"`
	LoadCrit    *float64 `json:"load_crit"`
	MemWarn     *float64 `json:"mem_warn"`
	MemCrit     *float64 `json:"mem_crit"`
	DiskWarn    *float64 `json:"disk_warn"`
	DiskCrit    *float64 `json:"disk_crit"`
	DiskPath    *string  `json:"disk_path"`
	Format      *string  `json:"format"`
	Targets     []string `json:"targets"`
	Procs       []string `json:"procs"`
	Services    []string `json:"services"`
	TCP         []string `json:"tcp"`
	TCPTargets  []string `json:"tcp_targets"`
	HTTP        []string `json:"http"`
	HTTPTargets []string `json:"http_targets"`
}

func (fc fileConfig) applyTo(cfg *Config) {
	if fc.LoadWarn != nil {
		cfg.LoadWarn = *fc.LoadWarn
	}
	if fc.LoadCrit != nil {
		cfg.LoadCrit = *fc.LoadCrit
	}
	if fc.MemWarn != nil {
		cfg.MemWarn = *fc.MemWarn
	}
	if fc.MemCrit != nil {
		cfg.MemCrit = *fc.MemCrit
	}
	if fc.DiskWarn != nil {
		cfg.DiskWarn = *fc.DiskWarn
	}
	if fc.DiskCrit != nil {
		cfg.DiskCrit = *fc.DiskCrit
	}
	if fc.DiskPath != nil {
		cfg.DiskPath = *fc.DiskPath
	}
	if fc.Format != nil {
		cfg.Format = *fc.Format
	}
	if len(fc.Targets) > 0 {
		cfg.Targets = fc.Targets
	}
	if len(fc.Procs) > 0 {
		cfg.Procs = fc.Procs
	}
	if len(fc.Services) > 0 {
		cfg.Services = fc.Services
	}
	if len(fc.TCP) > 0 {
		cfg.TCPTargets = fc.TCP
	} else if len(fc.TCPTargets) > 0 {
		cfg.TCPTargets = fc.TCPTargets
	}
	if len(fc.HTTP) > 0 {
		cfg.HTTPTargets = fc.HTTP
	} else if len(fc.HTTPTargets) > 0 {
		cfg.HTTPTargets = fc.HTTPTargets
	}
}

// LoadFile reads a JSON configuration file from disk and parses it into a Config.
// Unspecified fields retain their DefaultConfig() values.
func LoadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("reading config file %q: %w", path, err)
	}

	cfg := DefaultConfig()
	var fc fileConfig
	if err := json.Unmarshal(data, &fc); err != nil {
		return Config{}, fmt.Errorf("parsing config file %q: %w", path, err)
	}

	fc.applyTo(&cfg)
	cfg.ConfigFile = path

	if err := cfg.Validate(); err != nil {
		return Config{}, fmt.Errorf("validating config file %q: %w", path, err)
	}

	return cfg, nil
}

// Parse parses CLI arguments from args using an isolated flag.FlagSet.
// Precedence order: CLI flags > Configuration file > DefaultConfig().
// The output writer receives usage and flag error text.
func Parse(args []string, output io.Writer) (Config, error) {
	defaults := DefaultConfig()
	cfg := DefaultConfig()

	fs := flag.NewFlagSet("syscheck", flag.ContinueOnError)
	fs.SetOutput(output)

	var (
		configFlag   string
		loadWarn     float64
		loadCrit     float64
		memWarn      float64
		memCrit      float64
		diskWarn     float64
		diskCrit     float64
		diskPath     string
		format       string
		procsFlag    string
		servicesFlag string
		tcpFlag      string
		httpFlag     string
	)

	fs.StringVar(&configFlag, "config", "", "Path to JSON configuration file")
	fs.Float64Var(&loadWarn, "load-warn", defaults.LoadWarn, "1-minute load average warning threshold")
	fs.Float64Var(&loadCrit, "load-crit", defaults.LoadCrit, "1-minute load average critical threshold")
	fs.Float64Var(&memWarn, "mem-warn", defaults.MemWarn, "Memory percentage warning threshold (0-100)")
	fs.Float64Var(&memCrit, "mem-crit", defaults.MemCrit, "Memory percentage critical threshold (0-100)")
	fs.Float64Var(&diskWarn, "disk-warn", defaults.DiskWarn, "Disk percentage warning threshold (0-100)")
	fs.Float64Var(&diskCrit, "disk-crit", defaults.DiskCrit, "Disk percentage critical threshold (0-100)")
	fs.StringVar(&diskPath, "disk-path", defaults.DiskPath, "Filesystem path to monitor for disk space and inodes")
	fs.StringVar(&format, "format", defaults.Format, "Output format: 'text' (default) or 'json'")
	fs.StringVar(&procsFlag, "procs", "", "Comma-separated list of process names to monitor (e.g. 'sshd,cron')")
	fs.StringVar(&servicesFlag, "services", "", "Comma-separated list of system services to monitor (e.g. 'sshd,docker')")
	fs.StringVar(&tcpFlag, "tcp", "", "Comma-separated list of TCP targets to check in host:port format (e.g. '127.0.0.1:5432,localhost:80')")
	fs.StringVar(&httpFlag, "http", "", "Comma-separated list of HTTP/HTTPS URLs to check (e.g. 'http://localhost:8080/health,https://example.com')")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	// 1. If -config was provided, load file values over defaults
	if configFlag != "" {
		data, err := os.ReadFile(configFlag)
		if err != nil {
			return Config{}, fmt.Errorf("reading config file %q: %w", configFlag, err)
		}
		var fc fileConfig
		if err := json.Unmarshal(data, &fc); err != nil {
			return Config{}, fmt.Errorf("parsing config file %q: %w", configFlag, err)
		}
		fc.applyTo(&cfg)
		cfg.ConfigFile = configFlag
	}

	// 2. Explicitly supplied CLI flags override config file and defaults
	fs.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "load-warn":
			cfg.LoadWarn = loadWarn
		case "load-crit":
			cfg.LoadCrit = loadCrit
		case "mem-warn":
			cfg.MemWarn = memWarn
		case "mem-crit":
			cfg.MemCrit = memCrit
		case "disk-warn":
			cfg.DiskWarn = diskWarn
		case "disk-crit":
			cfg.DiskCrit = diskCrit
		case "disk-path":
			cfg.DiskPath = diskPath
		case "format":
			cfg.Format = format
		case "procs":
			cfg.Procs = splitComma(procsFlag)
		case "services":
			cfg.Services = splitComma(servicesFlag)
		case "tcp":
			cfg.TCPTargets = splitComma(tcpFlag)
		case "http":
			cfg.HTTPTargets = splitComma(httpFlag)
		}
	})

	// 3. Explicit positional arguments override targets
	if posArgs := fs.Args(); len(posArgs) > 0 {
		cfg.Targets = posArgs
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

func splitComma(s string) []string {
	var result []string
	for _, p := range strings.Split(s, ",") {
		trimmed := strings.TrimSpace(p)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
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
