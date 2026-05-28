package handler

import (
	"net"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustParseCIDRs(t *testing.T, specs ...string) []*net.IPNet {
	t.Helper()
	out := make([]*net.IPNet, 0, len(specs))
	for _, s := range specs {
		_, n, err := net.ParseCIDR(s)
		require.NoError(t, err)
		out = append(out, n)
	}
	return out
}

func TestClientIP_NoTrustedProxies_IgnoresHeaders(t *testing.T) {
	// With no trusted proxies, the headers MUST be ignored — anyone can
	// spoof X-Forwarded-For from a raw HTTP client, so trusting it
	// without a proxy gate is the original bug.
	cases := []struct {
		name       string
		realIP     string
		forwarded  string
		remoteAddr string
		want       string
	}{
		{name: "headers ignored, RemoteAddr wins", realIP: "203.0.113.5", forwarded: "1.1.1.1", remoteAddr: "10.0.0.1:1234", want: "10.0.0.1"},
		{name: "bare RemoteAddr without port", remoteAddr: "192.0.2.1", want: "192.0.2.1"},
		{name: "spoofed XFF ignored", forwarded: "1.2.3.4", remoteAddr: "203.0.113.99:55555", want: "203.0.113.99"},
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
			assert.Equal(t, c.want, clientIP(r, nil))
		})
	}
}

func TestClientIP_TrustedProxy_HonorsHeaders(t *testing.T) {
	trusted := mustParseCIDRs(t, "10.0.0.0/8")

	cases := []struct {
		name       string
		realIP     string
		forwarded  string
		remoteAddr string
		want       string
	}{
		{name: "X-Real-IP wins when peer is trusted", realIP: "203.0.113.5", forwarded: "1.1.1.1", remoteAddr: "10.0.0.1:1234", want: "203.0.113.5"},
		{name: "X-Forwarded-For single", forwarded: "198.51.100.7", remoteAddr: "10.0.0.1:1234", want: "198.51.100.7"},
		{name: "X-Forwarded-For takes first hop", forwarded: "198.51.100.7, 10.0.0.99, 10.0.0.100", remoteAddr: "10.0.0.1:1234", want: "198.51.100.7"},
		{name: "X-Forwarded-For whitespace trimmed", forwarded: "   198.51.100.7   ", remoteAddr: "10.0.0.1:1234", want: "198.51.100.7"},
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
			assert.Equal(t, c.want, clientIP(r, trusted))
		})
	}
}

func TestClientIP_UntrustedPeer_IgnoresHeadersEvenWithProxiesConfigured(t *testing.T) {
	// If TRUSTED_PROXIES=10.0.0.0/8 but the request comes straight from
	// the public internet (203.0.113.x), the headers must still be
	// ignored — otherwise an attacker can hit the app directly and
	// forge whatever IP they like.
	trusted := mustParseCIDRs(t, "10.0.0.0/8")

	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "203.0.113.99:55555"
	r.Header.Set("X-Forwarded-For", "1.2.3.4")
	r.Header.Set("X-Real-IP", "5.6.7.8")

	assert.Equal(t, "203.0.113.99", clientIP(r, trusted), "untrusted peer ⇒ headers dropped, RemoteAddr wins")
}

func TestParseTrustedProxies(t *testing.T) {
	t.Run("empty input returns nil", func(t *testing.T) {
		got, err := ParseTrustedProxies("")
		require.NoError(t, err)
		assert.Nil(t, got)
	})
	t.Run("CIDRs parse", func(t *testing.T) {
		got, err := ParseTrustedProxies("10.0.0.0/8,192.168.0.0/16")
		require.NoError(t, err)
		require.Len(t, got, 2)
	})
	t.Run("bare IP promotes to /32", func(t *testing.T) {
		got, err := ParseTrustedProxies("127.0.0.1")
		require.NoError(t, err)
		require.Len(t, got, 1)
		ones, _ := got[0].Mask.Size()
		assert.Equal(t, 32, ones)
	})
	t.Run("invalid entry surfaces an error", func(t *testing.T) {
		_, err := ParseTrustedProxies("not-an-ip")
		assert.Error(t, err)
	})
	t.Run("whitespace and empty entries tolerated", func(t *testing.T) {
		got, err := ParseTrustedProxies(" 10.0.0.0/8 , , 192.168.0.1 ")
		require.NoError(t, err)
		require.Len(t, got, 2)
	})
}
