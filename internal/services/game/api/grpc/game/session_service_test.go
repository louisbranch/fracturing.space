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
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestStartSession_NilRequest(t *testing.T) {
	svc := NewSessionService(Stores{})
	_, err := svc.StartSession(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
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
	if got := len(eventStore.events["c1"]); got != 2 {
		t.Fatalf("expected 2 events, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.TypeCampaignUpdated {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.TypeCampaignUpdated)
	}
	if eventStore.events["c1"][1].Type != event.TypeSessionStarted {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][1].Type, event.TypeSessionStarted)
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

func TestSetSessionSpotlight_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	spotlightStore := newFakeSessionSpotlightStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	sessionStore.sessions["c1"] = map[string]session.Session{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: now, UpdatedAt: now},
	}

	svc := &SessionService{
		stores: Stores{
			Campaign:         campaignStore,
			Session:          sessionStore,
			SessionSpotlight: spotlightStore,
			Event:            eventStore,
		},
		clock: fixedClock(now),
	}

	resp, err := svc.SetSessionSpotlight(context.Background(), &statev1.SetSessionSpotlightRequest{
		CampaignId:  "c1",
		SessionId:   "s1",
		Type:        statev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER,
		CharacterId: "char-1",
	})
	if err != nil {
		t.Fatalf("SetSessionSpotlight returned error: %v", err)
	}
	if resp.GetSpotlight() == nil {
		t.Fatal("expected spotlight in response")
	}
	if resp.GetSpotlight().GetCharacterId() != "char-1" {
		t.Fatalf("spotlight character_id = %q, want %q", resp.GetSpotlight().GetCharacterId(), "char-1")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.TypeSessionSpotlightSet {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.TypeSessionSpotlightSet)
	}
}

func TestGetSessionSpotlight_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	spotlightStore := newFakeSessionSpotlightStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	sessionStore.sessions["c1"] = map[string]session.Session{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: time.Now()},
	}
	spotlightStore.spotlights["c1"] = map[string]storage.SessionSpotlight{
		"s1": {
			CampaignID:    "c1",
			SessionID:     "s1",
			SpotlightType: string(session.SpotlightTypeGM),
			UpdatedAt:     time.Now(),
		},
	}

	svc := NewSessionService(Stores{
		Campaign:         campaignStore,
		Session:          sessionStore,
		SessionSpotlight: spotlightStore,
	})

	resp, err := svc.GetSessionSpotlight(context.Background(), &statev1.GetSessionSpotlightRequest{
		CampaignId: "c1",
		SessionId:  "s1",
	})
	if err != nil {
		t.Fatalf("GetSessionSpotlight returned error: %v", err)
	}
	if resp.GetSpotlight() == nil {
		t.Fatal("expected spotlight in response")
	}
	if resp.GetSpotlight().GetType() != statev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM {
		t.Fatalf("spotlight type = %v, want %v", resp.GetSpotlight().GetType(), statev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM)
	}
}

func TestClearSessionSpotlight_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	spotlightStore := newFakeSessionSpotlightStore()
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	sessionStore.sessions["c1"] = map[string]session.Session{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: time.Now()},
	}
	spotlightStore.spotlights["c1"] = map[string]storage.SessionSpotlight{
		"s1": {
			CampaignID:    "c1",
			SessionID:     "s1",
			SpotlightType: string(session.SpotlightTypeGM),
			UpdatedAt:     time.Now(),
		},
	}

	svc := NewSessionService(Stores{
		Campaign:         campaignStore,
		Session:          sessionStore,
		SessionSpotlight: spotlightStore,
		Event:            eventStore,
	})

	resp, err := svc.ClearSessionSpotlight(context.Background(), &statev1.ClearSessionSpotlightRequest{
		CampaignId: "c1",
		SessionId:  "s1",
		Reason:     "scene shift",
	})
	if err != nil {
		t.Fatalf("ClearSessionSpotlight returned error: %v", err)
	}
	if resp.GetSpotlight() == nil {
		t.Fatal("expected spotlight in response")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.TypeSessionSpotlightCleared {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.TypeSessionSpotlightCleared)
	}
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

func TestAbandonSessionGate_NilRequest(t *testing.T) {
	svc := NewSessionService(Stores{})
	_, err := svc.AbandonSessionGate(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSessionGate_MissingCampaignId(t *testing.T) {
	svc := NewSessionService(Stores{
		Campaign:    newFakeCampaignStore(),
		Session:     newFakeSessionStore(),
		SessionGate: newFakeSessionGateStore(),
		Event:       newFakeEventStore(),
	})
	_, err := svc.AbandonSessionGate(context.Background(), &statev1.AbandonSessionGateRequest{
		SessionId: "s1", GateId: "g1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSessionGate_MissingSessionId(t *testing.T) {
	svc := NewSessionService(Stores{
		Campaign:    newFakeCampaignStore(),
		Session:     newFakeSessionStore(),
		SessionGate: newFakeSessionGateStore(),
		Event:       newFakeEventStore(),
	})
	_, err := svc.AbandonSessionGate(context.Background(), &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", GateId: "g1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSessionGate_MissingGateId(t *testing.T) {
	svc := NewSessionService(Stores{
		Campaign:    newFakeCampaignStore(),
		Session:     newFakeSessionStore(),
		SessionGate: newFakeSessionGateStore(),
		Event:       newFakeEventStore(),
	})
	_, err := svc.AbandonSessionGate(context.Background(), &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", SessionId: "s1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSessionGate_CampaignNotFound(t *testing.T) {
	svc := NewSessionService(Stores{
		Campaign:    newFakeCampaignStore(),
		Session:     newFakeSessionStore(),
		SessionGate: newFakeSessionGateStore(),
		Event:       newFakeEventStore(),
	})
	_, err := svc.AbandonSessionGate(context.Background(), &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", SessionId: "s1", GateId: "g1",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestAbandonSessionGate_AlreadyAbandoned(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	gateStore := newFakeSessionGateStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	sessionStore.sessions["c1"] = map[string]session.Session{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: now, UpdatedAt: now},
	}
	gateStore.gates["c1:s1:g1"] = storage.SessionGate{
		CampaignID: "c1", SessionID: "s1", GateID: "g1",
		GateType: "decision", Status: string(session.GateStatusAbandoned),
		CreatedAt: now,
	}

	svc := &SessionService{
		stores: Stores{
			Campaign:    campaignStore,
			Session:     sessionStore,
			SessionGate: gateStore,
			Event:       eventStore,
		},
		clock: fixedClock(now),
	}

	resp, err := svc.AbandonSessionGate(context.Background(), &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", SessionId: "s1", GateId: "g1",
	})
	if err != nil {
		t.Fatalf("AbandonSessionGate returned error: %v", err)
	}
	if resp.GetGate() == nil {
		t.Fatal("expected gate in response")
	}
	if resp.GetGate().GetStatus() != statev1.SessionGateStatus_SESSION_GATE_ABANDONED {
		t.Fatalf("gate status = %v, want ABANDONED", resp.GetGate().GetStatus())
	}
	if len(eventStore.events["c1"]) != 0 {
		t.Fatalf("expected 0 events for already-abandoned gate, got %d", len(eventStore.events["c1"]))
	}
}

func TestAbandonSessionGate_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	gateStore := newFakeSessionGateStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	sessionStore.sessions["c1"] = map[string]session.Session{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: now, UpdatedAt: now},
	}
	gateStore.gates["c1:s1:g1"] = storage.SessionGate{
		CampaignID: "c1", SessionID: "s1", GateID: "g1",
		GateType: "decision", Status: string(session.GateStatusOpen),
		CreatedAt: now,
	}

	svc := &SessionService{
		stores: Stores{
			Campaign:    campaignStore,
			Session:     sessionStore,
			SessionGate: gateStore,
			Event:       eventStore,
		},
		clock: fixedClock(now),
	}

	ctx := contextWithParticipantID("part-1")
	resp, err := svc.AbandonSessionGate(ctx, &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", SessionId: "s1", GateId: "g1", Reason: "timeout",
	})
	if err != nil {
		t.Fatalf("AbandonSessionGate returned error: %v", err)
	}
	if resp.GetGate() == nil {
		t.Fatal("expected gate in response")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.TypeSessionGateAbandoned {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.TypeSessionGateAbandoned)
	}
}

func TestGetSessionSpotlight_NilRequest(t *testing.T) {
	svc := NewSessionService(Stores{})
	_, err := svc.GetSessionSpotlight(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetSessionSpotlight_MissingCampaignId(t *testing.T) {
	svc := NewSessionService(Stores{
		Campaign:         newFakeCampaignStore(),
		Session:          newFakeSessionStore(),
		SessionSpotlight: newFakeSessionSpotlightStore(),
	})
	_, err := svc.GetSessionSpotlight(context.Background(), &statev1.GetSessionSpotlightRequest{
		SessionId: "s1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetSessionSpotlight_MissingSessionId(t *testing.T) {
	svc := NewSessionService(Stores{
		Campaign:         newFakeCampaignStore(),
		Session:          newFakeSessionStore(),
		SessionSpotlight: newFakeSessionSpotlightStore(),
	})
	_, err := svc.GetSessionSpotlight(context.Background(), &statev1.GetSessionSpotlightRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetSessionSpotlight_NilRequest(t *testing.T) {
	svc := NewSessionService(Stores{})
	_, err := svc.SetSessionSpotlight(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetSessionSpotlight_MissingCampaignId(t *testing.T) {
	svc := NewSessionService(Stores{
		Campaign:         newFakeCampaignStore(),
		Session:          newFakeSessionStore(),
		SessionSpotlight: newFakeSessionSpotlightStore(),
		Event:            newFakeEventStore(),
	})
	_, err := svc.SetSessionSpotlight(context.Background(), &statev1.SetSessionSpotlightRequest{
		SessionId: "s1",
		Type:      statev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetSessionSpotlight_MissingSessionId(t *testing.T) {
	svc := NewSessionService(Stores{
		Campaign:         newFakeCampaignStore(),
		Session:          newFakeSessionStore(),
		SessionSpotlight: newFakeSessionSpotlightStore(),
		Event:            newFakeEventStore(),
	})
	_, err := svc.SetSessionSpotlight(context.Background(), &statev1.SetSessionSpotlightRequest{
		CampaignId: "c1",
		Type:       statev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetSessionSpotlight_InvalidType(t *testing.T) {
	svc := NewSessionService(Stores{
		Campaign:         newFakeCampaignStore(),
		Session:          newFakeSessionStore(),
		SessionSpotlight: newFakeSessionSpotlightStore(),
		Event:            newFakeEventStore(),
	})
	_, err := svc.SetSessionSpotlight(context.Background(), &statev1.SetSessionSpotlightRequest{
		CampaignId: "c1", SessionId: "s1",
		Type: statev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_UNSPECIFIED,
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSetSessionSpotlight_SessionNotActive(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = campaign.Campaign{ID: "c1", Status: campaign.CampaignStatusActive}
	endedAt := now.Add(-time.Hour)
	sessionStore.sessions["c1"] = map[string]session.Session{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.SessionStatusEnded, StartedAt: now.Add(-2 * time.Hour), EndedAt: &endedAt},
	}

	svc := NewSessionService(Stores{
		Campaign:         campaignStore,
		Session:          sessionStore,
		SessionSpotlight: newFakeSessionSpotlightStore(),
		Event:            newFakeEventStore(),
	})
	_, err := svc.SetSessionSpotlight(context.Background(), &statev1.SetSessionSpotlightRequest{
		CampaignId: "c1", SessionId: "s1",
		Type: statev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestClearSessionSpotlight_NilRequest(t *testing.T) {
	svc := NewSessionService(Stores{})
	_, err := svc.ClearSessionSpotlight(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestClearSessionSpotlight_MissingCampaignId(t *testing.T) {
	svc := NewSessionService(Stores{
		Campaign:         newFakeCampaignStore(),
		Session:          newFakeSessionStore(),
		SessionSpotlight: newFakeSessionSpotlightStore(),
		Event:            newFakeEventStore(),
	})
	_, err := svc.ClearSessionSpotlight(context.Background(), &statev1.ClearSessionSpotlightRequest{
		SessionId: "s1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestClearSessionSpotlight_MissingSessionId(t *testing.T) {
	svc := NewSessionService(Stores{
		Campaign:         newFakeCampaignStore(),
		Session:          newFakeSessionStore(),
		SessionSpotlight: newFakeSessionSpotlightStore(),
		Event:            newFakeEventStore(),
	})
	_, err := svc.ClearSessionSpotlight(context.Background(), &statev1.ClearSessionSpotlightRequest{
		CampaignId: "c1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
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
