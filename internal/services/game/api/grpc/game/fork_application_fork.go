package game

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/fork"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a forkApplication) ForkCampaign(ctx context.Context, sourceCampaignID string, in *campaignv1.ForkCampaignRequest) (storage.CampaignRecord, *campaignv1.Lineage, uint64, error) {
	if in.GetCopyParticipants() && a.stores.Participant == nil {
		return storage.CampaignRecord{}, nil, 0, status.Error(codes.Internal, "participant store is not configured")
	}

	// Get source campaign
	sourceCampaign, err := a.stores.Campaign.Get(ctx, sourceCampaignID)
	if err != nil {
		return storage.CampaignRecord{}, nil, 0, grpcerror.EnsureStatus(err)
	}
	if err := requirePolicy(ctx, a.auth, domainauthz.CapabilityManageCampaign, sourceCampaign); err != nil {
		return storage.CampaignRecord{}, nil, 0, err
	}
	// Stores.Validate guarantees Session in production wiring; keep a nil guard so
	// focused unit tests with partial stores remain supported.
	if a.stores.Session != nil {
		activeSession, err := a.stores.Session.GetActiveSession(ctx, sourceCampaignID)
		if err == nil {
			return storage.CampaignRecord{}, nil, 0, status.Errorf(
				codes.FailedPrecondition,
				"campaign has an active session: active_session_id=%s",
				activeSession.ID,
			)
		}
		if !errors.Is(err, storage.ErrNotFound) {
			return storage.CampaignRecord{}, nil, 0, grpcerror.Internal("check active session", err)
		}
	}

	// Determine fork point
	forkPoint := forkPointFromProto(in.GetForkPoint())
	forkEventSeq, err := a.resolveForkPoint(ctx, sourceCampaignID, forkPoint)
	if err != nil {
		return storage.CampaignRecord{}, nil, 0, err
	}

	// Get source campaign's fork metadata to determine origin
	sourceMetadata, err := a.stores.CampaignFork.GetCampaignForkMetadata(ctx, sourceCampaignID)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return storage.CampaignRecord{}, nil, 0, grpcerror.Internal("get source fork metadata", err)
	}

	originCampaignID := sourceMetadata.OriginCampaignID
	if originCampaignID == "" {
		originCampaignID = sourceCampaignID
	}

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
		GmMode:       gmModeToProto(sourceCampaign.GmMode).String(),
		Intent:       campaignIntentToProto(sourceCampaign.Intent).String(),
		AccessPolicy: campaignAccessPolicyToProto(sourceCampaign.AccessPolicy).String(),
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

	if _, err := a.copyForkEvents(ctx, sourceCampaignID, f.NewCampaignID, forkEventSeq, in.GetCopyParticipants()); err != nil {
		return storage.CampaignRecord{}, nil, 0, grpcerror.Internal("copy events", err)
	}

	// Calculate depth by walking the parent chain
	depth := calculateDepth(ctx, a.stores.CampaignFork, sourceCampaignID) + 1

	newCampaign, err := a.stores.Campaign.Get(ctx, f.NewCampaignID)
	if err != nil {
		return storage.CampaignRecord{}, nil, 0, grpcerror.Internal("load forked campaign", err)
	}

	lineage := &campaignv1.Lineage{
		CampaignId:       newCampaign.ID,
		ParentCampaignId: sourceCampaignID,
		ForkEventSeq:     forkEventSeq,
		OriginCampaignId: originCampaignID,
		Depth:            int32(depth),
	}

	return newCampaign, lineage, forkEventSeq, nil
}

func (a forkApplication) copyForkEvents(ctx context.Context, sourceCampaignID, forkCampaignID string, forkEventSeq uint64, copyParticipants bool) (time.Time, error) {
	if forkEventSeq == 0 {
		return time.Time{}, nil
	}

	inlineApplyEnabled := a.write.Runtime.InlineApplyEnabled()
	shouldApply := a.write.Runtime.ShouldApply()

	afterSeq := uint64(0)
	var lastEventAt time.Time
	for {
		events, err := a.stores.Event.ListEvents(ctx, sourceCampaignID, afterSeq, forkEventPageSize)
		if err != nil {
			return lastEventAt, fmt.Errorf("list events: %w", err)
		}
		if len(events) == 0 {
			return lastEventAt, nil
		}

		for _, evt := range events {
			if evt.Seq > forkEventSeq {
				return lastEventAt, nil
			}
			lastEventAt = evt.Timestamp
			shouldCopy, err := shouldCopyForkEvent(evt, copyParticipants)
			if err != nil {
				return lastEventAt, fmt.Errorf("filter forked event: %w", err)
			}
			if !shouldCopy {
				afterSeq = evt.Seq
				continue
			}

			forked := forkEventForCampaign(evt, forkCampaignID)
			stored, err := a.stores.Event.AppendEvent(ctx, forked)
			if err != nil {
				return lastEventAt, fmt.Errorf("append forked event: %w", err)
			}
			if inlineApplyEnabled && shouldApply(stored) {
				if err := a.applier.Apply(ctx, stored); err != nil {
					return lastEventAt, fmt.Errorf("apply forked event: %w", err)
				}
			}
			afterSeq = evt.Seq
		}

		if len(events) < forkEventPageSize {
			return lastEventAt, nil
		}
	}
}
