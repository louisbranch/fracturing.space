package profile

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	module "github.com/louisbranch/fracturing.space/internal/services/web2/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web2/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web2/routepath"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestMountServesProfileGet(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{summary: ProfileSummary{DisplayName: "Astra", Username: "astra"}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.ProfilePrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	if got := rr.Header().Get("Content-Type"); got != "text/html; charset=utf-8" {
		t.Fatalf("content-type = %q, want %q", got, "text/html; charset=utf-8")
	}
	if body := rr.Body.String(); !strings.Contains(body, "web2-scaffold-page") || !strings.Contains(body, "profile-root") {
		t.Fatalf("body = %q, want minimal scaffold profile page", body)
	}
}

func TestMountServesProfileHead(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{summary: ProfileSummary{DisplayName: "Astra", Username: "astra"}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodHead, routepath.AppProfile, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
}

func TestModuleIDReturnsProfile(t *testing.T) {
	t.Parallel()

	if got := New().ID(); got != "profile" {
		t.Fatalf("ID() = %q, want %q", got, "profile")
	}
}

func TestMountReturnsServiceUnavailableWhenGatewayNotConfigured(t *testing.T) {
	t.Parallel()

	m := New()
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.ProfilePrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	// Invariant: default module wiring must fail closed when profile backend is absent.
	if strings.Contains(body, "profile-root") {
		t.Fatalf("body unexpectedly rendered profile scaffold without backend: %q", body)
	}
}

func TestMountRejectsProfileNonGet(t *testing.T) {
	t.Parallel()

	m := New()
	mount, _ := m.Mount(module.Dependencies{})
	req := httptest.NewRequest(http.MethodPut, routepath.ProfilePrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusMethodNotAllowed)
	}
}

func TestMountMapsProfileGatewayErrorToHTTPStatus(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{err: apperrors.E(apperrors.KindUnauthorized, "missing session")})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.ProfilePrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusUnauthorized)
	}
}

func TestMountProfileGRPCNotFoundRendersAppErrorPage(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{err: status.Error(codes.NotFound, "user profile not found")})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.ProfilePrefix, nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	// Invariant: backend transport errors must never leak raw gRPC strings to user-facing pages.
	if strings.Contains(body, "rpc error:") {
		t.Fatalf("body leaked raw grpc error: %q", body)
	}
}

func TestMountProfileGRPCNotFoundHTMXRendersErrorFragment(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{err: status.Error(codes.NotFound, "user profile not found")})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.ProfilePrefix, nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	// Invariant: HTMX failures must swap a fragment and not a full document.
	if strings.Contains(strings.ToLower(body), "<!doctype html") || strings.Contains(strings.ToLower(body), "<html") {
		t.Fatalf("expected htmx error fragment without document wrapper")
	}
}

func TestMountProfileHTMXReturnsFragmentWithoutDocumentWrapper(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{summary: ProfileSummary{DisplayName: "Astra", Username: "astra"}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.ProfilePrefix, nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "profile-root") {
		t.Fatalf("body = %q, want profile marker", body)
	}
	// Invariant: HTMX requests must receive partial content, never a full document envelope.
	if strings.Contains(strings.ToLower(body), "<!doctype html") || strings.Contains(strings.ToLower(body), "<html") {
		t.Fatalf("expected htmx fragment without document wrapper")
	}
}

func TestMountProfileUnknownSubpathRendersSharedNotFoundPage(t *testing.T) {
	t.Parallel()

	m := NewWithGateway(fakeGateway{summary: ProfileSummary{DisplayName: "Astra", Username: "astra"}})
	mount, err := m.Mount(module.Dependencies{})
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	req := httptest.NewRequest(http.MethodGet, routepath.ProfilePrefix+"other", nil)
	rr := httptest.NewRecorder()
	mount.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	// Invariant: unknown app routes should use the shared not-found page, not net/http plain text.
	if strings.Contains(body, "404 page not found") {
		t.Fatalf("body unexpectedly rendered plain 404 text: %q", body)
	}
}

type fakeGateway struct {
	summary ProfileSummary
	err     error
}

func (f fakeGateway) LoadProfile(context.Context) (ProfileSummary, error) {
	if f.err != nil {
		return ProfileSummary{}, f.err
	}
	if f.summary.DisplayName == "" {
		return ProfileSummary{}, errors.New("missing profile")
	}
	return f.summary, nil
}
