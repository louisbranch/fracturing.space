package admin

import (
	"net/http"
	"net/http/httptest"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc"
)

func TestNewHandlerRedirectsRootWithoutAuth(t *testing.T) {
	t.Parallel()

	handler := NewHandler(nil)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)

	handler.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", recorder.Code, http.StatusFound)
	}
	if got := recorder.Header().Get("Location"); got != "/app/dashboard" {
		t.Fatalf("Location = %q, want %q", got, "/app/dashboard")
	}
}

func TestNewServiceHandlerConfiguresIntrospectorOnlyWhenAuthComplete(t *testing.T) {
	t.Parallel()

	withAuth := NewServiceHandler(nil, " 127.0.0.1:7777 ", &AuthConfig{
		IntrospectURL:  "https://auth.example.com/introspect",
		LoginURL:       "https://auth.example.com/login",
		ResourceSecret: "secret",
	}, nil)
	if withAuth.grpcAddr != "127.0.0.1:7777" {
		t.Fatalf("grpcAddr = %q, want trimmed value", withAuth.grpcAddr)
	}
	if withAuth.introspector == nil {
		t.Fatal("introspector = nil, want configured introspector")
	}

	withoutAuth := NewServiceHandler(nil, "", &AuthConfig{
		IntrospectURL: "https://auth.example.com/introspect",
	}, nil)
	if withoutAuth.introspector != nil {
		t.Fatalf("introspector = %#v, want nil for partial auth config", withoutAuth.introspector)
	}
}

func TestModuleBuildInputCopiesServerClients(t *testing.T) {
	t.Parallel()

	conn := &grpc.ClientConn{}
	statusClient := statusv1.NewStatusServiceClient(conn)
	handler := NewServiceHandler(&Server{
		authClient:        authv1.NewAuthServiceClient(conn),
		campaignClient:    statev1.NewCampaignServiceClient(conn),
		characterClient:   statev1.NewCharacterServiceClient(conn),
		participantClient: statev1.NewParticipantServiceClient(conn),
		inviteClient:      nil,
		sessionClient:     statev1.NewSessionServiceClient(conn),
		eventClient:       statev1.NewEventServiceClient(conn),
		statisticsClient:  statev1.NewStatisticsServiceClient(conn),
		systemClient:      statev1.NewSystemServiceClient(conn),
		contentClient:     daggerheartv1.NewDaggerheartContentServiceClient(conn),
	}, "127.0.0.1:7777", nil, statusClient)

	input := handler.moduleBuildInput()
	if input.GRPCAddr != "127.0.0.1:7777" {
		t.Fatalf("GRPCAddr = %q, want %q", input.GRPCAddr, "127.0.0.1:7777")
	}
	if input.StatusClient != statusClient || input.AuthClient == nil || input.CampaignClient == nil || input.DaggerheartContentClient == nil {
		t.Fatalf("moduleBuildInput() = %#v, want copied clients", input)
	}
}
