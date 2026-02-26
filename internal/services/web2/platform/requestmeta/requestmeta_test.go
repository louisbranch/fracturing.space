package requestmeta

import (
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHasSameOriginProof(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		req  *http.Request
		want bool
	}{
		{
			name: "origin same host and scheme",
			req: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "https://app.example.test/app/campaigns/c1/sessions/start", nil)
				req.Host = "app.example.test"
				req.Header.Set("Origin", "https://app.example.test")
				return req
			}(),
			want: true,
		},
		{
			name: "referer same host and scheme",
			req: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "https://app.example.test/logout", nil)
				req.Host = "app.example.test"
				req.Header.Set("Referer", "https://app.example.test/app/settings")
				return req
			}(),
			want: true,
		},
		{
			name: "origin scheme mismatch",
			req: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "https://app.example.test/logout", nil)
				req.Host = "app.example.test"
				req.Header.Set("Origin", "http://app.example.test")
				return req
			}(),
			want: false,
		},
		{
			name: "origin missing non-default port",
			req: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "https://app.example.test:8443/logout", nil)
				req.Host = "app.example.test:8443"
				req.Header.Set("Origin", "https://app.example.test")
				return req
			}(),
			want: false,
		},
		{
			name: "missing origin and referer",
			req: func() *http.Request {
				req := httptest.NewRequest(http.MethodPost, "https://app.example.test/logout", nil)
				req.Host = "app.example.test"
				return req
			}(),
			want: false,
		},
		{
			name: "nil request",
			req:  nil,
			want: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := HasSameOriginProof(tc.req); got != tc.want {
				t.Fatalf("HasSameOriginProof() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsHTTPS(t *testing.T) {
	t.Parallel()

	if IsHTTPS(nil) {
		t.Fatalf("expected nil request to be non-https")
	}

	req := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	if IsHTTPS(req) {
		t.Fatalf("expected http URL to be non-https")
	}

	req = httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	if !IsHTTPS(req) {
		t.Fatalf("expected forwarded https request")
	}

	req = httptest.NewRequest(http.MethodGet, "/", nil)
	req.TLS = &tls.ConnectionState{}
	if !IsHTTPS(req) {
		t.Fatalf("expected TLS request to be https")
	}
}
