//go:build integration
// +build integration

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
)

type reactionResolvedPayload struct {
	RollSeq     uint64 `json:"roll_seq"`
	CharacterID string `json:"character_id"`
}

func TestDaggerheartReactionFlow(t *testing.T) {
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
		Name:               "Reaction Flow Campaign",
		System:             commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:             gamev1.GmMode_HUMAN,
		ThemePrompt:        "reaction flow",
		CreatorDisplayName: "Reaction GM",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if createCampaign.GetCampaign() == nil {
		t.Fatal("expected campaign")
	}
	campaignID := createCampaign.GetCampaign().GetId()

	reactor := createCharacter(t, ctx, characterClient, campaignID, "Reaction Hero")
	patchDaggerheartProfile(t, ctx, characterClient, campaignID, reactor)

	startSession, err := sessionClient.StartSession(ctx, &gamev1.StartSessionRequest{
		CampaignId: campaignID,
		Name:       "Reaction Session",
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

	result, err := daggerheartClient.SessionReactionFlow(ctx, &daggerheartv1.SessionReactionFlowRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CharacterId: reactor,
		Trait:       "agility",
		Difficulty:  int32(difficulty),
		ReactionRng: &commonv1.RngRequest{
			Seed:     &seed,
			RollMode: commonv1.RollMode_REPLAY,
		},
	})
	if err != nil {
		t.Fatalf("session reaction flow: %v", err)
	}
	if result.GetActionRoll() == nil || result.GetRollOutcome() == nil || result.GetReactionOutcome() == nil {
		t.Fatal("expected roll, outcome, and reaction outcome results")
	}
	if result.GetActionRoll().GetRollSeq() == 0 {
		t.Fatal("expected roll seq")
	}
	if result.GetReactionOutcome().GetRollSeq() != result.GetActionRoll().GetRollSeq() {
		t.Fatal("expected reaction outcome roll seq to match action roll")
	}
	reactionResult := result.GetReactionOutcome().GetResult()
	if reactionResult == nil {
		t.Fatal("expected reaction outcome result")
	}
	if reactionResult.GetEffectsNegated() != (reactionResult.GetCrit() && reactionResult.GetCritNegatesEffects()) {
		t.Fatal("expected effects_negated to match crit and crit_negates_effects")
	}

	resolved, err := findReactionResolved(ctx, eventClient, campaignID, sessionID, result.GetActionRoll().GetRollSeq())
	if err != nil {
		t.Fatalf("find reaction resolved: %v", err)
	}
	if resolved.RollSeq != result.GetActionRoll().GetRollSeq() || resolved.CharacterID != reactor {
		t.Fatal("expected reaction resolved payload to match roll")
	}
}

func findReactionResolved(ctx context.Context, client gamev1.EventServiceClient, campaignID, sessionID string, rollSeq uint64) (reactionResolvedPayload, error) {
	response, err := client.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   200,
		OrderBy:    "seq desc",
		Filter:     "session_id = \"" + sessionID + "\" AND type = \"action.reaction_resolved\"",
	})
	if err != nil {
		return reactionResolvedPayload{}, err
	}
	for _, evt := range response.GetEvents() {
		var payload reactionResolvedPayload
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return reactionResolvedPayload{}, err
		}
		if payload.RollSeq == rollSeq {
			return payload, nil
		}
	}
	return reactionResolvedPayload{}, fmt.Errorf("reaction resolved event not found")
}
