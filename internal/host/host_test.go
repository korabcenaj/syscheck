package host_test

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/korabcenaj/syscheck/internal/host"
)

const sampleFedoraOSRelease = `
NAME="Fedora Linux"
VERSION="44 (KDE Plasma Desktop Edition)"
ID=fedora
VERSION_ID=44
PRETTY_NAME="Fedora Linux 44 (KDE Plasma Desktop Edition)"
HOME_URL="https://fedoraproject.org/"
`

const sampleUbuntuOSRelease = `
# This is a comment
NAME="Ubuntu"
VERSION="22.04.3 LTS (Jammy Jellyfish)"
ID=ubuntu
VERSION_ID="22.04"
PRETTY_NAME="Ubuntu 22.04.3 LTS"
`

const sampleMinimalOSRelease = `
NAME=Alpine
VERSION_ID=3.19.1
`

func TestParseOSRelease(t *testing.T) {
	t.Run("parses standard Fedora os-release", func(t *testing.T) {
		pretty, name, ver, err := host.ParseOSRelease(strings.NewReader(sampleFedoraOSRelease))
		if err != nil {
			t.Fatalf("ParseOSRelease err = %v", err)
		}
		if pretty != "Fedora Linux 44 (KDE Plasma Desktop Edition)" {
			t.Errorf("pretty = %q", pretty)
		}
		if name != "Fedora Linux" {
			t.Errorf("name = %q", name)
		}
		if ver != "44" {
			t.Errorf("ver = %q", ver)
		}
	})

	t.Run("parses Ubuntu with comments and quotes", func(t *testing.T) {
		pretty, name, ver, err := host.ParseOSRelease(strings.NewReader(sampleUbuntuOSRelease))
		if err != nil {
			t.Fatalf("ParseOSRelease err = %v", err)
		}
		if pretty != "Ubuntu 22.04.3 LTS" {
			t.Errorf("pretty = %q", pretty)
		}
		if name != "Ubuntu" {
			t.Errorf("name = %q", name)
		}
		if ver != "22.04" {
			t.Errorf("ver = %q", ver)
		}
	})

	t.Run("falls back to NAME + VERSION when PRETTY_NAME is absent", func(t *testing.T) {
		pretty, name, ver, err := host.ParseOSRelease(strings.NewReader(sampleMinimalOSRelease))
		if err != nil {
			t.Fatalf("ParseOSRelease err = %v", err)
		}
		if pretty != "Alpine 3.19.1" {
			t.Errorf("pretty = %q, want 'Alpine 3.19.1'", pretty)
		}
		if name != "Alpine" {
			t.Errorf("name = %q", name)
		}
		if ver != "3.19.1" {
			t.Errorf("ver = %q", ver)
		}
	})
}

func TestParseUptime(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{
			name:  "standard /proc/uptime entry",
			input: "306088.29 1244106.08\n",
			want:  time.Duration(306088.29 * float64(time.Second)),
		},
		{
			name:  "single field uptime",
			input: "120.00",
			want:  120 * time.Second,
		},
		{
			name:    "empty uptime",
			input:   "",
			wantErr: true,
		},
		{
			name:    "non-numeric uptime",
			input:   "notanumber 12345",
			wantErr: true,
		},
		{
			name:    "negative uptime",
			input:   "-100.5 500",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := host.ParseUptime(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseUptime(%q) error = %v, wantErr %v", tt.input, err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("ParseUptime() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFormattedUptime(t *testing.T) {
	tests := []struct {
		duration time.Duration
		want     string
	}{
		{0, "0s"},
		{-10 * time.Second, "0s"},
		{45 * time.Second, "0m 45s"},
		{125 * time.Second, "2m 5s"},
		{3665 * time.Second, "1h 1m 5s"},
		{90065 * time.Second, "1d 1h 1m"},
		{259200 * time.Second, "3d 0h 0m"},
	}

	for _, tt := range tests {
		info := host.Info{Uptime: tt.duration}
		if got := info.FormattedUptime(); got != tt.want {
			t.Errorf("FormattedUptime(%v) = %q, want %q", tt.duration, got, tt.want)
		}
	}
}

func TestRead(t *testing.T) {
	if _, err := os.Stat("/proc/uptime"); os.IsNotExist(err) {
		t.Skip("skipping on non-Linux host")
	}

	info, err := host.Read()
	if err != nil {
		t.Fatalf("host.Read() failed: %v", err)
	}

	if info.Hostname == "" {
		t.Error("expected non-empty Hostname")
	}
	if info.PrettyName == "" {
		t.Error("expected non-empty PrettyName")
	}
	if info.Uptime <= 0 {
		t.Error("expected positive Uptime")
	}
}
