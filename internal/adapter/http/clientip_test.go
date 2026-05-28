package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClientIP(t *testing.T) {
	cases := []struct {
		name       string
		realIP     string
		forwarded  string
		remoteAddr string
		want       string
	}{
		{name: "X-Real-IP wins", realIP: "203.0.113.5", forwarded: "1.1.1.1", remoteAddr: "10.0.0.1:1234", want: "203.0.113.5"},
		{name: "X-Forwarded-For single", forwarded: "198.51.100.7", remoteAddr: "10.0.0.1:1234", want: "198.51.100.7"},
		{name: "X-Forwarded-For takes first hop", forwarded: "198.51.100.7, 10.0.0.99, 10.0.0.100", remoteAddr: "10.0.0.1:1234", want: "198.51.100.7"},
		{name: "X-Forwarded-For whitespace trimmed", forwarded: "   198.51.100.7   ", remoteAddr: "10.0.0.1:1234", want: "198.51.100.7"},
		{name: "RemoteAddr host:port", remoteAddr: "192.0.2.1:55555", want: "192.0.2.1"},
		{name: "RemoteAddr without port falls back to RemoteAddr verbatim", remoteAddr: "192.0.2.1", want: "192.0.2.1"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/", nil)
			r.RemoteAddr = c.remoteAddr
			if c.realIP != "" {
				r.Header.Set("X-Real-IP", c.realIP)
			}
			if c.forwarded != "" {
				r.Header.Set("X-Forwarded-For", c.forwarded)
			}
			assert.Equal(t, c.want, clientIP(r))
		})
	}
}
