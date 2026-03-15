package game

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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a sessionApplication) StartSession(ctx context.Context, campaignID string, in *campaignv1.StartSessionRequest) (storage.SessionRecord, error) {
	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.SessionRecord{}, err
	}
	if err := authz.RequirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions, c); err != nil {
		return storage.SessionRecord{}, err
	}

	if err := validate.MaxLength(in.GetName(), "name", validate.MaxNameLen); err != nil {
		return storage.SessionRecord{}, err
	}

	sessionID, err := a.idGenerator()
	if err != nil {
		return storage.SessionRecord{}, grpcerror.Internal("generate session id", err)
	}
	sessionName := strings.TrimSpace(in.GetName())

	payload := session.StartPayload{
		SessionID:   ids.SessionID(sessionID),
		SessionName: sessionName,
	}
	if err := a.commands.Execute(ctx, sessionCommandExecutionInput{
		CommandType: handler.CommandTypeSessionStart,
		CampaignID:  campaignID,
		SessionID:   sessionID,
		Payload:     payload,
		Options:     domainwrite.RequireEvents("session.start did not emit an event"),
	}); err != nil {
		return storage.SessionRecord{}, err
	}

	participants, err := a.stores.Participant.ListParticipantsByCampaign(ctx, campaignID)
	if err != nil {
		return storage.SessionRecord{}, grpcerror.Internal("list campaign participants", err)
	}
	defaultAuthority, err := defaultGMAuthorityParticipant(c, participants)
	if err != nil {
		return storage.SessionRecord{}, grpcerror.Internal("resolve default gm authority", err)
	}
	if err := a.commands.Execute(ctx, sessionCommandExecutionInput{
		CommandType: commandTypeSessionGMAuthoritySet,
		CampaignID:  campaignID,
		SessionID:   sessionID,
		Payload: session.GMAuthoritySetPayload{
			SessionID:     ids.SessionID(sessionID),
			ParticipantID: ids.ParticipantID(defaultAuthority.ID),
		},
		Options: domainwrite.RequireEvents("session.gm_authority.set did not emit an event"),
	}); err != nil {
		return storage.SessionRecord{}, err
	}

	sess, err := a.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return storage.SessionRecord{}, grpcerror.Internal("load session", err)
	}
	return sess, nil
}

func (a sessionApplication) EndSession(ctx context.Context, campaignID string, in *campaignv1.EndSessionRequest) (storage.SessionRecord, error) {
	sessionID, err := validate.RequiredID(in.GetSessionId(), "session id")
	if err != nil {
		return storage.SessionRecord{}, err
	}

	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.SessionRecord{}, err
	}
	if err := authz.RequirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions, c); err != nil {
		return storage.SessionRecord{}, err
	}

	current, err := a.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return storage.SessionRecord{}, err
	}
	if current.Status == session.StatusEnded {
		return current, nil
	}
	payload := session.EndPayload{SessionID: ids.SessionID(sessionID)}
	if err := a.commands.Execute(ctx, sessionCommandExecutionInput{
		CommandType: handler.CommandTypeSessionEnd,
		CampaignID:  campaignID,
		SessionID:   sessionID,
		Payload:     payload,
		Options:     domainwrite.RequireEvents("session.end did not emit an event"),
	}); err != nil {
		return storage.SessionRecord{}, err
	}

	updated, err := a.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return storage.SessionRecord{}, grpcerror.Internal("load session", err)
	}

	return updated, nil
}
