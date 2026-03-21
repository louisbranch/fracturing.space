package sessiontransport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	systems "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

type fakeSessionInteractionStore struct {
	values map[string]storage.SessionInteraction
}

func (s *fakeSessionInteractionStore) GetSessionInteraction(_ context.Context, campaignID, sessionID string) (storage.SessionInteraction, error) {
	if s == nil || s.values == nil {
		return storage.SessionInteraction{}, storage.ErrNotFound
	}
	value, ok := s.values[campaignID+":"+sessionID]
	if !ok {
		return storage.SessionInteraction{}, storage.ErrNotFound
	}
	return value, nil
}

func (s *fakeSessionInteractionStore) PutSessionInteraction(_ context.Context, interaction storage.SessionInteraction) error {
	if s.values == nil {
		s.values = make(map[string]storage.SessionInteraction)
	}
	s.values[interaction.CampaignID+":"+interaction.SessionID] = interaction
	return nil
}

func TestStartSession_NilRequest(t *testing.T) {
	svc := NewSessionService(Deps{})
	_, err := svc.StartSession(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestStartSession_MissingCampaignId(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.StartSession(context.Background(), &statev1.StartSessionRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestStartSession_CampaignNotFound(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.StartSession(context.Background(), &statev1.StartSessionRequest{CampaignId: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestStartSession_CampaignArchivedDisallowed(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	domain := &fakeDomainEngine{result: engine.Result{
		Decision: command.Reject(command.Rejection{
			Code:    "SESSION_READINESS_CAMPAIGN_STATUS_DISALLOWS_START",
			Message: "campaign status does not allow session start",
		}),
	}}
	campaignStore.Campaigns["c1"] = gametest.ArchivedCampaignRecord("c1")

	svc := NewSessionService(Deps{
		Campaign:    campaignStore,
		Session:     sessionStore,
		Participant: participantStore,
		Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
	})
	_, err := svc.StartSession(gametest.ContextWithParticipantID("manager-1"), &statev1.StartSessionRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestStartSession_ActiveSessionExists(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	domain := &fakeDomainEngine{result: engine.Result{
		Decision: command.Reject(command.Rejection{
			Code:    "SESSION_READINESS_ACTIVE_SESSION_EXISTS",
			Message: "an active session already exists",
		}),
	}}
	now := time.Now().UTC()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	sessionStore.ActiveSession["c1"] = "s1"

	svc := NewSessionService(Deps{
		Campaign:    campaignStore,
		Session:     sessionStore,
		Participant: participantStore,
		Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
	})
	_, err := svc.StartSession(gametest.ContextWithParticipantID("manager-1"), &statev1.StartSessionRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestStartSession_RequiresDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")

	campaignStore.Campaigns["c1"] = gametest.DraftCampaignRecord("c1")

	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore, Participant: participantStore})
	_, err := svc.StartSession(gametest.ContextWithParticipantID("manager-1"), &statev1.StartSessionRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestStartSession_Success_ActivatesDraftCampaign(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = storage.CampaignRecord{
		ID:     "c1",
		Name:   "Test Campaign",
		Status: campaign.StatusDraft,
		System: systems.SystemIDDaggerheart,
		GmMode: campaign.GmModeHuman,
	}
	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(),
	}, resultsByType: map[command.Type]engine.Result{
		"session.start": {
			Decision: command.Accept(
				event.Event{
					CampaignID:  "c1",
					Type:        event.Type("campaign.updated"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					EntityType:  "campaign",
					EntityID:    "c1",
					PayloadJSON: []byte(`{"fields":{"status":"active"}}`),
				},
				event.Event{
					CampaignID:  "c1",
					Type:        event.Type("session.started"),
					Timestamp:   now,
					ActorType:   event.ActorTypeSystem,
					SessionID:   "session-123",
					EntityType:  "session",
					EntityID:    "session-123",
					PayloadJSON: []byte(`{"session_id":"session-123","session_name":"First Session"}`),
				},
			),
		},
		"session.gm_authority.set": {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("session.gm_authority_set"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "session-123",
				EntityType:  "session",
				EntityID:    "session-123",
				PayloadJSON: []byte(`{"session_id":"session-123","participant_id":"manager-1"}`),
			}),
		},
	}}

	svc := newTestSessionService(
		Deps{
			Campaign:           campaignStore,
			Session:            sessionStore,
			Participant:        participantStore,
			SessionInteraction: &fakeSessionInteractionStore{},
			Write:              domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("session-123"),
	)

	resp, err := svc.StartSession(gametest.ContextWithParticipantID("manager-1"), &statev1.StartSessionRequest{
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
	if domain.calls != 2 {
		t.Fatalf("expected domain to be called twice, got %d", domain.calls)
	}
	if got := len(eventStore.Events["c1"]); got != 3 {
		t.Fatalf("expected 3 events, got %d", got)
	}
	if eventStore.Events["c1"][0].Type != event.Type("campaign.updated") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][0].Type, event.Type("campaign.updated"))
	}
	if eventStore.Events["c1"][1].Type != event.Type("session.started") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][1].Type, event.Type("session.started"))
	}
	if eventStore.Events["c1"][2].Type != event.Type("session.gm_authority_set") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][2].Type, event.Type("session.gm_authority_set"))
	}

	storedCampaign, _ := campaignStore.Get(context.Background(), "c1")
	if storedCampaign.Status != campaign.StatusActive {
		t.Errorf("Campaign Status = %v, want %v", storedCampaign.Status, campaign.StatusActive)
	}
}

func TestStartSession_Success_AlreadyActive(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(),
	}, resultsByType: map[command.Type]engine.Result{
		"session.start": {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("session.started"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "session-123",
				EntityType:  "session",
				EntityID:    "session-123",
				PayloadJSON: []byte(`{"session_id":"session-123","session_name":"Session 1"}`),
			}),
		},
		"session.gm_authority.set": {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("session.gm_authority_set"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "session-123",
				EntityType:  "session",
				EntityID:    "session-123",
				PayloadJSON: []byte(`{"session_id":"session-123","participant_id":"manager-1"}`),
			}),
		},
	}}

	svc := newTestSessionService(
		Deps{
			Campaign:           campaignStore,
			Session:            sessionStore,
			Participant:        participantStore,
			SessionInteraction: &fakeSessionInteractionStore{},
			Write:              domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("session-123"),
	)

	resp, err := svc.StartSession(gametest.ContextWithParticipantID("manager-1"), &statev1.StartSessionRequest{CampaignId: "c1"})
	if err != nil {
		t.Fatalf("StartSession returned error: %v", err)
	}
	if resp.Session == nil {
		t.Fatal("StartSession response has nil session")
	}
	if got := len(eventStore.Events["c1"]); got != 2 {
		t.Fatalf("expected 2 events, got %d", got)
	}
	if eventStore.Events["c1"][0].Type != event.Type("session.started") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][0].Type, event.Type("session.started"))
	}
	if eventStore.Events["c1"][1].Type != event.Type("session.gm_authority_set") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][1].Type, event.Type("session.gm_authority_set"))
	}
}

func TestStartSession_BlankNameDefaultsToCampaignLocaleSequence(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		campaignLocale string
		existingCount  int
		wantLocale     commonv1.Locale
		wantName       string
	}{
		{
			name:           "english first session",
			campaignLocale: "en-US",
			existingCount:  0,
			wantLocale:     commonv1.Locale_LOCALE_EN_US,
			wantName:       "Session 1",
		},
		{
			name:           "portuguese second session",
			campaignLocale: "pt-BR",
			existingCount:  1,
			wantLocale:     commonv1.Locale_LOCALE_PT_BR,
			wantName:       "Sessão 2",
		},
		{
			name:           "invalid locale falls back to default",
			campaignLocale: "fr-FR",
			existingCount:  0,
			wantLocale:     commonv1.Locale_LOCALE_EN_US,
			wantName:       "Session 1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			campaignStore := gametest.NewFakeCampaignStore()
			sessionStore := gametest.NewFakeSessionStore()
			participantStore := sessionManagerParticipantStore("c1")
			eventStore := gametest.NewFakeEventStore()
			now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

			campaignStore.Campaigns["c1"] = storage.CampaignRecord{
				ID:     "c1",
				Name:   "Test Campaign",
				Locale: tc.campaignLocale,
				Status: campaign.StatusActive,
				System: systems.SystemIDDaggerheart,
				GmMode: campaign.GmModeHuman,
			}
			sessionStore.Sessions["c1"] = make(map[string]storage.SessionRecord, tc.existingCount)
			for i := 0; i < tc.existingCount; i++ {
				id := "existing-" + string(rune('1'+i))
				sessionStore.Sessions["c1"][id] = storage.SessionRecord{
					ID:         id,
					CampaignID: "c1",
					Name:       "Existing",
					Status:     session.StatusEnded,
					StartedAt:  now,
					UpdatedAt:  now,
					EndedAt:    ptrTime(now),
				}
			}

			domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
				Decision: command.Accept(),
			}, resultsByType: map[command.Type]engine.Result{
				"session.start": {
					Decision: command.Accept(event.Event{
						CampaignID:  "c1",
						Type:        event.Type("session.started"),
						Timestamp:   now,
						ActorType:   event.ActorTypeSystem,
						SessionID:   "session-123",
						EntityType:  "session",
						EntityID:    "session-123",
						PayloadJSON: mustJSON(t, session.StartPayload{SessionID: "session-123", SessionName: tc.wantName}),
					}),
				},
				"session.gm_authority.set": {
					Decision: command.Accept(event.Event{
						CampaignID:  "c1",
						Type:        event.Type("session.gm_authority_set"),
						Timestamp:   now,
						ActorType:   event.ActorTypeSystem,
						SessionID:   "session-123",
						EntityType:  "session",
						EntityID:    "session-123",
						PayloadJSON: []byte(`{"session_id":"session-123","participant_id":"manager-1"}`),
					}),
				},
			}}

			svc := newTestSessionService(
				Deps{
					Campaign:           campaignStore,
					Session:            sessionStore,
					Participant:        participantStore,
					SessionInteraction: &fakeSessionInteractionStore{},
					Write:              domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
				},
				gametest.FixedClock(now),
				gametest.FixedIDGenerator("session-123"),
			)

			resp, err := svc.StartSession(gametest.ContextWithParticipantID("manager-1"), &statev1.StartSessionRequest{
				CampaignId: "c1",
				Name:       "   ",
			})
			if err != nil {
				t.Fatalf("StartSession returned error: %v", err)
			}
			if got := resp.GetSession().GetName(); got != tc.wantName {
				t.Fatalf("response session name = %q, want %q", got, tc.wantName)
			}
			if len(domain.commands) == 0 {
				t.Fatal("expected captured commands")
			}

			var payload session.StartPayload
			if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
				t.Fatalf("unmarshal payload: %v", err)
			}
			if payload.SessionName != tc.wantName {
				t.Fatalf("payload session name = %q, want %q", payload.SessionName, tc.wantName)
			}
			if got := sessionStartLocale(tc.campaignLocale); got != tc.wantLocale {
				t.Fatalf("sessionStartLocale(%q) = %v, want %v", tc.campaignLocale, got, tc.wantLocale)
			}
		})
	}
}

func TestStartSession_UsesDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	domain := &fakeDomainEngine{store: eventStore, result: engine.Result{
		Decision: command.Accept(),
	}, resultsByType: map[command.Type]engine.Result{
		"session.start": {
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
		"session.gm_authority.set": {
			Decision: command.Accept(event.Event{
				CampaignID:  "c1",
				Type:        event.Type("session.gm_authority_set"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				SessionID:   "session-123",
				EntityType:  "session",
				EntityID:    "session-123",
				PayloadJSON: []byte(`{"session_id":"session-123","participant_id":"manager-1"}`),
			}),
		},
	}}

	svc := newTestSessionService(
		Deps{
			Campaign:           campaignStore,
			Session:            sessionStore,
			Participant:        participantStore,
			SessionInteraction: &fakeSessionInteractionStore{},
			Write:              domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("session-123"),
	)

	_, err := svc.StartSession(gametest.ContextWithParticipantID("manager-1"), &statev1.StartSessionRequest{
		CampaignId: "c1",
		Name:       "First Session",
	})
	if err != nil {
		t.Fatalf("StartSession returned error: %v", err)
	}
	if domain.calls != 2 {
		t.Fatalf("expected domain to be called twice, got %d", domain.calls)
	}
	if len(domain.commands) == 0 || domain.commands[0].Type != command.Type("session.start") {
		t.Fatalf("first command type = %v, want %s", domain.commands, "session.start")
	}
}

func ptrTime(value time.Time) *time.Time {
	return &value
}
