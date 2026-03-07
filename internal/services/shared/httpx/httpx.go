// Package httpx provides shared HTTP middleware used by web and admin services.
package httpx

import (
	"fmt"
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"sync/atomic"
	"time"
)

// Middleware wraps an HTTP handler.
type Middleware func(http.Handler) http.Handler

var requestIDCounter atomic.Uint64

// Chain applies middleware in declaration order.
func Chain(handler http.Handler, middleware ...Middleware) http.Handler {
	if handler == nil {
		handler = http.NotFoundHandler()
	}
	wrapped := handler
	for idx := len(middleware) - 1; idx >= 0; idx-- {
		if middleware[idx] == nil {
			continue
		}
		wrapped = middleware[idx](wrapped)
	}
	return wrapped
}

// RequestID injects and echoes a request id for correlation.
func RequestID(prefix string) Middleware {
	if prefix == "" {
		prefix = "req"
	}
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), requestIDCounter.Add(1))
				r.Header.Set("X-Request-ID", requestID)
			}
			w.Header().Set("X-Request-ID", requestID)
			next.ServeHTTP(w, r)
		})
	}
}

// RecoverPanic converts panics into HTTP 500 responses with structured logging.
func RecoverPanic() Middleware {
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					path := "-"
					method := "-"
					requestID := "-"
					if r != nil {
						path = strings.TrimSpace(r.URL.Path)
						method = strings.TrimSpace(r.Method)
						if rid := strings.TrimSpace(r.Header.Get("X-Request-ID")); rid != "" {
							requestID = rid
						}
					}
					log.Printf(
						"panic recovered method=%s path=%s request_id=%s panic=%v stack=%s",
						method,
						path,
						requestID,
						recovered,
						strings.TrimSpace(string(debug.Stack())),
					)
					w.WriteHeader(http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}
