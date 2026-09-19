// Package netcheck provides network reachability and port health verification.
package netcheck

import (
	"fmt"
	"net"
	"time"

	"github.com/korabcenaj/syscheck/internal/check"
)

// ReachabilityChecker checks whether a host is reachable over the network on a specific TCP port.
type ReachabilityChecker struct {
	Target  string
	Port    int
	Timeout time.Duration

	// DialFunc allows injecting custom dial functions during testing.
	DialFunc func(network, address string, timeout time.Duration) (net.Conn, error)
}

// Name returns the descriptive name of the check.
func (c *ReachabilityChecker) Name() string {
	port := c.Port
	if port <= 0 {
		port = 22 // Default to standard SSH port for server administration
	}
	return fmt.Sprintf("Reachability (%s:%d)", c.Target, port)
}

// Check attempts a TCP connection to the target host and evaluates its reachability.
func (c *ReachabilityChecker) Check() check.Result {
	port := c.Port
	if port <= 0 {
		port = 22
	}

	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	dialFn := c.DialFunc
	if dialFn == nil {
		dialFn = net.DialTimeout
	}

	addr := net.JoinHostPort(c.Target, fmt.Sprintf("%d", port))
	start := time.Now()
	conn, err := dialFn("tcp", addr, timeout)
	elapsed := time.Since(start).Round(time.Millisecond)

	if err != nil {
		return check.Result{
			Name:    c.Name(),
			Status:  check.StatusCritical,
			Message: fmt.Sprintf("target %s unreachable on TCP port %d: %v", c.Target, port, err),
			Err:     err,
		}
	}
	defer conn.Close()

	return check.Result{
		Name:    c.Name(),
		Status:  check.StatusOK,
		Message: fmt.Sprintf("host reachable (TCP handshake in %s)", elapsed),
	}
}
