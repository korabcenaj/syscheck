package config_test

import (
	"bytes"
	"errors"
	"flag"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/korabcenaj/syscheck/internal/config"
)

func TestParse(t *testing.T) {
	t.Run("returns defaults when no arguments provided", func(t *testing.T) {
		var buf bytes.Buffer
		cfg, err := config.Parse([]string{}, &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		defaults := config.DefaultConfig()
		if !reflect.DeepEqual(cfg, defaults) {
			t.Errorf("got %+v, want %+v", cfg, defaults)
		}
	})

	t.Run("extracts positional target arguments", func(t *testing.T) {
		var buf bytes.Buffer
		args := []string{"-format", "json", "server1", "server2", "192.168.1.50"}

		cfg, err := config.Parse(args, &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedTargets := []string{"server1", "server2", "192.168.1.50"}
		if !reflect.DeepEqual(cfg.Targets, expectedTargets) {
			t.Errorf("Targets = %v, want %v", cfg.Targets, expectedTargets)
		}
	})

	t.Run("parses valid custom flags", func(t *testing.T) {
		var buf bytes.Buffer
		args := []string{
			"-load-warn", "3.5",
			"-load-crit", "7.0",
			"-mem-warn", "75.5",
			"-mem-crit", "85.0",
			"-disk-warn", "60.0",
			"-disk-crit", "75.0",
			"-disk-path", "/var/log",
			"-format", "json",
		}

		cfg, err := config.Parse(args, &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.LoadWarn != 3.5 || cfg.LoadCrit != 7.0 {
			t.Errorf("load thresholds = (%.1f, %.1f), want (3.5, 7.0)", cfg.LoadWarn, cfg.LoadCrit)
		}
		if cfg.MemWarn != 75.5 || cfg.MemCrit != 85.0 {
			t.Errorf("mem thresholds = (%.1f, %.1f), want (75.5, 85.0)", cfg.MemWarn, cfg.MemCrit)
		}
		if cfg.DiskWarn != 60.0 || cfg.DiskCrit != 75.0 {
			t.Errorf("disk thresholds = (%.1f, %.1f), want (60.0, 75.0)", cfg.DiskWarn, cfg.DiskCrit)
		}
		if cfg.DiskPath != "/var/log" {
			t.Errorf("disk path = %q, want \"/var/log\"", cfg.DiskPath)
		}
		if cfg.Format != "json" {
			t.Errorf("format = %q, want \"json\"", cfg.Format)
		}
	})

	t.Run("handles -help flag by returning flag.ErrHelp", func(t *testing.T) {
		var buf bytes.Buffer
		_, err := config.Parse([]string{"-help"}, &buf)
		if !errors.Is(err, flag.ErrHelp) {
			t.Errorf("expected flag.ErrHelp, got: %v", err)
		}
		if buf.Len() == 0 {
			t.Error("expected help output to be written to output writer")
		}
	})

	t.Run("returns error on unknown flag", func(t *testing.T) {
		var buf bytes.Buffer
		_, err := config.Parse([]string{"-unknown-flag"}, &buf)
		if err == nil {
			t.Fatal("expected error for unknown flag, got nil")
		}
	})

	t.Run("parses procs and services flags with whitespace trimming", func(t *testing.T) {
		var buf bytes.Buffer
		args := []string{"-procs", "sshd, cron, nginx ", "-services", "docker, systemd-resolved"}
		cfg, err := config.Parse(args, &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedProcs := []string{"sshd", "cron", "nginx"}
		if !reflect.DeepEqual(cfg.Procs, expectedProcs) {
			t.Errorf("Procs = %v, want %v", cfg.Procs, expectedProcs)
		}

		expectedServices := []string{"docker", "systemd-resolved"}
		if !reflect.DeepEqual(cfg.Services, expectedServices) {
			t.Errorf("Services = %v, want %v", cfg.Services, expectedServices)
		}
	})

	t.Run("parses tcp and http flags with whitespace trimming", func(t *testing.T) {
		var buf bytes.Buffer
		args := []string{
			"-tcp", "127.0.0.1:5432, localhost:6379 ",
			"-http", "http://localhost:8080/health, https://example.com/api ",
		}
		cfg, err := config.Parse(args, &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expectedTCP := []string{"127.0.0.1:5432", "localhost:6379"}
		if !reflect.DeepEqual(cfg.TCPTargets, expectedTCP) {
			t.Errorf("TCPTargets = %v, want %v", cfg.TCPTargets, expectedTCP)
		}

		expectedHTTP := []string{"http://localhost:8080/health", "https://example.com/api"}
		if !reflect.DeepEqual(cfg.HTTPTargets, expectedHTTP) {
			t.Errorf("HTTPTargets = %v, want %v", cfg.HTTPTargets, expectedHTTP)
		}
	})
}

func TestValidate(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*config.Config)
		wantErr bool
	}{
		{
			name:    "valid default config",
			mutate:  func(c *config.Config) {},
			wantErr: false,
		},
		{
			name: "load warn exceeds load crit",
			mutate: func(c *config.Config) {
				c.LoadWarn = 10.0
				c.LoadCrit = 5.0
			},
			wantErr: true,
		},
		{
			name: "negative load warn",
			mutate: func(c *config.Config) {
				c.LoadWarn = -1.0
			},
			wantErr: true,
		},
		{
			name: "memory percentage over 100",
			mutate: func(c *config.Config) {
				c.MemCrit = 105.0
			},
			wantErr: true,
		},
		{
			name: "memory warn exceeds memory crit",
			mutate: func(c *config.Config) {
				c.MemWarn = 90.0
				c.MemCrit = 80.0
			},
			wantErr: true,
		},
		{
			name: "disk percentage negative",
			mutate: func(c *config.Config) {
				c.DiskWarn = -5.0
			},
			wantErr: true,
		},
		{
			name: "disk warn exceeds disk crit",
			mutate: func(c *config.Config) {
				c.DiskWarn = 95.0
				c.DiskCrit = 90.0
			},
			wantErr: true,
		},
		{
			name: "empty disk path",
			mutate: func(c *config.Config) {
				c.DiskPath = ""
			},
			wantErr: true,
		},
		{
			name: "unsupported format",
			mutate: func(c *config.Config) {
				c.Format = "yaml"
			},
			wantErr: true,
		},
		{
			name: "valid json format",
			mutate: func(c *config.Config) {
				c.Format = "json"
			},
			wantErr: false,
		},
		{
			name: "invalid tcp target missing port",
			mutate: func(c *config.Config) {
				c.TCPTargets = []string{"localhost"}
			},
			wantErr: true,
		},
		{
			name: "valid tcp target",
			mutate: func(c *config.Config) {
				c.TCPTargets = []string{"localhost:8080", "192.168.1.1:22"}
			},
			wantErr: false,
		},
		{
			name: "invalid http target missing scheme",
			mutate: func(c *config.Config) {
				c.HTTPTargets = []string{"localhost:8080/health"}
			},
			wantErr: true,
		},
		{
			name: "invalid http target unsupported scheme",
			mutate: func(c *config.Config) {
				c.HTTPTargets = []string{"ftp://example.com/file"}
			},
			wantErr: true,
		},
		{
			name: "valid http targets",
			mutate: func(c *config.Config) {
				c.HTTPTargets = []string{"http://localhost:8080/health", "https://example.com"}
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			tt.mutate(&cfg)
			err := cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoadFile(t *testing.T) {
	t.Run("loads complete valid configuration", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "syscheck.json")
		content := `{
			"load_warn": 3.0,
			"load_crit": 6.0,
			"mem_warn": 75.0,
			"mem_crit": 85.0,
			"disk_warn": 70.0,
			"disk_crit": 85.0,
			"disk_path": "/var",
			"format": "json",
			"targets": ["node1", "node2"],
			"procs": ["sshd", "cron"],
			"services": ["dbus"],
			"tcp": ["127.0.0.1:5432"],
			"http": ["http://localhost:8080/health"]
		}`
		if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := config.LoadFile(cfgPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.LoadWarn != 3.0 || cfg.LoadCrit != 6.0 {
			t.Errorf("Load thresholds = (%.1f, %.1f), want (3.0, 6.0)", cfg.LoadWarn, cfg.LoadCrit)
		}
		if cfg.MemWarn != 75.0 || cfg.MemCrit != 85.0 {
			t.Errorf("Mem thresholds = (%.1f, %.1f), want (75.0, 85.0)", cfg.MemWarn, cfg.MemCrit)
		}
		if cfg.DiskPath != "/var" || cfg.Format != "json" {
			t.Errorf("DiskPath=%q, Format=%q", cfg.DiskPath, cfg.Format)
		}
		if !reflect.DeepEqual(cfg.Targets, []string{"node1", "node2"}) {
			t.Errorf("Targets = %v", cfg.Targets)
		}
		if !reflect.DeepEqual(cfg.Procs, []string{"sshd", "cron"}) {
			t.Errorf("Procs = %v", cfg.Procs)
		}
		if !reflect.DeepEqual(cfg.Services, []string{"dbus"}) {
			t.Errorf("Services = %v", cfg.Services)
		}
		if !reflect.DeepEqual(cfg.TCPTargets, []string{"127.0.0.1:5432"}) {
			t.Errorf("TCPTargets = %v", cfg.TCPTargets)
		}
		if !reflect.DeepEqual(cfg.HTTPTargets, []string{"http://localhost:8080/health"}) {
			t.Errorf("HTTPTargets = %v", cfg.HTTPTargets)
		}
		if cfg.ConfigFile != cfgPath {
			t.Errorf("ConfigFile = %q, want %q", cfg.ConfigFile, cfgPath)
		}
	})

	t.Run("loads partial configuration with defaults preserved", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "partial.json")
		content := `{"load_warn": 1.5}`
		if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		cfg, err := config.LoadFile(cfgPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.LoadWarn != 1.5 {
			t.Errorf("LoadWarn = %.1f, want 1.5", cfg.LoadWarn)
		}
		defaults := config.DefaultConfig()
		if cfg.LoadCrit != defaults.LoadCrit {
			t.Errorf("LoadCrit = %.1f, want default %.1f", cfg.LoadCrit, defaults.LoadCrit)
		}
		if cfg.MemWarn != defaults.MemWarn {
			t.Errorf("MemWarn = %.1f, want default %.1f", cfg.MemWarn, defaults.MemWarn)
		}
	})

	t.Run("returns error on missing file", func(t *testing.T) {
		_, err := config.LoadFile("/path/does/not/exist/cfg.json")
		if err == nil {
			t.Error("expected error on missing file, got nil")
		}
	})

	t.Run("returns error on malformed JSON", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "bad.json")
		_ = os.WriteFile(cfgPath, []byte(`{not-json}`), 0644)

		_, err := config.LoadFile(cfgPath)
		if err == nil {
			t.Error("expected error on bad JSON, got nil")
		}
	})

	t.Run("returns error on invalid threshold configuration", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "invalid.json")
		_ = os.WriteFile(cfgPath, []byte(`{"load_warn": 10.0, "load_crit": 5.0}`), 0644)

		_, err := config.LoadFile(cfgPath)
		if err == nil {
			t.Error("expected error on invalid thresholds, got nil")
		}
	})
}

func TestParseWithConfigFile(t *testing.T) {
	t.Run("file configuration overrides defaults", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "file_override.json")
		_ = os.WriteFile(cfgPath, []byte(`{
			"load_warn": 3.5,
			"load_crit": 7.0,
			"targets": ["cfg-host-1", "cfg-host-2"]
		}`), 0644)

		var buf bytes.Buffer
		cfg, err := config.Parse([]string{"-config", cfgPath}, &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if cfg.LoadWarn != 3.5 || cfg.LoadCrit != 7.0 {
			t.Errorf("Load = (%.1f, %.1f), want (3.5, 7.0)", cfg.LoadWarn, cfg.LoadCrit)
		}
		expectedTargets := []string{"cfg-host-1", "cfg-host-2"}
		if !reflect.DeepEqual(cfg.Targets, expectedTargets) {
			t.Errorf("Targets = %v, want %v", cfg.Targets, expectedTargets)
		}
		// Default should still apply for unspecified fields
		if cfg.MemWarn != 80.0 {
			t.Errorf("MemWarn = %.1f, want default 80.0", cfg.MemWarn)
		}
	})

	t.Run("CLI flags override configuration file values", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "cli_precedence.json")
		_ = os.WriteFile(cfgPath, []byte(`{
			"load_warn": 3.5,
			"load_crit": 7.0,
			"mem_warn": 85.0
		}`), 0644)

		var buf bytes.Buffer
		cfg, err := config.Parse([]string{
			"-config", cfgPath,
			"-load-warn", "4.0",
		}, &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// -load-warn was explicitly provided on CLI -> overrides file (4.0)
		if cfg.LoadWarn != 4.0 {
			t.Errorf("LoadWarn = %.1f, want CLI value 4.0", cfg.LoadWarn)
		}
		// -load-crit was NOT provided on CLI -> keeps file value (7.0)
		if cfg.LoadCrit != 7.0 {
			t.Errorf("LoadCrit = %.1f, want file value 7.0", cfg.LoadCrit)
		}
		// -mem-warn was NOT provided on CLI -> keeps file value (85.0)
		if cfg.MemWarn != 85.0 {
			t.Errorf("MemWarn = %.1f, want file value 85.0", cfg.MemWarn)
		}
	})

	t.Run("CLI positional arguments override config file targets", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "targets.json")
		_ = os.WriteFile(cfgPath, []byte(`{"targets": ["file-node-1", "file-node-2"]}`), 0644)

		var buf bytes.Buffer
		cfg, err := config.Parse([]string{
			"-config", cfgPath,
			"cli-node-1", "cli-node-2", "cli-node-3",
		}, &buf)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		expected := []string{"cli-node-1", "cli-node-2", "cli-node-3"}
		if !reflect.DeepEqual(cfg.Targets, expected) {
			t.Errorf("Targets = %v, want %v", cfg.Targets, expected)
		}
	})

	t.Run("returns error when specified config file does not exist", func(t *testing.T) {
		var buf bytes.Buffer
		_, err := config.Parse([]string{"-config", "/non/existent/file.json"}, &buf)
		if err == nil {
			t.Error("expected error on missing config file, got nil")
		}
	})

	t.Run("returns error when config file contains invalid JSON", func(t *testing.T) {
		tempDir := t.TempDir()
		cfgPath := filepath.Join(tempDir, "bad.json")
		_ = os.WriteFile(cfgPath, []byte(`{invalid-json`), 0644)

		var buf bytes.Buffer
		_, err := config.Parse([]string{"-config", cfgPath}, &buf)
		if err == nil {
			t.Error("expected error on bad JSON config file, got nil")
		}
	})
}

