package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// ErrUnsafeWebhookURL is returned by validateOutgoingWebhookURL when a URL
// resolves to an internal-network address (loopback, link-local,
// metadata, private CIDRs). Treat as a 4xx — customer-supplied input,
// not an infra failure.
var ErrUnsafeWebhookURL = errors.New("webhook url targets an internal-network address")

// allowAllIPs is the test-only predicate. NEVER referenced from
// production code paths — the only way to install it is from tests in
// this package via WebhookSubscriptionService.ipPredicate.
func allowAllIPs(net.IP) bool { return true }

// validateOutgoingWebhookURL parses and pre-resolves a customer-supplied
// webhook URL, rejecting anything that points at internal infra. The
// predicate decides "is this IP OK" — production passes
// isPublicUnicast.
func validateOutgoingWebhookURL(ctx context.Context, raw string, allow func(net.IP) bool) error {
	if allow == nil {
		allow = isPublicUnicast
	}
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("invalid webhook url: %w", err)
	}
	if u.Scheme != "https" && u.Scheme != "http" {
		return fmt.Errorf("invalid webhook url scheme %q (must be http or https)", u.Scheme)
	}
	if u.Host == "" {
		return errors.New("invalid webhook url: missing host")
	}
	host := u.Hostname()
	if host == "" {
		return errors.New("invalid webhook url: missing host")
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return fmt.Errorf("webhook url DNS resolution failed: %w", err)
	}
	if len(ips) == 0 {
		return errors.New("webhook url DNS resolution returned no addresses")
	}
	for _, ip := range ips {
		if !allow(ip) {
			return ErrUnsafeWebhookURL
		}
	}
	return nil
}

// safeDialContextWith builds a DialContext that re-validates the
// resolved IP at connection time using `predicate()` — a getter so the
// service can swap its predicate (e.g. tests) and the transport picks
// up the change. This is the DNS-rebinding defense: the URL might have
// validated cleanly at Create / SendWebhook entry, but a malicious DNS
// server can change its A record between then and the TCP connect.
func safeDialContextWith(d *net.Dialer, predicate func() func(net.IP) bool) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		allow := predicate()
		if allow == nil {
			allow = isPublicUnicast
		}
		host, port, err := net.SplitHostPort(addr)
		if err != nil {
			return nil, err
		}
		ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
		if err != nil {
			return nil, err
		}
		for _, ip := range ips {
			if !allow(ip) {
				return nil, ErrUnsafeWebhookURL
			}
		}
		var lastErr error
		for _, ip := range ips {
			conn, err := d.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
			if err == nil {
				return conn, nil
			}
			lastErr = err
		}
		if lastErr == nil {
			lastErr = errors.New("no IPs to dial")
		}
		return nil, lastErr
	}
}

// isPublicUnicast reports whether an IP is suitable for outbound
// webhook delivery — i.e. it's a real, internet-routable destination
// rather than something internal. The Go std lib's IsPrivate covers
// RFC1918 + RFC4193; we add the rest of the "really not the public
// internet" ranges ourselves.
//
// Specifically rejected:
//   - Loopback (127/8, ::1)
//   - Link-local (169.254/16, fe80::/10) — includes the cloud metadata IP
//   - Private (10/8, 172.16/12, 192.168/16, fc00::/7)
//   - Multicast / broadcast / unspecified
//   - CGNAT (100.64/10) — ISP carrier-grade NAT, sometimes routable in
//     ways that surprise you
func isPublicUnicast(ip net.IP) bool {
	if ip == nil {
		return false
	}
	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() || ip.IsMulticast() ||
		ip.IsUnspecified() || ip.IsPrivate() {
		return false
	}
	// CGNAT: 100.64.0.0/10
	if v4 := ip.To4(); v4 != nil && v4[0] == 100 && v4[1]&0xc0 == 64 {
		return false
	}
	// IPv4 broadcast.
	if ip.Equal(net.IPv4bcast) {
		return false
	}
	return true
}
