package websupport

import (
	"net"
	"strconv"
	"strings"
)

// ResolveChatFallbackPort extracts the final component of a host:port pair.
func ResolveChatFallbackPort(rawAddr string) string {
	trimmed := strings.TrimSpace(rawAddr)
	if trimmed == "" {
		return ""
	}
	_, port, err := net.SplitHostPort(trimmed)
	if err == nil {
		return SanitizePort(port)
	}

	if strings.Count(trimmed, ":") <= 1 {
		if idx := strings.LastIndex(trimmed, ":"); idx >= 0 {
			return SanitizePort(trimmed[idx+1:])
		}
	}

	return SanitizePort(trimmed)
}

// SanitizePort validates and normalizes a port string.
func SanitizePort(raw string) string {
	port := strings.TrimSpace(raw)
	if port == "" {
		return ""
	}
	n, err := strconv.Atoi(port)
	if err != nil {
		return ""
	}
	if n < 1 || n > 65535 {
		return ""
	}
	return port
}
