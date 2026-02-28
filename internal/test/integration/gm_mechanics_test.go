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
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type gmFearChangedPayload struct {
	Before int    `json:"before"`
	After  int    `json:"after"`
	Reason string `json:"reason"`
}

type countdownCreatedPayload struct {
	CountdownID string `json:"countdown_id"`
}

type countdownUpdatedPayload struct {
	CountdownID string `json:"countdown_id"`
	Before      int    `json:"before"`
	After       int    `json:"after"`
	Delta       int    `json:"delta"`
}

type countdownDeletedPayload struct {
	CountdownID string `json:"countdown_id"`
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
	ensureSessionStartReadiness(t, ctx, participantClient, characterClient, campaignID, ownerParticipantID)

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

	createAdversary, err := daggerheartClient.CreateAdversary(ctx, &daggerheartv1.DaggerheartCreateAdversaryRequest{
		CampaignId: campaignID,
		Name:       "Adversary One",
		Kind:       "minion",
		SessionId:  wrapperspb.String(sessionID),
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
		Move:       "hard_move",
		FearSpent:  2,
	})
	if err != nil {
		t.Fatalf("apply gm move: %v", err)
	}
	if moveResp.GetGmFearBefore() != 3 || moveResp.GetGmFearAfter() != 1 {
		t.Fatalf("gm fear = %d -> %d, want 3 -> 1", moveResp.GetGmFearBefore(), moveResp.GetGmFearAfter())
	}

	if err := findGMFearChanged(ctx, eventClient, campaignID, sessionID, 3, 1); err != nil {
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
	ensureSessionStartReadiness(t, ctx, participantClient, characterClient, campaignID, ownerParticipantID)

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

	createResp, err := daggerheartClient.CreateCountdown(ctx, &daggerheartv1.DaggerheartCreateCountdownRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
		Name:       "Wagon Timer",
		Kind:       daggerheartv1.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_PROGRESS,
		Current:    0,
		Max:        4,
		Direction:  daggerheartv1.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Looping:    false,
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

	updateResp, err := daggerheartClient.UpdateCountdown(ctx, &daggerheartv1.DaggerheartUpdateCountdownRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
		CountdownId: countdownID,
		Delta:       2,
		Reason:      "advance",
	})
	if err != nil {
		t.Fatalf("update countdown: %v", err)
	}
	if updateResp.GetAfter() != 2 {
		t.Fatalf("countdown after = %d, want 2", updateResp.GetAfter())
	}
	if err := findCountdownUpdated(ctx, eventClient, campaignID, sessionID, countdownID, 0, 2); err != nil {
		t.Fatalf("find countdown updated: %v", err)
	}

	_, err = daggerheartClient.DeleteCountdown(ctx, &daggerheartv1.DaggerheartDeleteCountdownRequest{
		CampaignId:  campaignID,
		SessionId:   sessionID,
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
	ensureSessionStartReadiness(t, ctx, participantClient, characterClient, campaignID, ownerParticipantID)

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

	createAdversary, err := daggerheartClient.CreateAdversary(ctx, &daggerheartv1.DaggerheartCreateAdversaryRequest{
		CampaignId: campaignID,
		Name:       "Adversary One",
		Kind:       "minion",
		SessionId:  wrapperspb.String(sessionID),
	})
	if err != nil {
		t.Fatalf("create adversary: %v", err)
	}
	if createAdversary.GetAdversary() == nil {
		t.Fatal("expected adversary")
	}

	seed := uint64(42)
	rollResp, err := daggerheartClient.SessionAdversaryAttackRoll(ctx, &daggerheartv1.SessionAdversaryAttackRollRequest{
		CampaignId:     campaignID,
		SessionId:      sessionID,
		AdversaryId:    createAdversary.GetAdversary().GetId(),
		AttackModifier: 2,
		Advantage:      1,
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

func findGMFearChanged(ctx context.Context, client gamev1.EventServiceClient, campaignID, sessionID string, before, after int) error {
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
		if payload.Before == before && payload.After == after {
			return nil
		}
	}
	return fmt.Errorf("gm fear changed event not found")
}

func findCountdownCreated(ctx context.Context, client gamev1.EventServiceClient, campaignID, sessionID, countdownID string) error {
	response, err := client.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   200,
		OrderBy:    "seq desc",
		Filter:     "session_id = \"" + sessionID + "\" AND type = \"sys.daggerheart.countdown_created\"",
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
	response, err := client.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   200,
		OrderBy:    "seq desc",
		Filter:     "session_id = \"" + sessionID + "\" AND type = \"sys.daggerheart.countdown_updated\"",
	})
	if err != nil {
		return err
	}
	for _, evt := range response.GetEvents() {
		var payload countdownUpdatedPayload
		if err := json.Unmarshal(evt.GetPayloadJson(), &payload); err != nil {
			return err
		}
		if payload.CountdownID == countdownID && payload.Before == before && payload.After == after {
			return nil
		}
	}
	return fmt.Errorf("countdown updated event not found")
}

func findCountdownDeleted(ctx context.Context, client gamev1.EventServiceClient, campaignID, sessionID, countdownID string) error {
	response, err := client.ListEvents(ctx, &gamev1.ListEventsRequest{
		CampaignId: campaignID,
		PageSize:   200,
		OrderBy:    "seq desc",
		Filter:     "session_id = \"" + sessionID + "\" AND type = \"sys.daggerheart.countdown_deleted\"",
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
