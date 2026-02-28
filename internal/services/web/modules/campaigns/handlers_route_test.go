package campaigns

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// --- handleIndex ---

func TestHandleIndexRendersConfiguredCampaigns(t *testing.T) {
	t.Parallel()

	gw := fakeGateway{items: []CampaignSummary{
		{ID: "c1", Name: "Remote Stronghold"},
		{ID: "c2", Name: "Moonrise"},
	}}
	h := newTestHandlers(gw)
	mux := http.NewServeMux()
	registerStableRoutes(mux, h)

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaigns, nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	for _, want := range []string{"Remote Stronghold", "Moonrise"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body missing campaign name %q", want)
		}
	}
}

func TestHandleIndexReturnsErrorWhenGatewayFails(t *testing.T) {
	t.Parallel()

	gw := fakeGateway{err: apperrors.E(apperrors.KindUnavailable, "gateway down")}
	h := newTestHandlers(gw)
	mux := http.NewServeMux()
	registerStableRoutes(mux, h)

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaigns, nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

func TestHandleIndexHTMXRequestReturnsPartialResponse(t *testing.T) {
	t.Parallel()

	gw := fakeGateway{items: []CampaignSummary{{ID: "c1", Name: "Partial"}}}
	h := newTestHandlers(gw)
	mux := http.NewServeMux()
	registerStableRoutes(mux, h)

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaigns, nil)
	req.Header.Set("HX-Request", "true")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if strings.Contains(body, "<html") {
		t.Fatalf("HTMX response should not contain full HTML shell")
	}
	if !strings.Contains(body, "Partial") {
		t.Fatalf("body missing campaign name in partial response")
	}
}

// --- handleOverview ---

func TestHandleOverviewRendersWorkspace(t *testing.T) {
	t.Parallel()

	gw := fakeGateway{
		items:           []CampaignSummary{{ID: "c1", Name: "Remote"}},
		workspaceSystem: "Daggerheart",
	}
	h := newTestHandlers(gw)
	mux := http.NewServeMux()
	registerStableRoutes(mux, h)

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaign("c1"), nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, markerOverview) {
		t.Fatalf("body missing overview marker %q", markerOverview)
	}
}

func TestHandleOverviewReturnsNotFoundWhenWorkspaceLookupFails(t *testing.T) {
	t.Parallel()

	// CampaignWorkspace returns KindNotFound when the campaign ID isn't in items.
	gw := fakeGateway{}
	h := newTestHandlers(gw)
	mux := http.NewServeMux()
	registerStableRoutes(mux, h)

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaign("missing"), nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

// --- handleParticipants ---

func TestHandleParticipantsReturnsErrorWhenLookupFails(t *testing.T) {
	t.Parallel()

	gw := fakeGateway{
		items:           []CampaignSummary{{ID: "c1", Name: "Remote"}},
		participantsErr: apperrors.E(apperrors.KindUnavailable, "participants down"),
	}
	h := newTestHandlers(gw)
	mux := http.NewServeMux()
	registerStableRoutes(mux, h)

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignParticipants("c1"), nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

// --- handleCharacters ---

func TestHandleCharactersReturnsErrorWhenLookupFails(t *testing.T) {
	t.Parallel()

	gw := fakeGateway{
		items:         []CampaignSummary{{ID: "c1", Name: "Remote"}},
		charactersErr: apperrors.E(apperrors.KindUnavailable, "characters down"),
	}
	h := newTestHandlers(gw)
	mux := http.NewServeMux()
	registerStableRoutes(mux, h)

	req := httptest.NewRequest(http.MethodGet, routepath.AppCampaignCharacters("c1"), nil)
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusServiceUnavailable)
	}
}

// --- withCampaignID ---

func TestWithCampaignIDReturnsNotFoundForEmptyPath(t *testing.T) {
	t.Parallel()

	h := newTestHandlers(fakeGateway{})
	called := false
	handler := h.withCampaignID(func(w http.ResponseWriter, r *http.Request, id string) {
		called = true
	})

	// ServeMux pattern "{campaignID}" won't match empty, so test directly
	// by calling the handler with a request that has no campaignID path value.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if called {
		t.Fatalf("expected delegate not to be called for empty campaign ID")
	}
	if rr.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusNotFound)
	}
}

// --- helpers ---

func newTestHandlers(gw fakeGateway) handlers {
	return newHandlers(newService(gw), modulehandler.NewTestBase(), "")
}
