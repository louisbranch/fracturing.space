package game

import (
	"context"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func sessionManagerParticipantStore(campaignID string) *fakeParticipantStore {
	store := newFakeParticipantStore()
	store.participants[campaignID] = map[string]storage.ParticipantRecord{
		"manager-1": {
			ID:             "manager-1",
			CampaignID:     campaignID,
			CampaignAccess: participant.CampaignAccessManager,
		},
	}
	return store
}

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
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusArchived,
	}

	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore, Event: eventStore})
	_, err := svc.StartSession(contextWithParticipantID("manager-1"), &statev1.StartSessionRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestStartSession_ActiveSessionExists(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"

	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore, Event: eventStore})
	_, err := svc.StartSession(contextWithParticipantID("manager-1"), &statev1.StartSessionRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestStartSession_RequiresDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusDraft}

	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore, Event: eventStore})
	_, err := svc.StartSession(contextWithParticipantID("manager-1"), &statev1.StartSessionRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestStartSession_Success_ActivatesDraftCampaign(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Name:   "Test Campaign",
		Status: campaign.StatusDraft,
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode: campaign.GmModeHuman,
	}
	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("campaign.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("campaign.updated"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "campaign",
				EntityID:    "c1",
				PayloadJSON: []byte(`{"fields":{"status":"active"}}`),
			}),
		},
		command.Type("session.start"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("session.started"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "session-123",
				EntityType:  "session",
				EntityID:    "session-123",
				PayloadJSON: []byte(`{"session_id":"session-123","session_name":"First Session"}`),
			}),
		},
	}}

	svc := &SessionService{
		stores:      Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore, Event: eventStore, Domain: domain},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("session-123"),
	}

	resp, err := svc.StartSession(contextWithParticipantID("manager-1"), &statev1.StartSessionRequest{
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
	if eventStore.events["c1"][0].Type != event.Type("campaign.updated") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("campaign.updated"))
	}
	if eventStore.events["c1"][1].Type != event.Type("session.started") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][1].Type, event.Type("session.started"))
	}

	// Verify campaign was activated
	storedCampaign, _ := campaignStore.Get(context.Background(), "c1")
	if storedCampaign.Status != campaign.StatusActive {
		t.Errorf("Campaign Status = %v, want %v", storedCampaign.Status, campaign.StatusActive)
	}
}

func TestStartSession_Success_AlreadyActive(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.started"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			SessionID:   "session-123",
			EntityType:  "session",
			EntityID:    "session-123",
			PayloadJSON: []byte(`{"session_id":"session-123"}`),
		}),
	}}

	svc := &SessionService{
		stores:      Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore, Event: eventStore, Domain: domain},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("session-123"),
	}

	resp, err := svc.StartSession(contextWithParticipantID("manager-1"), &statev1.StartSessionRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("StartSession returned error: %v", err)
	}
	if resp.Session == nil {
		t.Fatal("StartSession response has nil session")
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].Type != event.Type("session.started") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("session.started"))
	}
}

func TestStartSession_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.started"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			SessionID:   "session-123",
			EntityType:  "session",
			EntityID:    "session-123",
			PayloadJSON: []byte(`{"session_id":"session-123","session_name":"First Session"}`),
		}),
	}}

	svc := &SessionService{
		stores: Stores{
			Campaign:    campaignStore,
			Session:     sessionStore,
			Participant: participantStore,
			Event:       eventStore,
			Domain:      domain,
		},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("session-123"),
	}

	_, err := svc.StartSession(contextWithParticipantID("manager-1"), &statev1.StartSessionRequest{
		CampaignId: "c1",
		Name:       "First Session",
	})
	if err != nil {
		t.Fatalf("StartSession returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("session.start") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "session.start")
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

func TestListSessions_DeniesMissingIdentity(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}

	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore})
	_, err := svc.ListSessions(context.Background(), &statev1.ListSessionsRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestSetSessionSpotlight_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	spotlightStore := newFakeSessionSpotlightStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.spotlight_set"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			SessionID:   "s1",
			EntityType:  "session",
			EntityID:    "s1",
			PayloadJSON: []byte(`{"spotlight_type":"character","character_id":"char-1"}`),
		}),
	}}

	svc := &SessionService{
		stores: Stores{
			Campaign:         campaignStore,
			Session:          sessionStore,
			SessionSpotlight: spotlightStore,
			Participant:      participantStore,
			Event:            eventStore,
			Domain:           domain,
		},
		clock: fixedClock(now),
	}

	resp, err := svc.SetSessionSpotlight(contextWithParticipantID("manager-1"), &statev1.SetSessionSpotlightRequest{
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
	if eventStore.events["c1"][0].Type != event.Type("session.spotlight_set") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("session.spotlight_set"))
	}
}

func TestSetSessionSpotlight_RequiresDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	spotlightStore := newFakeSessionSpotlightStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}

	svc := &SessionService{stores: Stores{Campaign: campaignStore, Session: sessionStore, SessionSpotlight: spotlightStore, Participant: participantStore, Event: eventStore}}
	_, err := svc.SetSessionSpotlight(contextWithParticipantID("manager-1"), &statev1.SetSessionSpotlightRequest{
		CampaignId:  "c1",
		SessionId:   "s1",
		Type:        statev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER,
		CharacterId: "char-1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestSetSessionSpotlight_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	spotlightStore := newFakeSessionSpotlightStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.spotlight_set"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			SessionID:   "s1",
			EntityType:  "session",
			EntityID:    "s1",
			PayloadJSON: []byte(`{"spotlight_type":"character","character_id":"char-1"}`),
		}),
	}}

	svc := &SessionService{
		stores: Stores{
			Campaign:         campaignStore,
			Session:          sessionStore,
			SessionSpotlight: spotlightStore,
			Participant:      participantStore,
			Event:            eventStore,
			Domain:           domain,
		},
		clock: fixedClock(now),
	}

	_, err := svc.SetSessionSpotlight(contextWithParticipantID("manager-1"), &statev1.SetSessionSpotlightRequest{
		CampaignId:  "c1",
		SessionId:   "s1",
		Type:        statev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER,
		CharacterId: "char-1",
	})
	if err != nil {
		t.Fatalf("SetSessionSpotlight returned error: %v", err)
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].EntityType != "session" {
		t.Fatalf("event entity type = %s, want %s", eventStore.events["c1"][0].EntityType, "session")
	}
}

type fakeDomainEngine struct {
	store         storage.EventStore
	result        engine.Result
	resultsByType map[command.Type]engine.Result
	calls         int
	lastCommand   command.Command
	commands      []command.Command
}

func (f *fakeDomainEngine) Execute(ctx context.Context, cmd command.Command) (engine.Result, error) {
	f.calls++
	f.lastCommand = cmd
	f.commands = append(f.commands, cmd)

	result := f.result
	if len(f.resultsByType) > 0 {
		if selected, ok := f.resultsByType[cmd.Type]; ok {
			result = selected
		}
	}
	if f.store == nil {
		return result, nil
	}
	if len(result.Decision.Events) == 0 {
		return result, nil
	}
	stored := make([]event.Event, 0, len(result.Decision.Events))
	for _, evt := range result.Decision.Events {
		storedEvent, err := f.store.AppendEvent(ctx, evt)
		if err != nil {
			return engine.Result{}, err
		}
		stored = append(stored, storedEvent)
	}
	result.Decision.Events = stored
	return result, nil
}

func TestGetSessionSpotlight_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	spotlightStore := newFakeSessionSpotlightStore()
	participantStore := sessionManagerParticipantStore("c1")

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: time.Now()},
	}
	spotlightStore.spotlights["c1"] = map[string]storage.SessionSpotlight{
		"s1": {
			CampaignID:    "c1",
			SessionID:     "s1",
			SpotlightType: session.SpotlightTypeGM,
			UpdatedAt:     time.Now(),
		},
	}

	svc := NewSessionService(Stores{
		Campaign:         campaignStore,
		Session:          sessionStore,
		SessionSpotlight: spotlightStore,
		Participant:      participantStore,
	})

	resp, err := svc.GetSessionSpotlight(contextWithParticipantID("manager-1"), &statev1.GetSessionSpotlightRequest{
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
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: time.Now()},
	}
	spotlightStore.spotlights["c1"] = map[string]storage.SessionSpotlight{
		"s1": {
			CampaignID:    "c1",
			SessionID:     "s1",
			SpotlightType: session.SpotlightTypeGM,
			UpdatedAt:     time.Now(),
		},
	}
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)
	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.spotlight_cleared"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			SessionID:   "s1",
			EntityType:  "session",
			EntityID:    "s1",
			PayloadJSON: []byte(`{"reason":"scene shift"}`),
		}),
	}}

	svc := NewSessionService(Stores{
		Campaign:         campaignStore,
		Session:          sessionStore,
		SessionSpotlight: spotlightStore,
		Participant:      participantStore,
		Event:            eventStore,
		Domain:           domain,
	})

	resp, err := svc.ClearSessionSpotlight(contextWithParticipantID("manager-1"), &statev1.ClearSessionSpotlightRequest{
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
	if eventStore.events["c1"][0].Type != event.Type("session.spotlight_cleared") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("session.spotlight_cleared"))
	}
}

func TestClearSessionSpotlight_RequiresDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	spotlightStore := newFakeSessionSpotlightStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	spotlightStore.spotlights["c1"] = map[string]storage.SessionSpotlight{
		"s1": {
			CampaignID: "c1", SessionID: "s1", SpotlightType: session.SpotlightTypeCharacter,
			CharacterID: "char-1", UpdatedAt: now,
		},
	}

	svc := &SessionService{stores: Stores{Campaign: campaignStore, Session: sessionStore, SessionSpotlight: spotlightStore, Participant: participantStore, Event: eventStore}}
	_, err := svc.ClearSessionSpotlight(contextWithParticipantID("manager-1"), &statev1.ClearSessionSpotlightRequest{
		CampaignId: "c1", SessionId: "s1", Reason: "break",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestClearSessionSpotlight_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	spotlightStore := newFakeSessionSpotlightStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	spotlightStore.spotlights["c1"] = map[string]storage.SessionSpotlight{
		"s1": {
			CampaignID:    "c1",
			SessionID:     "s1",
			SpotlightType: session.SpotlightTypeGM,
			UpdatedAt:     now,
		},
	}

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.spotlight_cleared"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			SessionID:   "s1",
			EntityType:  "session",
			EntityID:    "s1",
			PayloadJSON: []byte(`{"reason":"scene shift"}`),
		}),
	}}

	svc := &SessionService{
		stores: Stores{
			Campaign:         campaignStore,
			Session:          sessionStore,
			SessionSpotlight: spotlightStore,
			Participant:      participantStore,
			Event:            eventStore,
			Domain:           domain,
		},
		clock: fixedClock(now),
	}

	_, err := svc.ClearSessionSpotlight(contextWithParticipantID("manager-1"), &statev1.ClearSessionSpotlightRequest{
		CampaignId: "c1",
		SessionId:  "s1",
		Reason:     "scene shift",
	})
	if err != nil {
		t.Fatalf("ClearSessionSpotlight returned error: %v", err)
	}
	if got := len(eventStore.events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.events["c1"][0].EntityType != "session" {
		t.Fatalf("event entity type = %s, want %s", eventStore.events["c1"][0].EntityType, "session")
	}
}

func TestListSessions_EmptyList(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}

	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore})
	resp, err := svc.ListSessions(contextWithParticipantID("manager-1"), &statev1.ListSessionsRequest{CampaignId: "c1"})
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
	participantStore := sessionManagerParticipantStore("c1")
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Status: campaign.StatusActive,
	}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusEnded, StartedAt: now},
		"s2": {ID: "s2", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}

	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore})
	resp, err := svc.ListSessions(contextWithParticipantID("manager-1"), &statev1.ListSessionsRequest{CampaignId: "c1"})
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

func TestGetSession_DeniesMissingIdentity(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}

	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore})
	_, err := svc.GetSession(context.Background(), &statev1.GetSessionRequest{CampaignId: "c1", SessionId: "s1"})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestGetSession_SessionNotFound(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}

	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore})
	_, err := svc.GetSession(contextWithParticipantID("manager-1"), &statev1.GetSessionRequest{CampaignId: "c1", SessionId: "s1"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetSession_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Name: "Test Session", Status: session.StatusActive, StartedAt: now},
	}

	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore})
	resp, err := svc.GetSession(contextWithParticipantID("manager-1"), &statev1.GetSessionRequest{CampaignId: "c1", SessionId: "s1"})
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
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}

	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore, Event: eventStore})
	_, err := svc.EndSession(contextWithParticipantID("manager-1"), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestEndSession_DeniesMemberAccess(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"member-1": {
			ID:             "member-1",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessMember,
		},
	}

	svc := NewSessionService(Stores{
		Campaign:    campaignStore,
		Session:     sessionStore,
		Participant: participantStore,
		Event:       eventStore,
	})
	_, err := svc.EndSession(contextWithParticipantID("member-1"), &statev1.EndSessionRequest{
		CampaignId: "c1",
		SessionId:  "s1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestEndSession_RequiresDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"

	svc := NewSessionService(Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore, Event: eventStore})
	_, err := svc.EndSession(contextWithParticipantID("manager-1"), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestEndSession_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"
	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.ended"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			SessionID:   "s1",
			EntityType:  "session",
			EntityID:    "s1",
			PayloadJSON: []byte(`{"session_id":"s1"}`),
		}),
	}}

	svc := &SessionService{
		stores:      Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore, Event: eventStore, Domain: domain},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("session-123"),
	}

	resp, err := svc.EndSession(contextWithParticipantID("manager-1"), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
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
	if eventStore.events["c1"][0].Type != event.Type("session.ended") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("session.ended"))
	}
}

func TestEndSession_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	sessionStore.activeSession["c1"] = "s1"

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.ended"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			SessionID:   "s1",
			EntityType:  "session",
			EntityID:    "s1",
			PayloadJSON: []byte(`{"session_id":"s1"}`),
		}),
	}}

	svc := &SessionService{
		stores: Stores{
			Campaign:    campaignStore,
			Session:     sessionStore,
			Participant: participantStore,
			Event:       eventStore,
			Domain:      domain,
		},
		clock: fixedClock(now),
	}

	_, err := svc.EndSession(contextWithParticipantID("manager-1"), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	if err != nil {
		t.Fatalf("EndSession returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("session.end") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "session.end")
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

func TestAbandonSessionGate_DeniesMemberAccess(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	gateStore := newFakeSessionGateStore()
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	gateStore.gates["c1:s1:g1"] = storage.SessionGate{
		CampaignID: "c1", SessionID: "s1", GateID: "g1",
		GateType: "decision", Status: session.GateStatusOpen,
		CreatedAt: now,
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"member-1": {
			ID:             "member-1",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessMember,
		},
	}

	svc := NewSessionService(Stores{
		Campaign:    campaignStore,
		Session:     sessionStore,
		SessionGate: gateStore,
		Participant: participantStore,
		Event:       eventStore,
	})
	_, err := svc.AbandonSessionGate(contextWithParticipantID("member-1"), &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", SessionId: "s1", GateId: "g1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestAbandonSessionGate_AlreadyAbandoned(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	gateStore := newFakeSessionGateStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	gateStore.gates["c1:s1:g1"] = storage.SessionGate{
		CampaignID: "c1", SessionID: "s1", GateID: "g1",
		GateType: "decision", Status: session.GateStatusAbandoned,
		CreatedAt: now,
	}

	svc := &SessionService{
		stores: Stores{
			Campaign:    campaignStore,
			Session:     sessionStore,
			SessionGate: gateStore,
			Participant: participantStore,
			Event:       eventStore,
		},
		clock: fixedClock(now),
	}

	resp, err := svc.AbandonSessionGate(contextWithParticipantID("manager-1"), &statev1.AbandonSessionGateRequest{
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
	participantStore := newFakeParticipantStore()
	eventStore := newFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	gateStore.gates["c1:s1:g1"] = storage.SessionGate{
		CampaignID: "c1", SessionID: "s1", GateID: "g1",
		GateType: "decision", Status: session.GateStatusOpen,
		CreatedAt: now,
	}
	participantStore.participants["c1"] = map[string]storage.ParticipantRecord{
		"part-1": {
			ID:             "part-1",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessManager,
		},
	}
	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.gate_abandoned"),
			Timestamp:   now,
			ActorType:   event.ActorTypeParticipant,
			ActorID:     "part-1",
			SessionID:   "s1",
			EntityType:  "session_gate",
			EntityID:    "g1",
			PayloadJSON: []byte(`{"gate_id":"g1","reason":"timeout"}`),
		}),
	}}

	svc := &SessionService{
		stores: Stores{
			Campaign:    campaignStore,
			Session:     sessionStore,
			SessionGate: gateStore,
			Participant: participantStore,
			Event:       eventStore,
			Domain:      domain,
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
	if eventStore.events["c1"][0].Type != event.Type("session.gate_abandoned") {
		t.Fatalf("event type = %s, want %s", eventStore.events["c1"][0].Type, event.Type("session.gate_abandoned"))
	}
}

func TestAbandonSessionGate_RequiresDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	gateStore := newFakeSessionGateStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	gateStore.gates["c1:s1:g1"] = storage.SessionGate{
		CampaignID: "c1", SessionID: "s1", GateID: "g1",
		GateType: "decision", Status: session.GateStatusOpen,
		CreatedAt: now,
	}

	svc := &SessionService{stores: Stores{Campaign: campaignStore, Session: sessionStore, SessionGate: gateStore, Participant: participantStore, Event: eventStore}}
	_, err := svc.AbandonSessionGate(contextWithParticipantID("manager-1"), &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", SessionId: "s1", GateId: "g1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestOpenSessionGate_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	gateStore := newFakeSessionGateStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.gate_opened"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			SessionID:   "s1",
			EntityType:  "session_gate",
			EntityID:    "g1",
			PayloadJSON: []byte(`{"gate_id":"g1","gate_type":"spotlight"}`),
		}),
	}}

	svc := &SessionService{
		stores: Stores{
			Campaign:    campaignStore,
			Session:     sessionStore,
			SessionGate: gateStore,
			Participant: participantStore,
			Event:       eventStore,
			Domain:      domain,
		},
		clock: fixedClock(now),
	}

	_, err := svc.OpenSessionGate(contextWithParticipantID("manager-1"), &statev1.OpenSessionGateRequest{
		CampaignId: "c1",
		SessionId:  "s1",
		GateType:   "spotlight",
		GateId:     "g1",
	})
	if err != nil {
		t.Fatalf("OpenSessionGate returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("session.gate_open") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "session.gate_open")
	}
}

func TestResolveSessionGate_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	gateStore := newFakeSessionGateStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	gateStore.gates["c1:s1:g1"] = storage.SessionGate{
		CampaignID: "c1", SessionID: "s1", GateID: "g1",
		GateType: "decision", Status: session.GateStatusOpen,
		CreatedAt: now,
	}

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.gate_resolved"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			SessionID:   "s1",
			EntityType:  "session_gate",
			EntityID:    "g1",
			PayloadJSON: []byte(`{"gate_id":"g1","decision":"allow"}`),
		}),
	}}

	svc := &SessionService{
		stores: Stores{
			Campaign:    campaignStore,
			Session:     sessionStore,
			SessionGate: gateStore,
			Participant: participantStore,
			Event:       eventStore,
			Domain:      domain,
		},
		clock: fixedClock(now),
	}

	_, err := svc.ResolveSessionGate(contextWithParticipantID("manager-1"), &statev1.ResolveSessionGateRequest{
		CampaignId: "c1",
		SessionId:  "s1",
		GateId:     "g1",
		Decision:   "allow",
	})
	if err != nil {
		t.Fatalf("ResolveSessionGate returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("session.gate_resolve") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "session.gate_resolve")
	}
}

func TestAbandonSessionGate_UsesDomainEngine(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	gateStore := newFakeSessionGateStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	gateStore.gates["c1:s1:g1"] = storage.SessionGate{
		CampaignID: "c1", SessionID: "s1", GateID: "g1",
		GateType: "decision", Status: session.GateStatusOpen,
		CreatedAt: now,
	}

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(event.Event{
			CampaignID:  "c1",
			Type:        event.Type("session.gate_abandoned"),
			Timestamp:   now,
			ActorType:   event.ActorTypeSystem,
			SessionID:   "s1",
			EntityType:  "session_gate",
			EntityID:    "g1",
			PayloadJSON: []byte(`{"gate_id":"g1","reason":"timeout"}`),
		}),
	}}

	svc := &SessionService{
		stores: Stores{
			Campaign:    campaignStore,
			Session:     sessionStore,
			SessionGate: gateStore,
			Participant: participantStore,
			Event:       eventStore,
			Domain:      domain,
		},
		clock: fixedClock(now),
	}

	_, err := svc.AbandonSessionGate(contextWithParticipantID("manager-1"), &statev1.AbandonSessionGateRequest{
		CampaignId: "c1",
		SessionId:  "s1",
		GateId:     "g1",
		Reason:     "timeout",
	})
	if err != nil {
		t.Fatalf("AbandonSessionGate returned error: %v", err)
	}
	if domain.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", domain.calls)
	}
	if domain.lastCommand.Type != command.Type("session.gate_abandon") {
		t.Fatalf("command type = %s, want %s", domain.lastCommand.Type, "session.gate_abandon")
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

func TestGetSessionSpotlight_DeniesMissingIdentity(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	spotlightStore := newFakeSessionSpotlightStore()
	participantStore := sessionManagerParticipantStore("c1")

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: time.Now()},
	}
	spotlightStore.spotlights["c1"] = map[string]storage.SessionSpotlight{
		"s1": {
			CampaignID:    "c1",
			SessionID:     "s1",
			SpotlightType: session.SpotlightTypeGM,
			UpdatedAt:     time.Now(),
		},
	}

	svc := NewSessionService(Stores{
		Campaign:         campaignStore,
		Session:          sessionStore,
		SessionSpotlight: spotlightStore,
		Participant:      participantStore,
	})

	_, err := svc.GetSessionSpotlight(context.Background(), &statev1.GetSessionSpotlightRequest{
		CampaignId: "c1",
		SessionId:  "s1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
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
	participantStore := sessionManagerParticipantStore("c1")
	now := time.Now().UTC()

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	endedAt := now.Add(-time.Hour)
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusEnded, StartedAt: now.Add(-2 * time.Hour), EndedAt: &endedAt},
	}

	svc := NewSessionService(Stores{
		Campaign:         campaignStore,
		Session:          sessionStore,
		SessionSpotlight: newFakeSessionSpotlightStore(),
		Participant:      participantStore,
		Event:            newFakeEventStore(),
	})
	_, err := svc.SetSessionSpotlight(contextWithParticipantID("manager-1"), &statev1.SetSessionSpotlightRequest{
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
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	endedAt := now.Add(-1 * time.Hour)

	campaignStore.campaigns["c1"] = storage.CampaignRecord{ID: "c1", Status: campaign.StatusActive}
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusEnded, StartedAt: now.Add(-2 * time.Hour), EndedAt: &endedAt},
	}

	svc := &SessionService{
		stores:      Stores{Campaign: campaignStore, Session: sessionStore, Participant: participantStore, Event: eventStore},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("session-123"),
	}

	resp, err := svc.EndSession(contextWithParticipantID("manager-1"), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	if err != nil {
		t.Fatalf("EndSession returned error: %v", err)
	}
	if resp.Session.Status != statev1.SessionStatus_SESSION_ENDED {
		t.Errorf("Session Status = %v, want %v", resp.Session.Status, statev1.SessionStatus_SESSION_ENDED)
	}
}
