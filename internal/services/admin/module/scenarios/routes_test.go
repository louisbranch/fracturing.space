package scenarios

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeService struct {
	lastCall     string
	lastCampaign string
}

func (f *fakeService) HandleScenarios(http.ResponseWriter, *http.Request) {
	f.lastCall = "scenarios_page"
}

func (f *fakeService) HandleScenarioEvents(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "scenarios_events"
	f.lastCampaign = campaignID
}

func (f *fakeService) HandleScenarioEventsTable(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "scenarios_events_table"
	f.lastCampaign = campaignID
}

func (f *fakeService) HandleScenarioTimelineTable(_ http.ResponseWriter, _ *http.Request, campaignID string) {
	f.lastCall = "scenarios_timeline_table"
	f.lastCampaign = campaignID
}

func TestRegisterRoutes(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	mux := http.NewServeMux()
	RegisterRoutes(mux, svc)

	tests := []struct {
		path         string
		wantCode     int
		wantCall     string
		wantCampaign string
	}{
		{path: "/scenarios", wantCode: http.StatusOK, wantCall: "scenarios_page"},
		{path: "/scenarios/camp-1/events", wantCode: http.StatusOK, wantCall: "scenarios_events", wantCampaign: "camp-1"},
		{path: "/scenarios/camp-1/events/table", wantCode: http.StatusOK, wantCall: "scenarios_events_table", wantCampaign: "camp-1"},
		{path: "/scenarios/camp-1/timeline/table", wantCode: http.StatusOK, wantCall: "scenarios_timeline_table", wantCampaign: "camp-1"},
		{path: "/scenarios/camp-1/unknown", wantCode: http.StatusNotFound},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.path, func(t *testing.T) {
			svc.lastCall = ""
			svc.lastCampaign = ""

			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, req)

			if rec.Code != tc.wantCode {
				t.Fatalf("status = %d, want %d", rec.Code, tc.wantCode)
			}
			if svc.lastCall != tc.wantCall {
				t.Fatalf("lastCall = %q, want %q", svc.lastCall, tc.wantCall)
			}
			if svc.lastCampaign != tc.wantCampaign {
				t.Fatalf("lastCampaign = %q, want %q", svc.lastCampaign, tc.wantCampaign)
			}
		})
	}
}

func TestHandleScenarioPathRedirectsTrailingSlash(t *testing.T) {
	t.Parallel()

	svc := &fakeService{}
	req := httptest.NewRequest(http.MethodGet, "/scenarios/camp-1/events/", nil)
	rec := httptest.NewRecorder()

	HandleScenarioPath(rec, req, svc)

	if rec.Code != http.StatusMovedPermanently {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusMovedPermanently)
	}
	if location := rec.Header().Get("Location"); location != "/scenarios/camp-1/events" {
		t.Fatalf("location = %q, want %q", location, "/scenarios/camp-1/events")
	}
}
