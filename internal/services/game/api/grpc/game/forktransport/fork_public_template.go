package forktransport

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type publicForkShape struct {
	ownerSeat storage.ParticipantRecord
}

func (a forkApplication) validatePublicForkShape(ctx context.Context, sourceCampaign storage.CampaignRecord) (publicForkShape, error) {
	participants, err := a.stores.Participant.ListParticipantsByCampaign(ctx, sourceCampaign.ID)
	if err != nil {
		return publicForkShape{}, fmt.Errorf("list participants: %w", err)
	}
	if len(participants) == 0 {
		return publicForkShape{}, status.Error(codes.FailedPrecondition, "public fork source must include participants")
	}

	var ownerSeat storage.ParticipantRecord
	ownerSeatFound := false
	for _, record := range participants {
		if record.Controller == participant.ControllerAI {
			if strings.TrimSpace(record.UserID) != "" {
				return publicForkShape{}, status.Error(codes.FailedPrecondition, "public fork source contains ai seat bound to a user")
			}
			continue
		}
		if strings.TrimSpace(record.UserID) != "" {
			return publicForkShape{}, status.Error(codes.FailedPrecondition, "public fork source contains bound human participant seats")
		}
		if record.CampaignAccess != participant.CampaignAccessOwner {
			return publicForkShape{}, status.Error(codes.FailedPrecondition, "public fork source contains additional human participant seats")
		}
		if ownerSeatFound {
			return publicForkShape{}, status.Error(codes.FailedPrecondition, "public fork source must contain exactly one human owner seat")
		}
		ownerSeat = record
		ownerSeatFound = true
	}
	if !ownerSeatFound {
		return publicForkShape{}, status.Error(codes.FailedPrecondition, "public fork source must contain one human owner seat")
	}
	return publicForkShape{ownerSeat: ownerSeat}, nil
}

func (a forkApplication) reassignForkedPublicOwnerSeat(
	ctx context.Context,
	forkCampaignID string,
	shape publicForkShape,
) error {
	userID := strings.TrimSpace(grpcmeta.UserIDFromContext(ctx))
	if userID == "" {
		return status.Error(codes.Unauthenticated, "authenticated user is required to fork public campaigns")
	}

	payloadJSON, err := json.Marshal(participant.SeatReassignPayload{
		ParticipantID: ids.ParticipantID(shape.ownerSeat.ID),
		PriorUserID:   ids.UserID(""),
		UserID:        ids.UserID(userID),
		Reason:        "public_fork_claim",
	})
	if err != nil {
		return status.Error(codes.Internal, "encode seat reassignment payload")
	}

	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	_, err = handler.ExecuteAndApplyDomainCommand(
		ctx,
		a.write,
		a.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   forkCampaignID,
			Type:         handler.CommandTypeParticipantSeatReassign,
			ActorType:    command.ActorTypeSystem,
			ActorID:      "",
			RequestID:    requestID,
			InvocationID: invocationID,
			EntityType:   "participant",
			EntityID:     shape.ownerSeat.ID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{
			ApplyErr: handler.ApplyErrorWithCodePreserve("apply participant seat event"),
		},
	)
	if err != nil {
		return err
	}

	applyParticipantProfileSnapshot(
		ctx,
		a.write,
		a.applier,
		a.stores.Participant,
		a.stores.Character,
		a.stores.Social,
		forkCampaignID,
		shape.ownerSeat.ID,
		userID,
		requestID,
		invocationID,
		"",
		command.ActorTypeSystem,
	)

	return nil
}

func requiresPublicForkSeatReassignment(campaignRecord storage.CampaignRecord) bool {
	return campaignRecord.AccessPolicy == campaign.AccessPolicyPublic
}
