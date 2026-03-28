package forktransport

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/requestctx"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/runtimekit"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/testclients"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	daggerhearttestkit "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/testkit"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
)

func TestForkCampaign_AllowsPublicStarterTemplateForkAndReassignsOwnerSeat(t *testing.T) {
	ctx := requestctx.WithUserID(context.Background(), "user-launcher")
	now := time.Date(2025, 2, 4, 10, 0, 0, 0, time.UTC)

	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	eventStore := gametest.NewFakeEventStore()
	forkStore := gametest.NewFakeCampaignForkStore()
	socialClient := &testclients.FakeSocialClient{Profile: &socialv1.UserProfile{
		UserId: "user-launcher",
		Name:   "Launcher Name",
	}}

	campaignStore.Campaigns["source"] = storage.CampaignRecord{
		ID:           "source",
		Name:         "Starter Template",
		Status:       campaign.StatusActive,
		System:       bridge.SystemIDDaggerheart,
		GmMode:       campaign.GmModeAI,
		Intent:       campaign.IntentStarter,
		AccessPolicy: campaign.AccessPolicyPublic,
		ThemePrompt:  "starter theme",
	}
	participantStore.Participants["source"] = map[string]storage.ParticipantRecord{
		"owner-seat": {
			ID:             "owner-seat",
			CampaignID:     "source",
			Name:           "Template Hero",
			Role:           participant.RolePlayer,
			Controller:     participant.ControllerHuman,
			CampaignAccess: participant.CampaignAccessOwner,
		},
		"gm-seat": {
			ID:             "gm-seat",
			CampaignID:     "source",
			Name:           "AI GM",
			Role:           participant.RoleGM,
			Controller:     participant.ControllerAI,
			CampaignAccess: participant.CampaignAccessOwner,
		},
	}

	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-3 * time.Hour),
		Type:       event.Type("campaign.created"),
		EntityType: "campaign",
		EntityID:   "source",
		PayloadJSON: mustJSON(t, campaign.CreatePayload{
			Name:         "Starter Template",
			GameSystem:   commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
			GmMode:       statev1.GmMode_AI.String(),
			Intent:       statev1.CampaignIntent_STARTER.String(),
			AccessPolicy: statev1.CampaignAccessPolicy_PUBLIC.String(),
			ThemePrompt:  "starter theme",
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-2 * time.Hour),
		Type:       event.Type("participant.joined"),
		EntityType: "participant",
		EntityID:   "owner-seat",
		PayloadJSON: mustJSON(t, participant.JoinPayload{
			ParticipantID:  "owner-seat",
			Name:           "Template Hero",
			Role:           "PLAYER",
			Controller:     "CONTROLLER_HUMAN",
			CampaignAccess: "OWNER",
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-110 * time.Minute),
		Type:       event.Type("participant.joined"),
		EntityType: "participant",
		EntityID:   "gm-seat",
		PayloadJSON: mustJSON(t, participant.JoinPayload{
			ParticipantID:  "gm-seat",
			Name:           "AI GM",
			Role:           "GM",
			Controller:     "CONTROLLER_AI",
			CampaignAccess: "OWNER",
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-100 * time.Minute),
		Type:       event.Type("character.created"),
		EntityType: "character",
		EntityID:   "char-1",
		PayloadJSON: mustJSON(t, character.CreatePayload{
			CharacterID:        "char-1",
			OwnerParticipantID: "owner-seat",
			Name:               "Ser Rowan",
			Kind:               "PC",
		}),
	})

	createdJSON := mustJSON(t, campaign.CreatePayload{
		Name:         "Forked Starter",
		GameSystem:   commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		GmMode:       statev1.GmMode_AI.String(),
		Intent:       statev1.CampaignIntent_STARTER.String(),
		AccessPolicy: statev1.CampaignAccessPolicy_PUBLIC.String(),
		ThemePrompt:  "starter theme",
	})
	forkedJSON := mustJSON(t, campaign.ForkPayload{
		ParentCampaignID: "source",
		ForkEventSeq:     4,
		OriginCampaignID: "source",
		CopyParticipants: true,
	})
	seatReassignedJSON := mustJSON(t, participant.SeatReassignPayload{
		ParticipantID: "owner-seat",
		PriorUserID:   "",
		UserID:        "user-launcher",
		Reason:        "public_fork_claim",
	})
	participantUpdatedJSON := mustJSON(t, participant.UpdatePayload{
		ParticipantID: "owner-seat",
		Fields: map[string]string{
			"name": "Launcher Name",
		},
	})

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
		command.Type("participant.seat.reassign"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "fork-1",
				Type:        event.Type("participant.seat_reassigned"),
				Timestamp:   now.Add(time.Minute),
				ActorType:   event.ActorTypeSystem,
				EntityType:  "participant",
				EntityID:    "owner-seat",
				PayloadJSON: seatReassignedJSON,
			}),
		},
		command.Type("participant.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "fork-1",
				Type:        event.Type("participant.updated"),
				Timestamp:   now.Add(2 * time.Minute),
				ActorType:   event.ActorTypeSystem,
				EntityType:  "participant",
				EntityID:    "owner-seat",
				PayloadJSON: participantUpdatedJSON,
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
		Social:       socialClient,
		Write:        domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
	}
	deps.Applier = testApplier(t, deps, daggerhearttestkit.NewFakeDaggerheartStore())

	svc := newServiceForTest(deps, runtimekit.FixedClock(now), runtimekit.FixedIDGenerator("fork-1"))

	resp, err := svc.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Starter",
		CopyParticipants: true,
	})
	if err != nil {
		t.Fatalf("ForkCampaign returned error: %v", err)
	}
	if resp.Campaign.GetId() != "fork-1" {
		t.Fatalf("Campaign ID = %q, want %q", resp.Campaign.GetId(), "fork-1")
	}

	ownerSeat, err := participantStore.GetParticipant(ctx, "fork-1", "owner-seat")
	if err != nil {
		t.Fatalf("GetParticipant owner-seat returned error: %v", err)
	}
	if ownerSeat.UserID != "user-launcher" {
		t.Fatalf("owner seat user_id = %q, want %q", ownerSeat.UserID, "user-launcher")
	}
	if ownerSeat.Name != "Launcher Name" {
		t.Fatalf("owner seat name = %q, want %q", ownerSeat.Name, "Launcher Name")
	}

	aiSeat, err := participantStore.GetParticipant(ctx, "fork-1", "gm-seat")
	if err != nil {
		t.Fatalf("GetParticipant gm-seat returned error: %v", err)
	}
	if aiSeat.UserID != "" {
		t.Fatalf("ai seat user_id = %q, want empty", aiSeat.UserID)
	}
	if aiSeat.Controller != participant.ControllerAI {
		t.Fatalf("ai seat controller = %q, want %q", aiSeat.Controller, participant.ControllerAI)
	}

	forkedCharacter, err := characterStore.GetCharacter(ctx, "fork-1", "char-1")
	if err != nil {
		t.Fatalf("GetCharacter returned error: %v", err)
	}
	if forkedCharacter.OwnerParticipantID != "owner-seat" {
		t.Fatalf("owner_participant_id = %q, want %q", forkedCharacter.OwnerParticipantID, "owner-seat")
	}
	if socialClient.GetUserProfileCalls != 1 {
		t.Fatalf("GetUserProfile calls = %d, want 1", socialClient.GetUserProfileCalls)
	}
	if len(domain.commands) != 4 {
		t.Fatalf("expected 4 domain commands, got %d", len(domain.commands))
	}
	if domain.commands[2].Type != command.Type("participant.seat.reassign") {
		t.Fatalf("command[2] type = %s, want %s", domain.commands[2].Type, "participant.seat.reassign")
	}
	if domain.commands[3].Type != command.Type("participant.update") {
		t.Fatalf("command[3] type = %s, want %s", domain.commands[3].Type, "participant.update")
	}
}

func TestForkCampaign_RejectsPublicCampaignForkWithoutParticipantCopy(t *testing.T) {
	now := time.Date(2025, 2, 4, 10, 0, 0, 0, time.UTC)
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["source"] = storage.CampaignRecord{
		ID:           "source",
		Name:         "Starter Template",
		Status:       campaign.StatusActive,
		Intent:       campaign.IntentStarter,
		AccessPolicy: campaign.AccessPolicyPublic,
	}

	svc := newServiceForTest(Deps{
		Campaign:     campaignStore,
		CampaignFork: gametest.NewFakeCampaignForkStore(),
		Event:        gametest.NewFakeEventStore(),
		Participant:  gametest.NewFakeParticipantStore(),
	}, runtimekit.FixedClock(now), runtimekit.FixedIDGenerator("fork-1"))

	_, err := svc.ForkCampaign(requestctx.WithUserID(context.Background(), "user-launcher"), &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Starter",
		CopyParticipants: false,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestForkCampaign_PublicSeatClaimResyncsControlledCharacterAvatar(t *testing.T) {
	ctx := requestctx.WithUserID(context.Background(), "user-launcher")
	now := time.Date(2025, 2, 4, 10, 0, 0, 0, time.UTC)

	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	dhStore := daggerhearttestkit.NewFakeDaggerheartStore()
	eventStore := gametest.NewFakeEventStore()
	forkStore := gametest.NewFakeCampaignForkStore()
	socialClient := &testclients.FakeSocialClient{Profile: &socialv1.UserProfile{
		UserId:        "user-launcher",
		Name:          "Launcher Name",
		AvatarSetId:   "avatar-set-1",
		AvatarAssetId: "avatar-asset-1",
	}}

	campaignStore.Campaigns["source"] = storage.CampaignRecord{
		ID:           "source",
		Name:         "Starter Template",
		Status:       campaign.StatusActive,
		System:       bridge.SystemIDDaggerheart,
		GmMode:       campaign.GmModeAI,
		Intent:       campaign.IntentStarter,
		AccessPolicy: campaign.AccessPolicyPublic,
		ThemePrompt:  "starter theme",
	}
	participantStore.Participants["source"] = map[string]storage.ParticipantRecord{
		"owner-seat": {
			ID:             "owner-seat",
			CampaignID:     "source",
			Name:           "Template Hero",
			Role:           participant.RolePlayer,
			Controller:     participant.ControllerHuman,
			CampaignAccess: participant.CampaignAccessOwner,
		},
		"gm-seat": {
			ID:             "gm-seat",
			CampaignID:     "source",
			Name:           "AI GM",
			Role:           participant.RoleGM,
			Controller:     participant.ControllerAI,
			CampaignAccess: participant.CampaignAccessOwner,
		},
	}

	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-3 * time.Hour),
		Type:       event.Type("campaign.created"),
		EntityType: "campaign",
		EntityID:   "source",
		PayloadJSON: mustJSON(t, campaign.CreatePayload{
			Name:         "Starter Template",
			GameSystem:   commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
			GmMode:       statev1.GmMode_AI.String(),
			Intent:       statev1.CampaignIntent_STARTER.String(),
			AccessPolicy: statev1.CampaignAccessPolicy_PUBLIC.String(),
			ThemePrompt:  "starter theme",
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-2 * time.Hour),
		Type:       event.Type("participant.joined"),
		EntityType: "participant",
		EntityID:   "owner-seat",
		PayloadJSON: mustJSON(t, participant.JoinPayload{
			ParticipantID:  "owner-seat",
			Name:           "Template Hero",
			Role:           "PLAYER",
			Controller:     "CONTROLLER_HUMAN",
			CampaignAccess: "OWNER",
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-110 * time.Minute),
		Type:       event.Type("participant.joined"),
		EntityType: "participant",
		EntityID:   "gm-seat",
		PayloadJSON: mustJSON(t, participant.JoinPayload{
			ParticipantID:  "gm-seat",
			Name:           "AI GM",
			Role:           "GM",
			Controller:     "CONTROLLER_AI",
			CampaignAccess: "OWNER",
		}),
	})
	appendEvent(t, eventStore, event.Event{
		CampaignID: "source",
		Timestamp:  now.Add(-100 * time.Minute),
		Type:       event.Type("character.created"),
		EntityType: "character",
		EntityID:   "char-1",
		PayloadJSON: mustJSON(t, character.CreatePayload{
			CharacterID:        "char-1",
			OwnerParticipantID: "owner-seat",
			Name:               "Ser Rowan",
			Kind:               "PC",
			AvatarSetID:        "template-set",
			AvatarAssetID:      "template-asset",
			Pronouns:           "character-pronouns",
		}),
	})

	createdJSON := mustJSON(t, campaign.CreatePayload{
		Name:         "Forked Starter",
		GameSystem:   commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
		GmMode:       statev1.GmMode_AI.String(),
		Intent:       statev1.CampaignIntent_STARTER.String(),
		AccessPolicy: statev1.CampaignAccessPolicy_PUBLIC.String(),
		ThemePrompt:  "starter theme",
	})
	forkedJSON := mustJSON(t, campaign.ForkPayload{
		ParentCampaignID: "source",
		ForkEventSeq:     4,
		OriginCampaignID: "source",
		CopyParticipants: true,
	})
	seatReassignedJSON := mustJSON(t, participant.SeatReassignPayload{
		ParticipantID: "owner-seat",
		PriorUserID:   "",
		UserID:        "user-launcher",
		Reason:        "public_fork_claim",
	})
	participantUpdatedJSON := mustJSON(t, participant.UpdatePayload{
		ParticipantID: "owner-seat",
		Fields: map[string]string{
			"name":            "Launcher Name",
			"pronouns":        "they/them",
			"avatar_set_id":   "avatar-set-1",
			"avatar_asset_id": "avatar-asset-1",
		},
	})
	characterUpdatedJSON := mustJSON(t, character.UpdatePayload{
		CharacterID: "char-1",
		Fields: map[string]string{
			"avatar_set_id":   "avatar-set-1",
			"avatar_asset_id": "avatar-asset-1",
		},
	})

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
		command.Type("participant.seat.reassign"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "fork-1",
				Type:        event.Type("participant.seat_reassigned"),
				Timestamp:   now.Add(time.Minute),
				ActorType:   event.ActorTypeSystem,
				EntityType:  "participant",
				EntityID:    "owner-seat",
				PayloadJSON: seatReassignedJSON,
			}),
		},
		command.Type("participant.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "fork-1",
				Type:        event.Type("participant.updated"),
				Timestamp:   now.Add(2 * time.Minute),
				ActorType:   event.ActorTypeSystem,
				EntityType:  "participant",
				EntityID:    "owner-seat",
				PayloadJSON: participantUpdatedJSON,
			}),
		},
		command.Type("character.update"): {
			Decision: command.Accept(event.Event{
				CampaignID:  "fork-1",
				Type:        event.Type("character.updated"),
				Timestamp:   now.Add(3 * time.Minute),
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    "char-1",
				PayloadJSON: characterUpdatedJSON,
			}),
		},
	}}

	deps := Deps{
		Campaign:     campaignStore,
		Participant:  participantStore,
		Character:    characterStore,
		Event:        eventStore,
		CampaignFork: forkStore,
		Social:       socialClient,
		Write:        domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
	}
	deps.Applier = testApplier(t, deps, dhStore)

	svc := newServiceForTest(deps, runtimekit.FixedClock(now), runtimekit.FixedIDGenerator("fork-1"))

	if _, err := svc.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Starter",
		CopyParticipants: true,
	}); err != nil {
		t.Fatalf("ForkCampaign returned error: %v", err)
	}

	if len(domain.commands) != 6 {
		t.Fatalf("expected 6 commands, got %d", len(domain.commands))
	}
	if domain.commands[5].Type != command.Type("character.update") {
		t.Fatalf("command[5] type = %s, want character.update", domain.commands[5].Type)
	}

	var payload character.UpdatePayload
	if err := json.Unmarshal(domain.commands[5].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode character update payload: %v", err)
	}
	if payload.Fields["avatar_set_id"] != "avatar-set-1" {
		t.Fatalf("avatar_set_id = %q, want %q", payload.Fields["avatar_set_id"], "avatar-set-1")
	}
	if payload.Fields["avatar_asset_id"] != "avatar-asset-1" {
		t.Fatalf("avatar_asset_id = %q, want %q", payload.Fields["avatar_asset_id"], "avatar-asset-1")
	}
	if _, ok := payload.Fields["pronouns"]; ok {
		t.Fatalf("pronouns field should be omitted, got %q", payload.Fields["pronouns"])
	}

	forkedCharacter, err := characterStore.GetCharacter(ctx, "fork-1", "char-1")
	if err != nil {
		t.Fatalf("GetCharacter returned error: %v", err)
	}
	if forkedCharacter.AvatarSetID != "avatar-set-1" || forkedCharacter.AvatarAssetID != "avatar-asset-1" {
		t.Fatalf("character avatar = %q/%q, want avatar-set-1/avatar-asset-1", forkedCharacter.AvatarSetID, forkedCharacter.AvatarAssetID)
	}
	if forkedCharacter.Pronouns != "character-pronouns" {
		t.Fatalf("character pronouns = %q, want %q", forkedCharacter.Pronouns, "character-pronouns")
	}
}

func TestForkCampaign_RejectsInvalidPublicStarterTemplateShape(t *testing.T) {
	now := time.Date(2025, 2, 4, 10, 0, 0, 0, time.UTC)
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	campaignStore.Campaigns["source"] = storage.CampaignRecord{
		ID:           "source",
		Name:         "Starter Template",
		Status:       campaign.StatusActive,
		Intent:       campaign.IntentStarter,
		AccessPolicy: campaign.AccessPolicyPublic,
	}
	participantStore.Participants["source"] = map[string]storage.ParticipantRecord{
		"owner-seat": {
			ID:             "owner-seat",
			CampaignID:     "source",
			UserID:         "template-user",
			Name:           "Template Hero",
			Role:           participant.RolePlayer,
			Controller:     participant.ControllerHuman,
			CampaignAccess: participant.CampaignAccessOwner,
		},
	}

	svc := newServiceForTest(Deps{
		Campaign:     campaignStore,
		CampaignFork: gametest.NewFakeCampaignForkStore(),
		Event:        gametest.NewFakeEventStore(),
		Participant:  participantStore,
	}, runtimekit.FixedClock(now), runtimekit.FixedIDGenerator("fork-1"))

	_, err := svc.ForkCampaign(requestctx.WithUserID(context.Background(), "user-launcher"), &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Starter",
		CopyParticipants: true,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}
