package scenarios

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/admin/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
)

func TestScenarioServiceHandlersWithNilClients(t *testing.T) {
	svcIface := NewService(modulehandler.NewBase(nil), "localhost:8080")
	svc, ok := svcIface.(*service)
	if !ok {
		t.Fatalf("NewService() type = %T, want *service", svcIface)
	}

	req := httptest.NewRequest(http.MethodGet, "/app/scenarios", nil)
	rec := httptest.NewRecorder()
	svc.HandleScenarios(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleScenarios(GET) status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodPut, "/app/scenarios", nil)
	rec = httptest.NewRecorder()
	svc.HandleScenarios(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("HandleScenarios(PUT) status = %d", rec.Code)
	}
	if allow := rec.Header().Get("Allow"); allow != "GET, POST" {
		t.Fatalf("HandleScenarios(PUT) Allow = %q", allow)
	}

	req = httptest.NewRequest(http.MethodPost, "/app/scenarios", nil)
	rec = httptest.NewRecorder()
	svc.HandleScenarios(rec, req)
	if rec.Code != http.StatusForbidden {
		t.Fatalf("HandleScenarios(POST no origin) status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/app/scenarios/camp-1/events", nil)
	rec = httptest.NewRecorder()
	svc.HandleScenarioEvents(rec, req, "camp-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleScenarioEvents(nil clients) status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/app/scenarios/camp-1/events?fragment=rows", nil)
	rec = httptest.NewRecorder()
	svc.HandleScenarioEventsTable(rec, req, "camp-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleScenarioEventsTable(nil event client) status = %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/app/scenarios/camp-1/timeline?fragment=rows", nil)
	rec = httptest.NewRecorder()
	svc.HandleScenarioTimelineTable(rec, req, "camp-1")
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleScenarioTimelineTable(nil event client) status = %d", rec.Code)
	}
}

func TestScenarioServiceRunScriptMkdirError(t *testing.T) {
	svc := &service{base: modulehandler.NewBase(nil)}

	file, err := os.CreateTemp("", "scenario-tmpfile-*")
	if err != nil {
		t.Fatalf("CreateTemp() error = %v", err)
	}
	defer func() {
		_ = os.Remove(file.Name())
	}()
	_ = file.Close()

	t.Setenv(scenarioTempDirEnv, file.Name())
	logs, campaignID, runErr := svc.runScenarioScript(httptest.NewRequest(http.MethodGet, "/", nil).Context(), "return Scenario.new('x')")
	if runErr == nil {
		t.Fatal("runScenarioScript() expected error when temp dir path is a file")
	}
	if logs != "" || campaignID != "" {
		t.Fatalf("runScenarioScript() = (%q,%q,%v)", logs, campaignID, runErr)
	}
}

func TestScenarioServiceGetCampaignNameFallback(t *testing.T) {
	svc := &service{base: modulehandler.NewBase(nil)}
	loc := i18n.Printer(i18n.Default())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := svc.getCampaignName(req, "camp-1", loc); got == "" {
		t.Fatal("getCampaignName() returned empty fallback")
	}
}
