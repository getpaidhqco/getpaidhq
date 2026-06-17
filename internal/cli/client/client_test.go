package client_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"getpaidhq/internal/cli/client"
)

func TestDoSendsAuthAndBody(t *testing.T) {
	var gotKey, gotCT, gotBody, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("x-api-key")
		gotCT = r.Header.Get("Content-Type")
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		gotQuery = r.URL.RawQuery
		w.Write([]byte(`{"id":"x"}`))
	}))
	defer srv.Close()

	c := client.New(srv.URL+"/", "sk_test") // trailing slash must be trimmed
	raw, err := c.Do(context.Background(), http.MethodPost, "/api/customers",
		url.Values{"page": {"0"}}, map[string]string{"email": "a@b.c"})
	if err != nil {
		t.Fatal(err)
	}
	if gotKey != "sk_test" || gotCT != "application/json" {
		t.Fatalf("key=%q ct=%q", gotKey, gotCT)
	}
	if gotBody != `{"email":"a@b.c"}` || gotQuery != "page=0" {
		t.Fatalf("body=%q query=%q", gotBody, gotQuery)
	}
	if string(raw) != `{"id":"x"}` {
		t.Fatalf("raw=%q", raw)
	}
}

func TestDoRawMessagePassthrough(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()
	c := client.New(srv.URL, "k")
	if _, err := c.Do(context.Background(), http.MethodPost, "/api/orders", nil,
		json.RawMessage(`{"nested":{"deep":true}}`)); err != nil {
		t.Fatal(err)
	}
	if gotBody != `{"nested":{"deep":true}}` {
		t.Fatalf("body=%q", gotBody)
	}
}

func TestDoAPIError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(422)
		w.Write([]byte(`{"code":"validation_error","message":"Validation failed","details":["email"]}`))
	}))
	defer srv.Close()
	_, err := client.New(srv.URL, "k").Do(context.Background(), http.MethodGet, "/api/customers", nil, nil)
	var apiErr *client.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("want APIError, got %T %v", err, err)
	}
	if apiErr.Status != 422 || apiErr.Code != "validation_error" {
		t.Fatalf("%+v", apiErr)
	}
}

func TestDoNonEnvelopeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(502)
		w.Write([]byte("bad gateway"))
	}))
	defer srv.Close()
	_, err := client.New(srv.URL, "k").Do(context.Background(), http.MethodGet, "/api/health", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "HTTP 502") {
		t.Fatalf("err=%v", err)
	}
}

func TestDoNoAPIKey(t *testing.T) {
	c := client.New("http://127.0.0.1:1", "")
	_, err := c.Do(context.Background(), http.MethodGet, "/api/customers", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "no API key") {
		t.Fatalf("err=%v", err)
	}
}

func TestListQuery(t *testing.T) {
	q := client.ListQuery(2, 50, "email", "asc")
	if q.Encode() != "limit=50&page=2&sort_by=email&sort_order=asc" {
		t.Fatalf("%s", q.Encode())
	}
}
