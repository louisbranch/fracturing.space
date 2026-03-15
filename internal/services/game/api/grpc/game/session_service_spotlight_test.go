package game

import (
	"context"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestSetSessionSpotlight_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	spotlightStore := newFakeSessionSpotlightStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := newFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.campaigns["c1"] = activeCampaignRecord("c1")
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

	svc := newSessionServiceWithDependencies(
		Stores{
			Campaign:         campaignStore,
			Session:          sessionStore,
			SessionSpotlight: spotlightStore,
			Participant:      participantStore,
			Event:            eventStore,
			Write:            domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		fixedClock(now),
		nil,
	)

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

	campaignStore.campaigns["c1"] = activeCampaignRecord("c1")
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}

	svc := newSessionServiceWithDependencies(
		Stores{
			Campaign:         campaignStore,
			Session:          sessionStore,
			SessionSpotlight: spotlightStore,
			Participant:      participantStore,
			Event:            eventStore,
		},
		nil,
		nil,
	)
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

	campaignStore.campaigns["c1"] = activeCampaignRecord("c1")
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

	svc := newSessionServiceWithDependencies(
		Stores{
			Campaign:         campaignStore,
			Session:          sessionStore,
			SessionSpotlight: spotlightStore,
			Participant:      participantStore,
			Event:            eventStore,
			Write:            domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		fixedClock(now),
		nil,
	)

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

func TestGetSessionSpotlight_Success(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	sessionStore := newFakeSessionStore()
	spotlightStore := newFakeSessionSpotlightStore()
	participantStore := sessionManagerParticipantStore("c1")

	campaignStore.campaigns["c1"] = activeCampaignRecord("c1")
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

	campaignStore.campaigns["c1"] = activeCampaignRecord("c1")
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
			PayloadJSON: []byte(`{"reason":"scene change"}`),
		}),
	}}

	svc := NewSessionService(Stores{
		Campaign:         campaignStore,
		Session:          sessionStore,
		SessionSpotlight: spotlightStore,
		Participant:      participantStore,
		Event:            eventStore,
		Write:            domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
	})

	resp, err := svc.ClearSessionSpotlight(contextWithParticipantID("manager-1"), &statev1.ClearSessionSpotlightRequest{
		CampaignId: "c1",
		SessionId:  "s1",
		Reason:     "scene change",
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

	campaignStore.campaigns["c1"] = activeCampaignRecord("c1")
	sessionStore.sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	spotlightStore.spotlights["c1"] = map[string]storage.SessionSpotlight{
		"s1": {
			CampaignID: "c1", SessionID: "s1", SpotlightType: session.SpotlightTypeCharacter,
			CharacterID: "char-1", UpdatedAt: now,
		},
	}

	svc := newSessionServiceWithDependencies(
		Stores{
			Campaign:         campaignStore,
			Session:          sessionStore,
			SessionSpotlight: spotlightStore,
			Participant:      participantStore,
			Event:            eventStore,
		},
		nil,
		nil,
	)
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

	campaignStore.campaigns["c1"] = activeCampaignRecord("c1")
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
			PayloadJSON: []byte(`{"reason":"scene change"}`),
		}),
	}}

	svc := newSessionServiceWithDependencies(
		Stores{
			Campaign:         campaignStore,
			Session:          sessionStore,
			SessionSpotlight: spotlightStore,
			Participant:      participantStore,
			Event:            eventStore,
			Write:            domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		fixedClock(now),
		nil,
	)

	_, err := svc.ClearSessionSpotlight(contextWithParticipantID("manager-1"), &statev1.ClearSessionSpotlightRequest{
		CampaignId: "c1",
		SessionId:  "s1",
		Reason:     "scene change",
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

	campaignStore.campaigns["c1"] = activeCampaignRecord("c1")
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

	campaignStore.campaigns["c1"] = activeCampaignRecord("c1")
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
