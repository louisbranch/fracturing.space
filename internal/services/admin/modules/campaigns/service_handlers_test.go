package campaigns

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
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

func TestCampaignServiceHandlersWithUnavailableClients(t *testing.T) {
	var conn testUnavailableConn
	svcIface := NewHandlers(
		modulehandler.NewBase(),
		statev1.NewCampaignServiceClient(conn),
		statev1.NewCharacterServiceClient(conn),
		statev1.NewParticipantServiceClient(conn),
		statev1.NewInviteServiceClient(conn),
		statev1.NewSessionServiceClient(conn),
		statev1.NewEventServiceClient(conn),
		authv1.NewAuthServiceClient(conn),
	)
	svc, ok := svcIface.(*handlers)
	if !ok {
		t.Fatalf("NewHandlers() type = %T, want *handlers", svcIface)
	}

	run := func(name string, path string, fn func(http.ResponseWriter, *http.Request), wantStatus int) {
		t.Helper()
		t.Run(name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			rec := httptest.NewRecorder()
			fn(rec, req)
			if rec.Code != wantStatus {
				t.Fatalf("%s status = %d, want %d", name, rec.Code, wantStatus)
			}
		})
	}

	run("campaigns page", "/app/campaigns", svc.HandleCampaignsPage, http.StatusOK)
	run("campaigns table", "/app/campaigns?fragment=rows", svc.HandleCampaignsTable, http.StatusOK)
	run("campaign detail", "/app/campaigns/camp-1", func(w http.ResponseWriter, r *http.Request) {
		svc.HandleCampaignDetail(w, r, "camp-1")
	}, http.StatusOK)

	run("characters list", "/app/campaigns/camp-1/characters", func(w http.ResponseWriter, r *http.Request) {
		svc.HandleCharactersList(w, r, "camp-1")
	}, http.StatusOK)
	run("characters table", "/app/campaigns/camp-1/characters?fragment=rows", func(w http.ResponseWriter, r *http.Request) {
		svc.HandleCharactersTable(w, r, "camp-1")
	}, http.StatusOK)

	run("character sheet", "/app/campaigns/camp-1/characters/char-1", func(w http.ResponseWriter, r *http.Request) {
		svc.HandleCharacterSheet(w, r, "camp-1", "char-1")
	}, http.StatusNotFound)
	run("character activity", "/app/campaigns/camp-1/characters/char-1/activity", func(w http.ResponseWriter, r *http.Request) {
		svc.HandleCharacterActivity(w, r, "camp-1", "char-1")
	}, http.StatusNotFound)

	run("participants list", "/app/campaigns/camp-1/participants", func(w http.ResponseWriter, r *http.Request) {
		svc.HandleParticipantsList(w, r, "camp-1")
	}, http.StatusOK)
	run("participants table", "/app/campaigns/camp-1/participants?fragment=rows", func(w http.ResponseWriter, r *http.Request) {
		svc.HandleParticipantsTable(w, r, "camp-1")
	}, http.StatusOK)

	run("invites list", "/app/campaigns/camp-1/invites", func(w http.ResponseWriter, r *http.Request) {
		svc.HandleInvitesList(w, r, "camp-1")
	}, http.StatusOK)
	run("invites table", "/app/campaigns/camp-1/invites?fragment=rows", func(w http.ResponseWriter, r *http.Request) {
		svc.HandleInvitesTable(w, r, "camp-1")
	}, http.StatusOK)

	run("sessions list", "/app/campaigns/camp-1/sessions", func(w http.ResponseWriter, r *http.Request) {
		svc.HandleSessionsList(w, r, "camp-1")
	}, http.StatusOK)
	run("sessions table", "/app/campaigns/camp-1/sessions?fragment=rows", func(w http.ResponseWriter, r *http.Request) {
		svc.HandleSessionsTable(w, r, "camp-1")
	}, http.StatusOK)
	run("session detail", "/app/campaigns/camp-1/sessions/s-1", func(w http.ResponseWriter, r *http.Request) {
		svc.HandleSessionDetail(w, r, "camp-1", "s-1")
	}, http.StatusNotFound)

	run("session events", "/app/campaigns/camp-1/sessions/s-1/events", func(w http.ResponseWriter, r *http.Request) {
		svc.HandleSessionEvents(w, r, "camp-1", "s-1")
	}, http.StatusOK)
	run("event log", "/app/campaigns/camp-1/events", func(w http.ResponseWriter, r *http.Request) {
		svc.HandleEventLog(w, r, "camp-1")
	}, http.StatusOK)
	run("event log table", "/app/campaigns/camp-1/events?fragment=rows", func(w http.ResponseWriter, r *http.Request) {
		svc.HandleEventLogTable(w, r, "camp-1")
	}, http.StatusOK)
}

func TestCampaignServiceNameFallbacks(t *testing.T) {
	var conn testUnavailableConn
	svc := &handlers{
		base:           modulehandler.NewBase(),
		campaignClient: statev1.NewCampaignServiceClient(conn),
		sessionClient:  statev1.NewSessionServiceClient(conn),
	}
	loc := i18nhttp.Printer(i18nhttp.Default())
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	if got := svc.getCampaignName(req, "camp-1", loc); got == "" {
		t.Fatal("getCampaignName() returned empty fallback")
	}
	if got := svc.getSessionName(req, "camp-1", "session-1", loc); got == "" {
		t.Fatal("getSessionName() returned empty fallback")
	}
}

type fakeCampaignClient struct {
	statev1.CampaignServiceClient
	listResp *statev1.ListCampaignsResponse
	getResp  *statev1.GetCampaignResponse
}

func (c *fakeCampaignClient) ListCampaigns(context.Context, *statev1.ListCampaignsRequest, ...grpc.CallOption) (*statev1.ListCampaignsResponse, error) {
	return c.listResp, nil
}

func (c *fakeCampaignClient) GetCampaign(context.Context, *statev1.GetCampaignRequest, ...grpc.CallOption) (*statev1.GetCampaignResponse, error) {
	return c.getResp, nil
}

type fakeCharacterClient struct {
	statev1.CharacterServiceClient
	listResp  *statev1.ListCharactersResponse
	sheetResp *statev1.GetCharacterSheetResponse
}

func (c *fakeCharacterClient) ListCharacters(context.Context, *statev1.ListCharactersRequest, ...grpc.CallOption) (*statev1.ListCharactersResponse, error) {
	return c.listResp, nil
}

func (c *fakeCharacterClient) GetCharacterSheet(context.Context, *statev1.GetCharacterSheetRequest, ...grpc.CallOption) (*statev1.GetCharacterSheetResponse, error) {
	return c.sheetResp, nil
}

type fakeParticipantClient struct {
	statev1.ParticipantServiceClient
	listResp *statev1.ListParticipantsResponse
	getResp  *statev1.GetParticipantResponse
}

func (c *fakeParticipantClient) ListParticipants(context.Context, *statev1.ListParticipantsRequest, ...grpc.CallOption) (*statev1.ListParticipantsResponse, error) {
	return c.listResp, nil
}

func (c *fakeParticipantClient) GetParticipant(context.Context, *statev1.GetParticipantRequest, ...grpc.CallOption) (*statev1.GetParticipantResponse, error) {
	return c.getResp, nil
}

type fakeInviteClient struct {
	statev1.InviteServiceClient
	listResp *statev1.ListInvitesResponse
}

func (c *fakeInviteClient) ListInvites(context.Context, *statev1.ListInvitesRequest, ...grpc.CallOption) (*statev1.ListInvitesResponse, error) {
	return c.listResp, nil
}

type fakeSessionClient struct {
	statev1.SessionServiceClient
	listResp *statev1.ListSessionsResponse
	getResp  *statev1.GetSessionResponse
}

func (c *fakeSessionClient) ListSessions(context.Context, *statev1.ListSessionsRequest, ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
	return c.listResp, nil
}

func (c *fakeSessionClient) GetSession(context.Context, *statev1.GetSessionRequest, ...grpc.CallOption) (*statev1.GetSessionResponse, error) {
	return c.getResp, nil
}

type fakeEventClient struct {
	statev1.EventServiceClient
	listResp *statev1.ListEventsResponse
}

func (c *fakeEventClient) ListEvents(context.Context, *statev1.ListEventsRequest, ...grpc.CallOption) (*statev1.ListEventsResponse, error) {
	return c.listResp, nil
}

type fakeAuthClient struct {
	authv1.AuthServiceClient
	getUserResp *authv1.GetUserResponse
}

func (c *fakeAuthClient) GetUser(context.Context, *authv1.GetUserRequest, ...grpc.CallOption) (*authv1.GetUserResponse, error) {
	return c.getUserResp, nil
}

func TestCampaignServiceHandlersWithFakeClients(t *testing.T) {
	now := timestamppb.Now()

	campaignClient := &fakeCampaignClient{
		listResp: &statev1.ListCampaignsResponse{
			Campaigns: []*statev1.Campaign{
				{
					Id:               "camp-1",
					Name:             "Campaign",
					ParticipantCount: 2,
					CharacterCount:   1,
					CreatedAt:        now,
				},
			},
		},
		getResp: &statev1.GetCampaignResponse{
			Campaign: &statev1.Campaign{
				Id:               "camp-1",
				Name:             "Campaign",
				ParticipantCount: 2,
				CharacterCount:   1,
				CreatedAt:        now,
				UpdatedAt:        now,
			},
		},
	}
	sessionClient := &fakeSessionClient{
		listResp: &statev1.ListSessionsResponse{
			Sessions: []*statev1.Session{{Id: "s-1", CampaignId: "camp-1", Name: "Session 1", Status: statev1.SessionStatus_SESSION_ACTIVE, StartedAt: now}},
		},
		getResp: &statev1.GetSessionResponse{
			Session: &statev1.Session{Id: "s-1", CampaignId: "camp-1", Name: "Session 1", Status: statev1.SessionStatus_SESSION_ACTIVE, StartedAt: now},
		},
	}
	characterClient := &fakeCharacterClient{
		listResp: &statev1.ListCharactersResponse{
			Characters: []*statev1.Character{{Id: "char-1", CampaignId: "camp-1", Name: "Hero", ParticipantId: wrapperspb.String("part-1")}},
		},
		sheetResp: &statev1.GetCharacterSheetResponse{
			Character: &statev1.Character{Id: "char-1", CampaignId: "camp-1", Name: "Hero", ParticipantId: wrapperspb.String("part-1"), CreatedAt: now, UpdatedAt: now},
		},
	}
	participantClient := &fakeParticipantClient{
		listResp: &statev1.ListParticipantsResponse{
			Participants: []*statev1.Participant{{Id: "part-1", Name: "Alice", CreatedAt: now}},
		},
		getResp: &statev1.GetParticipantResponse{
			Participant: &statev1.Participant{Id: "part-1", Name: "Alice"},
		},
	}
	inviteClient := &fakeInviteClient{
		listResp: &statev1.ListInvitesResponse{
			Invites: []*statev1.Invite{{Id: "inv-1", CampaignId: "camp-1", ParticipantId: "part-1", RecipientUserId: "user-1", Status: statev1.InviteStatus_PENDING, CreatedAt: now, UpdatedAt: now}},
		},
	}
	eventClient := &fakeEventClient{
		listResp: &statev1.ListEventsResponse{
			Events: []*statev1.Event{
				{CampaignId: "camp-1", SessionId: "s-1", Seq: 1, Type: "campaign.created", Ts: now, EntityId: "char-1"},
			},
			TotalSize: 1,
		},
	}
	authClient := &fakeAuthClient{
		getUserResp: &authv1.GetUserResponse{User: &authv1.User{Id: "user-1", Email: "alice@example.com"}},
	}

	svcIface := NewHandlers(
		modulehandler.NewBase(),
		campaignClient,
		characterClient,
		participantClient,
		inviteClient,
		sessionClient,
		eventClient,
		authClient,
	)
	svc := svcIface.(*handlers)

	cases := []struct {
		name string
		path string
		call func(http.ResponseWriter, *http.Request)
	}{
		{name: "campaigns table", path: "/app/campaigns?fragment=rows", call: svc.HandleCampaignsTable},
		{name: "campaign detail", path: "/app/campaigns/camp-1", call: func(w http.ResponseWriter, r *http.Request) { svc.HandleCampaignDetail(w, r, "camp-1") }},
		{name: "characters table", path: "/app/campaigns/camp-1/characters?fragment=rows", call: func(w http.ResponseWriter, r *http.Request) { svc.HandleCharactersTable(w, r, "camp-1") }},
		{name: "character sheet", path: "/app/campaigns/camp-1/characters/char-1", call: func(w http.ResponseWriter, r *http.Request) { svc.HandleCharacterSheet(w, r, "camp-1", "char-1") }},
		{name: "participants table", path: "/app/campaigns/camp-1/participants?fragment=rows", call: func(w http.ResponseWriter, r *http.Request) { svc.HandleParticipantsTable(w, r, "camp-1") }},
		{name: "invites table", path: "/app/campaigns/camp-1/invites?fragment=rows", call: func(w http.ResponseWriter, r *http.Request) { svc.HandleInvitesTable(w, r, "camp-1") }},
		{name: "sessions table", path: "/app/campaigns/camp-1/sessions?fragment=rows", call: func(w http.ResponseWriter, r *http.Request) { svc.HandleSessionsTable(w, r, "camp-1") }},
		{name: "session detail", path: "/app/campaigns/camp-1/sessions/s-1", call: func(w http.ResponseWriter, r *http.Request) { svc.HandleSessionDetail(w, r, "camp-1", "s-1") }},
		{name: "session events", path: "/app/campaigns/camp-1/sessions/s-1/events", call: func(w http.ResponseWriter, r *http.Request) { svc.HandleSessionEvents(w, r, "camp-1", "s-1") }},
		{name: "event log", path: "/app/campaigns/camp-1/events", call: func(w http.ResponseWriter, r *http.Request) { svc.HandleEventLog(w, r, "camp-1") }},
		{name: "event log table", path: "/app/campaigns/camp-1/events?fragment=rows", call: func(w http.ResponseWriter, r *http.Request) { svc.HandleEventLogTable(w, r, "camp-1") }},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			tc.call(rec, req)
			if rec.Code != http.StatusOK {
				t.Fatalf("%s status = %d", tc.name, rec.Code)
			}
		})
	}
}
