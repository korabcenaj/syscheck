package check

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/korabcenaj/syscheck/internal/disk"
	"github.com/korabcenaj/syscheck/internal/load"
	"github.com/korabcenaj/syscheck/internal/memory"
	"github.com/korabcenaj/syscheck/internal/process"
)

// LoadChecker evaluates system load averages against thresholds.
type LoadChecker struct {
	WarnThreshold float64
	CritThreshold float64

	// ReadFunc allows overriding load.Read for testing.
	ReadFunc func() (load.LoadAvg, error)
}

// Name returns the descriptive name of the check.
func (c *LoadChecker) Name() string {
	return "Load Average"
}

// Check evaluates the 1-minute load average against warning and critical thresholds.
func (c *LoadChecker) Check() Result {
	readFn := c.ReadFunc
	if readFn == nil {
		readFn = load.Read
	}

	avg, err := readFn()
	if err != nil {
		return Result{
			Name:    c.Name(),
			Status:  StatusUnknown,
			Message: fmt.Sprintf("failed to read load: %v", err),
			Err:     err,
		}
	}

	switch {
	case c.CritThreshold > 0 && avg.One >= c.CritThreshold:
		return Result{
			Name:   c.Name(),
			Status: StatusCritical,
			Message: fmt.Sprintf("critical load: 1m=%.2f (>= %.2f threshold) [5m=%.2f, 15m=%.2f]",
				avg.One, c.CritThreshold, avg.Five, avg.Fifteen),
		}
	case c.WarnThreshold > 0 && avg.One >= c.WarnThreshold:
		return Result{
			Name:   c.Name(),
			Status: StatusWarning,
			Message: fmt.Sprintf("elevated load: 1m=%.2f (>= %.2f threshold) [5m=%.2f, 15m=%.2f]",
				avg.One, c.WarnThreshold, avg.Five, avg.Fifteen),
		}
	default:
		return Result{
			Name:   c.Name(),
			Status: StatusOK,
			Message: fmt.Sprintf("normal load: %.2f (1m), %.2f (5m), %.2f (15m)",
				avg.One, avg.Five, avg.Fifteen),
		}
	}
}

// MemoryChecker evaluates physical RAM utilization against percentage thresholds.
type MemoryChecker struct {
	WarnPercent float64
	CritPercent float64

	// ReadFunc allows overriding memory.Read for testing.
	ReadFunc func() (memory.Stats, error)
}

// Name returns the descriptive name of the check.
func (c *MemoryChecker) Name() string {
	return "Memory"
}

// Check evaluates memory usage percentage against warning and critical thresholds.
func (c *MemoryChecker) Check() Result {
	readFn := c.ReadFunc
	if readFn == nil {
		readFn = memory.Read
	}

	stats, err := readFn()
	if err != nil {
		return Result{
			Name:    c.Name(),
			Status:  StatusUnknown,
			Message: fmt.Sprintf("failed to read memory info: %v", err),
			Err:     err,
		}
	}

	usedPct := stats.UsedPercent()
	usageSummary := fmt.Sprintf("%s / %s (%.1f%% used)",
		FormatBytes(stats.UsedBytes()),
		FormatBytes(stats.TotalBytes),
		usedPct)

	switch {
	case c.CritPercent > 0 && usedPct >= c.CritPercent:
		return Result{
			Name:    c.Name(),
			Status:  StatusCritical,
			Message: fmt.Sprintf("critical memory usage: %s", usageSummary),
		}
	case c.WarnPercent > 0 && usedPct >= c.WarnPercent:
		return Result{
			Name:    c.Name(),
			Status:  StatusWarning,
			Message: fmt.Sprintf("elevated memory usage: %s", usageSummary),
		}
	default:
		return Result{
			Name:    c.Name(),
			Status:  StatusOK,
			Message: fmt.Sprintf("normal memory usage: %s", usageSummary),
		}
	}
}

// DiskChecker evaluates filesystem space and inode utilization for a mount path.
type DiskChecker struct {
	Path        string
	WarnPercent float64
	CritPercent float64

	// CheckFunc allows overriding disk.Check for testing.
	CheckFunc func(path string) (disk.Usage, error)
}

// Name returns the descriptive name of the check including the checked path.
func (c *DiskChecker) Name() string {
	path := c.Path
	if path == "" {
		path = "/"
	}
	return fmt.Sprintf("Disk (%s)", path)
}

// Check evaluates disk space and inode percentages against thresholds.
func (c *DiskChecker) Check() Result {
	checkFn := c.CheckFunc
	if checkFn == nil {
		checkFn = disk.Check
	}

	path := c.Path
	if path == "" {
		path = "/"
	}

	usage, err := checkFn(path)
	if err != nil {
		return Result{
			Name:    c.Name(),
			Status:  StatusUnknown,
			Message: fmt.Sprintf("failed to stat filesystem: %v", err),
			Err:     err,
		}
	}

	usedPct := usage.UsedPercent()
	inodePct := usage.InodesUsedPercent()

	var inodeInfo string
	if usage.HasInodes() {
		inodeInfo = fmt.Sprintf("[Inodes: %.1f%% used]", inodePct)
	} else {
		inodeInfo = "[Inodes: dynamic]"
	}

	usageSummary := fmt.Sprintf("%s / %s (%.1f%% used) %s",
		FormatBytes(usage.UsedBytes),
		FormatBytes(usage.TotalBytes),
		usedPct,
		inodeInfo)

	// Check critical threshold on either space or inodes
	isCritSpace := c.CritPercent > 0 && usedPct >= c.CritPercent
	isCritInode := c.CritPercent > 0 && usage.HasInodes() && inodePct >= c.CritPercent

	if isCritSpace || isCritInode {
		return Result{
			Name:    c.Name(),
			Status:  StatusCritical,
			Message: fmt.Sprintf("critical disk usage: %s", usageSummary),
		}
	}

	// Check warning threshold on either space or inodes
	isWarnSpace := c.WarnPercent > 0 && usedPct >= c.WarnPercent
	isWarnInode := c.WarnPercent > 0 && usage.HasInodes() && inodePct >= c.WarnPercent

	if isWarnSpace || isWarnInode {
		return Result{
			Name:    c.Name(),
			Status:  StatusWarning,
			Message: fmt.Sprintf("elevated disk usage: %s", usageSummary),
		}
	}

	return Result{
		Name:    c.Name(),
		Status:  StatusOK,
		Message: fmt.Sprintf("normal disk usage: %s", usageSummary),
	}
}

// FormatBytes formats a byte count into binary units (KiB, MiB, GiB, TiB).
func FormatBytes(b uint64) string {
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

// ProcessChecker verifies that a required process is currently running.
type ProcessChecker struct {
	ProcessName string
	MinCount    int
	ProcPath    string

	// FindFunc allows overriding process search for testing.
	FindFunc func(name string, procPath string) ([]process.Process, error)
}

// Name returns the descriptive name of the check.
func (c *ProcessChecker) Name() string {
	return fmt.Sprintf("Process (%s)", c.ProcessName)
}

// Check evaluates whether the process is running and meets the minimum count requirement.
func (c *ProcessChecker) Check() Result {
	min := c.MinCount
	if min <= 0 {
		min = 1
	}

	findFn := c.FindFunc
	if findFn == nil {
		findFn = process.FindByName
	}

	procs, err := findFn(c.ProcessName, c.ProcPath)
	if err != nil {
		return Result{
			Name:    c.Name(),
			Status:  StatusUnknown,
			Message: fmt.Sprintf("failed to scan processes: %v", err),
			Err:     err,
		}
	}

	count := len(procs)
	if count == 0 {
		return Result{
			Name:    c.Name(),
			Status:  StatusCritical,
			Message: fmt.Sprintf("process %q is not running (expected at least %d)", c.ProcessName, min),
		}
	}

	if count < min {
		return Result{
			Name:    c.Name(),
			Status:  StatusWarning,
			Message: fmt.Sprintf("process %q has %d instance(s) running (expected at least %d)", c.ProcessName, count, min),
		}
	}

	pidStrs := make([]string, 0, count)
	for _, p := range procs {
		pidStrs = append(pidStrs, strconv.Itoa(p.PID))
		if len(pidStrs) >= 3 {
			break
		}
	}
	pidDisplay := strings.Join(pidStrs, ", ")
	if count > 3 {
		pidDisplay += fmt.Sprintf(", +%d more", count-3)
	}

	return Result{
		Name:    c.Name(),
		Status:  StatusOK,
		Message: fmt.Sprintf("process %q is running (%d instance(s), PID: %s)", c.ProcessName, count, pidDisplay),
	}
}

// ServiceChecker verifies that a system service is currently active.
type ServiceChecker struct {
	ServiceName string

	// QueryFunc allows overriding service state queries for testing.
	QueryFunc process.ServiceQueryFunc
}

// Name returns the descriptive name of the check.
func (c *ServiceChecker) Name() string {
	return fmt.Sprintf("Service (%s)", c.ServiceName)
}

// Check evaluates whether the service is in an active state.
func (c *ServiceChecker) Check() Result {
	queryFn := c.QueryFunc
	if queryFn == nil {
		queryFn = process.QueryService
	}

	state, err := queryFn(c.ServiceName)
	if err != nil && state.State == "unknown" {
		return Result{
			Name:    c.Name(),
			Status:  StatusUnknown,
			Message: fmt.Sprintf("failed to query service %q: %v", c.ServiceName, err),
			Err:     err,
		}
	}

	if state.Active {
		return Result{
			Name:    c.Name(),
			Status:  StatusOK,
			Message: fmt.Sprintf("service %q is active (state: %s)", c.ServiceName, state.State),
		}
	}

	if state.State == "failed" {
		return Result{
			Name:    c.Name(),
			Status:  StatusCritical,
			Message: fmt.Sprintf("service %q is in failed state", c.ServiceName),
		}
	}

	return Result{
		Name:    c.Name(),
		Status:  StatusCritical,
		Message: fmt.Sprintf("service %q is not active (state: %s)", c.ServiceName, state.State),
	}
}

