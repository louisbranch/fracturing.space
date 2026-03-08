package game

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a snapshotApplication) UpdateSnapshotState(ctx context.Context, campaignID string, in *campaignv1.UpdateSnapshotStateRequest) (storage.DaggerheartSnapshot, error) {
	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.DaggerheartSnapshot{}, err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpCampaignMutate); err != nil {
		return storage.DaggerheartSnapshot{}, err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
		return storage.DaggerheartSnapshot{}, err
	}

	// Handle Daggerheart snapshot update
	if dhUpdate := in.GetDaggerheart(); dhUpdate != nil {
		gmFear := int(dhUpdate.GetGmFear())
		if gmFear < daggerheart.GMFearMin || gmFear > daggerheart.GMFearMax {
			return storage.DaggerheartSnapshot{}, status.Errorf(codes.InvalidArgument, "gm_fear %d exceeds range %d..%d", gmFear, daggerheart.GMFearMin, daggerheart.GMFearMax)
		}
		existingSnap, err := a.stores.SystemStores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
		if err != nil && !errors.Is(err, storage.ErrNotFound) {
			return storage.DaggerheartSnapshot{}, status.Errorf(codes.Internal, "load existing daggerheart snapshot: %v", err)
		}
		if errors.Is(err, storage.ErrNotFound) {
			existingSnap = storage.DaggerheartSnapshot{
				CampaignID: campaignID,
				GMFear:     daggerheart.GMFearDefault,
			}
		}
		if existingSnap.GMFear == gmFear {
			// FIXME(telemetry): count snapshot updates that are idempotent (no state change).
			return existingSnap, nil
		}

		actorID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
		applier := a.stores.Applier()
		requestID := grpcmeta.RequestIDFromContext(ctx)
		invocationID := grpcmeta.InvocationIDFromContext(ctx)
		after := gmFear
		payload := daggerheart.GMFearSetPayload{After: &after}
		payloadJSON, err := json.Marshal(payload)
		if err != nil {
			return storage.DaggerheartSnapshot{}, status.Errorf(codes.Internal, "encode payload: %v", err)
		}
		actorTypeForCommand := command.ActorTypeSystem
		if actorID != "" {
			actorTypeForCommand = command.ActorTypeGM
		}
		_, err = executeAndApplyDomainCommand(
			ctx,
			a.stores.Write,
			applier,
			commandbuild.System(commandbuild.SystemInput{
				CoreInput: commandbuild.CoreInput{
					CampaignID:   campaignID,
					Type:         commandTypeDaggerheartGMFearSet,
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
				ApplyErr:        domainApplyErrorWithCodePreserve("apply event"),
			},
		)
		if err != nil {
			return storage.DaggerheartSnapshot{}, err
		}

		dhSnapshot, err := a.stores.SystemStores.Daggerheart.GetDaggerheartSnapshot(ctx, campaignID)
		if err != nil {
			return storage.DaggerheartSnapshot{}, status.Errorf(codes.Internal, "load daggerheart snapshot: %v", err)
		}

		return dhSnapshot, nil
	}

	return storage.DaggerheartSnapshot{}, status.Error(codes.InvalidArgument, "no system snapshot update provided")
}
