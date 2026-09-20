package config_test

import (
	"bytes"
	"errors"
	"flag"
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
