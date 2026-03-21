package snapshottransport

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a snapshotApplication) UpdateSnapshotState(ctx context.Context, campaignID string, in *campaignv1.UpdateSnapshotStateRequest) (projectionstore.DaggerheartSnapshot, error) {
	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return projectionstore.DaggerheartSnapshot{}, err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return projectionstore.DaggerheartSnapshot{}, err
	}
	if err := authz.RequirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions(), c); err != nil {
		return projectionstore.DaggerheartSnapshot{}, err
	}

	// Handle Daggerheart snapshot update
	if dhUpdate := in.GetDaggerheart(); dhUpdate != nil {
		gmFear := int(dhUpdate.GetGmFear())
		if gmFear < daggerheart.GMFearMin || gmFear > daggerheart.GMFearMax {
			return projectionstore.DaggerheartSnapshot{}, status.Errorf(codes.InvalidArgument, "gm_fear %d exceeds range %d..%d", gmFear, daggerheart.GMFearMin, daggerheart.GMFearMax)
		}
		existingSnap, err := a.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return projectionstore.DaggerheartSnapshot{}, grpcerror.Internal("load existing daggerheart snapshot", err)
		}
		if errors.Is(err, storage.ErrNotFound) {
			existingSnap = projectionstore.DaggerheartSnapshot{
				CampaignID: campaignID,
				GMFear:     daggerheart.GMFearDefault,
			}
		}
		if existingSnap.GMFear == gmFear {
			// FIXME(telemetry): count snapshot updates that are idempotent (no state change).
			return existingSnap, nil
		}

		actorID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
		requestID := grpcmeta.RequestIDFromContext(ctx)
		invocationID := grpcmeta.InvocationIDFromContext(ctx)
		after := gmFear
		payload := daggerheart.GMFearSetPayload{After: &after}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return projectionstore.DaggerheartSnapshot{}, grpcerror.Internal("encode payload", err)
		}
		actorTypeForCommand := command.ActorTypeSystem
		if actorID != "" {
			actorTypeForCommand = command.ActorTypeGM
		}
		_, err = handler.ExecuteAndApplyDomainCommand(
			ctx,
			a.write,
			a.applier,
			commandbuild.System(commandbuild.SystemInput{
				CoreInput: commandbuild.CoreInput{
					CampaignID:   campaignID,
					Type:         handler.CommandTypeDaggerheartGMFearSet,
					ActorType:    actorTypeForCommand,
					ActorID:      actorID,
					SessionID:    grpcmeta.SessionIDFromContext(ctx),
					RequestID:    requestID,
					InvocationID: invocationID,
					EntityType:   "campaign",
					EntityID:     campaignID,
					PayloadJSON:  payloadJSON,
				},
				SystemID:      daggerheart.SystemID,
				SystemVersion: daggerheart.SystemVersion,
			}),
			domainwrite.Options{
				RequireEvents:   true,
				MissingEventMsg: "gm fear update did not emit an event",
				ApplyErr:        handler.ApplyErrorWithCodePreserve("apply event"),
			},
		)
		if err != nil {
			return projectionstore.DaggerheartSnapshot{}, err
		}

		dhSnapshot, err := a.stores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
		if err != nil {
			return projectionstore.DaggerheartSnapshot{}, grpcerror.Internal("load daggerheart snapshot", err)
		}

		return dhSnapshot, nil
	}

	return projectionstore.DaggerheartSnapshot{}, status.Error(codes.InvalidArgument, "no system snapshot update provided")
}
