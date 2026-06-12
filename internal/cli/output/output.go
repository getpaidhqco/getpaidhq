// Package output renders API responses as aligned tables or pretty JSON.
package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"
)

func Table(w io.Writer, headers []string, rows [][]string) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	fmt.Fprintln(tw, strings.Join(headers, "\t"))
	for _, r := range rows {
		fmt.Fprintln(tw, strings.Join(r, "\t"))
	}
	tw.Flush()
}

// JSON pretty-prints a raw API response body.
func JSON(w io.Writer, raw []byte) error {
	var buf bytes.Buffer
	if err := json.Indent(&buf, raw, "", "  "); err != nil {
		buf.Reset()
		buf.Write(raw) // not JSON (e.g. empty 204 body): pass through
	}
	if buf.Len() == 0 {
		return nil
	}
	buf.WriteByte('\n')
	_, err := w.Write(buf.Bytes())
	return err
}

// Time formats timestamps for table cells; zero values render as "-".
func Time(t time.Time) string {
	if t.IsZero() {
		return "-"
	}
	return t.Format("2006-01-02 15:04")
}

// Str substitutes "-" for empty strings in table cells.
func Str(s string) string {
	if s == "" {
		return "-"
	}
	return s
}
