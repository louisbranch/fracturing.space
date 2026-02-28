//go:build integration

package integration

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func TestDaggerheartRestConsequences(t *testing.T) {
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
	sessionClient := gamev1.NewSessionServiceClient(conn)
	snapshotClient := gamev1.NewSnapshotServiceClient(conn)
	eventClient := gamev1.NewEventServiceClient(conn)
	daggerheartClient := daggerheartv1.NewDaggerheartServiceClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	userID := createAuthUser(t, authAddr, "rest-gm")
	ctxWithUser := withUserID(ctx, userID)

	createCampaign, err := campaignClient.CreateCampaign(ctxWithUser, &gamev1.CreateCampaignRequest{
		Name:        "Rest Consequences Campaign",
		System:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:      gamev1.GmMode_HUMAN,
		ThemePrompt: "rest",
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	if createCampaign.GetCampaign() == nil {
		t.Fatal("expected campaign")
	}
	campaignID := createCampaign.GetCampaign().GetId()

	startSession, err := sessionClient.StartSession(ctxWithUser, &gamev1.StartSessionRequest{
		CampaignId: campaignID,
		Name:       "Rest Session",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if startSession.GetSession() == nil {
		t.Fatal("expected session")
	}
	sessionID := startSession.GetSession().GetId()

	createCountdown, err := daggerheartClient.CreateCountdown(ctxWithUser, &daggerheartv1.DaggerheartCreateCountdownRequest{
		CampaignId: campaignID,
		SessionId:  sessionID,
		Name:       "Long-Term Countdown",
		Kind:       daggerheartv1.DaggerheartCountdownKind_DAGGERHEART_COUNTDOWN_KIND_CONSEQUENCE,
		Current:    0,
		Max:        4,
		Direction:  daggerheartv1.DaggerheartCountdownDirection_DAGGERHEART_COUNTDOWN_DIRECTION_INCREASE,
		Looping:    false,
	})
	if err != nil {
		t.Fatalf("create countdown: %v", err)
	}
	if createCountdown.GetCountdown() == nil {
		t.Fatal("expected countdown")
	}
	countdownID := createCountdown.GetCountdown().GetCountdownId()
	if countdownID == "" {
		t.Fatal("expected countdown id")
	}

	_, err = snapshotClient.UpdateSnapshotState(ctxWithUser, &gamev1.UpdateSnapshotStateRequest{
		CampaignId: campaignID,
		SystemSnapshotUpdate: &gamev1.UpdateSnapshotStateRequest_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartSnapshot{
				GmFear:                2,
				ConsecutiveShortRests: 0,
			},
		},
	})
	if err != nil {
		t.Fatalf("update snapshot: %v", err)
	}

	partySize := 3
	seed := uint64(9)
	outcome, err := daggerheart.ResolveRestOutcome(
		daggerheart.RestState{ConsecutiveShortRests: 0},
		daggerheart.RestTypeLong,
		false,
		int64(seed),
		partySize,
	)
	if err != nil {
		t.Fatalf("resolve rest outcome: %v", err)
	}
	if !outcome.AdvanceCountdown {
		t.Fatal("expected long rest to advance countdown")
	}
	expectedFear := 2 + outcome.GMFearGain
	if expectedFear > daggerheart.GMFearMax {
		expectedFear = daggerheart.GMFearMax
	}

	sessionCtx := withSessionID(ctxWithUser, sessionID)
	resp, err := daggerheartClient.ApplyRest(sessionCtx, &daggerheartv1.DaggerheartApplyRestRequest{
		CampaignId: campaignID,
		Rest: &daggerheartv1.DaggerheartRestRequest{
			RestType:            daggerheartv1.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG,
			Interrupted:         false,
			PartySize:           int32(partySize),
			LongTermCountdownId: countdownID,
			Rng: &commonv1.RngRequest{
				Seed:     &seed,
				RollMode: commonv1.RollMode_REPLAY,
			},
		},
	})
	if err != nil {
		t.Fatalf("apply rest: %v", err)
	}
	if resp.GetSnapshot() == nil {
		t.Fatal("expected snapshot")
	}
	if resp.GetSnapshot().GetGmFear() != int32(expectedFear) {
		t.Fatalf("gm fear = %d, want %d", resp.GetSnapshot().GetGmFear(), expectedFear)
	}
	if resp.GetSnapshot().GetConsecutiveShortRests() != 0 {
		t.Fatalf("short rests = %d, want 0", resp.GetSnapshot().GetConsecutiveShortRests())
	}

	if err := findCountdownUpdated(ctxWithUser, eventClient, campaignID, sessionID, countdownID, 0, 1); err != nil {
		t.Fatalf("find countdown updated: %v", err)
	}
}
