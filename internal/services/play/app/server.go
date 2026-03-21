package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
	playui "github.com/louisbranch/fracturing.space/internal/services/play/ui"
	"github.com/louisbranch/fracturing.space/internal/services/shared/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	gogrpc "google.golang.org/grpc"
)

// Config defines startup inputs for the play service.
type Config struct {
	HTTPAddr            string
	WebHTTPAddr         string
	PlayUIDevServerURL  string
	RequestSchemePolicy requestmeta.SchemePolicy
	LaunchGrant         playlaunchgrant.Config
	Logger              *slog.Logger
}

// Dependencies defines the runtime collaborators required by the play service.
type Dependencies struct {
	Auth         authClient
	Interaction  interactionClient
	Campaign     campaignClient
	System       systemClient
	Participants participantClient
	Characters   characterClient
	Events       eventClient
	Transcripts  transcript.Store
}

type authClient interface {
	CreateWebSession(context.Context, *authv1.CreateWebSessionRequest, ...gogrpc.CallOption) (*authv1.CreateWebSessionResponse, error)
	GetWebSession(context.Context, *authv1.GetWebSessionRequest, ...gogrpc.CallOption) (*authv1.GetWebSessionResponse, error)
}

type interactionClient interface {
	GetInteractionState(context.Context, *gamev1.GetInteractionStateRequest, ...gogrpc.CallOption) (*gamev1.GetInteractionStateResponse, error)
	SetActiveScene(context.Context, *gamev1.SetActiveSceneRequest, ...gogrpc.CallOption) (*gamev1.SetActiveSceneResponse, error)
	StartScenePlayerPhase(context.Context, *gamev1.StartScenePlayerPhaseRequest, ...gogrpc.CallOption) (*gamev1.StartScenePlayerPhaseResponse, error)
	SubmitScenePlayerPost(context.Context, *gamev1.SubmitScenePlayerPostRequest, ...gogrpc.CallOption) (*gamev1.SubmitScenePlayerPostResponse, error)
	YieldScenePlayerPhase(context.Context, *gamev1.YieldScenePlayerPhaseRequest, ...gogrpc.CallOption) (*gamev1.YieldScenePlayerPhaseResponse, error)
	UnyieldScenePlayerPhase(context.Context, *gamev1.UnyieldScenePlayerPhaseRequest, ...gogrpc.CallOption) (*gamev1.UnyieldScenePlayerPhaseResponse, error)
	EndScenePlayerPhase(context.Context, *gamev1.EndScenePlayerPhaseRequest, ...gogrpc.CallOption) (*gamev1.EndScenePlayerPhaseResponse, error)
	CommitSceneGMOutput(context.Context, *gamev1.CommitSceneGMOutputRequest, ...gogrpc.CallOption) (*gamev1.CommitSceneGMOutputResponse, error)
	AcceptScenePlayerPhase(context.Context, *gamev1.AcceptScenePlayerPhaseRequest, ...gogrpc.CallOption) (*gamev1.AcceptScenePlayerPhaseResponse, error)
	RequestScenePlayerRevisions(context.Context, *gamev1.RequestScenePlayerRevisionsRequest, ...gogrpc.CallOption) (*gamev1.RequestScenePlayerRevisionsResponse, error)
	PauseSessionForOOC(context.Context, *gamev1.PauseSessionForOOCRequest, ...gogrpc.CallOption) (*gamev1.PauseSessionForOOCResponse, error)
	PostSessionOOC(context.Context, *gamev1.PostSessionOOCRequest, ...gogrpc.CallOption) (*gamev1.PostSessionOOCResponse, error)
	MarkOOCReadyToResume(context.Context, *gamev1.MarkOOCReadyToResumeRequest, ...gogrpc.CallOption) (*gamev1.MarkOOCReadyToResumeResponse, error)
	ClearOOCReadyToResume(context.Context, *gamev1.ClearOOCReadyToResumeRequest, ...gogrpc.CallOption) (*gamev1.ClearOOCReadyToResumeResponse, error)
	ResumeFromOOC(context.Context, *gamev1.ResumeFromOOCRequest, ...gogrpc.CallOption) (*gamev1.ResumeFromOOCResponse, error)
	SetSessionGMAuthority(context.Context, *gamev1.SetSessionGMAuthorityRequest, ...gogrpc.CallOption) (*gamev1.SetSessionGMAuthorityResponse, error)
	RetryAIGMTurn(context.Context, *gamev1.RetryAIGMTurnRequest, ...gogrpc.CallOption) (*gamev1.RetryAIGMTurnResponse, error)
}

type campaignClient interface {
	GetCampaign(context.Context, *gamev1.GetCampaignRequest, ...gogrpc.CallOption) (*gamev1.GetCampaignResponse, error)
}

type systemClient interface {
	GetGameSystem(context.Context, *gamev1.GetGameSystemRequest, ...gogrpc.CallOption) (*gamev1.GetGameSystemResponse, error)
}

type participantClient interface {
	ListParticipants(context.Context, *gamev1.ListParticipantsRequest, ...gogrpc.CallOption) (*gamev1.ListParticipantsResponse, error)
}

type characterClient interface {
	ListCharacters(context.Context, *gamev1.ListCharactersRequest, ...gogrpc.CallOption) (*gamev1.ListCharactersResponse, error)
	GetCharacterSheet(context.Context, *gamev1.GetCharacterSheetRequest, ...gogrpc.CallOption) (*gamev1.GetCharacterSheetResponse, error)
}

type eventClient interface {
	SubscribeCampaignUpdates(context.Context, *gamev1.SubscribeCampaignUpdatesRequest, ...gogrpc.CallOption) (gogrpc.ServerStreamingClient[gamev1.CampaignUpdate], error)
}

// Server hosts the play HTTP surface and lifecycle.
type Server struct {
	httpAddr            string
	httpServer          *http.Server
	logger              *slog.Logger
	webFallbackPort     string
	requestSchemePolicy requestmeta.SchemePolicy
	auth                authClient
	interaction         interactionClient
	campaign            campaignClient
	system              systemClient
	participants        participantClient
	characters          characterClient
	events              eventClient
	transcripts         transcript.Store
	shellAssets         shellAssets
	realtime            *realtimeHub
}

// NewServer constructs a play service runtime from injected dependencies.
func NewServer(cfg Config, deps Dependencies) (*Server, error) {
	httpAddr := strings.TrimSpace(cfg.HTTPAddr)
	if httpAddr == "" {
		return nil, errors.New("http address is required")
	}
	if err := playlaunchgrant.ValidateConfig(cfg.LaunchGrant); err != nil {
		return nil, fmt.Errorf("play launch grant config: %w", err)
	}
	if err := deps.validate(); err != nil {
		return nil, err
	}
	shellAssets, err := loadShellAssets(cfg.PlayUIDevServerURL)
	if err != nil {
		return nil, fmt.Errorf("load play ui assets: %w", err)
	}

	server := &Server{
		httpAddr:            httpAddr,
		logger:              loggerOrDefault(cfg.Logger),
		webFallbackPort:     websupport.ResolveHTTPFallbackPort(cfg.WebHTTPAddr),
		requestSchemePolicy: cfg.RequestSchemePolicy,
		auth:                deps.Auth,
		interaction:         deps.Interaction,
		campaign:            deps.Campaign,
		system:              deps.System,
		participants:        deps.Participants,
		characters:          deps.Characters,
		events:              deps.Events,
		transcripts:         deps.Transcripts,
		shellAssets:         shellAssets,
	}
	server.realtime = newRealtimeHub(server)

	handler, err := server.newHandler(cfg.LaunchGrant)
	if err != nil {
		server.Close()
		return nil, err
	}
	server.httpServer = &http.Server{
		Addr:              httpAddr,
		Handler:           handler,
		ReadHeaderTimeout: timeouts.ReadHeader,
	}
	return server, nil
}

func (d Dependencies) validate() error {
	if d.Auth == nil {
		return errors.New("auth dependency is required")
	}
	if d.Interaction == nil {
		return errors.New("interaction dependency is required")
	}
	if d.Campaign == nil {
		return errors.New("campaign dependency is required")
	}
	if d.System == nil {
		return errors.New("system dependency is required")
	}
	if d.Participants == nil {
		return errors.New("participant dependency is required")
	}
	if d.Characters == nil {
		return errors.New("character dependency is required")
	}
	if d.Events == nil {
		return errors.New("event dependency is required")
	}
	if d.Transcripts == nil {
		return errors.New("transcript store is required")
	}
	return nil
}

func (s *Server) newHandler(launchGrantCfg playlaunchgrant.Config) (http.Handler, error) {
	rootMux := http.NewServeMux()
	if strings.TrimSpace(s.shellAssets.devServerURL) == "" {
		distFS, err := fs.Sub(playui.DistFS, "dist")
		if err != nil {
			return nil, fmt.Errorf("play ui dist filesystem: %w", err)
		}
		rootMux.Handle("/assets/play/", http.StripPrefix("/assets/play/", http.FileServer(http.FS(distFS))))
	}
	s.registerRoutes(rootMux, launchGrantCfg)
	return httpx.Chain(rootMux,
		httpx.RecoverPanic(),
		httpx.RequestID("play"),
	), nil
}

// ListenAndServe serves HTTP traffic until context cancellation or server stop.
func (s *Server) ListenAndServe(ctx context.Context) error {
	if s == nil {
		return errors.New("play server is nil")
	}
	if ctx == nil {
		return errors.New("context is required")
	}
	serveErr := make(chan error, 1)
	s.logger.Info("play server listening", "addr", s.httpAddr)
	go func() {
		serveErr <- s.httpServer.ListenAndServe()
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), timeouts.Shutdown)
		err := s.httpServer.Shutdown(shutdownCtx)
		cancel()
		if err != nil {
			return fmt.Errorf("shutdown play http server: %w", err)
		}
		return nil
	case err := <-serveErr:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return fmt.Errorf("serve play http: %w", err)
	}
}

// Close closes open server resources.
func (s *Server) Close() {
	if s == nil {
		return
	}
	if s.realtime != nil {
		s.realtime.Close()
	}
	if s.httpServer != nil {
		_ = s.httpServer.Close()
	}
}
