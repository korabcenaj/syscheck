package load_test

import (
	"os"
	"path/filepath"
	"testing"

	"syscheck/internal/load"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    load.LoadAvg
		wantErr bool
	}{
		{
			name:  "standard linux /proc/loadavg line",
			input: "0.15 0.20 0.18 1/824 12345\n",
			want: load.LoadAvg{
				One:     0.15,
				Five:    0.20,
				Fifteen: 0.18,
			},
			wantErr: false,
		},
		{
			name:  "high load values with extra whitespace",
			input: "  24.50   18.25   12.10   8/1024 99999 \n",
			want: load.LoadAvg{
				One:     24.50,
				Five:    18.25,
				Fifteen: 12.10,
			},
			wantErr: false,
		},
		{
			name:    "empty input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "insufficient fields",
			input:   "0.15 0.20",
			wantErr: true,
		},
		{
			name:    "non-numeric 1m value",
			input:   "invalid 0.20 0.18 1/824 12345",
			wantErr: true,
		},
		{
			name:    "non-numeric 5m value",
			input:   "0.15 invalid 0.18 1/824 12345",
			wantErr: true,
		},
		{
			name:    "non-numeric 15m value",
			input:   "0.15 0.20 invalid 1/824 12345",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := load.Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.want {
				t.Errorf("Parse() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestReadFile(t *testing.T) {
	t.Run("reads valid file successfully", func(t *testing.T) {
		tempDir := t.TempDir()
		dummyPath := filepath.Join(tempDir, "loadavg")

		content := "1.25 2.50 3.75 2/450 6789\n"
		if err := os.WriteFile(dummyPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to create temporary test file: %v", err)
		}

		got, err := load.ReadFile(dummyPath)
		if err != nil {
			t.Fatalf("ReadFile() returned unexpected error: %v", err)
		}

		want := load.LoadAvg{
			One:     1.25,
			Five:    2.50,
			Fifteen: 3.75,
		}

		if got != want {
			t.Errorf("ReadFile() = %+v, want %+v", got, want)
		}
	})

	t.Run("returns error on missing file", func(t *testing.T) {
		_, err := load.ReadFile("/path/to/nonexistent/file/loadavg")
		if err == nil {
			t.Fatal("ReadFile() expected error for nonexistent file, got nil")
		}
	})
}

func TestRead(t *testing.T) {
	// If running on a Linux system where /proc/loadavg is accessible,
	// verify that Read() returns valid, non-negative numbers.
	if _, err := os.Stat(load.DefaultLoadAvgPath); os.IsNotExist(err) {
		t.Skipf("%s does not exist on this host, skipping live test", load.DefaultLoadAvgPath)
	}

	got, err := load.Read()
	if err != nil {
		t.Fatalf("Read() failed on live system: %v", err)
	}

	if got.One < 0 || got.Five < 0 || got.Fifteen < 0 {
		t.Errorf("Read() returned negative load averages: %+v", got)
	}
}
