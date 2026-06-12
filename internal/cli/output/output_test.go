package output_test

import (
	"bytes"
	"testing"
	"time"

	"getpaidhq/internal/cli/output"
)

func TestTableAligns(t *testing.T) {
	var buf bytes.Buffer
	output.Table(&buf, []string{"ID", "EMAIL"}, [][]string{
		{"cus_1", "a@b.c"},
		{"cus_22", "long.email@example.com"},
	})
	want := "ID      EMAIL\ncus_1   a@b.c\ncus_22  long.email@example.com\n"
	if buf.String() != want {
		t.Fatalf("got:\n%q\nwant:\n%q", buf.String(), want)
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
