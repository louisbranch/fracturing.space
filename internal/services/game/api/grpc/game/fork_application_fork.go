package game

import (
	"context"
	"encoding/json"
	"fmt"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/campaigntransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/fork"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a forkApplication) ForkCampaign(ctx context.Context, sourceCampaignID string, in *campaignv1.ForkCampaignRequest) (storage.CampaignRecord, *campaignv1.Lineage, uint64, error) {
	sourceState, err := a.loadForkSourceState(ctx, sourceCampaignID, in.GetCopyParticipants())
	if err != nil {
		return storage.CampaignRecord{}, nil, 0, err
	}
	sourceCampaign := sourceState.campaign

	// Determine fork point
	forkPoint := forkPointFromProto(in.GetForkPoint())
	forkEventSeq, err := a.resolveForkPoint(ctx, sourceCampaignID, forkPoint)
	if err != nil {
		return storage.CampaignRecord{}, nil, 0, err
	}
	originCampaignID := sourceState.originCampaignID

	// Create the fork record
	f, err := fork.CreateFork(fork.CreateForkInput{
		SourceCampaignID: sourceCampaignID,
		ForkPoint:        forkPoint,
		NewCampaignName:  in.GetNewCampaignName(),
		CopyParticipants: in.GetCopyParticipants(),
	}, originCampaignID, forkEventSeq, a.clock, a.idGenerator)
	if err != nil {
		return storage.CampaignRecord{}, nil, 0, err
	}

	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)

	newCampaignName := in.GetNewCampaignName()
	if newCampaignName == "" {
		newCampaignName = fmt.Sprintf("%s (Fork)", sourceCampaign.Name)
	}
	campaignPayload := campaign.CreatePayload{
		Name:         newCampaignName,
		Locale:       platformi18n.LocaleString(sourceCampaign.Locale),
		GameSystem:   sourceCampaign.System.String(),
		GmMode:       campaigntransport.GMModeToProto(sourceCampaign.GmMode).String(),
		Intent:       campaigntransport.CampaignIntentToProto(sourceCampaign.Intent).String(),
		AccessPolicy: campaigntransport.CampaignAccessPolicyToProto(sourceCampaign.AccessPolicy).String(),
		ThemePrompt:  sourceCampaign.ThemePrompt,
	}
	campaignJSON, err := json.Marshal(campaignPayload)
	if err != nil {
		return storage.CampaignRecord{}, nil, 0, grpcerror.Internal("encode payload", err)
	}
	actorID, actorType := resolveCommandActor(ctx)
	_, err = executeAndApplyDomainCommand(
		ctx,
		a.write,
		a.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   f.NewCampaignID,
			Type:         commandTypeCampaignCreate,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    requestID,
			InvocationID: invocationID,
			EntityType:   "campaign",
			EntityID:     f.NewCampaignID,
			PayloadJSON:  campaignJSON,
		}),
		domainwrite.Options{
			ApplyErrMessage: "apply campaign.created",
		},
	)
	if err != nil {
		return storage.CampaignRecord{}, nil, 0, err
	}

	forkPayload := campaign.ForkPayload{
		ParentCampaignID: ids.CampaignID(sourceCampaignID),
		ForkEventSeq:     forkEventSeq,
		OriginCampaignID: ids.CampaignID(originCampaignID),
		CopyParticipants: in.GetCopyParticipants(),
	}
	forkJSON, err := json.Marshal(forkPayload)
	if err != nil {
		return storage.CampaignRecord{}, nil, 0, grpcerror.Internal("encode payload", err)
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
			CampaignID:   f.NewCampaignID,
			Type:         commandTypeCampaignFork,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    requestID,
			InvocationID: invocationID,
			EntityType:   "campaign",
			EntityID:     f.NewCampaignID,
			PayloadJSON:  forkJSON,
		}),
		domainwrite.Options{
			ApplyErrMessage: "apply campaign.forked",
		},
	)
	if err != nil {
		return storage.CampaignRecord{}, nil, 0, err
	}

	if _, err := a.eventReplay.CopyToCampaign(ctx, sourceCampaignID, f.NewCampaignID, forkEventSeq, in.GetCopyParticipants()); err != nil {
		return storage.CampaignRecord{}, nil, 0, grpcerror.Internal("copy events", err)
	}

	newCampaign, err := a.stores.Campaign.Get(ctx, f.NewCampaignID)
	if err != nil {
		return storage.CampaignRecord{}, nil, 0, grpcerror.Internal("load forked campaign", err)
	}
	return newCampaign, a.buildLineage(ctx, newCampaign.ID, sourceCampaignID, originCampaignID, forkEventSeq), forkEventSeq, nil
}
