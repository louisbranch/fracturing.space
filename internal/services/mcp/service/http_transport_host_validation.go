package service

import (
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
)

// validateLocalRequest enforces host access to mitigate DNS rebinding.
// It checks Host and Origin headers against allowed hosts per MCP guidance so
// remote web pages cannot reach local MCP servers via rebinding.
// This is the transport-side "network guardrail" before we have richer auth.
func (t *HTTPTransport) validateLocalRequest(r *http.Request) error {
	if r == nil {
		return fmt.Errorf("invalid request")
	}

	if !t.isAllowedHostHeader(r.Host) {
		return fmt.Errorf("invalid host")
	}

	origin := strings.TrimSpace(r.Header.Get("Origin"))
	if origin == "" {
		return nil
	}

	parsed, err := url.Parse(origin)
	if err != nil {
		return fmt.Errorf("invalid origin")
	}

	originHost := parsed.Host
	if originHost == "" {
		return fmt.Errorf("invalid origin")
	}

	if !t.isAllowedHostHeader(originHost) {
		return fmt.Errorf("invalid origin")
	}

	return nil
}

// isAllowedHostHeader reports whether a Host/Origin header resolves to an allowed host.
// The default posture is local-only unless explicit hosts are configured.
func (t *HTTPTransport) isAllowedHostHeader(host string) bool {
	resolvedHost, ok := normalizeHost(host)
	if !ok {
		return false
	}

	if isLoopbackHost(resolvedHost) {
		return true
	}

	allowed := t.allowedHosts
	if len(allowed) == 0 {
		return false
	}

	_, ok = allowed[strings.ToLower(resolvedHost)]
	return ok
}

// isLoopbackHost reports whether a host resolves to loopback.
// It is intentionally strict: only explicit local loopback hosts pass by default.
func isLoopbackHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	switch host {
	case "localhost", "127.0.0.1", "::1":
		return true
	default:
		return false
	}
}

// parseAllowedHosts parses allowed hosts from env-loaded values.
func parseAllowedHosts(hosts []string) map[string]struct{} {
	result := make(map[string]struct{}, len(hosts))
	for _, entry := range hosts {
		trimmed := strings.TrimSpace(entry)
		if trimmed == "" {
			continue
		}
		result[strings.ToLower(trimmed)] = struct{}{}
	}
	return result
}

// normalizeHost extracts the hostname portion from Host/Origin headers.
func normalizeHost(host string) (string, bool) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", false
	}

	if strings.HasPrefix(host, "[") {
		if splitHost, _, err := net.SplitHostPort(host); err == nil {
			return splitHost, true
		}
		if strings.HasSuffix(host, "]") {
			return strings.TrimSuffix(strings.TrimPrefix(host, "["), "]"), true
		}
		return "", false
	}

	if strings.Count(host, ":") > 1 {
		return host, true
	}

	if strings.Contains(host, ":") {
		splitHost, _, err := net.SplitHostPort(host)
		if err != nil {
			return "", false
		}
		return splitHost, true
	}

	return host, true
}

// handleHealth handles GET /mcp/health for health checks.
func (t *HTTPTransport) handleHealth(w http.ResponseWriter, r *http.Request) {
	if err := t.validateLocalRequest(r); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		log.Printf("Failed to write health response: %v", err)
	}
}
