// Package httpx provides HTTP middleware helpers used by web modules.
package httpx

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/shared/htmx"
	sharedhttpx "github.com/louisbranch/fracturing.space/internal/services/shared/httpx"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

const htmxRedirectHeader = "HX-Redirect"

// MethodNotAllowed writes a 405 response with an Allow header.
func MethodNotAllowed(allow string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		if w == nil {
			return
		}
		w.Header().Set("Allow", strings.TrimSpace(allow))
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

// RequireMethod rejects requests outside the allowed method.
func RequireMethod(method string) sharedhttpx.Middleware {
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != method {
				w.Header().Set("Allow", method)
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// WriteJSON writes a JSON response with the provided status code.
func WriteJSON(w http.ResponseWriter, status int, payload any) error {
	if w == nil {
		return fmt.Errorf("response writer is required")
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(payload)
}

// WriteJSONError writes a JSON error response with the given status code and message.
func WriteJSONError(w http.ResponseWriter, statusCode int, message string) error {
	return WriteJSON(w, statusCode, map[string]any{"error": message})
}

// WriteError writes an error response using typed web status mapping.
func WriteError(w http.ResponseWriter, err error) {
	if w == nil {
		return
	}
	if err == nil {
		w.WriteHeader(http.StatusOK)
		return
	}
	statusCode := apperrors.HTTPStatus(err)
	http.Error(w, strings.TrimSpace(http.StatusText(statusCode)), statusCode)
}

// RequestContext returns r.Context() with a nil-safe fallback to context.Background().
func RequestContext(r *http.Request) context.Context {
	if r == nil {
		return context.Background()
	}
	return r.Context()
}

// IsHTMXRequest delegates to the shared htmx package.
func IsHTMXRequest(r *http.Request) bool {
	return htmx.IsHTMXRequest(r)
}

// WriteHTML writes an HTML payload with the provided status code.
func WriteHTML(w http.ResponseWriter, status int, payload string) error {
	if w == nil {
		return fmt.Errorf("response writer is required")
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, err := io.WriteString(w, payload)
	return err
}

// WriteHXRedirect writes an HTMX redirect response header.
func WriteHXRedirect(w http.ResponseWriter, location string) {
	if w == nil {
		return
	}
	w.Header().Set(htmxRedirectHeader, location)
	w.WriteHeader(http.StatusOK)
}

// WriteRedirect writes an HTMX-aware redirect response.
func WriteRedirect(w http.ResponseWriter, r *http.Request, location string) {
	if w == nil {
		return
	}
	if IsHTMXRequest(r) {
		WriteHXRedirect(w, location)
		return
	}
	if r == nil {
		w.Header().Set("Location", location)
		w.WriteHeader(http.StatusFound)
		return
	}
	http.Redirect(w, r, location, http.StatusFound)
}
