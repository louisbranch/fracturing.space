package scenarios

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// testUnavailableConn implements grpc.ClientConnInterface and returns
// codes.Unavailable for every RPC, simulating a disconnected backend.
type testUnavailableConn struct{}

func (testUnavailableConn) Invoke(context.Context, string, any, any, ...grpc.CallOption) error {
	return status.Error(codes.Unavailable, "test: service not connected")
}

func (testUnavailableConn) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, status.Error(codes.Unavailable, "test: service not connected")
}

func TestScenarioServiceHandlersWithUnavailableClients(t *testing.T) {
	var conn testUnavailableConn
	svcIface := NewHandlers(modulehandler.NewBase(), "localhost:8080", statev1.NewEventServiceClient(conn), statev1.NewCampaignServiceClient(conn))
	svc, ok := svcIface.(*handlers)
	if !ok {
		t.Fatalf("NewHandlers() type = %T, want *handlers", svcIface)
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
	var conn testUnavailableConn
	svc := &handlers{
		base:           modulehandler.NewBase(),
		eventClient:    statev1.NewEventServiceClient(conn),
		campaignClient: statev1.NewCampaignServiceClient(conn),
	}

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
	var conn testUnavailableConn
	svc := &handlers{
		base:           modulehandler.NewBase(),
		campaignClient: statev1.NewCampaignServiceClient(conn),
	}
	loc := i18nhttp.Printer(i18nhttp.Default())
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := svc.getCampaignName(req, "camp-1", loc); got == "" {
		t.Fatal("getCampaignName() returned empty fallback")
	}
}
