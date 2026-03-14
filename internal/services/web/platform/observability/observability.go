// Package observability provides request middleware for web telemetry.
package observability

import (
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// responseMetricsWriter defines an internal contract used at this web package boundary.
type responseMetricsWriter struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

// WriteHeader centralizes this web behavior in one helper seam.
func (w *responseMetricsWriter) WriteHeader(statusCode int) {
	if w.statusCode == 0 {
		w.statusCode = statusCode
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

// Write centralizes this web behavior in one helper seam.
func (w *responseMetricsWriter) Write(payload []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(payload)
	w.bytes += n
	return n, err
}

// StatusCode centralizes this web behavior in one helper seam.
func (w *responseMetricsWriter) StatusCode() int {
	if w.statusCode == 0 {
		return http.StatusOK
	}
	return w.statusCode
}

// RequestLogger logs method, path, and latency for each request.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}
		if logger == nil {
			logger = slog.Default()
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
			if requestID == "" {
				requestID = "-"
			}
			writer := &responseMetricsWriter{ResponseWriter: w}
			start := time.Now()
			next.ServeHTTP(writer, r)
			logger.Info(
				"http_request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", writer.StatusCode(),
				"bytes", writer.bytes,
				"request_id", requestID,
				"latency", time.Since(start),
			)
		})
	}
}
