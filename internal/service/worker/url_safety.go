package worker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

func (w *worker) validateExternalURL(rawURL string) error {
	return validateExternalURL(rawURL, w.allowPrivateStreamURLs)
}

func validateExternalURL(rawURL string, allowPrivate bool) error {
	parsed, err := url.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return fmt.Errorf("invalid url: %w", err)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return errors.New("url scheme must be http or https")
	}
	host := strings.TrimSpace(parsed.Hostname())
	if host == "" {
		return errors.New("url host is required")
	}
	if allowPrivate {
		return nil
	}

	if strings.EqualFold(host, "localhost") {
		return errors.New("localhost host is not allowed")
	}

	if ip, err := netip.ParseAddr(host); err == nil {
		if isPrivateAddr(ip) {
			return errors.New("private or loopback ip is not allowed")
		}
		return nil
	}

	if isLikelyInternalHostname(host) {
		return errors.New("internal hostnames are not allowed")
	}

	resolveCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	resolvedIPs, err := net.DefaultResolver.LookupNetIP(resolveCtx, "ip", host)
	if err != nil {
		return fmt.Errorf("host resolve failed: %w", err)
	}
	if len(resolvedIPs) == 0 {
		return errors.New("host resolve returned no addresses")
	}
	for _, ip := range resolvedIPs {
		addr, parseErr := netip.ParseAddr(ip.String())
		if parseErr != nil {
			return fmt.Errorf("failed to parse resolved ip: %w", parseErr)
		}
		if isPrivateAddr(addr) {
			return errors.New("resolved to private or loopback ip")
		}
	}
	return nil
}

func isLikelyInternalHostname(host string) bool {
	normalized := strings.ToLower(strings.TrimSpace(host))
	return !strings.Contains(normalized, ".") ||
		strings.HasSuffix(normalized, ".local") ||
		strings.HasSuffix(normalized, ".internal") ||
		strings.HasSuffix(normalized, ".localhost")
}

func isPrivateAddr(addr netip.Addr) bool {
	return addr.IsLoopback() ||
		addr.IsPrivate() ||
		addr.IsLinkLocalUnicast() ||
		addr.IsLinkLocalMulticast() ||
		addr.IsMulticast() ||
		addr.IsUnspecified()
}
