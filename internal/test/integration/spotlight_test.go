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
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/domain"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type sessionSpotlightPayload struct {
	SpotlightType string `json:"spotlight_type"`
	CharacterID   string `json:"character_id,omitempty"`
}

type sessionGatePayload struct {
	GateType string `json:"gate_type"`
}

func TestSessionSpotlightLifecycle(t *testing.T) {
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

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	userID := createAuthUser(t, authAddr, "spotlight-gm")
	ctxWithUser := withUserID(ctx, userID)

	createCampaign, err := campaignClient.CreateCampaign(ctxWithUser, &gamev1.CreateCampaignRequest{
		Name:               "Spotlight Campaign",
		System:             commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:             gamev1.GmMode_HUMAN,
		ThemePrompt:        "spotlight",
		CreatorDisplayName: "Spotlight GM",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if createCampaign.GetCampaign() == nil {
		t.Fatal("expected campaign")
	}
	campaignID := createCampaign.GetCampaign().GetId()

	characterID := createCharacter(t, ctxWithUser, characterClient, campaignID, "Spotlight Hero")

	startSession, err := sessionClient.StartSession(ctxWithUser, &gamev1.StartSessionRequest{
		CampaignId: campaignID,
		Name:       "Spotlight Session",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if startSession.GetSession() == nil {
		t.Fatal("expected session")
	}
	sessionID := startSession.GetSession().GetId()

	setResp, err := sessionClient.SetSessionSpotlight(ctxWithUser, &gamev1.SetSessionSpotlightRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		Type:        gamev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER,
		CharacterId: characterID,
	})
	if err != nil {
		t.Fatalf("set spotlight: %v", err)
	}
	if setResp.GetSpotlight() == nil {
		t.Fatal("expected spotlight in set response")
	}
	if setResp.GetSpotlight().GetCharacterId() != characterID {
		t.Fatalf("spotlight character id = %q, want %q", setResp.GetSpotlight().GetCharacterId(), characterID)
	}

	getResp, err := sessionClient.GetSessionSpotlight(ctxWithUser, &gamev1.GetSessionSpotlightRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err != nil {
		t.Fatalf("get spotlight: %v", err)
	}
	if getResp.GetSpotlight() == nil {
		t.Fatal("expected spotlight in get response")
	}
	if getResp.GetSpotlight().GetCharacterId() != characterID {
		t.Fatalf("spotlight character id = %q, want %q", getResp.GetSpotlight().GetCharacterId(), characterID)
	}

	_, err = sessionClient.ClearSessionSpotlight(ctxWithUser, &gamev1.ClearSessionSpotlightRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
		Reason:     "scene shift",
	})
	if err != nil {
		t.Fatalf("clear spotlight: %v", err)
	}

	_, err = sessionClient.GetSessionSpotlight(ctxWithUser, &gamev1.GetSessionSpotlightRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
	})
	if err == nil {
		t.Fatal("expected get spotlight to fail after clear")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected status error, got %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Fatalf("expected not found, got %s", st.Code())
	}
}

func TestGmConsequenceOpensGateAndSpotlight(t *testing.T) {
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

	userID := createAuthUser(t, authAddr, "consequence-gm")
	ctxWithUser := withUserID(ctx, userID)

	createCampaign, err := campaignClient.CreateCampaign(ctxWithUser, &gamev1.CreateCampaignRequest{
		Name:               "GM Consequence Campaign",
		System:             commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:             gamev1.GmMode_HUMAN,
		ThemePrompt:        "consequence",
		CreatorDisplayName: "Consequence GM",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if createCampaign.GetCampaign() == nil {
		t.Fatal("expected campaign")
	}
	campaignID := createCampaign.GetCampaign().GetId()

	characterID := createCharacter(t, ctxWithUser, characterClient, campaignID, "Consequence Hero")
	patchDaggerheartProfile(t, ctxWithUser, characterClient, campaignID, characterID)

	startSession, err := sessionClient.StartSession(ctxWithUser, &gamev1.StartSessionRequest{
		CampaignId: campaignID,
		Name:       "Consequence Session",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if startSession.GetSession() == nil {
		t.Fatal("expected session")
	}
	sessionID := startSession.GetSession().GetId()

	difficulty := 8
	seed := findReplaySeedForFearOutcome(t, difficulty)

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

	ctxWithMeta := metadata.NewOutgoingContext(ctx, metadata.Pairs(
		grpcmeta.CampaignIDHeader, campaignID,
		grpcmeta.SessionIDHeader, sessionID,
	))
	_, err = daggerheartClient.ApplyRollOutcome(ctxWithMeta, &daggerheartv1.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		RollSeq:   rollResp.GetRollSeq(),
	})
	if err != nil {
		t.Fatalf("apply roll outcome: %v", err)
	}

	if err := findSessionSpotlightSet(ctx, eventClient, campaignID, sessionID, "gm"); err != nil {
		t.Fatalf("find spotlight set: %v", err)
	}
	if err := findSessionGateOpened(ctx, eventClient, campaignID, sessionID, "gm_consequence"); err != nil {
		t.Fatalf("find gate opened: %v", err)
	}
}

func findReplaySeedForFearOutcome(t *testing.T, difficulty int) uint64 {
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
			continue
		}
		switch result.Outcome {
		case daggerheartdomain.OutcomeFailureWithFear, daggerheartdomain.OutcomeSuccessWithFear:
			return seed
		}
	}
	t.Fatal("no replay seed found for fear outcome")
	return 0
}

func findSessionSpotlightSet(ctx context.Context, client gamev1.EventServiceClient, campaignID, sessionID, spotlightType string) error {
	response, err := client.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   200,
		OrderBy:    "seq desc",
		Filter:     "session_id = \"" + sessionID + "\" AND type = \"session.spotlight_set\"",
	})
	if err != nil {
		return err
	}
	for _, evt := range response.GetEvents() {
		var payload sessionSpotlightPayload
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return err
		}
		if payload.SpotlightType == spotlightType {
			return nil
		}
	}
	return fmt.Errorf("session spotlight set event not found")
}

func findSessionGateOpened(ctx context.Context, client gamev1.EventServiceClient, campaignID, sessionID, gateType string) error {
	response, err := client.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   200,
		OrderBy:    "seq desc",
		Filter:     "session_id = \"" + sessionID + "\" AND type = \"session.gate_opened\"",
	})
	if err != nil {
		return err
	}
	for _, evt := range response.GetEvents() {
		var payload sessionGatePayload
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return err
		}
		if payload.GateType == gateType {
			return nil
		}
	}
	return fmt.Errorf("session gate opened event not found")
}
