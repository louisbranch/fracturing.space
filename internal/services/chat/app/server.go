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

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
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

	maxFramePayloadBytes   = 16 * 1024
	maxFramesPerSecond     = 40
	maxDecodeErrorsPerConn = 3

	maxMessageBodyRunes     = 12000
	maxClientMessageIDRunes = 128

	maxRoomMessages      = 1000
	maxIdempotencyRecord = 4000
)

// newManagedConn wraps platformgrpc.NewManagedConn for testability.
var newManagedConn = platformgrpc.NewManagedConn

var errCampaignParticipantRequired = errors.New("campaign participant access required")

// Config defines the inputs for the chat transport boundary.
//
// Chat remains transport-only: it authenticates the caller and validates
// campaign/session membership, but it does not own gameplay routing.
type Config struct {
	HTTPAddr            string
	AuthAddr            string
	GameAddr            string
	AuthBaseURL         string
	OAuthResourceSecret string
	ReadHeaderTimeout   time.Duration
	ShutdownTimeout     time.Duration
}

// Server hosts the chat HTTP/WebSocket process.
//
// It delegates campaign/session validation to game and identity resolution to
// auth so chat only owns realtime delivery concerns.
type Server struct {
	httpAddr        string
	shutdownTimeout time.Duration
	httpServer      *http.Server
	gameMc          *platformgrpc.ManagedConn
	authMc          *platformgrpc.ManagedConn
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
	SessionID      string `json:"session_id"`
	LastSequenceID int64  `json:"last_sequence_id,omitempty"`
}

type joinedPayload struct {
	CampaignID       string `json:"campaign_id"`
	CampaignName     string `json:"campaign_name,omitempty"`
	SessionID        string `json:"session_id"`
	SessionName      string `json:"session_name,omitempty"`
	ParticipantID    string `json:"participant_id,omitempty"`
	ParticipantName  string `json:"participant_name,omitempty"`
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

type wsJoinState struct {
	participantID   string
	participantName string
}

type wsSession struct {
	mu     sync.Mutex
	userID string
	room   *sessionRoom
	peer   *wsPeer
	state  wsJoinState
}

type wsAuthorizer interface {
	Authenticate(ctx context.Context, accessToken string) (string, error)
	ResolveJoinWelcome(ctx context.Context, campaignID string, sessionID string, userID string) (joinWelcome, error)
}

type joinWelcome struct {
	ParticipantID   string
	ParticipantName string
	CampaignName    string
	SessionID       string
	SessionName     string
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
	var authMc *platformgrpc.ManagedConn
	var authSessionClient webSessionAuthClient
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
	httpServer := &http.Server{
		Addr:              httpAddr,
		Handler:           newHandler(authorizer, true, newRoomHub()),
		ReadHeaderTimeout: config.ReadHeaderTimeout,
	}

	return &Server{
		httpAddr:        httpAddr,
		shutdownTimeout: config.ShutdownTimeout,
		httpServer:      httpServer,
		gameMc:          gameMc,
		authMc:          authMc,
	}, nil
}

// Run creates and serves a chat server until the context ends.
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
	closeManagedConn(s.gameMc, "game")
	closeManagedConn(s.authMc, "auth")
}

func closeManagedConn(mc *platformgrpc.ManagedConn, name string) {
	if mc == nil {
		return
	}
	if err := mc.Close(); err != nil {
		log.Printf("close chat %s managed conn: %v", name, err)
	}
}
