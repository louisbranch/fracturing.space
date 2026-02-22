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
	"sync"
	"time"
	"unicode/utf8"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"golang.org/x/net/websocket"
	gogrpc "google.golang.org/grpc"
)

const (
	tokenCookieName = "fs_token"

	defaultSessionID = "active"

	maxFramePayloadBytes   = 16 * 1024
	maxFramesPerSecond     = 40
	maxDecodeErrorsPerConn = 3

	maxMessageBodyRunes     = 2000
	maxClientMessageIDRunes = 128

	maxRoomMessages      = 1000
	maxIdempotencyRecord = 4000
)

var errCampaignSessionInactive = errors.New("campaign has no active session")
var errCampaignParticipantRequired = errors.New("campaign participant access required")

// Config defines the inputs for the chat transport boundary.
//
// The settings intentionally couple the chat WebSocket layer to game membership and
// auth token introspection without owning gameplay state.
type Config struct {
	HTTPAddr            string
	GameAddr            string
	AuthBaseURL         string
	OAuthResourceSecret string
	GRPCDialTimeout     time.Duration
	ReadHeaderTimeout   time.Duration
	ShutdownTimeout     time.Duration
}

// Server hosts the chat HTTP/WebSocket process.
//
// It delegates campaign membership and identity resolution to external service
// clients so chat remains transport-only.
type Server struct {
	httpAddr                       string
	shutdownTimeout                time.Duration
	httpServer                     *http.Server
	gameConn                       *gogrpc.ClientConn
	campaignUpdateSubscriptionDone chan struct{}
	campaignUpdateSubscriptionStop context.CancelFunc
}

type wsFrame struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

type wsErrorEnvelope struct {
	Error wsError `json:"error"`
}

type wsError struct {
	Code      string         `json:"code"`
	Message   string         `json:"message"`
	Retryable bool           `json:"retryable"`
	Details   map[string]any `json:"details,omitempty"`
}

type gameGRPCClients struct {
	conn              *gogrpc.ClientConn
	participantClient statev1.ParticipantServiceClient
	eventClient       statev1.EventServiceClient
}

type joinPayload struct {
	CampaignID     string `json:"campaign_id"`
	LastSequenceID int64  `json:"last_sequence_id,omitempty"`
}

type joinedPayload struct {
	CampaignID       string `json:"campaign_id"`
	SessionID        string `json:"session_id"`
	LatestSequenceID int64  `json:"latest_sequence_id"`
	ServerTime       string `json:"server_time"`
}

type sendPayload struct {
	ClientMessageID string `json:"client_message_id"`
	Body            string `json:"body"`
}

type historyBeforePayload struct {
	BeforeSequenceID int64 `json:"before_sequence_id"`
	Limit            int   `json:"limit"`
}

type messageEnvelope struct {
	Message chatMessage `json:"message"`
}

type chatMessage struct {
	MessageID       string       `json:"message_id"`
	CampaignID      string       `json:"campaign_id"`
	SessionID       string       `json:"session_id"`
	SequenceID      int64        `json:"sequence_id"`
	SentAt          string       `json:"sent_at"`
	Kind            string       `json:"kind"`
	Actor           messageActor `json:"actor"`
	Body            string       `json:"body"`
	ClientMessageID string       `json:"client_message_id,omitempty"`
}

type messageActor struct {
	ParticipantID string `json:"participant_id"`
	Name          string `json:"name"`
}

type ackEnvelope struct {
	Result ackResult `json:"result"`
}

type ackResult struct {
	Status     string `json:"status"`
	MessageID  string `json:"message_id,omitempty"`
	SequenceID int64  `json:"sequence_id,omitempty"`
	Count      int    `json:"count,omitempty"`
}

type wsSession struct {
	mu     sync.Mutex
	userID string
	room   *campaignRoom
	peer   *wsPeer
}

func newWSSession(userID string, peer *wsPeer) *wsSession {
	return &wsSession{
		userID: userID,
		peer:   peer,
	}
}

func (s *wsSession) setRoom(next *campaignRoom) *campaignRoom {
	s.mu.Lock()
	previous := s.room
	s.room = next
	s.mu.Unlock()
	return previous
}

func (s *wsSession) currentRoom() *campaignRoom {
	s.mu.Lock()
	room := s.room
	s.mu.Unlock()
	return room
}

type wsPeer struct {
	mu      sync.Mutex
	encoder *json.Encoder
}

func newWSPeer(encoder *json.Encoder) *wsPeer {
	return &wsPeer{encoder: encoder}
}

func (p *wsPeer) writeFrame(frame wsFrame) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.encoder.Encode(frame)
}

type roomHub struct {
	mu    sync.Mutex
	rooms map[string]*campaignRoom
}

func newRoomHub() *roomHub {
	return &roomHub{rooms: make(map[string]*campaignRoom)}
}

func (h *roomHub) room(campaignID string) *campaignRoom {
	h.mu.Lock()
	defer h.mu.Unlock()

	room, ok := h.rooms[campaignID]
	if ok {
		return room
	}

	room = newCampaignRoom(campaignID)
	h.rooms[campaignID] = room
	return room
}

type campaignRoom struct {
	mu               sync.Mutex
	campaignID       string
	sessionID        string
	nextSequence     int64
	messages         []chatMessage
	idempotencyBy    map[string]chatMessage
	idempotencyOrder []string
	subscribers      map[*wsPeer]struct{}
}

func newCampaignRoom(campaignID string) *campaignRoom {
	return &campaignRoom{
		campaignID:    campaignID,
		sessionID:     defaultSessionID,
		idempotencyBy: make(map[string]chatMessage),
		subscribers:   make(map[*wsPeer]struct{}),
	}
}

func (r *campaignRoom) join(peer *wsPeer) int64 {
	r.mu.Lock()
	r.subscribers[peer] = struct{}{}
	latest := r.nextSequence
	r.mu.Unlock()
	return latest
}

func (r *campaignRoom) setSessionID(sessionID string) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	r.mu.Lock()
	r.sessionID = sessionID
	r.mu.Unlock()
}

func (r *campaignRoom) currentSessionID() string {
	r.mu.Lock()
	id := r.sessionID
	r.mu.Unlock()
	return id
}

func (r *campaignRoom) leave(peer *wsPeer) bool {
	r.mu.Lock()
	delete(r.subscribers, peer)
	empty := len(r.subscribers) == 0
	r.mu.Unlock()
	return empty
}

func (r *campaignRoom) appendMessage(actorID string, body string, clientMessageID string) (chatMessage, bool, []*wsPeer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existing, ok := r.idempotencyBy[clientMessageID]; ok {
		return existing, true, nil
	}

	r.nextSequence++
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		actorID = "participant"
	}
	msg := chatMessage{
		MessageID:  fmt.Sprintf("msg_%d", time.Now().UnixNano()),
		CampaignID: r.campaignID,
		SessionID:  r.sessionID,
		SequenceID: r.nextSequence,
		SentAt:     time.Now().UTC().Format(time.RFC3339),
		Kind:       "text",
		Actor: messageActor{
			ParticipantID: actorID,
			Name:          actorID,
		},
		Body:            body,
		ClientMessageID: clientMessageID,
	}

	r.messages = append(r.messages, msg)
	if len(r.messages) > maxRoomMessages {
		r.messages = r.messages[len(r.messages)-maxRoomMessages:]
	}

	r.idempotencyBy[clientMessageID] = msg
	r.idempotencyOrder = append(r.idempotencyOrder, clientMessageID)
	if len(r.idempotencyOrder) > maxIdempotencyRecord {
		evict := r.idempotencyOrder[0]
		r.idempotencyOrder = r.idempotencyOrder[1:]
		delete(r.idempotencyBy, evict)
	}

	subscribers := make([]*wsPeer, 0, len(r.subscribers))
	for subscriber := range r.subscribers {
		subscribers = append(subscribers, subscriber)
	}
	return msg, false, subscribers
}

func (r *campaignRoom) historyBefore(beforeSequenceID int64, limit int) []chatMessage {
	r.mu.Lock()
	defer r.mu.Unlock()

	history := make([]chatMessage, 0, limit)
	for _, msg := range r.messages {
		if msg.SequenceID < beforeSequenceID {
			history = append(history, msg)
		}
	}
	if len(history) > limit {
		history = history[len(history)-limit:]
	}
	return history
}

type wsAuthorizer interface {
	Authenticate(ctx context.Context, accessToken string) (string, error)
	IsCampaignParticipant(ctx context.Context, campaignID string, userID string) (bool, error)
}

type wsJoinWelcomeProvider interface {
	ResolveJoinWelcome(ctx context.Context, campaignID string, userID string) (joinWelcome, error)
}

type joinWelcome struct {
	ParticipantName string
	CampaignName    string
	SessionID       string
	SessionName     string
	Locale          commonv1.Locale
}

type campaignAuthorizer struct {
	authBaseURL         string
	oauthResourceSecret string
	httpClient          *http.Client
	participantClient   statev1.ParticipantServiceClient
	sessionClient       statev1.SessionServiceClient
	campaignClient      statev1.CampaignServiceClient
}

type authIntrospectResponse struct {
	Active bool   `json:"active"`
	UserID string `json:"user_id"`
}

func newCampaignAuthorizer(
	config Config,
	participantClient statev1.ParticipantServiceClient,
	sessionClient statev1.SessionServiceClient,
	campaignClient statev1.CampaignServiceClient,
) wsAuthorizer {
	authBaseURL := strings.TrimSpace(config.AuthBaseURL)
	resourceSecret := strings.TrimSpace(config.OAuthResourceSecret)
	if authBaseURL == "" || resourceSecret == "" {
		return nil
	}

	return &campaignAuthorizer{
		authBaseURL:         authBaseURL,
		oauthResourceSecret: resourceSecret,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		participantClient: participantClient,
		sessionClient:     sessionClient,
		campaignClient:    campaignClient,
	}
}

func (a *campaignAuthorizer) Authenticate(ctx context.Context, accessToken string) (string, error) {
	if a == nil || a.httpClient == nil {
		return "", errors.New("auth is not configured")
	}
	accessToken = strings.TrimSpace(accessToken)
	if accessToken == "" {
		return "", errors.New("access token is required")
	}

	endpoint := strings.TrimRight(a.authBaseURL, "/") + "/introspect"
	authCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(authCtx, http.MethodPost, endpoint, nil)
	if err != nil {
		return "", fmt.Errorf("build introspection request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("X-Resource-Secret", a.oauthResourceSecret)

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("call auth introspection: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("auth introspection status %d", resp.StatusCode)
	}

	var payload authIntrospectResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", fmt.Errorf("decode introspection response: %w", err)
	}
	if !payload.Active {
		return "", errors.New("inactive access token")
	}

	userID := strings.TrimSpace(payload.UserID)
	if userID == "" {
		return "", errors.New("introspection returned empty user id")
	}
	return userID, nil
}

func (a *campaignAuthorizer) IsCampaignParticipant(ctx context.Context, campaignID string, userID string) (bool, error) {
	if a == nil || a.participantClient == nil {
		return false, errors.New("participant client is not configured")
	}

	campaignID = strings.TrimSpace(campaignID)
	userID = strings.TrimSpace(userID)
	if campaignID == "" || userID == "" {
		return false, nil
	}

	participant, err := a.findParticipantByUserID(ctx, campaignID, userID)
	if err != nil {
		return false, err
	}
	return participant != nil, nil
}

func (a *campaignAuthorizer) ResolveJoinWelcome(ctx context.Context, campaignID string, userID string) (joinWelcome, error) {
	campaignID = strings.TrimSpace(campaignID)
	userID = strings.TrimSpace(userID)
	if campaignID == "" {
		return joinWelcome{}, errors.New("campaign id is required")
	}

	var activeSession *statev1.Session
	if a.sessionClient != nil {
		var err error
		activeSession, err = a.findActiveSession(ctx, campaignID, userID)
		if err != nil && !errors.Is(err, errCampaignSessionInactive) {
			return joinWelcome{}, err
		}
	}

	participant, err := a.findParticipantByUserID(ctx, campaignID, userID)
	if err != nil {
		return joinWelcome{}, err
	}
	if participant == nil {
		return joinWelcome{}, errCampaignParticipantRequired
	}

	participantName := userID
	if strings.TrimSpace(participant.GetName()) != "" {
		participantName = strings.TrimSpace(participant.GetName())
	}

	campaignName := campaignID
	locale := commonv1.Locale_LOCALE_EN_US
	if a.campaignClient != nil {
		callCtx, cancel := context.WithTimeout(grpcauthctx.WithUserID(ctx, userID), 3*time.Second)
		resp, err := a.campaignClient.GetCampaign(callCtx, &statev1.GetCampaignRequest{CampaignId: campaignID})
		cancel()
		if err != nil {
			return joinWelcome{}, fmt.Errorf("get campaign: %w", err)
		}
		if campaign := resp.GetCampaign(); campaign != nil {
			if name := strings.TrimSpace(campaign.GetName()); name != "" {
				campaignName = name
			}
			if campaign.GetLocale() != commonv1.Locale_LOCALE_UNSPECIFIED {
				locale = campaign.GetLocale()
			}
		}
	}

	sessionID := ""
	sessionName := ""
	if activeSession != nil {
		sessionID = strings.TrimSpace(activeSession.GetId())
		sessionName = strings.TrimSpace(activeSession.GetName())
		if sessionName == "" {
			sessionName = sessionID
		}
	}

	return joinWelcome{
		ParticipantName: strings.TrimSpace(participantName),
		CampaignName:    campaignName,
		SessionID:       sessionID,
		SessionName:     sessionName,
		Locale:          locale,
	}, nil
}

func (a *campaignAuthorizer) findActiveSession(ctx context.Context, campaignID string, userID string) (*statev1.Session, error) {
	if a == nil || a.sessionClient == nil {
		return nil, errors.New("session client is not configured")
	}
	pageToken := ""
	for {
		callCtx, cancel := context.WithTimeout(grpcauthctx.WithUserID(ctx, userID), 3*time.Second)
		resp, err := a.sessionClient.ListSessions(callCtx, &statev1.ListSessionsRequest{
			CampaignId: campaignID,
			PageSize:   10,
			PageToken:  pageToken,
		})
		cancel()
		if err != nil {
			return nil, fmt.Errorf("list campaign sessions: %w", err)
		}
		for _, session := range resp.GetSessions() {
			if session.GetStatus() == statev1.SessionStatus_SESSION_ACTIVE {
				return session, nil
			}
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}
	return nil, errCampaignSessionInactive
}

func (a *campaignAuthorizer) findParticipantByUserID(ctx context.Context, campaignID string, userID string) (*statev1.Participant, error) {
	if a == nil || a.participantClient == nil {
		return nil, errors.New("participant client is not configured")
	}
	pageToken := ""
	for {
		callCtx, cancel := context.WithTimeout(grpcauthctx.WithUserID(ctx, userID), 3*time.Second)
		resp, err := a.participantClient.ListParticipants(callCtx, &statev1.ListParticipantsRequest{
			CampaignId: campaignID,
			PageSize:   100,
			PageToken:  pageToken,
		})
		cancel()
		if err != nil {
			return nil, fmt.Errorf("list campaign participants: %w", err)
		}
		for _, participant := range resp.GetParticipants() {
			if strings.TrimSpace(participant.GetUserId()) == userID {
				return participant, nil
			}
		}
		pageToken = strings.TrimSpace(resp.GetNextPageToken())
		if pageToken == "" {
			break
		}
	}
	return nil, nil
}

type wsUserIDContextKey struct{}

// NewHandler creates chat routes for tests and offline paths.
// WebSocket auth is intentionally disabled in this constructor.
func NewHandler() http.Handler {
	return newHandler(nil, false, nil, nil)
}

// NewHandlerWithAuthorizer creates chat routes with enforced websocket identity checks.
func NewHandlerWithAuthorizer(authorizer wsAuthorizer) http.Handler {
	return newHandler(authorizer, true, nil, nil)
}

func newHandler(authorizer wsAuthorizer, requireAuth bool, ensureCampaignUpdateSubscription func(string), releaseCampaignUpdateSubscription func(string)) http.Handler {
	hub := newRoomHub()
	mux := http.NewServeMux()
	mux.HandleFunc("/up", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})

	wsHandler := websocket.Handler(func(conn *websocket.Conn) {
		handleWSConn(conn, hub, authorizer, ensureCampaignUpdateSubscription, releaseCampaignUpdateSubscription)
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
				log.Printf("chat: websocket unauthorized: missing fs_token for host=%q remote=%s path=%q", r.Host, r.RemoteAddr, r.URL.Path)
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
	if err != nil {
		return ""
	}
	return strings.TrimSpace(cookie.Value)
}

func handleWSConn(conn *websocket.Conn, hub *roomHub, authorizer wsAuthorizer, ensureCampaignUpdateSubscription func(string), releaseCampaignUpdateSubscription func(string)) {
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
			leaveCampaignRoom(room, session.peer, releaseCampaignUpdateSubscription)
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
			handleJoinFrame(conn.Request().Context(), session, hub, authorizer, frame, ensureCampaignUpdateSubscription, releaseCampaignUpdateSubscription)
		case "chat.send":
			handleSendFrame(session, frame)
		case "chat.history.before":
			handleHistoryBeforeFrame(session, frame)
		default:
			_ = writeWSError(session.peer, frame.RequestID, "INVALID_ARGUMENT", "unsupported frame type")
		}
	}
}

func leaveCampaignRoom(room *campaignRoom, peer *wsPeer, releaseCampaignUpdateSubscription func(string)) {
	if room == nil || peer == nil {
		return
	}
	if room.leave(peer) && releaseCampaignUpdateSubscription != nil {
		releaseCampaignUpdateSubscription(room.campaignID)
	}
}

func handleJoinFrame(ctx context.Context, session *wsSession, hub *roomHub, authorizer wsAuthorizer, frame wsFrame, ensureCampaignUpdateSubscription func(string), releaseCampaignUpdateSubscription func(string)) {
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
			if errors.Is(err, errCampaignSessionInactive) {
				log.Printf("chat: campaign has no active session for user=%q campaign=%q", session.userID, campaignID)
				_ = writeWSError(session.peer, frame.RequestID, "FAILED_PRECONDITION", "campaign session is not active")
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
			if errors.Is(err, errCampaignSessionInactive) {
				log.Printf("chat: campaign session inactive during membership check for user=%q campaign=%q", session.userID, campaignID)
				_ = writeWSError(session.peer, frame.RequestID, "FAILED_PRECONDITION", "campaign session is not active")
				return
			}
			log.Printf("chat: campaign membership check failed user=%q campaign=%q err=%v", session.userID, campaignID, err)
			_ = writeWSError(session.peer, frame.RequestID, "UNAVAILABLE", "campaign membership verification unavailable")
			return
		}
		if !allowed {
			_ = writeWSError(session.peer, frame.RequestID, "FORBIDDEN", "participant access required for campaign")
			return
		}
	}
	if ensureCampaignUpdateSubscription != nil {
		ensureCampaignUpdateSubscription(campaignID)
	}

	room := hub.room(campaignID)
	room.setSessionID(welcome.SessionID)
	if strings.TrimSpace(welcome.SessionName) == "" {
		welcome.SessionName = room.currentSessionID()
	}
	previous := session.setRoom(room)
	if previous != nil && previous != room {
		leaveCampaignRoom(previous, session.peer, releaseCampaignUpdateSubscription)
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

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		log.Printf("failed to marshal websocket frame payload: %v", err)
		return nil
	}
	return b
}

// NewServer builds a configured chat server and wires membership checks if game is reachable.
func NewServer(config Config) (*Server, error) {
	return NewServerWithContext(context.Background(), config)
}

// NewServerWithContext builds a configured chat server with an explicit context.
func NewServerWithContext(ctx context.Context, config Config) (*Server, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	httpAddr := strings.TrimSpace(config.HTTPAddr)
	if httpAddr == "" {
		return nil, errors.New("http address is required")
	}
	if config.ReadHeaderTimeout <= 0 {
		config.ReadHeaderTimeout = timeouts.ReadHeader
	}
	if config.ShutdownTimeout <= 0 {
		config.ShutdownTimeout = timeouts.Shutdown
	}
	if config.GRPCDialTimeout <= 0 {
		config.GRPCDialTimeout = timeouts.GRPCDial
	}

	var gameConn *gogrpc.ClientConn
	var participantClient statev1.ParticipantServiceClient
	var sessionClient statev1.SessionServiceClient
	var campaignClient statev1.CampaignServiceClient
	var eventClient statev1.EventServiceClient
	if strings.TrimSpace(config.GameAddr) != "" {
		clients, err := dialGameGRPC(ctx, config)
		if err != nil {
			log.Printf("game gRPC dial failed, campaign membership checks unavailable: %v", err)
		} else {
			gameConn = clients.conn
			participantClient = clients.participantClient
			sessionClient = statev1.NewSessionServiceClient(clients.conn)
			campaignClient = statev1.NewCampaignServiceClient(clients.conn)
			eventClient = clients.eventClient
		}
	}

	authorizer := newCampaignAuthorizer(config, participantClient, sessionClient, campaignClient)
	ensureCampaignUpdateSubscription, releaseCampaignUpdateSubscription, campaignUpdateStop, campaignUpdateDone := startCampaignEventCommittedSubscriptionWorker(eventClient)
	httpServer := &http.Server{
		Addr:              httpAddr,
		Handler:           newHandler(authorizer, true, ensureCampaignUpdateSubscription, releaseCampaignUpdateSubscription),
		ReadHeaderTimeout: config.ReadHeaderTimeout,
	}

	return &Server{
		httpAddr:                       httpAddr,
		shutdownTimeout:                config.ShutdownTimeout,
		httpServer:                     httpServer,
		gameConn:                       gameConn,
		campaignUpdateSubscriptionDone: campaignUpdateDone,
		campaignUpdateSubscriptionStop: campaignUpdateStop,
	}, nil
}

// Run creates and serves a chat server until the context ends.
//
// Operators can treat this as the lifecycle boundary for the real-time surface.
func Run(ctx context.Context, config Config) error {
	server, err := NewServer(config)
	if err != nil {
		return fmt.Errorf("init chat server: %w", err)
	}
	defer server.Close()

	if err := server.ListenAndServe(ctx); err != nil {
		return fmt.Errorf("serve chat: %w", err)
	}
	return nil
}

// ListenAndServe runs the HTTP server until the context ends.
func (s *Server) ListenAndServe(ctx context.Context) error {
	if s == nil {
		return errors.New("chat server is nil")
	}
	if ctx == nil {
		return errors.New("context is required")
	}

	serveErr := make(chan error, 1)
	log.Printf("chat server listening on %s", s.httpAddr)
	go func() {
		serveErr <- s.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.shutdownTimeout)
		err := s.httpServer.Shutdown(shutdownCtx)
		cancel()
		if err != nil {
			return fmt.Errorf("shutdown http server: %w", err)
		}
		return nil
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve http: %w", err)
	}
}

// Close releases server resources.
func (s *Server) Close() {
	if s == nil {
		return
	}
	if s.campaignUpdateSubscriptionStop != nil {
		s.campaignUpdateSubscriptionStop()
	}
	if s.campaignUpdateSubscriptionDone != nil {
		<-s.campaignUpdateSubscriptionDone
	}
	if s.gameConn != nil {
		if err := s.gameConn.Close(); err != nil {
			log.Printf("close game gRPC connection: %v", err)
		}
	}
}

func dialGameGRPC(ctx context.Context, config Config) (gameGRPCClients, error) {
	gameAddr := strings.TrimSpace(config.GameAddr)
	if gameAddr == "" {
		return gameGRPCClients{}, nil
	}
	if ctx == nil {
		return gameGRPCClients{}, errors.New("context is required")
	}
	if config.GRPCDialTimeout <= 0 {
		config.GRPCDialTimeout = timeouts.GRPCDial
	}

	logf := func(format string, args ...any) {
		log.Printf("game %s", fmt.Sprintf(format, args...))
	}

	conn, err := platformgrpc.DialWithHealth(
		ctx,
		nil,
		gameAddr,
		config.GRPCDialTimeout,
		logf,
		platformgrpc.DefaultClientDialOptions()...,
	)
	if err != nil {
		return gameGRPCClients{}, fmt.Errorf("dial game gRPC %s: %w", gameAddr, err)
	}
	return gameGRPCClients{
		conn:              conn,
		participantClient: statev1.NewParticipantServiceClient(conn),
		eventClient:       statev1.NewEventServiceClient(conn),
	}, nil
}
