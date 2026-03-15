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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func sessionManagerParticipantStore(campaignID string) *gametest.FakeParticipantStore {
	store := gametest.NewFakeParticipantStore()
	record := gametest.ManagerParticipantRecord(campaignID, "manager-1")
	record.Role = participant.RoleGM
	record.Controller = participant.ControllerHuman
	store.Participants[campaignID] = map[string]storage.ParticipantRecord{"manager-1": record}
	return store
}

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
		System: bridge.SystemIDDaggerheart,
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

	// Verify campaign was activated
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
				System: bridge.SystemIDDaggerheart,
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

func TestEndSession_NilRequest(t *testing.T) {
	svc := NewSessionService(Deps{})
	_, err := svc.EndSession(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestEndSession_MissingCampaignId(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.EndSession(context.Background(), &statev1.EndSessionRequest{SessionId: "s1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestEndSession_MissingSessionId(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.EndSession(context.Background(), &statev1.EndSessionRequest{CampaignId: "c1"})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestEndSession_CampaignNotFound(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore})
	_, err := svc.EndSession(context.Background(), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestEndSession_SessionNotFound(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")

	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore, Participant: participantStore})
	_, err := svc.EndSession(gametest.ContextWithParticipantID("manager-1"), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestEndSession_DeniesMemberAccess(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := gametest.NewFakeParticipantStore()
	now := time.Now().UTC()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"member-1": {
			ID:             "member-1",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessMember,
		},
	}

	svc := NewSessionService(Deps{
		Campaign:    campaignStore,
		Session:     sessionStore,
		Participant: participantStore,
	})
	_, err := svc.EndSession(gametest.ContextWithParticipantID("member-1"), &statev1.EndSessionRequest{
		CampaignId: "c1",
		SessionId:  "s1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestEndSession_RequiresDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	sessionStore.ActiveSession["c1"] = "s1"

	svc := NewSessionService(Deps{Campaign: campaignStore, Session: sessionStore, Participant: participantStore})
	_, err := svc.EndSession(gametest.ContextWithParticipantID("manager-1"), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	assertStatusCode(t, err, codes.Internal)
}

func TestEndSession_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	sessionStore.ActiveSession["c1"] = "s1"
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

	svc := newTestSessionService(
		Deps{
			Campaign:    campaignStore,
			Session:     sessionStore,
			Participant: participantStore,
			Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("session-123"),
	)

	resp, err := svc.EndSession(gametest.ContextWithParticipantID("manager-1"), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	if err != nil {
		t.Fatalf("EndSession returned error: %v", err)
	}
	if resp.Session.Status != statev1.SessionStatus_SESSION_ENDED {
		t.Errorf("Session Status = %v, want %v", resp.Session.Status, statev1.SessionStatus_SESSION_ENDED)
	}
	if resp.Session.EndedAt == nil {
		t.Error("Session EndedAt is nil")
	}
	if got := len(eventStore.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.Events["c1"][0].Type != event.Type("session.ended") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][0].Type, event.Type("session.ended"))
	}
}

func TestEndSession_UsesDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now},
	}
	sessionStore.ActiveSession["c1"] = "s1"

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

	svc := newTestSessionService(
		Deps{
			Campaign:    campaignStore,
			Session:     sessionStore,
			Participant: participantStore,
			Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		nil,
	)

	_, err := svc.EndSession(gametest.ContextWithParticipantID("manager-1"), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
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
	svc := NewSessionService(Deps{})
	_, err := svc.AbandonSessionGate(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSessionGate_MissingCampaignId(t *testing.T) {
	svc := NewSessionService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Session:     gametest.NewFakeSessionStore(),
		SessionGate: gametest.NewFakeSessionGateStore(),
	})
	_, err := svc.AbandonSessionGate(context.Background(), &statev1.AbandonSessionGateRequest{
		SessionId: "s1", GateId: "g1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSessionGate_MissingSessionId(t *testing.T) {
	svc := NewSessionService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Session:     gametest.NewFakeSessionStore(),
		SessionGate: gametest.NewFakeSessionGateStore(),
	})
	_, err := svc.AbandonSessionGate(context.Background(), &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", GateId: "g1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSessionGate_MissingGateId(t *testing.T) {
	svc := NewSessionService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Session:     gametest.NewFakeSessionStore(),
		SessionGate: gametest.NewFakeSessionGateStore(),
	})
	_, err := svc.AbandonSessionGate(context.Background(), &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", SessionId: "s1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestAbandonSessionGate_CampaignNotFound(t *testing.T) {
	svc := NewSessionService(Deps{
		Campaign:    gametest.NewFakeCampaignStore(),
		Session:     gametest.NewFakeSessionStore(),
		SessionGate: gametest.NewFakeSessionGateStore(),
	})
	_, err := svc.AbandonSessionGate(context.Background(), &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", SessionId: "s1", GateId: "g1",
	})
	assertStatusCode(t, err, codes.NotFound)
}

func TestAbandonSessionGate_DeniesMemberAccess(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	gateStore := gametest.NewFakeSessionGateStore()
	participantStore := gametest.NewFakeParticipantStore()
	now := time.Now().UTC()

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	gateStore.Gates["c1:s1:g1"] = storage.SessionGate{
		CampaignID: "c1", SessionID: "s1", GateID: "g1",
		GateType: "decision", Status: session.GateStatusOpen,
		CreatedAt: now,
	}
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
		"member-1": {
			ID:             "member-1",
			CampaignID:     "c1",
			CampaignAccess: participant.CampaignAccessMember,
		},
	}

	svc := NewSessionService(Deps{
		Campaign:    campaignStore,
		Session:     sessionStore,
		SessionGate: gateStore,
		Participant: participantStore,
	})
	_, err := svc.AbandonSessionGate(gametest.ContextWithParticipantID("member-1"), &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", SessionId: "s1", GateId: "g1",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestAbandonSessionGate_AlreadyAbandoned(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	gateStore := gametest.NewFakeSessionGateStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	gateStore.Gates["c1:s1:g1"] = storage.SessionGate{
		CampaignID: "c1", SessionID: "s1", GateID: "g1",
		GateType: "decision", Status: session.GateStatusAbandoned,
		CreatedAt: now,
	}

	svc := newTestSessionService(
		Deps{
			Campaign:    campaignStore,
			Session:     sessionStore,
			SessionGate: gateStore,
			Participant: participantStore,
		},
		gametest.FixedClock(now),
		nil,
	)

	resp, err := svc.AbandonSessionGate(gametest.ContextWithParticipantID("manager-1"), &statev1.AbandonSessionGateRequest{
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
	if len(eventStore.Events["c1"]) != 0 {
		t.Fatalf("expected 0 events for already-abandoned gate, got %d", len(eventStore.Events["c1"]))
	}
}

func TestAbandonSessionGate_Success(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	gateStore := gametest.NewFakeSessionGateStore()
	participantStore := gametest.NewFakeParticipantStore()
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	gateStore.Gates["c1:s1:g1"] = storage.SessionGate{
		CampaignID: "c1", SessionID: "s1", GateID: "g1",
		GateType: "decision", Status: session.GateStatusOpen,
		CreatedAt: now,
	}
	participantStore.Participants["c1"] = map[string]storage.ParticipantRecord{
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

	svc := newTestSessionService(
		Deps{
			Campaign:    campaignStore,
			Session:     sessionStore,
			SessionGate: gateStore,
			Participant: participantStore,
			Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		nil,
	)

	ctx := gametest.ContextWithParticipantID("part-1")
	resp, err := svc.AbandonSessionGate(ctx, &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", SessionId: "s1", GateId: "g1", Reason: "timeout",
	})
	if err != nil {
		t.Fatalf("AbandonSessionGate returned error: %v", err)
	}
	if resp.GetGate() == nil {
		t.Fatal("expected gate in response")
	}
	if got := len(eventStore.Events["c1"]); got != 1 {
		t.Fatalf("expected 1 event, got %d", got)
	}
	if eventStore.Events["c1"][0].Type != event.Type("session.gate_abandoned") {
		t.Fatalf("event type = %s, want %s", eventStore.Events["c1"][0].Type, event.Type("session.gate_abandoned"))
	}
}

func TestAbandonSessionGate_RequiresDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	gateStore := gametest.NewFakeSessionGateStore()
	participantStore := sessionManagerParticipantStore("c1")
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	gateStore.Gates["c1:s1:g1"] = storage.SessionGate{
		CampaignID: "c1", SessionID: "s1", GateID: "g1",
		GateType: "decision", Status: session.GateStatusOpen,
		CreatedAt: now,
	}

	svc := newTestSessionService(
		Deps{
			Campaign:    campaignStore,
			Session:     sessionStore,
			SessionGate: gateStore,
			Participant: participantStore,
		},
		nil,
		nil,
	)
	_, err := svc.AbandonSessionGate(gametest.ContextWithParticipantID("manager-1"), &statev1.AbandonSessionGateRequest{
		CampaignId: "c1", SessionId: "s1", GateId: "g1",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestOpenSessionGate_UsesDomainEngine(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	gateStore := gametest.NewFakeSessionGateStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
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

	svc := newTestSessionService(
		Deps{
			Campaign:    campaignStore,
			Session:     sessionStore,
			SessionGate: gateStore,
			Participant: participantStore,
			Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		nil,
	)

	_, err := svc.OpenSessionGate(gametest.ContextWithParticipantID("manager-1"), &statev1.OpenSessionGateRequest{
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
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	gateStore := gametest.NewFakeSessionGateStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	gateStore.Gates["c1:s1:g1"] = storage.SessionGate{
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

	svc := newTestSessionService(
		Deps{
			Campaign:    campaignStore,
			Session:     sessionStore,
			SessionGate: gateStore,
			Participant: participantStore,
			Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		nil,
	)

	_, err := svc.ResolveSessionGate(gametest.ContextWithParticipantID("manager-1"), &statev1.ResolveSessionGateRequest{
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
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	gateStore := gametest.NewFakeSessionGateStore()
	participantStore := sessionManagerParticipantStore("c1")
	eventStore := gametest.NewFakeEventStore()
	now := time.Date(2025, 2, 1, 12, 0, 0, 0, time.UTC)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now, UpdatedAt: now},
	}
	gateStore.Gates["c1:s1:g1"] = storage.SessionGate{
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

	svc := newTestSessionService(
		Deps{
			Campaign:    campaignStore,
			Session:     sessionStore,
			SessionGate: gateStore,
			Participant: participantStore,
			Write:       domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
		},
		gametest.FixedClock(now),
		nil,
	)

	_, err := svc.AbandonSessionGate(gametest.ContextWithParticipantID("manager-1"), &statev1.AbandonSessionGateRequest{
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

func TestEndSession_AlreadyEnded(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	participantStore := sessionManagerParticipantStore("c1")
	now := time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC)
	endedAt := now.Add(-1 * time.Hour)

	campaignStore.Campaigns["c1"] = gametest.ActiveCampaignRecord("c1")
	sessionStore.Sessions["c1"] = map[string]storage.SessionRecord{
		"s1": {ID: "s1", CampaignID: "c1", Status: session.StatusEnded, StartedAt: now.Add(-2 * time.Hour), EndedAt: &endedAt},
	}

	svc := newTestSessionService(
		Deps{
			Campaign:    campaignStore,
			Session:     sessionStore,
			Participant: participantStore,
		},
		gametest.FixedClock(now),
		gametest.FixedIDGenerator("session-123"),
	)

	resp, err := svc.EndSession(gametest.ContextWithParticipantID("manager-1"), &statev1.EndSessionRequest{CampaignId: "c1", SessionId: "s1"})
	if err != nil {
		t.Fatalf("EndSession returned error: %v", err)
	}
	if resp.Session.Status != statev1.SessionStatus_SESSION_ENDED {
		t.Errorf("Session Status = %v, want %v", resp.Session.Status, statev1.SessionStatus_SESSION_ENDED)
	}
}
