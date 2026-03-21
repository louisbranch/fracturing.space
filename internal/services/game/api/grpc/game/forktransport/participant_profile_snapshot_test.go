package forktransport

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/testclients"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	participantdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
)

func TestApplyParticipantProfileSnapshot_AvatarlessSnapshotSkipsCharacterSync(t *testing.T) {
	domain := &fakeDomainEngine{}

	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["camp-1"] = map[string]storage.ParticipantRecord{
		"seat-1": {ID: "seat-1", CampaignID: "camp-1"},
	}
	characterStore := gametest.NewFakeCharacterStore()
	characterStore.Characters["camp-1"] = map[string]storage.CharacterRecord{
		"char-1": {ID: "char-1", CampaignID: "camp-1", ParticipantID: "seat-1"},
	}

	applyParticipantProfileSnapshot(
		context.Background(),
		domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
		projection.Applier{},
		participantStore,
		characterStore,
		&testclients.FakeSocialClient{Profile: &socialv1.UserProfile{
			UserId:   "user-1",
			Name:     "Ariadne",
			Pronouns: sharedpronouns.ToProto("she/her"),
		}},
		"camp-1",
		"seat-1",
		"user-1",
		"req-1",
		"inv-1",
		"seat-1",
		command.ActorTypeParticipant,
	)

	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != handler.CommandTypeParticipantUpdate {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, handler.CommandTypeParticipantUpdate)
	}

	var payload participantdomain.UpdatePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode participant update payload: %v", err)
	}
	if _, ok := payload.Fields["avatar_set_id"]; ok {
		t.Fatalf("avatar_set_id should be omitted, got %q", payload.Fields["avatar_set_id"])
	}
	if _, ok := payload.Fields["avatar_asset_id"]; ok {
		t.Fatalf("avatar_asset_id should be omitted, got %q", payload.Fields["avatar_asset_id"])
	}
}

func TestSyncControlledCharacterAvatars_UpdatesOnlyMismatchedCharacters(t *testing.T) {
	domain := &fakeDomainEngine{}

	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["camp-1"] = map[string]storage.ParticipantRecord{
		"seat-1": {
			ID:            "seat-1",
			CampaignID:    "camp-1",
			AvatarSetID:   "avatar-set-1",
			AvatarAssetID: "avatar-asset-1",
		},
	}
	characterStore := gametest.NewFakeCharacterStore()
	characterStore.Characters["camp-1"] = map[string]storage.CharacterRecord{
		"char-same": {
			ID:            "char-same",
			CampaignID:    "camp-1",
			ParticipantID: "seat-1",
			AvatarSetID:   "avatar-set-1",
			AvatarAssetID: "avatar-asset-1",
		},
		"char-stale": {
			ID:            "char-stale",
			CampaignID:    "camp-1",
			ParticipantID: "seat-1",
			AvatarSetID:   "old-set",
			AvatarAssetID: "old-asset",
		},
		"char-other": {
			ID:            "char-other",
			CampaignID:    "camp-1",
			ParticipantID: "seat-2",
			AvatarSetID:   "old-set",
			AvatarAssetID: "old-asset",
		},
	}

	syncControlledCharacterAvatars(
		context.Background(),
		domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
		projection.Applier{},
		participantStore,
		characterStore,
		"camp-1",
		"seat-1",
		"req-1",
		"inv-1",
		"seat-1",
		command.ActorTypeParticipant,
	)

	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != handler.CommandTypeCharacterUpdate {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, handler.CommandTypeCharacterUpdate)
	}
	if domain.commands[0].EntityID != "char-stale" {
		t.Fatalf("entity id = %q, want %q", domain.commands[0].EntityID, "char-stale")
	}

	var payload character.UpdatePayload
	if err := json.Unmarshal(domain.commands[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("decode character update payload: %v", err)
	}
	if payload.Fields["avatar_set_id"] != "avatar-set-1" {
		t.Fatalf("avatar_set_id = %q, want %q", payload.Fields["avatar_set_id"], "avatar-set-1")
	}
	if payload.Fields["avatar_asset_id"] != "avatar-asset-1" {
		t.Fatalf("avatar_asset_id = %q, want %q", payload.Fields["avatar_asset_id"], "avatar-asset-1")
	}
}

func TestSyncControlledCharacterAvatars_SkipsWhenParticipantLookupFails(t *testing.T) {
	domain := &fakeDomainEngine{}

	participantStore := gametest.NewFakeParticipantStore()
	participantStore.GetErr = errors.New("boom")

	syncControlledCharacterAvatars(
		context.Background(),
		domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
		projection.Applier{},
		participantStore,
		gametest.NewFakeCharacterStore(),
		"camp-1",
		"seat-1",
		"req-1",
		"inv-1",
		"seat-1",
		command.ActorTypeParticipant,
	)

	if len(domain.commands) != 0 {
		t.Fatalf("expected no commands, got %d", len(domain.commands))
	}
}

func TestApplyParticipantProfileSnapshot_SkipsCharacterSyncWhenParticipantUpdateFails(t *testing.T) {
	eventStore := gametest.NewFakeEventStore()
	eventStore.AppendErr = errors.New("boom")
	domain := &fakeDomainEngine{
		store: eventStore,
		resultsByType: map[command.Type]engine.Result{
			handler.CommandTypeParticipantUpdate: {
				Decision: command.Accept(event.Event{
					CampaignID: "camp-1",
					Type:       handler.EventTypeParticipantUpdated,
					EntityType: "participant",
					EntityID:   "seat-1",
				}),
			},
		},
	}

	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["camp-1"] = map[string]storage.ParticipantRecord{
		"seat-1": {ID: "seat-1", CampaignID: "camp-1", AvatarSetID: "old-set", AvatarAssetID: "old-asset"},
	}
	characterStore := gametest.NewFakeCharacterStore()
	characterStore.Characters["camp-1"] = map[string]storage.CharacterRecord{
		"char-1": {ID: "char-1", CampaignID: "camp-1", ParticipantID: "seat-1", AvatarSetID: "old-set", AvatarAssetID: "old-asset"},
	}

	applyParticipantProfileSnapshot(
		context.Background(),
		domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
		projection.Applier{},
		participantStore,
		characterStore,
		&testclients.FakeSocialClient{Profile: &socialv1.UserProfile{
			UserId:        "user-1",
			AvatarSetId:   "avatar-set-1",
			AvatarAssetId: "avatar-asset-1",
		}},
		"camp-1",
		"seat-1",
		"user-1",
		"req-1",
		"inv-1",
		"seat-1",
		command.ActorTypeParticipant,
	)

	if len(domain.commands) != 1 {
		t.Fatalf("expected 1 command, got %d", len(domain.commands))
	}
	if domain.commands[0].Type != handler.CommandTypeParticipantUpdate {
		t.Fatalf("command type = %s, want %s", domain.commands[0].Type, handler.CommandTypeParticipantUpdate)
	}
}

func TestSyncControlledCharacterAvatars_SkipsWhenCharacterListFails(t *testing.T) {
	domain := &fakeDomainEngine{}

	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["camp-1"] = map[string]storage.ParticipantRecord{
		"seat-1": {
			ID:            "seat-1",
			CampaignID:    "camp-1",
			AvatarSetID:   "avatar-set-1",
			AvatarAssetID: "avatar-asset-1",
		},
	}
	characterStore := gametest.NewFakeCharacterStore()
	characterStore.ListErr = errors.New("boom")

	syncControlledCharacterAvatars(
		context.Background(),
		domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
		projection.Applier{},
		participantStore,
		characterStore,
		"camp-1",
		"seat-1",
		"req-1",
		"inv-1",
		"seat-1",
		command.ActorTypeParticipant,
	)

	if len(domain.commands) != 0 {
		t.Fatalf("expected no commands, got %d", len(domain.commands))
	}
}

func TestSyncControlledCharacterAvatars_SkipsWhenStoresMissing(t *testing.T) {
	domain := &fakeDomainEngine{}

	syncControlledCharacterAvatars(
		context.Background(),
		domainwrite.WritePath{Executor: domain, Runtime: testRuntime},
		projection.Applier{},
		nil,
		nil,
		"camp-1",
		"seat-1",
		"req-1",
		"inv-1",
		"seat-1",
		command.ActorTypeParticipant,
	)

	if len(domain.commands) != 0 {
		t.Fatalf("expected no commands, got %d", len(domain.commands))
	}
}
