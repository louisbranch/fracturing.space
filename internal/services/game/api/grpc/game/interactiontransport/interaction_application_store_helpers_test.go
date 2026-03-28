package interactiontransport

import (
	"context"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestInteractionApplicationExecuteSceneCommandReturnsInternalWithoutWritePath(t *testing.T) {
	t.Parallel()

	app := interactionApplication{}

	err := app.executeSceneCommand(
		context.Background(),
		commandTypeScenePlayerPhasePost,
		"camp-1",
		"sess-1",
		"scene-1",
		struct{ Summary string }{Summary: "advance"},
		"scene.player_phase.post",
	)
	assertStatusCode(t, err, codes.Internal)
}

func TestInteractionApplicationEndScenePhaseAndYieldHelpers(t *testing.T) {
	t.Parallel()

	app := interactionApplication{
		stores: interactionApplicationStores{
			Scene: interactionSceneStoreStub{
				scenes: map[string]storage.SceneRecord{
					"camp-1:scene-1": {CampaignID: "camp-1", SceneID: "scene-1", SessionID: "sess-1", Name: "Bridge"},
				},
			},
			SceneInteraction: interactionSceneInteractionStoreStub{
				interactions: map[string]storage.SceneInteraction{
					"camp-1:scene-1": {
						CampaignID:           "camp-1",
						SceneID:              "scene-1",
						SessionID:            "sess-1",
						PhaseOpen:            true,
						PhaseID:              "phase-1",
						ActingParticipantIDs: []string{"p1"},
						Slots:                []storage.ScenePlayerSlot{},
					},
				},
			},
		},
	}

	err := app.endScenePhase(context.Background(), "camp-1", "sess-1", "scene-1", "phase-1", "gm_interrupted")
	assertStatusCode(t, err, codes.Internal)

	err = app.endScenePhaseIfOpen(context.Background(), "camp-1", "scene-1", "all_yielded")
	assertStatusCode(t, err, codes.Internal)

	err = app.yieldScenePhase(context.Background(), "camp-1", "sess-1", "scene-1", "phase-1", "p1")
	assertStatusCode(t, err, codes.Internal)
}

func TestInteractionApplicationEndScenePhaseIfOpenNoopsWithoutOpenPhase(t *testing.T) {
	t.Parallel()

	app := interactionApplication{
		stores: interactionApplicationStores{
			Scene: interactionSceneStoreStub{
				scenes: map[string]storage.SceneRecord{
					"camp-1:scene-1": {CampaignID: "camp-1", SceneID: "scene-1", SessionID: "sess-1", Name: "Bridge"},
				},
			},
			SceneInteraction: interactionSceneInteractionStoreStub{
				interactions: map[string]storage.SceneInteraction{
					"camp-1:scene-1": {
						CampaignID: "camp-1",
						SceneID:    "scene-1",
						SessionID:  "sess-1",
						PhaseOpen:  false,
					},
				},
			},
		},
	}

	if err := app.endScenePhaseIfOpen(context.Background(), "camp-1", "scene-missing", "ignored"); err != nil {
		t.Fatalf("missing scene error = %v, want nil", err)
	}
	if err := app.endScenePhaseIfOpen(context.Background(), "camp-1", "scene-1", "ignored"); err != nil {
		t.Fatalf("closed phase error = %v, want nil", err)
	}
}

func TestInteractionApplicationEndScenePhaseIfOpenNoopsWhenSceneMissingOrClosed(t *testing.T) {
	t.Parallel()

	app := interactionApplication{
		stores: interactionApplicationStores{
			Scene: interactionSceneStoreStub{
				scenes: map[string]storage.SceneRecord{
					"camp-1:scene-1": {CampaignID: "camp-1", SceneID: "scene-1", SessionID: "sess-1", Name: "Bridge"},
				},
			},
			SceneInteraction: interactionSceneInteractionStoreStub{
				interactions: map[string]storage.SceneInteraction{
					"camp-1:scene-1": {CampaignID: "camp-1", SceneID: "scene-1", SessionID: "sess-1", PhaseOpen: false},
				},
			},
		},
	}

	if err := app.endScenePhaseIfOpen(context.Background(), "camp-1", "missing", "all_yielded"); err != nil {
		t.Fatalf("missing scene error = %v", err)
	}
	if err := app.endScenePhaseIfOpen(context.Background(), "camp-1", "scene-1", "all_yielded"); err != nil {
		t.Fatalf("closed phase error = %v", err)
	}
}

func TestAITurnStatusToProto(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status session.AITurnStatus
		want   statev1.AITurnStatus
	}{
		{name: "idle", status: session.AITurnStatusIdle, want: statev1.AITurnStatus_AI_TURN_STATUS_IDLE},
		{name: "queued", status: session.AITurnStatusQueued, want: statev1.AITurnStatus_AI_TURN_STATUS_QUEUED},
		{name: "running", status: session.AITurnStatusRunning, want: statev1.AITurnStatus_AI_TURN_STATUS_RUNNING},
		{name: "failed", status: session.AITurnStatusFailed, want: statev1.AITurnStatus_AI_TURN_STATUS_FAILED},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := aiTurnStatusToProto(tc.status); got != tc.want {
				t.Fatalf("aiTurnStatusToProto(%q) = %v, want %v", tc.status, got, tc.want)
			}
		})
	}
}

type interactionSceneStoreStub struct {
	scenes map[string]storage.SceneRecord
	err    error
}

func (s interactionSceneStoreStub) GetScene(_ context.Context, campaignID, sceneID string) (storage.SceneRecord, error) {
	if s.err != nil {
		return storage.SceneRecord{}, s.err
	}
	record, ok := s.scenes[campaignID+":"+sceneID]
	if !ok {
		return storage.SceneRecord{}, storage.ErrNotFound
	}
	return record, nil
}

func (interactionSceneStoreStub) ListScenes(context.Context, string, string, int, string) (storage.ScenePage, error) {
	return storage.ScenePage{}, nil
}

func (interactionSceneStoreStub) ListOpenScenes(context.Context, string) ([]storage.SceneRecord, error) {
	return nil, nil
}

func (interactionSceneStoreStub) ListVisibleOpenScenesForCharacters(context.Context, string, string, []string) ([]storage.SceneRecord, error) {
	return nil, nil
}

func (interactionSceneStoreStub) PutScene(context.Context, storage.SceneRecord) error {
	return nil
}

func (interactionSceneStoreStub) EndScene(context.Context, string, string, time.Time) error {
	return nil
}

type interactionSceneCharacterStoreStub struct {
	records map[string][]storage.SceneCharacterRecord
	err     error
}

func (s interactionSceneCharacterStoreStub) ListSceneCharacters(_ context.Context, campaignID, sceneID string) ([]storage.SceneCharacterRecord, error) {
	if s.err != nil {
		return nil, s.err
	}
	return append([]storage.SceneCharacterRecord(nil), s.records[campaignID+":"+sceneID]...), nil
}

func (interactionSceneCharacterStoreStub) PutSceneCharacter(context.Context, storage.SceneCharacterRecord) error {
	return nil
}

func (interactionSceneCharacterStoreStub) DeleteSceneCharacter(context.Context, string, string, string) error {
	return nil
}

type interactionSessionInteractionStoreStub struct {
	interactions map[string]storage.SessionInteraction
	err          error
}

func (s interactionSessionInteractionStoreStub) GetSessionInteraction(_ context.Context, campaignID, sessionID string) (storage.SessionInteraction, error) {
	if s.err != nil {
		return storage.SessionInteraction{}, s.err
	}
	interaction, ok := s.interactions[campaignID+":"+sessionID]
	if !ok {
		return storage.SessionInteraction{}, storage.ErrNotFound
	}
	return interaction, nil
}

func (interactionSessionInteractionStoreStub) PutSessionInteraction(context.Context, storage.SessionInteraction) error {
	return nil
}

type interactionSceneInteractionStoreStub struct {
	interactions map[string]storage.SceneInteraction
	err          error
}

func (s interactionSceneInteractionStoreStub) GetSceneInteraction(_ context.Context, campaignID, sceneID string) (storage.SceneInteraction, error) {
	if s.err != nil {
		return storage.SceneInteraction{}, s.err
	}
	interaction, ok := s.interactions[campaignID+":"+sceneID]
	if !ok {
		return storage.SceneInteraction{}, storage.ErrNotFound
	}
	return interaction, nil
}

func (interactionSceneInteractionStoreStub) PutSceneInteraction(context.Context, storage.SceneInteraction) error {
	return nil
}

type interactionSceneGMInteractionStoreStub struct {
	interactions map[string][]storage.SceneGMInteraction
	err          error
}

func (s interactionSceneGMInteractionStoreStub) ListSceneGMInteractions(_ context.Context, campaignID, sceneID string) ([]storage.SceneGMInteraction, error) {
	if s.err != nil {
		return nil, s.err
	}
	return append([]storage.SceneGMInteraction(nil), s.interactions[campaignID+":"+sceneID]...), nil
}

func (interactionSceneGMInteractionStoreStub) PutSceneGMInteraction(context.Context, storage.SceneGMInteraction) error {
	return nil
}

func TestInteractionApplicationLoadActiveSessionInteractionDefaultsMissingProjection(t *testing.T) {
	t.Parallel()

	sessionStore := gametest.NewFakeSessionStore()
	sessionStore.Sessions["camp-1"] = map[string]storage.SessionRecord{
		"sess-1": {ID: "sess-1", CampaignID: "camp-1", Name: "Session One", Status: session.StatusActive},
	}
	sessionStore.ActiveSession["camp-1"] = "sess-1"

	app := interactionApplication{
		stores: interactionApplicationStores{
			Session:            sessionStore,
			SessionInteraction: interactionSessionInteractionStoreStub{},
		},
	}

	active, interaction, err := app.loadActiveSessionInteraction(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("loadActiveSessionInteraction error = %v", err)
	}
	if active == nil || active.ID != "sess-1" {
		t.Fatalf("active session = %#v", active)
	}
	if interaction.SessionID != "sess-1" || len(interaction.OOCPosts) != 0 || len(interaction.ReadyToResumeParticipantIDs) != 0 {
		t.Fatalf("interaction = %#v", interaction)
	}
}

func TestInteractionApplicationLoadSceneStateSortsCharactersAndDefaultsInteraction(t *testing.T) {
	t.Parallel()

	characters := gametest.NewFakeCharacterStore()
	_ = characters.PutCharacter(context.Background(), storage.CharacterRecord{CampaignID: "camp-1", ID: "char-2", Name: "Borin", OwnerParticipantID: "p2"})
	_ = characters.PutCharacter(context.Background(), storage.CharacterRecord{CampaignID: "camp-1", ID: "char-1", Name: "Aria", OwnerParticipantID: "p1"})

	app := interactionApplication{
		stores: interactionApplicationStores{
			Character: characters,
			SceneCharacter: interactionSceneCharacterStoreStub{
				records: map[string][]storage.SceneCharacterRecord{
					"camp-1:scene-1": {
						{CampaignID: "camp-1", SceneID: "scene-1", CharacterID: "char-2"},
						{CampaignID: "camp-1", SceneID: "scene-1", CharacterID: "missing"},
						{CampaignID: "camp-1", SceneID: "scene-1", CharacterID: "char-1"},
					},
				},
			},
			SceneInteraction:   interactionSceneInteractionStoreStub{},
			SceneGMInteraction: interactionSceneGMInteractionStoreStub{},
		},
	}

	sceneView, interaction, err := app.loadSceneState(context.Background(), "camp-1", storage.SceneRecord{
		CampaignID:  "camp-1",
		SceneID:     "scene-1",
		SessionID:   "sess-1",
		Name:        "Bridge",
		Description: "A rope bridge.",
	}, storage.SessionInteraction{})
	if err != nil {
		t.Fatalf("loadSceneState error = %v", err)
	}
	if sceneView.GetCharacters()[0].GetName() != "Aria" || sceneView.GetCharacters()[1].GetName() != "Borin" {
		t.Fatalf("scene characters = %#v, want sorted surviving records", sceneView.GetCharacters())
	}
	if interaction.SceneID != "scene-1" || len(interaction.Slots) != 0 || len(interaction.ActingParticipantIDs) != 0 {
		t.Fatalf("interaction = %#v", interaction)
	}
}

func TestInteractionApplicationLoadSceneStateDefaultsMissingGMInteractionStore(t *testing.T) {
	t.Parallel()

	characters := gametest.NewFakeCharacterStore()
	_ = characters.PutCharacter(context.Background(), storage.CharacterRecord{CampaignID: "camp-1", ID: "char-1", Name: "Aria", OwnerParticipantID: "p1"})

	app := interactionApplication{
		stores: interactionApplicationStores{
			Character: characters,
			SceneCharacter: interactionSceneCharacterStoreStub{
				records: map[string][]storage.SceneCharacterRecord{
					"camp-1:scene-1": {
						{CampaignID: "camp-1", SceneID: "scene-1", CharacterID: "char-1"},
					},
				},
			},
			SceneInteraction: interactionSceneInteractionStoreStub{},
		},
	}

	sceneView, interaction, err := app.loadSceneState(context.Background(), "camp-1", storage.SceneRecord{
		CampaignID:  "camp-1",
		SceneID:     "scene-1",
		SessionID:   "sess-1",
		Name:        "Bridge",
		Description: "A rope bridge.",
	}, storage.SessionInteraction{})
	if err != nil {
		t.Fatalf("loadSceneState error = %v", err)
	}
	if sceneView.GetCurrentInteraction() != nil || len(sceneView.GetInteractionHistory()) != 0 {
		t.Fatalf("scene interactions = %#v, want empty defaults", sceneView)
	}
	if interaction.SceneID != "scene-1" || len(interaction.Slots) != 0 {
		t.Fatalf("interaction = %#v", interaction)
	}
}

func TestInteractionApplicationRequireActiveScenePhaseGuardsAndHappyPath(t *testing.T) {
	t.Parallel()

	app := interactionApplication{
		stores: interactionApplicationStores{
			Scene: interactionSceneStoreStub{
				scenes: map[string]storage.SceneRecord{
					"camp-1:scene-1": {CampaignID: "camp-1", SceneID: "scene-1", SessionID: "sess-1", Name: "Bridge"},
				},
			},
			SceneInteraction: interactionSceneInteractionStoreStub{
				interactions: map[string]storage.SceneInteraction{
					"camp-1:scene-1": {
						CampaignID:           "camp-1",
						SceneID:              "scene-1",
						SessionID:            "sess-1",
						PhaseOpen:            true,
						PhaseID:              "phase-1",
						ActingCharacterIDs:   []string{"char-1"},
						ActingParticipantIDs: []string{"p1"},
						Slots:                []storage.ScenePlayerSlot{},
					},
				},
			},
		},
	}

	_, _, err := app.requireActiveScenePhase(context.Background(), "camp-1", "sess-1", "scene-1", storage.SessionInteraction{OOCPaused: true, ActiveSceneID: "scene-1"})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("ooc paused code = %v, want failed precondition", status.Code(err))
	}

	_, _, err = app.requireActiveScenePhase(context.Background(), "camp-1", "sess-1", "scene-1", storage.SessionInteraction{ActiveSceneID: "scene-2"})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("inactive scene code = %v, want failed precondition", status.Code(err))
	}

	sceneRecord, interaction, err := app.requireActiveScenePhase(context.Background(), "camp-1", "sess-1", "scene-1", storage.SessionInteraction{ActiveSceneID: "scene-1"})
	if err != nil {
		t.Fatalf("requireActiveScenePhase error = %v", err)
	}
	if sceneRecord.SceneID != "scene-1" || interaction.PhaseID != "phase-1" {
		t.Fatalf("sceneRecord=%#v interaction=%#v", sceneRecord, interaction)
	}
}

func TestInteractionApplicationResolveActingSet(t *testing.T) {
	t.Parallel()

	characters := gametest.NewFakeCharacterStore()
	_ = characters.PutCharacter(context.Background(), storage.CharacterRecord{CampaignID: "camp-1", ID: "char-1", Name: "Aria", OwnerParticipantID: "p1"})
	_ = characters.PutCharacter(context.Background(), storage.CharacterRecord{CampaignID: "camp-1", ID: "char-2", Name: "Sable", OwnerParticipantID: "p1"})
	_ = characters.PutCharacter(context.Background(), storage.CharacterRecord{CampaignID: "camp-1", ID: "char-3", Name: "Corin", OwnerParticipantID: "p2"})

	app := interactionApplication{
		stores: interactionApplicationStores{
			Character: characters,
			SceneCharacter: interactionSceneCharacterStoreStub{
				records: map[string][]storage.SceneCharacterRecord{
					"camp-1:scene-1": {
						{CampaignID: "camp-1", SceneID: "scene-1", CharacterID: "char-1"},
						{CampaignID: "camp-1", SceneID: "scene-1", CharacterID: "char-2"},
						{CampaignID: "camp-1", SceneID: "scene-1", CharacterID: "char-3"},
					},
				},
			},
		},
	}

	gotCharacters, gotParticipants, err := app.resolveActingSet(
		context.Background(),
		"camp-1",
		storage.SceneRecord{SceneID: "scene-1"},
		storage.SessionInteraction{},
		[]string{"char-1", " ", "char-2", "char-3"},
	)
	if err != nil {
		t.Fatalf("resolveActingSet error = %v", err)
	}
	if len(gotCharacters) != 3 || len(gotParticipants) != 2 {
		t.Fatalf("acting characters=%#v participants=%#v", gotCharacters, gotParticipants)
	}

	_, _, err = app.resolveActingSet(
		context.Background(),
		"camp-1",
		storage.SceneRecord{SceneID: "scene-1"},
		storage.SessionInteraction{},
		[]string{"missing"},
	)
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("missing character code = %v, want failed precondition", status.Code(err))
	}
}

func TestInteractionApplicationResolveParticipantPostCharacters(t *testing.T) {
	t.Parallel()

	characters := gametest.NewFakeCharacterStore()
	_ = characters.PutCharacter(context.Background(), storage.CharacterRecord{CampaignID: "camp-1", ID: "char-1", Name: "Aria", OwnerParticipantID: "p1"})
	_ = characters.PutCharacter(context.Background(), storage.CharacterRecord{CampaignID: "camp-1", ID: "char-2", Name: "Corin", OwnerParticipantID: "p2"})

	app := interactionApplication{
		stores: interactionApplicationStores{
			Character: characters,
			SceneCharacter: interactionSceneCharacterStoreStub{
				records: map[string][]storage.SceneCharacterRecord{
					"camp-1:scene-1": {
						{CampaignID: "camp-1", SceneID: "scene-1", CharacterID: "char-1"},
						{CampaignID: "camp-1", SceneID: "scene-1", CharacterID: "char-2"},
					},
				},
			},
		},
	}

	got, err := app.resolveParticipantPostCharacters(
		context.Background(),
		"camp-1",
		storage.SceneRecord{SceneID: "scene-1"},
		storage.SessionInteraction{},
		"p1",
		[]string{"char-1"},
		[]string{"char-1", "char-2"},
	)
	if err != nil {
		t.Fatalf("resolveParticipantPostCharacters error = %v", err)
	}
	if len(got) != 1 || string(got[0]) != "char-1" {
		t.Fatalf("character ids = %#v", got)
	}

	_, err = app.resolveParticipantPostCharacters(
		context.Background(),
		"camp-1",
		storage.SceneRecord{SceneID: "scene-1"},
		storage.SessionInteraction{},
		"p1",
		[]string{"char-2"},
		[]string{"char-1", "char-2"},
	)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("ownership code = %v, want permission denied", status.Code(err))
	}

	_, err = app.resolveParticipantPostCharacters(
		context.Background(),
		"camp-1",
		storage.SceneRecord{SceneID: "scene-1"},
		storage.SessionInteraction{},
		"p1",
		[]string{"missing"},
		[]string{"char-1", "char-2"},
	)
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("in-scene code = %v, want failed precondition", status.Code(err))
	}

	_, err = app.resolveParticipantPostCharacters(
		context.Background(),
		"camp-1",
		storage.SceneRecord{SceneID: "scene-1"},
		storage.SessionInteraction{},
		"p1",
		[]string{"char-1"},
		[]string{"char-2"},
	)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("acting set code = %v, want permission denied", status.Code(err))
	}

	_, err = app.resolveParticipantPostCharacters(
		context.Background(),
		"camp-1",
		storage.SceneRecord{SceneID: "scene-1"},
		storage.SessionInteraction{},
		"p1",
		nil,
		[]string{"char-1"},
	)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("empty ids code = %v, want invalid argument", status.Code(err))
	}
}

func TestInteractionApplicationResolveRevisionRequestsRejectsForeignCharacter(t *testing.T) {
	t.Parallel()

	characters := gametest.NewFakeCharacterStore()
	_ = characters.PutCharacter(context.Background(), storage.CharacterRecord{CampaignID: "camp-1", ID: "char-1", Name: "Aria", OwnerParticipantID: "p1"})
	_ = characters.PutCharacter(context.Background(), storage.CharacterRecord{CampaignID: "camp-1", ID: "char-2", Name: "Corin", OwnerParticipantID: "p2"})

	app := interactionApplication{
		stores: interactionApplicationStores{
			Character: characters,
			SceneCharacter: interactionSceneCharacterStoreStub{
				records: map[string][]storage.SceneCharacterRecord{
					"camp-1:scene-1": {
						{CampaignID: "camp-1", SceneID: "scene-1", CharacterID: "char-1"},
						{CampaignID: "camp-1", SceneID: "scene-1", CharacterID: "char-2"},
					},
				},
			},
		},
	}

	_, err := app.resolveRevisionRequests(
		context.Background(),
		"camp-1",
		storage.SceneRecord{SceneID: "scene-1"},
		storage.SessionInteraction{},
		storage.SceneInteraction{
			ActingCharacterIDs:   []string{"char-1", "char-2"},
			ActingParticipantIDs: []string{"p1", "p2"},
		},
		[]*statev1.ScenePlayerRevisionRequest{{
			ParticipantId: "p1",
			Reason:        "Fix the spell choice.",
			CharacterIds:  []string{"char-2"},
		}},
	)
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("foreign character code = %v, want permission denied", status.Code(err))
	}
}

func TestInteractionApplicationResolveRevisionRequestsRequiresReason(t *testing.T) {
	t.Parallel()

	characters := gametest.NewFakeCharacterStore()
	_ = characters.PutCharacter(context.Background(), storage.CharacterRecord{CampaignID: "camp-1", ID: "char-1", Name: "Aria", OwnerParticipantID: "p1"})

	app := interactionApplication{
		stores: interactionApplicationStores{
			Character: characters,
			SceneCharacter: interactionSceneCharacterStoreStub{
				records: map[string][]storage.SceneCharacterRecord{
					"camp-1:scene-1": {
						{CampaignID: "camp-1", SceneID: "scene-1", CharacterID: "char-1"},
					},
				},
			},
		},
	}

	_, err := app.resolveRevisionRequests(
		context.Background(),
		"camp-1",
		storage.SceneRecord{SceneID: "scene-1"},
		storage.SessionInteraction{},
		storage.SceneInteraction{
			ActingCharacterIDs:   []string{"char-1"},
			ActingParticipantIDs: []string{"p1"},
		},
		[]*statev1.ScenePlayerRevisionRequest{{
			ParticipantId: "p1",
			Reason:        " ",
			CharacterIds:  []string{"char-1"},
		}},
	)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("blank reason code = %v, want invalid argument", status.Code(err))
	}
}

func TestInteractionApplicationRequireActiveScenePhaseDefaultsMissingInteractionToClosed(t *testing.T) {
	t.Parallel()

	app := interactionApplication{
		stores: interactionApplicationStores{
			Scene: interactionSceneStoreStub{
				scenes: map[string]storage.SceneRecord{
					"camp-1:scene-1": {CampaignID: "camp-1", SceneID: "scene-1", SessionID: "sess-1", Name: "Bridge"},
				},
			},
			SceneInteraction: interactionSceneInteractionStoreStub{},
		},
	}

	_, _, err := app.requireActiveScenePhase(context.Background(), "camp-1", "sess-1", "scene-1", storage.SessionInteraction{ActiveSceneID: "scene-1"})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("missing interaction code = %v, want failed precondition", status.Code(err))
	}
}

func TestInteractionApplicationLoadSceneStateReturnsStoredInteraction(t *testing.T) {
	t.Parallel()

	app := interactionApplication{
		stores: interactionApplicationStores{
			Character:      gametest.NewFakeCharacterStore(),
			SceneCharacter: interactionSceneCharacterStoreStub{},
			SceneInteraction: interactionSceneInteractionStoreStub{
				interactions: map[string]storage.SceneInteraction{
					"camp-1:scene-1": {
						CampaignID:           "camp-1",
						SceneID:              "scene-1",
						SessionID:            "sess-1",
						PhaseOpen:            true,
						PhaseID:              "phase-1",
						ActingCharacterIDs:   []string{"char-1"},
						ActingParticipantIDs: []string{"p1"},
						Slots: []storage.ScenePlayerSlot{{
							ParticipantID: "p1",
							SummaryText:   "Test",
							Yielded:       true,
						}},
					},
				},
			},
			SceneGMInteraction: interactionSceneGMInteractionStoreStub{},
		},
	}

	_, interaction, err := app.loadSceneState(context.Background(), "camp-1", storage.SceneRecord{
		CampaignID: "camp-1",
		SceneID:    "scene-1",
		SessionID:  "sess-1",
	}, storage.SessionInteraction{})
	if err != nil {
		t.Fatalf("loadSceneState error = %v", err)
	}
	if !interaction.PhaseOpen || interaction.PhaseID != "phase-1" || len(interaction.Slots) != 1 {
		t.Fatalf("interaction = %#v", interaction)
	}
}

func TestInteractionApplicationResolveActingSetRejectsOwnerlessCharacter(t *testing.T) {
	t.Parallel()

	characters := gametest.NewFakeCharacterStore()
	_ = characters.PutCharacter(context.Background(), storage.CharacterRecord{CampaignID: "camp-1", ID: "char-1", Name: "Aria"})

	app := interactionApplication{
		stores: interactionApplicationStores{
			Character: characters,
			SceneCharacter: interactionSceneCharacterStoreStub{
				records: map[string][]storage.SceneCharacterRecord{
					"camp-1:scene-1": {
						{CampaignID: "camp-1", SceneID: "scene-1", CharacterID: "char-1"},
					},
				},
			},
		},
	}

	_, _, err := app.resolveActingSet(
		context.Background(),
		"camp-1",
		storage.SceneRecord{SceneID: "scene-1"},
		storage.SessionInteraction{},
		[]string{"char-1"},
	)
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("code = %v, want failed precondition", status.Code(err))
	}
}

func TestInteractionApplicationRequireActiveScenePhaseRejectsSceneFromDifferentSession(t *testing.T) {
	t.Parallel()

	app := interactionApplication{
		stores: interactionApplicationStores{
			Scene: interactionSceneStoreStub{
				scenes: map[string]storage.SceneRecord{
					"camp-1:scene-1": {CampaignID: "camp-1", SceneID: "scene-1", SessionID: "sess-2", Name: "Bridge"},
				},
			},
			SceneInteraction: interactionSceneInteractionStoreStub{
				interactions: map[string]storage.SceneInteraction{
					"camp-1:scene-1": {
						CampaignID: "camp-1",
						SceneID:    "scene-1",
						SessionID:  "sess-2",
						PhaseOpen:  true,
						PhaseID:    "phase-1",
					},
				},
			},
		},
	}

	_, _, err := app.requireActiveScenePhase(context.Background(), "camp-1", "sess-1", "scene-1", storage.SessionInteraction{ActiveSceneID: "scene-1"})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("code = %v, want failed precondition", status.Code(err))
	}
}

func TestInteractionApplicationLoadActiveSessionInteractionReturnsNilWhenNoActiveSession(t *testing.T) {
	t.Parallel()

	app := interactionApplication{
		stores: interactionApplicationStores{
			Session:            gametest.NewFakeSessionStore(),
			SessionInteraction: interactionSessionInteractionStoreStub{},
		},
	}

	active, interaction, err := app.loadActiveSessionInteraction(context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("loadActiveSessionInteraction error = %v", err)
	}
	if active != nil {
		t.Fatalf("active session = %#v, want nil", active)
	}
	if interaction.SessionID != "" || interaction.ActiveSceneID != "" || interaction.OOCPaused || len(interaction.OOCPosts) != 0 || len(interaction.ReadyToResumeParticipantIDs) != 0 {
		t.Fatalf("interaction = %#v, want zero value", interaction)
	}
}

var _ storage.SceneStore = interactionSceneStoreStub{}
var _ storage.SceneCharacterStore = interactionSceneCharacterStoreStub{}
var _ storage.SessionInteractionStore = interactionSessionInteractionStoreStub{}
var _ storage.SceneInteractionStore = interactionSceneInteractionStoreStub{}
