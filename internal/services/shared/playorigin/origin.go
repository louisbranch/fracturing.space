package playorigin

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

const playHostPrefix = "play."

// PlayURL resolves the play-host absolute URL for the provided request host.
func PlayURL(r *http.Request, policy requestmeta.SchemePolicy, playFallbackPort string, path string) string {
	return absoluteURL(derivePlayHost(requestHost(r), playFallbackPort), requestScheme(r, policy), path)
}

// WebURL resolves the apex-web absolute URL for the provided request host.
func WebURL(r *http.Request, policy requestmeta.SchemePolicy, webFallbackPort string, path string) string {
	return absoluteURL(deriveWebHost(requestHost(r), webFallbackPort), requestScheme(r, policy), path)
}

func derivePlayHost(host string, fallbackPort string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	name, port := splitHostPort(host)
	if isLoopbackLikeHost(name) && strings.TrimSpace(fallbackPort) != "" {
		return joinHostPort(name, fallbackPort)
	}
	if strings.HasPrefix(name, playHostPrefix) {
		return joinHostPort(name, port)
	}
	return joinHostPort(playHostPrefix+name, port)
}

func deriveWebHost(host string, fallbackPort string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return ""
	}
	name, port := splitHostPort(host)
	name = strings.TrimPrefix(name, playHostPrefix)
	if isLoopbackLikeHost(name) && strings.TrimSpace(fallbackPort) != "" {
		port = strings.TrimSpace(fallbackPort)
	}
	return joinHostPort(name, port)
}

func absoluteURL(host, scheme, path string) string {
	host = strings.TrimSpace(host)
	if host == "" {
		return path
	}
	scheme = strings.TrimSpace(scheme)
	if scheme == "" {
		scheme = "http"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + strings.TrimLeft(path, "/")
	}
	return scheme + "://" + host + path
}

func requestScheme(r *http.Request, policy requestmeta.SchemePolicy) string {
	if requestmeta.IsHTTPSWithPolicy(r, policy) {
		return "https"
	}
	return "http"
}

func requestHost(r *http.Request) string {
	if r == nil {
		return ""
	}
	if host := strings.TrimSpace(r.Host); host != "" {
		return host
	}
	if r.URL == nil {
		return ""
	}
	return strings.TrimSpace(r.URL.Host)
}

func splitHostPort(host string) (string, string) {
	parsed, err := url.Parse("//" + host)
	if err != nil {
		return strings.ToLower(strings.TrimSpace(host)), ""
	}
	return strings.ToLower(strings.TrimSpace(parsed.Hostname())), strings.TrimSpace(parsed.Port())
}

func joinHostPort(host, port string) string {
	host = strings.TrimSpace(host)
	port = strings.TrimSpace(port)
	if host == "" {
		return ""
	}
	if port == "" {
		return host
	}
	return host + ":" + port
}

func isLoopbackLikeHost(host string) bool {
	host = strings.ToLower(strings.TrimSpace(host))
	if host == "" {
		return false
	}
	if host == "localhost" || strings.HasSuffix(host, ".localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
