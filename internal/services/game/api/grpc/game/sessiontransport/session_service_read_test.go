package sessiontransport

import (
	"context"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestListSessions_NilRequest(t *testing.T) {
	svc := NewSessionService(Deps{})
	_, err := svc.ListSessions(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListSessions_MissingCampaignId(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.ListSessions(context.Background(), &statev1.ListSessionsRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListSessions_CampaignNotFound(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.ListSessions(context.Background(), &statev1.ListSessionsRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestListSessions_DeniesMissingIdentity(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore, Participant: participantStore})
	_, err := svc.ListSessions(context.Background(), &statev1.ListSessionsRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestListSessions_EmptyList(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore, Participant: participantStore})
	resp, err := svc.ListSessions(gametest.ContextWithParticipantID("manager-1"), &statev1.ListSessionsRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ListSessions returned error: %v", err)
	}
	if len(resp.Sessions) != 0 {
		t.Errorf("ListSessions returned %d sessions, want 0", len(resp.Sessions))
	}
}

func TestListSessions_WithSessions(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	now := time.Now().UTC()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusEnded, StartedAt: now},
		"s2": {ID: "s2", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}

	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore, Participant: participantStore})
	resp, err := svc.ListSessions(gametest.ContextWithParticipantID("manager-1"), &statev1.ListSessionsRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ListSessions returned error: %v", err)
	}
	if len(resp.Sessions) != 2 {
		t.Errorf("ListSessions returned %d sessions, want 2", len(resp.Sessions))
	}
}

func TestGetSession_NilRequest(t *testing.T) {
	svc := NewSessionService(Deps{})
	_, err := svc.GetSession(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetSession_MissingCampaignId(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.GetSession(context.Background(), &statev1.GetSessionRequest{SessionId: "s1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetSession_MissingSessionId(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.GetSession(context.Background(), &statev1.GetSessionRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetSession_CampaignNotFound(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.GetSession(context.Background(), &statev1.GetSessionRequest{CampaignId: "c1", SessionId: "s1"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetSession_DeniesMissingIdentity(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore, Participant: participantStore})
	_, err := svc.GetSession(context.Background(), &statev1.GetSessionRequest{CampaignId: "c1", SessionId: "s1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestGetSession_SessionNotFound(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore, Participant: participantStore})
	_, err := svc.GetSession(gametest.ContextWithParticipantID("manager-1"), &statev1.GetSessionRequest{CampaignId: "c1", SessionId: "s1"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetSession_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	now := time.Now().UTC()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Name: "Test Session", Status: session.StatusActive, StartedAt: now},
	}

	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore, Participant: participantStore})
	resp, err := svc.GetSession(gametest.ContextWithParticipantID("manager-1"), &statev1.GetSessionRequest{CampaignId: "c1", SessionId: "s1"})
	if err != nil {
		t.Fatalf("GetSession returned error: %v", err)
	}
	if resp.Session == nil {
		t.Fatal("GetSession response has nil session")
	}
	if resp.Session.Id != "s1" {
		t.Errorf("Session ID = %q, want %q", resp.Session.Id, "s1")
	}
	if resp.Session.Name != "Test Session" {
		t.Errorf("Session Name = %q, want %q", resp.Session.Name, "Test Session")
	}
}
