// Package requestmeta provides normalized request metadata helpers.
package requestmeta

import (
	"net/http"
	"net/url"
	"strings"
)

// SchemePolicy controls how request metadata resolves request scheme.
//
// TrustForwardedProto must be explicitly enabled for X-Forwarded-Proto to be
// considered. Keeping this explicit avoids trusting headers from untrusted clients.
type SchemePolicy struct {
	TrustForwardedProto bool
}

// IsHTTPS reports whether a request should be treated as HTTPS.
func IsHTTPS(r *http.Request) bool {
	return IsHTTPSWithPolicy(r, SchemePolicy{})
}

// IsHTTPSWithPolicy reports whether a request should be treated as HTTPS using
// the provided scheme policy.
func IsHTTPSWithPolicy(r *http.Request, policy SchemePolicy) bool {
	return requestScheme(r, policy) == "https"
}

// HasSameOriginProof reports whether Origin or Referer proves same-origin.
func HasSameOriginProof(r *http.Request) bool {
	return HasSameOriginProofWithPolicy(r, SchemePolicy{})
}

// HasSameOriginProofWithPolicy reports whether Origin or Referer proves same-origin
// under the provided scheme policy.
func HasSameOriginProofWithPolicy(r *http.Request, policy SchemePolicy) bool {
	if r == nil {
		return false
	}
	requestScheme, requestHost, requestPort := requestOriginParts(r, policy)
	if requestHost == "" {
		return false
	}
	if origin := strings.TrimSpace(r.Header.Get("Origin")); origin != "" {
		return sameOriginHostPort(origin, requestScheme, requestHost, requestPort)
	}
	if referer := strings.TrimSpace(r.Header.Get("Referer")); referer != "" {
		return sameOriginHostPort(referer, requestScheme, requestHost, requestPort)
	}
	return false
}

func sameOriginHostPort(raw string, requestScheme string, requestHost string, requestPort string) bool {
	parsed, err := url.Parse(raw)
	if err != nil {
		return false
	}
	originScheme := strings.ToLower(strings.TrimSpace(parsed.Scheme))
	if originScheme == "" {
		return false
	}
	if requestScheme != "" && originScheme != requestScheme {
		return false
	}
	originHost := strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	if originHost == "" || originHost != requestHost {
		return false
	}
	originPort := strings.TrimSpace(parsed.Port())
	if originPort == "" {
		originPort = defaultPortForScheme(originScheme)
	}
	if requestPort == "" {
		requestPort = defaultPortForScheme(requestScheme)
	}
	if originPort == "" || requestPort == "" {
		return false
	}
	return originPort == requestPort
}

func requestOriginParts(r *http.Request, policy SchemePolicy) (string, string, string) {
	if r == nil {
		return "", "", ""
	}
	scheme := requestScheme(r, policy)
	host, port := requestHostParts(r.Host)
	if host == "" && r.URL != nil {
		host, port = requestHostParts(r.URL.Host)
	}
	if port == "" {
		port = defaultPortForScheme(scheme)
	}
	return scheme, host, port
}

func requestScheme(r *http.Request, policy SchemePolicy) string {
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

func defaultPortForScheme(scheme string) string {
	switch strings.ToLower(strings.TrimSpace(scheme)) {
	case "https":
		return "443"
	case "http":
		return "80"
	default:
		return ""
	}
}

func requestHostParts(rawHost string) (string, string) {
	parsed, err := url.Parse("//" + strings.TrimSpace(rawHost))
	if err != nil {
		return "", ""
	}
	return strings.ToLower(strings.TrimSpace(parsed.Hostname())), strings.TrimSpace(parsed.Port())
}
