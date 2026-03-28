package interactiontransport

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/runtimekit"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

const concludeSessionSummaryMarkdown = `## Key Events

The party sealed the breach.

## NPCs Met

Captain Vale.

## Decisions Made

They chose to end the war at the harbor.

## Unresolved Threads

Who armed the raiders?

## Next Session Hooks

Return to the city in triumph.`

type concludeSceneStore struct {
	scenes map[string]storage.SceneRecord
}

func (s *concludeSceneStore) GetScene(_ context.Context, campaignID, sceneID string) (storage.SceneRecord, error) {
	record, ok := s.scenes[campaignID+":"+sceneID]
	if !ok {
		return storage.SceneRecord{}, storage.ErrNotFound
	}
	return record, nil
}

func (s *concludeSceneStore) ListScenes(_ context.Context, campaignID, sessionID string, _ int, _ string) (storage.ScenePage, error) {
	result := make([]storage.SceneRecord, 0, len(s.scenes))
	for _, record := range s.scenes {
		if record.CampaignID == campaignID && record.SessionID == sessionID {
			result = append(result, record)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].SceneID < result[j].SceneID
	})
	return storage.ScenePage{Scenes: result}, nil
}

func (s *concludeSceneStore) ListOpenScenes(_ context.Context, campaignID string) ([]storage.SceneRecord, error) {
	result := make([]storage.SceneRecord, 0, len(s.scenes))
	for _, record := range s.scenes {
		if record.CampaignID == campaignID && record.EndedAt == nil && record.Open {
			result = append(result, record)
		}
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].SceneID < result[j].SceneID
	})
	return result, nil
}

func (*concludeSceneStore) ListVisibleOpenScenesForCharacters(context.Context, string, string, []string) ([]storage.SceneRecord, error) {
	return nil, nil
}

func (s *concludeSceneStore) PutScene(_ context.Context, record storage.SceneRecord) error {
	s.scenes[record.CampaignID+":"+record.SceneID] = record
	return nil
}

func (s *concludeSceneStore) EndScene(_ context.Context, campaignID, sceneID string, endedAt time.Time) error {
	record, ok := s.scenes[campaignID+":"+sceneID]
	if !ok {
		return storage.ErrNotFound
	}
	record.Open = false
	record.EndedAt = &endedAt
	record.UpdatedAt = endedAt
	s.scenes[campaignID+":"+sceneID] = record
	return nil
}

type concludeSceneInteractionStore struct {
	values map[string]storage.SceneInteraction
}

func (s *concludeSceneInteractionStore) GetSceneInteraction(_ context.Context, campaignID, sceneID string) (storage.SceneInteraction, error) {
	record, ok := s.values[campaignID+":"+sceneID]
	if !ok {
		return storage.SceneInteraction{}, storage.ErrNotFound
	}
	return record, nil
}

func (s *concludeSceneInteractionStore) PutSceneInteraction(_ context.Context, interaction storage.SceneInteraction) error {
	s.values[interaction.CampaignID+":"+interaction.SceneID] = interaction
	return nil
}

type concludeSceneGMInteractionStore struct {
	values map[string][]storage.SceneGMInteraction
}

func (s *concludeSceneGMInteractionStore) ListSceneGMInteractions(_ context.Context, campaignID, sceneID string) ([]storage.SceneGMInteraction, error) {
	items := s.values[campaignID+":"+sceneID]
	return append([]storage.SceneGMInteraction(nil), items...), nil
}

func (s *concludeSceneGMInteractionStore) PutSceneGMInteraction(_ context.Context, interaction storage.SceneGMInteraction) error {
	key := interaction.CampaignID + ":" + interaction.SceneID
	s.values[key] = append(s.values[key], interaction)
	return nil
}

type concludeSceneSpotlightStore struct{}

func (concludeSceneSpotlightStore) GetSceneSpotlight(context.Context, string, string) (storage.SceneSpotlight, error) {
	return storage.SceneSpotlight{}, storage.ErrNotFound
}

func (concludeSceneSpotlightStore) PutSceneSpotlight(context.Context, storage.SceneSpotlight) error {
	return nil
}

func (concludeSceneSpotlightStore) ClearSceneSpotlight(context.Context, string, string) error {
	return storage.ErrNotFound
}

type concludeSessionExecutor struct {
	now time.Time
}

func (e *concludeSessionExecutor) Execute(_ context.Context, cmd command.Command) (engine.Result, error) {
	evt, err := e.buildEvent(cmd)
	if err != nil {
		return engine.Result{}, err
	}
	return engine.Result{Decision: command.Accept(evt)}, nil
}

func (e *concludeSessionExecutor) buildEvent(cmd command.Command) (event.Event, error) {
	timestamp := e.now.UTC()
	e.now = e.now.Add(time.Second)

	switch cmd.Type {
	case commandTypeSceneGMInteractionCommit:
		return command.NewEvent(cmd, scene.EventTypeGMInteractionCommitted, "scene", strings.TrimSpace(cmd.EntityID), cmd.PayloadJSON, timestamp), nil
	case commandTypeSessionRecapRecord:
		return command.NewEvent(cmd, session.EventTypeRecapRecorded, "session", strings.TrimSpace(cmd.EntityID), cmd.PayloadJSON, timestamp), nil
	case commandTypeSceneEnd:
		return command.NewEvent(cmd, scene.EventTypeEnded, "scene", strings.TrimSpace(cmd.EntityID), cmd.PayloadJSON, timestamp), nil
	case commandTypeSessionEnd:
		return command.NewEvent(cmd, session.EventTypeEnded, "session", strings.TrimSpace(cmd.EntityID), cmd.PayloadJSON, timestamp), nil
	case commandTypeCampaignEnd:
		payloadJSON, err := json.Marshal(campaign.UpdatePayload{
			Fields: map[string]string{"status": string(campaign.StatusCompleted)},
		})
		if err != nil {
			return event.Event{}, err
		}
		return command.NewEvent(cmd, campaign.EventTypeUpdated, "campaign", string(cmd.CampaignID), payloadJSON, timestamp), nil
	default:
		return event.Event{}, fmt.Errorf("unexpected command type %s", cmd.Type)
	}
}

type concludeSessionFixture struct {
	campaign        *gametest.FakeCampaignStore
	session         *gametest.FakeSessionStore
	recap           *gametest.FakeSessionRecapStore
	sessionState    *gametest.FakeSessionInteractionStore
	scene           *concludeSceneStore
	sceneState      *concludeSceneInteractionStore
	sceneGM         *concludeSceneGMInteractionStore
	app             AIOrchestrationApplication
	activeSceneID   string
	campaignID      string
	sessionID       string
	gmParticipantID string
}

func newConcludeSessionFixture(t *testing.T, now time.Time) concludeSessionFixture {
	t.Helper()

	campaignStore := gametest.NewFakeCampaignStore()
	sessionStore := gametest.NewFakeSessionStore()
	recapStore := gametest.NewFakeSessionRecapStore()
	sessionInteractionStore := gametest.NewFakeSessionInteractionStore()
	sceneStore := &concludeSceneStore{scenes: make(map[string]storage.SceneRecord)}
	sceneInteractionStore := &concludeSceneInteractionStore{values: make(map[string]storage.SceneInteraction)}
	sceneGMInteractionStore := &concludeSceneGMInteractionStore{values: make(map[string][]storage.SceneGMInteraction)}

	runtime := runtimekit.SetupRuntime()
	app := NewAIOrchestrationApplication(Deps{
		Campaign:           campaignStore,
		Session:            sessionStore,
		SessionRecap:       recapStore,
		SessionInteraction: sessionInteractionStore,
		Scene:              sceneStore,
		SceneInteraction:   sceneInteractionStore,
		SceneGMInteraction: sceneGMInteractionStore,
		Write: domainwrite.WritePath{
			Executor: &concludeSessionExecutor{now: now},
			Runtime:  runtime,
		},
		Applier: projection.Applier{
			Campaign:           campaignStore,
			Session:            sessionStore,
			SessionRecap:       recapStore,
			SessionInteraction: sessionInteractionStore,
			Scene:              sceneStore,
			SceneInteraction:   sceneInteractionStore,
			SceneGMInteraction: sceneGMInteractionStore,
			SceneSpotlight:     concludeSceneSpotlightStore{},
		},
	}, runtimekit.FixedIDGenerator("gm-interaction-1"))

	return concludeSessionFixture{
		campaign:        campaignStore,
		session:         sessionStore,
		recap:           recapStore,
		sessionState:    sessionInteractionStore,
		scene:           sceneStore,
		sceneState:      sceneInteractionStore,
		sceneGM:         sceneGMInteractionStore,
		app:             app,
		activeSceneID:   "scene-1",
		campaignID:      "camp-1",
		sessionID:       "sess-1",
		gmParticipantID: "gm-1",
	}
}

func (f concludeSessionFixture) seedActiveSession(openSceneIDs ...string) {
	f.campaign.Campaigns[f.campaignID] = storage.CampaignRecord{
		ID:     f.campaignID,
		Status: campaign.StatusActive,
	}
	f.session.Sessions[f.campaignID] = map[string]storage.SessionRecord{
		f.sessionID: {
			ID:         f.sessionID,
			CampaignID: f.campaignID,
			Name:       "Final Session",
			Status:     session.StatusActive,
			StartedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		},
	}
	f.session.ActiveSession[f.campaignID] = f.sessionID
	f.sessionState.Values = map[string]storage.SessionInteraction{
		f.campaignID + ":" + f.sessionID: {
			CampaignID:               f.campaignID,
			SessionID:                f.sessionID,
			ActiveSceneID:            f.activeSceneID,
			GMAuthorityParticipantID: f.gmParticipantID,
		},
	}
	for _, sceneID := range openSceneIDs {
		f.scene.scenes[f.campaignID+":"+sceneID] = storage.SceneRecord{
			CampaignID: f.campaignID,
			SceneID:    sceneID,
			SessionID:  f.sessionID,
			Name:       "Scene " + sceneID,
			Open:       true,
			CreatedAt:  time.Now().UTC(),
			UpdatedAt:  time.Now().UTC(),
		}
		f.sceneState.values[f.campaignID+":"+sceneID] = storage.SceneInteraction{
			CampaignID: f.campaignID,
			SceneID:    sceneID,
			SessionID:  f.sessionID,
			PhaseOpen:  sceneID == f.activeSceneID,
			PhaseID:    "phase-" + sceneID,
		}
	}
}

func TestConcludeSessionEndCampaignCompletesCampaign(t *testing.T) {
	t.Parallel()

	fixture := newConcludeSessionFixture(t, time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC))
	fixture.seedActiveSession("scene-1", "scene-2")

	result, err := fixture.app.ConcludeSession(
		context.Background(),
		fixture.campaignID,
		fixture.sessionID,
		"The harbor quiets as dawn breaks over the ruined gate.",
		concludeSessionSummaryMarkdown,
		true,
		"Peace holds at the harbor for the first time in a generation.",
	)
	if err != nil {
		t.Fatalf("ConcludeSession() error = %v", err)
	}
	if !result.CampaignCompleted {
		t.Fatal("campaign_completed = false, want true")
	}
	if result.SessionID != fixture.sessionID {
		t.Fatalf("session_id = %q, want %q", result.SessionID, fixture.sessionID)
	}
	if got, want := strings.Join(result.EndedSceneIDs, ","), "scene-1,scene-2"; got != want {
		t.Fatalf("ended_scene_ids = %q, want %q", got, want)
	}

	campaignRecord, err := fixture.campaign.Get(context.Background(), fixture.campaignID)
	if err != nil {
		t.Fatalf("campaign.Get() error = %v", err)
	}
	if campaignRecord.Status != campaign.StatusCompleted {
		t.Fatalf("campaign status = %q, want %q", campaignRecord.Status, campaign.StatusCompleted)
	}
	if campaignRecord.CompletedAt == nil {
		t.Fatal("campaign completed_at = nil, want timestamp")
	}

	sessionRecord, err := fixture.session.GetSession(context.Background(), fixture.campaignID, fixture.sessionID)
	if err != nil {
		t.Fatalf("session.GetSession() error = %v", err)
	}
	if sessionRecord.Status != session.StatusEnded {
		t.Fatalf("session status = %q, want %q", sessionRecord.Status, session.StatusEnded)
	}

	recap, err := fixture.recap.GetSessionRecap(context.Background(), fixture.campaignID, fixture.sessionID)
	if err != nil {
		t.Fatalf("GetSessionRecap() error = %v", err)
	}
	if !strings.Contains(recap.Markdown, "## Campaign Epilogue") {
		t.Fatalf("recap markdown = %q, want campaign epilogue heading", recap.Markdown)
	}
	if !strings.Contains(recap.Markdown, "Peace holds at the harbor") {
		t.Fatalf("recap markdown = %q, want epilogue text", recap.Markdown)
	}

	for _, sceneID := range []string{"scene-1", "scene-2"} {
		record, err := fixture.scene.GetScene(context.Background(), fixture.campaignID, sceneID)
		if err != nil {
			t.Fatalf("scene.GetScene(%q) error = %v", sceneID, err)
		}
		if record.EndedAt == nil {
			t.Fatalf("scene %s ended_at = nil, want timestamp", sceneID)
		}
	}
	interactions, err := fixture.sceneGM.ListSceneGMInteractions(context.Background(), fixture.campaignID, fixture.activeSceneID)
	if err != nil {
		t.Fatalf("ListSceneGMInteractions() error = %v", err)
	}
	if len(interactions) != 1 {
		t.Fatalf("gm interactions = %d, want 1", len(interactions))
	}
}

func TestConcludeSessionAlreadyEndedSessionStillCompletesCampaign(t *testing.T) {
	t.Parallel()

	fixture := newConcludeSessionFixture(t, time.Date(2026, 3, 28, 14, 0, 0, 0, time.UTC))
	endedAt := time.Date(2026, 3, 28, 13, 0, 0, 0, time.UTC)
	fixture.campaign.Campaigns[fixture.campaignID] = storage.CampaignRecord{
		ID:     fixture.campaignID,
		Status: campaign.StatusActive,
	}
	fixture.session.Sessions[fixture.campaignID] = map[string]storage.SessionRecord{
		fixture.sessionID: {
			ID:         fixture.sessionID,
			CampaignID: fixture.campaignID,
			Name:       "Final Session",
			Status:     session.StatusEnded,
			StartedAt:  endedAt.Add(-time.Hour),
			UpdatedAt:  endedAt,
			EndedAt:    &endedAt,
		},
	}
	fixture.recap.Recaps[fixture.campaignID+":"+fixture.sessionID] = storage.SessionRecap{
		CampaignID: fixture.campaignID,
		SessionID:  fixture.sessionID,
		Markdown:   concludeSessionSummaryMarkdown + "\n\n## Campaign Epilogue\n\nThe old war is over.",
		UpdatedAt:  endedAt,
	}
	fixture.scene.scenes[fixture.campaignID+":"+"scene-1"] = storage.SceneRecord{
		CampaignID: fixture.campaignID,
		SceneID:    "scene-1",
		SessionID:  fixture.sessionID,
		Name:       "Scene scene-1",
		Open:       false,
		CreatedAt:  endedAt.Add(-2 * time.Hour),
		UpdatedAt:  endedAt,
		EndedAt:    &endedAt,
	}

	result, err := fixture.app.ConcludeSession(
		context.Background(),
		fixture.campaignID,
		fixture.sessionID,
		"Already concluded.",
		concludeSessionSummaryMarkdown,
		true,
		"The old war is over.",
	)
	if err != nil {
		t.Fatalf("ConcludeSession() error = %v", err)
	}
	if !result.CampaignCompleted {
		t.Fatal("campaign_completed = false, want true")
	}

	campaignRecord, err := fixture.campaign.Get(context.Background(), fixture.campaignID)
	if err != nil {
		t.Fatalf("campaign.Get() error = %v", err)
	}
	if campaignRecord.Status != campaign.StatusCompleted {
		t.Fatalf("campaign status = %q, want %q", campaignRecord.Status, campaign.StatusCompleted)
	}
}

func TestConcludeSessionWithoutEndCampaignLeavesCampaignActive(t *testing.T) {
	t.Parallel()

	fixture := newConcludeSessionFixture(t, time.Date(2026, 3, 28, 16, 0, 0, 0, time.UTC))
	fixture.seedActiveSession("scene-1")

	result, err := fixture.app.ConcludeSession(
		context.Background(),
		fixture.campaignID,
		fixture.sessionID,
		"The company agrees to rest before the next voyage.",
		concludeSessionSummaryMarkdown,
		false,
		"",
	)
	if err != nil {
		t.Fatalf("ConcludeSession() error = %v", err)
	}
	if result.CampaignCompleted {
		t.Fatal("campaign_completed = true, want false")
	}

	campaignRecord, err := fixture.campaign.Get(context.Background(), fixture.campaignID)
	if err != nil {
		t.Fatalf("campaign.Get() error = %v", err)
	}
	if campaignRecord.Status != campaign.StatusActive {
		t.Fatalf("campaign status = %q, want %q", campaignRecord.Status, campaign.StatusActive)
	}
	recap, err := fixture.recap.GetSessionRecap(context.Background(), fixture.campaignID, fixture.sessionID)
	if err != nil {
		t.Fatalf("GetSessionRecap() error = %v", err)
	}
	if strings.Contains(recap.Markdown, "## Campaign Epilogue") {
		t.Fatalf("recap markdown = %q, want no campaign epilogue", recap.Markdown)
	}
}
