package game

import (
	"context"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"google.golang.org/grpc/codes"
)

func TestStartSession_NilRequest(t *testing.T) {
	svc := NewSessionService(Stores{})
	_, err := svc.StartSession(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestStartSession_MissingCampaignStore(t *testing.T) {
	svc := NewSessionService(Stores{Session: newFakeSessionStore()})
	_, err := svc.StartSession(context.Background(), &statev1.StartSessionRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestStartSession_MissingSessionStore(t *testing.T) {
	svc := NewSessionService(Stores{Campaign: newFakeCampaignStore()})
	_, err := svc.StartSession(context.Background(), &statev1.StartSessionRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestStartSession_MissingEventStore(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.StartSession(context.Background(), &statev1.StartSessionRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestStartSession_MissingCampaignId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore})
	_, err := svc.StartSession(context.Background(), &statev1.StartSessionRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestStartSession_CampaignNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore})
	_, err := svc.StartSession(context.Background(), &statev1.StartSessionRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestStartSession_CampaignArchivedDisallowed(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusArchived,
	}

	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore})
	_, err := svc.StartSession(context.Background(), &statev1.StartSessionRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestStartSession_ActiveSessionExists(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusActive,
	}
	sessionStore.sessions["c1"] = map[string]session.Session{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"

	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore})
	_, err := svc.StartSession(context.Background(), &statev1.StartSessionRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestStartSession_Success_ActivatesDraftCampaign(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Name:   "Test Campaign",
		Status: campaign.CampaignStatusDraft,
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: campaign.GmModeHuman,
	}

	svc := &SessionService{
		stores:      Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("session-123"),
	}

	resp, err := svc.StartSession(context.Background(), &statev1.StartSessionRequest{
		CampaignId: "c1",
		Name:       "First Session",
	})
	if err != nil {
		t.Fatalf("StartSession returned error: %v", err)
	}
	if resp.Session == nil {
		t.Fatal("StartSession response has nil session")
	}
	if resp.Session.Id != "session-123" {
		t.Errorf("Session ID = %q, want %q", resp.Session.Id, "session-123")
	}
	if resp.Session.Status != statev1.SessionStatus_SESSION_ACTIVE {
		t.Errorf("Session Status = %v, want %v", resp.Session.Status, statev1.SessionStatus_SESSION_ACTIVE)
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.TypeSessionStarted {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.TypeSessionStarted)
	}

	// Verify campaign was activated
	storedCampaign, _ := campaignStore.Get(context.Background(), "c1")
	if storedCampaign.Status != campaign.CampaignStatusActive {
		t.Errorf("Campaign Status = %v, want %v", storedCampaign.Status, campaign.CampaignStatusActive)
	}
}

func TestStartSession_Success_AlreadyActive(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusActive,
	}

	svc := &SessionService{
		stores:      Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("session-123"),
	}

	resp, err := svc.StartSession(context.Background(), &statev1.StartSessionRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("StartSession returned error: %v", err)
	}
	if resp.Session == nil {
		t.Fatal("StartSession response has nil session")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.TypeSessionStarted {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.TypeSessionStarted)
	}
}

func TestListSessions_NilRequest(t *testing.T) {
	svc := NewSessionService(Stores{})
	_, err := svc.ListSessions(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListSessions_MissingCampaignId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.ListSessions(context.Background(), &statev1.ListSessionsRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListSessions_CampaignNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.ListSessions(context.Background(), &statev1.ListSessionsRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestListSessions_EmptyList(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusActive,
	}

	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore})
	resp, err := svc.ListSessions(context.Background(), &statev1.ListSessionsRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ListSessions returned error: %v", err)
	}
	if len(resp.Sessions) != 0 {
		t.Errorf("ListSessions returned %d sessions, want 0", len(resp.Sessions))
	}
}

func TestListSessions_WithSessions(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = campaign.Campaign{
		ID:     "c1",
		Status: campaign.CampaignStatusActive,
	}
	sessionStore.sessions["c1"] = map[string]session.Session{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.SessionStatusEnded, StartedAt: now},
		"s2": {ID: "s2", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: now},
	}

	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore})
	resp, err := svc.ListSessions(context.Background(), &statev1.ListSessionsRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("ListSessions returned error: %v", err)
	}
	if len(resp.Sessions) != 2 {
		t.Errorf("ListSessions returned %d sessions, want 2", len(resp.Sessions))
	}
}

func TestGetSession_NilRequest(t *testing.T) {
	svc := NewSessionService(Stores{})
	_, err := svc.GetSession(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetSession_MissingCampaignId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.GetSession(context.Background(), &statev1.GetSessionRequest{SessionId: "s1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetSession_MissingSessionId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.GetSession(context.Background(), &statev1.GetSessionRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetSession_CampaignNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.GetSession(context.Background(), &statev1.GetSessionRequest{CampaignId: "c1", SessionId: "s1"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetSession_SessionNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}

	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.GetSession(context.Background(), &statev1.GetSessionRequest{CampaignId: "c1", SessionId: "s1"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetSession_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	sessionStore.sessions["c1"] = map[string]session.Session{
		"s1": {ID: "s1", CampaignID: "c1", Name: "Test Session", Status: session.SessionStatusActive, StartedAt: now},
	}

	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore})
	resp, err := svc.GetSession(context.Background(), &statev1.GetSessionRequest{CampaignId: "c1", SessionId: "s1"})
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

func TestEndSession_NilRequest(t *testing.T) {
	svc := NewSessionService(Stores{})
	_, err := svc.EndSession(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestEndSession_MissingEventStore(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	sessionStore.sessions["c1"] = map[string]session.Session{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"

	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.EndSession(context.Background(), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestEndSession_MissingCampaignId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore})
	_, err := svc.EndSession(context.Background(), &statev1.EndSessionRequest{SessionId: "s1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestEndSession_MissingSessionId(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore})
	_, err := svc.EndSession(context.Background(), &statev1.EndSessionRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestEndSession_CampaignNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore})
	_, err := svc.EndSession(context.Background(), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestEndSession_SessionNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}

	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore})
	_, err := svc.EndSession(context.Background(), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestEndSession_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	sessionStore.sessions["c1"] = map[string]session.Session{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"

	svc := &SessionService{
		stores:      Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("session-123"),
	}

	resp, err := svc.EndSession(context.Background(), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	if err != nil {
		t.Fatalf("EndSession returned error: %v", err)
	}
	if resp.Session.Status != statev1.SessionStatus_SESSION_ENDED {
		t.Errorf("Session Status = %v, want %v", resp.Session.Status, statev1.SessionStatus_SESSION_ENDED)
	}
	if resp.Session.EndedAt == nil {
		t.Error("Session EndedAt is nil")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.TypeSessionEnded {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.TypeSessionEnded)
	}
}

func TestEndSession_AlreadyEnded(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	endedAt := now.Add(-1 * time.Hour)

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	sessionStore.sessions["c1"] = map[string]session.Session{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.SessionStatusEnded, StartedAt: now.Add(-2 * time.Hour), EndedAt: &endedAt},
	}

	svc := &SessionService{
		stores:      Stores{Campaign: campaignStore, Session: sessionStore, Event: eventStore},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("session-123"),
	}

	resp, err := svc.EndSession(context.Background(), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	if err != nil {
		t.Fatalf("EndSession returned error: %v", err)
	}
	if resp.Session.Status != statev1.SessionStatus_SESSION_ENDED {
		t.Errorf("Session Status = %v, want %v", resp.Session.Status, statev1.SessionStatus_SESSION_ENDED)
	}
}
