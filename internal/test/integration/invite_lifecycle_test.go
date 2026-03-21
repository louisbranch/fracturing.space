//go:build integration

package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

// inviteLifecycleSuite bundles clients and state for invite integration tests.
type inviteLifecycleSuite struct {
	invite      invitev1.InviteServiceClient
	participant gamev1.ParticipantServiceClient
	campaign    gamev1.CampaignServiceClient
	authAddr    string
	ownerUserID string
}

func runInviteLifecycleTests(t *testing.T, fixture *suiteFixture) {
	t.Helper()

	inviteAddr := startInviteServer(t, fixture.grpcAddr, fixture.authAddr)

	ownerUserID := createAuthUser(t, fixture.authAddr, uniqueTestUsername(t, "invite-owner"))

	inviteConn, err := grpc.NewClient(
		inviteAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial invite gRPC: %v", err)
	}
	t.Cleanup(func() { _ = inviteConn.Close() })

	gameConn, err := grpc.NewClient(
		fixture.grpcAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		t.Fatalf("dial game gRPC: %v", err)
	}
	t.Cleanup(func() { _ = gameConn.Close() })

	suite := &inviteLifecycleSuite{
		invite:      invitev1.NewInviteServiceClient(inviteConn),
		participant: gamev1.NewParticipantServiceClient(gameConn),
		campaign:    gamev1.NewCampaignServiceClient(gameConn),
		authAddr:    fixture.authAddr,
		ownerUserID: ownerUserID,
	}

	t.Run("create unassigned invite", suite.testCreateUnassignedInvite)
	t.Run("create targeted invite", suite.testCreateTargetedInvite)
	t.Run("reject invite for bound seat", suite.testRejectInviteForBoundSeat)
	t.Run("reject self invite", suite.testRejectSelfInvite)
	t.Run("claim invite binds participant", suite.testClaimInviteBindsParticipant)
	t.Run("decline invite", suite.testDeclineInvite)
	t.Run("revoke invite", suite.testRevokeInvite)
	t.Run("reject duplicate recipient", suite.testRejectDuplicateRecipient)
}

// createCampaignWithPlayerSeat creates a campaign owned by the suite owner and
// returns the campaign ID and an unbound player participant ID.
func (s *inviteLifecycleSuite) createCampaignWithPlayerSeat(t *testing.T) (campaignID, participantID string) {
	t.Helper()
	ctx, cancel := context.WithTimeout(withUserID(context.Background(), s.ownerUserID), integrationTimeout())
	defer cancel()

	campaignResp, err := s.campaign.CreateCampaign(ctx, &gamev1.CreateCampaignRequest{
		Name:   "invite-test-" + t.Name(),
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	campaignID = campaignResp.GetCampaign().GetId()
	if campaignID == "" {
		t.Fatal("campaign ID is empty")
	}

	participantResp, err := s.participant.CreateParticipant(ctx, &gamev1.CreateParticipantRequest{
		CampaignId: campaignID,
		Name:       "Player Seat",
		Role:       gamev1.ParticipantRole_PLAYER,
		Controller: gamev1.Controller_CONTROLLER_HUMAN,
	})
	if err != nil {
		t.Fatalf("create participant: %v", err)
	}
	participantID = participantResp.GetParticipant().GetId()
	if participantID == "" {
		t.Fatal("participant ID is empty")
	}

	return campaignID, participantID
}

func (s *inviteLifecycleSuite) testCreateUnassignedInvite(t *testing.T) {
	campaignID, participantID := s.createCampaignWithPlayerSeat(t)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	resp, err := s.invite.CreateInvite(ctx, &invitev1.CreateInviteRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
	})
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	inv := resp.GetInvite()
	if inv == nil {
		t.Fatal("expected invite in response")
	}
	if inv.GetId() == "" {
		t.Fatal("expected non-empty invite ID")
	}
	if inv.GetStatus() != invitev1.InviteStatus_PENDING {
		t.Fatalf("expected PENDING, got %s", inv.GetStatus())
	}
	if inv.GetRecipientUserId() != "" {
		t.Fatalf("expected empty recipient, got %q", inv.GetRecipientUserId())
	}
}

func (s *inviteLifecycleSuite) testCreateTargetedInvite(t *testing.T) {
	campaignID, participantID := s.createCampaignWithPlayerSeat(t)
	recipientUserID := createAuthUser(t, s.authAddr, uniqueTestUsername(t, "invite-recipient"))

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	resp, err := s.invite.CreateInvite(ctx, &invitev1.CreateInviteRequest{
		CampaignId:      campaignID,
		ParticipantId:   participantID,
		RecipientUserId: recipientUserID,
	})
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	inv := resp.GetInvite()
	if inv == nil {
		t.Fatal("expected invite in response")
	}
	if inv.GetStatus() != invitev1.InviteStatus_PENDING {
		t.Fatalf("expected PENDING, got %s", inv.GetStatus())
	}
	if inv.GetRecipientUserId() != recipientUserID {
		t.Fatalf("expected recipient %q, got %q", recipientUserID, inv.GetRecipientUserId())
	}
}

func (s *inviteLifecycleSuite) testRejectInviteForBoundSeat(t *testing.T) {
	ctx, cancel := context.WithTimeout(withUserID(context.Background(), s.ownerUserID), integrationTimeout())
	defer cancel()

	campaignResp, err := s.campaign.CreateCampaign(ctx, &gamev1.CreateCampaignRequest{
		Name:   "invite-bound-test-" + t.Name(),
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	})
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}
	campaignID := campaignResp.GetCampaign().GetId()

	// The owner participant is automatically created and bound to the owner
	// user. Find it.
	listResp, err := s.participant.ListParticipants(ctx, &gamev1.ListParticipantsRequest{
		CampaignId: campaignID,
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("list participants: %v", err)
	}

	var ownerParticipantID string
	for _, p := range listResp.GetParticipants() {
		if strings.TrimSpace(p.GetUserId()) != "" {
			ownerParticipantID = p.GetId()
			break
		}
	}
	if ownerParticipantID == "" {
		t.Fatal("expected to find owner participant with user_id set")
	}

	// Creating an invite for a bound seat should fail.
	createCtx, createCancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer createCancel()

	_, err = s.invite.CreateInvite(createCtx, &invitev1.CreateInviteRequest{
		CampaignId:    campaignID,
		ParticipantId: ownerParticipantID,
	})
	if err == nil {
		t.Fatal("expected error for bound seat, got nil")
	}
	if st, ok := status.FromError(err); !ok || st.Code() != codes.AlreadyExists {
		t.Fatalf("expected AlreadyExists, got %v", err)
	}
}

func (s *inviteLifecycleSuite) testRejectSelfInvite(t *testing.T) {
	campaignID, participantID := s.createCampaignWithPlayerSeat(t)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	// The owner already has a bound seat in the campaign. Creating a targeted
	// invite for the owner should fail.
	_, err := s.invite.CreateInvite(ctx, &invitev1.CreateInviteRequest{
		CampaignId:      campaignID,
		ParticipantId:   participantID,
		RecipientUserId: s.ownerUserID,
	})
	if err == nil {
		t.Fatal("expected error for self-invite, got nil")
	}
	if st, ok := status.FromError(err); !ok || st.Code() != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", err)
	}
}

func (s *inviteLifecycleSuite) testClaimInviteBindsParticipant(t *testing.T) {
	campaignID, participantID := s.createCampaignWithPlayerSeat(t)
	claimerUserID := createAuthUser(t, s.authAddr, uniqueTestUsername(t, "invite-claimer"))

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	// Create invite.
	createResp, err := s.invite.CreateInvite(ctx, &invitev1.CreateInviteRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
	})
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	inviteID := createResp.GetInvite().GetId()

	// Claim invite with join grant.
	token := joinGrantToken(t, campaignID, inviteID, claimerUserID, time.Now().UTC())
	claimCtx := withUserID(ctx, claimerUserID)
	claimResp, err := s.invite.ClaimInvite(claimCtx, &invitev1.ClaimInviteRequest{
		CampaignId: campaignID,
		InviteId:   inviteID,
		JoinGrant:  token,
	})
	if err != nil {
		t.Fatalf("ClaimInvite: %v", err)
	}

	if claimResp.GetInvite().GetStatus() != invitev1.InviteStatus_CLAIMED {
		t.Fatalf("expected CLAIMED, got %s", claimResp.GetInvite().GetStatus())
	}

	// Verify participant is now bound in game service.
	gameCtx := withUserID(ctx, claimerUserID)
	getResp, err := s.participant.GetParticipant(gameCtx, &gamev1.GetParticipantRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
	})
	if err != nil {
		t.Fatalf("GetParticipant: %v", err)
	}
	if got := strings.TrimSpace(getResp.GetParticipant().GetUserId()); got != claimerUserID {
		t.Fatalf("expected participant user_id %q, got %q", claimerUserID, got)
	}
}

func (s *inviteLifecycleSuite) testDeclineInvite(t *testing.T) {
	campaignID, participantID := s.createCampaignWithPlayerSeat(t)
	claimerUserID := createAuthUser(t, s.authAddr, uniqueTestUsername(t, "invite-decliner"))

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	// Create invite.
	createResp, err := s.invite.CreateInvite(ctx, &invitev1.CreateInviteRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
	})
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	inviteID := createResp.GetInvite().GetId()

	// Decline invite.
	declineResp, err := s.invite.DeclineInvite(ctx, &invitev1.DeclineInviteRequest{
		InviteId: inviteID,
	})
	if err != nil {
		t.Fatalf("DeclineInvite: %v", err)
	}
	if declineResp.GetInvite().GetStatus() != invitev1.InviteStatus_DECLINED {
		t.Fatalf("expected DECLINED, got %s", declineResp.GetInvite().GetStatus())
	}

	// Subsequent claim should fail.
	token := joinGrantToken(t, campaignID, inviteID, claimerUserID, time.Now().UTC())
	claimCtx := withUserID(ctx, claimerUserID)
	_, err = s.invite.ClaimInvite(claimCtx, &invitev1.ClaimInviteRequest{
		CampaignId: campaignID,
		InviteId:   inviteID,
		JoinGrant:  token,
	})
	if err == nil {
		t.Fatal("expected error claiming declined invite, got nil")
	}
	if st, ok := status.FromError(err); !ok || st.Code() != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", err)
	}
}

func (s *inviteLifecycleSuite) testRevokeInvite(t *testing.T) {
	campaignID, participantID := s.createCampaignWithPlayerSeat(t)
	claimerUserID := createAuthUser(t, s.authAddr, uniqueTestUsername(t, "invite-revoker"))

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	// Create invite.
	createResp, err := s.invite.CreateInvite(ctx, &invitev1.CreateInviteRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
	})
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	inviteID := createResp.GetInvite().GetId()

	// Revoke invite.
	revokeResp, err := s.invite.RevokeInvite(ctx, &invitev1.RevokeInviteRequest{
		InviteId: inviteID,
	})
	if err != nil {
		t.Fatalf("RevokeInvite: %v", err)
	}
	if revokeResp.GetInvite().GetStatus() != invitev1.InviteStatus_REVOKED {
		t.Fatalf("expected REVOKED, got %s", revokeResp.GetInvite().GetStatus())
	}

	// Subsequent claim should fail.
	token := joinGrantToken(t, campaignID, inviteID, claimerUserID, time.Now().UTC())
	claimCtx := withUserID(ctx, claimerUserID)
	_, err = s.invite.ClaimInvite(claimCtx, &invitev1.ClaimInviteRequest{
		CampaignId: campaignID,
		InviteId:   inviteID,
		JoinGrant:  token,
	})
	if err == nil {
		t.Fatal("expected error claiming revoked invite, got nil")
	}
	if st, ok := status.FromError(err); !ok || st.Code() != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", err)
	}
}

func (s *inviteLifecycleSuite) testRejectDuplicateRecipient(t *testing.T) {
	campaignID, participantID := s.createCampaignWithPlayerSeat(t)
	recipientUserID := createAuthUser(t, s.authAddr, uniqueTestUsername(t, "invite-dup"))

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	// Create first invite and claim it.
	createResp, err := s.invite.CreateInvite(ctx, &invitev1.CreateInviteRequest{
		CampaignId:      campaignID,
		ParticipantId:   participantID,
		RecipientUserId: recipientUserID,
	})
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	inviteID := createResp.GetInvite().GetId()

	token := joinGrantToken(t, campaignID, inviteID, recipientUserID, time.Now().UTC())
	claimCtx := withUserID(ctx, recipientUserID)
	_, err = s.invite.ClaimInvite(claimCtx, &invitev1.ClaimInviteRequest{
		CampaignId: campaignID,
		InviteId:   inviteID,
		JoinGrant:  token,
	})
	if err != nil {
		t.Fatalf("ClaimInvite: %v", err)
	}

	// Create a second player seat.
	ownerCtx := withUserID(ctx, s.ownerUserID)
	participant2Resp, err := s.participant.CreateParticipant(ownerCtx, &gamev1.CreateParticipantRequest{
		CampaignId: campaignID,
		Name:       "Player Seat 2",
		Role:       gamev1.ParticipantRole_PLAYER,
		Controller: gamev1.Controller_CONTROLLER_HUMAN,
	})
	if err != nil {
		t.Fatalf("create second participant: %v", err)
	}
	participant2ID := participant2Resp.GetParticipant().GetId()

	// Creating an invite for the same recipient should fail.
	_, err = s.invite.CreateInvite(ctx, &invitev1.CreateInviteRequest{
		CampaignId:      campaignID,
		ParticipantId:   participant2ID,
		RecipientUserId: recipientUserID,
	})
	if err == nil {
		t.Fatal("expected error for duplicate recipient, got nil")
	}
	if st, ok := status.FromError(err); !ok || st.Code() != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", err)
	}
}
