package seed

import (
	"context"
	"net"
	"strings"
)

// LookupHost resolves a hostname for local fallback checks. It is exposed for tests.
var LookupHost = net.DefaultResolver.LookupHost

// ResolveLocalFallbackAddr returns the original address when host resolution succeeds.
// If host resolution fails, it falls back to localhost with the same port.
func ResolveLocalFallbackAddr(addr string) string {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return addr
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	if host == "" || port == "" {
		return addr
	}
	if _, err := LookupHost(context.Background(), host); err == nil {
		return addr
	}
	return "127.0.0.1:" + port
}
