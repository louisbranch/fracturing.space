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
			leaveCampaignRoom(room, session.peer, releaseCampaignUpdateSubscription, releaseAITurnSubscription)
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
				conn.Request().Context(),
				session,
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
	peer *wsPeer,
	releaseCampaignUpdateSubscription func(string),
	releaseAITurnSubscription func(string),
) {
	if room == nil || peer == nil {
		return
	}
	if room.leave(peer) {
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

	welcome := joinWelcome{
		ParticipantName: session.userID,
		CampaignName:    campaignID,
		SessionID:       "",
		SessionName:     "",
		GmMode:          "",
		AIAgentID:       "",
		Locale:          commonv1.Locale_LOCALE_EN_US,
	}
	if provider, ok := authorizer.(wsJoinWelcomeProvider); ok {
		resolved, err := provider.ResolveJoinWelcome(ctx, campaignID, session.userID)
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
		welcome = resolved
	} else if authorizer != nil {
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
	}
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
	previous := session.setRoom(room)
	if previous != nil && previous != room {
		leaveCampaignRoom(previous, session.peer, releaseCampaignUpdateSubscription, releaseAITurnSubscription)
	}
	latest := room.join(session.peer)

	_ = session.peer.writeFrame(wsFrame{
		Type: "chat.joined",
		Payload: mustJSON(joinedPayload{
			CampaignID:       campaignID,
			SessionID:        room.currentSessionID(),
			LatestSequenceID: latest,
			ServerTime:       time.Now().UTC().Format(time.RFC3339),
		}),
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
				Actor: messageActor{
					ParticipantID: "system",
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
	ctx context.Context,
	session *wsSession,
	frame wsFrame,
	aiInvocationClient aiv1.InvocationServiceClient,
	ensureAITurnSubscription func(string, string, string),
	issueAISessionGrant func(context.Context, *campaignRoom, string) error,
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
		_ = writeWSError(session.peer, frame.RequestID, "INVALID_ARGUMENT", "body must be at most 2000 characters")
		return
	}

	room := session.currentRoom()
	if room == nil {
		_ = writeWSError(session.peer, frame.RequestID, "FORBIDDEN", "must join campaign room before sending")
		return
	}

	msg, duplicate, subscribers := room.appendMessage(session.userID, body, clientMessageID)

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

	if aiInvocationClient != nil && room.aiRelayEnabled() {
		if !room.aiRelayReady() && issueAISessionGrant != nil {
			if err := issueAISessionGrant(ctx, room, session.userID); err != nil {
				log.Printf("chat: refresh ai session grant on send failed campaign=%q err=%v", room.campaignID, err)
				room.clearAISessionGrant()
			}
			if room.aiRelayReady() && ensureAITurnSubscription != nil {
				ensureAITurnSubscription(room.campaignID, room.currentSessionID(), room.aiAgentIDValue())
			}
		}
	}

	if aiInvocationClient != nil && room.aiRelayReady() {
		if err := submitCampaignTurnToAI(ctx, aiInvocationClient, room, session, msg); err != nil {
			log.Printf("chat: submit campaign turn to ai failed campaign=%q err=%v", room.campaignID, err)
		}
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

	history := room.historyBefore(payload.BeforeSequenceID, payload.Limit)
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

func submitCampaignTurnToAI(ctx context.Context, invocationClient aiv1.InvocationServiceClient, room *campaignRoom, session *wsSession, msg chatMessage) error {
	if invocationClient == nil || room == nil || session == nil {
		return nil
	}
	aiAgentID := room.aiAgentIDValue()
	if aiAgentID == "" {
		return nil
	}
	callCtx := grpcauthctx.WithUserID(ctx, session.userID)
	_, err := invocationClient.SubmitCampaignTurn(callCtx, &aiv1.SubmitCampaignTurnRequest{
		CampaignId:      room.campaignID,
		SessionId:       room.currentSessionID(),
		AgentId:         aiAgentID,
		ParticipantId:   msg.Actor.ParticipantID,
		ParticipantName: msg.Actor.Name,
		MessageId:       msg.MessageID,
		Body:            msg.Body,
		SessionGrant:    room.aiSessionGrantValue(),
	})
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

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("failed to marshal websocket frame payload: %v", err)
		return nil
	}
	return b
}
