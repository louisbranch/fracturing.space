package coreprojection

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestSessionInteractionLifecycle(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 27, 12, 0, 0, 0, time.UTC)

	seedCampaign(t, store, "camp-int", now)
	seedSession(t, store, "camp-int", "sess-1", now)
	seedParticipant(t, store, "camp-int", "part-1", "user-1", now)
	seedParticipant(t, store, "camp-int", "part-2", "user-2", now)

	expected := storage.SessionInteraction{
		CampaignID:                  "camp-int",
		SessionID:                   "sess-1",
		CharacterControllers:        []storage.SessionCharacterController{{CharacterID: "char-1", ParticipantID: "part-1"}},
		ActiveSceneID:               "scene-1",
		GMAuthorityParticipantID:    "part-2",
		OOCPaused:                   true,
		OOCRequestedByParticipantID: "part-1",
		OOCReason:                   "break",
		OOCInterruptedSceneID:       "scene-1",
		OOCInterruptedPhaseID:       "phase-1",
		OOCInterruptedPhaseStatus:   "players",
		OOCResolutionPending:        true,
		OOCPosts: []storage.SessionOOCPost{{
			PostID:        "post-1",
			ParticipantID: "part-1",
			Body:          "Need a quick pause",
			CreatedAt:     now,
		}},
		ReadyToResumeParticipantIDs: []string{"part-1", "part-2"},
		AITurn: storage.SessionAITurn{
			Status:             session.AITurnStatusQueued,
			TurnToken:          "turn-1",
			OwnerParticipantID: "part-2",
			SourceEventType:    "session.ai_turn.queue",
			SourceSceneID:      "scene-1",
			SourcePhaseID:      "phase-1",
			LastError:          "retry later",
		},
		UpdatedAt: now.Add(time.Minute),
	}

	if err := store.PutSessionInteraction(ctx, expected); err != nil {
		t.Fatalf("put session interaction: %v", err)
	}

	got, err := store.GetSessionInteraction(ctx, "camp-int", "sess-1")
	if err != nil {
		t.Fatalf("get session interaction: %v", err)
	}
	if got.ActiveSceneID != expected.ActiveSceneID || got.GMAuthorityParticipantID != expected.GMAuthorityParticipantID {
		t.Fatalf("got interaction = %+v", got)
	}
	if !got.OOCPaused || !got.OOCResolutionPending || got.OOCReason != expected.OOCReason {
		t.Fatalf("got interaction = %+v", got)
	}
	if len(got.CharacterControllers) != 1 || got.CharacterControllers[0].ParticipantID != "part-1" {
		t.Fatalf("character controllers = %+v", got.CharacterControllers)
	}
	if len(got.OOCPosts) != 1 || got.OOCPosts[0].Body != "Need a quick pause" || !got.OOCPosts[0].CreatedAt.Equal(now) {
		t.Fatalf("ooc posts = %+v", got.OOCPosts)
	}
	if len(got.ReadyToResumeParticipantIDs) != 2 || got.ReadyToResumeParticipantIDs[1] != "part-2" {
		t.Fatalf("ready list = %+v", got.ReadyToResumeParticipantIDs)
	}
	if got.AITurn.Status != session.AITurnStatusQueued || got.AITurn.TurnToken != "turn-1" || got.AITurn.LastError != "retry later" {
		t.Fatalf("ai turn = %+v", got.AITurn)
	}
	if !got.UpdatedAt.Equal(expected.UpdatedAt) {
		t.Fatalf("updated_at = %v, want %v", got.UpdatedAt, expected.UpdatedAt)
	}

	_, err = store.GetSessionInteraction(ctx, "camp-int", "missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSceneInteractionLifecycle(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 27, 12, 5, 0, 0, time.UTC)

	seedCampaign(t, store, "camp-scene-int", now)
	seedSession(t, store, "camp-scene-int", "sess-1", now)
	if err := store.PutScene(ctx, storage.SceneRecord{
		CampaignID: "camp-scene-int",
		SceneID:    "scene-1",
		SessionID:  "sess-1",
		Name:       "Opening Scene",
		Open:       true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		t.Fatalf("put scene: %v", err)
	}

	expected := storage.SceneInteraction{
		CampaignID:           "camp-scene-int",
		SceneID:              "scene-1",
		SessionID:            "sess-1",
		PhaseOpen:            true,
		PhaseID:              "phase-1",
		PhaseStatus:          scene.PlayerPhaseStatusGMReview,
		ActingCharacterIDs:   []string{"char-1", "char-2"},
		ActingParticipantIDs: []string{"part-1"},
		Slots: []storage.ScenePlayerSlot{{
			ParticipantID:      "part-1",
			SummaryText:        "Aria scouts ahead",
			CharacterIDs:       []string{"char-1"},
			UpdatedAt:          now,
			Yielded:            false,
			ReviewStatus:       scene.PlayerPhaseSlotReviewStatusUnderReview,
			ReviewReason:       "GM reviewing move",
			ReviewCharacterIDs: []string{"char-1"},
		}},
		UpdatedAt: now.Add(2 * time.Minute),
	}

	if err := store.PutSceneInteraction(ctx, expected); err != nil {
		t.Fatalf("put scene interaction: %v", err)
	}

	got, err := store.GetSceneInteraction(ctx, "camp-scene-int", "scene-1")
	if err != nil {
		t.Fatalf("get scene interaction: %v", err)
	}
	if got.SessionID != "sess-1" || !got.PhaseOpen || got.PhaseID != "phase-1" || got.PhaseStatus != scene.PlayerPhaseStatusGMReview {
		t.Fatalf("scene interaction = %+v", got)
	}
	if len(got.ActingCharacterIDs) != 2 || got.ActingCharacterIDs[1] != "char-2" {
		t.Fatalf("acting characters = %+v", got.ActingCharacterIDs)
	}
	if len(got.Slots) != 1 || got.Slots[0].ReviewStatus != scene.PlayerPhaseSlotReviewStatusUnderReview {
		t.Fatalf("slots = %+v", got.Slots)
	}
	if !got.Slots[0].UpdatedAt.Equal(now) || got.Slots[0].ReviewReason != "GM reviewing move" {
		t.Fatalf("slot = %+v", got.Slots[0])
	}
	if !got.UpdatedAt.Equal(expected.UpdatedAt) {
		t.Fatalf("updated_at = %v, want %v", got.UpdatedAt, expected.UpdatedAt)
	}

	_, err = store.GetSceneInteraction(ctx, "camp-scene-int", "missing")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestSceneGMInteractionLifecycle(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 27, 12, 10, 0, 0, time.UTC)

	seedCampaign(t, store, "camp-gm-int", now)
	seedSession(t, store, "camp-gm-int", "sess-1", now)
	if err := store.PutScene(ctx, storage.SceneRecord{
		CampaignID: "camp-gm-int",
		SceneID:    "scene-1",
		SessionID:  "sess-1",
		Name:       "Opening Scene",
		Open:       true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}); err != nil {
		t.Fatalf("put scene: %v", err)
	}

	first := storage.SceneGMInteraction{
		CampaignID:    "camp-gm-int",
		SceneID:       "scene-1",
		SessionID:     "sess-1",
		InteractionID: "int-1",
		PhaseID:       "phase-1",
		ParticipantID: "gm-1",
		Title:         "A cold wind rises",
		CharacterIDs:  []string{"char-1"},
		Illustration: &storage.SceneGMInteractionIllustration{
			ImageURL: "https://example.com/wind.png",
			Alt:      "Wind over the canyon",
			Caption:  "The air changes",
		},
		Beats: []storage.SceneGMInteractionBeat{{
			BeatID: "beat-1",
			Type:   scene.GMInteractionBeatTypeFiction,
			Text:   "The lanterns flicker.",
		}},
		CreatedAt: now,
	}
	second := storage.SceneGMInteraction{
		CampaignID:    "camp-gm-int",
		SceneID:       "scene-1",
		SessionID:     "sess-1",
		InteractionID: "int-2",
		PhaseID:       "phase-1",
		ParticipantID: "gm-1",
		Title:         "What do you do?",
		CharacterIDs:  []string{"char-1", "char-2"},
		Beats: []storage.SceneGMInteractionBeat{{
			BeatID: "beat-2",
			Type:   scene.GMInteractionBeatTypePrompt,
			Text:   "Who moves first?",
		}},
		CreatedAt: now.Add(time.Minute),
	}

	if err := store.PutSceneGMInteraction(ctx, first); err != nil {
		t.Fatalf("put first interaction: %v", err)
	}
	if err := store.PutSceneGMInteraction(ctx, second); err != nil {
		t.Fatalf("put second interaction: %v", err)
	}

	got, err := store.ListSceneGMInteractions(ctx, "camp-gm-int", "scene-1")
	if err != nil {
		t.Fatalf("list scene gm interactions: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 interactions, got %d", len(got))
	}
	if got[0].InteractionID != "int-2" || got[1].InteractionID != "int-1" {
		t.Fatalf("interaction order = %+v", got)
	}
	if got[1].Illustration == nil || got[1].Illustration.Alt != "Wind over the canyon" {
		t.Fatalf("illustration = %+v", got[1].Illustration)
	}
	if len(got[0].Beats) != 1 || got[0].Beats[0].Type != scene.GMInteractionBeatTypePrompt {
		t.Fatalf("beats = %+v", got[0].Beats)
	}
}

func TestInteractionProjectionHelpers(t *testing.T) {
	if got := normalizeAITurnStatus(" queued "); got != session.AITurnStatusQueued {
		t.Fatalf("normalizeAITurnStatus(valid) = %q", got)
	}
	if got := normalizeAITurnStatus("not-a-status"); got != session.AITurnStatusIdle {
		t.Fatalf("normalizeAITurnStatus(invalid) = %q", got)
	}
	if got := normalizeScenePhaseStatus("gm_review"); got != scene.PlayerPhaseStatusGMReview {
		t.Fatalf("normalizeScenePhaseStatus(valid) = %q", got)
	}
	if got := normalizeScenePhaseStatus("unknown"); got != "" {
		t.Fatalf("normalizeScenePhaseStatus(invalid) = %q", got)
	}
}

func TestProjectionLookupHelpers(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 27, 12, 20, 0, 0, time.UTC)

	camp1 := seedCampaign(t, store, "camp-lookup-1", now)
	camp1.AIAgentID = "agent-1"
	if err := store.Put(ctx, camp1); err != nil {
		t.Fatalf("put campaign 1 ai binding: %v", err)
	}
	camp2 := seedCampaign(t, store, "camp-lookup-2", now)
	camp2.AIAgentID = "agent-1"
	if err := store.Put(ctx, camp2); err != nil {
		t.Fatalf("put campaign 2 ai binding: %v", err)
	}

	seedParticipant(t, store, "camp-lookup-1", "shared-participant", "user-1", now)
	seedParticipant(t, store, "camp-lookup-2", "shared-participant", "user-1", now)
	seedCharacter(t, store, "camp-lookup-1", "char-1", "Aria", character.KindPC, now)
	seedCharacter(t, store, "camp-lookup-1", "char-2", "Brim", character.KindNPC, now)

	count, err := store.CountCharacters(ctx, "camp-lookup-1")
	if err != nil {
		t.Fatalf("count characters: %v", err)
	}
	if count != 2 {
		t.Fatalf("character count = %d, want 2", count)
	}

	campaignIDs, err := store.ListCampaignIDsByUser(ctx, "user-1")
	if err != nil {
		t.Fatalf("list campaign ids by user: %v", err)
	}
	sort.Strings(campaignIDs)
	if len(campaignIDs) != 2 || campaignIDs[0] != "camp-lookup-1" || campaignIDs[1] != "camp-lookup-2" {
		t.Fatalf("campaign ids by user = %+v", campaignIDs)
	}

	campaignIDs, err = store.ListCampaignIDsByParticipant(ctx, "shared-participant")
	if err != nil {
		t.Fatalf("list campaign ids by participant: %v", err)
	}
	sort.Strings(campaignIDs)
	if len(campaignIDs) != 2 || campaignIDs[0] != "camp-lookup-1" || campaignIDs[1] != "camp-lookup-2" {
		t.Fatalf("campaign ids by participant = %+v", campaignIDs)
	}

	campaignIDs, err = store.ListCampaignIDsByAIAgent(ctx, "agent-1")
	if err != nil {
		t.Fatalf("list campaign ids by ai agent: %v", err)
	}
	sort.Strings(campaignIDs)
	if len(campaignIDs) != 2 || campaignIDs[0] != "camp-lookup-1" || campaignIDs[1] != "camp-lookup-2" {
		t.Fatalf("campaign ids by ai agent = %+v", campaignIDs)
	}
}

func TestPutAndGetSessionInteraction_RoundTrip(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 10, 14, 0, 0, 0, time.UTC)

	seedCampaign(t, store, "camp-interact-rt", now)
	seedSession(t, store, "camp-interact-rt", "sess-1", now)

	postTime := now.Add(5 * time.Minute)
	interaction := storage.SessionInteraction{
		CampaignID: "camp-interact-rt",
		SessionID:  "sess-1",
		CharacterControllers: []storage.SessionCharacterController{
			{CharacterID: "char-1", ParticipantID: "part-1"},
			{CharacterID: "char-2", ParticipantID: "part-2"},
		},
		ActiveSceneID:               "scene-1",
		GMAuthorityParticipantID:    "part-gm",
		OOCPaused:                   true,
		OOCRequestedByParticipantID: "part-1",
		OOCReason:                   "Need a break",
		OOCInterruptedSceneID:       "scene-1",
		OOCInterruptedPhaseID:       "phase-1",
		OOCInterruptedPhaseStatus:   "players",
		OOCResolutionPending:        true,
		OOCPosts: []storage.SessionOOCPost{
			{PostID: "post-1", ParticipantID: "part-1", Body: "Taking five", CreatedAt: postTime},
		},
		ReadyToResumeParticipantIDs: []string{"part-2"},
		AITurn: storage.SessionAITurn{
			Status:             session.AITurnStatusRunning,
			TurnToken:          "tok-abc",
			OwnerParticipantID: "part-gm",
			SourceEventType:    "scene.player_phase_submitted",
			SourceSceneID:      "scene-1",
			SourcePhaseID:      "phase-1",
		},
		UpdatedAt: now,
	}

	if err := store.PutSessionInteraction(ctx, interaction); err != nil {
		t.Fatalf("put session interaction: %v", err)
	}

	got, err := store.GetSessionInteraction(ctx, interaction.CampaignID, interaction.SessionID)
	if err != nil {
		t.Fatalf("get session interaction: %v", err)
	}

	if got.CampaignID != interaction.CampaignID {
		t.Fatalf("campaign_id = %q, want %q", got.CampaignID, interaction.CampaignID)
	}
	if !got.OOCPaused {
		t.Fatal("expected ooc_paused = true")
	}
	if len(got.OOCPosts) != 1 || got.OOCPosts[0].PostID != "post-1" {
		t.Fatalf("ooc_posts = %+v, want 1 post with id post-1", got.OOCPosts)
	}
	if len(got.CharacterControllers) != 2 {
		t.Fatalf("character_controllers length = %d, want 2", len(got.CharacterControllers))
	}
	if got.AITurn.Status != session.AITurnStatusRunning {
		t.Fatalf("ai_turn_status = %q, want %q", got.AITurn.Status, session.AITurnStatusRunning)
	}
}

func TestGetSessionInteraction_NotFound(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	_, err := store.GetSessionInteraction(ctx, "no-camp", "no-sess")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPutAndGetSceneInteraction_RoundTrip(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()
	now := time.Date(2026, 3, 10, 17, 0, 0, 0, time.UTC)

	seedCampaign(t, store, "camp-sc-interact", now)
	if err := store.PutScene(ctx, storage.SceneRecord{
		CampaignID: "camp-sc-interact", SceneID: "scene-1", SessionID: "sess-1",
		Name: "Test Scene", Open: true, CreatedAt: now, UpdatedAt: now,
	}); err != nil {
		t.Fatalf("seed scene: %v", err)
	}

	interaction := storage.SceneInteraction{
		CampaignID:           "camp-sc-interact",
		SceneID:              "scene-1",
		SessionID:            "sess-1",
		PhaseOpen:            true,
		PhaseID:              "phase-1",
		PhaseStatus:          scene.PlayerPhaseStatusPlayers,
		ActingCharacterIDs:   []string{"char-1", "char-2"},
		ActingParticipantIDs: []string{"part-1", "part-2"},
		Slots: []storage.ScenePlayerSlot{
			{ParticipantID: "part-1", SummaryText: "I attack", CharacterIDs: []string{"char-1"},
				UpdatedAt: now, ReviewStatus: scene.PlayerPhaseSlotReviewStatusOpen},
		},
		UpdatedAt: now,
	}

	if err := store.PutSceneInteraction(ctx, interaction); err != nil {
		t.Fatalf("put scene interaction: %v", err)
	}

	got, err := store.GetSceneInteraction(ctx, interaction.CampaignID, interaction.SceneID)
	if err != nil {
		t.Fatalf("get scene interaction: %v", err)
	}

	if !got.PhaseOpen {
		t.Fatal("expected phase_open = true")
	}
	if len(got.ActingCharacterIDs) != 2 {
		t.Fatalf("acting_character_ids length = %d, want 2", len(got.ActingCharacterIDs))
	}
	if len(got.Slots) != 1 || got.Slots[0].ParticipantID != "part-1" {
		t.Fatalf("slots = %+v, want 1 slot for part-1", got.Slots)
	}
}

func TestGetSceneInteraction_NotFound(t *testing.T) {
	store := openTestStore(t)
	ctx := context.Background()

	_, err := store.GetSceneInteraction(ctx, "no-camp", "no-scene")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}
