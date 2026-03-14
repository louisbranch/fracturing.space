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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a sessionApplication) StartSession(ctx context.Context, campaignID string, in *campaignv1.StartSessionRequest) (storage.SessionRecord, error) {
	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.SessionRecord{}, err
	}
	if err := requirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions, c); err != nil {
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

	actorID, actorType := resolveCommandActor(ctx)

	payload := session.StartPayload{
		SessionID:   ids.SessionID(sessionID),
		SessionName: sessionName,
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.SessionRecord{}, grpcerror.Internal("encode payload", err)
	}
	_, err = executeAndApplyDomainCommand(
		ctx,
		a.write,
		a.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeSessionStart,
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    sessionID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session",
			EntityID:     sessionID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents("session.start did not emit an event"),
	)
	if err != nil {
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
	if err := requirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions, c); err != nil {
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
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return storage.SessionRecord{}, grpcerror.Internal("encode payload", err)
	}

	actorID, actorType := resolveCommandActor(ctx)

	_, err = executeAndApplyDomainCommand(
		ctx,
		a.write,
		a.applier,
		commandbuild.Core(commandbuild.CoreInput{
			CampaignID:   campaignID,
			Type:         commandTypeSessionEnd,
			ActorType:    actorType,
			ActorID:      actorID,
			SessionID:    sessionID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			EntityType:   "session",
			EntityID:     sessionID,
			PayloadJSON:  payloadJSON,
		}),
		domainwrite.RequireEvents("session.end did not emit an event"),
	)
	if err != nil {
		return storage.SessionRecord{}, err
	}

	updated, err := a.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		return storage.SessionRecord{}, grpcerror.Internal("load session", err)
	}

	return updated, nil
}
