package cli_test

import (
	"bytes"
	"strings"
	"testing"

	"getpaidhq/internal/cli"
)

func run(t *testing.T, args ...string) (code int, out, errOut string) {
	t.Helper()
	var o, e bytes.Buffer
	code = cli.Run(args, strings.NewReader(""), &o, &e)
	return code, o.String(), e.String()
}

func TestVersion(t *testing.T) {
	code, out, _ := run(t, "version")
	if code != 0 || !strings.Contains(out, "gphq version") {
		t.Fatalf("code=%d out=%q", code, out)
	}
}

func TestUnknownFlagIsUsageError(t *testing.T) {
	if code, _, _ := run(t, "version", "--nope"); code != 2 {
		t.Fatalf("want exit 2, got %d", code)
	}
}

func TestUnknownCommandIsUsageError(t *testing.T) {
	if code, _, _ := run(t, "frobnicate"); code != 2 {
		t.Fatalf("want exit 2, got %d", code)
	}
}

func TestInvalidOutputFormat(t *testing.T) {
	code, _, errOut := run(t, "version", "-o", "yaml")
	if code != 2 || !strings.Contains(errOut, "invalid --output") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}
