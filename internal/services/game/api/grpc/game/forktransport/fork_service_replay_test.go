package forktransport

import (
	"encoding/json"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartpayload "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func TestForkCampaign_ReplaysEvents_CopyParticipantsFalse(t *testing.T) {
	ctx := gametest.ContextWithAdminOverride("fork-test")
	now := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)

	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	eventStore := gametest.NewFakeEventStore()
	forkStore := gametest.NewFakeCampaignForkStore()

	campaignStore.Campaigns["source"] = storage.CampaignRecord{
		ID:          "source",
		Name:        "Source Campaign",
		Status:      campaign.StatusActive,
		System:      bridge.SystemIDDaggerheart,
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
		CampaignID:    "source",
		Timestamp:     now.Add(-7 * time.Hour),
		Type:          daggerheartpayload.EventTypeCharacterProfileReplaced,
		EntityType:    "character",
		EntityID:      "char-1",
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON: mustJSON(t, daggerheartstate.CharacterProfileReplacedPayload{
			CharacterID: "char-1",
			Profile: testDaggerheartProfile(func(profile *daggerheartstate.CharacterProfile) {
				profile.HpMax = 6
				profile.StressMax = 6
				profile.Evasion = 11
				profile.Agility = 1
				profile.Strength = 1
				profile.Finesse = 1
				profile.Instinct = 1
				profile.Presence = 1
				profile.Knowledge = 1
				profile.MajorThreshold = 3
				profile.SevereThreshold = 5
			}),
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
		Type:          event.Type("sys.daggerheart.character_state_patched"),
		EntityType:    "character",
		EntityID:      "char-1",
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON: mustJSON(t, daggerheartpayload.CharacterStatePatchedPayload{
			CharacterID: "char-1",
			HP:          intPtr(6),
			Hope:        intPtr(2),
			Stress:      intPtr(1),
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

	deps := Deps{
		Auth:         authz.PolicyDeps{Participant: participantStore, Character: characterStore},
		Campaign:     campaignStore,
		Participant:  participantStore,
		Character:    characterStore,
		Event:        eventStore,
		CampaignFork: forkStore,
		Write:        domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
	}
	deps.Applier = testApplier(t, deps, dhStore)

	svc := newServiceForTest(deps, gametest.FixedClock(now), gametest.FixedIDGenerator("fork-1"))

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

	forkedEvents := eventStore.Events["fork-1"]
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
	if forkedEvents[3].Type != daggerheartpayload.EventTypeCharacterProfileReplaced {
		t.Fatalf("event[3] type = %s, want %s", forkedEvents[3].Type, daggerheartpayload.EventTypeCharacterProfileReplaced)
	}
	if forkedEvents[4].Type != event.Type("sys.daggerheart.character_state_patched") {
		t.Fatalf("event[4] type = %s, want %s", forkedEvents[4].Type, event.Type("sys.daggerheart.character_state_patched"))
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

func TestForkCampaign_CopiesAuditOnlyEventsWithoutProjectionApplyFailure(t *testing.T) {
	ctx := gametest.ContextWithAdminOverride("fork-test")
	now := time.Date(2025, 2, 3, 11, 0, 0, 0, time.UTC)

	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	eventStore := gametest.NewFakeEventStore()
	forkStore := gametest.NewFakeCampaignForkStore()

	campaignStore.Campaigns["source"] = storage.CampaignRecord{
		ID:          "source",
		Name:        "Source Campaign",
		Status:      campaign.StatusActive,
		System:      bridge.SystemIDDaggerheart,
		GmMode:      campaign.GmModeHuman,
		ThemePrompt: "theme",
	}

	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-2 * time.Hour),
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
		Timestamp:  now.Add(-90 * time.Minute),
		Type:       event.Type("story.note_added"),
		EntityType: "note",
		EntityID:   "note-1",
		PayloadJSON: mustJSON(t, action.NoteAddPayload{
			Content: "Fork note",
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
		ForkEventSeq:     2,
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

	deps := Deps{
		Auth:         authz.PolicyDeps{Participant: participantStore, Character: characterStore},
		Campaign:     campaignStore,
		Participant:  participantStore,
		Character:    characterStore,
		Event:        eventStore,
		CampaignFork: forkStore,
		Write:        domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
	}
	deps.Applier = testApplier(t, deps, gametest.NewFakeDaggerheartStore())

	svc := newServiceForTest(deps, gametest.FixedClock(now), gametest.FixedIDGenerator("fork-1"))

	if _, err := svc.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Campaign",
		CopyParticipants: false,
	}); err != nil {
		t.Fatalf("ForkCampaign returned error: %v", err)
	}

	forkedEvents := eventStore.Events["fork-1"]
	if len(forkedEvents) != 3 {
		t.Fatalf("expected 3 forked events, got %d", len(forkedEvents))
	}
	if forkedEvents[2].Type != event.Type("story.note_added") {
		t.Fatalf("event[2] type = %s, want %s", forkedEvents[2].Type, event.Type("story.note_added"))
	}
}

func TestForkCampaign_SeedsSnapshotStateAtHead(t *testing.T) {
	ctx := gametest.ContextWithAdminOverride("fork-test")
	now := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)

	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	eventStore := gametest.NewFakeEventStore()
	forkStore := gametest.NewFakeCampaignForkStore()

	campaignStore.Campaigns["source"] = storage.CampaignRecord{
		ID:          "source",
		Name:        "Source Campaign",
		Status:      campaign.StatusActive,
		System:      bridge.SystemIDDaggerheart,
		GmMode:      campaign.GmModeHuman,
		ThemePrompt: "theme",
	}
	characterStore.Characters["source"] = map[string]storage.CharacterRecord{
		"char-1": {
			ID:         "char-1",
			CampaignID: "source",
			Name:       "Hero",
			Kind:       character.KindPC,
			CreatedAt:  now.Add(-8 * time.Hour),
			UpdatedAt:  now.Add(-8 * time.Hour),
		},
	}
	dhStore.States["source"] = map[string]projectionstore.DaggerheartCharacterState{
		"char-1": {
			CampaignID:  "source",
			CharacterID: "char-1",
			Hp:          6,
			Hope:        2,
			Stress:      1,
		},
	}
	dhStore.Snapshots["source"] = projectionstore.DaggerheartSnapshot{
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
		CampaignID:    "source",
		Timestamp:     now.Add(-7 * time.Hour),
		Type:          daggerheartpayload.EventTypeCharacterProfileReplaced,
		EntityType:    "character",
		EntityID:      "char-1",
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON: mustJSON(t, daggerheartstate.CharacterProfileReplacedPayload{
			CharacterID: "char-1",
			Profile: testDaggerheartProfile(func(profile *daggerheartstate.CharacterProfile) {
				profile.HpMax = 6
				profile.StressMax = 6
				profile.Evasion = 11
				profile.Agility = 1
				profile.Strength = 1
				profile.Finesse = 1
				profile.Instinct = 1
				profile.Presence = 1
				profile.Knowledge = 1
				profile.MajorThreshold = 3
				profile.SevereThreshold = 5
			}),
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
		Type:          event.Type("sys.daggerheart.character_state_patched"),
		EntityType:    "character",
		EntityID:      "char-1",
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON: mustJSON(t, daggerheartpayload.CharacterStatePatchedPayload{
			CharacterID: "char-1",
			HP:          intPtr(6),
			Hope:        intPtr(2),
			Stress:      intPtr(1),
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID:    "source",
		Timestamp:     now.Add(-4 * time.Hour),
		Type:          event.Type("sys.daggerheart.gm_fear_changed"),
		EntityType:    "campaign",
		EntityID:      "source",
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON: mustJSON(t, daggerheartpayload.GMFearChangedPayload{
			Value: 4,
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

	deps := Deps{
		Auth:         authz.PolicyDeps{Participant: participantStore, Character: characterStore},
		Campaign:     campaignStore,
		Participant:  participantStore,
		Character:    characterStore,
		Event:        eventStore,
		CampaignFork: forkStore,
		Write:        domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
	}
	deps.Applier = testApplier(t, deps, dhStore)

	svc := newServiceForTest(deps, gametest.FixedClock(now), gametest.FixedIDGenerator("fork-1"))

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

	if dhStore.StatePuts["fork-1"] != 2 {
		t.Fatalf("daggerheart state puts = %d, want 2", dhStore.StatePuts["fork-1"])
	}
	if dhStore.SnapPuts["fork-1"] != 1 {
		t.Fatalf("daggerheart snapshot puts = %d, want 1", dhStore.SnapPuts["fork-1"])
	}
}

func TestForkCampaign_UsesDomainEngine(t *testing.T) {
	ctx := gametest.ContextWithAdminOverride("fork-test")
	now := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)

	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	eventStore := gametest.NewFakeEventStore()
	forkStore := gametest.NewFakeCampaignForkStore()

	campaignStore.Campaigns["source"] = storage.CampaignRecord{
		ID:          "source",
		Name:        "Source Campaign",
		Status:      campaign.StatusActive,
		System:      bridge.SystemIDDaggerheart,
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

	deps := Deps{
		Auth:         authz.PolicyDeps{Participant: participantStore, Character: characterStore},
		Campaign:     campaignStore,
		Participant:  participantStore,
		Character:    characterStore,
		Event:        eventStore,
		CampaignFork: forkStore,
		Write:        domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
	}
	deps.Applier = testApplier(t, deps, gametest.NewFakeDaggerheartStore())
	svc := newServiceForTest(deps, gametest.FixedClock(now), gametest.FixedIDGenerator("fork-1"))

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
	if got := len(eventStore.Events["fork-1"]); got != 2 {
		t.Fatalf("expected 2 events, got %d", got)
	}
	if eventStore.Events["fork-1"][0].Type != event.Type("campaign.created") {
		t.Fatalf("event[0] type = %s, want %s", eventStore.Events["fork-1"][0].Type, event.Type("campaign.created"))
	}
	if eventStore.Events["fork-1"][1].Type != event.Type("campaign.forked") {
		t.Fatalf("event[1] type = %s, want %s", eventStore.Events["fork-1"][1].Type, event.Type("campaign.forked"))
	}
}

func TestForkCampaign_SessionBoundaryForkPoint(t *testing.T) {
	ctx := gametest.ContextWithAdminOverride("fork-test")
	now := time.Date(2025, 2, 2, 9, 0, 0, 0, time.UTC)

	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	eventStore := gametest.NewFakeEventStore()
	forkStore := gametest.NewFakeCampaignForkStore()
	sessionStore := gametest.NewFakeSessionStore()

	campaignStore.Campaigns["source"] = storage.CampaignRecord{
		ID:          "source",
		Name:        "Source Campaign",
		Status:      campaign.StatusActive,
		System:      bridge.SystemIDDaggerheart,
		GmMode:      campaign.GmModeHuman,
		ThemePrompt: "theme",
	}
	endedAt := now.Add(-30 * time.Minute)
	sessionStore.Sessions["source"] = map[string]storage.SessionRecord{
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
		Type:          event.Type("sys.daggerheart.gm_fear_changed"),
		EntityType:    "campaign",
		EntityID:      "source",
		SessionID:     "sess-1",
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON: mustJSON(t, daggerheartpayload.GMFearChangedPayload{
			Value: 2,
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

	deps := Deps{
		Auth:         authz.PolicyDeps{Participant: participantStore, Character: characterStore},
		Campaign:     campaignStore,
		Participant:  participantStore,
		Character:    characterStore,
		Event:        eventStore,
		CampaignFork: forkStore,
		Session:      sessionStore,
		Write:        domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
	}
	deps.Applier = testApplier(t, deps, gametest.NewFakeDaggerheartStore())
	svc := newServiceForTest(deps, gametest.FixedClock(now), gametest.FixedIDGenerator("fork-1"))

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

	forkedEvents := eventStore.Events["fork-1"]
	if len(forkedEvents) != 4 {
		t.Fatalf("expected 4 forked events, got %d", len(forkedEvents))
	}
}
