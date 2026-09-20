// Package netcheck provides network reachability, TCP socket verification, and HTTP endpoint health checks.
package netcheck

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/korabcenaj/syscheck/internal/check"
)

// DialFunc represents a network dial function signature.
type DialFunc func(network, address string, timeout time.Duration) (net.Conn, error)

// TCPChecker checks whether a TCP endpoint at host:port is reachable and accepting connections.
type TCPChecker struct {
	Address  string
	Timeout  time.Duration
	DialFunc DialFunc
}

// NewTCPChecker creates a new TCPChecker for the specified address and timeout.
func NewTCPChecker(address string, timeout time.Duration) *TCPChecker {
	return &TCPChecker{
		Address: address,
		Timeout: timeout,
	}
}

// Name returns the descriptive name of the check.
func (c *TCPChecker) Name() string {
	return fmt.Sprintf("TCP (%s)", c.Address)
}

// Check attempts a TCP handshake with Address and reports the outcome.
func (c *TCPChecker) Check() check.Result {
	if c.Address == "" {
		return check.Result{
			Name:    c.Name(),
			Status:  check.StatusUnknown,
			Message: "TCP address cannot be empty",
			Err:     errors.New("tcp address cannot be empty"),
		}
	}

	// Validate host:port structure
	_, _, err := net.SplitHostPort(c.Address)
	if err != nil {
		return check.Result{
			Name:    c.Name(),
			Status:  check.StatusUnknown,
			Message: fmt.Sprintf("invalid address format %q: expected host:port", c.Address),
			Err:     err,
		}
	}

	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 2 * time.Second
	}

	dialFn := c.DialFunc
	if dialFn == nil {
		dialFn = net.DialTimeout
	}

	start := time.Now()
	conn, err := dialFn("tcp", c.Address, timeout)
	elapsed := time.Since(start).Round(time.Millisecond)

	if err != nil {
		return check.Result{
			Name:    c.Name(),
			Status:  check.StatusCritical,
			Message: fmt.Sprintf("connection to %s failed after %s: %v", c.Address, elapsed, err),
			Err:     err,
		}
	}
	_ = conn.Close()

	return check.Result{
		Name:    c.Name(),
		Status:  check.StatusOK,
		Message: fmt.Sprintf("connection to %s succeeded in %s", c.Address, elapsed),
	}
}
