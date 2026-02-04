package web

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestCampaignPageRendering verifies layout rendering based on HTMX requests.
func TestCampaignPageRendering(t *testing.T) {
	handler := NewHandler(nil)

	tests := []struct {
		name        string
		path        string
		htmx        bool
		contains    []string
		notContains []string
	}{
		{
			name: "campaigns full page",
			path: "/campaigns",
			contains: []string{
				"<!doctype html>",
				"<h1>Duality Engine</h1>",
				"<h2>Campaigns</h2>",
			},
		},
		{
			name: "campaigns htmx",
			path: "/campaigns",
			htmx: true,
			contains: []string{
				"<h2>Campaigns</h2>",
			},
			notContains: []string{
				"<!doctype html>",
				"<h1>Duality Engine</h1>",
				"<html",
			},
		},
		{
			name: "campaign detail full page",
			path: "/campaigns/camp-123",
			contains: []string{
				"<!doctype html>",
				"<h1>Duality Engine</h1>",
				"Campaign service unavailable.",
				"<h2>Campaign</h2>",
			},
		},
		{
			name: "campaign detail htmx",
			path: "/campaigns/camp-123",
			htmx: true,
			contains: []string{
				"Campaign service unavailable.",
				"<h2>Campaign</h2>",
			},
			notContains: []string{
				"<!doctype html>",
				"<h1>Duality Engine</h1>",
				"<html",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "http://example.com"+tc.path, nil)
			if tc.htmx {
				req.Header.Set("HX-Request", "true")
			}
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, req)

			if recorder.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, recorder.Code)
			}

			body := recorder.Body.String()
			for _, expected := range tc.contains {
				assertContains(t, body, expected)
			}
			for _, unexpected := range tc.notContains {
				assertNotContains(t, body, unexpected)
			}
		})
	}
}

// assertContains fails the test when the body lacks the expected fragment.
func assertContains(t *testing.T, body string, expected string) {
	t.Helper()
	if !strings.Contains(body, expected) {
		t.Fatalf("expected response to contain %q", expected)
	}
}

// assertNotContains fails the test when the body includes an unexpected fragment.
func assertNotContains(t *testing.T, body string, unexpected string) {
	t.Helper()
	if strings.Contains(body, unexpected) {
		t.Fatalf("expected response to not contain %q", unexpected)
	}
}
