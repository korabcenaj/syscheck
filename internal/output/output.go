// Package output formats health check reports into human-readable text tables or JSON.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"text/tabwriter"
	"time"

	"github.com/korabcenaj/syscheck/internal/check"
)

// Report represents the complete evaluated health report of the system.
type Report struct {
	Timestamp time.Time      `json:"timestamp"`
	Hostname  string         `json:"hostname"`
	Overall   check.Status   `json:"overall"`
	Checks    []check.Result `json:"checks"`
}

// Formatter defines the contract for rendering reports to an io.Writer.
type Formatter interface {
	Format(w io.Writer, report Report) error
}

// TextFormatter formats reports as an aligned terminal table using text/tabwriter.
type TextFormatter struct{}

// Format writes the report as a formatted table to w.
func (f *TextFormatter) Format(w io.Writer, report Report) error {
	tw := tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)

	fmt.Fprintln(tw, "STATUS\tCHECK\tMESSAGE")
	for _, res := range report.Checks {
		fmt.Fprintf(tw, "%s\t%s\t%s\n", res.Status, res.Name, res.Message)
	}

	if err := tw.Flush(); err != nil {
		return fmt.Errorf("flushing tabwriter: %w", err)
	}

	fmt.Fprintf(w, "\nOverall Health: %s\n", report.Overall)
	return nil
}

// JSONFormatter formats reports as indented JSON.
type JSONFormatter struct{}

// Format writes the report as indented JSON to w.
func (f *JSONFormatter) Format(w io.Writer, report Report) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(report); err != nil {
		return fmt.Errorf("encoding json report: %w", err)
	}
	return nil
}

// NewFormatter returns a Formatter for the requested format name ("text" or "json").
func NewFormatter(format string) (Formatter, error) {
	switch format {
	case "text", "":
		return &TextFormatter{}, nil
	case "json":
		return &JSONFormatter{}, nil
	default:
		return nil, fmt.Errorf("unsupported output format %q: choose 'text' or 'json'", format)
	}
}
