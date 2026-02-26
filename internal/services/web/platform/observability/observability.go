// Package observability provides request middleware for web telemetry.
package observability

import (
	"log"
	"net/http"
	"strings"
	"time"
)

type responseMetricsWriter struct {
	http.ResponseWriter
	statusCode int
	bytes      int
}

func (w *responseMetricsWriter) WriteHeader(statusCode int) {
	if w.statusCode == 0 {
		w.statusCode = statusCode
	}
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseMetricsWriter) Write(payload []byte) (int, error) {
	if w.statusCode == 0 {
		w.statusCode = http.StatusOK
	}
	n, err := w.ResponseWriter.Write(payload)
	w.bytes += n
	return n, err
}

func (w *responseMetricsWriter) StatusCode() int {
	if w.statusCode == 0 {
		return http.StatusOK
	}
	return w.statusCode
}

// RequestLogger logs method, path, and latency for each request.
func RequestLogger(logger *log.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}
		if logger == nil {
			logger = log.Default()
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
			if requestID == "" {
				requestID = "-"
			}
			writer := &responseMetricsWriter{ResponseWriter: w}
			start := time.Now()
			next.ServeHTTP(writer, r)
			logger.Printf(
				"method=%s path=%s status=%d bytes=%d request_id=%s latency=%s",
				r.Method,
				r.URL.Path,
				writer.StatusCode(),
				writer.bytes,
				requestID,
				time.Since(start).String(),
			)
		})
	}
}
