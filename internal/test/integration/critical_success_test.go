//go:build integration

package integration

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

func TestDaggerheartActionRollCriticalEffects(t *testing.T) {
	grpcAddr, authAddr, stopServer := startGRPCServer(t)
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
	snapshotClient := gamev1.NewSnapshotServiceClient(conn)
	daggerheartClient := daggerheartv1.NewDaggerheartServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	userID := createAuthUser(t, authAddr, "critical-gm")
	ctxWithUser := withUserID(ctx, userID)

	createCampaign, err := campaignClient.CreateCampaign(ctxWithUser, &gamev1.CreateCampaignRequest{
		Name:               "Critical Roll Campaign",
		System:             commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:             gamev1.GmMode_HUMAN,
		ThemePrompt:        "critical roll",
		CreatorDisplayName: "Critical GM",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if createCampaign.GetCampaign() == nil {
		t.Fatal("expected campaign")
	}
	campaignID := createCampaign.GetCampaign().GetId()

	characterID := createCharacter(t, ctx, characterClient, campaignID, "Critical Hero")
	patchDaggerheartProfile(t, ctx, characterClient, campaignID, characterID)

	_, err = snapshotClient.PatchCharacterState(ctx, &gamev1.PatchCharacterStateRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
		SystemStatePatch: &gamev1.PatchCharacterStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCharacterState{
				Hp:     6,
				Hope:   2,
				Stress: 3,
			},
		},
	})
	if err != nil {
		t.Fatalf("patch character state: %v", err)
	}

	startSession, err := sessionClient.StartSession(ctx, &gamev1.StartSessionRequest{
		CampaignId: campaignID,
		Name:       "Critical Session",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if startSession.GetSession() == nil {
		t.Fatal("expected session")
	}
	sessionID := startSession.GetSession().GetId()

	difficulty := 8
	seed := findReplaySeedForCritical(t, difficulty)

	rollResp, err := daggerheartClient.SessionActionRoll(ctx, &daggerheartv1.SessionActionRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CharacterId: characterID,
		Trait:       "presence",
		RollKind:    daggerheartv1.RollKind_ROLL_KIND_ACTION,
		Difficulty:  int32(difficulty),
		Rng: &commonv1.RngRequest{
			Seed:     &seed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		t.Fatalf("session action roll: %v", err)
	}
	if !rollResp.GetCrit() {
		t.Fatal("expected critical action roll")
	}

	outcomeCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(
		grpcmeta.CampaignIDHeader, campaignID,
		grpcmeta.SessionIDHeader, sessionID,
	))
	_, err = daggerheartClient.ApplyRollOutcome(outcomeCtx, &daggerheartv1.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		t.Fatalf("apply roll outcome: %v", err)
	}

	state := fetchCharacterState(t, ctx, snapshotClient, campaignID, characterID)
	if state.GetHope() != 3 {
		t.Fatalf("expected hope 3 after crit, got %d", state.GetHope())
	}
	if state.GetStress() != 2 {
		t.Fatalf("expected stress 2 after crit, got %d", state.GetStress())
	}
}

func TestDaggerheartAttackFlowCriticalDamageBonus(t *testing.T) {
	grpcAddr, authAddr, stopServer := startGRPCServer(t)
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
	daggerheartClient := daggerheartv1.NewDaggerheartServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	userID := createAuthUser(t, authAddr, "critical-attack-gm")
	ctxWithUser := withUserID(ctx, userID)

	createCampaign, err := campaignClient.CreateCampaign(ctxWithUser, &gamev1.CreateCampaignRequest{
		Name:               "Critical Attack Campaign",
		System:             commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:             gamev1.GmMode_HUMAN,
		ThemePrompt:        "critical attack",
		CreatorDisplayName: "Critical Attack GM",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if createCampaign.GetCampaign() == nil {
		t.Fatal("expected campaign")
	}
	campaignID := createCampaign.GetCampaign().GetId()

	attacker := createCharacter(t, ctx, characterClient, campaignID, "Critical Attacker")
	target := createCharacter(t, ctx, characterClient, campaignID, "Critical Target")

	patchDaggerheartProfile(t, ctx, characterClient, campaignID, attacker)
	patchDaggerheartProfile(t, ctx, characterClient, campaignID, target)

	startSession, err := sessionClient.StartSession(ctx, &gamev1.StartSessionRequest{
		CampaignId: campaignID,
		Name:       "Critical Attack Session",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if startSession.GetSession() == nil {
		t.Fatal("expected session")
	}
	sessionID := startSession.GetSession().GetId()

	difficulty := 8
	seed := findReplaySeedForCritical(t, difficulty)
	damageSeed := uint64(99)

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
	if result.GetAttackOutcome() == nil || result.GetAttackOutcome().GetResult() == nil {
		t.Fatal("expected attack outcome result")
	}
	if !result.GetAttackOutcome().GetResult().GetCrit() {
		t.Fatal("expected critical attack outcome")
	}
	if result.GetDamageRoll() == nil {
		t.Fatal("expected damage roll")
	}
	if result.GetDamageRoll().GetCriticalBonus() != 6 {
		t.Fatalf("expected critical bonus 6, got %d", result.GetDamageRoll().GetCriticalBonus())
	}
}

func findReplaySeedForCritical(t *testing.T, difficulty int) uint64 {
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
		if result.IsCrit {
			return seed
		}
	}
	t.Fatal("no replay seed found for critical roll")
	return 0
}

func fetchCharacterState(t *testing.T, ctx context.Context, snapshotClient gamev1.SnapshotServiceClient, campaignID, characterID string) *daggerheartv1.DaggerheartCharacterState {
	t.Helper()
	snapshot, err := snapshotClient.GetSnapshot(ctx, &gamev1.GetSnapshotRequest{CampaignId: campaignID})
	if err != nil {
		t.Fatalf("get snapshot: %v", err)
	}
	if snapshot.GetSnapshot() == nil {
		t.Fatal("expected snapshot")
	}
	for _, state := range snapshot.GetSnapshot().GetCharacterStates() {
		if state.GetCharacterId() == characterID {
			if state.GetDaggerheart() == nil {
				t.Fatal("expected daggerheart state")
			}
			return state.GetDaggerheart()
		}
	}
	t.Fatalf("character state not found: %s", characterID)
	return nil
}
