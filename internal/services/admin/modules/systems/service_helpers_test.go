package systems

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
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

func TestSystemHelpersFormatAndParse(t *testing.T) {
	loc := i18nhttp.Printer(i18nhttp.Default())

	tests := []struct {
		input string
		want  commonv1.GameSystem
	}{
		{input: "", want: commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED},
		{input: "daggerheart", want: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART},
		{input: "DAGGERHEART", want: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART},
		{input: " GAME_SYSTEM_DAGGERHEART ", want: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART},
		{input: "missing", want: commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED},
	}
	for _, tc := range tests {
		if got := parseSystemID(tc.input); got != tc.want {
			t.Fatalf("parseSystemID(%q) = %v, want %v", tc.input, got, tc.want)
		}
	}

	if got := formatImplementationStage(commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_COMPLETE, loc); got == "" {
		t.Fatal("formatImplementationStage() returned empty label")
	}
	if got := formatImplementationStage(commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_UNSPECIFIED, loc); got != loc.Sprintf("label.unspecified") {
		t.Fatalf("formatImplementationStage(unspecified) = %q", got)
	}

	if got := formatOperationalStatus(commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL, loc); got == "" {
		t.Fatal("formatOperationalStatus() returned empty label")
	}
	if got := formatOperationalStatus(commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_UNSPECIFIED, loc); got != loc.Sprintf("label.unspecified") {
		t.Fatalf("formatOperationalStatus(unspecified) = %q", got)
	}

	if got := formatAccessLevel(commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_PUBLIC, loc); got == "" {
		t.Fatal("formatAccessLevel() returned empty label")
	}
	if got := formatAccessLevel(commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_UNSPECIFIED, loc); got != loc.Sprintf("label.unspecified") {
		t.Fatalf("formatAccessLevel(unspecified) = %q", got)
	}
}

func TestSystemHelpersBuilders(t *testing.T) {
	loc := i18nhttp.Printer(i18nhttp.Default())

	rows := buildSystemRows([]*statev1.GameSystemInfo{
		nil,
		{
			Id:                  commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			Name:                "Daggerheart",
			Version:             "1.0.0",
			ImplementationStage: commonv1.GameSystemImplementationStage_GAME_SYSTEM_IMPLEMENTATION_STAGE_COMPLETE,
			OperationalStatus:   commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL,
			AccessLevel:         commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_PUBLIC,
			IsDefault:           true,
		},
	}, loc)
	if len(rows) != 1 || rows[0].Name != "Daggerheart" {
		t.Fatalf("buildSystemRows() = %#v", rows)
	}
	if rows[0].DetailURL == "" {
		t.Fatalf("buildSystemRows() missing detail URL: %#v", rows[0])
	}

	detail := buildSystemDetail(&statev1.GameSystemInfo{
		Id:                commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		Name:              "Daggerheart",
		Version:           "1.0.0",
		OperationalStatus: commonv1.GameSystemOperationalStatus_GAME_SYSTEM_OPERATIONAL_STATUS_OPERATIONAL,
		AccessLevel:       commonv1.GameSystemAccessLevel_GAME_SYSTEM_ACCESS_LEVEL_PUBLIC,
		IsDefault:         true,
	}, loc)
	if detail.Name != "Daggerheart" || detail.ID == "" {
		t.Fatalf("buildSystemDetail() = %#v", detail)
	}
	if empty := buildSystemDetail(nil, loc); empty.ID != "" {
		t.Fatalf("buildSystemDetail(nil) = %#v", empty)
	}
}

func TestSystemServiceUnavailableClients(t *testing.T) {
	var conn testUnavailableConn
	svc := handlers{
		base:         modulehandler.NewBase(),
		systemClient: statev1.NewSystemServiceClient(conn),
	}

	rec := httptest.NewRecorder()
	svc.HandleSystemsPage(rec, httptest.NewRequest(http.MethodGet, "/app/systems", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleSystemsPage() status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	svc.HandleSystemsTable(rec, httptest.NewRequest(http.MethodGet, "/app/systems?fragment=rows", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleSystemsTable(nil client) status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	svc.HandleSystemDetail(rec, httptest.NewRequest(http.MethodGet, "/app/systems/daggerheart", nil), "daggerheart")
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleSystemDetail(nil client) status = %d", rec.Code)
	}
}
