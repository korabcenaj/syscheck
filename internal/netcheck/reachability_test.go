package netcheck_test

import (
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/korabcenaj/syscheck/internal/check"
	"github.com/korabcenaj/syscheck/internal/netcheck"
)

func TestReachabilityChecker(t *testing.T) {
	t.Run("returns OK when dial succeeds", func(t *testing.T) {
		client, server := net.Pipe()
		defer client.Close()
		defer server.Close()

		c := &netcheck.ReachabilityChecker{
			Target: "server1.internal",
			Port:   22,
			DialFunc: func(network, address string, timeout time.Duration) (net.Conn, error) {
				return client, nil
			},
		}

		res := c.Check()
		if res.Status != check.StatusOK {
			t.Errorf("Status = %v, want OK", res.Status)
		}
		if !strings.Contains(res.Message, "host reachable") {
			t.Errorf("Message = %q, expected 'host reachable'", res.Message)
		}
	})

	t.Run("returns Critical when dial fails", func(t *testing.T) {
		c := &netcheck.ReachabilityChecker{
			Target: "offline-server.internal",
			Port:   22,
			DialFunc: func(network, address string, timeout time.Duration) (net.Conn, error) {
				return nil, errors.New("connection refused")
			},
		}

		res := c.Check()
		if res.Status != check.StatusCritical {
			t.Errorf("Status = %v, want Critical", res.Status)
		}
		if res.Err == nil {
			t.Error("expected non-nil Err on failure")
		}
		if !strings.Contains(res.Message, "unreachable") {
			t.Errorf("Message = %q, expected 'unreachable'", res.Message)
		}
	})

	t.Run("connects to live local listener", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to open test listener: %v", err)
		}
		defer ln.Close()

		tcpAddr := ln.Addr().(*net.TCPAddr)

		c := &netcheck.ReachabilityChecker{
			Target:  "127.0.0.1",
			Port:    tcpAddr.Port,
			Timeout: 1 * time.Second,
		}

		res := c.Check()
		if res.Status != check.StatusOK {
			t.Fatalf("live dial failed: %v (%s)", res.Status, res.Message)
		}
	})
}
