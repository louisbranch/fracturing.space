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
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type inviteApplication struct {
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
	authClient  authv1.AuthServiceClient
}

func newInviteApplication(service *InviteService) inviteApplication {
	app := inviteApplication{stores: service.stores, clock: service.clock, idGenerator: service.idGenerator, authClient: service.authClient}
	if app.clock == nil {
		app.clock = time.Now
	}
	return app
}

func (a inviteApplication) CreateInvite(ctx context.Context, campaignID string, in *campaignv1.CreateInviteRequest) (storage.InviteRecord, error) {
	participantID := strings.TrimSpace(in.GetParticipantId())
	if participantID == "" {
		return storage.InviteRecord{}, status.Error(codes.InvalidArgument, "participant id is required")
	}
	recipientUserID := strings.TrimSpace(in.GetRecipientUserId())

	campaignRecord, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.InviteRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.InviteRecord{}, err
	}
	if err := requirePolicy(ctx, a.stores, policyActionManageInvites, campaignRecord); err != nil {
		return storage.InviteRecord{}, err
	}
	if _, err := a.stores.Participant.GetParticipant(ctx, campaignID, participantID); err != nil {
		return storage.InviteRecord{}, err
	}
	if recipientUserID != "" {
		if a.authClient == nil {
			return storage.InviteRecord{}, status.Error(codes.Internal, "auth client is not configured")
		}
		userResponse, err := a.authClient.GetUser(ctx, &authv1.GetUserRequest{UserId: recipientUserID})
		if err != nil {
			if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.NotFound {
				return storage.InviteRecord{}, apperrors.New(
					apperrors.CodeInviteRecipientUserMissing,
					"invite recipient user not found",
				)
			}
			return storage.InviteRecord{}, status.Errorf(codes.Internal, "get auth user: %v", err)
		}
		if userResponse == nil || userResponse.GetUser() == nil {
			return storage.InviteRecord{}, status.Error(codes.Internal, "auth user response is missing")
		}
	}

	inviteID, err := a.idGenerator()
	if err != nil {
		return storage.InviteRecord{}, status.Errorf(codes.Internal, "generate invite id: %v", err)
	}

	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	actorID := grpcmeta.ParticipantIDFromContext(ctx)

	applier := a.stores.Applier()
	if a.stores.Domain == nil {
		return storage.InviteRecord{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	payload := invite.CreatePayload{
		InviteID:               inviteID,
		ParticipantID:          participantID,
		RecipientUserID:        recipientUserID,
		CreatedByParticipantID: strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx)),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.InviteRecord{}, status.Errorf(codes.Internal, "encode invite payload: %v", err)
	}
	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}
	result, err := a.stores.Domain.Execute(ctx, command.Command{
		CampaignID:   campaignID,
		Type:         command.Type("invite.create"),
		ActorType:    actorType,
		ActorID:      actorID,
		RequestID:    requestID,
		InvocationID: invocationID,
		EntityType:   "invite",
		EntityID:     inviteID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return storage.InviteRecord{}, status.Errorf(codes.Internal, "execute domain command: %v", err)
	}
	if len(result.Decision.Rejections) > 0 {
		return storage.InviteRecord{}, status.Error(codes.FailedPrecondition, result.Decision.Rejections[0].Message)
	}
	for _, evt := range result.Decision.Events {
		if err := applier.Apply(ctx, evt); err != nil {
			if apperrors.GetCode(err) != apperrors.CodeUnknown {
				return storage.InviteRecord{}, err
			}
			return storage.InviteRecord{}, status.Errorf(codes.Internal, "apply invite event: %v", err)
		}
	}

	inv, err := a.stores.Invite.GetInvite(ctx, inviteID)
	if err != nil {
		return storage.InviteRecord{}, status.Errorf(codes.Internal, "load invite: %v", err)
	}

	return inv, nil
}

func (a inviteApplication) ClaimInvite(ctx context.Context, campaignID string, in *campaignv1.ClaimInviteRequest) (storage.InviteRecord, storage.ParticipantRecord, error) {
	inviteID := strings.TrimSpace(in.GetInviteId())
	if inviteID == "" {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.InvalidArgument, "invite id is required")
	}
	if strings.TrimSpace(in.GetJoinGrant()) == "" {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.InvalidArgument, "join grant is required")
	}
	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	if userID == "" {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.InvalidArgument, "user id is required")
	}

	inv, err := a.stores.Invite.GetInvite(ctx, inviteID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}
	if inv.CampaignID != campaignID {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.InvalidArgument, "invite campaign does not match")
	}
	if recipient := strings.TrimSpace(inv.RecipientUserID); recipient != "" && recipient != userID {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.PermissionDenied, "invite recipient does not match")
	}
	campaignRecord, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}

	config, err := invite.LoadJoinGrantConfigFromEnv(a.clock)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "join grant validation is not configured: %v", err)
	}
	claims, err := invite.ValidateJoinGrant(in.GetJoinGrant(), invite.JoinGrantExpectation{
		CampaignID: campaignID,
		InviteID:   inv.ID,
		UserID:     userID,
	}, config)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, err
		}
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "validate join grant: %v", err)
	}
	if a.stores.ClaimIndex != nil {
		claim, err := a.stores.ClaimIndex.GetParticipantClaim(ctx, campaignID, userID)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "load participant claim: %v", err)
		}
		if err == nil && claim.ParticipantID != inv.ParticipantID {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, apperrors.WithMetadata(
				apperrors.CodeParticipantUserAlreadyClaimed,
				"participant user already claimed",
				map[string]string{
					"CampaignID": campaignID,
					"UserID":     userID,
				},
			)
		}
	}
	claimEvent, err := findInviteClaimByJTI(ctx, a.stores.Event, campaignID, claims.JWTID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "lookup join grant: %v", err)
	}
	if claimEvent != nil {
		var payload invite.ClaimPayload
		if err := json.Unmarshal(claimEvent.PayloadJSON, &payload); err != nil {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "decode prior claim: %v", err)
		}
		if payload.InviteID != inv.ID || payload.UserID != userID {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, apperrors.New(apperrors.CodeInviteJoinGrantUsed, "join grant already used")
		}
		updatedInvite, err := a.stores.Invite.GetInvite(ctx, inv.ID)
		if err != nil {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "load invite: %v", err)
		}
		updatedParticipant, err := a.stores.Participant.GetParticipant(ctx, campaignID, inv.ParticipantID)
		if err != nil {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "load participant: %v", err)
		}
		return updatedInvite, updatedParticipant, nil
	}
	if inv.Status == invite.StatusClaimed {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, "invite already claimed")
	}
	if inv.Status == invite.StatusRevoked {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, "invite already revoked")
	}

	seat, err := a.stores.Participant.GetParticipant(ctx, campaignID, inv.ParticipantID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}
	if strings.TrimSpace(seat.UserID) != "" {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, "participant already claimed")
	}

	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	actorID := grpcmeta.ParticipantIDFromContext(ctx)

	applier := a.stores.Applier()
	if a.stores.Domain == nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	payload := participant.BindPayload{
		ParticipantID: seat.ID,
		UserID:        userID,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "encode participant payload: %v", err)
	}

	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}
	result, err := a.stores.Domain.Execute(ctx, command.Command{
		CampaignID:   campaignID,
		Type:         command.Type("participant.bind"),
		ActorType:    actorType,
		ActorID:      actorID,
		RequestID:    requestID,
		InvocationID: invocationID,
		EntityType:   "participant",
		EntityID:     seat.ID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "execute domain command: %v", err)
	}
	if len(result.Decision.Rejections) > 0 {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, result.Decision.Rejections[0].Message)
	}
	for _, evt := range result.Decision.Events {
		if err := applier.Apply(ctx, evt); err != nil {
			if apperrors.GetCode(err) != apperrors.CodeUnknown {
				return storage.InviteRecord{}, storage.ParticipantRecord{}, err
			}
			return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "apply participant event: %v", err)
		}
	}

	claimPayload := invite.ClaimPayload{
		InviteID:      inv.ID,
		ParticipantID: inv.ParticipantID,
		UserID:        userID,
		JWTID:         claims.JWTID,
	}
	claimJSON, err := json.Marshal(claimPayload)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "encode invite payload: %v", err)
	}
	actorType = command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}
	result, err = a.stores.Domain.Execute(ctx, command.Command{
		CampaignID:   campaignID,
		Type:         command.Type("invite.claim"),
		ActorType:    actorType,
		ActorID:      actorID,
		RequestID:    requestID,
		InvocationID: invocationID,
		EntityType:   "invite",
		EntityID:     inv.ID,
		PayloadJSON:  claimJSON,
	})
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "execute domain command: %v", err)
	}
	if len(result.Decision.Rejections) > 0 {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Error(codes.FailedPrecondition, result.Decision.Rejections[0].Message)
	}
	for _, evt := range result.Decision.Events {
		if err := applier.Apply(ctx, evt); err != nil {
			if apperrors.GetCode(err) != apperrors.CodeUnknown {
				return storage.InviteRecord{}, storage.ParticipantRecord{}, err
			}
			return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "apply invite event: %v", err)
		}
	}

	updatedInvite, err := a.stores.Invite.GetInvite(ctx, inv.ID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "load invite: %v", err)
	}
	updatedParticipant, err := a.stores.Participant.GetParticipant(ctx, campaignID, seat.ID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, status.Errorf(codes.Internal, "load participant: %v", err)
	}

	return updatedInvite, updatedParticipant, nil
}

func (a inviteApplication) RevokeInvite(ctx context.Context, in *campaignv1.RevokeInviteRequest) (storage.InviteRecord, error) {
	inviteID := strings.TrimSpace(in.GetInviteId())
	if inviteID == "" {
		return storage.InviteRecord{}, status.Error(codes.InvalidArgument, "invite id is required")
	}

	inv, err := a.stores.Invite.GetInvite(ctx, inviteID)
	if err != nil {
		return storage.InviteRecord{}, err
	}
	campaignRecord, err := a.stores.Campaign.Get(ctx, inv.CampaignID)
	if err != nil {
		return storage.InviteRecord{}, err
	}
	if err := requirePolicy(ctx, a.stores, policyActionManageInvites, campaignRecord); err != nil {
		return storage.InviteRecord{}, err
	}
	if inv.Status == invite.StatusRevoked {
		return storage.InviteRecord{}, status.Error(codes.FailedPrecondition, "invite already revoked")
	}
	if inv.Status == invite.StatusClaimed {
		return storage.InviteRecord{}, status.Error(codes.FailedPrecondition, "invite already claimed")
	}

	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	actorID := grpcmeta.ParticipantIDFromContext(ctx)

	applier := a.stores.Applier()
	if a.stores.Domain == nil {
		return storage.InviteRecord{}, status.Error(codes.Internal, "domain engine is not configured")
	}
	payload := invite.RevokePayload{InviteID: inv.ID}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.InviteRecord{}, status.Errorf(codes.Internal, "encode invite payload: %v", err)
	}
	actorType := command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}
	result, err := a.stores.Domain.Execute(ctx, command.Command{
		CampaignID:   inv.CampaignID,
		Type:         command.Type("invite.revoke"),
		ActorType:    actorType,
		ActorID:      actorID,
		RequestID:    requestID,
		InvocationID: invocationID,
		EntityType:   "invite",
		EntityID:     inv.ID,
		PayloadJSON:  payloadJSON,
	})
	if err != nil {
		return storage.InviteRecord{}, status.Errorf(codes.Internal, "execute domain command: %v", err)
	}
	if len(result.Decision.Rejections) > 0 {
		return storage.InviteRecord{}, status.Error(codes.FailedPrecondition, result.Decision.Rejections[0].Message)
	}
	for _, evt := range result.Decision.Events {
		if err := applier.Apply(ctx, evt); err != nil {
			if apperrors.GetCode(err) != apperrors.CodeUnknown {
				return storage.InviteRecord{}, err
			}
			return storage.InviteRecord{}, status.Errorf(codes.Internal, "apply invite event: %v", err)
		}
	}

	updated, err := a.stores.Invite.GetInvite(ctx, inv.ID)
	if err != nil {
		return storage.InviteRecord{}, status.Errorf(codes.Internal, "load invite: %v", err)
	}

	return updated, nil
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
			FilterParams: []any{string(event.Type("invite.claimed"))},
		})
		if err != nil {
			return nil, err
		}
		for i := range page.Events {
			evt := page.Events[i]
			if evt.Type != event.Type("invite.claimed") {
				continue
			}
			var payload invite.ClaimPayload
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
