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
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type adversaryAttackResolvedPayload struct {
	AdversaryID string `json:"adversary_id"`
	RollSeq     uint64 `json:"roll_seq"`
	Success     bool   `json:"success"`
}

func TestDaggerheartAdversaryAttackFlow(t *testing.T) {
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
		Name:               "Adversary Attack Flow Campaign",
		System:             commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:             gamev1.GmMode_HUMAN,
		ThemePrompt:        "adversary attack flow",
		CreatorDisplayName: "GM",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if createCampaign.GetCampaign() == nil {
		t.Fatal("expected campaign")
	}
	campaignID := createCampaign.GetCampaign().GetId()

	target := createCharacter(t, ctx, characterClient, campaignID, "Adversary Target")
	patchDaggerheartProfile(t, ctx, characterClient, campaignID, target)

	startSession, err := sessionClient.StartSession(ctx, &gamev1.StartSessionRequest{
		CampaignId: campaignID,
		Name:       "Adversary Attack Session",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if startSession.GetSession() == nil {
		t.Fatal("expected session")
	}
	sessionID := startSession.GetSession().GetId()

	createAdversary, err := daggerheartClient.CreateAdversary(ctx, &daggerheartv1.DaggerheartCreateAdversaryRequest{
		CampaignId: campaignID,
		Name:       "Adversary One",
		Kind:       "elite",
		SessionId:  wrapperspb.String(sessionID),
	})
	if err != nil {
		t.Fatalf("create adversary: %v", err)
	}
	if createAdversary.GetAdversary() == nil {
		t.Fatal("expected adversary")
	}

	attackSeed := uint64(21)
	damageSeed := uint64(42)
	result, err := daggerheartClient.SessionAdversaryAttackFlow(ctx, &daggerheartv1.SessionAdversaryAttackFlowRequest{
		CampaignId:     campaignID,
		SessionId:      sessionID,
		AdversaryId:    createAdversary.GetAdversary().GetId(),
		TargetId:       target,
		Difficulty:     1,
		AttackModifier: 0,
		DamageDice:     []*daggerheartv1.DiceSpec{{Sides: 6, Count: 1}},
		Damage: &daggerheartv1.DaggerheartAttackDamageSpec{
			DamageType: daggerheartv1.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
			Source:     "adversary attack",
		},
		RequireDamageRoll: true,
		AttackRng: &commonv1.RngRequest{
			Seed:     &attackSeed,
			RollMode: commonv1.RollMode_REPLAY,
		},
		DamageRng: &commonv1.RngRequest{
			Seed:     &damageSeed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		t.Fatalf("session adversary attack flow: %v", err)
	}
	if result.GetAttackRoll() == nil || result.GetAttackOutcome() == nil {
		t.Fatal("expected attack roll and outcome")
	}
	if result.GetAttackOutcome().GetResult() == nil || !result.GetAttackOutcome().GetResult().GetSuccess() {
		t.Fatal("expected successful adversary attack")
	}
	if result.GetDamageRoll() == nil || result.GetDamageApplied() == nil {
		t.Fatal("expected damage roll and damage applied results")
	}
	if result.GetDamageRoll().GetRollSeq() == 0 {
		t.Fatal("expected damage roll seq")
	}

	if err := findAdversaryAttackResolved(ctx, eventClient, campaignID, sessionID, result.GetAttackRoll().GetRollSeq()); err != nil {
		t.Fatalf("find adversary attack resolved: %v", err)
	}
	if _, err := findDamageApplied(ctx, eventClient, campaignID, sessionID, result.GetDamageRoll().GetRollSeq()); err != nil {
		t.Fatalf("find damage applied: %v", err)
	}
}

func findAdversaryAttackResolved(ctx context.Context, client gamev1.EventServiceClient, campaignID, sessionID string, rollSeq uint64) error {
	response, err := client.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   200,
		OrderBy:    "seq desc",
		Filter:     "session_id = \"" + sessionID + "\" AND type = \"action.adversary_attack_resolved\"",
	})
	if err != nil {
		return err
	}
	for _, evt := range response.GetEvents() {
		var payload adversaryAttackResolvedPayload
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return err
		}
		if payload.RollSeq == rollSeq {
			return nil
		}
	}
	return fmt.Errorf("adversary attack resolved event not found")
}
