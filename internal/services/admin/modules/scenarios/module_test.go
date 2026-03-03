package scenarios

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

type fakeService struct {
	lastCall     string
	lastCampaign string
	lastPath     string
}

func (f *fakeService) HandleScenarios(w http.ResponseWriter, r *http.Request) {
	f.lastCall = "scenarios_page"
	if r != nil && r.URL != nil {
		f.lastPath = r.URL.Path
	}
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleScenarioEventsTable(w http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "scenarios_events_table"
	f.lastCampaign = campaignID
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleScenarioTimelineTable(w http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "scenarios_timeline_table"
	f.lastCampaign = campaignID
	w.WriteHeader(http.StatusNoContent)
}

func (f *fakeService) HandleScenarioEvents(w http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "scenarios_events"
	f.lastCampaign = campaignID
	w.WriteHeader(http.StatusNoContent)
}

func TestMount(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	m, err := New(svc).Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}
	if m.Prefix != routepath.ScenariosPrefix {
		t.Fatalf("prefix = %q, want %q", m.Prefix, routepath.ScenariosPrefix)
	}

	tests := []struct {
		name         string
		method       string
		path         string
		wantCode     int
		wantCall     string
		wantCampaign string
		wantPath     string
		wantAllow    string
	}{
		{name: "page", method: http.MethodGet, path: "/app/scenarios", wantCode: http.StatusNoContent, wantCall: "scenarios_page", wantPath: "/app/scenarios"},
		{name: "run post", method: http.MethodPost, path: "/app/scenarios/run", wantCode: http.StatusNoContent, wantCall: "scenarios_page", wantPath: "/app/scenarios/run"},
		{name: "run method not allowed", method: http.MethodGet, path: "/app/scenarios/run", wantCode: http.StatusMethodNotAllowed, wantAllow: http.MethodPost},
		{name: "events table", method: http.MethodGet, path: "/app/scenarios/camp-1/events?fragment=rows", wantCode: http.StatusNoContent, wantCall: "scenarios_events_table", wantCampaign: "camp-1"},
		{name: "timeline table", method: http.MethodGet, path: "/app/scenarios/camp-1/timeline?fragment=rows", wantCode: http.StatusNoContent, wantCall: "scenarios_timeline_table", wantCampaign: "camp-1"},
		{name: "legacy table blocked", method: http.MethodGet, path: "/app/scenarios/camp-1/events/table", wantCode: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			svc.lastCall = ""
			svc.lastCampaign = ""
			svc.lastPath = ""
			req := httptest.NewRequest(tc.method, tc.path, nil)
			rec := httptest.NewRecorder()
			m.Handler.ServeHTTP(rec, req)
			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantCode)
			}
			if svc.lastCall != tc.wantCall {
				t.Fatalf("lastCall = %q, want %q", svc.lastCall, tc.wantCall)
			}
			if svc.lastCampaign != tc.wantCampaign {
				t.Fatalf("lastCampaign = %q, want %q", svc.lastCampaign, tc.wantCampaign)
			}
			if tc.wantPath != "" && svc.lastPath != tc.wantPath {
				t.Fatalf("lastPath = %q, want %q", svc.lastPath, tc.wantPath)
			}
			if tc.wantAllow != "" {
				if got := rec.Header().Get("Allow"); got != tc.wantAllow {
					t.Fatalf("Allow header = %q, want %q", got, tc.wantAllow)
				}
			}
		})
	}
}

func TestMountNilService(t *testing.T) {
	t.Parallel()

	m, err := New(nil).Mount()
	if err != nil {
		t.Fatalf("Mount() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/app/scenarios/camp-1/events?fragment=rows", nil)
	rec := httptest.NewRecorder()
	m.Handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
