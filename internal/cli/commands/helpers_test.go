package commands

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
