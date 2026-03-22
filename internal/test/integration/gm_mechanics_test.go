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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type gmFearChangedPayload struct {
	After  int    `json:"after"`
	Reason string `json:"reason"`
}

type countdownCreatedPayload struct {
	CountdownID string `json:"countdown_id"`
}

type countdownUpdatedPayload struct {
	CountdownID     string `json:"countdown_id"`
	BeforeRemaining int    `json:"before_remaining"`
	AfterRemaining  int    `json:"after_remaining"`
	AdvancedBy      int    `json:"advanced_by"`
}

type countdownDeletedPayload struct {
	CountdownID string `json:"countdown_id"`
}

type countdownTriggerResolvedPayload struct {
	CountdownID          string `json:"countdown_id"`
	StartingValueBefore  int    `json:"starting_value_before"`
	StartingValueAfter   int    `json:"starting_value_after"`
	RemainingValueBefore int    `json:"remaining_value_before"`
	RemainingValueAfter  int    `json:"remaining_value_after"`
	StatusBefore         string `json:"status_before"`
	StatusAfter          string `json:"status_after"`
}

func TestDaggerheartGmMoveSpendFear(t *testing.T) {
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
	participantClient := gamev1.NewParticipantServiceClient(conn)
	sessionClient := gamev1.NewSessionServiceClient(conn)
	sceneClient := gamev1.NewSceneServiceClient(conn)
	snapshotClient := gamev1.NewSnapshotServiceClient(conn)
	eventClient := gamev1.NewEventServiceClient(conn)
	daggerheartClient := daggerheartv1.NewDaggerheartServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()
	userID := createAuthUser(t, authAddr, "gm-user")
	ctx = withUserID(ctx, userID)

	createCampaign, err := campaignClient.CreateCampaign(ctx, &gamev1.CreateCampaignRequest{
		Name:        "GM Move Campaign",
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      gamev1.GmMode_HUMAN,
		ThemePrompt: "gm move",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if createCampaign.GetCampaign() == nil {
		t.Fatal("expected campaign")
	}
	campaignID := createCampaign.GetCampaign().GetId()
	ownerParticipantID := createCampaign.GetOwnerParticipant().GetId()
	sceneAnchor := createCharacter(t, ctx, characterClient, campaignID, "GM Scene Anchor")
	patchDaggerheartProfile(t, ctx, characterClient, campaignID, sceneAnchor)
	ensureSessionStartReadiness(t, ctx, participantClient, characterClient, campaignID, ownerParticipantID, sceneAnchor)

	startSession, err := sessionClient.StartSession(ctx, &gamev1.StartSessionRequest{
		CampaignId: campaignID,
		Name:       "GM Session",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if startSession.GetSession() == nil {
		t.Fatal("expected session")
	}
	sessionID := startSession.GetSession().GetId()
	createScene, err := sceneClient.CreateScene(ctx, &gamev1.CreateSceneRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
		Name:       "GM Scene",
		CharacterIds: []string{
			sceneAnchor,
		},
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	sceneID := createScene.GetSceneId()
	if sceneID == "" {
		t.Fatal("expected scene")
	}

	createAdversary, err := daggerheartClient.CreateAdversary(ctx, &daggerheartv1.DaggerheartCreateAdversaryRequest{
		CampaignId:       campaignID,
		SessionId:        sessionID,
		SceneId:          sceneID,
		AdversaryEntryId: "adversary.integration-foe",
	})
	if err != nil {
		t.Fatalf("create adversary: %v", err)
	}
	if createAdversary.GetAdversary() == nil {
		t.Fatal("expected adversary")
	}

	_, err = snapshotClient.UpdateSnapshotState(ctx, &gamev1.UpdateSnapshotStateRequest{
		CampaignId: campaignID,
		SystemSnapshotUpdate: &gamev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{
				GmFear:                3,
				ConsecutiveShortRests: 0,
			},
		},
	})
	if err != nil {
		t.Fatalf("update snapshot: %v", err)
	}

	moveResp, err := daggerheartClient.ApplyGmMove(ctx, &daggerheartv1.DaggerheartApplyGmMoveRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
		FearSpent:  2,
		SpendTarget: &daggerheartv1.DaggerheartApplyGmMoveRequest_DirectMove{
			DirectMove: &daggerheartv1.DaggerheartDirectGmMoveTarget{
				Kind:  daggerheartv1.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_ADDITIONAL_MOVE,
				Shape: daggerheartv1.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SHIFT_ENVIRONMENT,
			},
		},
	})
	if err != nil {
		t.Fatalf("apply gm move: %v", err)
	}
	if moveResp.GetGmFearBefore() != 3 || moveResp.GetGmFearAfter() != 1 {
		t.Fatalf("gm fear = %d -> %d, want 3 -> 1", moveResp.GetGmFearBefore(), moveResp.GetGmFearAfter())
	}

	if err := findGMFearChanged(ctx, eventClient, campaignID, sessionID, 1); err != nil {
		t.Fatalf("find gm fear changed: %v", err)
	}
}

func TestDaggerheartCountdownLifecycle(t *testing.T) {
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
	participantClient := gamev1.NewParticipantServiceClient(conn)
	sessionClient := gamev1.NewSessionServiceClient(conn)
	sceneClient := gamev1.NewSceneServiceClient(conn)
	eventClient := gamev1.NewEventServiceClient(conn)
	daggerheartClient := daggerheartv1.NewDaggerheartServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()
	userID := createAuthUser(t, authAddr, "countdown-gm")
	ctx = withUserID(ctx, userID)

	createCampaign, err := campaignClient.CreateCampaign(ctx, &gamev1.CreateCampaignRequest{
		Name:        "Countdown Campaign",
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      gamev1.GmMode_HUMAN,
		ThemePrompt: "countdown",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if createCampaign.GetCampaign() == nil {
		t.Fatal("expected campaign")
	}
	campaignID := createCampaign.GetCampaign().GetId()
	ownerParticipantID := createCampaign.GetOwnerParticipant().GetId()
	sceneAnchor := createCharacter(t, ctx, characterClient, campaignID, "Adversary Scene Anchor")
	patchDaggerheartProfile(t, ctx, characterClient, campaignID, sceneAnchor)
	ensureSessionStartReadiness(t, ctx, participantClient, characterClient, campaignID, ownerParticipantID, sceneAnchor)

	startSession, err := sessionClient.StartSession(ctx, &gamev1.StartSessionRequest{
		CampaignId: campaignID,
		Name:       "Countdown Session",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if startSession.GetSession() == nil {
		t.Fatal("expected session")
	}
	sessionID := startSession.GetSession().GetId()
	createScene, err := sceneClient.CreateScene(ctx, &gamev1.CreateSceneRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
		Name:       "Countdown Scene",
		CharacterIds: []string{
			sceneAnchor,
		},
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	sceneID := createScene.GetSceneId()
	if sceneID == "" {
		t.Fatal("expected scene")
	}

	createResp, err := daggerheartClient.CreateSceneCountdown(ctx, &daggerheartv1.DaggerheartCreateSceneCountdownRequest{
		CampaignId:        campaignID,
		SessionId:         sessionID,
		SceneId:           sceneID,
		Name:              "Wagon Timer",
		Tone:              daggerheartv1.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS,
		AdvancementPolicy: daggerheartv1.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL,
		StartingValue:     &daggerheartv1.DaggerheartCreateSceneCountdownRequest_FixedStartingValue{FixedStartingValue: 4},
		LoopBehavior:      daggerheartv1.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE,
	})
	if err != nil {
		t.Fatalf("create countdown: %v", err)
	}
	if createResp.GetCountdown() == nil {
		t.Fatal("expected countdown")
	}
	countdownID := createResp.GetCountdown().GetCountdownId()
	if countdownID == "" {
		t.Fatal("expected countdown id")
	}

	if err := findCountdownCreated(ctx, eventClient, campaignID, sessionID, countdownID); err != nil {
		t.Fatalf("find countdown created: %v", err)
	}

	updateResp, err := daggerheartClient.AdvanceSceneCountdown(ctx, &daggerheartv1.DaggerheartAdvanceSceneCountdownRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CountdownId: countdownID,
		Amount:      2,
		Reason:      "advance",
	})
	if err != nil {
		t.Fatalf("advance countdown: %v", err)
	}
	if updateResp.GetAdvance() == nil {
		t.Fatal("expected countdown advance")
	}
	if updateResp.GetAdvance().GetRemainingAfter() != 2 {
		t.Fatalf("countdown remaining after = %d, want 2", updateResp.GetAdvance().GetRemainingAfter())
	}
	if err := findCountdownUpdated(ctx, eventClient, campaignID, sessionID, countdownID, 4, 2); err != nil {
		t.Fatalf("find countdown updated: %v", err)
	}

	triggerResp, err := daggerheartClient.AdvanceSceneCountdown(ctx, &daggerheartv1.DaggerheartAdvanceSceneCountdownRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CountdownId: countdownID,
		Amount:      2,
		Reason:      "collapse imminent",
	})
	if err != nil {
		t.Fatalf("advance countdown to trigger: %v", err)
	}
	if triggerResp.GetAdvance() == nil || triggerResp.GetCountdown() == nil {
		t.Fatal("expected countdown advance and countdown")
	}
	if triggerResp.GetAdvance().GetRemainingBefore() != 2 || triggerResp.GetAdvance().GetRemainingAfter() != 0 {
		t.Fatalf("trigger advance remaining = %d -> %d, want 2 -> 0", triggerResp.GetAdvance().GetRemainingBefore(), triggerResp.GetAdvance().GetRemainingAfter())
	}
	if !triggerResp.GetAdvance().GetTriggered() {
		t.Fatalf("expected countdown trigger, got %+v", triggerResp.GetAdvance())
	}
	if triggerResp.GetCountdown().GetStatus() != daggerheartv1.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_TRIGGER_PENDING {
		t.Fatalf("countdown status = %s, want trigger_pending", triggerResp.GetCountdown().GetStatus())
	}
	if err := findCountdownUpdated(ctx, eventClient, campaignID, sessionID, countdownID, 2, 0); err != nil {
		t.Fatalf("find countdown triggered advance: %v", err)
	}

	resolveResp, err := daggerheartClient.ResolveSceneCountdownTrigger(ctx, &daggerheartv1.DaggerheartResolveSceneCountdownTriggerRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CountdownId: countdownID,
		Reason:      "wagon crashes through the gate",
	})
	if err != nil {
		t.Fatalf("resolve scene countdown trigger: %v", err)
	}
	if resolveResp.GetCountdown() == nil {
		t.Fatal("expected resolved countdown")
	}
	if resolveResp.GetCountdown().GetStatus() != daggerheartv1.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE {
		t.Fatalf("resolved countdown status = %s, want active", resolveResp.GetCountdown().GetStatus())
	}
	if resolveResp.GetCountdown().GetRemainingValue() != 0 {
		t.Fatalf("resolved countdown remaining = %d, want 0", resolveResp.GetCountdown().GetRemainingValue())
	}
	if err := findCountdownTriggerResolved(ctx, eventClient, campaignID, sessionID, countdownID, 4, 4, 0, 0); err != nil {
		t.Fatalf("find countdown trigger resolved: %v", err)
	}

	getResp, err := daggerheartClient.GetSceneCountdown(ctx, &daggerheartv1.DaggerheartGetSceneCountdownRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CountdownId: countdownID,
	})
	if err != nil {
		t.Fatalf("get scene countdown: %v", err)
	}
	if getResp.GetCountdown() == nil {
		t.Fatal("expected fetched countdown")
	}
	if getResp.GetCountdown().GetStatus() != daggerheartv1.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE {
		t.Fatalf("fetched countdown status = %s, want active", getResp.GetCountdown().GetStatus())
	}
	if getResp.GetCountdown().GetRemainingValue() != 0 {
		t.Fatalf("fetched countdown remaining = %d, want 0", getResp.GetCountdown().GetRemainingValue())
	}

	_, err = daggerheartClient.DeleteSceneCountdown(ctx, &daggerheartv1.DaggerheartDeleteSceneCountdownRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		SceneId:     sceneID,
		CountdownId: countdownID,
		Reason:      "resolved",
	})
	if err != nil {
		t.Fatalf("delete countdown: %v", err)
	}
	if err := findCountdownDeleted(ctx, eventClient, campaignID, sessionID, countdownID); err != nil {
		t.Fatalf("find countdown deleted: %v", err)
	}
}

func TestDaggerheartCampaignCountdownLifecycle(t *testing.T) {
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
	eventClient := gamev1.NewEventServiceClient(conn)
	daggerheartClient := daggerheartv1.NewDaggerheartServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()
	userID := createAuthUser(t, authAddr, "campaign-countdown-gm")
	ctx = withUserID(ctx, userID)

	createCampaign, err := campaignClient.CreateCampaign(ctx, &gamev1.CreateCampaignRequest{
		Name:        "Campaign Countdown Lifecycle",
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      gamev1.GmMode_HUMAN,
		ThemePrompt: "campaign countdown",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if createCampaign.GetCampaign() == nil {
		t.Fatal("expected campaign")
	}
	campaignID := createCampaign.GetCampaign().GetId()

	createResp, err := daggerheartClient.CreateCampaignCountdown(ctx, &daggerheartv1.DaggerheartCreateCampaignCountdownRequest{
		CampaignId:        campaignID,
		Name:              "Rebuild the Causeway",
		Tone:              daggerheartv1.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS,
		AdvancementPolicy: daggerheartv1.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL,
		StartingValue:     &daggerheartv1.DaggerheartCreateCampaignCountdownRequest_FixedStartingValue{FixedStartingValue: 2},
		LoopBehavior:      daggerheartv1.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_INCREASE_START,
	})
	if err != nil {
		t.Fatalf("create campaign countdown: %v", err)
	}
	if createResp.GetCountdown() == nil {
		t.Fatal("expected campaign countdown")
	}
	countdownID := createResp.GetCountdown().GetCountdownId()
	if countdownID == "" {
		t.Fatal("expected countdown id")
	}
	if err := findCountdownCreated(ctx, eventClient, campaignID, "", countdownID); err != nil {
		t.Fatalf("find campaign countdown created: %v", err)
	}

	advanceResp, err := daggerheartClient.AdvanceCampaignCountdown(ctx, &daggerheartv1.DaggerheartAdvanceCampaignCountdownRequest{
		CampaignId:  campaignID,
		CountdownId: countdownID,
		Amount:      2,
		Reason:      "downtime effort",
	})
	if err != nil {
		t.Fatalf("advance campaign countdown: %v", err)
	}
	if advanceResp.GetAdvance() == nil || advanceResp.GetCountdown() == nil {
		t.Fatal("expected campaign countdown advance and countdown")
	}
	if !advanceResp.GetAdvance().GetTriggered() {
		t.Fatalf("expected triggered campaign countdown, got %+v", advanceResp.GetAdvance())
	}
	if advanceResp.GetCountdown().GetStatus() != daggerheartv1.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_TRIGGER_PENDING {
		t.Fatalf("campaign countdown status = %s, want trigger_pending", advanceResp.GetCountdown().GetStatus())
	}
	if err := findCountdownUpdated(ctx, eventClient, campaignID, "", countdownID, 2, 0); err != nil {
		t.Fatalf("find campaign countdown updated: %v", err)
	}

	resolveResp, err := daggerheartClient.ResolveCampaignCountdownTrigger(ctx, &daggerheartv1.DaggerheartResolveCampaignCountdownTriggerRequest{
		CampaignId:  campaignID,
		CountdownId: countdownID,
		Reason:      "milestone reached",
	})
	if err != nil {
		t.Fatalf("resolve campaign countdown trigger: %v", err)
	}
	if resolveResp.GetCountdown() == nil {
		t.Fatal("expected resolved campaign countdown")
	}
	if resolveResp.GetCountdown().GetStatus() != daggerheartv1.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE {
		t.Fatalf("resolved campaign countdown status = %s, want active", resolveResp.GetCountdown().GetStatus())
	}
	if resolveResp.GetCountdown().GetStartingValue() != 3 || resolveResp.GetCountdown().GetRemainingValue() != 3 {
		t.Fatalf("resolved campaign countdown values = %d/%d, want 3/3", resolveResp.GetCountdown().GetStartingValue(), resolveResp.GetCountdown().GetRemainingValue())
	}
	if err := findCountdownTriggerResolved(ctx, eventClient, campaignID, "", countdownID, 2, 3, 0, 3); err != nil {
		t.Fatalf("find campaign countdown trigger resolved: %v", err)
	}

	getResp, err := daggerheartClient.GetCampaignCountdown(ctx, &daggerheartv1.DaggerheartGetCampaignCountdownRequest{
		CampaignId:  campaignID,
		CountdownId: countdownID,
	})
	if err != nil {
		t.Fatalf("get campaign countdown: %v", err)
	}
	if getResp.GetCountdown() == nil {
		t.Fatal("expected fetched campaign countdown")
	}
	if getResp.GetCountdown().GetStartingValue() != 3 || getResp.GetCountdown().GetRemainingValue() != 3 {
		t.Fatalf("fetched campaign countdown values = %d/%d, want 3/3", getResp.GetCountdown().GetStartingValue(), getResp.GetCountdown().GetRemainingValue())
	}

	_, err = daggerheartClient.DeleteCampaignCountdown(ctx, &daggerheartv1.DaggerheartDeleteCampaignCountdownRequest{
		CampaignId:  campaignID,
		CountdownId: countdownID,
		Reason:      "cleanup",
	})
	if err != nil {
		t.Fatalf("delete campaign countdown: %v", err)
	}
	if err := findCountdownDeleted(ctx, eventClient, campaignID, "", countdownID); err != nil {
		t.Fatalf("find campaign countdown deleted: %v", err)
	}
}

func TestDaggerheartAdversaryAttackRoll(t *testing.T) {
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
	participantClient := gamev1.NewParticipantServiceClient(conn)
	sessionClient := gamev1.NewSessionServiceClient(conn)
	sceneClient := gamev1.NewSceneServiceClient(conn)
	eventClient := gamev1.NewEventServiceClient(conn)
	daggerheartClient := daggerheartv1.NewDaggerheartServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()
	userID := createAuthUser(t, authAddr, "adversary-gm")
	ctx = withUserID(ctx, userID)

	createCampaign, err := campaignClient.CreateCampaign(ctx, &gamev1.CreateCampaignRequest{
		Name:        "Adversary Roll Campaign",
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      gamev1.GmMode_HUMAN,
		ThemePrompt: "adversary",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if createCampaign.GetCampaign() == nil {
		t.Fatal("expected campaign")
	}
	campaignID := createCampaign.GetCampaign().GetId()
	ownerParticipantID := createCampaign.GetOwnerParticipant().GetId()
	sceneAnchor := createCharacter(t, ctx, characterClient, campaignID, "Adversary Scene Anchor")
	patchDaggerheartProfile(t, ctx, characterClient, campaignID, sceneAnchor)
	ensureSessionStartReadiness(t, ctx, participantClient, characterClient, campaignID, ownerParticipantID, sceneAnchor)

	startSession, err := sessionClient.StartSession(ctx, &gamev1.StartSessionRequest{
		CampaignId: campaignID,
		Name:       "Adversary Session",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if startSession.GetSession() == nil {
		t.Fatal("expected session")
	}
	sessionID := startSession.GetSession().GetId()
	createScene, err := sceneClient.CreateScene(ctx, &gamev1.CreateSceneRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
		Name:       "Adversary Scene",
		CharacterIds: []string{
			sceneAnchor,
		},
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	sceneID := createScene.GetSceneId()
	if sceneID == "" {
		t.Fatal("expected scene")
	}

	createAdversary, err := daggerheartClient.CreateAdversary(ctx, &daggerheartv1.DaggerheartCreateAdversaryRequest{
		CampaignId:       campaignID,
		SessionId:        sessionID,
		SceneId:          sceneID,
		AdversaryEntryId: "adversary.integration-foe",
	})
	if err != nil {
		t.Fatalf("create adversary: %v", err)
	}
	if createAdversary.GetAdversary() == nil {
		t.Fatal("expected adversary")
	}

	seed := uint64(42)
	rollResp, err := daggerheartClient.SessionAdversaryAttackRoll(ctx, &daggerheartv1.SessionAdversaryAttackRollRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		AdversaryId: createAdversary.GetAdversary().GetId(),
		Modifiers: []*daggerheartv1.ActionRollModifier{
			{Source: "attack_modifier", Value: 2},
		},
		Advantage: 1,
		Rng: &commonv1.RngRequest{
			Seed:     &seed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		t.Fatalf("adversary roll: %v", err)
	}
	if rollResp.GetRollSeq() == 0 {
		t.Fatal("expected roll seq")
	}
	if len(rollResp.GetRolls()) != 2 {
		t.Fatalf("expected two rolls with advantage")
	}

	if err := findAdversaryRollResolved(ctx, eventClient, campaignID, sessionID, rollResp.GetRollSeq()); err != nil {
		t.Fatalf("find adversary roll resolved: %v", err)
	}
}

func findGMFearChanged(ctx context.Context, client gamev1.EventServiceClient, campaignID, sessionID string, after int) error {
	response, err := client.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   200,
		OrderBy:    "seq desc",
		Filter:     "session_id = \"" + sessionID + "\" AND type = \"sys.daggerheart.gm_fear_changed\"",
	})
	if err != nil {
		return err
	}
	for _, evt := range response.GetEvents() {
		var payload gmFearChangedPayload
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return err
		}
		if payload.After == after {
			return nil
		}
	}
	return fmt.Errorf("gm fear changed event not found")
}

func findCountdownCreated(ctx context.Context, client gamev1.EventServiceClient, campaignID, sessionID, countdownID string) error {
	filter := `type = "sys.daggerheart.campaign_countdown_created"`
	if sessionID != "" {
		filter = `session_id = "` + sessionID + `" AND type = "sys.daggerheart.scene_countdown_created"`
	}
	response, err := client.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   200,
		OrderBy:    "seq desc",
		Filter:     filter,
	})
	if err != nil {
		return err
	}
	for _, evt := range response.GetEvents() {
		var payload countdownCreatedPayload
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return err
		}
		if payload.CountdownID == countdownID {
			return nil
		}
	}
	return fmt.Errorf("countdown created event not found")
}

func findCountdownUpdated(ctx context.Context, client gamev1.EventServiceClient, campaignID, sessionID, countdownID string, before, after int) error {
	filter := `type = "sys.daggerheart.campaign_countdown_advanced"`
	if sessionID != "" {
		filter = `session_id = "` + sessionID + `" AND type = "sys.daggerheart.scene_countdown_advanced"`
	}
	response, err := client.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   200,
		OrderBy:    "seq desc",
		Filter:     filter,
	})
	if err != nil {
		return err
	}
	for _, evt := range response.GetEvents() {
		var payload countdownUpdatedPayload
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return err
		}
		if payload.CountdownID == countdownID && payload.BeforeRemaining == before && payload.AfterRemaining == after {
			return nil
		}
	}
	return fmt.Errorf("countdown updated event not found")
}

func findCountdownDeleted(ctx context.Context, client gamev1.EventServiceClient, campaignID, sessionID, countdownID string) error {
	filter := `type = "sys.daggerheart.campaign_countdown_deleted"`
	if sessionID != "" {
		filter = `session_id = "` + sessionID + `" AND type = "sys.daggerheart.scene_countdown_deleted"`
	}
	response, err := client.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   200,
		OrderBy:    "seq desc",
		Filter:     filter,
	})
	if err != nil {
		return err
	}
	for _, evt := range response.GetEvents() {
		var payload countdownDeletedPayload
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return err
		}
		if payload.CountdownID == countdownID {
			return nil
		}
	}
	return fmt.Errorf("countdown deleted event not found")
}

func findCountdownTriggerResolved(ctx context.Context, client gamev1.EventServiceClient, campaignID, sessionID, countdownID string, beforeStart, afterStart, beforeRemaining, afterRemaining int) error {
	filter := `type = "sys.daggerheart.campaign_countdown_trigger_resolved"`
	if sessionID != "" {
		filter = `session_id = "` + sessionID + `" AND type = "sys.daggerheart.scene_countdown_trigger_resolved"`
	}
	response, err := client.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   200,
		OrderBy:    "seq desc",
		Filter:     filter,
	})
	if err != nil {
		return err
	}
	for _, evt := range response.GetEvents() {
		var payload countdownTriggerResolvedPayload
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return err
		}
		if payload.CountdownID == countdownID &&
			payload.StartingValueBefore == beforeStart &&
			payload.StartingValueAfter == afterStart &&
			payload.RemainingValueBefore == beforeRemaining &&
			payload.RemainingValueAfter == afterRemaining {
			return nil
		}
	}
	return fmt.Errorf("countdown trigger resolved event not found")
}

func findAdversaryRollResolved(ctx context.Context, client gamev1.EventServiceClient, campaignID, sessionID string, rollSeq uint64) error {
	response, err := client.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   200,
		OrderBy:    "seq desc",
		Filter:     "session_id = \"" + sessionID + "\" AND type = \"action.roll_resolved\"",
	})
	if err != nil {
		return err
	}
	for _, evt := range response.GetEvents() {
		var payload action.RollResolvePayload
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return err
		}
		if payload.RollSeq == rollSeq {
			return nil
		}
	}
	return fmt.Errorf("adversary roll resolved event not found")
}
