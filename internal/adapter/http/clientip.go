package handler

import (
	"net"
	"net/http"
	"strings"
)

// ParseTrustedProxies parses a comma-separated list of CIDR blocks (e.g.
// "10.0.0.0/8,127.0.0.0/8") into *net.IPNet values. Bare IPs are accepted
// and treated as /32 (or /128 for IPv6). Empty input yields a nil slice
// — semantically "trust no upstream", which is the safe default.
//
// Returns the first parse error encountered; the caller decides whether
// to fail boot or fall back to a safe default.
func ParseTrustedProxies(spec string) ([]*net.IPNet, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, nil
	}
	parts := strings.Split(spec, ",")
	out := make([]*net.IPNet, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if _, ipnet, err := net.ParseCIDR(p); err == nil {
			out = append(out, ipnet)
			continue
		}
		// Bare IP — promote to /32 or /128.
		ip := net.ParseIP(p)
		if ip == nil {
			return nil, &net.ParseError{Type: "trusted proxy", Text: p}
		}
		bits := 32
		if ip.To4() == nil {
			bits = 128
		}
		mask := net.CIDRMask(bits, bits)
		out = append(out, &net.IPNet{IP: ip, Mask: mask})
	}
	return out, nil
}

// remoteAddrIP extracts the immediate peer IP from r.RemoteAddr, which
// the std lib formats as "host:port" (or "[v6]:port"). Returns nil if
// the address isn't parseable.
func remoteAddrIP(r *http.Request) net.IP {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// Some test harnesses pass a bare host; try as-is.
		host = r.RemoteAddr
	}
	return net.ParseIP(host)
}

// isFromTrustedProxy reports whether the immediate peer is in one of the
// configured CIDRs. An empty list means "trust nothing", so the function
// returns false — which is exactly what we want when no proxy is
// configured: the X-Forwarded-For header gets ignored.
func isFromTrustedProxy(r *http.Request, trusted []*net.IPNet) bool {
	if len(trusted) == 0 {
		return false
	}
	ip := remoteAddrIP(r)
	if ip == nil {
		return false
	}
	for _, cidr := range trusted {
		if cidr.Contains(ip) {
			return true
		}
	}
	return false
}

// clientIP resolves the originating client IP for a request. The proxy
// headers (X-Real-IP, X-Forwarded-For) are honored only when the
// immediate peer is in the trusted-proxy list — otherwise they're
// trivially forgeable and would let an attacker poison audit logs or
// bypass per-IP rate limiting. With no trusted proxies configured we
// fall straight back to RemoteAddr, which is the safe default.
func clientIP(r *http.Request, trusted []*net.IPNet) string {
	if isFromTrustedProxy(r, trusted) {
		if v := r.Header.Get("X-Real-IP"); v != "" {
			return strings.TrimSpace(v)
		}
		if v := r.Header.Get("X-Forwarded-For"); v != "" {
			// The leftmost entry is the client per the convention; the
			// proxies in between are also in this header but they don't
			// help identify the user.
			if idx := strings.Index(v, ","); idx >= 0 {
				return strings.TrimSpace(v[:idx])
			}
			return strings.TrimSpace(v)
		}
	}
	if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
		return host
	}
	return r.RemoteAddr
}
