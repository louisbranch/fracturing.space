package httpx

import (
	"net/http"
	"strings"
)

// SchemePolicy controls how request metadata resolves the request scheme.
//
// TrustForwardedProto must be explicitly enabled for X-Forwarded-Proto to be
// considered. Keeping this explicit avoids trusting headers from untrusted
// clients.
type SchemePolicy struct {
	TrustForwardedProto bool
}

// IsHTTPSWithPolicy reports whether a request should be treated as HTTPS using
// the provided scheme policy.
func IsHTTPSWithPolicy(r *http.Request, policy SchemePolicy) bool {
	return RequestScheme(r, policy) == "https"
}

// RequestScheme resolves the effective scheme ("http" or "https") for a request
// under the provided policy.
func RequestScheme(r *http.Request, policy SchemePolicy) string {
	if r == nil {
		return ""
	}
	if policy.TrustForwardedProto {
		if forwarded := strings.ToLower(strings.TrimSpace(r.Header.Get("X-Forwarded-Proto"))); forwarded == "http" || forwarded == "https" {
			return forwarded
		}
	}
	if r.URL != nil {
		if scheme := strings.ToLower(strings.TrimSpace(r.URL.Scheme)); scheme == "http" || scheme == "https" {
			return scheme
		}
	}
	if r.TLS != nil {
		return "https"
	}
	return "http"
}
