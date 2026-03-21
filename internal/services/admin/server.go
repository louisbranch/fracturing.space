package admin

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	platformstatus "github.com/louisbranch/fracturing.space/internal/platform/status"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"google.golang.org/grpc"
)

// adminAuthzOverrideReason records why admin service uses platform override.
const adminAuthzOverrideReason = "admin_dashboard"

// newManagedConn wraps platformgrpc.NewManagedConn for testability.
var newManagedConn = platformgrpc.NewManagedConn

// Config defines the inputs for the admin operator process.
type Config struct {
	HTTPAddr   string
	GRPCAddr   string
	AuthAddr   string
	InviteAddr string
	StatusAddr string
	// AuthConfig enables token-based authentication when set.
	AuthConfig *AuthConfig
	// StatusReporter receives health transitions for dependency capabilities.
	StatusReporter *platformstatus.Reporter
}

// Server hosts the admin dashboard and manages authenticated gRPC clients.
type Server struct {
	httpAddr string

	gameMc   *platformgrpc.ManagedConn
	authMc   *platformgrpc.ManagedConn
	inviteMc *platformgrpc.ManagedConn
	statusMc *platformgrpc.ManagedConn

	// Game service clients — created at construction, always non-nil when gameMc exists.
	daggerheartClient daggerheartv1.DaggerheartServiceClient
	contentClient     daggerheartv1.DaggerheartContentServiceClient
	campaignClient    statev1.CampaignServiceClient
	sessionClient     statev1.SessionServiceClient
	characterClient   statev1.CharacterServiceClient
	participantClient statev1.ParticipantServiceClient
	snapshotClient    statev1.SnapshotServiceClient
	eventClient       statev1.EventServiceClient
	statisticsClient  statev1.StatisticsServiceClient
	systemClient      statev1.SystemServiceClient

	// Invite service client — created at construction, always non-nil when inviteMc exists.
	inviteClient invitev1.InviteServiceClient

	// Auth service clients — created at construction, always non-nil when authMc exists.
	authClient    authv1.AuthServiceClient
	accountClient authv1.AccountServiceClient

	statusClient statusv1.StatusServiceClient
	httpServer   *http.Server
}

// CampaignClient returns the current campaign client.
func (s *Server) CampaignClient() statev1.CampaignServiceClient {
	if s == nil {
		return nil
	}
	return s.campaignClient
}

// AuthClient returns the current auth client.
func (s *Server) AuthClient() authv1.AuthServiceClient {
	if s == nil {
		return nil
	}
	return s.authClient
}

// AccountClient returns the current account client.
func (s *Server) AccountClient() authv1.AccountServiceClient {
	if s == nil {
		return nil
	}
	return s.accountClient
}

// SessionClient returns the current session client.
func (s *Server) SessionClient() statev1.SessionServiceClient {
	if s == nil {
		return nil
	}
	return s.sessionClient
}

// CharacterClient returns the current character client.
func (s *Server) CharacterClient() statev1.CharacterServiceClient {
	if s == nil {
		return nil
	}
	return s.characterClient
}

// ParticipantClient returns the current participant client.
func (s *Server) ParticipantClient() statev1.ParticipantServiceClient {
	if s == nil {
		return nil
	}
	return s.participantClient
}

// InviteClient returns the current invite client.
func (s *Server) InviteClient() invitev1.InviteServiceClient {
	if s == nil {
		return nil
	}
	return s.inviteClient
}

// SnapshotClient returns the current snapshot client.
func (s *Server) SnapshotClient() statev1.SnapshotServiceClient {
	if s == nil {
		return nil
	}
	return s.snapshotClient
}

// EventClient returns the current event client.
func (s *Server) EventClient() statev1.EventServiceClient {
	if s == nil {
		return nil
	}
	return s.eventClient
}

// StatisticsClient returns the current statistics client.
func (s *Server) StatisticsClient() statev1.StatisticsServiceClient {
	if s == nil {
		return nil
	}
	return s.statisticsClient
}

// SystemClient returns the current system client.
func (s *Server) SystemClient() statev1.SystemServiceClient {
	if s == nil {
		return nil
	}
	return s.systemClient
}

// DaggerheartContentClient returns the Daggerheart content client.
func (s *Server) DaggerheartContentClient() daggerheartv1.DaggerheartContentServiceClient {
	if s == nil {
		return nil
	}
	return s.contentClient
}

// NewServer builds a configured admin server.
func NewServer(ctx context.Context, cfg Config) (*Server, error) {
	httpAddr := strings.TrimSpace(cfg.HTTPAddr)
	if httpAddr == "" {
		return nil, errors.New("http address is required")
	}
	logf := func(format string, args ...any) {
		log.Printf(format, args...)
	}

	srv := &Server{
		httpAddr: httpAddr,
	}

	// Game service — optional, admin starts immediately even if game is down.
	if addr := strings.TrimSpace(cfg.GRPCAddr); addr != "" {
		mc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
			Name:             "game",
			Addr:             addr,
			Mode:             platformgrpc.ModeOptional,
			Logf:             logf,
			StatusReporter:   cfg.StatusReporter,
			StatusCapability: "admin.game.integration",
			DialOpts: append(
				platformgrpc.LenientDialOptions(),
				grpc.WithChainUnaryInterceptor(grpcauthctx.AdminOverrideUnaryClientInterceptor(adminAuthzOverrideReason)),
				grpc.WithChainStreamInterceptor(grpcauthctx.AdminOverrideStreamClientInterceptor(adminAuthzOverrideReason)),
			),
		})
		if err != nil {
			return nil, fmt.Errorf("admin: managed conn game: %w", err)
		}
		srv.gameMc = mc
		conn := mc.Conn()
		srv.daggerheartClient = daggerheartv1.NewDaggerheartServiceClient(conn)
		srv.contentClient = daggerheartv1.NewDaggerheartContentServiceClient(conn)
		srv.campaignClient = statev1.NewCampaignServiceClient(conn)
		srv.sessionClient = statev1.NewSessionServiceClient(conn)
		srv.characterClient = statev1.NewCharacterServiceClient(conn)
		srv.participantClient = statev1.NewParticipantServiceClient(conn)
		srv.snapshotClient = statev1.NewSnapshotServiceClient(conn)
		srv.eventClient = statev1.NewEventServiceClient(conn)
		srv.statisticsClient = statev1.NewStatisticsServiceClient(conn)
		srv.systemClient = statev1.NewSystemServiceClient(conn)
	}

	// Auth service — optional, same as game.
	if addr := strings.TrimSpace(cfg.AuthAddr); addr != "" {
		mc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
			Name:             "auth",
			Addr:             addr,
			Mode:             platformgrpc.ModeOptional,
			Logf:             logf,
			StatusReporter:   cfg.StatusReporter,
			StatusCapability: "admin.auth.integration",
		})
		if err != nil {
			srv.closeConns()
			return nil, fmt.Errorf("admin: managed conn auth: %w", err)
		}
		srv.authMc = mc
		conn := mc.Conn()
		srv.authClient = authv1.NewAuthServiceClient(conn)
		srv.accountClient = authv1.NewAccountServiceClient(conn)
	}

	// Invite service — optional, same as game.
	if addr := strings.TrimSpace(cfg.InviteAddr); addr != "" {
		mc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
			Name:             "invite",
			Addr:             addr,
			Mode:             platformgrpc.ModeOptional,
			Logf:             logf,
			StatusReporter:   cfg.StatusReporter,
			StatusCapability: "admin.invite.integration",
			DialOpts: append(
				platformgrpc.LenientDialOptions(),
				grpc.WithChainUnaryInterceptor(grpcauthctx.AdminOverrideUnaryClientInterceptor(adminAuthzOverrideReason)),
			),
		})
		if err != nil {
			srv.closeConns()
			return nil, fmt.Errorf("admin: managed conn invite: %w", err)
		}
		srv.inviteMc = mc
		srv.inviteClient = invitev1.NewInviteServiceClient(mc.Conn())
	}

	// Status service — optional.
	if addr := strings.TrimSpace(cfg.StatusAddr); addr != "" {
		mc, err := newManagedConn(ctx, platformgrpc.ManagedConnConfig{
			Name: "status",
			Addr: addr,
			Mode: platformgrpc.ModeOptional,
			Logf: logf,
		})
		if err != nil {
			srv.closeConns()
			return nil, fmt.Errorf("admin: managed conn status: %w", err)
		}
		srv.statusMc = mc
		srv.statusClient = statusv1.NewStatusServiceClient(mc.Conn())

		// Late-bind: once the status service is reachable, attach the
		// client to the reporter so accumulated capabilities flush.
		if cfg.StatusReporter != nil {
			client := srv.statusClient
			go func() {
				if mc.WaitReady(ctx) == nil {
					cfg.StatusReporter.SetClient(client)
				}
			}()
		}
	}

	handler := NewHandlerWithConfig(srv, cfg.GRPCAddr, cfg.AuthConfig, srv.statusClient)
	srv.httpServer = &http.Server{
		Addr:              httpAddr,
		Handler:           handler,
		ReadHeaderTimeout: timeouts.ReadHeader,
	}

	return srv, nil
}

// StatusClient returns the status service client used by the server.
func (s *Server) StatusClient() statusv1.StatusServiceClient {
	if s == nil {
		return nil
	}
	return s.statusClient
}

// ListenAndServe runs the HTTP server until the context ends.
func (s *Server) ListenAndServe(ctx context.Context) error {
	if s == nil {
		return errors.New("admin server is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	serveErr := make(chan error, 1)
	log.Printf("admin listening on %s", s.httpAddr)
	go func() {
		serveErr <- s.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeouts.Shutdown)
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

// Close releases any gRPC resources held by the server.
func (s *Server) Close() {
	if s == nil {
		return
	}
	s.closeConns()
	if s.httpServer != nil {
		_ = s.httpServer.Close()
	}
}

func (s *Server) closeConns() {
	closeManagedConn(s.statusMc, "status")
	s.statusMc = nil
	closeManagedConn(s.inviteMc, "invite")
	s.inviteMc = nil
	closeManagedConn(s.authMc, "auth")
	s.authMc = nil
	closeManagedConn(s.gameMc, "game")
	s.gameMc = nil
}

func closeManagedConn(mc *platformgrpc.ManagedConn, name string) {
	if mc == nil {
		return
	}
	if err := mc.Close(); err != nil {
		log.Printf("close admin %s managed conn: %v", name, err)
	}
}
