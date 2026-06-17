package cli_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"getpaidhq/internal/cli"
)

func TestGeneratedDocsAreCurrent(t *testing.T) {
	tmp := t.TempDir()
	if code, _, errOut := run(t, "docs", "--dir", tmp); code != 0 {
		t.Fatalf("docs generation failed: %s", errOut)
	}
	want, err := os.ReadDir(tmp)
	if err != nil {
		t.Fatal(err)
	}
	const committed = "../../docs/cli/reference"
	got, err := os.ReadDir(committed)
	if err != nil {
		t.Fatalf("%v — run `make docs-cli` and commit the result", err)
	}
	if len(want) != len(got) {
		t.Fatalf("reference has %d files, regenerated %d — run `make docs-cli`", len(got), len(want))
	}
	for _, f := range want {
		a, _ := os.ReadFile(filepath.Join(tmp, f.Name()))
		b, err := os.ReadFile(filepath.Join(committed, f.Name()))
		if err != nil || !bytes.Equal(a, b) {
			t.Errorf("docs/cli/reference/%s is stale — run `make docs-cli`", f.Name())
		}
	}
}

func run(t *testing.T, args ...string) (code int, out, errOut string) {
	t.Helper()
	// Isolate from host environment so tests don't pick up developer config,
	// API keys, or output settings.
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GPHQ_API_KEY", "")
	t.Setenv("GPHQ_BASE_URL", "")
	t.Setenv("GPHQ_OUTPUT", "")
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

func TestMissingAPIKey(t *testing.T) {
	code, _, errOut := run(t, "customers", "list", "--base-url", "http://127.0.0.1:1")
	if code != 1 || !strings.Contains(errOut, "no API key") {
		t.Fatalf("code=%d err=%q", code, errOut)
	}
}

// TestConfigPrecedence verifies the flags > env > config file > defaults chain.
func TestConfigPrecedence(t *testing.T) {
	// (a) GPHQ_OUTPUT=yaml env → invalid output, exit 2.
	t.Run("env_invalid_output", func(t *testing.T) {
		xdg := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdg)
		t.Setenv("GPHQ_OUTPUT", "yaml")
		t.Setenv("GPHQ_API_KEY", "")
		t.Setenv("GPHQ_BASE_URL", "")
		var o, e bytes.Buffer
		code := cli.Run([]string{"version"}, strings.NewReader(""), &o, &e)
		if code != 2 || !strings.Contains(e.String(), "invalid --output") {
			t.Fatalf("(a) want exit 2 with invalid --output; code=%d err=%q", code, e.String())
		}
	})

	// (b) -o json flag beats GPHQ_OUTPUT=yaml env → exit 0.
	t.Run("flag_beats_env", func(t *testing.T) {
		xdg := t.TempDir()
		t.Setenv("XDG_CONFIG_HOME", xdg)
		t.Setenv("GPHQ_OUTPUT", "yaml")
		t.Setenv("GPHQ_API_KEY", "")
		t.Setenv("GPHQ_BASE_URL", "")
		var o, e bytes.Buffer
		code := cli.Run([]string{"version", "-o", "json"}, strings.NewReader(""), &o, &e)
		if code != 0 {
			t.Fatalf("(b) want exit 0; code=%d err=%q", code, e.String())
		}
	})

	// (c) Config file output=yaml (no env override) → invalid output, exit 2.
	t.Run("config_file_invalid_output", func(t *testing.T) {
		xdg := t.TempDir()
		cfgDir := filepath.Join(xdg, "gphq")
		if err := os.MkdirAll(cfgDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"),
			[]byte(fmt.Sprintf("output = %q\n", "yaml")), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("XDG_CONFIG_HOME", xdg)
		t.Setenv("GPHQ_OUTPUT", "")
		t.Setenv("GPHQ_API_KEY", "")
		t.Setenv("GPHQ_BASE_URL", "")
		var o, e bytes.Buffer
		code := cli.Run([]string{"version"}, strings.NewReader(""), &o, &e)
		if code != 2 || !strings.Contains(e.String(), "invalid --output") {
			t.Fatalf("(c) want exit 2 with invalid --output; code=%d err=%q", code, e.String())
		}
	})

	// (d) GPHQ_OUTPUT=table env beats config file output=yaml → exit 0.
	t.Run("env_beats_config_file", func(t *testing.T) {
		xdg := t.TempDir()
		cfgDir := filepath.Join(xdg, "gphq")
		if err := os.MkdirAll(cfgDir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(cfgDir, "config.toml"),
			[]byte(fmt.Sprintf("output = %q\n", "yaml")), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Setenv("XDG_CONFIG_HOME", xdg)
		t.Setenv("GPHQ_OUTPUT", "table")
		t.Setenv("GPHQ_API_KEY", "")
		t.Setenv("GPHQ_BASE_URL", "")
		var o, e bytes.Buffer
		code := cli.Run([]string{"version"}, strings.NewReader(""), &o, &e)
		if code != 0 {
			t.Fatalf("(d) want exit 0; code=%d err=%q", code, e.String())
		}
	})
}
