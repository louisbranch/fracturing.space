package discovery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	discoveryapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/discovery/app"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/publichandler"
)

type serviceStub struct {
	page   discoveryapp.Page
	called bool
}

func (s *serviceStub) LoadPage(context.Context) discoveryapp.Page {
	s.called = true
	return s.page
}

func TestHandleIndexRendersDiscoveryPageForDegradedServiceState(t *testing.T) {
	t.Parallel()

	svc := &serviceStub{page: discoveryapp.Page{Status: discoveryapp.PageStatusUnavailable}}
	h := newHandlers(publichandler.NewBase(), svc)

	req := httptest.NewRequest(http.MethodGet, "/discover", nil)
	rr := httptest.NewRecorder()
	h.handleIndex(rr, req)

	if !svc.called {
		t.Fatal("expected service to be called")
	}
	if rr.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rr.Code, http.StatusOK)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "discover-root") {
		t.Fatalf("body missing discovery marker: %q", body)
	}
	if !strings.Contains(body, `data-discovery-status="unavailable"`) {
		t.Fatalf("body missing unavailable discovery status: %q", body)
	}
	if !strings.Contains(body, `data-discovery-unavailable="true"`) {
		t.Fatalf("body missing unavailable notice: %q", body)
	}
}

func TestHandleIndexMapsServiceEntries(t *testing.T) {
	t.Parallel()

	svc := &serviceStub{
		page: discoveryapp.Page{
			Status: discoveryapp.PageStatusReady,
			Entries: []discoveryapp.StarterEntry{{
				EntryID:     "starter:one",
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
		},
	}
	h := newHandlers(publichandler.NewBase(), svc)

	req := httptest.NewRequest(http.MethodGet, "/discover", nil)
	rr := httptest.NewRecorder()
	h.handleIndex(rr, req)

	if !svc.called {
		t.Fatal("expected service to be called")
	}
	body := rr.Body.String()
	if !strings.Contains(body, "Starter One") {
		t.Fatalf("body missing mapped entry: %q", body)
	}
	if !strings.Contains(body, "View Campaign") {
		t.Fatalf("body missing localized starter CTA: %q", body)
	}
	if !strings.Contains(body, `data-discovery-status="ready"`) {
		t.Fatalf("body missing ready discovery status: %q", body)
	}
}
