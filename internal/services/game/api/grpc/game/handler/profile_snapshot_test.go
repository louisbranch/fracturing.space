package handler

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	socialv1 "github.com/louisbranch/fracturing.space/api/gen/go/social/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/gametest"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	participantdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeHandlerExecutor struct {
	execute  func(context.Context, command.Command) (engine.Result, error)
	commands []command.Command
}

func (f *fakeHandlerExecutor) Execute(ctx context.Context, cmd command.Command) (engine.Result, error) {
	f.commands = append(f.commands, cmd)
	if f.execute != nil {
		return f.execute(ctx, cmd)
	}
	return engine.Result{}, nil
}

type fakeHandlerSocialClient struct {
	profile *socialv1.UserProfile
	err     error
	lastReq *socialv1.GetUserProfileRequest
}

func (f *fakeHandlerSocialClient) GetUserProfile(_ context.Context, req *socialv1.GetUserProfileRequest, _ ...grpc.CallOption) (*socialv1.GetUserProfileResponse, error) {
	f.lastReq = req
	if f.err != nil {
		return nil, f.err
	}
	if f.profile == nil {
		return &socialv1.GetUserProfileResponse{}, nil
	}
	return &socialv1.GetUserProfileResponse{UserProfile: f.profile}, nil
}

func TestLoadSocialProfileSnapshot(t *testing.T) {
	t.Parallel()

	t.Run("success trims fields and maps pronouns", func(t *testing.T) {
		t.Parallel()

		client := &fakeHandlerSocialClient{
			profile: &socialv1.UserProfile{
				Name:          " Aria ",
				Pronouns:      sharedpronouns.ToProto("she/her"),
				AvatarSetId:   " set-1 ",
				AvatarAssetId: " asset-1 ",
			},
		}

		snapshot := LoadSocialProfileSnapshot(context.Background(), client, " user-1 ")
		if client.lastReq.GetUserId() != "user-1" {
			t.Fatalf("user_id = %q, want user-1", client.lastReq.GetUserId())
		}
		if snapshot.Name != "Aria" || snapshot.Pronouns != "she/her" || snapshot.AvatarSetID != "set-1" || snapshot.AvatarAssetID != "asset-1" {
			t.Fatalf("snapshot = %#v", snapshot)
		}
	})

	t.Run("missing user or failed lookup returns empty snapshot", func(t *testing.T) {
		t.Parallel()

		if snapshot := LoadSocialProfileSnapshot(context.Background(), nil, "user-1"); snapshot != (SocialProfileSnapshot{}) {
			t.Fatalf("nil client snapshot = %#v", snapshot)
		}
		client := &fakeHandlerSocialClient{err: errors.New("boom")}
		if snapshot := LoadSocialProfileSnapshot(context.Background(), client, ""); snapshot != (SocialProfileSnapshot{}) {
			t.Fatalf("empty user snapshot = %#v", snapshot)
		}
		if snapshot := LoadSocialProfileSnapshot(context.Background(), client, "user-1"); snapshot != (SocialProfileSnapshot{}) {
			t.Fatalf("error snapshot = %#v", snapshot)
		}
	})
}

func TestApplyParticipantProfileSnapshot_AppliesProfileAvatarAndCharacterSync(t *testing.T) {
	t.Parallel()

	executor := &fakeHandlerExecutor{}
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["camp-1"] = map[string]storage.ParticipantRecord{
		"part-1": {
			ID:            "part-1",
			CampaignID:    "camp-1",
			AvatarSetID:   "set-1",
			AvatarAssetID: "asset-1",
		},
	}
	characterStore := gametest.NewFakeCharacterStore()
	characterStore.Characters["camp-1"] = map[string]storage.CharacterRecord{
		"char-stale": {
			ID:                 "char-stale",
			CampaignID:         "camp-1",
			OwnerParticipantID: "part-1",
			AvatarSetID:        "old-set",
			AvatarAssetID:      "old-asset",
		},
		"char-same": {
			ID:                 "char-same",
			CampaignID:         "camp-1",
			OwnerParticipantID: "part-1",
			AvatarSetID:        "set-1",
			AvatarAssetID:      "asset-1",
		},
	}

	ApplyParticipantProfileSnapshot(
		context.Background(),
		domainwrite.WritePath{Executor: executor, Runtime: domainwrite.NewRuntime()},
		projection.Applier{},
		participantStore,
		characterStore,
		&fakeHandlerSocialClient{
			profile: &socialv1.UserProfile{
				UserId:        "user-1",
				Name:          "Ariadne",
				Pronouns:      sharedpronouns.ToProto("she/her"),
				AvatarSetId:   "set-1",
				AvatarAssetId: "asset-1",
			},
		},
		"camp-1",
		"part-1",
		"user-1",
		"req-1",
		"inv-1",
		"actor-1",
		command.ActorTypeParticipant,
	)

	if len(executor.commands) != 3 {
		t.Fatalf("len(commands) = %d, want 3", len(executor.commands))
	}
	if executor.commands[0].Type != CommandTypeParticipantUpdate || executor.commands[1].Type != CommandTypeParticipantUpdate || executor.commands[2].Type != CommandTypeCharacterUpdate {
		t.Fatalf("command types = %#v", []command.Type{executor.commands[0].Type, executor.commands[1].Type, executor.commands[2].Type})
	}

	var participantProfile participantdomain.UpdatePayload
	decodeHandlerPayload(t, executor.commands[0].PayloadJSON, &participantProfile)
	if participantProfile.Fields["name"] != "Ariadne" || participantProfile.Fields["pronouns"] != "she/her" {
		t.Fatalf("participant profile payload = %#v", participantProfile.Fields)
	}

	var participantAvatar participantdomain.UpdatePayload
	decodeHandlerPayload(t, executor.commands[1].PayloadJSON, &participantAvatar)
	if participantAvatar.Fields["avatar_set_id"] != "set-1" || participantAvatar.Fields["avatar_asset_id"] != "asset-1" {
		t.Fatalf("participant avatar payload = %#v", participantAvatar.Fields)
	}

	var characterAvatar character.UpdatePayload
	decodeHandlerPayload(t, executor.commands[2].PayloadJSON, &characterAvatar)
	if executor.commands[2].EntityID != "char-stale" {
		t.Fatalf("character entity_id = %q, want char-stale", executor.commands[2].EntityID)
	}
	if characterAvatar.Fields["avatar_set_id"] != "set-1" || characterAvatar.Fields["avatar_asset_id"] != "asset-1" {
		t.Fatalf("character avatar payload = %#v", characterAvatar.Fields)
	}
}

func TestApplyParticipantProfileSnapshot_SkipsCharacterSyncWhenAvatarWriteFails(t *testing.T) {
	t.Parallel()

	executor := &fakeHandlerExecutor{
		execute: func(_ context.Context, cmd command.Command) (engine.Result, error) {
			if len(cmd.PayloadJSON) == 0 {
				t.Fatal("payload_json = empty")
			}
			if cmd.Type == CommandTypeParticipantUpdate {
				var payload participantdomain.UpdatePayload
				if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
					t.Fatalf("decode participant payload: %v", err)
				}
				if _, ok := payload.Fields["avatar_set_id"]; ok {
					return engine.Result{}, status.Error(codes.Internal, "avatar manifest missing")
				}
			}
			return engine.Result{}, nil
		},
	}
	participantStore := gametest.NewFakeParticipantStore()
	participantStore.Participants["camp-1"] = map[string]storage.ParticipantRecord{
		"part-1": {ID: "part-1", CampaignID: "camp-1", AvatarSetID: "set-1", AvatarAssetID: "asset-1"},
	}
	characterStore := gametest.NewFakeCharacterStore()
	characterStore.Characters["camp-1"] = map[string]storage.CharacterRecord{
		"char-1": {ID: "char-1", CampaignID: "camp-1", OwnerParticipantID: "part-1", AvatarSetID: "old", AvatarAssetID: "old"},
	}

	ApplyParticipantProfileSnapshot(
		context.Background(),
		domainwrite.WritePath{Executor: executor, Runtime: domainwrite.NewRuntime()},
		projection.Applier{},
		participantStore,
		characterStore,
		&fakeHandlerSocialClient{
			profile: &socialv1.UserProfile{
				Name:          "Ariadne",
				AvatarSetId:   "set-1",
				AvatarAssetId: "asset-1",
			},
		},
		"camp-1",
		"part-1",
		"user-1",
		"req-1",
		"inv-1",
		"actor-1",
		command.ActorTypeParticipant,
	)

	if len(executor.commands) != 2 {
		t.Fatalf("len(commands) = %d, want 2", len(executor.commands))
	}
	if executor.commands[0].Type != CommandTypeParticipantUpdate || executor.commands[1].Type != CommandTypeParticipantUpdate {
		t.Fatalf("command types = %#v", []command.Type{executor.commands[0].Type, executor.commands[1].Type})
	}
}

func TestSyncOwnedCharacterAvatars_SkipsWhenStoresAreUnavailable(t *testing.T) {
	t.Parallel()

	executor := &fakeHandlerExecutor{}
	SyncOwnedCharacterAvatars(
		context.Background(),
		domainwrite.WritePath{Executor: executor, Runtime: domainwrite.NewRuntime()},
		projection.Applier{},
		nil,
		nil,
		"camp-1",
		"part-1",
		"req-1",
		"inv-1",
		"actor-1",
		command.ActorTypeParticipant,
	)
	if len(executor.commands) != 0 {
		t.Fatalf("len(commands) = %d, want 0", len(executor.commands))
	}
}

func decodeHandlerPayload(t *testing.T, payload []byte, target any) {
	t.Helper()

	if err := json.Unmarshal(payload, target); err != nil {
		t.Fatalf("json.Unmarshal(%s) error = %v", string(payload), err)
	}
}
