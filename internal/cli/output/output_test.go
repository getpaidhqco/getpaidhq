package output_test

import (
	"bytes"
	"strings"
	"testing"
	"time"

	"getpaidhq/internal/cli/output"
)

func TestTableAligns(t *testing.T) {
	var buf bytes.Buffer
	if err := output.Table(&buf, []string{"ID", "EMAIL"}, [][]string{
		{"cus_1", "a@b.c"},
		{"cus_22", "long.email@example.com"},
	}); err != nil {
		t.Fatal(err)
	}
	want := "ID      EMAIL\ncus_1   a@b.c\ncus_22  long.email@example.com\n"
	if buf.String() != want {
		t.Fatalf("got:\n%q\nwant:\n%q", buf.String(), want)
	}
}

func TestTableSanitizesCells(t *testing.T) {
	var buf bytes.Buffer
	if err := output.Table(&buf, []string{"NAME"}, [][]string{
		{"a\tb\nc"},
	}); err != nil {
		t.Fatal(err)
	}
	got := buf.String()
	// The cell should have had its tab and newline replaced with spaces.
	const wantCell = "a b c"
	if !strings.Contains(got, wantCell) {
		t.Fatalf("expected cell %q in output %q", wantCell, got)
	}
}

func TestJSONPrettyPrints(t *testing.T) {
	var buf bytes.Buffer
	if err := output.JSON(&buf, []byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "{\n  \"a\": 1\n}\n" {
		t.Fatalf("%q", buf.String())
	}
}

func TestJSONPassthrough(t *testing.T) {
	var buf bytes.Buffer
	if err := output.JSON(&buf, []byte("bad gateway")); err != nil {
		t.Fatal(err)
	}
	if got := buf.String(); got != "bad gateway\n" {
		t.Fatalf("got %q", got)
	}
}

func TestJSONEmptyInput(t *testing.T) {
	var buf bytes.Buffer
	if err := output.JSON(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if buf.Len() != 0 {
		t.Fatalf("expected empty buffer, got %q", buf.String())
	}
}

func TestTimeAndStr(t *testing.T) {
	if got := output.Time(time.Time{}); got != "-" {
		t.Fatal(got)
	}
	ts := time.Date(2026, 6, 12, 9, 30, 0, 0, time.UTC)
	if got := output.Time(ts); got != "2026-06-12 09:30" {
		t.Fatal(got)
	}
	if output.Str("") != "-" || output.Str("x") != "x" {
		t.Fatal("Str")
	}
}
