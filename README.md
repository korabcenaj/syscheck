# syscheck

`syscheck` is a lightweight, dependency-free Linux server health-check CLI tool written in Go.

## Overview

Designed for systems administrators, site reliability engineers, and infrastructure operators, `syscheck` inspects critical system metrics directly via Linux kernel interfaces (such as `procfs`) and system calls without requiring heavy runtimes or external dependencies.

## Project Structure

```
syscheck/
├── cmd/
│   └── syscheck/
│       └── main.go          # Application entrypoint & CLI exit code handling
├── internal/
│   └── load/
│       ├── load.go          # /proc/loadavg parsing & domain models
│       └── load_test.go     # Table-driven unit and integration tests
├── bin/                     # Compiled binaries (ignored by git)
├── .gitignore
├── Makefile                 # Standard developer targets (build, test, run)
├── go.mod                   # Go module definition
└── README.md
```

## Quickstart

### Prerequisites

- Go 1.22+ (Linux environment)

### Building and Testing

```bash
# Run unit and race tests
make test

# Build binary into bin/syscheck
make build

# Build and execute
make run
```
