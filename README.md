# syscheck

`syscheck` is a lightweight, zero-dependency Linux server health-check CLI tool written in modern, idiomatic Go.

## Overview

Designed for systems administrators, site reliability engineers (SREs), and infrastructure operators, `syscheck` inspects core Linux system health directly through kernel interfaces (`procfs`) and POSIX system calls (`statfs`) without shelling out to external utilities or requiring heavy language runtimes.

It evaluates system states against operational thresholds, conforms to Unix/Nagios monitoring exit code conventions, and supports both human-readable aligned tables and machine-readable JSON.

## Key Features

- **Kernel & Syscall Integration**:
  - **Load Average**: Parsed directly from `/proc/loadavg` (1m, 5m, 15m).
  - **Memory & Swap**: Parsed line-by-line from `/proc/meminfo` using buffered streaming I/O (`bufio.Scanner`), with automated fallback for legacy kernels.
  - **Disk Space & Inodes**: Direct kernel POSIX system calls (`syscall.Statfs`) distinguishing between root and unprivileged user block reservations (`Bavail` vs `Bfree`), with dynamic detection for dynamic inode filesystems (e.g. Btrfs, ZFS).
- **Interface-Driven Health Engine**:
  - Polymorphic `Checker` interface and centralized `Runner`.
  - Severity-based health aggregation (`CRITICAL` > `UNKNOWN` > `WARNING` > `OK`).
- **Standard Monitoring Exit Codes**:
  - `0` — OK
  - `1` — WARNING
  - `2` — CRITICAL
  - `3` — UNKNOWN / CLI Error
- **Dual Output Formats**:
  - `text`: Aligned columnar table formatted via `text/tabwriter`.
  - `json`: Structured JSON with ISO 8601 timestamps and host metadata, ready for `jq`, Datadog, Vector, or FluentBit.
- **Zero External Dependencies**: Implemented entirely with the Go standard library.
- **Comprehensive Test Suite**: Table-driven unit tests, mock dependency injection, and race detection across all packages.

## Project Structure

```
syscheck/
├── cmd/
│   └── syscheck/
│       ├── main.go          # Application entrypoint & testable run() engine
│       └── main_test.go     # End-to-end CLI lifecycle & exit code tests
├── internal/
│   ├── check/
│   │   ├── check.go         # Core Checker interface, Result, and Runner
│   │   ├── check_test.go    # Aggregation, severity ranking, and threshold tests
│   │   ├── checkers.go      # Concrete LoadChecker, MemoryChecker, DiskChecker
│   │   └── status.go        # Status enum (iota), Stringer, and JSON marshaling
│   ├── config/
│   │   ├── config.go        # Isolated flag.FlagSet parser and validation
│   │   └── config_test.go   # CLI argument parsing and boundary tests
│   ├── disk/
│   │   ├── disk.go          # POSIX syscall.Statfs filesystem monitoring
│   │   └── disk_test.go     # Block & inode calculations and Btrfs handling
│   ├── load/
│   │   ├── load.go          # /proc/loadavg parser and domain models
│   │   └── load_test.go     # Table-driven parser tests
│   ├── memory/
│   │   ├── memory.go        # /proc/meminfo streaming parser and swap checks
│   │   └── memory_test.go   # In-memory reader tests and legacy kernel fallbacks
│   └── output/
│       ├── output.go        # Formatter interface (TextFormatter, JSONFormatter)
│       └── output_test.go   # Tabwriter and JSON encoding tests
├── Makefile                 # Standard developer workflow targets
├── go.mod                   # Go module definition
└── README.md
```

## Quickstart

### Prerequisites

- Go 1.22+ (Linux environment)

### Building and Testing

```bash
# Run all unit tests with race detection
make test

# Build binary into bin/syscheck
make build

# Build and execute with default settings
make run
```

## Usage and Examples

### Default Aligned Table Output

```bash
$ ./bin/syscheck
STATUS  CHECK         MESSAGE
OK      Load Average  normal load: 0.36 (1m), 0.19 (5m), 0.15 (15m)
OK      Memory        normal memory usage: 3.07 GiB / 12.70 GiB (24.2% used)
OK      Disk (/)      normal disk usage: 47.05 GiB / 86.48 GiB (54.4% used) [Inodes: dynamic]

Overall Health: OK
```

### JSON Output with `jq`

```bash
$ ./bin/syscheck -format json | jq .
{
  "timestamp": "2026-09-19T20:12:15.006617754Z",
  "hostname": "server-01",
  "overall": "OK",
  "checks": [
    {
      "name": "Load Average",
      "status": "OK",
      "message": "normal load: 0.28 (1m), 0.18 (5m), 0.15 (15m)"
    },
    {
      "name": "Memory",
      "status": "OK",
      "message": "normal memory usage: 3.06 GiB / 12.70 GiB (24.1% used)"
    },
    {
      "name": "Disk (/)",
      "status": "OK",
      "message": "normal disk usage: 47.05 GiB / 86.48 GiB (54.4% used) [Inodes: dynamic]"
    }
  ]
}
```

### Custom Thresholds & Exit Codes

```bash
# Set custom memory and disk thresholds, and check /var
$ ./bin/syscheck -mem-warn 70 -mem-crit 85 -disk-path /var

# Inspect process exit code in scripts
$ ./bin/syscheck -load-crit 0.1
[CRITICAL] Load Average critical load: 1m=0.35 (>= 0.10 threshold)
...
$ echo $?
2
```

### Available Command-Line Flags

| Flag | Type | Default | Description |
| :--- | :--- | :--- | :--- |
| `-format` | string | `text` | Output format: `text` or `json` |
| `-load-warn` | float | `2.0` | 1-minute load average warning threshold |
| `-load-crit` | float | `5.0` | 1-minute load average critical threshold |
| `-mem-warn` | float | `80.0` | Memory percentage warning threshold (0-100) |
| `-mem-crit` | float | `90.0` | Memory percentage critical threshold (0-100) |
| `-disk-warn` | float | `80.0` | Disk percentage warning threshold (0-100) |
| `-disk-crit` | float | `90.0` | Disk percentage critical threshold (0-100) |
| `-disk-path` | string | `/` | Filesystem path to monitor for space and inodes |
