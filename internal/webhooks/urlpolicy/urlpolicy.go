package urlpolicy

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

// ErrInvalid reports a webhook endpoint URL that violates the endpoint policy.
var ErrInvalid = errors.New("goncho: invalid webhook url")

// NormalizeEndpoint trims, parses, and validates a webhook endpoint URL.
func NormalizeEndpoint(raw string, maxLength int) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > maxLength {
		return "", ErrInvalid
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "", ErrInvalid
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return "", ErrInvalid
	}
	host := parsed.Hostname()
	if host == "" || privateHost(host) {
		return "", ErrInvalid
	}
	return parsed.String(), nil
}

// RedactEndpoint removes secret-bearing URL components before audit/evidence use.
func RedactEndpoint(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return ""
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return "<redacted>"
	}
	parsed.User = nil
	if parsed.RawQuery != "" || parsed.ForceQuery {
		parsed.RawQuery = "<redacted>"
	}
	parsed.Fragment = ""
	return parsed.String()
}

func privateHost(host string) bool {
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()
}
