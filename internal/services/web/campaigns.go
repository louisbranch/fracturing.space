package web

import (
	"context"
	"errors"
	"strings"

	webgrpc "github.com/louisbranch/fracturing.space/internal/services/web/infra/grpc"
)

func (h *handler) sessionUserID(ctx context.Context, accessToken string) (string, error) {
	if h == nil || h.campaignAccess == nil {
		return "", errors.New("campaign access checker is not configured")
	}
	return h.campaignAccess.ResolveUserID(ctx, accessToken)
}

func (h *handler) sessionUserIDForSession(ctx context.Context, sess *session) (string, error) {
	if sess == nil {
		return "", errors.New("session is not available")
	}
	userID, ok := sess.cachedUserIDValue()
	if ok {
		return userID, nil
	}

	resolvedID, err := h.sessionUserID(ctx, sess.accessToken)
	if err != nil {
		return "", err
	}
	resolvedID = strings.TrimSpace(resolvedID)
	sess.setCachedUserID(resolvedID)
	return resolvedID, nil
}

func (h *handler) ensureCampaignClients(ctx context.Context) error {
	if h == nil {
		return errors.New("web handler is not configured")
	}
	if h.campaignClient != nil {
		return nil
	}

	h.clientInitMu.Lock()
	defer h.clientInitMu.Unlock()

	if h.campaignClient != nil {
		return nil
	}

	clients, err := webgrpc.DialGame(ctx, h.config.GameAddr, h.config.GRPCDialTimeout)
	if err != nil {
		return err
	}
	if clients.CampaignClient == nil || clients.SessionClient == nil || clients.ParticipantClient == nil || clients.CharacterClient == nil || clients.InviteClient == nil {
		return errors.New("campaign service client is not configured")
	}

	h.participantClient = clients.ParticipantClient
	h.campaignClient = clients.CampaignClient
	h.eventClient = clients.EventClient
	h.sessionClient = clients.SessionClient
	h.characterClient = clients.CharacterClient
	h.inviteClient = clients.InviteClient
	h.campaignAccess = newCampaignAccessChecker(h.config, clients.ParticipantClient)
	return nil
}
