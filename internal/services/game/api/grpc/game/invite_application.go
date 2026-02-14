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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/policy"
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

func (a inviteApplication) CreateInvite(ctx context.Context, campaignID string, in *campaignv1.CreateInviteRequest) (invite.Invite, error) {
	participantID := strings.TrimSpace(in.GetParticipantId())
	if participantID == "" {
		return invite.Invite{}, status.Error(codes.InvalidArgument, "participant id is required")
	}
	recipientUserID := strings.TrimSpace(in.GetRecipientUserId())

	campaignRecord, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return invite.Invite{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return invite.Invite{}, err
	}
	if err := requirePolicy(ctx, a.stores, policy.ActionManageInvites, campaignRecord); err != nil {
		return invite.Invite{}, err
	}
	if _, err := a.stores.Participant.GetParticipant(ctx, campaignID, participantID); err != nil {
		return invite.Invite{}, err
	}
	if recipientUserID != "" {
		if a.authClient == nil {
			return invite.Invite{}, status.Error(codes.Internal, "auth client is not configured")
		}
		userResponse, err := a.authClient.GetUser(ctx, &authv1.GetUserRequest{UserId: recipientUserID})
		if err != nil {
			if statusErr, ok := status.FromError(err); ok && statusErr.Code() == codes.NotFound {
				return invite.Invite{}, apperrors.New(
					apperrors.CodeInviteRecipientUserMissing,
					"invite recipient user not found",
				)
			}
			return invite.Invite{}, status.Errorf(codes.Internal, "get auth user: %v", err)
		}
		if userResponse == nil || userResponse.GetUser() == nil {
			return invite.Invite{}, status.Error(codes.Internal, "auth user response is missing")
		}
	}

	created, err := invite.CreateInvite(invite.CreateInviteInput{
		CampaignID:             campaignID,
		ParticipantID:          participantID,
		RecipientUserID:        recipientUserID,
		CreatedByParticipantID: strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx)),
	}, a.clock, a.idGenerator)
	if err != nil {
		return invite.Invite{}, err
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
		return invite.Invite{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := a.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    a.clock().UTC(),
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
		return invite.Invite{}, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := a.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return invite.Invite{}, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	inv, err := a.stores.Invite.GetInvite(ctx, created.ID)
	if err != nil {
		return invite.Invite{}, status.Errorf(codes.Internal, "load invite: %v", err)
	}

	return inv, nil
}

func (a inviteApplication) ClaimInvite(ctx context.Context, campaignID string, in *campaignv1.ClaimInviteRequest) (invite.Invite, participant.Participant, error) {
	inviteID := strings.TrimSpace(in.GetInviteId())
	if inviteID == "" {
		return invite.Invite{}, participant.Participant{}, status.Error(codes.InvalidArgument, "invite id is required")
	}
	if strings.TrimSpace(in.GetJoinGrant()) == "" {
		return invite.Invite{}, participant.Participant{}, status.Error(codes.InvalidArgument, "join grant is required")
	}
	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	if userID == "" {
		return invite.Invite{}, participant.Participant{}, status.Error(codes.InvalidArgument, "user id is required")
	}

	inv, err := a.stores.Invite.GetInvite(ctx, inviteID)
	if err != nil {
		return invite.Invite{}, participant.Participant{}, err
	}
	if inv.CampaignID != campaignID {
		return invite.Invite{}, participant.Participant{}, status.Error(codes.InvalidArgument, "invite campaign does not match")
	}
	if recipient := strings.TrimSpace(inv.RecipientUserID); recipient != "" && recipient != userID {
		return invite.Invite{}, participant.Participant{}, status.Error(codes.PermissionDenied, "invite recipient does not match")
	}
	campaignRecord, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return invite.Invite{}, participant.Participant{}, err
	}
	if err := campaign.ValidateCampaignOperation(campaignRecord.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return invite.Invite{}, participant.Participant{}, err
	}

	config, err := invite.LoadJoinGrantConfigFromEnv(a.clock)
	if err != nil {
		return invite.Invite{}, participant.Participant{}, status.Errorf(codes.Internal, "join grant validation is not configured: %v", err)
	}
	claims, err := invite.ValidateJoinGrant(in.GetJoinGrant(), invite.JoinGrantExpectation{
		CampaignID: campaignID,
		InviteID:   inv.ID,
		UserID:     userID,
	}, config)
	if err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return invite.Invite{}, participant.Participant{}, err
		}
		return invite.Invite{}, participant.Participant{}, status.Errorf(codes.Internal, "validate join grant: %v", err)
	}
	if a.stores.ClaimIndex != nil {
		claim, err := a.stores.ClaimIndex.GetParticipantClaim(ctx, campaignID, userID)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return invite.Invite{}, participant.Participant{}, status.Errorf(codes.Internal, "load participant claim: %v", err)
		}
		if err == nil && claim.ParticipantID != inv.ParticipantID {
			return invite.Invite{}, participant.Participant{}, apperrors.WithMetadata(
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
		return invite.Invite{}, participant.Participant{}, status.Errorf(codes.Internal, "lookup join grant: %v", err)
	}
	if claimEvent != nil {
		var payload event.InviteClaimedPayload
		if err := json.Unmarshal(claimEvent.PayloadJSON, &payload); err != nil {
			return invite.Invite{}, participant.Participant{}, status.Errorf(codes.Internal, "decode prior claim: %v", err)
		}
		if payload.InviteID != inv.ID || payload.UserID != userID {
			return invite.Invite{}, participant.Participant{}, apperrors.New(apperrors.CodeInviteJoinGrantUsed, "join grant already used")
		}
		updatedInvite, err := a.stores.Invite.GetInvite(ctx, inv.ID)
		if err != nil {
			return invite.Invite{}, participant.Participant{}, status.Errorf(codes.Internal, "load invite: %v", err)
		}
		updatedParticipant, err := a.stores.Participant.GetParticipant(ctx, campaignID, inv.ParticipantID)
		if err != nil {
			return invite.Invite{}, participant.Participant{}, status.Errorf(codes.Internal, "load participant: %v", err)
		}
		return updatedInvite, updatedParticipant, nil
	}
	if inv.Status == invite.StatusClaimed {
		return invite.Invite{}, participant.Participant{}, status.Error(codes.FailedPrecondition, "invite already claimed")
	}
	if inv.Status == invite.StatusRevoked {
		return invite.Invite{}, participant.Participant{}, status.Error(codes.FailedPrecondition, "invite already revoked")
	}

	seat, err := a.stores.Participant.GetParticipant(ctx, campaignID, inv.ParticipantID)
	if err != nil {
		return invite.Invite{}, participant.Participant{}, err
	}
	if strings.TrimSpace(seat.UserID) != "" {
		return invite.Invite{}, participant.Participant{}, status.Error(codes.FailedPrecondition, "participant already claimed")
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
		return invite.Invite{}, participant.Participant{}, status.Errorf(codes.Internal, "encode participant payload: %v", err)
	}
	boundEvent, err := a.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    a.clock().UTC(),
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
		return invite.Invite{}, participant.Participant{}, status.Errorf(codes.Internal, "append participant bound event: %v", err)
	}

	claimedPayload := event.InviteClaimedPayload{
		InviteID:      inv.ID,
		ParticipantID: inv.ParticipantID,
		UserID:        userID,
		JWTID:         claims.JWTID,
	}
	claimedJSON, err := json.Marshal(claimedPayload)
	if err != nil {
		return invite.Invite{}, participant.Participant{}, status.Errorf(codes.Internal, "encode invite payload: %v", err)
	}
	claimedEvent, err := a.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   campaignID,
		Timestamp:    a.clock().UTC(),
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
		return invite.Invite{}, participant.Participant{}, status.Errorf(codes.Internal, "append invite claimed event: %v", err)
	}

	applier := a.stores.Applier()
	if err := applier.Apply(ctx, boundEvent); err != nil {
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return invite.Invite{}, participant.Participant{}, err
		}
		return invite.Invite{}, participant.Participant{}, status.Errorf(codes.Internal, "apply participant event: %v", err)
	}
	if err := applier.Apply(ctx, claimedEvent); err != nil {
		return invite.Invite{}, participant.Participant{}, status.Errorf(codes.Internal, "apply invite event: %v", err)
	}

	updatedInvite, err := a.stores.Invite.GetInvite(ctx, inv.ID)
	if err != nil {
		return invite.Invite{}, participant.Participant{}, status.Errorf(codes.Internal, "load invite: %v", err)
	}
	updatedParticipant, err := a.stores.Participant.GetParticipant(ctx, campaignID, seat.ID)
	if err != nil {
		return invite.Invite{}, participant.Participant{}, status.Errorf(codes.Internal, "load participant: %v", err)
	}

	return updatedInvite, updatedParticipant, nil
}

func (a inviteApplication) RevokeInvite(ctx context.Context, in *campaignv1.RevokeInviteRequest) (invite.Invite, error) {
	inviteID := strings.TrimSpace(in.GetInviteId())
	if inviteID == "" {
		return invite.Invite{}, status.Error(codes.InvalidArgument, "invite id is required")
	}

	inv, err := a.stores.Invite.GetInvite(ctx, inviteID)
	if err != nil {
		return invite.Invite{}, err
	}
	campaignRecord, err := a.stores.Campaign.Get(ctx, inv.CampaignID)
	if err != nil {
		return invite.Invite{}, err
	}
	if err := requirePolicy(ctx, a.stores, policy.ActionManageInvites, campaignRecord); err != nil {
		return invite.Invite{}, err
	}
	if inv.Status == invite.StatusRevoked {
		return invite.Invite{}, status.Error(codes.FailedPrecondition, "invite already revoked")
	}
	if inv.Status == invite.StatusClaimed {
		return invite.Invite{}, status.Error(codes.FailedPrecondition, "invite already claimed")
	}

	payload := event.InviteRevokedPayload{InviteID: inv.ID}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return invite.Invite{}, status.Errorf(codes.Internal, "encode payload: %v", err)
	}

	actorID := grpcmeta.ParticipantIDFromContext(ctx)
	actorType := event.ActorTypeSystem
	if actorID != "" {
		actorType = event.ActorTypeParticipant
	}

	stored, err := a.stores.Event.AppendEvent(ctx, event.Event{
		CampaignID:   inv.CampaignID,
		Timestamp:    a.clock().UTC(),
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
		return invite.Invite{}, status.Errorf(codes.Internal, "append event: %v", err)
	}

	applier := a.stores.Applier()
	if err := applier.Apply(ctx, stored); err != nil {
		return invite.Invite{}, status.Errorf(codes.Internal, "apply event: %v", err)
	}

	updated, err := a.stores.Invite.GetInvite(ctx, inv.ID)
	if err != nil {
		return invite.Invite{}, status.Errorf(codes.Internal, "load invite: %v", err)
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
