package service

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsPublicUnicast_RejectsInternalNetworks(t *testing.T) {
	bad := []string{
		// Loopback
		"127.0.0.1", "::1",
		// Link-local + the cloud metadata IP
		"169.254.0.1", "169.254.169.254", "fe80::1",
		// Private
		"10.0.0.1", "172.16.0.1", "192.168.1.1", "fc00::1", "fd00::1",
		// CGNAT
		"100.64.0.1", "100.127.255.254",
		// Multicast / broadcast / unspecified
		"224.0.0.1", "255.255.255.255", "0.0.0.0", "::",
	}
	for _, s := range bad {
		ip := net.ParseIP(s)
		assert.False(t, isPublicUnicast(ip), "expected %s to be rejected as internal", s)
	}
}

func TestIsPublicUnicast_AcceptsPublicAddresses(t *testing.T) {
	// Sanity-check a few real public addresses. We do NOT do DNS here —
	// the function operates on net.IP values directly.
	good := []string{
		"8.8.8.8", // Google DNS
		"1.1.1.1", // Cloudflare DNS
		"2606:4700:4700::1111",
		"104.16.0.1", // Cloudflare
	}
	for _, s := range good {
		ip := net.ParseIP(s)
		assert.True(t, isPublicUnicast(ip), "expected %s to be accepted as public", s)
	}
}

func TestValidateOutgoingWebhookURL_RejectsBadScheme(t *testing.T) {
	for _, raw := range []string{
		"file:///etc/passwd",
		"gopher://example.com/",
		"ftp://example.com/",
		"javascript:alert(1)",
	} {
		err := validateOutgoingWebhookURL(context.Background(), raw, nil)
		assert.Error(t, err, "expected scheme rejection for %s", raw)
	}
}

func TestValidateOutgoingWebhookURL_RejectsLoopbackLiteral(t *testing.T) {
	for _, raw := range []string{
		"http://127.0.0.1/webhook",
		"http://127.0.0.1:8080/webhook",
		"http://[::1]/webhook",
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.5:5432/",
	} {
		err := validateOutgoingWebhookURL(context.Background(), raw, nil)
		assert.ErrorIs(t, err, ErrUnsafeWebhookURL, "expected SSRF rejection for %s", raw)
	}
}

func TestValidateOutgoingWebhookURL_MissingHost(t *testing.T) {
	err := validateOutgoingWebhookURL(context.Background(), "http:///bad", nil)
	assert.Error(t, err)
}
