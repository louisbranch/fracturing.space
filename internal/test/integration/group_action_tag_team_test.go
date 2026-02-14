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
)

type groupActionResolvedPayload struct {
	LeaderCharacterID string `json:"leader_character_id"`
	LeaderRollSeq     uint64 `json:"leader_roll_seq"`
	SupportSuccesses  int    `json:"support_successes"`
	SupportFailures   int    `json:"support_failures"`
	SupportModifier   int    `json:"support_modifier"`
	Supporters        []struct {
		CharacterID string `json:"character_id"`
		RollSeq     uint64 `json:"roll_seq"`
		Success     bool   `json:"success"`
	} `json:"supporters"`
}

type tagTeamResolvedPayload struct {
	FirstCharacterID    string `json:"first_character_id"`
	FirstRollSeq        uint64 `json:"first_roll_seq"`
	SecondCharacterID   string `json:"second_character_id"`
	SecondRollSeq       uint64 `json:"second_roll_seq"`
	SelectedCharacterID string `json:"selected_character_id"`
	SelectedRollSeq     uint64 `json:"selected_roll_seq"`
}

func TestDaggerheartGroupActionFlow(t *testing.T) {
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
	eventClient := gamev1.NewEventServiceClient(conn)
	daggerheartClient := daggerheartv1.NewDaggerheartServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()
	userID := createAuthUser(t, authAddr, "Group Action GM")
	ctx = withUserID(ctx, userID)

	createCampaign, err := campaignClient.CreateCampaign(ctx, &gamev1.CreateCampaignRequest{
		Name:               "Group Action Campaign",
		System:             commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:             gamev1.GmMode_HUMAN,
		ThemePrompt:        "group action",
		CreatorDisplayName: "Group GM",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if createCampaign.GetCampaign() == nil {
		t.Fatal("expected campaign")
	}
	campaignID := createCampaign.GetCampaign().GetId()

	leader := createCharacter(t, ctx, characterClient, campaignID, "Group Leader")
	supporterOne := createCharacter(t, ctx, characterClient, campaignID, "Support One")
	supporterTwo := createCharacter(t, ctx, characterClient, campaignID, "Support Two")

	patchDaggerheartProfile(t, ctx, characterClient, campaignID, leader)
	patchDaggerheartProfile(t, ctx, characterClient, campaignID, supporterOne)
	patchDaggerheartProfile(t, ctx, characterClient, campaignID, supporterTwo)

	startSession, err := sessionClient.StartSession(ctx, &gamev1.StartSessionRequest{
		CampaignId: campaignID,
		Name:       "Group Session",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if startSession.GetSession() == nil {
		t.Fatal("expected session")
	}
	sessionID := startSession.GetSession().GetId()

	difficulty := 10
	leaderSeed := findReplaySeedForSuccess(t, difficulty)
	supportSeedOne := findReplaySeedForReaction(t, difficulty, true)
	supportSeedTwo := findReplaySeedForReaction(t, difficulty, true)

	result, err := daggerheartClient.SessionGroupActionFlow(ctx, &daggerheartv1.SessionGroupActionFlowRequest{
		CampaignId:        campaignID,
		SessionId:         sessionID,
		LeaderCharacterId: leader,
		LeaderTrait:       "finesse",
		Difficulty:        int32(difficulty),
		LeaderRng:         &commonv1.RngRequest{Seed: &leaderSeed, RollMode: commonv1.RollMode_REPLAY},
		Supporters: []*daggerheartv1.GroupActionSupporter{
			{
				CharacterId: supporterOne,
				Trait:       "agility",
				Rng:         &commonv1.RngRequest{Seed: &supportSeedOne, RollMode: commonv1.RollMode_REPLAY},
			},
			{
				CharacterId: supporterTwo,
				Trait:       "agility",
				Rng:         &commonv1.RngRequest{Seed: &supportSeedTwo, RollMode: commonv1.RollMode_REPLAY},
			},
		},
	})
	if err != nil {
		t.Fatalf("session group action flow: %v", err)
	}
	if result.GetLeaderRoll() == nil || result.GetLeaderOutcome() == nil {
		t.Fatal("expected leader roll and outcome")
	}
	if result.GetSupportModifier() != 2 || result.GetSupportSuccesses() != 2 || result.GetSupportFailures() != 0 {
		t.Fatal("expected support modifier to reflect two successes")
	}

	resolved, err := findGroupActionResolved(ctx, eventClient, campaignID, sessionID, result.GetLeaderRoll().GetRollSeq())
	if err != nil {
		t.Fatalf("find group action resolved: %v", err)
	}
	if resolved.LeaderCharacterID != leader || resolved.LeaderRollSeq != result.GetLeaderRoll().GetRollSeq() {
		t.Fatal("expected group action payload to match leader roll")
	}
}

func TestDaggerheartTagTeamFlow(t *testing.T) {
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
	eventClient := gamev1.NewEventServiceClient(conn)
	daggerheartClient := daggerheartv1.NewDaggerheartServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()
	userID := createAuthUser(t, authAddr, "Tag Team GM")
	ctx = withUserID(ctx, userID)

	createCampaign, err := campaignClient.CreateCampaign(ctx, &gamev1.CreateCampaignRequest{
		Name:               "Tag Team Campaign",
		System:             commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:             gamev1.GmMode_HUMAN,
		ThemePrompt:        "tag team",
		CreatorDisplayName: "Tag Team GM",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if createCampaign.GetCampaign() == nil {
		t.Fatal("expected campaign")
	}
	campaignID := createCampaign.GetCampaign().GetId()

	first := createCharacter(t, ctx, characterClient, campaignID, "Tag One")
	second := createCharacter(t, ctx, characterClient, campaignID, "Tag Two")

	patchDaggerheartProfile(t, ctx, characterClient, campaignID, first)
	patchDaggerheartProfile(t, ctx, characterClient, campaignID, second)

	startSession, err := sessionClient.StartSession(ctx, &gamev1.StartSessionRequest{
		CampaignId: campaignID,
		Name:       "Tag Session",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if startSession.GetSession() == nil {
		t.Fatal("expected session")
	}
	sessionID := startSession.GetSession().GetId()

	difficulty := 8
	firstSeed := findReplaySeedForSuccess(t, difficulty)
	secondSeed := findReplaySeedForSuccess(t, difficulty)

	result, err := daggerheartClient.SessionTagTeamFlow(ctx, &daggerheartv1.SessionTagTeamFlowRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
		Difficulty: int32(difficulty),
		First: &daggerheartv1.TagTeamParticipant{
			CharacterId: first,
			Trait:       "presence",
			Rng:         &commonv1.RngRequest{Seed: &firstSeed, RollMode: commonv1.RollMode_REPLAY},
		},
		Second: &daggerheartv1.TagTeamParticipant{
			CharacterId: second,
			Trait:       "knowledge",
			Rng:         &commonv1.RngRequest{Seed: &secondSeed, RollMode: commonv1.RollMode_REPLAY},
		},
		SelectedCharacterId: first,
	})
	if err != nil {
		t.Fatalf("session tag team flow: %v", err)
	}
	if result.GetFirstRoll() == nil || result.GetSecondRoll() == nil {
		t.Fatal("expected both tag team rolls")
	}
	if result.GetSelectedOutcome() == nil {
		t.Fatal("expected selected outcome")
	}
	if result.GetSelectedRollSeq() != result.GetFirstRoll().GetRollSeq() {
		t.Fatal("expected selected roll seq to match first roll")
	}

	resolved, err := findTagTeamResolved(ctx, eventClient, campaignID, sessionID, result.GetSelectedRollSeq())
	if err != nil {
		t.Fatalf("find tag team resolved: %v", err)
	}
	if resolved.SelectedCharacterID != first || resolved.SelectedRollSeq != result.GetSelectedRollSeq() {
		t.Fatal("expected tag team payload to match selected roll")
	}
}

func findReplaySeedForReaction(t *testing.T, difficulty int, success bool) uint64 {
	t.Helper()
	for seed := uint64(1); seed < 50000; seed++ {
		difficultyValue := difficulty
		result, err := daggerheartdomain.RollReaction(daggerheartdomain.ReactionRequest{
			Modifier:   0,
			Difficulty: &difficultyValue,
			Seed:       int64(seed),
		})
		if err != nil {
			continue
		}
		if result.MeetsDifficulty == success {
			return seed
		}
	}
	if success {
		t.Fatal("no replay seed found for successful reaction roll")
	}
	t.Fatal("no replay seed found for failed reaction roll")
	return 0
}

func findGroupActionResolved(ctx context.Context, client gamev1.EventServiceClient, campaignID, sessionID string, leaderRollSeq uint64) (groupActionResolvedPayload, error) {
	response, err := client.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   200,
		OrderBy:    "seq desc",
		Filter:     "session_id = \"" + sessionID + "\" AND type = \"action.group_action_resolved\"",
	})
	if err != nil {
		return groupActionResolvedPayload{}, err
	}
	for _, evt := range response.GetEvents() {
		var payload groupActionResolvedPayload
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return groupActionResolvedPayload{}, err
		}
		if payload.LeaderRollSeq == leaderRollSeq {
			return payload, nil
		}
	}
	return groupActionResolvedPayload{}, fmt.Errorf("group action resolved event not found")
}

func findTagTeamResolved(ctx context.Context, client gamev1.EventServiceClient, campaignID, sessionID string, selectedRollSeq uint64) (tagTeamResolvedPayload, error) {
	response, err := client.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   200,
		OrderBy:    "seq desc",
		Filter:     "session_id = \"" + sessionID + "\" AND type = \"action.tag_team_resolved\"",
	})
	if err != nil {
		return tagTeamResolvedPayload{}, err
	}
	for _, evt := range response.GetEvents() {
		var payload tagTeamResolvedPayload
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return tagTeamResolvedPayload{}, err
		}
		if payload.SelectedRollSeq == selectedRollSeq {
			return payload, nil
		}
	}
	return tagTeamResolvedPayload{}, fmt.Errorf("tag team resolved event not found")
}
