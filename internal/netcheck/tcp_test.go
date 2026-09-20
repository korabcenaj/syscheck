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

func TestTCPChecker(t *testing.T) {
	t.Run("returns OK when connection succeeds", func(t *testing.T) {
		client, server := net.Pipe()
		defer client.Close()
		defer server.Close()

		c := &netcheck.TCPChecker{
			Address: "127.0.0.1:5432",
			DialFunc: func(network, address string, timeout time.Duration) (net.Conn, error) {
				return client, nil
			},
		}

		res := c.Check()
		if res.Status != check.StatusOK {
			t.Errorf("Status = %v, want OK", res.Status)
		}
		if res.Name != "TCP (127.0.0.1:5432)" {
			t.Errorf("Name = %q, want 'TCP (127.0.0.1:5432)'", res.Name)
		}
		if !strings.Contains(res.Message, "succeeded") {
			t.Errorf("Message missing 'succeeded': %q", res.Message)
		}
	})

	t.Run("returns Critical when connection is refused or times out", func(t *testing.T) {
		c := &netcheck.TCPChecker{
			Address: "10.255.255.1:9999",
			DialFunc: func(network, address string, timeout time.Duration) (net.Conn, error) {
				return nil, errors.New("connection refused")
			},
		}

		res := c.Check()
		if res.Status != check.StatusCritical {
			t.Errorf("Status = %v, want Critical", res.Status)
		}
		if res.Err == nil {
			t.Error("expected non-nil Err")
		}
		if !strings.Contains(res.Message, "failed") {
			t.Errorf("Message missing 'failed': %q", res.Message)
		}
	})

	t.Run("returns Unknown for empty address", func(t *testing.T) {
		c := &netcheck.TCPChecker{
			Address: "",
		}

		res := c.Check()
		if res.Status != check.StatusUnknown {
			t.Errorf("Status = %v, want Unknown", res.Status)
		}
	})

	t.Run("returns Unknown for malformed address missing port", func(t *testing.T) {
		c := &netcheck.TCPChecker{
			Address: "localhost",
		}

		res := c.Check()
		if res.Status != check.StatusUnknown {
			t.Errorf("Status = %v, want Unknown", res.Status)
		}
		if !strings.Contains(res.Message, "expected host:port") {
			t.Errorf("Message missing expected host:port note: %q", res.Message)
		}
	})

	t.Run("connects to live local listener with NewTCPChecker", func(t *testing.T) {
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to listen: %v", err)
		}
		defer ln.Close()

		c := netcheck.NewTCPChecker(ln.Addr().String(), 1*time.Second)
		res := c.Check()
		if res.Status != check.StatusOK {
			t.Errorf("Status = %v, want OK. Message: %s", res.Status, res.Message)
		}
	})
}
