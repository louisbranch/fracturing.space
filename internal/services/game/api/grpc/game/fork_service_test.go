package game

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestForkCampaign_ReplaysEvents_CopyParticipantsFalse(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)

	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	characterStore := newFakeCharacterStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()
	forkStore := newFakeCampaignForkStore()

	campaignStore.campaigns["source"] = storage.CampaignRecord{
		ID:          "source",
		Name:        "Source Campaign",
		Status:      campaign.StatusActive,
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      campaign.GmModeHuman,
		ThemePrompt: "theme",
	}

	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-10 * time.Hour),
		Type:       event.Type("campaign.created"),
		EntityType: "campaign",
		EntityID:   "source",
		PayloadJSON: mustJSON(t, campaign.CreatePayload{
			Name:        "Source Campaign",
			GameSystem:  commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
			GmMode:      statev1.GmMode_HUMAN.String(),
			ThemePrompt: "theme",
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-9 * time.Hour),
		Type:       event.Type("participant.joined"),
		EntityType: "participant",
		EntityID:   "part-1",
		PayloadJSON: mustJSON(t, participant.JoinPayload{
			ParticipantID:  "part-1",
			Name:           "Alice",
			Role:           "PLAYER",
			Controller:     "CONTROLLER_HUMAN",
			CampaignAccess: "MEMBER",
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-8 * time.Hour),
		Type:       event.Type("character.created"),
		EntityType: "character",
		EntityID:   "char-1",
		PayloadJSON: mustJSON(t, character.CreatePayload{
			CharacterID: "char-1",
			Name:        "Hero",
			Kind:        "PC",
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-7 * time.Hour),
		Type:       event.Type("character.profile_updated"),
		EntityType: "character",
		EntityID:   "char-1",
		PayloadJSON: mustJSON(t, character.ProfileUpdatePayload{
			CharacterID: "char-1",
			SystemProfile: map[string]any{
				"daggerheart": map[string]any{
					"hp_max":           6,
					"stress_max":       6,
					"evasion":          11,
					"agility":          1,
					"strength":         1,
					"finesse":          1,
					"instinct":         1,
					"presence":         1,
					"knowledge":        1,
					"major_threshold":  3,
					"severe_threshold": 5,
				},
			},
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-6 * time.Hour),
		Type:       event.Type("character.updated"),
		EntityType: "character",
		EntityID:   "char-1",
		PayloadJSON: mustJSON(t, character.UpdatePayload{
			CharacterID: "char-1",
			Fields: map[string]string{
				"participant_id": "part-1",
			},
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID:    "source",
		Timestamp:     now.Add(-5 * time.Hour),
		Type:          event.Type("sys.daggerheart.action.character_state_patched"),
		EntityType:    "character",
		EntityID:      "char-1",
		SystemID:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON: mustJSON(t, daggerheart.CharacterStatePatchedPayload{
			CharacterID: "char-1",
			HPAfter:     intPtr(6),
			HopeAfter:   intPtr(2),
			StressAfter: intPtr(1),
		}),
	})

	createdPayload := campaign.CreatePayload{
		Name:        "Forked Campaign",
		GameSystem:  commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		GmMode:      statev1.GmMode_HUMAN.String(),
		ThemePrompt: "theme",
	}
	createdJSON, err := json.Marshal(createdPayload)
	if err != nil {
		t.Fatalf("encode created payload: %v", err)
	}
	forkedPayload := campaign.ForkPayload{
		ParentCampaignID: "source",
		ForkEventSeq:     6,
		OriginCampaignID: "source",
		CopyParticipants: false,
	}
	forkedJSON, err := json.Marshal(forkedPayload)
	if err != nil {
		t.Fatalf("encode fork payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("campaign.create"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "fork-1",
				Type:        event.Type("campaign.created"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "campaign",
				EntityID:    "fork-1",
				PayloadJSON: createdJSON,
			}),
		},
		command.Type("campaign.fork"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "fork-1",
				Type:        event.Type("campaign.forked"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "campaign",
				EntityID:    "fork-1",
				PayloadJSON: forkedJSON,
			}),
		},
	}}

	svc := &ForkService{
		stores: Stores{
			Campaign:     campaignStore,
			Participant:  participantStore,
			Character:    characterStore,
			Daggerheart:  dhStore,
			Event:        eventStore,
			CampaignFork: forkStore,
			Domain:       domain,
		},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("fork-1"),
	}

	resp, err := svc.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Campaign",
		CopyParticipants: false,
	})
	if err != nil {
		t.Fatalf("ForkCampaign returned error: %v", err)
	}
	if resp.Campaign.GetId() != "fork-1" {
		t.Fatalf("Campaign ID = %q, want %q", resp.Campaign.GetId(), "fork-1")
	}
	if resp.Campaign.GetName() != "Forked Campaign" {
		t.Fatalf("Campaign Name = %q, want %q", resp.Campaign.GetName(), "Forked Campaign")
	}

	if _, err := participantStore.ListParticipantsByCampaign(ctx, "fork-1"); err != nil {
		t.Fatalf("ListParticipantsByCampaign returned error: %v", err)
	}

	if _, err := characterStore.GetCharacter(ctx, "fork-1", "char-1"); err != nil {
		t.Fatalf("expected character in forked campaign: %v", err)
	}

	if _, err := dhStore.GetDaggerheartCharacterProfile(ctx, "fork-1", "char-1"); err != nil {
		t.Fatalf("expected daggerheart profile in forked campaign: %v", err)
	}
	if _, err := dhStore.GetDaggerheartCharacterState(ctx, "fork-1", "char-1"); err != nil {
		t.Fatalf("expected daggerheart state in forked campaign: %v", err)
	}

	forkedCharacter, err := characterStore.GetCharacter(ctx, "fork-1", "char-1")
	if err != nil {
		t.Fatalf("expected character in forked campaign: %v", err)
	}
	if forkedCharacter.ParticipantID != "" {
		t.Fatalf("ParticipantID = %q, want empty", forkedCharacter.ParticipantID)
	}

	forkedEvents := eventStore.events["fork-1"]
	if len(forkedEvents) != 5 {
		t.Fatalf("expected 5 forked events, got %d", len(forkedEvents))
	}
	if forkedEvents[0].Type != event.Type("campaign.created") {
		t.Fatalf("event[0] type = %s, want %s", forkedEvents[0].Type, event.Type("campaign.created"))
	}
	if forkedEvents[1].Type != event.Type("campaign.forked") {
		t.Fatalf("event[1] type = %s, want %s", forkedEvents[1].Type, event.Type("campaign.forked"))
	}
	if forkedEvents[2].Type != event.Type("character.created") {
		t.Fatalf("event[2] type = %s, want %s", forkedEvents[2].Type, event.Type("character.created"))
	}
	if forkedEvents[3].Type != event.Type("character.profile_updated") {
		t.Fatalf("event[3] type = %s, want %s", forkedEvents[3].Type, event.Type("character.profile_updated"))
	}
	if forkedEvents[4].Type != event.Type("sys.daggerheart.action.character_state_patched") {
		t.Fatalf("event[4] type = %s, want %s", forkedEvents[4].Type, event.Type("sys.daggerheart.action.character_state_patched"))
	}

	metadata, err := forkStore.GetCampaignForkMetadata(ctx, "fork-1")
	if err != nil {
		t.Fatalf("fork metadata not stored: %v", err)
	}
	if metadata.ParentCampaignID != "source" {
		t.Fatalf("ParentCampaignID = %q, want %q", metadata.ParentCampaignID, "source")
	}
	if metadata.ForkEventSeq != 6 {
		t.Fatalf("ForkEventSeq = %d, want 6", metadata.ForkEventSeq)
	}
}

func TestForkCampaign_RequiresDomainEngine(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)

	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	characterStore := newFakeCharacterStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()
	forkStore := newFakeCampaignForkStore()

	campaignStore.campaigns["source"] = storage.CampaignRecord{
		ID:          "source",
		Name:        "Source Campaign",
		Status:      campaign.StatusActive,
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      campaign.GmModeHuman,
		ThemePrompt: "theme",
	}

	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-10 * time.Hour),
		Type:       event.Type("campaign.created"),
		EntityType: "campaign",
		EntityID:   "source",
		PayloadJSON: mustJSON(t, campaign.CreatePayload{
			Name:        "Source Campaign",
			GameSystem:  commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
			GmMode:      statev1.GmMode_HUMAN.String(),
			ThemePrompt: "theme",
		}),
	})

	svc := &ForkService{
		stores: Stores{
			Campaign:     campaignStore,
			Participant:  participantStore,
			Character:    characterStore,
			Daggerheart:  dhStore,
			Event:        eventStore,
			CampaignFork: forkStore,
		},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("fork-1"),
	}

	_, err := svc.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Campaign",
		CopyParticipants: false,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestForkCampaign_SeedsSnapshotStateAtHead(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)

	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	characterStore := newFakeCharacterStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()
	forkStore := newFakeCampaignForkStore()

	campaignStore.campaigns["source"] = storage.CampaignRecord{
		ID:          "source",
		Name:        "Source Campaign",
		Status:      campaign.StatusActive,
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      campaign.GmModeHuman,
		ThemePrompt: "theme",
	}
	characterStore.characters["source"] = map[string]storage.CharacterRecord{
		"char-1": {
			ID:         "char-1",
			CampaignID: "source",
			Name:       "Hero",
			Kind:       character.KindPC,
			CreatedAt:  now.Add(-8 * time.Hour),
			UpdatedAt:  now.Add(-8 * time.Hour),
		},
	}
	dhStore.states["source"] = map[string]storage.DaggerheartCharacterState{
		"char-1": {
			CampaignID:  "source",
			CharacterID: "char-1",
			Hp:          6,
			Hope:        2,
			Stress:      1,
		},
	}
	dhStore.snapshots["source"] = storage.DaggerheartSnapshot{
		CampaignID: "source",
		GMFear:     4,
	}

	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-10 * time.Hour),
		Type:       event.Type("campaign.created"),
		EntityType: "campaign",
		EntityID:   "source",
		PayloadJSON: mustJSON(t, campaign.CreatePayload{
			Name:        "Source Campaign",
			GameSystem:  commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
			GmMode:      statev1.GmMode_HUMAN.String(),
			ThemePrompt: "theme",
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-9 * time.Hour),
		Type:       event.Type("participant.joined"),
		EntityType: "participant",
		EntityID:   "part-1",
		PayloadJSON: mustJSON(t, participant.JoinPayload{
			ParticipantID:  "part-1",
			Name:           "Alice",
			Role:           "PLAYER",
			Controller:     "CONTROLLER_HUMAN",
			CampaignAccess: "MEMBER",
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-8 * time.Hour),
		Type:       event.Type("character.created"),
		EntityType: "character",
		EntityID:   "char-1",
		PayloadJSON: mustJSON(t, character.CreatePayload{
			CharacterID: "char-1",
			Name:        "Hero",
			Kind:        "PC",
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-7 * time.Hour),
		Type:       event.Type("character.profile_updated"),
		EntityType: "character",
		EntityID:   "char-1",
		PayloadJSON: mustJSON(t, character.ProfileUpdatePayload{
			CharacterID: "char-1",
			SystemProfile: map[string]any{
				"daggerheart": map[string]any{
					"hp_max":           6,
					"stress_max":       6,
					"evasion":          11,
					"agility":          1,
					"strength":         1,
					"finesse":          1,
					"instinct":         1,
					"presence":         1,
					"knowledge":        1,
					"major_threshold":  3,
					"severe_threshold": 5,
				},
			},
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-6 * time.Hour),
		Type:       event.Type("character.updated"),
		EntityType: "character",
		EntityID:   "char-1",
		PayloadJSON: mustJSON(t, character.UpdatePayload{
			CharacterID: "char-1",
			Fields: map[string]string{
				"participant_id": "part-1",
			},
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID:    "source",
		Timestamp:     now.Add(-5 * time.Hour),
		Type:          event.Type("sys.daggerheart.action.character_state_patched"),
		EntityType:    "character",
		EntityID:      "char-1",
		SystemID:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON: mustJSON(t, daggerheart.CharacterStatePatchedPayload{
			CharacterID: "char-1",
			HPAfter:     intPtr(6),
			HopeAfter:   intPtr(2),
			StressAfter: intPtr(1),
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID:    "source",
		Timestamp:     now.Add(-4 * time.Hour),
		Type:          event.Type("sys.daggerheart.action.gm_fear_changed"),
		EntityType:    "campaign",
		EntityID:      "source",
		SystemID:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON: mustJSON(t, daggerheart.GMFearChangedPayload{
			Before: 2,
			After:  4,
		}),
	})

	createdPayload := campaign.CreatePayload{
		Name:        "Forked Campaign",
		GameSystem:  commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		GmMode:      statev1.GmMode_HUMAN.String(),
		ThemePrompt: "theme",
	}
	createdJSON, err := json.Marshal(createdPayload)
	if err != nil {
		t.Fatalf("encode created payload: %v", err)
	}
	forkedPayload := campaign.ForkPayload{
		ParentCampaignID: "source",
		ForkEventSeq:     7,
		OriginCampaignID: "source",
		CopyParticipants: false,
	}
	forkedJSON, err := json.Marshal(forkedPayload)
	if err != nil {
		t.Fatalf("encode fork payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("campaign.create"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "fork-1",
				Type:        event.Type("campaign.created"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "campaign",
				EntityID:    "fork-1",
				PayloadJSON: createdJSON,
			}),
		},
		command.Type("campaign.fork"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "fork-1",
				Type:        event.Type("campaign.forked"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "campaign",
				EntityID:    "fork-1",
				PayloadJSON: forkedJSON,
			}),
		},
	}}

	svc := &ForkService{
		stores: Stores{
			Campaign:     campaignStore,
			Participant:  participantStore,
			Character:    characterStore,
			Daggerheart:  dhStore,
			Event:        eventStore,
			CampaignFork: forkStore,
			Domain:       domain,
		},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("fork-1"),
	}

	_, err = svc.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Campaign",
		CopyParticipants: false,
	})
	if err != nil {
		t.Fatalf("ForkCampaign returned error: %v", err)
	}

	state, err := dhStore.GetDaggerheartCharacterState(ctx, "fork-1", "char-1")
	if err != nil {
		t.Fatalf("expected daggerheart state in forked campaign: %v", err)
	}
	if state.Hp != 6 || state.Hope != 2 || state.Stress != 1 {
		t.Fatalf("forked state = %+v, want hp=6 hope=2 stress=1", state)
	}

	snapshot, err := dhStore.GetDaggerheartSnapshot(ctx, "fork-1")
	if err != nil {
		t.Fatalf("expected daggerheart snapshot in forked campaign: %v", err)
	}
	if snapshot.GMFear != 4 {
		t.Fatalf("forked gm fear = %d, want 4", snapshot.GMFear)
	}

	if dhStore.statePuts["fork-1"] != 1 {
		t.Fatalf("daggerheart state puts = %d, want 1", dhStore.statePuts["fork-1"])
	}
	if dhStore.snapPuts["fork-1"] != 1 {
		t.Fatalf("daggerheart snapshot puts = %d, want 1", dhStore.snapPuts["fork-1"])
	}
}

func TestForkCampaign_UsesDomainEngine(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)

	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	characterStore := newFakeCharacterStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()
	forkStore := newFakeCampaignForkStore()

	campaignStore.campaigns["source"] = storage.CampaignRecord{
		ID:          "source",
		Name:        "Source Campaign",
		Status:      campaign.StatusActive,
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      campaign.GmModeHuman,
		ThemePrompt: "theme",
	}

	createdPayload := campaign.CreatePayload{
		Name:        "Forked Campaign",
		GameSystem:  commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		GmMode:      statev1.GmMode_HUMAN.String(),
		ThemePrompt: "theme",
	}
	createdJSON, err := json.Marshal(createdPayload)
	if err != nil {
		t.Fatalf("encode created payload: %v", err)
	}
	forkedPayload := campaign.ForkPayload{
		ParentCampaignID: "source",
		ForkEventSeq:     0,
		OriginCampaignID: "source",
		CopyParticipants: false,
	}
	forkedJSON, err := json.Marshal(forkedPayload)
	if err != nil {
		t.Fatalf("encode fork payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("campaign.create"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "fork-1",
				Type:        event.Type("campaign.created"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "campaign",
				EntityID:    "fork-1",
				PayloadJSON: createdJSON,
			}),
		},
		command.Type("campaign.fork"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "fork-1",
				Type:        event.Type("campaign.forked"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "campaign",
				EntityID:    "fork-1",
				PayloadJSON: forkedJSON,
			}),
		},
	}}

	svc := &ForkService{
		stores: Stores{
			Campaign:     campaignStore,
			Participant:  participantStore,
			Character:    characterStore,
			Daggerheart:  dhStore,
			Event:        eventStore,
			CampaignFork: forkStore,
			Domain:       domain,
		},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("fork-1"),
	}

	_, err = svc.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Campaign",
		CopyParticipants: false,
	})
	if err != nil {
		t.Fatalf("ForkCampaign returned error: %v", err)
	}
	if domain.calls != 2 {
		t.Fatalf("expected domain to be called twice, got %d", domain.calls)
	}
	if len(domain.commands) != 2 {
		t.Fatalf("expected 2 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != command.Type("campaign.create") {
		t.Fatalf("command[0] type = %s, want %s", domain.commands[0].Type, "campaign.create")
	}
	if domain.commands[1].Type != command.Type("campaign.fork") {
		t.Fatalf("command[1] type = %s, want %s", domain.commands[1].Type, "campaign.fork")
	}
	if got := len(eventStore.events["fork-1"]); got != 2 {
		t.Fatalf("expected 2 events, got %d", got)
	}
	if eventStore.events["fork-1"][0].Type != event.Type("campaign.created") {
		t.Fatalf("event[0] type = %s, want %s", eventStore.events["fork-1"][0].Type, event.Type("campaign.created"))
	}
	if eventStore.events["fork-1"][1].Type != event.Type("campaign.forked") {
		t.Fatalf("event[1] type = %s, want %s", eventStore.events["fork-1"][1].Type, event.Type("campaign.forked"))
	}
}

func TestForkCampaign_SessionBoundaryForkPoint(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2025, 2, 2, 9, 0, 0, 0, time.UTC)

	campaignStore := newFakeCampaignStore()
	participantStore := newFakeParticipantStore()
	characterStore := newFakeCharacterStore()
	dhStore := newFakeDaggerheartStore()
	eventStore := newFakeEventStore()
	forkStore := newFakeCampaignForkStore()
	sessionStore := newFakeSessionStore()

	campaignStore.campaigns["source"] = storage.CampaignRecord{
		ID:          "source",
		Name:        "Source Campaign",
		Status:      campaign.StatusActive,
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      campaign.GmModeHuman,
		ThemePrompt: "theme",
	}
	endedAt := now.Add(-30 * time.Minute)
	sessionStore.sessions["source"] = map[string]storage.SessionRecord{
		"sess-1": {
			ID:         "sess-1",
			CampaignID: "source",
			Name:       "Session 1",
			Status:     session.StatusEnded,
			StartedAt:  now.Add(-2 * time.Hour),
			UpdatedAt:  endedAt,
			EndedAt:    &endedAt,
		},
	}

	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-3 * time.Hour),
		Type:       event.Type("campaign.created"),
		EntityType: "campaign",
		EntityID:   "source",
		PayloadJSON: mustJSON(t, campaign.CreatePayload{
			Name:        "Source Campaign",
			GameSystem:  commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
			GmMode:      statev1.GmMode_HUMAN.String(),
			ThemePrompt: "theme",
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-2 * time.Hour),
		Type:       event.Type("character.created"),
		EntityType: "character",
		EntityID:   "char-1",
		SessionID:  "sess-1",
		PayloadJSON: mustJSON(t, character.CreatePayload{
			CharacterID: "char-1",
			Name:        "Hero",
			Kind:        "PC",
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID:    "source",
		Timestamp:     now.Add(-90 * time.Minute),
		Type:          event.Type("sys.daggerheart.action.gm_fear_changed"),
		EntityType:    "campaign",
		EntityID:      "source",
		SessionID:     "sess-1",
		SystemID:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON: mustJSON(t, daggerheart.GMFearChangedPayload{
			Before: 1,
			After:  2,
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-45 * time.Minute),
		Type:       event.Type("character.updated"),
		EntityType: "character",
		EntityID:   "char-1",
		SessionID:  "sess-2",
		PayloadJSON: mustJSON(t, character.UpdatePayload{
			CharacterID: "char-1",
			Fields: map[string]string{
				"notes": "after session",
			},
		}),
	})

	createdPayload := campaign.CreatePayload{
		Name:        "Forked Campaign",
		GameSystem:  commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		GmMode:      statev1.GmMode_HUMAN.String(),
		ThemePrompt: "theme",
	}
	createdJSON, err := json.Marshal(createdPayload)
	if err != nil {
		t.Fatalf("encode created payload: %v", err)
	}
	forkedPayload := campaign.ForkPayload{
		ParentCampaignID: "source",
		ForkEventSeq:     3,
		OriginCampaignID: "source",
		CopyParticipants: false,
	}
	forkedJSON, err := json.Marshal(forkedPayload)
	if err != nil {
		t.Fatalf("encode fork payload: %v", err)
	}

	domain := &fakeDomainEngine{store: eventStore, resultsByType: map[command.Type]engine.Result{
		command.Type("campaign.create"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "fork-1",
				Type:        event.Type("campaign.created"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "campaign",
				EntityID:    "fork-1",
				PayloadJSON: createdJSON,
			}),
		},
		command.Type("campaign.fork"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "fork-1",
				Type:        event.Type("campaign.forked"),
				Timestamp:   now,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "campaign",
				EntityID:    "fork-1",
				PayloadJSON: forkedJSON,
			}),
		},
	}}

	svc := &ForkService{
		stores: Stores{
			Campaign:     campaignStore,
			Participant:  participantStore,
			Character:    characterStore,
			Daggerheart:  dhStore,
			Event:        eventStore,
			CampaignFork: forkStore,
			Session:      sessionStore,
			Domain:       domain,
		},
		clock:       fixedClock(now),
		idGenerator: fixedIDGenerator("fork-1"),
	}

	resp, err := svc.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Campaign",
		CopyParticipants: false,
		ForkPoint: &statev1.ForkPoint{
			SessionId: "sess-1",
		},
	})
	if err != nil {
		t.Fatalf("ForkCampaign returned error: %v", err)
	}
	if resp.ForkEventSeq != 3 {
		t.Fatalf("ForkEventSeq = %d, want 3", resp.ForkEventSeq)
	}

	forkedEvents := eventStore.events["fork-1"]
	if len(forkedEvents) != 4 {
		t.Fatalf("expected 4 forked events, got %d", len(forkedEvents))
	}
}

func TestShouldCopyForkEvent(t *testing.T) {
	tests := []struct {
		name             string
		eventType        event.Type
		copyParticipants bool
		payload          []byte
		wantCopy         bool
		wantErr          bool
	}{
		{
			name:      "campaign_created_always_skipped",
			eventType: event.Type("campaign.created"),
			wantCopy:  false,
		},
		{
			name:      "campaign_forked_always_skipped",
			eventType: event.Type("campaign.forked"),
			wantCopy:  false,
		},
		{
			name:             "participant_joined_skip_when_no_copy",
			eventType:        event.Type("participant.joined"),
			copyParticipants: false,
			wantCopy:         false,
		},
		{
			name:             "participant_joined_copy_when_enabled",
			eventType:        event.Type("participant.joined"),
			copyParticipants: true,
			wantCopy:         true,
		},
		{
			name:             "participant_updated_skip_when_no_copy",
			eventType:        event.Type("participant.updated"),
			copyParticipants: false,
			wantCopy:         false,
		},
		{
			name:             "participant_left_skip_when_no_copy",
			eventType:        event.Type("participant.left"),
			copyParticipants: false,
			wantCopy:         false,
		},
		{
			name:             "character_updated_copy_when_participants_enabled",
			eventType:        event.Type("character.updated"),
			copyParticipants: true,
			payload:          []byte(`{"fields":{"participant_id":"p1"}}`),
			wantCopy:         true,
		},
		{
			name:             "character_updated_no_participant_field",
			eventType:        event.Type("character.updated"),
			copyParticipants: false,
			payload:          []byte(`{"fields":{"name":"Hero"}}`),
			wantCopy:         true,
		},
		{
			name:             "character_updated_only_participant_id_field",
			eventType:        event.Type("character.updated"),
			copyParticipants: false,
			payload:          []byte(`{"fields":{"participant_id":"p1"}}`),
			wantCopy:         false,
		},
		{
			name:             "character_updated_participant_id_plus_others",
			eventType:        event.Type("character.updated"),
			copyParticipants: false,
			payload:          []byte(`{"fields":{"participant_id":"p1","name":"Hero"}}`),
			wantCopy:         true,
		},
		{
			name:             "character_updated_empty_participant_id",
			eventType:        event.Type("character.updated"),
			copyParticipants: false,
			payload:          []byte(`{"fields":{"participant_id":""}}`),
			wantCopy:         true,
		},
		{
			name:      "session_started_always_copied",
			eventType: event.Type("session.started"),
			wantCopy:  true,
		},
		{
			name:      "unknown_event_always_copied",
			eventType: event.Type("custom.event"),
			wantCopy:  true,
		},
		{
			name:             "character_updated_invalid_json",
			eventType:        event.Type("character.updated"),
			copyParticipants: false,
			payload:          []byte(`not json`),
			wantErr:          true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			evt := event.Event{Type: tc.eventType, PayloadJSON: tc.payload}
			got, err := shouldCopyForkEvent(evt, tc.copyParticipants)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.wantCopy {
				t.Errorf("shouldCopyForkEvent = %v, want %v", got, tc.wantCopy)
			}
		})
	}
}

func TestForkEventForCampaign(t *testing.T) {
	evt := event.Event{
		CampaignID: "old-camp",
		Seq:        42,
		Hash:       "abc",
		EntityType: "campaign",
		EntityID:   "old-camp",
		Type:       event.Type("campaign.updated"),
	}
	forked := forkEventForCampaign(evt, "new-camp")

	if forked.CampaignID != "new-camp" {
		t.Fatalf("CampaignID = %q, want %q", forked.CampaignID, "new-camp")
	}
	if forked.Seq != 0 {
		t.Fatalf("Seq = %d, want 0", forked.Seq)
	}
	if forked.Hash != "" {
		t.Fatalf("Hash = %q, want empty", forked.Hash)
	}
	if forked.EntityID != "new-camp" {
		t.Fatalf("EntityID = %q, want %q (campaign entity should be updated)", forked.EntityID, "new-camp")
	}

	// Non-campaign entity type should not change EntityID
	evt2 := event.Event{
		CampaignID: "old-camp",
		Seq:        10,
		Hash:       "def",
		EntityType: "character",
		EntityID:   "char-1",
	}
	forked2 := forkEventForCampaign(evt2, "new-camp")
	if forked2.EntityID != "char-1" {
		t.Fatalf("EntityID = %q, want %q (non-campaign entity should stay)", forked2.EntityID, "char-1")
	}
}

func TestListForks_NilRequest(t *testing.T) {
	svc := &ForkService{stores: Stores{}}
	_, err := svc.ListForks(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListForks_MissingSourceCampaignId(t *testing.T) {
	svc := &ForkService{stores: Stores{Campaign: newFakeCampaignStore()}}
	_, err := svc.ListForks(context.Background(), &statev1.ListForksRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListForks_Unimplemented(t *testing.T) {
	svc := &ForkService{stores: Stores{Campaign: newFakeCampaignStore()}}
	_, err := svc.ListForks(context.Background(), &statev1.ListForksRequest{SourceCampaignId: "camp-1"})
	assertStatusCode(t, err, codes.Unimplemented)
}

func TestForkPointFromProto(t *testing.T) {
	// Nil input
	fp := forkPointFromProto(nil)
	if fp.EventSeq != 0 || fp.SessionID != "" {
		t.Fatalf("expected zero ForkPoint for nil input, got %+v", fp)
	}

	// With values
	fp = forkPointFromProto(&statev1.ForkPoint{EventSeq: 42, SessionId: "sess-1"})
	if fp.EventSeq != 42 || fp.SessionID != "sess-1" {
		t.Fatalf("ForkPoint = %+v, want EventSeq=42 SessionID=sess-1", fp)
	}
}

func TestIsNotFound(t *testing.T) {
	if !isNotFound(storage.ErrNotFound) {
		t.Fatal("expected true for ErrNotFound")
	}
	if isNotFound(nil) {
		t.Fatal("expected false for nil")
	}
}

func appendEvent(t *testing.T, store *fakeEventStore, evt event.Event) event.Event {
	t.Helper()
	stored, err := store.AppendEvent(context.Background(), evt)
	if err != nil {
		t.Fatalf("append event failed: %v", err)
	}
	return stored
}

func mustJSON(t *testing.T, payload any) []byte {
	t.Helper()
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload failed: %v", err)
	}
	return data
}

func intPtr(value int) *int {
	return &value
}
