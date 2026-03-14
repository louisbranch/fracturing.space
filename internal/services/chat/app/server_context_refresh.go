package server

import (
	"context"
	"errors"
	"log"
	"reflect"
	"strings"
	"time"

	gogrpccodes "google.golang.org/grpc/codes"
	gogrpcstatus "google.golang.org/grpc/status"
)

func joinedPayloadForRoom(room *campaignRoom, contextState communicationContext, latestSequenceID int64) joinedPayload {
	sessionID := strings.TrimSpace(contextState.Welcome.SessionID)
	campaignID := ""
	if room != nil {
		campaignID = room.campaignID
	}
	return joinedPayload{
		CampaignID:             campaignID,
		SessionID:              sessionID,
		LatestSequenceID:       latestSequenceID,
		ServerTime:             time.Now().UTC().Format(time.RFC3339),
		DefaultStreamID:        contextState.DefaultStreamID,
		DefaultPersonaID:       contextState.DefaultPersonaID,
		ActiveSessionGate:      contextState.ActiveSessionGate,
		ActiveSessionSpotlight: contextState.ActiveSessionSpotlight,
		Streams:                contextState.Streams,
		Personas:               contextState.Personas,
	}
}

func refreshRoomCommunicationContext(
	ctx context.Context,
	authorizer wsAuthorizer,
	room *campaignRoom,
	releaseCampaignUpdateSubscription func(string),
	releaseAITurnSubscription func(string),
) error {
	if ctx == nil || authorizer == nil || room == nil {
		return nil
	}
	provider, ok := authorizer.(wsCommunicationContextProvider)
	if !ok || provider == nil {
		return nil
	}

	sessions := room.sessionsSnapshot()
	if len(sessions) == 0 {
		return nil
	}

	previousSessionID := room.currentSessionID()
	previousGate := room.activeSessionGateState()
	previousSpotlight := room.activeSessionSpotlightState()

	var sharedContext *communicationContext
	for _, session := range sessions {
		if session == nil || session.peer == nil {
			continue
		}
		contextState, err := provider.ResolveCommunicationContext(ctx, room.campaignID, session.userID)
		if err != nil {
			if shouldEvictRoomSessionOnRefresh(err) {
				_ = writeWSError(session.peer, "", "FORBIDDEN", "campaign access is no longer available")
				leaveCampaignRoom(room, session, releaseCampaignUpdateSubscription, releaseAITurnSubscription)
				continue
			}
			log.Printf("chat: refresh communication context failed campaign=%q user=%q err=%v", room.campaignID, session.userID, err)
			continue
		}

		room.updateSessionSubscription(session, communicationStreamIDs(contextState.Streams))
		session.setCommunicationState(contextState)
		_ = session.peer.writeFrame(wsFrame{
			Type:    "chat.context",
			Payload: mustJSON(joinedPayloadForRoom(room, contextState, room.latestSequenceID())),
		})

		if sharedContext == nil {
			copied := contextState
			sharedContext = &copied
		}
	}

	if sharedContext == nil {
		return nil
	}

	nextSessionID := strings.TrimSpace(sharedContext.Welcome.SessionID)
	nextGate := sharedContext.ActiveSessionGate
	nextSpotlight := sharedContext.ActiveSessionSpotlight

	room.setSessionID(nextSessionID)
	room.setControlState(nextGate, nextSpotlight)

	if previousSessionID == nextSessionID &&
		reflect.DeepEqual(previousGate, nextGate) &&
		reflect.DeepEqual(previousSpotlight, nextSpotlight) {
		return nil
	}

	stateFrame := wsFrame{
		Type: "chat.state",
		Payload: mustJSON(statePayload{
			CampaignID:             room.campaignID,
			SessionID:              room.currentSessionID(),
			ActiveSessionGate:      nextGate,
			ActiveSessionSpotlight: nextSpotlight,
		}),
	}
	for _, subscriber := range room.subscribersSnapshot() {
		_ = subscriber.writeFrame(stateFrame)
	}
	return nil
}

func shouldEvictRoomSessionOnRefresh(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, errCampaignParticipantRequired) {
		return true
	}
	switch gogrpcstatus.Code(err) {
	case gogrpccodes.PermissionDenied, gogrpccodes.FailedPrecondition, gogrpccodes.NotFound:
		return true
	default:
		return false
	}
}

func isCommunicationCampaignContextEvent(eventType string) bool {
	eventType = strings.TrimSpace(eventType)
	switch {
	case strings.HasPrefix(eventType, "session."):
		return true
	case strings.HasPrefix(eventType, "scene."):
		return true
	case strings.HasPrefix(eventType, "participant."):
		return true
	case strings.HasPrefix(eventType, "character."):
		return true
	default:
		return false
	}
}
