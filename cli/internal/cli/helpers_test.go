package cli

// Internal (white-box) test: parseKV, readData, bindBody, rawBodyJSON and
// dataConflict are unexported helpers in package cli, so the suite lives in the
// package rather than cli_test. bodyOrData (the old combined helper) was split
// into bindBody (typed input types) and rawBodyJSON (free-form jx.Raw bodies);
// coverage for both is added below.

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestParseKV(t *testing.T) {
	tests := []struct {
		name    string
		pairs   []string
		flag    string
		want    map[string]string
		wantErr string
	}{
		{
			name:  "nil input",
			pairs: nil,
			want:  nil,
		},
		{
			name:  "empty slice",
			pairs: []string{},
			want:  nil,
		},
		{
			name:  "single pair",
			pairs: []string{"tier=gold"},
			flag:  "metadata",
			want:  map[string]string{"tier": "gold"},
		},
		{
			name:  "multiple pairs",
			pairs: []string{"tier=gold", "source=api"},
			flag:  "metadata",
			want:  map[string]string{"tier": "gold", "source": "api"},
		},
		{
			name:  "value with equals sign",
			pairs: []string{"url=https://example.com?a=b"},
			flag:  "metadata",
			want:  map[string]string{"url": "https://example.com?a=b"},
		},
		{
			name:    "missing equals",
			pairs:   []string{"nogood"},
			flag:    "metadata",
			wantErr: "--metadata expects key=value, got",
		},
		{
			name:    "empty key",
			pairs:   []string{"=value"},
			flag:    "metadata",
			wantErr: "--metadata expects key=value, got",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseKV(tc.pairs, tc.flag)
			if tc.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tc.wantErr)
				}
				if !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %q, want to contain %q", err.Error(), tc.wantErr)
				}
				var usageErr *UsageError
				if !errors.As(err, &usageErr) {
					t.Fatalf("error should be UsageError, got %T", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("got map %v, want %v", got, tc.want)
			}
			for k, v := range tc.want {
				if got[k] != v {
					t.Errorf("key %q: got %q, want %q", k, got[k], v)
				}
			}
		})
	}
}

func TestReadData(t *testing.T) {
	t.Run("inline JSON", func(t *testing.T) {
		raw, err := readData(nil, `{"key":"val"}`)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(raw) != `{"key":"val"}` {
			t.Errorf("got %q", string(raw))
		}
	})

	t.Run("@file", func(t *testing.T) {
		dir := t.TempDir()
		path := filepath.Join(dir, "body.json")
		if err := os.WriteFile(path, []byte(`{"from":"file"}`), 0o644); err != nil {
			t.Fatal(err)
		}
		raw, err := readData(nil, "@"+path)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(raw) != `{"from":"file"}` {
			t.Errorf("got %q", string(raw))
		}
	})

	t.Run("stdin dash", func(t *testing.T) {
		stdin := strings.NewReader(`{"from":"stdin"}`)
		raw, err := readData(stdin, "-")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(raw) != `{"from":"stdin"}` {
			t.Errorf("got %q", string(raw))
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		_, err := readData(nil, "not-json")
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
		var usageErr *UsageError
		if !errors.As(err, &usageErr) {
			t.Fatalf("expected UsageError, got %T: %v", err, err)
		}
		if !strings.Contains(err.Error(), "not valid JSON") {
			t.Errorf("error = %q", err.Error())
		}
	})

	t.Run("missing file", func(t *testing.T) {
		_, err := readData(nil, "@/nonexistent/path/body.json")
		if err == nil {
			t.Fatal("expected error for missing file")
		}
		var usageErr *UsageError
		if !errors.As(err, &usageErr) {
			t.Fatalf("expected UsageError, got %T: %v", err, err)
		}
		if !strings.Contains(err.Error(), "reading --data") {
			t.Errorf("error = %q", err.Error())
		}
	})
}

// bindTestCmd builds a command with a --data flag plus a single typed --name
// flag, mirroring the real bindBody contract used by the create/set commands.
func bindTestCmd(args []string) (*cobra.Command, error) {
	cmd := &cobra.Command{Use: "test", RunE: func(*cobra.Command, []string) error { return nil }}
	cmd.Flags().String("data", "", "raw JSON body")
	cmd.Flags().String("name", "", "name")
	cmd.SetArgs(args)
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	if err := cmd.ParseFlags(args); err != nil {
		return nil, err
	}
	return cmd, nil
}

type testBody struct {
	Name string `json:"name"`
}

// buildFromName populates testBody from the --name flag, the same shape the
// real RunE closures use with bindBody.
func buildFromName(cmd *cobra.Command) func(*testBody) error {
	return func(in *testBody) error {
		in.Name, _ = cmd.Flags().GetString("name")
		return nil
	}
}

func TestBindBody(t *testing.T) {
	t.Run("from typed flags", func(t *testing.T) {
		cmd, err := bindTestCmd([]string{"--name", "from-flag"})
		if err != nil {
			t.Fatal(err)
		}
		out, err := bindBody(cmd, buildFromName(cmd))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Name != "from-flag" {
			t.Errorf("Name = %q, want from-flag", out.Name)
		}
	})

	t.Run("from --data decodes typed input", func(t *testing.T) {
		cmd, err := bindTestCmd([]string{"--data", `{"name":"from-data"}`})
		if err != nil {
			t.Fatal(err)
		}
		out, err := bindBody(cmd, buildFromName(cmd))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if out.Name != "from-data" {
			t.Errorf("Name = %q, want from-data", out.Name)
		}
	})

	t.Run("--data XOR other flags", func(t *testing.T) {
		cmd, err := bindTestCmd([]string{"--data", `{"name":"x"}`, "--name", "y"})
		if err != nil {
			t.Fatal(err)
		}
		_, err = bindBody(cmd, buildFromName(cmd))
		if err == nil {
			t.Fatal("expected conflict error")
		}
		var usageErr *UsageError
		if !errors.As(err, &usageErr) {
			t.Fatalf("expected UsageError, got %T: %v", err, err)
		}
		if !strings.Contains(err.Error(), "--data cannot be combined with --name") {
			t.Errorf("error = %q", err.Error())
		}
	})

	t.Run("--data invalid JSON", func(t *testing.T) {
		cmd, err := bindTestCmd([]string{"--data", `not-json`})
		if err != nil {
			t.Fatal(err)
		}
		_, err = bindBody(cmd, buildFromName(cmd))
		if err == nil {
			t.Fatal("expected error for invalid JSON")
		}
		var usageErr *UsageError
		if !errors.As(err, &usageErr) {
			t.Fatalf("expected UsageError, got %T: %v", err, err)
		}
	})
}

func TestRawBodyJSON(t *testing.T) {
	t.Run("from build func", func(t *testing.T) {
		cmd, err := bindTestCmd([]string{"--name", "x"})
		if err != nil {
			t.Fatal(err)
		}
		raw, err := rawBodyJSON(cmd, func() (any, error) {
			return map[string]string{"k": "v"}, nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		var got map[string]string
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatalf("not JSON: %v", err)
		}
		if got["k"] != "v" {
			t.Errorf("got %v", got)
		}
	})

	t.Run("--data passes through", func(t *testing.T) {
		cmd, err := bindTestCmd([]string{"--data", `{"raw":true}`})
		if err != nil {
			t.Fatal(err)
		}
		raw, err := rawBodyJSON(cmd, func() (any, error) {
			t.Fatal("build should not be called when --data set")
			return nil, nil
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(raw) != `{"raw":true}` {
			t.Errorf("got %q", string(raw))
		}
	})

	t.Run("--data XOR other flags", func(t *testing.T) {
		cmd, err := bindTestCmd([]string{"--data", `{"raw":true}`, "--name", "y"})
		if err != nil {
			t.Fatal(err)
		}
		_, err = rawBodyJSON(cmd, func() (any, error) { return nil, nil })
		if err == nil {
			t.Fatal("expected conflict error")
		}
		var usageErr *UsageError
		if !errors.As(err, &usageErr) {
			t.Fatalf("expected UsageError, got %T: %v", err, err)
		}
	})
}
