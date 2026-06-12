package commands_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"

	"getpaidhq/internal/cli"
)

type cmdCase struct {
	name       string
	args       []string   // CLI args, without --base-url/--api-key
	stdin      string     // piped into the command (--data -)
	wantMethod string     // expected request method ("" = no request expected)
	wantPath   string     // exact request path
	wantQuery  url.Values // subset match
	wantBody   string     // JSON-equal match ("" = skip)
	wantNoBody bool       // when true, assert the captured request body is empty
	respStatus int        // default 200
	respBody   string
	wantOut    []string // substrings on stdout
	wantErr    []string // substrings on stderr
	wantCode   int
}

func runCase(t *testing.T, tc cmdCase) {
	t.Helper()
	// isolate from host env/config (mirrors root_test.go)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("GPHQ_API_KEY", "")
	t.Setenv("GPHQ_BASE_URL", "")
	t.Setenv("GPHQ_OUTPUT", "")
	var mu sync.Mutex
	var gotMethod, gotPath, gotBody, gotKey string
	var gotQuery url.Values
	requests := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requests++
		gotMethod, gotPath, gotQuery = r.Method, r.URL.Path, r.URL.Query()
		gotKey = r.Header.Get("x-api-key")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		mu.Unlock()
		if tc.respStatus != 0 {
			w.WriteHeader(tc.respStatus)
		}
		_, _ = w.Write([]byte(tc.respBody))
	}))
	defer srv.Close()

	args := append(append([]string{}, tc.args...), "--base-url", srv.URL, "--api-key", "sk_test_123")
	var out, errOut bytes.Buffer
	code := cli.Run(args, strings.NewReader(tc.stdin), &out, &errOut)

	mu.Lock()
	defer mu.Unlock()

	if code != tc.wantCode {
		t.Fatalf("exit=%d want %d\nstdout: %s\nstderr: %s", code, tc.wantCode, out.String(), errOut.String())
	}
	if tc.wantMethod == "" && requests > 0 {
		t.Fatalf("expected no request, server saw %s %s", gotMethod, gotPath)
	}
	if tc.wantMethod != "" {
		if gotMethod != tc.wantMethod || gotPath != tc.wantPath {
			t.Fatalf("request = %s %s, want %s %s", gotMethod, gotPath, tc.wantMethod, tc.wantPath)
		}
		if gotKey != "sk_test_123" {
			t.Errorf("x-api-key = %q", gotKey)
		}
		for k := range tc.wantQuery {
			if got := gotQuery.Get(k); got != tc.wantQuery.Get(k) {
				t.Errorf("query[%s] = %q, want %q", k, got, tc.wantQuery.Get(k))
			}
		}
		if tc.wantNoBody {
			if gotBody != "" {
				t.Errorf("expected empty request body, got %q", gotBody)
			}
		}
		if tc.wantBody != "" {
			var got, want any
			if err := json.Unmarshal([]byte(gotBody), &got); err != nil {
				t.Fatalf("request body not JSON: %q", gotBody)
			}
			if err := json.Unmarshal([]byte(tc.wantBody), &want); err != nil {
				t.Fatalf("bad wantBody in test: %v", err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("body = %s\nwant  %s", gotBody, tc.wantBody)
			}
		}
	}
	for _, s := range tc.wantOut {
		if !strings.Contains(out.String(), s) {
			t.Errorf("stdout missing %q in:\n%s", s, out.String())
		}
	}
	for _, s := range tc.wantErr {
		if !strings.Contains(errOut.String(), s) {
			t.Errorf("stderr missing %q in:\n%s", s, errOut.String())
		}
	}
}

func runCases(t *testing.T, cases []cmdCase) {
	t.Helper()
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) { runCase(t, tc) })
	}
}
