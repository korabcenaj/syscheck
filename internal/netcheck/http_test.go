package netcheck_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/korabcenaj/syscheck/internal/check"
	"github.com/korabcenaj/syscheck/internal/netcheck"
)

type mockHTTPClient struct {
	doFunc func(req *http.Request) (*http.Response, error)
}

func (m *mockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	return m.doFunc(req)
}

func TestHTTPChecker(t *testing.T) {
	t.Run("returns OK for 200 OK with default settings", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"status":"up"}`))
		}))
		defer ts.Close()

		c := netcheck.NewHTTPChecker(ts.URL, 1*time.Second)
		res := c.Check()

		if res.Status != check.StatusOK {
			t.Errorf("Status = %v, want OK. Msg: %s", res.Status, res.Message)
		}
		if res.Name != "HTTP ("+ts.URL+")" {
			t.Errorf("Name = %q, want %q", res.Name, "HTTP ("+ts.URL+")")
		}
		if !strings.Contains(res.Message, "200 OK") {
			t.Errorf("Message missing '200 OK': %q", res.Message)
		}
	})

	t.Run("returns Critical when server returns 500 error", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer ts.Close()

		c := netcheck.NewHTTPChecker(ts.URL, 1*time.Second)
		res := c.Check()

		if res.Status != check.StatusCritical {
			t.Errorf("Status = %v, want Critical. Msg: %s", res.Status, res.Message)
		}
		if !strings.Contains(res.Message, "500") {
			t.Errorf("Message missing '500': %q", res.Message)
		}
	})

	t.Run("matches custom ExpectedStatus", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent) // 204
		}))
		defer ts.Close()

		c := &netcheck.HTTPChecker{
			URL:            ts.URL,
			ExpectedStatus: http.StatusNoContent,
		}
		res := c.Check()

		if res.Status != check.StatusOK {
			t.Errorf("Status = %v, want OK. Msg: %s", res.Status, res.Message)
		}
	})

	t.Run("returns Critical when response code does not match ExpectedStatus", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK) // 200
		}))
		defer ts.Close()

		c := &netcheck.HTTPChecker{
			URL:            ts.URL,
			ExpectedStatus: http.StatusCreated, // 201 expected
		}
		res := c.Check()

		if res.Status != check.StatusCritical {
			t.Errorf("Status = %v, want Critical. Msg: %s", res.Status, res.Message)
		}
		if !strings.Contains(res.Message, "expected 201") {
			t.Errorf("Message missing expected 201: %q", res.Message)
		}
	})

	t.Run("returns Warning on 302 redirect when ExpectedStatus is 0", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "/login", http.StatusFound)
		}))
		defer ts.Close()

		// Use custom Transport that does not follow redirects
		client := &http.Client{
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		c := &netcheck.HTTPChecker{
			URL:            ts.URL,
			ExpectedStatus: 0,
			Client:         client,
		}
		res := c.Check()

		if res.Status != check.StatusWarning {
			t.Errorf("Status = %v, want Warning. Msg: %s", res.Status, res.Message)
		}
	})

	t.Run("returns Critical on network transport failure", func(t *testing.T) {
		c := &netcheck.HTTPChecker{
			URL: "http://127.0.0.1:59999/healthz",
			Client: &mockHTTPClient{
				doFunc: func(req *http.Request) (*http.Response, error) {
					return nil, errors.New("connection refused")
				},
			},
		}

		res := c.Check()
		if res.Status != check.StatusCritical {
			t.Errorf("Status = %v, want Critical", res.Status)
		}
		if res.Err == nil {
			t.Error("expected non-nil Err")
		}
	})

	t.Run("returns Unknown for invalid or empty URL", func(t *testing.T) {
		invalidURLs := []string{"", "   ", "not-a-url", "ftp://example.com/file"}
		for _, u := range invalidURLs {
			c := &netcheck.HTTPChecker{URL: u}
			res := c.Check()
			if res.Status != check.StatusUnknown {
				t.Errorf("URL %q: Status = %v, want Unknown", u, res.Status)
			}
		}
	})
}
