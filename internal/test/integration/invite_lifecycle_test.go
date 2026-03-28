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
	t.Run("list pending invites", suite.testListPendingInvites)
	t.Run("claim with invalid grant", suite.testClaimWithInvalidGrant)
	t.Run("claim already claimed invite", suite.testClaimAlreadyClaimedInvite)
	t.Run("get invite", suite.testGetInvite)
	t.Run("list invites with status filter", suite.testListInvitesWithStatusFilter)
	t.Run("get public invite with enrichment", suite.testGetPublicInviteWithEnrichment)
	t.Run("outbox events after create", suite.testOutboxEventsAfterCreate)
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

func (s *inviteLifecycleSuite) testListPendingInvites(t *testing.T) {
	campaignID, participantID := s.createCampaignWithPlayerSeat(t)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	// Create two invites for the campaign (one per player seat).
	for i := 0; i < 2; i++ {
		_, err := s.invite.CreateInvite(ctx, &invitev1.CreateInviteRequest{
			CampaignId:    campaignID,
			ParticipantId: participantID,
		})
		if err != nil {
			t.Fatalf("CreateInvite %d: %v", i, err)
		}

		if i < 1 {
			ownerCtx := withUserID(ctx, s.ownerUserID)
			resp, err := s.participant.CreateParticipant(ownerCtx, &gamev1.CreateParticipantRequest{
				CampaignId: campaignID,
				Name:       "Extra Seat",
				Role:       gamev1.ParticipantRole_PLAYER,
				Controller: gamev1.Controller_CONTROLLER_HUMAN,
			})
			if err != nil {
				t.Fatalf("create extra participant: %v", err)
			}
			participantID = resp.GetParticipant().GetId()
		}
	}

	listResp, err := s.invite.ListPendingInvites(ctx, &invitev1.ListPendingInvitesRequest{
		CampaignId: campaignID,
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("ListPendingInvites: %v", err)
	}
	if len(listResp.GetInvites()) < 2 {
		t.Fatalf("expected at least 2 pending invites, got %d", len(listResp.GetInvites()))
	}
}

func (s *inviteLifecycleSuite) testClaimWithInvalidGrant(t *testing.T) {
	campaignID, participantID := s.createCampaignWithPlayerSeat(t)
	claimerUserID := createAuthUser(t, s.authAddr, uniqueTestUsername(t, "invite-bad-grant"))

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	createResp, err := s.invite.CreateInvite(ctx, &invitev1.CreateInviteRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
	})
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	inviteID := createResp.GetInvite().GetId()

	// Claim with a bogus token.
	claimCtx := withUserID(ctx, claimerUserID)
	_, err = s.invite.ClaimInvite(claimCtx, &invitev1.ClaimInviteRequest{
		CampaignId: campaignID,
		InviteId:   inviteID,
		JoinGrant:  "invalid.bogus.token",
	})
	if err == nil {
		t.Fatal("expected error for invalid grant, got nil")
	}
	if st, ok := status.FromError(err); !ok || st.Code() != codes.PermissionDenied {
		t.Fatalf("expected PermissionDenied, got %v", err)
	}
}

func (s *inviteLifecycleSuite) testClaimAlreadyClaimedInvite(t *testing.T) {
	campaignID, participantID := s.createCampaignWithPlayerSeat(t)
	claimer1 := createAuthUser(t, s.authAddr, uniqueTestUsername(t, "invite-claimer1"))
	claimer2 := createAuthUser(t, s.authAddr, uniqueTestUsername(t, "invite-claimer2"))

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	createResp, err := s.invite.CreateInvite(ctx, &invitev1.CreateInviteRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
	})
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	inviteID := createResp.GetInvite().GetId()

	// First claim succeeds.
	token1 := joinGrantToken(t, campaignID, inviteID, claimer1, time.Now().UTC())
	_, err = s.invite.ClaimInvite(withUserID(ctx, claimer1), &invitev1.ClaimInviteRequest{
		CampaignId: campaignID,
		InviteId:   inviteID,
		JoinGrant:  token1,
	})
	if err != nil {
		t.Fatalf("ClaimInvite first: %v", err)
	}

	// Second claim with different user fails — already claimed.
	token2 := joinGrantToken(t, campaignID, inviteID, claimer2, time.Now().UTC())
	_, err = s.invite.ClaimInvite(withUserID(ctx, claimer2), &invitev1.ClaimInviteRequest{
		CampaignId: campaignID,
		InviteId:   inviteID,
		JoinGrant:  token2,
	})
	if err == nil {
		t.Fatal("expected error for double claim, got nil")
	}
	if st, ok := status.FromError(err); !ok || st.Code() != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", err)
	}
}

func (s *inviteLifecycleSuite) testGetInvite(t *testing.T) {
	campaignID, participantID := s.createCampaignWithPlayerSeat(t)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	createResp, err := s.invite.CreateInvite(ctx, &invitev1.CreateInviteRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
	})
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	inviteID := createResp.GetInvite().GetId()

	getResp, err := s.invite.GetInvite(ctx, &invitev1.GetInviteRequest{
		InviteId: inviteID,
	})
	if err != nil {
		t.Fatalf("GetInvite: %v", err)
	}
	if getResp.GetInvite().GetId() != inviteID {
		t.Fatalf("GetInvite ID = %q, want %q", getResp.GetInvite().GetId(), inviteID)
	}
	if getResp.GetInvite().GetCampaignId() != campaignID {
		t.Fatalf("GetInvite CampaignId = %q, want %q", getResp.GetInvite().GetCampaignId(), campaignID)
	}
	if getResp.GetInvite().GetStatus() != invitev1.InviteStatus_PENDING {
		t.Fatalf("GetInvite Status = %v, want PENDING", getResp.GetInvite().GetStatus())
	}
}

func (s *inviteLifecycleSuite) testListInvitesWithStatusFilter(t *testing.T) {
	campaignID, participantID := s.createCampaignWithPlayerSeat(t)
	declineUserID := createAuthUser(t, s.authAddr, uniqueTestUsername(t, "invite-filter"))

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	// Create and decline an invite.
	createResp, err := s.invite.CreateInvite(ctx, &invitev1.CreateInviteRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
	})
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	_, err = s.invite.DeclineInvite(ctx, &invitev1.DeclineInviteRequest{
		InviteId: createResp.GetInvite().GetId(),
	})
	if err != nil {
		t.Fatalf("DeclineInvite: %v", err)
	}

	// Create a second seat and a pending invite.
	ownerCtx := withUserID(ctx, s.ownerUserID)
	seat2, err := s.participant.CreateParticipant(ownerCtx, &gamev1.CreateParticipantRequest{
		CampaignId: campaignID,
		Name:       "Seat 2",
		Role:       gamev1.ParticipantRole_PLAYER,
		Controller: gamev1.Controller_CONTROLLER_HUMAN,
	})
	if err != nil {
		t.Fatalf("create seat 2: %v", err)
	}
	_, err = s.invite.CreateInvite(ctx, &invitev1.CreateInviteRequest{
		CampaignId:      campaignID,
		ParticipantId:   seat2.GetParticipant().GetId(),
		RecipientUserId: declineUserID,
	})
	if err != nil {
		t.Fatalf("CreateInvite 2: %v", err)
	}

	// List with PENDING filter — should return only the second invite.
	listResp, err := s.invite.ListInvites(ctx, &invitev1.ListInvitesRequest{
		CampaignId: campaignID,
		Status:     invitev1.InviteStatus_PENDING,
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("ListInvites: %v", err)
	}
	if len(listResp.GetInvites()) != 1 {
		t.Fatalf("ListInvites(PENDING) = %d invites, want 1", len(listResp.GetInvites()))
	}

	// List with DECLINED filter — should return only the first.
	listResp, err = s.invite.ListInvites(ctx, &invitev1.ListInvitesRequest{
		CampaignId: campaignID,
		Status:     invitev1.InviteStatus_DECLINED,
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("ListInvites(DECLINED): %v", err)
	}
	if len(listResp.GetInvites()) != 1 {
		t.Fatalf("ListInvites(DECLINED) = %d invites, want 1", len(listResp.GetInvites()))
	}
}

func (s *inviteLifecycleSuite) testGetPublicInviteWithEnrichment(t *testing.T) {
	campaignID, participantID := s.createCampaignWithPlayerSeat(t)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	createResp, err := s.invite.CreateInvite(ctx, &invitev1.CreateInviteRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
	})
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}
	inviteID := createResp.GetInvite().GetId()

	pubResp, err := s.invite.GetPublicInvite(ctx, &invitev1.GetPublicInviteRequest{
		InviteId: inviteID,
	})
	if err != nil {
		t.Fatalf("GetPublicInvite: %v", err)
	}
	if pubResp.GetInvite() == nil {
		t.Fatal("GetPublicInvite invite is nil")
	}
	if pubResp.GetInvite().GetId() != inviteID {
		t.Fatalf("invite ID = %q, want %q", pubResp.GetInvite().GetId(), inviteID)
	}
	// Campaign enrichment: the invite service should fetch campaign details.
	if pubResp.GetCampaign() == nil {
		t.Fatal("GetPublicInvite campaign enrichment is nil")
	}
	if pubResp.GetCampaign().GetId() != campaignID {
		t.Fatalf("campaign ID = %q, want %q", pubResp.GetCampaign().GetId(), campaignID)
	}
	if pubResp.GetCampaign().GetName() == "" {
		t.Fatal("campaign name is empty")
	}
	// Participant enrichment: the invite service should fetch participant details.
	if pubResp.GetParticipant() == nil {
		t.Fatal("GetPublicInvite participant enrichment is nil")
	}
	if pubResp.GetParticipant().GetId() != participantID {
		t.Fatalf("participant ID = %q, want %q", pubResp.GetParticipant().GetId(), participantID)
	}
}

func (s *inviteLifecycleSuite) testOutboxEventsAfterCreate(t *testing.T) {
	campaignID, participantID := s.createCampaignWithPlayerSeat(t)

	ctx, cancel := context.WithTimeout(context.Background(), integrationTimeout())
	defer cancel()

	_, err := s.invite.CreateInvite(ctx, &invitev1.CreateInviteRequest{
		CampaignId:    campaignID,
		ParticipantId: participantID,
	})
	if err != nil {
		t.Fatalf("CreateInvite: %v", err)
	}

	// Lease outbox events — should have at least one invite.created event.
	leaseResp, err := s.invite.LeaseIntegrationOutboxEvents(ctx, &invitev1.LeaseIntegrationOutboxEventsRequest{
		Consumer:   "test-worker",
		Limit:      10,
		LeaseTtlMs: 30000,
	})
	if err != nil {
		t.Fatalf("LeaseIntegrationOutboxEvents: %v", err)
	}
	if len(leaseResp.GetEvents()) == 0 {
		t.Fatal("expected at least one outbox event after CreateInvite")
	}

	found := false
	for _, evt := range leaseResp.GetEvents() {
		if strings.Contains(evt.GetEventType(), "created") {
			found = true
			if evt.GetPayloadJson() == "" {
				t.Fatal("outbox event payload is empty")
			}
			break
		}
	}
	if !found {
		t.Fatal("expected invite.created outbox event")
	}
}
