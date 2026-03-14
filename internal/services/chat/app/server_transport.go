package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"golang.org/x/net/websocket"
	gogrpccodes "google.golang.org/grpc/codes"
	gogrpcstatus "google.golang.org/grpc/status"
)

// NewHandler creates chat routes for tests and offline paths.
// WebSocket auth is intentionally disabled in this constructor.
func NewHandler() http.Handler {
	return newHandler(nil, false, nil, nil, nil, nil, nil, nil, nil)
}

// NewHandlerWithAuthorizer creates chat routes with enforced websocket identity checks.
func NewHandlerWithAuthorizer(authorizer wsAuthorizer) http.Handler {
	return newHandler(authorizer, true, nil, nil, nil, nil, nil, nil, nil)
}

func newHandler(
	authorizer wsAuthorizer,
	requireAuth bool,
	hub *roomHub,
	ensureCampaignUpdateSubscription func(string),
	releaseCampaignUpdateSubscription func(string),
	ensureAITurnSubscription func(string, string, string),
	releaseAITurnSubscription func(string),
	issueAISessionGrant func(context.Context, *campaignRoom, string) error,
	aiInvocationClient aiv1.InvocationServiceClient,
) http.Handler {
	if hub == nil {
		hub = newRoomHub()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/up", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	wsHandler := websocket.Handler(func(conn *websocket.Conn) {
		handleWSConn(
			conn,
			hub,
			authorizer,
			ensureCampaignUpdateSubscription,
			releaseCampaignUpdateSubscription,
			ensureAITurnSubscription,
			releaseAITurnSubscription,
			issueAISessionGrant,
			aiInvocationClient,
		)
	})

	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if requireAuth {
			if authorizer == nil {
				http.Error(w, "websocket auth is not configured", http.StatusServiceUnavailable)
				return
			}

			accessToken := accessTokenFromRequest(r)
			if accessToken == "" {
				log.Printf("chat: websocket unauthorized: missing auth cookie (fs_token/web_session) for host=%q remote=%s path=%q", r.Host, r.RemoteAddr, r.URL.Path)
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}

			userID, err := authorizer.Authenticate(r.Context(), accessToken)
			if err != nil || strings.TrimSpace(userID) == "" {
				if err != nil {
					log.Printf("chat: websocket unauthorized: auth introspection failed for host=%q remote=%s path=%q err=%v", r.Host, r.RemoteAddr, r.URL.Path, err)
				} else {
					log.Printf("chat: websocket unauthorized: empty user id after auth for host=%q remote=%s path=%q", r.Host, r.RemoteAddr, r.URL.Path)
				}
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), wsUserIDContextKey{}, strings.TrimSpace(userID))
			r = r.WithContext(ctx)
		}

		wsHandler.ServeHTTP(w, r)
	})

	return mux
}

func accessTokenFromRequest(r *http.Request) string {
	if r == nil {
		return ""
	}
	cookie, err := r.Cookie(tokenCookieName)
	if err == nil {
		if token := strings.TrimSpace(cookie.Value); token != "" {
			return token
		}
	}
	webSessionCookie, err := r.Cookie(webSessionCookieName)
	if err != nil {
		return ""
	}
	if sessionID := strings.TrimSpace(webSessionCookie.Value); sessionID != "" {
		return webSessionTokenPrefix + sessionID
	}
	return ""
}

func handleWSConn(
	conn *websocket.Conn,
	hub *roomHub,
	authorizer wsAuthorizer,
	ensureCampaignUpdateSubscription func(string),
	releaseCampaignUpdateSubscription func(string),
	ensureAITurnSubscription func(string, string, string),
	releaseAITurnSubscription func(string),
	issueAISessionGrant func(context.Context, *campaignRoom, string) error,
	aiInvocationClient aiv1.InvocationServiceClient,
) {
	defer func() {
		_ = conn.Close()
	}()

	decoder := json.NewDecoder(conn)
	peer := newWSPeer(json.NewEncoder(conn))
	userID := "participant"
	if request := conn.Request(); request != nil {
		if resolved, ok := request.Context().Value(wsUserIDContextKey{}).(string); ok && strings.TrimSpace(resolved) != "" {
			userID = strings.TrimSpace(resolved)
		}
	}
	session := newWSSession(userID, peer)
	defer func() {
		if room := session.currentRoom(); room != nil {
			leaveCampaignRoom(room, session, releaseCampaignUpdateSubscription, releaseAITurnSubscription)
		}
	}()

	windowStart := time.Now()
	framesInWindow := 0
	decodeErrors := 0

	for {
		var frame wsFrame
		if err := decoder.Decode(&frame); err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			decodeErrors++
			_ = writeWSError(session.peer, "", "INVALID_ARGUMENT", "invalid frame payload")
			if decodeErrors >= maxDecodeErrorsPerConn {
				return
			}
			continue
		}
		decodeErrors = 0

		if len(frame.Payload) > maxFramePayloadBytes {
			_ = writeWSError(session.peer, frame.RequestID, "INVALID_ARGUMENT", "payload too large")
			continue
		}

		now := time.Now()
		if now.Sub(windowStart) >= time.Second {
			windowStart = now
			framesInWindow = 0
		}
		framesInWindow++
		if framesInWindow > maxFramesPerSecond {
			_ = writeWSError(session.peer, frame.RequestID, "RESOURCE_EXHAUSTED", "rate limit exceeded")
			return
		}

		switch frame.Type {
		case "chat.join":
			handleJoinFrame(
				conn.Request().Context(),
				session,
				hub,
				authorizer,
				frame,
				ensureCampaignUpdateSubscription,
				releaseCampaignUpdateSubscription,
				ensureAITurnSubscription,
				releaseAITurnSubscription,
				issueAISessionGrant,
			)
		case "chat.send":
			handleSendFrame(
				session,
				frame,
			)
		case "chat.control":
			handleControlFrame(
				conn.Request().Context(),
				session,
				authorizer,
				frame,
				aiInvocationClient,
				ensureAITurnSubscription,
				issueAISessionGrant,
			)
		case "chat.history.before":
			handleHistoryBeforeFrame(session, frame)
		default:
			_ = writeWSError(session.peer, frame.RequestID, "INVALID_ARGUMENT", "unsupported frame type")
		}
	}
}

func leaveCampaignRoom(
	room *campaignRoom,
	session *wsSession,
	releaseCampaignUpdateSubscription func(string),
	releaseAITurnSubscription func(string),
) {
	if room == nil || session == nil {
		return
	}
	if session.currentRoom() == room {
		session.setRoom(nil)
	}
	if room.leave(session) {
		if releaseCampaignUpdateSubscription != nil {
			releaseCampaignUpdateSubscription(room.campaignID)
		}
		if releaseAITurnSubscription != nil {
			releaseAITurnSubscription(room.campaignID)
		}
	}
}

func handleJoinFrame(
	ctx context.Context,
	session *wsSession,
	hub *roomHub,
	authorizer wsAuthorizer,
	frame wsFrame,
	ensureCampaignUpdateSubscription func(string),
	releaseCampaignUpdateSubscription func(string),
	ensureAITurnSubscription func(string, string, string),
	releaseAITurnSubscription func(string),
	issueAISessionGrant func(context.Context, *campaignRoom, string) error,
) {
	var payload joinPayload
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		_ = writeWSError(session.peer, frame.RequestID, "INVALID_ARGUMENT", "invalid join payload")
		return
	}

	campaignID := strings.TrimSpace(payload.CampaignID)
	if campaignID == "" {
		_ = writeWSError(session.peer, frame.RequestID, "INVALID_ARGUMENT", "campaign_id is required")
		return
	}

	contextState, err := resolveSessionCommunicationContext(ctx, authorizer, campaignID, session.userID)
	if err != nil {
		if errors.Is(err, errCampaignParticipantRequired) {
			log.Printf("chat: campaign participant required for user=%q campaign=%q", session.userID, campaignID)
			_ = writeWSError(session.peer, frame.RequestID, "FORBIDDEN", "participant access required for campaign")
			return
		}
		log.Printf("chat: failed to resolve campaign context user=%q campaign=%q err=%v", session.userID, campaignID, err)
		_ = writeWSError(session.peer, frame.RequestID, "UNAVAILABLE", "campaign context lookup unavailable")
		return
	}
	if contextState.Welcome == (joinWelcome{}) && authorizer != nil {
		allowed, err := authorizer.IsCampaignParticipant(ctx, campaignID, session.userID)
		if err != nil {
			log.Printf("chat: campaign membership check failed user=%q campaign=%q err=%v", session.userID, campaignID, err)
			_ = writeWSError(session.peer, frame.RequestID, "UNAVAILABLE", "campaign membership verification unavailable")
			return
		}
		if !allowed {
			_ = writeWSError(session.peer, frame.RequestID, "FORBIDDEN", "participant access required for campaign")
			return
		}
		contextState = fallbackCommunicationContext(campaignID, joinWelcome{
			ParticipantName: session.userID,
			CampaignName:    campaignID,
			Locale:          commonv1.Locale_LOCALE_EN_US,
		}, session.userID)
	}
	if contextState.Welcome == (joinWelcome{}) {
		contextState = fallbackCommunicationContext(campaignID, joinWelcome{
			ParticipantName: session.userID,
			CampaignName:    campaignID,
			Locale:          commonv1.Locale_LOCALE_EN_US,
		}, session.userID)
	}
	welcome := contextState.Welcome
	if gmModeRequiresAIBinding(welcome.GmMode) && strings.TrimSpace(welcome.AIAgentID) == "" {
		_ = writeWSError(session.peer, frame.RequestID, "FAILED_PRECONDITION", "campaign ai binding is required")
		return
	}
	if ensureCampaignUpdateSubscription != nil {
		ensureCampaignUpdateSubscription(campaignID)
	}

	room := hub.room(campaignID)
	room.setSessionID(welcome.SessionID)
	room.setAIBinding(welcome.GmMode, welcome.AIAgentID)
	room.setControlState(contextState.ActiveSessionGate, contextState.ActiveSessionSpotlight)
	if room.aiRelayEnabled() && issueAISessionGrant != nil {
		if err := issueAISessionGrant(ctx, room, session.userID); err != nil {
			log.Printf("chat: failed to issue ai session grant campaign=%q err=%v", campaignID, err)
			room.clearAISessionGrant()
		}
	}
	if room.aiRelayReady() && ensureAITurnSubscription != nil {
		ensureAITurnSubscription(campaignID, room.currentSessionID(), welcome.AIAgentID)
	}
	if strings.TrimSpace(welcome.SessionName) == "" {
		welcome.SessionName = room.currentSessionID()
	}
	if strings.TrimSpace(contextState.DefaultStreamID) == "" {
		contextState.DefaultStreamID = chatDefaultStreamID(campaignID)
	}
	session.setCommunicationState(contextState)
	previous := session.setRoom(room)
	if previous != nil && previous != room {
		leaveCampaignRoom(previous, session, releaseCampaignUpdateSubscription, releaseAITurnSubscription)
	}
	latest := room.join(session, communicationStreamIDs(contextState.Streams))

	_ = session.peer.writeFrame(wsFrame{
		Type:    "chat.joined",
		Payload: mustJSON(joinedPayloadForRoom(room, contextState, latest)),
	})
	_ = session.peer.writeFrame(wsFrame{
		Type: "chat.message",
		Payload: mustJSON(messageEnvelope{
			Message: chatMessage{
				MessageID:  fmt.Sprintf("sys_%d", time.Now().UnixNano()),
				CampaignID: campaignID,
				SessionID:  room.currentSessionID(),
				SequenceID: latest,
				SentAt:     time.Now().UTC().Format(time.RFC3339),
				Kind:       "system",
				StreamID:   chatSystemStreamID(campaignID),
				Actor: messageActor{
					ParticipantID: "system",
					PersonaID:     "participant:system",
					Mode:          "participant",
					Name:          localizedSystemLabel(welcome.Locale),
				},
				Body: localizedJoinWelcomeBody(welcome),
			},
		}),
	})
}

func localizedSystemLabel(locale commonv1.Locale) string {
	switch locale {
	case commonv1.Locale_LOCALE_PT_BR:
		return "sistema"
	default:
		return "system"
	}
}

func localizedJoinWelcomeBody(welcome joinWelcome) string {
	participantName := strings.TrimSpace(welcome.ParticipantName)
	if participantName == "" {
		participantName = "participant"
	}
	campaignName := strings.TrimSpace(welcome.CampaignName)
	if campaignName == "" {
		campaignName = "campaign"
	}
	sessionName := strings.TrimSpace(welcome.SessionName)
	if sessionName == "" {
		sessionName = strings.TrimSpace(welcome.SessionID)
	}

	switch welcome.Locale {
	case commonv1.Locale_LOCALE_PT_BR:
		if sessionName == "" {
			return fmt.Sprintf("Bem-vindo %s. Você entrou na Campanha %s.", participantName, campaignName)
		}
		return fmt.Sprintf("Bem-vindo %s. Você entrou na Campanha %s, Sessão %s.", participantName, campaignName, sessionName)
	default:
		if sessionName == "" {
			return fmt.Sprintf("Welcome %s. You've joined Campaign %s.", participantName, campaignName)
		}
		return fmt.Sprintf("Welcome %s. You've joined Campaign %s, Session %s.", participantName, campaignName, sessionName)
	}
}

func handleSendFrame(
	session *wsSession,
	frame wsFrame,
) {
	var payload sendPayload
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		_ = writeWSError(session.peer, frame.RequestID, "INVALID_ARGUMENT", "invalid send payload")
		return
	}

	clientMessageID := strings.TrimSpace(payload.ClientMessageID)
	if clientMessageID == "" {
		_ = writeWSError(session.peer, frame.RequestID, "INVALID_ARGUMENT", "client_message_id is required")
		return
	}
	if utf8.RuneCountInString(clientMessageID) > maxClientMessageIDRunes {
		_ = writeWSError(session.peer, frame.RequestID, "INVALID_ARGUMENT", "client_message_id must be at most 128 characters")
		return
	}

	body := strings.TrimSpace(payload.Body)
	if body == "" {
		_ = writeWSError(session.peer, frame.RequestID, "INVALID_ARGUMENT", "body is required")
		return
	}
	if utf8.RuneCountInString(body) > maxMessageBodyRunes {
		_ = writeWSError(session.peer, frame.RequestID, "INVALID_ARGUMENT", "body must be at most 12000 characters")
		return
	}

	room := session.currentRoom()
	if room == nil {
		_ = writeWSError(session.peer, frame.RequestID, "FORBIDDEN", "must join campaign room before sending")
		return
	}

	state := session.communicationState()
	streamID, err := resolveOutgoingStreamID(state, payload.StreamID)
	if err != nil {
		_ = writeWSError(session.peer, frame.RequestID, "FORBIDDEN", err.Error())
		return
	}
	actor, err := resolveOutgoingActor(state, session.userID, payload.PersonaID)
	if err != nil {
		_ = writeWSError(session.peer, frame.RequestID, "FORBIDDEN", err.Error())
		return
	}

	msg, duplicate, subscribers := room.appendMessage(actor, body, clientMessageID, streamID)

	_ = session.peer.writeFrame(wsFrame{
		Type:      "chat.ack",
		RequestID: frame.RequestID,
		Payload: mustJSON(ackEnvelope{
			Result: ackResult{
				Status:     "ok",
				MessageID:  msg.MessageID,
				SequenceID: msg.SequenceID,
			},
		}),
	})

	if duplicate {
		return
	}

	messageFrame := wsFrame{
		Type:    "chat.message",
		Payload: mustJSON(messageEnvelope{Message: msg}),
	}
	for _, subscriber := range subscribers {
		_ = subscriber.writeFrame(messageFrame)
	}

}

func handleControlFrame(
	ctx context.Context,
	session *wsSession,
	authorizer wsAuthorizer,
	frame wsFrame,
	aiInvocationClient aiv1.InvocationServiceClient,
	ensureAITurnSubscription func(string, string, string),
	issueAISessionGrant func(context.Context, *campaignRoom, string) error,
) {
	var payload controlPayload
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		_ = writeWSError(session.peer, frame.RequestID, "INVALID_ARGUMENT", "invalid control payload")
		return
	}

	room := session.currentRoom()
	if room == nil {
		_ = writeWSError(session.peer, frame.RequestID, "FORBIDDEN", "must join campaign room before sending control actions")
		return
	}

	provider, ok := authorizer.(wsCommunicationControlProvider)
	if !ok || provider == nil {
		_ = writeWSError(session.peer, frame.RequestID, "UNAVAILABLE", "communication control is unavailable")
		return
	}

	action := strings.ToLower(strings.TrimSpace(payload.Action))
	if action == "" {
		_ = writeWSError(session.peer, frame.RequestID, "INVALID_ARGUMENT", "action is required")
		return
	}

	state := session.communicationState()
	callCtx := grpcauthctx.WithUserID(ctx, session.userID)
	callCtx = grpcauthctx.WithParticipantID(callCtx, state.participantID)
	previousGate := room.activeSessionGateState()

	var (
		updated communicationContext
		err     error
	)
	switch action {
	case "gate.open":
		updated, err = provider.OpenCommunicationGate(callCtx, room.campaignID, state.participantID, payload.GateType, payload.Reason, payload.Metadata)
	case "gate.respond":
		updated, err = provider.RespondToCommunicationGate(callCtx, room.campaignID, state.participantID, payload.Decision, payload.Response)
	case "gate.resolve":
		updated, err = provider.ResolveCommunicationGate(callCtx, room.campaignID, state.participantID, payload.Decision, payload.Resolution)
	case "gate.abandon":
		updated, err = provider.AbandonCommunicationGate(callCtx, room.campaignID, state.participantID, payload.Reason)
	case "gm_handoff.request":
		updated, err = provider.RequestGMHandoff(callCtx, room.campaignID, state.participantID, payload.Reason, payload.Metadata)
	case "gm_handoff.resolve":
		updated, err = provider.ResolveGMHandoff(callCtx, room.campaignID, state.participantID, payload.Decision, payload.Resolution)
	case "gm_handoff.abandon":
		updated, err = provider.AbandonGMHandoff(callCtx, room.campaignID, state.participantID, payload.Reason)
	default:
		_ = writeWSError(session.peer, frame.RequestID, "INVALID_ARGUMENT", "unsupported control action")
		return
	}
	if err != nil {
		_ = writeWSRPCError(session.peer, frame.RequestID, err)
		return
	}

	if strings.TrimSpace(updated.Welcome.SessionID) != "" {
		room.setSessionID(updated.Welcome.SessionID)
	}
	room.setAIBinding(updated.Welcome.GmMode, updated.Welcome.AIAgentID)
	room.setControlState(updated.ActiveSessionGate, updated.ActiveSessionSpotlight)
	session.setCommunicationState(updated)

	_ = session.peer.writeFrame(wsFrame{
		Type:      "chat.ack",
		RequestID: frame.RequestID,
		Payload: mustJSON(ackEnvelope{
			Result: ackResult{Status: "ok"},
		}),
	})

	if shouldTriggerAIOnGMHandoffRequest(action, previousGate, updated.ActiveSessionGate) && aiInvocationClient != nil && room.aiRelayEnabled() {
		if !room.aiRelayReady() && issueAISessionGrant != nil {
			if err := issueAISessionGrant(ctx, room, session.userID); err != nil {
				log.Printf("chat: refresh ai session grant on control failed campaign=%q err=%v", room.campaignID, err)
				room.clearAISessionGrant()
			}
			if room.aiRelayReady() && ensureAITurnSubscription != nil {
				ensureAITurnSubscription(room.campaignID, room.currentSessionID(), room.aiAgentIDValue())
			}
		}
		if room.aiRelayReady() {
			if err := submitBufferedCampaignTurnToAI(ctx, aiInvocationClient, room, session, payload.Reason); err != nil {
				log.Printf("chat: submit buffered campaign turn to ai failed campaign=%q err=%v", room.campaignID, err)
			}
		}
	}

	stateFrame := wsFrame{
		Type: "chat.state",
		Payload: mustJSON(statePayload{
			CampaignID:             room.campaignID,
			SessionID:              room.currentSessionID(),
			ActiveSessionGate:      updated.ActiveSessionGate,
			ActiveSessionSpotlight: updated.ActiveSessionSpotlight,
		}),
	}
	for _, subscriber := range room.subscribersSnapshot() {
		_ = subscriber.writeFrame(stateFrame)
	}
}

func handleHistoryBeforeFrame(session *wsSession, frame wsFrame) {
	var payload historyBeforePayload
	if err := json.Unmarshal(frame.Payload, &payload); err != nil {
		_ = writeWSError(session.peer, frame.RequestID, "INVALID_ARGUMENT", "invalid history payload")
		return
	}
	if payload.BeforeSequenceID < 1 {
		_ = writeWSError(session.peer, frame.RequestID, "INVALID_ARGUMENT", "before_sequence_id must be >= 1")
		return
	}
	if payload.Limit <= 0 {
		payload.Limit = 50
	}
	if payload.Limit > 200 {
		payload.Limit = 200
	}

	room := session.currentRoom()
	if room == nil {
		_ = writeWSError(session.peer, frame.RequestID, "FORBIDDEN", "must join campaign room before requesting history")
		return
	}

	streamID, err := resolveOutgoingStreamID(session.communicationState(), payload.StreamID)
	if err != nil {
		_ = writeWSError(session.peer, frame.RequestID, "FORBIDDEN", err.Error())
		return
	}

	history := room.historyBefore(streamID, payload.BeforeSequenceID, payload.Limit)
	for _, msg := range history {
		_ = session.peer.writeFrame(wsFrame{
			Type:    "chat.message",
			Payload: mustJSON(messageEnvelope{Message: msg}),
		})
	}
	_ = session.peer.writeFrame(wsFrame{
		Type:      "chat.ack",
		RequestID: frame.RequestID,
		Payload: mustJSON(ackEnvelope{
			Result: ackResult{
				Status: "ok",
				Count:  len(history),
			},
		}),
	})
}

func shouldTriggerAIOnGMHandoffRequest(action string, previousGate *chatSessionGate, updatedGate *chatSessionGate) bool {
	if action != "gm_handoff.request" {
		return false
	}
	return !isOpenGMHandoffGate(previousGate) && isOpenGMHandoffGate(updatedGate)
}

func isOpenGMHandoffGate(gate *chatSessionGate) bool {
	if gate == nil {
		return false
	}
	return strings.TrimSpace(gate.GateType) == "gm_handoff" && strings.TrimSpace(gate.Status) == "open"
}

func writeWSError(peer *wsPeer, requestID string, code string, message string) error {
	return peer.writeFrame(wsFrame{
		Type:      "chat.error",
		RequestID: requestID,
		Payload: mustJSON(wsErrorEnvelope{
			Error: wsError{
				Code:      code,
				Message:   message,
				Retryable: false,
			},
		}),
	})
}

func writeWSRPCError(peer *wsPeer, requestID string, err error) error {
	statusErr := gogrpcstatus.Convert(err)
	code := wsErrorCodeFromRPC(statusErr.Code())
	return peer.writeFrame(wsFrame{
		Type:      "chat.error",
		RequestID: requestID,
		Payload: mustJSON(wsErrorEnvelope{
			Error: wsError{
				Code:      code,
				Message:   statusErr.Message(),
				Retryable: statusErr.Code() == gogrpccodes.Unavailable || statusErr.Code() == gogrpccodes.DeadlineExceeded,
			},
		}),
	})
}

func wsErrorCodeFromRPC(code gogrpccodes.Code) string {
	switch code {
	case gogrpccodes.InvalidArgument:
		return "INVALID_ARGUMENT"
	case gogrpccodes.PermissionDenied:
		return "FORBIDDEN"
	case gogrpccodes.NotFound:
		return "NOT_FOUND"
	case gogrpccodes.FailedPrecondition:
		return "FAILED_PRECONDITION"
	case gogrpccodes.ResourceExhausted:
		return "RESOURCE_EXHAUSTED"
	case gogrpccodes.DeadlineExceeded:
		return "UNAVAILABLE"
	case gogrpccodes.Unavailable:
		return "UNAVAILABLE"
	default:
		return "INTERNAL"
	}
}

func submitBufferedCampaignTurnToAI(ctx context.Context, invocationClient aiv1.InvocationServiceClient, room *campaignRoom, session *wsSession, handoffReason string) error {
	if invocationClient == nil || room == nil || session == nil {
		return nil
	}
	aiAgentID := room.aiAgentIDValue()
	if aiAgentID == "" {
		return nil
	}
	submission, ok := room.pendingAITurnSubmission(handoffReason)
	if !ok {
		return nil
	}
	state := session.communicationState()
	participantName := strings.TrimSpace(session.userID)
	if persona, ok := state.personasByID[state.defaultPersonaID]; ok && strings.TrimSpace(persona.DisplayName) != "" {
		participantName = strings.TrimSpace(persona.DisplayName)
	}
	callCtx := grpcauthctx.WithUserID(ctx, session.userID)
	_, err := invocationClient.SubmitCampaignTurn(callCtx, &aiv1.SubmitCampaignTurnRequest{
		CampaignId:      room.campaignID,
		SessionId:       room.currentSessionID(),
		AgentId:         aiAgentID,
		ParticipantId:   state.participantID,
		ParticipantName: participantName,
		MessageId:       submission.correlationMessageID,
		Body:            submission.body,
		SessionGrant:    room.aiSessionGrantValue(),
	})
	if err == nil {
		room.markAITurnSubmitted(submission.highestSequenceID)
	}
	return err
}

func gmModeRequiresAIBinding(mode string) bool {
	switch strings.ToUpper(strings.TrimSpace(mode)) {
	case "AI", "HYBRID", "GM_MODE_AI", "GM_MODE_HYBRID":
		return true
	default:
		return false
	}
}

func resolveSessionCommunicationContext(ctx context.Context, authorizer wsAuthorizer, campaignID string, userID string) (communicationContext, error) {
	if provider, ok := authorizer.(wsCommunicationContextProvider); ok {
		return provider.ResolveCommunicationContext(ctx, campaignID, userID)
	}
	if provider, ok := authorizer.(wsJoinWelcomeProvider); ok {
		resolved, err := provider.ResolveJoinWelcome(ctx, campaignID, userID)
		if err != nil {
			return communicationContext{}, err
		}
		return fallbackCommunicationContext(campaignID, resolved, userID), nil
	}
	return communicationContext{}, nil
}

func fallbackCommunicationContext(campaignID string, welcome joinWelcome, userID string) communicationContext {
	streams := []chatStream{
		{
			StreamID:  chatSystemStreamID(campaignID),
			Kind:      "system",
			Scope:     "session",
			SessionID: welcome.SessionID,
			Label:     chatStreamSystemLabel,
		},
		{
			StreamID:  chatDefaultStreamID(campaignID),
			Kind:      "table",
			Scope:     "session",
			SessionID: welcome.SessionID,
			Label:     chatStreamTableLabel,
		},
		{
			StreamID:  chatControlStreamID(campaignID),
			Kind:      "control",
			Scope:     "session",
			SessionID: welcome.SessionID,
			Label:     chatStreamControlLabel,
		},
	}
	participantID := strings.TrimSpace(userID)
	personas := []chatPersona{
		{
			PersonaID:     "participant:" + participantID,
			Kind:          "participant",
			ParticipantID: participantID,
			DisplayName:   welcome.ParticipantName,
		},
	}
	return communicationContext{
		Welcome:          welcome,
		ParticipantID:    participantID,
		DefaultStreamID:  chatDefaultStreamID(campaignID),
		DefaultPersonaID: "participant:" + participantID,
		Streams:          streams,
		Personas:         personas,
	}
}

func communicationStreamIDs(streams []chatStream) []string {
	ids := make([]string, 0, len(streams))
	for _, stream := range streams {
		if streamID := strings.TrimSpace(stream.StreamID); streamID != "" {
			ids = append(ids, streamID)
		}
	}
	return ids
}

func resolveOutgoingStreamID(state wsCommunicationState, requested string) (string, error) {
	streamID := strings.TrimSpace(requested)
	if streamID == "" {
		streamID = strings.TrimSpace(state.defaultStreamID)
	}
	if streamID == "" {
		return "", errors.New("stream_id is required")
	}
	if _, ok := state.streamsByID[streamID]; !ok {
		return "", errors.New("stream is not available to this participant")
	}
	return streamID, nil
}

func resolveOutgoingActor(state wsCommunicationState, fallbackUserID string, requestedPersonaID string) (messageActor, error) {
	personaID := strings.TrimSpace(requestedPersonaID)
	if personaID == "" {
		personaID = strings.TrimSpace(state.defaultPersonaID)
	}
	if personaID == "" {
		personaID = "participant:" + strings.TrimSpace(fallbackUserID)
	}
	persona, ok := state.personasByID[personaID]
	if !ok {
		return messageActor{}, errors.New("persona is not available to this participant")
	}
	actor := messageActor{
		ParticipantID: strings.TrimSpace(persona.ParticipantID),
		CharacterID:   strings.TrimSpace(persona.CharacterID),
		PersonaID:     persona.PersonaID,
		Name:          persona.DisplayName,
	}
	switch strings.TrimSpace(persona.Kind) {
	case "character":
		actor.Mode = "character"
	default:
		actor.Mode = "participant"
	}
	return actor, nil
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("failed to marshal websocket frame payload: %v", err)
		return nil
	}
	return b
}
