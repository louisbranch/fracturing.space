package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	gogrpc "google.golang.org/grpc"
)

const (
	tokenCookieName      = "fs_token"
	webSessionCookieName = "web_session"

	webSessionTokenPrefix = "web_session:"

	defaultSessionID = "active"

	maxFramePayloadBytes   = 16 * 1024
	maxFramesPerSecond     = 40
	maxDecodeErrorsPerConn = 3

	maxMessageBodyRunes     = 2000
	maxClientMessageIDRunes = 128

	maxRoomMessages      = 1000
	maxIdempotencyRecord = 4000
)

// newManagedConn wraps platformgrpc.NewManagedConn for testability.
var newManagedConn = platformgrpc.NewManagedConn

var errCampaignSessionInactive = errors.New("campaign has no active session")
var errCampaignParticipantRequired = errors.New("campaign participant access required")

// Config defines the inputs for the chat transport boundary.
//
// The settings intentionally couple the chat WebSocket layer to game membership and
// auth token introspection without owning gameplay state.
type Config struct {
	HTTPAddr            string
	AuthAddr            string
	GameAddr            string
	AIAddr              string
	AuthBaseURL         string
	OAuthResourceSecret string
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
	gameMc                         *platformgrpc.ManagedConn
	authMc                         *platformgrpc.ManagedConn
	aiMc                           *platformgrpc.ManagedConn
	campaignUpdateSubscriptionDone chan struct{}
	campaignUpdateSubscriptionStop context.CancelFunc
	aiTurnSubscriptionDone         chan struct{}
	aiTurnSubscriptionStop         context.CancelFunc
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

type webSessionAuthClient interface {
	GetWebSession(context.Context, *authv1.GetWebSessionRequest, ...gogrpc.CallOption) (*authv1.GetWebSessionResponse, error)
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
	GmMode          string
	AIAgentID       string
	Locale          commonv1.Locale
}

type campaignAuthorizer struct {
	authBaseURL         string
	oauthResourceSecret string
	httpClient          *http.Client
	authSessionClient   webSessionAuthClient
	participantClient   statev1.ParticipantServiceClient
	sessionClient       statev1.SessionServiceClient
	campaignClient      statev1.CampaignServiceClient
}

type authIntrospectResponse struct {
	Active bool   `json:"active"`
	UserID string `json:"user_id"`
}

type wsUserIDContextKey struct{}

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
	logf := func(format string, args ...any) {
		log.Printf(format, args...)
	}

	var gameMc *platformgrpc.ManagedConn
	var participantClient statev1.ParticipantServiceClient
	var sessionClient statev1.SessionServiceClient
	var campaignClient statev1.CampaignServiceClient
	var campaignAIClient statev1.CampaignAIServiceClient
	var eventClient statev1.EventServiceClient
	var authMc *platformgrpc.ManagedConn
	var authSessionClient webSessionAuthClient
	var aiMc *platformgrpc.ManagedConn
	var aiInvocationClient aiv1.InvocationServiceClient
	if gameAddr := strings.TrimSpace(config.GameAddr); gameAddr != "" {
		mc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
			Name: "game",
			Addr: gameAddr,
			Mode: platformgrpc.ModeOptional,
			Logf: logf,
			DialOpts: append(
				platformgrpc.LenientDialOptions(),
				gogrpc.WithChainUnaryInterceptor(grpcauthctx.ServiceIDUnaryClientInterceptor(serviceaddr.ServiceChat)),
				gogrpc.WithChainStreamInterceptor(grpcauthctx.ServiceIDStreamClientInterceptor(serviceaddr.ServiceChat)),
			),
		})
		if err != nil {
			log.Printf("game managed conn failed, campaign membership checks unavailable: %v", err)
		} else {
			gameMc = mc
			conn := mc.Conn()
			participantClient = statev1.NewParticipantServiceClient(conn)
			sessionClient = statev1.NewSessionServiceClient(conn)
			campaignClient = statev1.NewCampaignServiceClient(conn)
			campaignAIClient = statev1.NewCampaignAIServiceClient(conn)
			eventClient = statev1.NewEventServiceClient(conn)
		}
	}
	if aiAddr := strings.TrimSpace(config.AIAddr); aiAddr != "" {
		mc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
			Name: "ai",
			Addr: aiAddr,
			Mode: platformgrpc.ModeOptional,
			Logf: logf,
		})
		if err != nil {
			log.Printf("ai managed conn failed, campaign ai relay unavailable: %v", err)
		} else {
			aiMc = mc
			aiInvocationClient = aiv1.NewInvocationServiceClient(mc.Conn())
		}
	}
	if authAddr := strings.TrimSpace(config.AuthAddr); authAddr != "" {
		mc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
			Name: "auth",
			Addr: authAddr,
			Mode: platformgrpc.ModeOptional,
			Logf: logf,
		})
		if err != nil {
			log.Printf("auth managed conn failed, web session chat auth unavailable: %v", err)
		} else {
			authMc = mc
			authSessionClient = authv1.NewAuthServiceClient(mc.Conn())
		}
	}

	authorizer := newCampaignAuthorizer(config, participantClient, sessionClient, campaignClient, authSessionClient)
	roomHub := newRoomHub()
	issueAISessionGrant := func(callCtx context.Context, room *campaignRoom, userID string) error {
		return issueAISessionGrantForRoom(callCtx, campaignAIClient, room, userID)
	}
	var ensureAITurnSubscription func(string, string, string)
	onCampaignEvent := func(campaignID string, eventType string) {
		if !isAICampaignContextEvent(eventType) {
			return
		}
		room := roomHub.roomIfExists(campaignID)
		if room == nil {
			return
		}
		if err := syncRoomAIContextFromGame(ctx, campaignAIClient, room); err != nil {
			log.Printf("chat: sync ai room context failed campaign=%q event=%q err=%v", campaignID, eventType, err)
			room.clearAISessionGrant()
			return
		}
		if !room.aiRelayEnabled() {
			room.clearAISessionGrant()
			return
		}
		if err := issueAISessionGrantForRoom(ctx, campaignAIClient, room, ""); err != nil {
			log.Printf("chat: refresh ai session grant failed campaign=%q event=%q err=%v", campaignID, eventType, err)
			room.clearAISessionGrant()
			return
		}
		if ensureAITurnSubscription != nil {
			ensureAITurnSubscription(campaignID, room.currentSessionID(), room.aiAgentIDValue())
		}
	}
	ensureCampaignUpdateSubscription, releaseCampaignUpdateSubscription, campaignUpdateStop, campaignUpdateDone := startCampaignEventCommittedSubscriptionWorker(ctx, eventClient, onCampaignEvent)
	ensureAITurnSubscription, releaseAITurnSubscription, aiTurnStop, aiTurnDone := startCampaignAITurnSubscriptionWorker(ctx, aiInvocationClient, roomHub)
	httpServer := &http.Server{
		Addr: httpAddr,
		Handler: newHandler(
			authorizer,
			true,
			roomHub,
			ensureCampaignUpdateSubscription,
			releaseCampaignUpdateSubscription,
			ensureAITurnSubscription,
			releaseAITurnSubscription,
			issueAISessionGrant,
			aiInvocationClient,
		),
		ReadHeaderTimeout: config.ReadHeaderTimeout,
	}

	return &Server{
		httpAddr:                       httpAddr,
		shutdownTimeout:                config.ShutdownTimeout,
		httpServer:                     httpServer,
		gameMc:                         gameMc,
		authMc:                         authMc,
		aiMc:                           aiMc,
		campaignUpdateSubscriptionDone: campaignUpdateDone,
		campaignUpdateSubscriptionStop: campaignUpdateStop,
		aiTurnSubscriptionDone:         aiTurnDone,
		aiTurnSubscriptionStop:         aiTurnStop,
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
	if s.aiTurnSubscriptionStop != nil {
		s.aiTurnSubscriptionStop()
	}
	if s.aiTurnSubscriptionDone != nil {
		<-s.aiTurnSubscriptionDone
	}
	closeManagedConn(s.gameMc, "game")
	closeManagedConn(s.authMc, "auth")
	closeManagedConn(s.aiMc, "ai")
}

func closeManagedConn(mc *platformgrpc.ManagedConn, name string) {
	if mc == nil {
		return
	}
	if err := mc.Close(); err != nil {
		log.Printf("close chat %s managed conn: %v", name, err)
	}
}
