// Package check defines the core health check interfaces, status codes, and execution engine.
package check

// Status represents the health status of a check.
type Status int

const (
	// StatusOK indicates the monitored metric is within healthy parameters.
	StatusOK Status = iota
	// StatusWarning indicates the metric exceeds normal levels but has not reached failure threshold.
	StatusWarning
	// StatusCritical indicates the metric exceeds safe operational limits and requires attention.
	StatusCritical
	// StatusUnknown indicates an error occurred while attempting to gather or evaluate the metric.
	StatusUnknown
)

// String implements the fmt.Stringer interface for human-readable output.
func (s Status) String() string {
	switch s {
	case StatusOK:
		return "OK"
	case StatusWarning:
		return "WARNING"
	case StatusCritical:
		return "CRITICAL"
	case StatusUnknown:
		return "UNKNOWN"
	default:
		return "UNKNOWN"
	}
}

// Severity returns an integer rank for comparing status levels.
// Higher values represent more severe conditions.
func (s Status) Severity() int {
	switch s {
	case StatusOK:
		return 0
	case StatusWarning:
		return 1
	case StatusUnknown:
		return 2
	case StatusCritical:
		return 3
	default:
		return 2
	}
}
