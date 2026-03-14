package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/shared/joingrant"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a inviteApplication) ClaimInvite(ctx context.Context, campaignID string, in *campaignv1.ClaimInviteRequest) (storage.InviteRecord, storage.ParticipantRecord, error) {
	inviteID, err := validate.RequiredID(in.GetInviteId(), "invite id")
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}
	if _, err := validate.RequiredID(in.GetJoinGrant(), "join grant"); err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}
	userID, err := validate.RequiredID(grpcmeta.UserIDFromContext(ctx), "user id")
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
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

	claims, err := a.joinGrantVerifier.Validate(in.GetJoinGrant(), joingrant.Expectation{
		CampaignID: campaignID,
		InviteID:   inv.ID,
		UserID:     userID,
	})
	if err != nil {
		if errors.Is(err, joingrant.ErrVerifierNotConfigured) {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("join grant validation is not configured", err)
		}
		if apperrors.GetCode(err) != apperrors.CodeUnknown {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, err
		}
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("validate join grant", err)
	}
	if a.stores.ClaimIndex != nil {
		claim, err := a.stores.ClaimIndex.GetParticipantClaim(ctx, campaignID, userID)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load participant claim", err)
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
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("lookup join grant", err)
	}
	if claimEvent != nil {
		var payload invite.ClaimPayload
		if err := json.Unmarshal(claimEvent.PayloadJSON, &payload); err != nil {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("decode prior claim", err)
		}
		if payload.InviteID != ids.InviteID(inv.ID) || payload.UserID != ids.UserID(userID) {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, apperrors.New(apperrors.CodeInviteJoinGrantUsed, "join grant already used")
		}
		updatedInvite, err := a.stores.Invite.GetInvite(ctx, inv.ID)
		if err != nil {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load invite", err)
		}
		updatedParticipant, err := a.stores.Participant.GetParticipant(ctx, campaignID, inv.ParticipantID)
		if err != nil {
			return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load participant", err)
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
	payload := participant.BindPayload{
		ParticipantID: ids.ParticipantID(seat.ID),
		UserID:        ids.UserID(userID),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("encode participant payload", err)
	}

	actorID, actorType := resolveCommandActor(ctx)
	_, err = executeAndApplyDomainCommand(
		ctx,
		a.write,
		a.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeParticipantBind,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    requestID,
			InvocationID: invocationID,
			EntityType:   "participant",
			EntityID:     seat.ID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{
			ApplyErr: domainApplyErrorWithCodePreserve("apply participant event"),
		},
	)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}

	claimPayload := invite.ClaimPayload{
		InviteID:      ids.InviteID(inv.ID),
		ParticipantID: ids.ParticipantID(inv.ParticipantID),
		UserID:        ids.UserID(userID),
		JWTID:         claims.JWTID,
	}
	claimJSON, err := json.Marshal(claimPayload)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("encode invite payload", err)
	}
	actorType = command.ActorTypeSystem
	if actorID != "" {
		actorType = command.ActorTypeParticipant
	}
	_, err = executeAndApplyDomainCommand(
		ctx,
		a.write,
		a.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeInviteClaim,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    requestID,
			InvocationID: invocationID,
			EntityType:   "invite",
			EntityID:     inv.ID,
			PayloadJSON:  claimJSON,
		}),
		domainwrite.Options{
			ApplyErr: domainApplyErrorWithCodePreserve("apply invite event"),
		},
	)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, err
	}

	updatedInvite, err := a.stores.Invite.GetInvite(ctx, inv.ID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load invite", err)
	}
	updatedParticipant, err := a.stores.Participant.GetParticipant(ctx, campaignID, seat.ID)
	if err != nil {
		return storage.InviteRecord{}, storage.ParticipantRecord{}, grpcerror.Internal("load participant", err)
	}

	return updatedInvite, updatedParticipant, nil
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
			CampaignID: campaignID,
			PageSize:   200,
			CursorSeq:  cursor,
			CursorDir:  "fwd",
			Descending: false,
			Filter: storage.EventQueryFilter{
				EventType: string(eventTypeInviteClaimed),
			},
		})
		if err != nil {
			return nil, err
		}
		for i := range page.Events {
			evt := page.Events[i]
			if evt.Type != eventTypeInviteClaimed {
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
