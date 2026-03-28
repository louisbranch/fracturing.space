//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration/gametools"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestAIDirectSessionDaggerheartCombatFlowTools(t *testing.T) {
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
	interactionClient := gamev1.NewInteractionServiceClient(conn)
	daggerheartClient := daggerheartv1.NewDaggerheartServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	userID := createAuthUser(t, authAddr, "ai-gametools-combat-flow")
	ctxWithUser := withUserID(ctx, userID)

	createCampaign, err := campaignClient.CreateCampaign(ctxWithUser, &gamev1.CreateCampaignRequest{
		Name:        "AI Combat Flow Campaign",
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      gamev1.GmMode_HUMAN,
		ThemePrompt: "ai combat flow",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	campaignID := createCampaign.GetCampaign().GetId()
	ownerParticipantID := createCampaign.GetOwnerParticipant().GetId()

	attackerID := createCharacter(t, ctxWithUser, characterClient, campaignID, "Flow Attacker")
	targetID := createCharacter(t, ctxWithUser, characterClient, campaignID, "Flow Target")
	supporterOneID := createCharacter(t, ctxWithUser, characterClient, campaignID, "Flow Support One")
	supporterTwoID := createCharacter(t, ctxWithUser, characterClient, campaignID, "Flow Support Two")

	patchDaggerheartProfile(t, ctxWithUser, characterClient, campaignID, attackerID)
	patchDaggerheartProfile(t, ctxWithUser, characterClient, campaignID, targetID)
	patchDaggerheartProfile(t, ctxWithUser, characterClient, campaignID, supporterOneID)
	patchDaggerheartProfile(t, ctxWithUser, characterClient, campaignID, supporterTwoID)
	ensureSessionStartReadiness(t, ctxWithUser, participantClient, characterClient, campaignID, ownerParticipantID, attackerID, targetID, supporterOneID, supporterTwoID)

	startSession := startSessionWithDefaultControllers(t, ctxWithUser, sessionClient, characterClient, campaignID, "AI Combat Flow Session")
	sessionID := startSession.GetSession().GetId()

	createSceneResp, err := sceneClient.CreateScene(ctxWithUser, &gamev1.CreateSceneRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
		Name:       "Combat Flow Scene",
		CharacterIds: []string{
			attackerID,
			targetID,
			supporterOneID,
			supporterTwoID,
		},
	})
	if err != nil {
		t.Fatalf("create scene: %v", err)
	}
	sceneID := createSceneResp.GetSceneId()
	if sceneID == "" {
		t.Fatal("expected scene id")
	}

	createAdversary, err := daggerheartClient.CreateAdversary(ctxWithUser, &daggerheartv1.DaggerheartCreateAdversaryRequest{
		CampaignId:       campaignID,
		SessionId:        sessionID,
		SceneId:          sceneID,
		AdversaryEntryId: "adversary.integration-foe",
	})
	if err != nil {
		t.Fatalf("create adversary: %v", err)
	}
	adversaryID := createAdversary.GetAdversary().GetId()
	if adversaryID == "" {
		t.Fatal("expected adversary id")
	}

	directSession := gametools.NewDirectSession(gametools.Clients{
		Interaction: interactionClient,
		Scene:       sceneClient,
		Campaign:    campaignClient,
		Participant: participantClient,
		Character:   characterClient,
		Session:     sessionClient,
		Snapshot:    snapshotClient,
		Daggerheart: daggerheartClient,
	}, gametools.SessionContext{
		CampaignID:    campaignID,
		SessionID:     sessionID,
		ParticipantID: ownerParticipantID,
	})

	t.Run("attack flow resolve", func(t *testing.T) {
		difficulty := 8
		actionSeed := findReplaySeedForSuccess(t, difficulty)
		damageSeed := uint64(42)

		result, err := directSession.CallTool(ctxWithUser, "daggerheart_attack_flow_resolve", map[string]any{
			"character_id": attackerID,
			"difficulty":   difficulty,
			"target_id":    adversaryID,
			"damage": map[string]any{
				"damage_type":          "PHYSICAL",
				"source":               "attack",
				"source_character_ids": []string{attackerID},
			},
			"require_damage_roll": true,
			"action_rng": map[string]any{
				"seed":      actionSeed,
				"roll_mode": "REPLAY",
			},
			"damage_rng": map[string]any{
				"seed":      damageSeed,
				"roll_mode": "REPLAY",
			},
			"scene_id":            sceneID,
			"target_is_adversary": true,
			"standard_attack": map[string]any{
				"trait":           "strength",
				"damage_dice":     []map[string]any{{"sides": 6, "count": 1}},
				"damage_modifier": 0,
				"attack_range":    "MELEE",
				"damage_critical": false,
			},
		})
		if err != nil {
			t.Fatalf("daggerheart_attack_flow_resolve: %v", err)
		}
		if result.IsError {
			t.Fatalf("daggerheart_attack_flow_resolve returned tool error: %s", result.Output)
		}
		var resolved struct {
			AttackOutcome struct {
				Result struct {
					Success bool `json:"success"`
				} `json:"result"`
			} `json:"attack_outcome"`
			DamageRoll struct {
				RollSeq uint64 `json:"roll_seq"`
			} `json:"damage_roll"`
			AdversaryDamageApplied struct {
				Adversary struct {
					HP    int `json:"hp"`
					HPMax int `json:"hp_max"`
				} `json:"adversary"`
			} `json:"adversary_damage_applied"`
		}
		if err := json.Unmarshal([]byte(result.Output), &resolved); err != nil {
			t.Fatalf("decode attack flow result: %v", err)
		}
		if !resolved.AttackOutcome.Result.Success {
			t.Fatalf("expected successful attack flow, got %s", result.Output)
		}
		if resolved.DamageRoll.RollSeq == 0 {
			t.Fatalf("expected damage roll sequence, got %s", result.Output)
		}
		if resolved.AdversaryDamageApplied.Adversary.HP >= resolved.AdversaryDamageApplied.Adversary.HPMax {
			t.Fatalf("expected adversary hp to drop, got %s", result.Output)
		}
	})

	t.Run("adversary attack flow resolve", func(t *testing.T) {
		attackSeed := uint64(21)
		damageSeed := uint64(42)

		result, err := directSession.CallTool(ctxWithUser, "daggerheart_adversary_attack_flow_resolve", map[string]any{
			"adversary_id": adversaryID,
			"target_id":    targetID,
			"difficulty":   1,
			"damage": map[string]any{
				"damage_type": "PHYSICAL",
				"source":      "adversary attack",
			},
			"require_damage_roll": true,
			"attack_rng": map[string]any{
				"seed":      attackSeed,
				"roll_mode": "REPLAY",
			},
			"damage_rng": map[string]any{
				"seed":      damageSeed,
				"roll_mode": "REPLAY",
			},
			"scene_id": sceneID,
		})
		if err != nil {
			t.Fatalf("daggerheart_adversary_attack_flow_resolve: %v", err)
		}
		if result.IsError {
			t.Fatalf("daggerheart_adversary_attack_flow_resolve returned tool error: %s", result.Output)
		}
		var resolved struct {
			AttackOutcome struct {
				Result struct {
					Success bool `json:"success"`
				} `json:"result"`
			} `json:"attack_outcome"`
			DamageRoll struct {
				RollSeq uint64 `json:"roll_seq"`
			} `json:"damage_roll"`
			DamageApplied struct {
				State struct {
					HP int `json:"hp"`
				} `json:"state"`
			} `json:"damage_applied"`
		}
		if err := json.Unmarshal([]byte(result.Output), &resolved); err != nil {
			t.Fatalf("decode adversary attack flow result: %v", err)
		}
		if !resolved.AttackOutcome.Result.Success {
			t.Fatalf("expected successful adversary attack flow, got %s", result.Output)
		}
		if resolved.DamageRoll.RollSeq == 0 {
			t.Fatalf("expected adversary damage roll sequence, got %s", result.Output)
		}
		state := fetchCharacterState(t, ctxWithUser, snapshotClient, campaignID, targetID)
		if got := int(state.GetHp()); got >= 6 {
			t.Fatalf("expected target hp to drop below 6, got %d", got)
		}
	})

	t.Run("group action flow resolve", func(t *testing.T) {
		difficulty := 10
		leaderSeed := findReplaySeedForSuccess(t, difficulty)
		supportSeedOne := findReplaySeedForReaction(t, difficulty, true)
		supportSeedTwo := findReplaySeedForReaction(t, difficulty, true)

		result, err := directSession.CallTool(ctxWithUser, "daggerheart_group_action_flow_resolve", map[string]any{
			"leader_character_id": attackerID,
			"leader_trait":        "finesse",
			"difficulty":          difficulty,
			"leader_rng": map[string]any{
				"seed":      leaderSeed,
				"roll_mode": "REPLAY",
			},
			"supporters": []map[string]any{
				{
					"character_id": supporterOneID,
					"trait":        "agility",
					"rng": map[string]any{
						"seed":      supportSeedOne,
						"roll_mode": "REPLAY",
					},
				},
				{
					"character_id": supporterTwoID,
					"trait":        "agility",
					"rng": map[string]any{
						"seed":      supportSeedTwo,
						"roll_mode": "REPLAY",
					},
				},
			},
			"scene_id": sceneID,
		})
		if err != nil {
			t.Fatalf("daggerheart_group_action_flow_resolve: %v", err)
		}
		if result.IsError {
			t.Fatalf("daggerheart_group_action_flow_resolve returned tool error: %s", result.Output)
		}
		var resolved struct {
			SupportModifier  int `json:"support_modifier"`
			SupportSuccesses int `json:"support_successes"`
			SupportFailures  int `json:"support_failures"`
		}
		if err := json.Unmarshal([]byte(result.Output), &resolved); err != nil {
			t.Fatalf("decode group action flow result: %v", err)
		}
		if resolved.SupportModifier != 2 || resolved.SupportSuccesses != 2 || resolved.SupportFailures != 0 {
			t.Fatalf("unexpected group action support summary: %s", result.Output)
		}
	})

	t.Run("reaction flow resolve", func(t *testing.T) {
		difficulty := 8
		seed := findReplaySeedForSuccess(t, difficulty)

		result, err := directSession.CallTool(ctxWithUser, "daggerheart_reaction_flow_resolve", map[string]any{
			"character_id": supporterTwoID,
			"trait":        "agility",
			"difficulty":   difficulty,
			"reaction_rng": map[string]any{
				"seed":      seed,
				"roll_mode": "REPLAY",
			},
			"scene_id": sceneID,
		})
		if err != nil {
			t.Fatalf("daggerheart_reaction_flow_resolve: %v", err)
		}
		if result.IsError {
			t.Fatalf("daggerheart_reaction_flow_resolve returned tool error: %s", result.Output)
		}
		var resolved struct {
			ActionRoll struct {
				RollSeq uint64 `json:"roll_seq"`
			} `json:"action_roll"`
			ReactionOutcome struct {
				RollSeq uint64 `json:"roll_seq"`
				Result  struct {
					Success            bool `json:"success"`
					Crit               bool `json:"crit"`
					CritNegatesEffects bool `json:"crit_negates_effects"`
					EffectsNegated     bool `json:"effects_negated"`
				} `json:"result"`
			} `json:"reaction_outcome"`
		}
		if err := json.Unmarshal([]byte(result.Output), &resolved); err != nil {
			t.Fatalf("decode reaction flow result: %v", err)
		}
		if resolved.ActionRoll.RollSeq == 0 || resolved.ReactionOutcome.RollSeq != resolved.ActionRoll.RollSeq {
			t.Fatalf("expected reaction flow roll_seq alignment, got %s", result.Output)
		}
		if resolved.ReactionOutcome.Result.EffectsNegated != (resolved.ReactionOutcome.Result.Crit && resolved.ReactionOutcome.Result.CritNegatesEffects) {
			t.Fatalf("expected effects_negated contract to hold, got %s", result.Output)
		}
	})

	t.Run("tag team flow resolve", func(t *testing.T) {
		difficulty := 8
		firstSeed := findReplaySeedForSuccess(t, difficulty)
		secondSeed := findReplaySeedForSuccess(t, difficulty)

		result, err := directSession.CallTool(ctxWithUser, "daggerheart_tag_team_flow_resolve", map[string]any{
			"first": map[string]any{
				"character_id": attackerID,
				"trait":        "presence",
				"rng": map[string]any{
					"seed":      firstSeed,
					"roll_mode": "REPLAY",
				},
			},
			"second": map[string]any{
				"character_id": supporterOneID,
				"trait":        "knowledge",
				"rng": map[string]any{
					"seed":      secondSeed,
					"roll_mode": "REPLAY",
				},
			},
			"difficulty":            difficulty,
			"selected_character_id": attackerID,
			"scene_id":              sceneID,
		})
		if err != nil {
			t.Fatalf("daggerheart_tag_team_flow_resolve: %v", err)
		}
		if result.IsError {
			t.Fatalf("daggerheart_tag_team_flow_resolve returned tool error: %s", result.Output)
		}
		var resolved struct {
			FirstRoll struct {
				RollSeq uint64 `json:"roll_seq"`
			} `json:"first_roll"`
			SelectedRollSeq uint64 `json:"selected_roll_seq"`
		}
		if err := json.Unmarshal([]byte(result.Output), &resolved); err != nil {
			t.Fatalf("decode tag team flow result: %v", err)
		}
		if resolved.SelectedRollSeq == 0 || resolved.SelectedRollSeq != resolved.FirstRoll.RollSeq {
			t.Fatalf("expected selected roll seq to match the first roll, got %s", result.Output)
		}
	})
}
