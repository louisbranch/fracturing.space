// Package redirectpath validates safe first-party continuation targets used by
// public-auth login, signup, and recovery flows.
package redirectpath

import (
	"net/url"
	"path"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// ResolveSafe returns a canonical in-app or invite continuation path, or the
// empty string when the input is unsafe.
func ResolveSafe(raw string) string {
	next := strings.TrimSpace(raw)
	if next == "" {
		return ""
	}
	parsed, err := url.Parse(next)
	if err != nil || parsed.Scheme != "" || parsed.Host != "" || parsed.Opaque != "" {
		return ""
	}
	rawPath := strings.TrimSpace(parsed.EscapedPath())
	if hasEncodedSlash(rawPath) {
		return ""
	}
	decodedPath, err := url.PathUnescape(strings.TrimSpace(parsed.Path))
	if err != nil {
		return ""
	}
	if hasDotSegment(decodedPath) {
		return ""
	}
	canonicalPath := path.Clean(decodedPath)
	if strings.TrimSpace(canonicalPath) == "." {
		canonicalPath = "/"
	}
	canonicalPath = ensureLeadingSlash(canonicalPath)
	if canonicalPath == routepath.AppPrefix || canonicalPath == strings.TrimRight(routepath.InvitePrefix, "/") {
		return ""
	}
	if !strings.HasPrefix(canonicalPath, routepath.AppPrefix) && !strings.HasPrefix(canonicalPath, routepath.InvitePrefix) {
		return ""
	}
	if parsed.RawQuery != "" {
		return canonicalPath + "?" + parsed.RawQuery
	}
	return canonicalPath
}

// hasDotSegment rejects traversal-style paths before redirect canonicalization.
func hasDotSegment(rawPath string) bool {
	for _, part := range strings.Split(rawPath, "/") {
		if part == "." || part == ".." {
			return true
		}
	}
	return false
}

// hasEncodedSlash blocks encoded path separators that could bypass prefix checks.
func hasEncodedSlash(rawPath string) bool {
	lower := strings.ToLower(rawPath)
	return strings.Contains(lower, "%2f") || strings.Contains(lower, "%5c")
}

// ensureLeadingSlash keeps accepted redirect targets path-absolute.
func ensureLeadingSlash(pathValue string) string {
	pathValue = strings.TrimSpace(pathValue)
	if pathValue == "" {
		return "/"
	}
	if strings.HasPrefix(pathValue, "/") {
		return pathValue
	}
	return "/" + pathValue
}
