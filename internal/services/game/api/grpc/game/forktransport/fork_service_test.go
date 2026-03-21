package forktransport

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwriteexec"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// testRuntime is a shared write-path runtime configured once for all tests.
var testRuntime *domainwrite.Runtime

func TestMain(m *testing.M) {
	testRuntime = gametest.SetupRuntime()
	os.Exit(m.Run())
}

// --- test helpers (local copies; not exported from root package) ---

func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error with code %v", want)
	}
	statusErr, ok := status.FromError(err)
	if !ok {
		err = grpcerror.HandleDomainError(err)
		statusErr, ok = status.FromError(err)
		if !ok {
			t.Fatalf("expected gRPC status error, got %T", err)
		}
	}
	if statusErr.Code() != want {
		t.Fatalf("status code = %v, want %v (message: %s)", statusErr.Code(), want, statusErr.Message())
	}
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

func testDaggerheartProfile(overrides func(*daggerheart.CharacterProfile)) daggerheart.CharacterProfile {
	profile := daggerheart.CharacterProfile{
		Level:           1,
		HpMax:           6,
		StressMax:       6,
		Evasion:         10,
		MajorThreshold:  1,
		SevereThreshold: 2,
		Proficiency:     1,
		ArmorScore:      0,
		ArmorMax:        0,
	}
	if overrides != nil {
		overrides(&profile)
	}
	return profile
}

func newServiceForTest(
	deps Deps,
	clock func() time.Time,
	idGenerator func() (string, error),
) *Service {
	return newServiceWithDependencies(deps, clock, idGenerator)
}

// testApplier builds a projection.Applier wired to the given fake stores
// including daggerheart adapter support for tests that verify event replay
// applies projections.
func testApplier(t *testing.T, deps Deps, dhStore *gametest.FakeDaggerheartStore) projection.Applier {
	t.Helper()
	adapters, err := manifest.AdapterRegistry(dhStore)
	if err != nil {
		t.Fatalf("build adapter registry: %v", err)
	}
	return projection.Applier{
		Campaign:     deps.Campaign,
		Character:    deps.Character,
		CampaignFork: deps.CampaignFork,
		Participant:  deps.Participant,
		Adapters:     adapters,
	}
}

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
		Type:          daggerheart.EventTypeCharacterProfileReplaced,
		EntityType:    "character",
		EntityID:      "char-1",
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON: mustJSON(t, daggerheart.CharacterProfileReplacedPayload{
			CharacterID: "char-1",
			Profile: testDaggerheartProfile(func(profile *daggerheart.CharacterProfile) {
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
		PayloadJSON: mustJSON(t, daggerheart.CharacterStatePatchedPayload{
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
		Write:        domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
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
	if forkedEvents[3].Type != daggerheart.EventTypeCharacterProfileReplaced {
		t.Fatalf("event[3] type = %s, want %s", forkedEvents[3].Type, daggerheart.EventTypeCharacterProfileReplaced)
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

func TestForkCampaign_RequiresCampaignManagePolicy(t *testing.T) {
	now := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["source"] = storage.CampaignRecord{
		ID:     "source",
		Name:   "Source Campaign",
		Status: campaign.StatusActive,
	}

	participantStore := gametest.NewFakeParticipantStore()
	svc := newServiceForTest(Deps{
		Auth:         authz.PolicyDeps{Participant: participantStore},
		Campaign:     campaignStore,
		CampaignFork: gametest.NewFakeCampaignForkStore(),
		Event:        gametest.NewFakeEventStore(),
		Participant:  participantStore,
	}, gametest.FixedClock(now), gametest.FixedIDGenerator("fork-1"))

	_, err := svc.ForkCampaign(context.Background(), &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Campaign",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestForkCampaign_AllowsManagerManagePolicy(t *testing.T) {
	now := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	campaignStore.Campaigns["source"] = storage.CampaignRecord{
		ID:     "source",
		Name:   "Source Campaign",
		Status: campaign.StatusActive,
	}
	participantStore.Participants["source"] = map[string]storage.ParticipantRecord{
		"manager-1": {ID: "manager-1", CampaignID: "source", CampaignAccess: participant.CampaignAccessManager},
	}

	svc := newServiceForTest(Deps{
		Auth:         authz.PolicyDeps{Participant: participantStore},
		Campaign:     campaignStore,
		CampaignFork: gametest.NewFakeCampaignForkStore(),
		Event:        gametest.NewFakeEventStore(),
		Participant:  participantStore,
	}, gametest.FixedClock(now), gametest.FixedIDGenerator("fork-1"))

	_, err := svc.ForkCampaign(gametest.ContextWithParticipantID("manager-1"), &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Campaign",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestForkCampaign_AllowsPublicStarterTemplateForkAndReassignsOwnerSeat(t *testing.T) {
	ctx := gametest.ContextWithUserID("user-launcher")
	now := time.Date(2025, 2, 4, 10, 0, 0, 0, time.UTC)

	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	eventStore := gametest.NewFakeEventStore()
	forkStore := gametest.NewFakeCampaignForkStore()
	socialClient := &gametest.FakeSocialClient{Profile: &socialv1.UserProfile{
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
			ParticipantID:      "owner-seat",
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
		Write:        domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
	}
	deps.Applier = testApplier(t, deps, gametest.NewFakeDaggerheartStore())

	svc := newServiceForTest(deps, gametest.FixedClock(now), gametest.FixedIDGenerator("fork-1"))

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
	if forkedCharacter.ParticipantID != "owner-seat" {
		t.Fatalf("participant_id = %q, want %q", forkedCharacter.ParticipantID, "owner-seat")
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
	}, gametest.FixedClock(now), gametest.FixedIDGenerator("fork-1"))

	_, err := svc.ForkCampaign(gametest.ContextWithUserID("user-launcher"), &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Starter",
		CopyParticipants: false,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestForkCampaign_PublicSeatClaimResyncsControlledCharacterAvatar(t *testing.T) {
	ctx := gametest.ContextWithUserID("user-launcher")
	now := time.Date(2025, 2, 4, 10, 0, 0, 0, time.UTC)

	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	characterStore := gametest.NewFakeCharacterStore()
	dhStore := gametest.NewFakeDaggerheartStore()
	eventStore := gametest.NewFakeEventStore()
	forkStore := gametest.NewFakeCampaignForkStore()
	socialClient := &gametest.FakeSocialClient{Profile: &socialv1.UserProfile{
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
			ParticipantID:      "owner-seat",
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
		Write:        domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
	}
	deps.Applier = testApplier(t, deps, dhStore)

	svc := newServiceForTest(deps, gametest.FixedClock(now), gametest.FixedIDGenerator("fork-1"))

	if _, err := svc.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Starter",
		CopyParticipants: true,
	}); err != nil {
		t.Fatalf("ForkCampaign returned error: %v", err)
	}

	if len(domain.commands) != 5 {
		t.Fatalf("expected 5 commands, got %d", len(domain.commands))
	}
	if domain.commands[4].Type != command.Type("character.update") {
		t.Fatalf("command[4] type = %s, want character.update", domain.commands[4].Type)
	}

	var payload character.UpdatePayload
	if err := json.Unmarshal(domain.commands[4].PayloadJSON, &payload); err != nil {
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
	}, gametest.FixedClock(now), gametest.FixedIDGenerator("fork-1"))

	_, err := svc.ForkCampaign(gametest.ContextWithUserID("user-launcher"), &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Starter",
		CopyParticipants: true,
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestForkCampaign_DeniesMemberManagePolicy(t *testing.T) {
	now := time.Date(2025, 2, 1, 10, 0, 0, 0, time.UTC)
	campaignStore := gametest.NewFakeCampaignStore()
	participantStore := gametest.NewFakeParticipantStore()
	campaignStore.Campaigns["source"] = storage.CampaignRecord{
		ID:     "source",
		Name:   "Source Campaign",
		Status: campaign.StatusActive,
	}
	participantStore.Participants["source"] = map[string]storage.ParticipantRecord{
		"member-1": {ID: "member-1", CampaignID: "source", CampaignAccess: participant.CampaignAccessMember},
	}

	svc := newServiceForTest(Deps{
		Auth:         authz.PolicyDeps{Participant: participantStore},
		Campaign:     campaignStore,
		CampaignFork: gametest.NewFakeCampaignForkStore(),
		Event:        gametest.NewFakeEventStore(),
		Participant:  participantStore,
	}, gametest.FixedClock(now), gametest.FixedIDGenerator("fork-1"))

	_, err := svc.ForkCampaign(gametest.ContextWithParticipantID("member-1"), &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Campaign",
	})
	assertStatusCode(t, err, codes.PermissionDenied)
}

func TestGetLineage_RequiresCampaignReadPolicy(t *testing.T) {
	campaignStore := gametest.NewFakeCampaignStore()
	campaignStore.Campaigns["camp-1"] = storage.CampaignRecord{
		ID:     "camp-1",
		Status: campaign.StatusActive,
	}

	participantStore := gametest.NewFakeParticipantStore()
	svc := newServiceForTest(Deps{
		Auth:         authz.PolicyDeps{Participant: participantStore},
		Campaign:     campaignStore,
		CampaignFork: gametest.NewFakeCampaignForkStore(),
		Participant:  participantStore,
	}, nil, nil)

	_, err := svc.GetLineage(context.Background(), &statev1.GetLineageRequest{CampaignId: "camp-1"})
	assertStatusCode(t, err, codes.PermissionDenied)
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
		Write:        domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
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

func TestForkCampaign_RequiresDomainEngine(t *testing.T) {
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

	svc := newServiceForTest(Deps{
		Auth:         authz.PolicyDeps{Participant: participantStore, Character: characterStore},
		Campaign:     campaignStore,
		Participant:  participantStore,
		Character:    characterStore,
		Event:        eventStore,
		CampaignFork: forkStore,
	}, gametest.FixedClock(now), gametest.FixedIDGenerator("fork-1"))

	_, err := svc.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Campaign",
		CopyParticipants: false,
	})
	assertStatusCode(t, err, codes.Internal)
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
		Type:          daggerheart.EventTypeCharacterProfileReplaced,
		EntityType:    "character",
		EntityID:      "char-1",
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON: mustJSON(t, daggerheart.CharacterProfileReplacedPayload{
			CharacterID: "char-1",
			Profile: testDaggerheartProfile(func(profile *daggerheart.CharacterProfile) {
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
		PayloadJSON: mustJSON(t, daggerheart.CharacterStatePatchedPayload{
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
		PayloadJSON: mustJSON(t, daggerheart.GMFearChangedPayload{
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
		Write:        domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
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
		Write:        domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
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
		PayloadJSON: mustJSON(t, daggerheart.GMFearChangedPayload{
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
		Write:        domainwriteexec.WritePath{Executor: domain, Runtime: testRuntime},
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

func TestForkCampaign_RejectsWhenSourceCampaignHasActiveSession(t *testing.T) {
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
	sessionStore.Sessions["source"] = map[string]storage.SessionRecord{
		"sess-1": {
			ID:         "sess-1",
			CampaignID: "source",
			Name:       "Active Session",
			Status:     session.StatusActive,
			StartedAt:  now.Add(-1 * time.Hour),
		},
	}
	sessionStore.ActiveSession["source"] = "sess-1"

	svc := newServiceForTest(Deps{
		Auth:         authz.PolicyDeps{Participant: participantStore, Character: characterStore},
		Campaign:     campaignStore,
		Participant:  participantStore,
		Character:    characterStore,
		Session:      sessionStore,
		Event:        eventStore,
		CampaignFork: forkStore,
		Write:        domainwriteexec.WritePath{Executor: &fakeDomainEngine{store: eventStore}},
	}, gametest.FixedClock(now), gametest.FixedIDGenerator("fork-1"))

	_, err := svc.ForkCampaign(ctx, &statev1.ForkCampaignRequest{
		SourceCampaignId: "source",
		NewCampaignName:  "Forked Campaign",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
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
	svc := newServiceForTest(Deps{}, nil, nil)
	_, err := svc.ListForks(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListForks_MissingSourceCampaignId(t *testing.T) {
	svc := newServiceForTest(Deps{Campaign: gametest.NewFakeCampaignStore()}, nil, nil)
	_, err := svc.ListForks(context.Background(), &statev1.ListForksRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListForks_Unimplemented(t *testing.T) {
	svc := newServiceForTest(Deps{Campaign: gametest.NewFakeCampaignStore()}, nil, nil)
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

func appendEvent(t *testing.T, store *gametest.FakeEventStore, evt event.Event) event.Event {
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
