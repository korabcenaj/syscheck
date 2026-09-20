package netcheck

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/korabcenaj/syscheck/internal/check"
)

// HTTPDoer defines the interface for executing an HTTP request.
// Standard *http.Client satisfies this interface.
type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

// HTTPChecker checks the availability and response status code of an HTTP/HTTPS endpoint.
type HTTPChecker struct {
	URL            string
	Method         string
	ExpectedStatus int
	Timeout        time.Duration
	Client         HTTPDoer
}

// NewHTTPChecker creates an HTTPChecker for the specified URL and timeout with default GET and 200 OK expectation.
func NewHTTPChecker(endpointURL string, timeout time.Duration) *HTTPChecker {
	return &HTTPChecker{
		URL:            endpointURL,
		Method:         http.MethodGet,
		ExpectedStatus: http.StatusOK,
		Timeout:        timeout,
	}
}

// Name returns the descriptive name of the check.
func (c *HTTPChecker) Name() string {
	return fmt.Sprintf("HTTP (%s)", c.URL)
}

// Check issues an HTTP request to URL and validates the returned HTTP status code.
func (c *HTTPChecker) Check() check.Result {
	if strings.TrimSpace(c.URL) == "" {
		return check.Result{
			Name:    c.Name(),
			Status:  check.StatusUnknown,
			Message: "HTTP check URL cannot be empty",
			Err:     errors.New("empty URL"),
		}
	}

	parsed, err := url.Parse(c.URL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return check.Result{
			Name:    c.Name(),
			Status:  check.StatusUnknown,
			Message: fmt.Sprintf("invalid URL %q: scheme and host required", c.URL),
			Err:     fmt.Errorf("invalid URL: %q", c.URL),
		}
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return check.Result{
			Name:    c.Name(),
			Status:  check.StatusUnknown,
			Message: fmt.Sprintf("unsupported URL scheme %q in %q: must be http or https", parsed.Scheme, c.URL),
			Err:     fmt.Errorf("unsupported URL scheme: %q", parsed.Scheme),
		}
	}

	method := c.Method
	if method == "" {
		method = http.MethodGet
	}

	timeout := c.Timeout
	if timeout <= 0 {
		timeout = 3 * time.Second
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, method, c.URL, nil)
	if err != nil {
		return check.Result{
			Name:    c.Name(),
			Status:  check.StatusUnknown,
			Message: fmt.Sprintf("failed to create request: %v", err),
			Err:     err,
		}
	}

	client := c.Client
	if client == nil {
		client = &http.Client{
			Timeout: timeout,
		}
	}

	start := time.Now()
	resp, err := client.Do(req)
	elapsed := time.Since(start).Round(time.Millisecond)

	if err != nil {
		return check.Result{
			Name:    c.Name(),
			Status:  check.StatusCritical,
			Message: fmt.Sprintf("HTTP %s %s failed: %v", method, c.URL, err),
			Err:     err,
		}
	}
	defer resp.Body.Close()
	// Drain response body up to 4KB to allow persistent connection reuse
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	expected := c.ExpectedStatus
	if expected > 0 {
		if resp.StatusCode == expected {
			return check.Result{
				Name:    c.Name(),
				Status:  check.StatusOK,
				Message: fmt.Sprintf("HTTP %s %s returned %d %s in %s", method, c.URL, resp.StatusCode, http.StatusText(resp.StatusCode), elapsed),
			}
		}
		return check.Result{
			Name:    c.Name(),
			Status:  check.StatusCritical,
			Message: fmt.Sprintf("HTTP %s %s returned %d %s (expected %d) in %s", method, c.URL, resp.StatusCode, http.StatusText(resp.StatusCode), expected, elapsed),
		}
	}

	// If ExpectedStatus is 0, any 2xx status code is considered OK
	switch {
	case resp.StatusCode >= 200 && resp.StatusCode < 300:
		return check.Result{
			Name:    c.Name(),
			Status:  check.StatusOK,
			Message: fmt.Sprintf("HTTP %s %s returned %d %s in %s", method, c.URL, resp.StatusCode, http.StatusText(resp.StatusCode), elapsed),
		}
	case resp.StatusCode >= 300 && resp.StatusCode < 400:
		return check.Result{
			Name:    c.Name(),
			Status:  check.StatusWarning,
			Message: fmt.Sprintf("HTTP %s %s returned redirect %d %s in %s", method, c.URL, resp.StatusCode, http.StatusText(resp.StatusCode), elapsed),
		}
	default:
		return check.Result{
			Name:    c.Name(),
			Status:  check.StatusCritical,
			Message: fmt.Sprintf("HTTP %s %s returned %d %s in %s", method, c.URL, resp.StatusCode, http.StatusText(resp.StatusCode), elapsed),
		}
	}
}
