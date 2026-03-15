package sessiontransport

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"

	"context"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
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
	spotlightType, err := SpotlightTypeFromProto(in.GetType())
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
	if err := authz.RequirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions, c); err != nil {
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
	if err := a.commands.Execute(ctx, sessionCommandExecutionInput{
		CommandType: handler.CommandTypeSessionSpotlightSet,
		CampaignID:  campaignID,
		SessionID:   sessionID,
		Payload:     payload,
		Options: domainwrite.Options{
			ApplyErr:        handler.ApplyErrorWithCodePreserve("apply event"),
			RequireEvents:   true,
			MissingEventMsg: "session.spotlight_set did not emit an event",
		},
	}); err != nil {
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
	if err := authz.RequirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions, c); err != nil {
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
	if err := a.commands.Execute(ctx, sessionCommandExecutionInput{
		CommandType: handler.CommandTypeSessionSpotlightClear,
		CampaignID:  campaignID,
		SessionID:   sessionID,
		Payload:     payload,
		Options: domainwrite.Options{
			ApplyErr:        handler.ApplyErrorWithCodePreserve("apply event"),
			RequireEvents:   true,
			MissingEventMsg: "session.spotlight_clear did not emit an event",
		},
	}); err != nil {
		return storage.SessionSpotlight{}, err
	}

	return spotlight, nil
}
