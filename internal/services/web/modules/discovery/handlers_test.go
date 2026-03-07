package discovery

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
)

type listGatewayStub struct {
	entries []StarterEntry
	err     error
	called  bool
}

func (s *listGatewayStub) ListStarterEntries(context.Context) ([]StarterEntry, error) {
	s.called = true
	return s.entries, s.err
}

func TestLoadStarterEntriesViewSoftDegradesOnGatewayError(t *testing.T) {
	t.Parallel()

	gw := &listGatewayStub{err: errors.New("boom")}
	h := newHandlers(publichandler.NewBase(), gw)

	got := h.loadStarterEntriesView(context.Background())
	if !gw.called {
		t.Fatalf("expected gateway to be called")
	}
	if got != nil {
		t.Fatalf("loadStarterEntriesView() = %v, want nil on soft degrade", got)
	}
}

func TestLoadStarterEntriesViewMapsGatewayResults(t *testing.T) {
	t.Parallel()

	gw := &listGatewayStub{
		entries: []StarterEntry{{
			CampaignID:  "c1",
			Title:       "Starter One",
			Description: "A first step",
			Tags:        []string{"beginner"},
			Difficulty:  "Beginner",
			Duration:    "2 sessions",
			GmMode:      "AI",
			System:      "Daggerheart",
			Level:       1,
			Players:     "2-4",
		}},
	}
	h := newHandlers(publichandler.NewBase(), gw)

	got := h.loadStarterEntriesView(context.Background())
	if len(got) != 1 {
		t.Fatalf("len(loadStarterEntriesView()) = %d, want 1", len(got))
	}
	if got[0].CampaignID != "c1" {
		t.Fatalf("CampaignID = %q, want %q", got[0].CampaignID, "c1")
	}
	if got[0].Title != "Starter One" {
		t.Fatalf("Title = %q, want %q", got[0].Title, "Starter One")
	}
}

func TestHandleIndexSoftDegradesAndRendersDiscoveryPage(t *testing.T) {
	t.Parallel()

	h := newHandlers(publichandler.NewBase(), &listGatewayStub{err: errors.New("backend down")})

	req := httptest.NewRequest(http.MethodGet, "/discover", nil)
	rr := httptest.NewRecorder()
	h.handleIndex(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "discover-root") {
		t.Fatalf("body missing discovery marker: %q", body)
	}
}
