package sessiontransport

import (
	"context"
	"fmt"
	"sort"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a sessionApplication) StartSession(ctx context.Context, campaignID string, in *campaignv1.StartSessionRequest) (storage.SessionRecord, error) {
	c, err := a.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return storage.SessionRecord{}, err
	}
	if err := authz.RequirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions(), c); err != nil {
		return storage.SessionRecord{}, err
	}

	if err := validate.MaxLength(in.GetName(), "name", validate.MaxNameLen); err != nil {
		return storage.SessionRecord{}, err
	}

	sessionID, err := a.idGenerator()
	if err != nil {
		return storage.SessionRecord{}, grpcerror.Internal("generate session id", err)
	}
	sessionName, err := a.resolveSessionStartName(ctx, c, campaignID, in.GetName())
	if err != nil {
		return storage.SessionRecord{}, err
	}

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
		CommandType: command.Type(commandids.SessionGMAuthoritySet),
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

// defaultGMAuthorityParticipant resolves the default GM authority participant
// for a session start. Local copy avoids importing the root game package.
func defaultGMAuthorityParticipant(campaignRecord storage.CampaignRecord, participants []storage.ParticipantRecord) (storage.ParticipantRecord, error) {
	desiredController := participant.ControllerHuman
	if campaignRecord.GmMode == campaign.GmModeAI {
		desiredController = participant.ControllerAI
	}
	candidates := make([]storage.ParticipantRecord, 0, len(participants))
	for _, record := range participants {
		if record.Role != participant.RoleGM || record.Controller != desiredController {
			continue
		}
		candidates = append(candidates, record)
	}
	if len(candidates) == 0 {
		return storage.ParticipantRecord{}, fmt.Errorf("no matching gm participant found for controller %s", desiredController)
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		if desiredController == participant.ControllerHuman {
			iOwner := candidates[i].CampaignAccess == participant.CampaignAccessOwner
			jOwner := candidates[j].CampaignAccess == participant.CampaignAccessOwner
			if iOwner != jOwner {
				return iOwner
			}
		}
		return strings.TrimSpace(candidates[i].ID) < strings.TrimSpace(candidates[j].ID)
	})
	return candidates[0], nil
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
	if err := authz.RequirePolicy(ctx, a.auth, domainauthz.CapabilityManageSessions(), c); err != nil {
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

// resolveSessionStartName ensures session.start commands always persist a usable name.
func (a sessionApplication) resolveSessionStartName(ctx context.Context, campaignRecord storage.CampaignRecord, campaignID, rawName string) (string, error) {
	name := strings.TrimSpace(rawName)
	if name != "" {
		return name, nil
	}
	count, err := a.stores.Session.CountSessions(ctx, campaignID)
	if err != nil {
		return "", grpcerror.Internal("count sessions", err)
	}
	return handler.DefaultSessionName(sessionStartLocale(campaignRecord.Locale), count+1), nil
}

func sessionStartLocale(value string) commonv1.Locale {
	locale, _ := platformi18n.ParseLocale(value)
	return platformi18n.NormalizeLocale(locale)
}
