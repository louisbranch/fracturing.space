package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/platform/id"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/policy"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	defaultListInvitesPageSize = 10
	maxListInvitesPageSize     = 10
)

// InviteService implements the game.v1.InviteService gRPC API.
type InviteService struct {
	campaignv1.UnimplementedInviteServiceServer
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
	authClient  authv1.AuthServiceClient
}

// NewInviteService creates an InviteService with default dependencies.
func NewInviteService(stores Stores) *InviteService {
	return &InviteService{
		stores:      stores,
		clock:       time.Now,
		idGenerator: id.NewID,
	}
}

// NewInviteServiceWithAuth creates an InviteService with an auth client.
func NewInviteServiceWithAuth(stores Stores, authClient authv1.AuthServiceClient) *InviteService {
	service := NewInviteService(stores)
	service.authClient = authClient
	return service
}

// CreateInvite creates a seat-targeted invite.
func (s *InviteService) CreateInvite(ctx context.Context, in *campaignv1.CreateInviteRequest) (*campaignv1.CreateInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "create invite request is required")
	}
	if s.stores.Invite == nil {
		return nil, status.Error(codes.Internal, "invite store is not configured")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Participant == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	participantID := strings.TrimSpace(in.GetParticipantId())
	if participantID == "" {
		return nil, status.Error(codes.InvalidArgument, "participant id is required")
	}

	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requirePolicy(ctx, s.stores, policy.ActionManageInvites, campaignRecord); err != nil {
		return nil, err
	}
	if _, err := s.stores.Participant.GetParticipant(ctx, campaignID, participantID); err != nil {
		return nil, handleDomainError(err)
	}

	created, err := invite.CreateInvite(invite.CreateInviteInput{
		CampaignID:             campaignID,
		ParticipantID:          participantID,
		RecipientUserID:        strings.TrimSpace(in.GetRecipientUserId()),
		CreatedByParticipantID: strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx)),
	}, s.clock, s.idGenerator)
	if err != nil {
		return nil, handleDomainError(err)
	}

	payload := event.InviteCreatedPayload{
		InviteID:               created.ID,
		ParticipantID:          created.ParticipantID,
		RecipientUserID:        created.RecipientUserID,
		Status:                 invite.StatusLabel(created.Status),
		CreatedByParticipantID: created.CreatedByParticipantID,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.clock().UTC(),
		Type:         event.TypeInviteCreated,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "invite",
		EntityID:     created.ID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := projection.Applier{Campaign: s.stores.Campaign, Invite: s.stores.Invite, ClaimIndex: s.stores.ClaimIndex}
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	inv, err := s.stores.Invite.GetInvite(ctx, created.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load invite: %v", err)
	}

	return &campaignv1.CreateInviteResponse{Invite: inviteToProto(inv)}, nil
}

// ClaimInvite claims a seat-targeted invite.
func (s *InviteService) ClaimInvite(ctx context.Context, in *campaignv1.ClaimInviteRequest) (*campaignv1.ClaimInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "claim invite request is required")
	}
	if s.stores.Invite == nil {
		return nil, status.Error(codes.Internal, "invite store is not configured")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Participant == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	inviteID := strings.TrimSpace(in.GetInviteId())
	if inviteID == "" {
		return nil, status.Error(codes.InvalidArgument, "invite id is required")
	}
	if strings.TrimSpace(in.GetJoinGrant()) == "" {
		return nil, status.Error(codes.InvalidArgument, "join grant is required")
	}
	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	inv, err := s.stores.Invite.GetInvite(ctx, inviteID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if inv.CampaignID != campaignID {
		return nil, status.Error(codes.InvalidArgument, "invite campaign does not match")
	}
	if recipient := strings.TrimSpace(inv.RecipientUserID); recipient != "" && recipient != userID {
		return nil, status.Error(codes.PermissionDenied, "invite recipient does not match")
	}
	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}

	config, err := invite.LoadJoinGrantConfigFromEnv(s.clock)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "join grant validation is not configured: %v", err)
	}
	claims, err := invite.ValidateJoinGrant(in.GetJoinGrant(), invite.JoinGrantExpectation{
		CampaignID: campaignID,
		InviteID:   inv.ID,
		UserID:     userID,
	}, config)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, status.Errorf(codes.Internal, "validate join grant: %v", err)
	}
	if s.stores.ClaimIndex != nil {
		claim, err := s.stores.ClaimIndex.GetParticipantClaim(ctx, campaignID, userID)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return nil, status.Errorf(codes.Internal, "load participant claim: %v", err)
		}
		if err == nil && claim.ParticipantID != inv.ParticipantID {
			return nil, handleDomainError(apperrors.WithMetadata(
				apperrors.CodeParticipantUserAlreadyClaimed,
				"participant user already claimed",
				map[string]string{
					"CampaignID": campaignID,
					"UserID":     userID,
				},
			))
		}
	}
	claimEvent, err := findInviteClaimByJTI(ctx, s.stores.Event, campaignID, claims.JWTID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "lookup join grant: %v", err)
	}
	if claimEvent != nil {
		var payload event.InviteClaimedPayload
		if err := json.Unmarshal(claimEvent.PayloadJSON, &payload); err != nil {
			return nil, status.Errorf(codes.Internal, "decode prior claim: %v", err)
		}
		if payload.InviteID != inv.ID || payload.UserID != userID {
			return nil, handleDomainError(apperrors.New(apperrors.CodeInviteJoinGrantUsed, "join grant already used"))
		}
		updatedInvite, err := s.stores.Invite.GetInvite(ctx, inv.ID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "load invite: %v", err)
		}
		updatedParticipant, err := s.stores.Participant.GetParticipant(ctx, campaignID, inv.ParticipantID)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "load participant: %v", err)
		}
		return &campaignv1.ClaimInviteResponse{
			Invite:      inviteToProto(updatedInvite),
			Participant: participantToProto(updatedParticipant),
		}, nil
	}
	if inv.Status == invite.StatusClaimed {
		return nil, status.Error(codes.FailedPrecondition, "invite already claimed")
	}
	if inv.Status == invite.StatusRevoked {
		return nil, status.Error(codes.FailedPrecondition, "invite already revoked")
	}

	seat, err := s.stores.Participant.GetParticipant(ctx, campaignID, inv.ParticipantID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if strings.TrimSpace(seat.UserID) != "" {
		return nil, status.Error(codes.FailedPrecondition, "participant already claimed")
	}

	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	boundPayload := event.ParticipantBoundPayload{
		ParticipantID: seat.ID,
		UserID:        userID,
	}
	boundJSON, err := json.Marshal(boundPayload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode participant payload: %v", err)
	}
	boundEvent, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.clock().UTC(),
		Type:         event.TypeParticipantBound,
		RequestID:    requestID,
		InvocationID: invocationID,
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "participant",
		EntityID:     seat.ID,
		PayloadJSON:  boundJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append participant bound event: %v", err)
	}

	claimedPayload := event.InviteClaimedPayload{
		InviteID:      inv.ID,
		ParticipantID: inv.ParticipantID,
		UserID:        userID,
		JWTID:         claims.JWTID,
	}
	claimedJSON, err := json.Marshal(claimedPayload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode invite payload: %v", err)
	}
	claimedEvent, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    s.clock().UTC(),
		Type:         event.TypeInviteClaimed,
		RequestID:    requestID,
		InvocationID: invocationID,
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "invite",
		EntityID:     inv.ID,
		PayloadJSON:  claimedJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append invite claimed event: %v", err)
	}

	applier := projection.Applier{Campaign: s.stores.Campaign, Participant: s.stores.Participant, Invite: s.stores.Invite, ClaimIndex: s.stores.ClaimIndex}
	if err := applier.Apply(ctx, boundEvent); err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return nil, handleDomainError(err)
		}
		return nil, status.Errorf(codes.Internal, "apply participant event: %v", err)
	}
	if err := applier.Apply(ctx, claimedEvent); err != nil {
		return nil, status.Errorf(codes.Internal, "apply invite event: %v", err)
	}

	updatedInvite, err := s.stores.Invite.GetInvite(ctx, inv.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load invite: %v", err)
	}
	updatedParticipant, err := s.stores.Participant.GetParticipant(ctx, campaignID, seat.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load participant: %v", err)
	}

	return &campaignv1.ClaimInviteResponse{
		Invite:      inviteToProto(updatedInvite),
		Participant: participantToProto(updatedParticipant),
	}, nil
}

// GetInvite returns an invite by ID.
func (s *InviteService) GetInvite(ctx context.Context, in *campaignv1.GetInviteRequest) (*campaignv1.GetInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get invite request is required")
	}
	if s.stores.Invite == nil {
		return nil, status.Error(codes.Internal, "invite store is not configured")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}

	inviteID := strings.TrimSpace(in.GetInviteId())
	if inviteID == "" {
		return nil, status.Error(codes.InvalidArgument, "invite id is required")
	}

	inv, err := s.stores.Invite.GetInvite(ctx, inviteID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	campaignRecord, err := s.stores.Campaign.Get(ctx, inv.CampaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requirePolicy(ctx, s.stores, policy.ActionManageInvites, campaignRecord); err != nil {
		return nil, err
	}

	return &campaignv1.GetInviteResponse{Invite: inviteToProto(inv)}, nil
}

// ListInvites returns a page of invites for a campaign.
func (s *InviteService) ListInvites(ctx context.Context, in *campaignv1.ListInvitesRequest) (*campaignv1.ListInvitesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list invites request is required")
	}
	if s.stores.Invite == nil {
		return nil, status.Error(codes.Internal, "invite store is not configured")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := requirePolicy(ctx, s.stores, policy.ActionManageInvites, campaignRecord); err != nil {
		return nil, err
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = defaultListInvitesPageSize
	}
	if pageSize > maxListInvitesPageSize {
		pageSize = maxListInvitesPageSize
	}

	page, err := s.stores.Invite.ListInvites(ctx, campaignID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list invites: %v", err)
	}

	response := &campaignv1.ListInvitesResponse{NextPageToken: page.NextPageToken}
	if len(page.Invites) == 0 {
		return response, nil
	}

	response.Invites = make([]*campaignv1.Invite, 0, len(page.Invites))
	for _, inv := range page.Invites {
		response.Invites = append(response.Invites, inviteToProto(inv))
	}

	return response, nil
}

// ListPendingInvites returns a page of pending invites for a campaign.
func (s *InviteService) ListPendingInvites(ctx context.Context, in *campaignv1.ListPendingInvitesRequest) (*campaignv1.ListPendingInvitesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list pending invites request is required")
	}
	if s.stores.Invite == nil {
		return nil, status.Error(codes.Internal, "invite store is not configured")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Participant == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	campaignRecord, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := requirePolicy(ctx, s.stores, policy.ActionManageInvites, campaignRecord); err != nil {
		return nil, err
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = defaultListInvitesPageSize
	}
	if pageSize > maxListInvitesPageSize {
		pageSize = maxListInvitesPageSize
	}

	page, err := s.stores.Invite.ListPendingInvites(ctx, campaignID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list pending invites: %v", err)
	}

	response := &campaignv1.ListPendingInvitesResponse{NextPageToken: page.NextPageToken}
	if len(page.Invites) == 0 {
		return response, nil
	}

	participants, err := s.stores.Participant.ListParticipantsByCampaign(ctx, campaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list participants: %v", err)
	}
	participantsByID := make(map[string]participant.Participant, len(participants))
	for _, p := range participants {
		participantsByID[p.ID] = p
	}

	userCache := make(map[string]*authv1.User)
	response.Invites = make([]*campaignv1.PendingInvite, 0, len(page.Invites))
	for _, inv := range page.Invites {
		seat, ok := participantsByID[inv.ParticipantID]
		if !ok {
			return nil, status.Errorf(codes.Internal, "participant seat not found: %s", inv.ParticipantID)
		}
		var createdByUser *authv1.User
		creatorID := strings.TrimSpace(inv.CreatedByParticipantID)
		if creatorID != "" {
			creator, ok := participantsByID[creatorID]
			if !ok {
				return nil, status.Errorf(codes.Internal, "creator participant not found: %s", creatorID)
			}
			creatorUserID := strings.TrimSpace(creator.UserID)
			if creatorUserID != "" {
				if s.authClient == nil {
					return nil, status.Error(codes.Internal, "auth client is not configured")
				}
				cached, ok := userCache[creatorUserID]
				if !ok {
					userResponse, err := s.authClient.GetUser(ctx, &authv1.GetUserRequest{UserId: creatorUserID})
					if err != nil {
						return nil, status.Errorf(codes.Internal, "get auth user: %v", err)
					}
					if userResponse == nil || userResponse.GetUser() == nil {
						return nil, status.Error(codes.Internal, "auth user response is missing")
					}
					cached = userResponse.GetUser()
					userCache[creatorUserID] = cached
				}
				createdByUser = cached
			}
		}

		response.Invites = append(response.Invites, &campaignv1.PendingInvite{
			Invite:        inviteToProto(inv),
			Participant:   participantToProto(seat),
			CreatedByUser: createdByUser,
		})
	}

	return response, nil
}

// ListPendingInvitesForUser returns a page of pending invites for the current user.
func (s *InviteService) ListPendingInvitesForUser(ctx context.Context, in *campaignv1.ListPendingInvitesForUserRequest) (*campaignv1.ListPendingInvitesForUserResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list pending invites for user request is required")
	}
	if s.stores.Invite == nil {
		return nil, status.Error(codes.Internal, "invite store is not configured")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Participant == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}

	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	if userID == "" {
		return nil, status.Error(codes.InvalidArgument, "user id is required")
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = defaultListInvitesPageSize
	}
	if pageSize > maxListInvitesPageSize {
		pageSize = maxListInvitesPageSize
	}

	page, err := s.stores.Invite.ListPendingInvitesForRecipient(ctx, userID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list pending invites for user: %v", err)
	}

	response := &campaignv1.ListPendingInvitesForUserResponse{NextPageToken: page.NextPageToken}
	if len(page.Invites) == 0 {
		return response, nil
	}

	campaignsByID := make(map[string]campaign.Campaign)
	participantsByID := make(map[string]participant.Participant)
	response.Invites = make([]*campaignv1.PendingUserInvite, 0, len(page.Invites))
	for _, inv := range page.Invites {
		campaignRecord, ok := campaignsByID[inv.CampaignID]
		if !ok {
			record, err := s.stores.Campaign.Get(ctx, inv.CampaignID)
			if err != nil {
				return nil, handleDomainError(err)
			}
			campaignRecord = record
			campaignsByID[inv.CampaignID] = campaignRecord
		}

		participantKey := inv.CampaignID + ":" + inv.ParticipantID
		seat, ok := participantsByID[participantKey]
		if !ok {
			record, err := s.stores.Participant.GetParticipant(ctx, inv.CampaignID, inv.ParticipantID)
			if err != nil {
				return nil, handleDomainError(err)
			}
			seat = record
			participantsByID[participantKey] = seat
		}

		response.Invites = append(response.Invites, &campaignv1.PendingUserInvite{
			Invite:      inviteToProto(inv),
			Campaign:    campaignToProto(campaignRecord),
			Participant: participantToProto(seat),
		})
	}

	return response, nil
}

// RevokeInvite revokes an invite.
func (s *InviteService) RevokeInvite(ctx context.Context, in *campaignv1.RevokeInviteRequest) (*campaignv1.RevokeInviteResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "revoke invite request is required")
	}
	if s.stores.Invite == nil {
		return nil, status.Error(codes.Internal, "invite store is not configured")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "event store is not configured")
	}

	inviteID := strings.TrimSpace(in.GetInviteId())
	if inviteID == "" {
		return nil, status.Error(codes.InvalidArgument, "invite id is required")
	}

	inv, err := s.stores.Invite.GetInvite(ctx, inviteID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	campaignRecord, err := s.stores.Campaign.Get(ctx, inv.CampaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := requirePolicy(ctx, s.stores, policy.ActionManageInvites, campaignRecord); err != nil {
		return nil, err
	}
	if inv.Status == invite.StatusRevoked {
		return nil, status.Error(codes.FailedPrecondition, "invite already revoked")
	}
	if inv.Status == invite.StatusClaimed {
		return nil, status.Error(codes.FailedPrecondition, "invite already claimed")
	}

	payload := event.InviteRevokedPayload{InviteID: inv.ID}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := s.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   inv.CampaignID,
		Timestamp:    s.clock().UTC(),
		Type:         event.TypeInviteRevoked,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		ActorType:    actorType,
		ActorID:      actorID,
		EntityType:   "invite",
		EntityID:     inv.ID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := projection.Applier{Campaign: s.stores.Campaign, Invite: s.stores.Invite, ClaimIndex: s.stores.ClaimIndex}
	if err := applier.Apply(ctx, stored); err != nil {
		return nil, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	updated, err := s.stores.Invite.GetInvite(ctx, inv.ID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "load invite: %v", err)
	}

	return &campaignv1.RevokeInviteResponse{Invite: inviteToProto(updated)}, nil
}

func findInviteClaimByJTI(ctx context.Context, store storage.EventStore, campaignID, jti string) (*event.Event, error) {
	if strings.TrimSpace(jti) == "" {
		return nil, nil
	}
	if store == nil {
		return nil, fmt.Errorf("event store is not configured")
	}

	var cursor uint64
	for {
		page, err := store.ListEventsPage(ctx, storage.ListEventsPageRequest{
			CampaignID:   campaignID,
			PageSize:     200,
			CursorSeq:    cursor,
			CursorDir:    "fwd",
			Descending:   false,
			FilterClause: "event_type = ?",
			FilterParams: []any{event.TypeInviteClaimed},
		})
		if err != nil {
			return nil, err
		}
		for i := range page.Events {
			evt := page.Events[i]
			if evt.Type != event.TypeInviteClaimed {
				continue
			}
			var payload event.InviteClaimedPayload
			if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
				return nil, err
			}
			if payload.JWTID == jti {
				return &evt, nil
			}
		}
		if !page.HasNextPage || len(page.Events) == 0 {
			return nil, nil
		}
		cursor = page.Events[len(page.Events)-1].Seq
	}
}
