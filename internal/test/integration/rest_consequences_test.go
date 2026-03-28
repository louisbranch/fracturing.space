//go:build integration

package integration

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
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
	participantClient := gamev1.NewParticipantServiceClient(conn)
	characterClient := gamev1.NewCharacterServiceClient(conn)
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
	participantsResp, err := participantClient.ListParticipants(ctxWithUser, &gamev1.ListParticipantsRequest{
		CampaignId: campaignID,
		PageSize:   200,
	})
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}
	if len(participantsResp.GetParticipants()) == 0 {
		t.Fatal("expected owner participant")
	}
	ownerParticipantID := participantsResp.GetParticipants()[0].GetId()
	ensureSessionStartReadiness(t, ctxWithUser, participantClient, characterClient, campaignID, ownerParticipantID)
	characters := listAllCharactersForReadiness(t, ctxWithUser, characterClient, campaignID)
	if len(characters) == 0 {
		t.Fatal("expected at least one character for rest participants")
	}
	participants := make([]*daggerheartv1.DaggerheartRestParticipant, 0, len(characters))
	for _, ch := range characters {
		if ch.GetId() == "" {
			continue
		}
		participants = append(participants, &daggerheartv1.DaggerheartRestParticipant{CharacterId: ch.GetId()})
	}
	if len(participants) == 0 {
		t.Fatal("expected non-empty rest participants")
	}

	startSession := startSessionWithDefaultControllers(t, ctxWithUser, sessionClient, characterClient, campaignID, "Rest Session")
	if startSession.GetSession() == nil {
		t.Fatal("expected session")
	}
	sessionID := startSession.GetSession().GetId()

	createCountdown, err := daggerheartClient.CreateCampaignCountdown(ctxWithUser, &daggerheartv1.DaggerheartCreateCampaignCountdownRequest{
		CampaignId:        campaignID,
		Name:              "Long-Term Countdown",
		Tone:              daggerheartv1.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_CONSEQUENCE,
		AdvancementPolicy: daggerheartv1.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_LONG_REST,
		StartingValue:     &daggerheartv1.DaggerheartCreateCampaignCountdownRequest_FixedStartingValue{FixedStartingValue: 4},
		LoopBehavior:      daggerheartv1.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE,
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

	partySize := len(participants)
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
	if expectedFear > daggerheartstate.GMFearMax {
		expectedFear = daggerheartstate.GMFearMax
	}

	sessionCtx := withSessionID(ctxWithUser, sessionID)
	resp, err := daggerheartClient.ApplyRest(sessionCtx, &daggerheartv1.DaggerheartApplyRestRequest{
		CampaignId: campaignID,
		Rest: &daggerheartv1.DaggerheartRestRequest{
			RestType:                    daggerheartv1.DaggerheartRestType_DAGGERHEART_REST_TYPE_LONG,
			Interrupted:                 false,
			LongRestCampaignCountdownId: countdownID,
			Participants:                participants,
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
	if got := resp.GetCountdownAdvances(); len(got) != 1 {
		t.Fatalf("countdown advances = %d, want 1", len(got))
	} else {
		advance := got[0]
		if advance.GetCountdownId() != countdownID {
			t.Fatalf("countdown advance id = %q, want %q", advance.GetCountdownId(), countdownID)
		}
		if advance.GetAdvancementPolicy() != daggerheartv1.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_LONG_REST {
			t.Fatalf("countdown advancement_policy = %s, want LONG_REST", advance.GetAdvancementPolicy())
		}
		if advance.GetRemainingBefore() != 4 || advance.GetRemainingAfter() != 3 || advance.GetAdvancedBy() != 1 {
			t.Fatalf("countdown advance remaining = %d -> %d by %d, want 4 -> 3 by 1", advance.GetRemainingBefore(), advance.GetRemainingAfter(), advance.GetAdvancedBy())
		}
	}

	if err := findCountdownUpdated(ctxWithUser, eventClient, campaignID, "", countdownID, 4, 3); err != nil {
		t.Fatalf("find countdown updated: %v", err)
	}
}
