package game

import (
	"context"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetCommunicationContextForPlayerScopesSceneStreamsToOwnedCharacters(t *testing.T) {
	now := time.Date(2026, 3, 9, 12, 0, 0, 0, time.UTC)
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	characterStore := newFakeCharacterStore()
	sessionStore := newFakeSessionStore()
	sceneStore := newFakeCommunicationSceneStore()
	sceneCharacterStore := newFakeCommunicationSceneCharacterStore()
	sessionGateStore := newFakeSessionGateStore()
	sessionSpotlightStore := newFakeSessionSpotlightStore()

	if err := campaignStore.Put(context.Background(), storage.CampaignRecord{
		ID:           "camp-1",
		Name:         "Ashes of the Vale",
		Locale:       commonv1.Locale_LOCALE_EN_US,
		System:       bridge.SystemIDDaggerheart,
		Status:       campaign.StatusActive,
		AccessPolicy: campaign.AccessPolicyPrivate,
		GmMode:       campaign.GmModeHybrid,
		AIAgentID:    "agent-1",
		CreatedAt:    now,
		UpdatedAt:    now,
	}); err != nil {
		t.Fatalf("put campaign: %v", err)
	}
	if err := participantStore.PutParticipant(context.Background(), storage.ParticipantRecord{
		ID:             "part-player",
		CampaignID:     "camp-1",
		UserID:         "user-player",
		Name:           "Mira",
		Role:           participant.RolePlayer,
		CampaignAccess: participant.CampaignAccessMember,
		CreatedAt:      now,
		UpdatedAt:      now,
	}); err != nil {
		t.Fatalf("put participant: %v", err)
	}
	if err := characterStore.PutCharacter(context.Background(), storage.CharacterRecord{
		ID:                 "char-1",
		CampaignID:         "camp-1",
		OwnerParticipantID: "part-player",
		Name:               "Vera",
		Kind:               character.KindPC,
		CreatedAt:          now,
		UpdatedAt:          now,
	}); err != nil {
		t.Fatalf("put owned character: %v", err)
	}
	if err := characterStore.PutCharacter(context.Background(), storage.CharacterRecord{
		ID:                 "char-2",
		CampaignID:         "camp-1",
		OwnerParticipantID: "part-other",
		Name:               "Nox",
		Kind:               character.KindPC,
		CreatedAt:          now,
		UpdatedAt:          now,
	}); err != nil {
		t.Fatalf("put foreign character: %v", err)
	}
	if err := sessionStore.PutSession(context.Background(), storage.SessionRecord{
		ID:         "sess-1",
		CampaignID: "camp-1",
		Name:       "Session One",
		Status:     session.StatusActive,
		StartedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		t.Fatalf("put session: %v", err)
	}
	if err := sessionGateStore.PutSessionGate(context.Background(), storage.SessionGate{
		CampaignID:         "camp-1",
		SessionID:          "sess-1",
		GateID:             "gate-1",
		GateType:           "choice",
		Status:             session.GateStatusOpen,
		Reason:             "Awaiting table decision",
		CreatedAt:          now,
		CreatedByActorType: "participant",
		CreatedByActorID:   "part-player",
	}); err != nil {
		t.Fatalf("put session gate: %v", err)
	}
	if err := sessionSpotlightStore.PutSessionSpotlight(context.Background(), storage.SessionSpotlight{
		CampaignID:         "camp-1",
		SessionID:          "sess-1",
		SpotlightType:      session.SpotlightTypeCharacter,
		CharacterID:        "char-1",
		UpdatedAt:          now,
		UpdatedByActorType: "participant",
		UpdatedByActorID:   "part-player",
	}); err != nil {
		t.Fatalf("put session spotlight: %v", err)
	}
	if err := sceneStore.PutScene(context.Background(), storage.SceneRecord{
		CampaignID: "camp-1",
		SceneID:    "scene-1",
		SessionID:  "sess-1",
		Name:       "Ruined Hall",
		Active:     true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		t.Fatalf("put scene one: %v", err)
	}
	if err := sceneStore.PutScene(context.Background(), storage.SceneRecord{
		CampaignID: "camp-1",
		SceneID:    "scene-2",
		SessionID:  "sess-1",
		Name:       "Watchtower",
		Active:     true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		t.Fatalf("put scene two: %v", err)
	}
	if err := sceneCharacterStore.PutSceneCharacter(context.Background(), storage.SceneCharacterRecord{
		CampaignID:  "camp-1",
		SceneID:     "scene-1",
		CharacterID: "char-1",
		AddedAt:     now,
	}); err != nil {
		t.Fatalf("put scene character one: %v", err)
	}
	if err := sceneCharacterStore.PutSceneCharacter(context.Background(), storage.SceneCharacterRecord{
		CampaignID:  "camp-1",
		SceneID:     "scene-2",
		CharacterID: "char-2",
		AddedAt:     now,
	}); err != nil {
		t.Fatalf("put scene character two: %v", err)
	}

	service := NewCommunicationService(Stores{
		Campaign:         campaignStore,
		Participant:      participantStore,
		Character:        characterStore,
		Session:          sessionStore,
		SessionGate:      sessionGateStore,
		SessionSpotlight: sessionSpotlightStore,
		Scene:            sceneStore,
		SceneCharacter:   sceneCharacterStore,
	})

	resp, err := service.GetCommunicationContext(contextWithUserID("user-player"), &campaignv1.GetCommunicationContextRequest{
		CampaignId: "camp-1",
	})
	if err != nil {
		t.Fatalf("get communication context: %v", err)
	}

	context := resp.GetContext()
	if context.GetDefaultStreamId() != "campaign:camp-1:table" {
		t.Fatalf("default stream id = %q, want campaign table stream", context.GetDefaultStreamId())
	}
	if context.GetDefaultPersonaId() != "participant:part-player" {
		t.Fatalf("default persona id = %q, want participant persona", context.GetDefaultPersonaId())
	}
	if context.GetActiveSession().GetSessionId() != "sess-1" {
		t.Fatalf("active session id = %q, want %q", context.GetActiveSession().GetSessionId(), "sess-1")
	}
	if context.GetActiveSessionGate().GetId() != "gate-1" {
		t.Fatalf("active session gate id = %q, want %q", context.GetActiveSessionGate().GetId(), "gate-1")
	}
	if context.GetActiveSessionSpotlight().GetCharacterId() != "char-1" {
		t.Fatalf("active session spotlight character = %q, want %q", context.GetActiveSessionSpotlight().GetCharacterId(), "char-1")
	}
	if len(context.GetStreams()) != 4 {
		t.Fatalf("stream count = %d, want 4", len(context.GetStreams()))
	}
	if got := context.GetStreams()[3].GetStreamId(); got != "scene:scene-1:character" {
		t.Fatalf("scene stream id = %q, want visible owned scene stream", got)
	}
	if len(context.GetPersonas()) != 2 {
		t.Fatalf("persona count = %d, want 2", len(context.GetPersonas()))
	}
	if context.GetPersonas()[1].GetCharacterId() != "char-1" {
		t.Fatalf("character persona id = %q, want %q", context.GetPersonas()[1].GetCharacterId(), "char-1")
	}
}

func TestGetCommunicationContextForGMIncludesAllActiveSceneStreams(t *testing.T) {
	now := time.Date(2026, 3, 9, 13, 0, 0, 0, time.UTC)
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	characterStore := newFakeCharacterStore()
	sessionStore := newFakeSessionStore()
	sceneStore := newFakeCommunicationSceneStore()
	sceneCharacterStore := newFakeCommunicationSceneCharacterStore()
	sessionGateStore := newFakeSessionGateStore()
	sessionSpotlightStore := newFakeSessionSpotlightStore()

	_ = campaignStore.Put(context.Background(), storage.CampaignRecord{
		ID:           "camp-1",
		Name:         "Ashes of the Vale",
		Locale:       commonv1.Locale_LOCALE_PT_BR,
		System:       bridge.SystemIDDaggerheart,
		Status:       campaign.StatusActive,
		AccessPolicy: campaign.AccessPolicyPrivate,
		GmMode:       campaign.GmModeHuman,
		CreatedAt:    now,
		UpdatedAt:    now,
	})
	_ = participantStore.PutParticipant(context.Background(), storage.ParticipantRecord{
		ID:             "part-gm",
		CampaignID:     "camp-1",
		UserID:         "user-gm",
		Name:           "GM",
		Role:           participant.RoleGM,
		CampaignAccess: participant.CampaignAccessOwner,
		CreatedAt:      now,
		UpdatedAt:      now,
	})
	_ = sessionStore.PutSession(context.Background(), storage.SessionRecord{
		ID:         "sess-1",
		CampaignID: "camp-1",
		Name:       "Session One",
		Status:     session.StatusActive,
		StartedAt:  now,
		UpdatedAt:  now,
	})
	_ = sceneStore.PutScene(context.Background(), storage.SceneRecord{
		CampaignID: "camp-1",
		SceneID:    "scene-1",
		SessionID:  "sess-1",
		Name:       "Ruined Hall",
		Active:     true,
		CreatedAt:  now,
		UpdatedAt:  now,
	})
	_ = sceneStore.PutScene(context.Background(), storage.SceneRecord{
		CampaignID: "camp-1",
		SceneID:    "scene-2",
		SessionID:  "sess-1",
		Name:       "Watchtower",
		Active:     true,
		CreatedAt:  now,
		UpdatedAt:  now,
	})

	service := NewCommunicationService(Stores{
		Campaign:         campaignStore,
		Participant:      participantStore,
		Character:        characterStore,
		Session:          sessionStore,
		SessionGate:      sessionGateStore,
		SessionSpotlight: sessionSpotlightStore,
		Scene:            sceneStore,
		SceneCharacter:   sceneCharacterStore,
	})

	resp, err := service.GetCommunicationContext(contextWithUserID("user-gm"), &campaignv1.GetCommunicationContextRequest{
		CampaignId: "camp-1",
	})
	if err != nil {
		t.Fatalf("get communication context: %v", err)
	}

	context := resp.GetContext()
	if context.GetLocale() != commonv1.Locale_LOCALE_PT_BR {
		t.Fatalf("locale = %v, want %v", context.GetLocale(), commonv1.Locale_LOCALE_PT_BR)
	}
	if len(context.GetStreams()) != 5 {
		t.Fatalf("stream count = %d, want 5", len(context.GetStreams()))
	}
	if context.GetStreams()[3].GetSceneId() != "scene-1" || context.GetStreams()[4].GetSceneId() != "scene-2" {
		t.Fatalf("expected gm to see both scene streams, got %#v", context.GetStreams())
	}
	if len(context.GetPersonas()) != 1 {
		t.Fatalf("persona count = %d, want 1", len(context.GetPersonas()))
	}
}

func TestGetCommunicationContextRequiresParticipantIdentity(t *testing.T) {
	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	_ = campaignStore.Put(context.Background(), storage.CampaignRecord{
		ID:           "camp-1",
		Name:         "Ashes of the Vale",
		Locale:       commonv1.Locale_LOCALE_EN_US,
		System:       bridge.SystemIDDaggerheart,
		Status:       campaign.StatusActive,
		AccessPolicy: campaign.AccessPolicyPrivate,
		CreatedAt:    time.Now().UTC(),
		UpdatedAt:    time.Now().UTC(),
	})

	service := NewCommunicationService(Stores{
		Campaign:    campaignStore,
		Participant: participantStore,
	})

	_, err := service.GetCommunicationContext(context.Background(), &campaignv1.GetCommunicationContextRequest{
		CampaignId: "camp-1",
	})
	if err == nil {
		t.Fatal("expected permission error")
	}
	if status.Code(err) != codes.PermissionDenied {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.PermissionDenied)
	}
}

type fakeCommunicationSceneStore struct {
	scenes map[string]storage.SceneRecord
}

func newFakeCommunicationSceneStore() *fakeCommunicationSceneStore {
	return &fakeCommunicationSceneStore{scenes: make(map[string]storage.SceneRecord)}
}

func (s *fakeCommunicationSceneStore) PutScene(_ context.Context, record storage.SceneRecord) error {
	s.scenes[record.CampaignID+":"+record.SceneID] = record
	return nil
}

func (s *fakeCommunicationSceneStore) EndScene(_ context.Context, campaignID, sceneID string, endedAt time.Time) error {
	record, ok := s.scenes[campaignID+":"+sceneID]
	if !ok {
		return storage.ErrNotFound
	}
	record.Active = false
	record.EndedAt = &endedAt
	record.UpdatedAt = endedAt
	s.scenes[campaignID+":"+sceneID] = record
	return nil
}

func (s *fakeCommunicationSceneStore) GetScene(_ context.Context, campaignID, sceneID string) (storage.SceneRecord, error) {
	record, ok := s.scenes[campaignID+":"+sceneID]
	if !ok {
		return storage.SceneRecord{}, storage.ErrNotFound
	}
	return record, nil
}

func (s *fakeCommunicationSceneStore) ListScenes(_ context.Context, campaignID, sessionID string, _ int, _ string) (storage.ScenePage, error) {
	scenes := make([]storage.SceneRecord, 0)
	for _, record := range s.scenes {
		if record.CampaignID == campaignID && record.SessionID == sessionID {
			scenes = append(scenes, record)
		}
	}
	return storage.ScenePage{Scenes: scenes}, nil
}

func (s *fakeCommunicationSceneStore) ListActiveScenes(_ context.Context, campaignID string) ([]storage.SceneRecord, error) {
	scenes := make([]storage.SceneRecord, 0)
	for _, record := range s.scenes {
		if record.CampaignID == campaignID && record.Active {
			scenes = append(scenes, record)
		}
	}
	return scenes, nil
}

type fakeCommunicationSceneCharacterStore struct {
	byScene map[string][]storage.SceneCharacterRecord
}

func newFakeCommunicationSceneCharacterStore() *fakeCommunicationSceneCharacterStore {
	return &fakeCommunicationSceneCharacterStore{byScene: make(map[string][]storage.SceneCharacterRecord)}
}

func (s *fakeCommunicationSceneCharacterStore) PutSceneCharacter(_ context.Context, record storage.SceneCharacterRecord) error {
	key := record.CampaignID + ":" + record.SceneID
	s.byScene[key] = append(s.byScene[key], record)
	return nil
}

func (s *fakeCommunicationSceneCharacterStore) DeleteSceneCharacter(_ context.Context, campaignID, sceneID, characterID string) error {
	key := campaignID + ":" + sceneID
	records := s.byScene[key]
	filtered := records[:0]
	found := false
	for _, record := range records {
		if record.CharacterID == characterID {
			found = true
			continue
		}
		filtered = append(filtered, record)
	}
	if !found {
		return storage.ErrNotFound
	}
	s.byScene[key] = filtered
	return nil
}

func (s *fakeCommunicationSceneCharacterStore) ListSceneCharacters(_ context.Context, campaignID, sceneID string) ([]storage.SceneCharacterRecord, error) {
	return append([]storage.SceneCharacterRecord(nil), s.byScene[campaignID+":"+sceneID]...), nil
}
