package game

import (
	"context"
	"encoding/json"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (a sessionApplication) SetSessionSpotlight(ctx context.Context, campaignID string, in *campaignv1.SetSessionSpotlightRequest) (storage.SessionSpotlight, error) {
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return storage.SessionSpotlight{}, err
	}
	spotlightType, err := sessionSpotlightTypeFromProto(in.GetType())
	if err != nil {
		return storage.SessionSpotlight{}, status.Error(codes.InvalidArgument, err.Error())
	}
	characterID := strings.TrimSpace(in.GetCharacterId())
	if err := session.ValidateSpotlightTarget(spotlightType, characterID); err != nil {
		return storage.SessionSpotlight{}, status.Error(codes.InvalidArgument, err.Error())
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.SessionSpotlight{}, err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
		return storage.SessionSpotlight{}, err
	}
	if err := campaign.ValidateCampaignOperation(c.Status, campaign.CampaignOpSessionAction); err != nil {
		return storage.SessionSpotlight{}, err
	}
	sess, err := a.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return storage.SessionSpotlight{}, err
	}
	if sess.Status != session.StatusActive {
		return storage.SessionSpotlight{}, status.Error(codes.FailedPrecondition, "session is not active")
	}
	payload := session.SpotlightSetPayload{
		SpotlightType: string(spotlightType),
		CharacterID:   ids.CharacterID(characterID),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.SessionSpotlight{}, grpcerror.Internal("encode payload", err)
	}

	actorID, actorType := resolveCommandActor(ctx)

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores.Write,
		a.stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeSessionSpotlightSet,
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    sessionID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session",
			EntityID:     sessionID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{
			ApplyErr:        domainApplyErrorWithCodePreserve("apply event"),
			RequireEvents:   true,
			MissingEventMsg: "session.spotlight_set did not emit an event",
		},
	)
	if err != nil {
		return storage.SessionSpotlight{}, err
	}
	spotlight, err := a.stores.SessionSpotlight.GetSessionSpotlight(ctx, campaignID, sessionID)
	if err != nil {
		return storage.SessionSpotlight{}, grpcerror.Internal("load session spotlight", err)
	}

	return spotlight, nil
}

func (a sessionApplication) ClearSessionSpotlight(ctx context.Context, campaignID string, in *campaignv1.ClearSessionSpotlightRequest) (storage.SessionSpotlight, error) {
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return storage.SessionSpotlight{}, err
	}
	reason := strings.TrimSpace(in.GetReason())

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.SessionSpotlight{}, err
	}
	if err := requirePolicy(ctx, a.stores, domainauthz.CapabilityManageSessions, c); err != nil {
		return storage.SessionSpotlight{}, err
	}
	if _, err := a.stores.Session.GetSession(ctx, campaignID, sessionID); err != nil {
		return storage.SessionSpotlight{}, err
	}

	spotlight, err := a.stores.SessionSpotlight.GetSessionSpotlight(ctx, campaignID, sessionID)
	if err != nil {
		return storage.SessionSpotlight{}, err
	}
	payload := session.SpotlightClearedPayload{Reason: reason}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.SessionSpotlight{}, grpcerror.Internal("encode payload", err)
	}

	actorID, actorType := resolveCommandActor(ctx)

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.stores.Write,
		a.stores.Applier(),
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeSessionSpotlightClear,
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    sessionID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session",
			EntityID:     sessionID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.Options{
			ApplyErr:        domainApplyErrorWithCodePreserve("apply event"),
			RequireEvents:   true,
			MissingEventMsg: "session.spotlight_clear did not emit an event",
		},
	)
	if err != nil {
		return storage.SessionSpotlight{}, err
	}

	return spotlight, nil
}
