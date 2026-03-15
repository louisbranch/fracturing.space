package server

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/net/websocket"
	gogrpccodes "google.golang.org/grpc/codes"
	gogrpcstatus "google.golang.org/grpc/status"
)

// NewHandler creates chat routes for tests and offline paths.
// WebSocket auth is intentionally disabled in this constructor.
func NewHandler() http.Handler {
	return newHandler(nil, false, nil)
}

// NewHandlerWithAuthorizer creates chat routes with enforced websocket identity checks.
func NewHandlerWithAuthorizer(authorizer wsAuthorizer) http.Handler {
	return newHandler(authorizer, true, nil)
}

func newHandler(authorizer wsAuthorizer, requireAuth bool, hub *roomHub) http.Handler {
	if hub == nil {
		hub = newRoomHub()
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/up", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	wsHandler := websocket.Handler(func(conn *websocket.Conn) {
		handleWSConn(conn, hub, authorizer)
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

func handleWSConn(conn *websocket.Conn, hub *roomHub, authorizer wsAuthorizer) {
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
			leaveSessionRoom(room, session)
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
			handleJoinFrame(conn.Request().Context(), session, hub, authorizer, frame)
		case "chat.send":
			handleSendFrame(session, frame)
		case "chat.history.before":
			handleHistoryBeforeFrame(session, frame)
		default:
			_ = writeWSError(session.peer, frame.RequestID, "INVALID_ARGUMENT", "unsupported frame type")
		}
	}
}

func leaveSessionRoom(room *sessionRoom, session *wsSession) {
	if room == nil || session == nil {
		return
	}
	if session.currentRoom() == room {
		session.setRoom(nil)
	}
	_ = room.leave(session)
}

func handleJoinFrame(ctx context.Context, session *wsSession, hub *roomHub, authorizer wsAuthorizer, frame wsFrame) {
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
	sessionID := strings.TrimSpace(payload.SessionID)
	if sessionID == "" {
		_ = writeWSError(session.peer, frame.RequestID, "INVALID_ARGUMENT", "session_id is required")
		return
	}

	welcome, err := resolveJoinWelcome(ctx, authorizer, campaignID, sessionID, session.userID)
	if err != nil {
		if errors.Is(err, errCampaignParticipantRequired) {
			log.Printf("chat: campaign participant required for user=%q campaign=%q session=%q", session.userID, campaignID, sessionID)
			_ = writeWSError(session.peer, frame.RequestID, "FORBIDDEN", "participant access required for campaign")
			return
		}
		if grpcStatus := gogrpcstatus.Convert(err); grpcStatus.Code() != gogrpccodes.Unknown {
			log.Printf("chat: join context rejected user=%q campaign=%q session=%q code=%s err=%v", session.userID, campaignID, sessionID, grpcStatus.Code(), err)
			_ = writeWSRPCError(session.peer, frame.RequestID, err)
			return
		}
		log.Printf("chat: failed to resolve join context user=%q campaign=%q session=%q err=%v", session.userID, campaignID, sessionID, err)
		_ = writeWSError(session.peer, frame.RequestID, "UNAVAILABLE", "session chat lookup unavailable")
		return
	}

	room := hub.room(campaignID, welcome.SessionID)
	session.setJoinState(welcome)
	previous := session.setRoom(room)
	if previous != nil && previous != room {
		leaveSessionRoom(previous, session)
	}
	latest, history := room.joinWithHistory(session, payload.LastSequenceID)

	_ = session.peer.writeFrame(wsFrame{
		Type: "chat.joined",
		Payload: mustJSON(joinedPayload{
			CampaignID:       campaignID,
			CampaignName:     welcome.CampaignName,
			SessionID:        welcome.SessionID,
			SessionName:      welcome.SessionName,
			ParticipantID:    welcome.ParticipantID,
			ParticipantName:  welcome.ParticipantName,
			LatestSequenceID: latest,
			ServerTime:       time.Now().UTC().Format(time.RFC3339),
		}),
	})

	if payload.LastSequenceID >= latest {
		return
	}
	for _, msg := range history {
		_ = session.peer.writeFrame(wsFrame{
			Type:    "chat.message",
			Payload: mustJSON(messageEnvelope{Message: msg}),
		})
	}
}

func resolveJoinWelcome(ctx context.Context, authorizer wsAuthorizer, campaignID string, sessionID string, userID string) (joinWelcome, error) {
	if authorizer != nil {
		return authorizer.ResolveJoinWelcome(ctx, campaignID, sessionID, userID)
	}
	return joinWelcome{
		ParticipantID:   strings.TrimSpace(userID),
		ParticipantName: strings.TrimSpace(userID),
		CampaignName:    strings.TrimSpace(campaignID),
		SessionID:       strings.TrimSpace(sessionID),
		SessionName:     strings.TrimSpace(sessionID),
	}, nil
}

func handleSendFrame(session *wsSession, frame wsFrame) {
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
		_ = writeWSError(session.peer, frame.RequestID, "FORBIDDEN", "must join session chat before sending")
		return
	}

	state := session.joinState()
	msg, duplicate, subscribers := room.appendMessage(messageActor{
		ParticipantID: state.participantID,
		Name:          state.participantName,
	}, body, clientMessageID)

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
		_ = writeWSError(session.peer, frame.RequestID, "FORBIDDEN", "must join session chat before requesting history")
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

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("failed to marshal websocket frame payload: %v", err)
		return nil
	}
	return b
}
