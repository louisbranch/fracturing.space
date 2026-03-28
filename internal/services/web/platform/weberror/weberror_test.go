package weberror

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	platformerrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	webi18n "github.com/louisbranch/fracturing.space/internal/services/web/platform/i18n"
	"golang.org/x/text/message"
)

func TestWriteModuleErrorRendersAppErrorPageForNotFound(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/missing", nil)
	rr := httptest.NewRecorder()
	WriteModuleError(rr, req, apperrors.E(apperrors.KindNotFound, "missing"), nil)
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	if body := rr.Body.String(); !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
}

func TestWriteModuleErrorRendersStyledPageForBadRequest(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	rr := httptest.NewRecorder()
	WriteModuleError(rr, req, apperrors.E(apperrors.KindInvalidInput, "bad form"), nil)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	// Invariant: user-facing transport errors must not leak raw internal strings.
	if strings.Contains(body, "bad form") {
		t.Fatalf("body leaked internal error text: %q", body)
	}
}

func TestWriteModuleErrorRendersRichTransportMessage(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/c1/invites", nil)
	rr := httptest.NewRecorder()
	WriteModuleError(rr, req, apperrors.Error{
		Kind:                apperrors.KindConflict,
		Message:             "failed to create invite",
		PublicMessage:       "Mensagem em portugues",
		PublicMessageLocale: "pt-BR",
		ReasonDomain:        platformerrors.Domain,
		Reason:              string(platformerrors.CodeInviteRecipientAlreadyInvited),
	}, nil)
	if rr.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusConflict)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "User already has a pending invite in this campaign") {
		t.Fatalf("body missing rich transport message: %q", body)
	}
	if strings.Contains(body, "failed to create invite") {
		t.Fatalf("body leaked generic fallback text: %q", body)
	}
}

func TestPublicMessageUsesLocalizedKeyWhenAvailable(t *testing.T) {
	t.Parallel()

	loc := stubLocalizer{"web.error.invalid": "Localized invalid request"}
	err := apperrors.EK(apperrors.KindInvalidInput, "web.error.invalid", "unsafe detail")
	if got := PublicMessage(loc, err); got != "Localized invalid request" {
		t.Fatalf("PublicMessage() = %q, want %q", got, "Localized invalid request")
	}
}

func TestPublicMessageReturnsEmptyForNilError(t *testing.T) {
	t.Parallel()

	if got := PublicMessage(nil, nil); got != "" {
		t.Fatalf("PublicMessage(nil, nil) = %q, want empty", got)
	}
}

func TestPublicMessageFallsBackToHTTPStatusText(t *testing.T) {
	t.Parallel()

	err := apperrors.E(apperrors.KindUnavailable, "backend timed out")
	if got := PublicMessage(nil, err); got != http.StatusText(http.StatusServiceUnavailable) {
		t.Fatalf("PublicMessage() = %q, want %q", got, http.StatusText(http.StatusServiceUnavailable))
	}
}

func TestPublicMessageFallsBackWhenLocalizationBlank(t *testing.T) {
	t.Parallel()

	loc := stubLocalizer{"web.error.untranslated": "   "}
	err := apperrors.EK(apperrors.KindUnknown, "web.error.untranslated", "unsafe detail")
	if got := PublicMessage(loc, err); got != http.StatusText(http.StatusInternalServerError) {
		t.Fatalf("PublicMessage() = %q, want %q", got, http.StatusText(http.StatusInternalServerError))
	}
}

func TestPublicMessageUsesRichPlatformReasonWhenAvailable(t *testing.T) {
	t.Parallel()

	err := apperrors.Error{
		Kind:                apperrors.KindConflict,
		Message:             "failed to create invite",
		PublicMessage:       "Mensagem em portugues",
		PublicMessageLocale: "pt-BR",
		ReasonDomain:        platformerrors.Domain,
		Reason:              string(platformerrors.CodeInviteRecipientAlreadyInvited),
	}
	if got := PublicMessage(nil, err, "en-US"); got != "User already has a pending invite in this campaign" {
		t.Fatalf("PublicMessage(rich reason) = %q", got)
	}
}

func TestWriteAppStatusErrorHTMXRendersFragment(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/campaigns/missing", nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	WriteAppStatusError(rr, req, http.StatusNotFound, nil)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error marker: %q", body)
	}
	// Invariant: HTMX responses must return fragments rather than full HTML documents.
	if strings.Contains(strings.ToLower(body), "<!doctype html") || strings.Contains(strings.ToLower(body), "<html") {
		t.Fatalf("expected HTMX fragment without full HTML document")
	}
}

func TestWriteAppStatusErrorPreservesBadRequestStatus(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
	rr := httptest.NewRecorder()
	WriteAppStatusError(rr, req, http.StatusBadRequest, nil)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
}

func TestWriteAppStatusErrorUsesResolverForViewerAndLanguage(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
	rr := httptest.NewRecorder()
	resolver := &stubResolver{}
	WriteAppStatusError(rr, req, http.StatusInternalServerError, resolver)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusInternalServerError)
	}
	if resolver.viewerCalls != 1 {
		t.Fatalf("resolver ResolveRequestViewer call count = %d, want 1", resolver.viewerCalls)
	}
	if resolver.languageCalls != 1 {
		t.Fatalf("resolver ResolveRequestLanguage call count = %d, want 1", resolver.languageCalls)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `lang="pt-BR"`) {
		t.Fatalf("body missing resolver language marker: %q", body)
	}
}

func TestWriteAppStatusErrorAllowsNilWriter(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/dashboard", nil)
	WriteAppStatusError(nil, req, http.StatusInternalServerError, nil)
}

func TestWritePublicStatusErrorRendersPublicShell(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/discover", nil)
	rr := httptest.NewRecorder()
	WritePublicStatusError(rr, req, http.StatusNotFound)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="auth-shell"`) {
		t.Fatalf("body missing public auth shell: %q", body)
	}
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error marker: %q", body)
	}
}

func TestWritePublicStatusErrorPreservesBadRequestStatus(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/discover", nil)
	rr := httptest.NewRecorder()
	WritePublicStatusError(rr, req, http.StatusBadRequest)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
}

func TestWritePublicStatusErrorAllowsNilWriter(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/discover", nil)
	WritePublicStatusError(nil, req, http.StatusNotFound)
}

func TestWritePublicErrorRendersAppErrorForNotFound(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/u/missing", nil)
	rr := httptest.NewRecorder()
	WritePublicError(rr, req, apperrors.E(apperrors.KindNotFound, "missing profile"))

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
	body := rr.Body.String()
	if strings.Contains(body, "missing profile") {
		t.Fatalf("body leaked internal error text: %q", body)
	}
}

func TestWritePublicErrorRendersStyledPageForBadRequest(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodPost, "/passkeys/login/start", nil)
	rr := httptest.NewRecorder()
	WritePublicError(rr, req, apperrors.E(apperrors.KindInvalidInput, "unsafe parser detail"))

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusBadRequest)
	}
	body := rr.Body.String()
	if !strings.Contains(body, `id="app-error-state"`) {
		t.Fatalf("body missing app error state marker: %q", body)
	}
	if strings.Contains(body, "unsafe parser detail") {
		t.Fatalf("body leaked internal error text: %q", body)
	}
}

func TestWritePublicErrorAllowsNilWriter(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/discover", nil)
	WritePublicError(nil, req, apperrors.E(apperrors.KindInvalidInput, "invalid"))
}

func TestWriteModuleErrorAllowsNilWriter(t *testing.T) {
	t.Parallel()

	req := httptest.NewRequest(http.MethodGet, "/app/settings/profile", nil)
	WriteModuleError(nil, req, apperrors.E(apperrors.KindInvalidInput, "invalid"), nil)
}

func TestShouldRenderAppError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status int
		want   bool
	}{
		{http.StatusOK, false},
		{http.StatusMovedPermanently, false},
		{http.StatusBadRequest, true},
		{http.StatusForbidden, true},
		{http.StatusNotFound, true},
		{http.StatusConflict, true},
		{http.StatusInternalServerError, true},
		{http.StatusBadGateway, true},
		{http.StatusServiceUnavailable, true},
	}
	for _, tc := range tests {
		if got := ShouldRenderAppError(tc.status); got != tc.want {
			t.Errorf("ShouldRenderAppError(%d) = %v, want %v", tc.status, got, tc.want)
		}
	}
}

type stubLocalizer map[string]string

func (s stubLocalizer) Sprintf(key message.Reference, _ ...any) string {
	resolved := fmt.Sprint(key)
	if translated, ok := s[resolved]; ok {
		return translated
	}
	return resolved
}

var _ webi18n.Localizer = stubLocalizer{}

type stubResolver struct {
	viewerCalls   int
	languageCalls int
}

func (r *stubResolver) ResolveRequestViewer(*http.Request) module.Viewer {
	r.viewerCalls++
	return module.Viewer{DisplayName: "Rhea"}
}

func (r *stubResolver) ResolveRequestLanguage(*http.Request) string {
	r.languageCalls++
	return "pt-BR"
}
