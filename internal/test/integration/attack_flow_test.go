//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type damageAppliedPayload struct {
	RollSeq *uint64 `json:"roll_seq,omitempty"`
}

func TestDaggerheartAttackFlow(t *testing.T) {
	grpcAddr, _, stopServer := startGRPCServer(t)
	defer stopServer()

	conn, err := grpc.NewClient(
		grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial gRPC: %v", err)
	}
	defer conn.Close()

	campaignClient := gamev1.NewCampaignServiceClient(conn)
	characterClient := gamev1.NewCharacterServiceClient(conn)
	sessionClient := gamev1.NewSessionServiceClient(conn)
	eventClient := gamev1.NewEventServiceClient(conn)
	daggerheartClient := daggerheartv1.NewDaggerheartServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	createCampaign, err := campaignClient.CreateCampaign(ctx, &gamev1.CreateCampaignRequest{
		Name:               "Attack Flow Campaign",
		System:             commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:             gamev1.GmMode_HUMAN,
		ThemePrompt:        "attack flow",
		CreatorDisplayName: "Attack GM",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if createCampaign.GetCampaign() == nil {
		t.Fatal("expected campaign")
	}
	campaignID := createCampaign.GetCampaign().GetId()

	attacker := createCharacter(t, ctx, characterClient, campaignID, "Attack Hero")
	target := createCharacter(t, ctx, characterClient, campaignID, "Attack Target")

	patchDaggerheartProfile(t, ctx, characterClient, campaignID, attacker)
	patchDaggerheartProfile(t, ctx, characterClient, campaignID, target)

	startSession, err := sessionClient.StartSession(ctx, &gamev1.StartSessionRequest{
		CampaignId: campaignID,
		Name:       "Attack Session",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if startSession.GetSession() == nil {
		t.Fatal("expected session")
	}
	sessionID := startSession.GetSession().GetId()

	difficulty := 8
	seed := findReplaySeedForSuccess(t, difficulty)
	damageSeed := uint64(42)

	result, err := daggerheartClient.SessionAttackFlow(ctx, &daggerheartv1.SessionAttackFlowRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CharacterId: attacker,
		Trait:       "strength",
		Difficulty:  int32(difficulty),
		TargetId:    target,
		DamageDice:  []*daggerheartv1.DiceSpec{{Sides: 6, Count: 1}},
		Damage: &daggerheartv1.DaggerheartAttackDamageSpec{
			DamageType:         daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
			Source:             "attack",
			SourceCharacterIds: []string{attacker},
		},
		RequireDamageRoll: true,
		ActionRng: &commonv1.RngRequest{
			Seed:     &seed,
			RollMode: commonv1.RollMode_REPLAY,
		},
		DamageRng: &commonv1.RngRequest{
			Seed:     &damageSeed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		t.Fatalf("session attack flow: %v", err)
	}
	if result.GetActionRoll() == nil || result.GetAttackOutcome() == nil {
		t.Fatal("expected roll and attack outcome results")
	}
	if result.GetDamageRoll() == nil || result.GetDamageApplied() == nil {
		t.Fatal("expected damage roll and damage applied results")
	}
	if result.GetDamageRoll().GetRollSeq() == 0 {
		t.Fatal("expected damage roll seq")
	}

	applied, err := findDamageApplied(ctx, eventClient, campaignID, sessionID, result.GetDamageRoll().GetRollSeq())
	if err != nil {
		t.Fatalf("find damage applied: %v", err)
	}
	if applied.RollSeq == nil || *applied.RollSeq != result.GetDamageRoll().GetRollSeq() {
		t.Fatal("expected damage applied roll_seq to match damage roll")
	}
}

func createCharacter(t *testing.T, ctx context.Context, client gamev1.CharacterServiceClient, campaignID, name string) string {
	t.Helper()
	response, err := client.CreateCharacter(ctx, &gamev1.CreateCharacterRequest{
		CampaignId: campaignID,
		Name:       name,
		Kind:       gamev1.CharacterKind_PC,
	})
	if err != nil {
		t.Fatalf("create character: %v", err)
	}
	if response.GetCharacter() == nil {
		t.Fatal("expected character")
	}
	return response.GetCharacter().GetId()
}

func patchDaggerheartProfile(t *testing.T, ctx context.Context, client gamev1.CharacterServiceClient, campaignID, characterID string) {
	t.Helper()
	_, err := client.PatchCharacterProfile(ctx, &gamev1.PatchCharacterProfileRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
		SystemProfilePatch: &gamev1.PatchCharacterProfileRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartProfile{
				Level:           1,
				HpMax:           6,
				StressMax:       wrapperspb.Int32(6),
				Evasion:         wrapperspb.Int32(10),
				MajorThreshold:  wrapperspb.Int32(3),
				SevereThreshold: wrapperspb.Int32(6),
			},
		},
	})
	if err != nil {
		t.Fatalf("patch character profile: %v", err)
	}
}

func findReplaySeedForSuccess(t *testing.T, difficulty int) uint64 {
	t.Helper()
	for seed := uint64(1); seed < 50000; seed++ {
		difficultyValue := difficulty
		result, err := daggerheartdomain.RollAction(daggerheartdomain.ActionRequest{
			Modifier:   0,
			Difficulty: &difficultyValue,
			Seed:       int64(seed),
		})
		if err != nil {
			continue
		}
		if result.MeetsDifficulty {
			return seed
		}
	}
	t.Fatal("no replay seed found for success roll")
	return 0
}

func findDamageApplied(ctx context.Context, client gamev1.EventServiceClient, campaignID, sessionID string, rollSeq uint64) (damageAppliedPayload, error) {
	response, err := client.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   200,
		OrderBy:    "seq desc",
		Filter:     "session_id = \"" + sessionID + "\" AND type = \"action.damage_applied\"",
	})
	if err != nil {
		return damageAppliedPayload{}, err
	}
	for _, evt := range response.GetEvents() {
		var payload damageAppliedPayload
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return damageAppliedPayload{}, err
		}
		if payload.RollSeq != nil && *payload.RollSeq == rollSeq {
			return payload, nil
		}
	}
	return damageAppliedPayload{}, fmt.Errorf("damage applied event not found")
}
