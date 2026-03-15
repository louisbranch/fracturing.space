package app

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"net/url"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/platform/serviceaddr"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	playsqlite "github.com/louisbranch/fracturing.space/internal/services/play/storage/sqlite"
	"github.com/louisbranch/fracturing.space/internal/services/play/transcript"
	playui "github.com/louisbranch/fracturing.space/internal/services/play/ui"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"github.com/louisbranch/fracturing.space/internal/services/shared/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playlaunchgrant"
	"github.com/louisbranch/fracturing.space/internal/services/shared/playorigin"
	websupport "github.com/louisbranch/fracturing.space/internal/services/shared/websupport"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
	gogrpc "google.golang.org/grpc"
	gogrpccodes "google.golang.org/grpc/codes"
	gogrpcstatus "google.golang.org/grpc/status"
)

// Config defines startup inputs for the play service.
type Config struct {
	HTTPAddr            string
	WebHTTPAddr         string
	AuthAddr            string
	GameAddr            string
	DBPath              string
	PlayUIDevServerURL  string
	RequestSchemePolicy requestmeta.SchemePolicy
	LaunchGrant         playlaunchgrant.Config
	Logger              *slog.Logger
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

type eventClient interface {
	SubscribeCampaignUpdates(context.Context, *gamev1.SubscribeCampaignUpdatesRequest, ...gogrpc.CallOption) (gogrpc.ServerStreamingClient[gamev1.CampaignUpdate], error)
}

type transcriptStore interface {
	LatestSequence(context.Context, string, string) (int64, error)
	AppendMessage(context.Context, string, string, transcript.MessageActor, string, string) (transcript.Message, bool, error)
	HistoryAfter(context.Context, string, string, int64) ([]transcript.Message, error)
	HistoryBefore(context.Context, string, string, int64, int) ([]transcript.Message, error)
	Close() error
}

// Server hosts the play HTTP surface and lifecycle.
type Server struct {
	httpAddr            string
	httpServer          *http.Server
	logger              *slog.Logger
	webFallbackPort     string
	requestSchemePolicy requestmeta.SchemePolicy
	authMc              *platformgrpc.ManagedConn
	gameMc              *platformgrpc.ManagedConn
	auth                authClient
	interaction         interactionClient
	campaign            campaignClient
	system              systemClient
	events              eventClient
	transcripts         transcriptStore
	shellAssets         shellAssets
	realtime            *realtimeHub
}

// NewServer constructs a play service runtime with required auth/game dependencies.
func NewServer(ctx context.Context, cfg Config) (*Server, error) {
	if ctx == nil {
		return nil, errors.New("context is required")
	}
	httpAddr := strings.TrimSpace(cfg.HTTPAddr)
	if httpAddr == "" {
		return nil, errors.New("http address is required")
	}
	if strings.TrimSpace(cfg.DBPath) == "" {
		return nil, errors.New("play db path is required")
	}
	if err := playlaunchgrant.ValidateConfig(cfg.LaunchGrant); err != nil {
		return nil, fmt.Errorf("play launch grant config: %w", err)
	}
	shellAssets, err := loadShellAssets(cfg.PlayUIDevServerURL)
	if err != nil {
		return nil, fmt.Errorf("load play ui assets: %w", err)
	}

	authMc, err := platformgrpc.NewManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "auth",
		Addr: strings.TrimSpace(cfg.AuthAddr),
		Mode: platformgrpc.ModeRequired,
	})
	if err != nil {
		return nil, fmt.Errorf("connect auth: %w", err)
	}
	gameMc, err := platformgrpc.NewManagedConn(ctx, platformgrpc.ManagedConnConfig{
		Name: "game",
		Addr: strings.TrimSpace(cfg.GameAddr),
		Mode: platformgrpc.ModeRequired,
		DialOpts: append(
			platformgrpc.LenientDialOptions(),
			gogrpc.WithChainUnaryInterceptor(grpcauthctx.ServiceIDUnaryClientInterceptor(serviceaddr.ServicePlay)),
			gogrpc.WithChainStreamInterceptor(grpcauthctx.ServiceIDStreamClientInterceptor(serviceaddr.ServicePlay)),
		),
	})
	if err != nil {
		_ = authMc.Close()
		return nil, fmt.Errorf("connect game: %w", err)
	}
	store, err := playsqlite.Open(cfg.DBPath)
	if err != nil {
		_ = gameMc.Close()
		_ = authMc.Close()
		return nil, fmt.Errorf("open play transcript store: %w", err)
	}

	server := &Server{
		httpAddr:            httpAddr,
		logger:              loggerOrDefault(cfg.Logger),
		webFallbackPort:     websupport.ResolveHTTPFallbackPort(cfg.WebHTTPAddr),
		requestSchemePolicy: cfg.RequestSchemePolicy,
		authMc:              authMc,
		gameMc:              gameMc,
		auth:                authv1.NewAuthServiceClient(authMc.Conn()),
		interaction:         gamev1.NewInteractionServiceClient(gameMc.Conn()),
		campaign:            gamev1.NewCampaignServiceClient(gameMc.Conn()),
		system:              gamev1.NewSystemServiceClient(gameMc.Conn()),
		events:              gamev1.NewEventServiceClient(gameMc.Conn()),
		transcripts:         store,
		shellAssets:         shellAssets,
	}
	server.realtime = newRealtimeHub(server)

	handler, err := server.newHandler(cfg)
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

func (s *Server) newHandler(cfg Config) (http.Handler, error) {
	rootMux := http.NewServeMux()
	if strings.TrimSpace(s.shellAssets.devServerURL) == "" {
		distFS, err := fs.Sub(playui.DistFS, "dist")
		if err != nil {
			return nil, fmt.Errorf("play ui dist filesystem: %w", err)
		}
		rootMux.Handle("/assets/play/", http.StripPrefix("/assets/play/", http.FileServer(http.FS(distFS))))
	}
	rootMux.HandleFunc("GET /up", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	rootMux.HandleFunc("GET /campaigns/{campaignID}", func(w http.ResponseWriter, r *http.Request) {
		s.handleCampaignShell(w, r, cfg.LaunchGrant)
	})
	rootMux.HandleFunc("GET /api/campaigns/{campaignID}/bootstrap", s.handleBootstrap)
	rootMux.HandleFunc("GET /api/campaigns/{campaignID}/chat/history", s.handleChatHistory)
	rootMux.HandleFunc("POST /api/campaigns/{campaignID}/interaction/set-active-scene", s.handleSetActiveScene)
	rootMux.HandleFunc("POST /api/campaigns/{campaignID}/interaction/start-scene-player-phase", s.handleStartScenePlayerPhase)
	rootMux.HandleFunc("POST /api/campaigns/{campaignID}/interaction/submit-scene-player-post", s.handleSubmitScenePlayerPost)
	rootMux.HandleFunc("POST /api/campaigns/{campaignID}/interaction/yield-scene-player-phase", s.handleYieldScenePlayerPhase)
	rootMux.HandleFunc("POST /api/campaigns/{campaignID}/interaction/unyield-scene-player-phase", s.handleUnyieldScenePlayerPhase)
	rootMux.HandleFunc("POST /api/campaigns/{campaignID}/interaction/end-scene-player-phase", s.handleEndScenePlayerPhase)
	rootMux.HandleFunc("POST /api/campaigns/{campaignID}/interaction/commit-scene-gm-output", s.handleCommitSceneGMOutput)
	rootMux.HandleFunc("POST /api/campaigns/{campaignID}/interaction/accept-scene-player-phase", s.handleAcceptScenePlayerPhase)
	rootMux.HandleFunc("POST /api/campaigns/{campaignID}/interaction/request-scene-player-revisions", s.handleRequestScenePlayerRevisions)
	rootMux.HandleFunc("POST /api/campaigns/{campaignID}/interaction/pause-session-for-ooc", s.handlePauseSessionForOOC)
	rootMux.HandleFunc("POST /api/campaigns/{campaignID}/interaction/post-session-ooc", s.handlePostSessionOOC)
	rootMux.HandleFunc("POST /api/campaigns/{campaignID}/interaction/mark-ooc-ready-to-resume", s.handleMarkOOCReadyToResume)
	rootMux.HandleFunc("POST /api/campaigns/{campaignID}/interaction/clear-ooc-ready-to-resume", s.handleClearOOCReadyToResume)
	rootMux.HandleFunc("POST /api/campaigns/{campaignID}/interaction/resume-from-ooc", s.handleResumeFromOOC)
	rootMux.HandleFunc("POST /api/campaigns/{campaignID}/interaction/set-session-gm-authority", s.handleSetSessionGMAuthority)
	rootMux.HandleFunc("POST /api/campaigns/{campaignID}/interaction/retry-ai-gm-turn", s.handleRetryAIGMTurn)
	rootMux.Handle("GET /realtime", s.realtime.handler())
	return httpx.Chain(rootMux,
		httpx.RecoverPanic(),
		httpx.RequestID("play"),
	), nil
}

func (s *Server) handleCampaignShell(w http.ResponseWriter, r *http.Request, launchGrantCfg playlaunchgrant.Config) {
	campaignID := strings.TrimSpace(r.PathValue("campaignID"))
	if campaignID == "" {
		http.NotFound(w, r)
		return
	}
	userID, handled := s.resolveOrCreatePlaySession(w, r, campaignID, launchGrantCfg)
	if handled {
		return
	}
	if strings.TrimSpace(userID) == "" {
		http.Redirect(w, r, playorigin.WebURL(r, s.requestSchemePolicy, s.webFallbackPort, routepath.AppCampaignGame(campaignID)), http.StatusSeeOther)
		return
	}
	if _, err := s.buildBootstrap(r.Context(), campaignID, userID); err != nil {
		writeRPCError(w, err)
		return
	}
	html, err := s.shellAssets.renderHTML(shellRenderInput{
		CampaignID:    campaignID,
		BootstrapPath: pathForCampaignAPI(campaignID, "bootstrap"),
		RealtimePath:  "/realtime",
		BackURL:       routepath.AppCampaignGame(campaignID),
	})
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to render play shell")
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(html)
}

func (s *Server) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	campaignID, userID, ok := s.requireAPIUser(w, r)
	if !ok {
		return
	}
	bootstrap, err := s.buildBootstrap(r.Context(), campaignID, userID)
	if err != nil {
		writeRPCError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, bootstrap)
}

func (s *Server) handleChatHistory(w http.ResponseWriter, r *http.Request) {
	campaignID, userID, ok := s.requireAPIUser(w, r)
	if !ok {
		return
	}
	before := int64(1 << 62)
	if raw := strings.TrimSpace(r.URL.Query().Get("before_seq")); raw != "" {
		var err error
		before, err = parseInt64(raw)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid before_seq")
			return
		}
	}
	limit := defaultChatHistoryLimit
	if raw := strings.TrimSpace(r.URL.Query().Get("limit")); raw != "" {
		value, err := parseInt(raw)
		if err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid limit")
			return
		}
		limit = value
	}
	state, err := s.loadInteractionState(r.Context(), campaignID, userID)
	if err != nil {
		writeRPCError(w, err)
		return
	}
	sessionID := strings.TrimSpace(state.GetActiveSession().GetSessionId())
	if sessionID == "" {
		writeJSON(w, http.StatusOK, playHistoryResponse{SessionID: "", Messages: []playChatMessage{}})
		return
	}
	messages, err := s.transcripts.HistoryBefore(r.Context(), campaignID, sessionID, before, limit)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to load chat history")
		return
	}
	writeJSON(w, http.StatusOK, playHistoryResponse{
		SessionID: sessionID,
		Messages:  transcriptMessagesToPayload(messages),
	})
}

func (s *Server) handleSetActiveScene(w http.ResponseWriter, r *http.Request) {
	var req gamev1.SetActiveSceneRequest
	s.handleInteractionMutation(w, r, &req, func(ctx context.Context, campaignID string) (*gamev1.InteractionState, error) {
		req.CampaignId = campaignID
		resp, err := s.interaction.SetActiveScene(ctx, &req)
		if err != nil {
			return nil, err
		}
		return resp.GetState(), nil
	})
}

func (s *Server) handleStartScenePlayerPhase(w http.ResponseWriter, r *http.Request) {
	var req gamev1.StartScenePlayerPhaseRequest
	s.handleInteractionMutation(w, r, &req, func(ctx context.Context, campaignID string) (*gamev1.InteractionState, error) {
		req.CampaignId = campaignID
		resp, err := s.interaction.StartScenePlayerPhase(ctx, &req)
		if err != nil {
			return nil, err
		}
		return resp.GetState(), nil
	})
}

func (s *Server) handleSubmitScenePlayerPost(w http.ResponseWriter, r *http.Request) {
	var req gamev1.SubmitScenePlayerPostRequest
	s.handleInteractionMutation(w, r, &req, func(ctx context.Context, campaignID string) (*gamev1.InteractionState, error) {
		req.CampaignId = campaignID
		resp, err := s.interaction.SubmitScenePlayerPost(ctx, &req)
		if err != nil {
			return nil, err
		}
		return resp.GetState(), nil
	})
}

func (s *Server) handleYieldScenePlayerPhase(w http.ResponseWriter, r *http.Request) {
	var req gamev1.YieldScenePlayerPhaseRequest
	s.handleInteractionMutation(w, r, &req, func(ctx context.Context, campaignID string) (*gamev1.InteractionState, error) {
		req.CampaignId = campaignID
		resp, err := s.interaction.YieldScenePlayerPhase(ctx, &req)
		if err != nil {
			return nil, err
		}
		return resp.GetState(), nil
	})
}

func (s *Server) handleUnyieldScenePlayerPhase(w http.ResponseWriter, r *http.Request) {
	var req gamev1.UnyieldScenePlayerPhaseRequest
	s.handleInteractionMutation(w, r, &req, func(ctx context.Context, campaignID string) (*gamev1.InteractionState, error) {
		req.CampaignId = campaignID
		resp, err := s.interaction.UnyieldScenePlayerPhase(ctx, &req)
		if err != nil {
			return nil, err
		}
		return resp.GetState(), nil
	})
}

func (s *Server) handleEndScenePlayerPhase(w http.ResponseWriter, r *http.Request) {
	var req gamev1.EndScenePlayerPhaseRequest
	s.handleInteractionMutation(w, r, &req, func(ctx context.Context, campaignID string) (*gamev1.InteractionState, error) {
		req.CampaignId = campaignID
		resp, err := s.interaction.EndScenePlayerPhase(ctx, &req)
		if err != nil {
			return nil, err
		}
		return resp.GetState(), nil
	})
}

func (s *Server) handleCommitSceneGMOutput(w http.ResponseWriter, r *http.Request) {
	var req gamev1.CommitSceneGMOutputRequest
	s.handleInteractionMutation(w, r, &req, func(ctx context.Context, campaignID string) (*gamev1.InteractionState, error) {
		req.CampaignId = campaignID
		resp, err := s.interaction.CommitSceneGMOutput(ctx, &req)
		if err != nil {
			return nil, err
		}
		return resp.GetState(), nil
	})
}

func (s *Server) handleAcceptScenePlayerPhase(w http.ResponseWriter, r *http.Request) {
	var req gamev1.AcceptScenePlayerPhaseRequest
	s.handleInteractionMutation(w, r, &req, func(ctx context.Context, campaignID string) (*gamev1.InteractionState, error) {
		req.CampaignId = campaignID
		resp, err := s.interaction.AcceptScenePlayerPhase(ctx, &req)
		if err != nil {
			return nil, err
		}
		return resp.GetState(), nil
	})
}

func (s *Server) handleRequestScenePlayerRevisions(w http.ResponseWriter, r *http.Request) {
	var req gamev1.RequestScenePlayerRevisionsRequest
	s.handleInteractionMutation(w, r, &req, func(ctx context.Context, campaignID string) (*gamev1.InteractionState, error) {
		req.CampaignId = campaignID
		resp, err := s.interaction.RequestScenePlayerRevisions(ctx, &req)
		if err != nil {
			return nil, err
		}
		return resp.GetState(), nil
	})
}

func (s *Server) handlePauseSessionForOOC(w http.ResponseWriter, r *http.Request) {
	var req gamev1.PauseSessionForOOCRequest
	s.handleInteractionMutation(w, r, &req, func(ctx context.Context, campaignID string) (*gamev1.InteractionState, error) {
		req.CampaignId = campaignID
		resp, err := s.interaction.PauseSessionForOOC(ctx, &req)
		if err != nil {
			return nil, err
		}
		return resp.GetState(), nil
	})
}

func (s *Server) handlePostSessionOOC(w http.ResponseWriter, r *http.Request) {
	var req gamev1.PostSessionOOCRequest
	s.handleInteractionMutation(w, r, &req, func(ctx context.Context, campaignID string) (*gamev1.InteractionState, error) {
		req.CampaignId = campaignID
		resp, err := s.interaction.PostSessionOOC(ctx, &req)
		if err != nil {
			return nil, err
		}
		return resp.GetState(), nil
	})
}

func (s *Server) handleMarkOOCReadyToResume(w http.ResponseWriter, r *http.Request) {
	s.handleInteractionMutation(w, r, nil, func(ctx context.Context, campaignID string) (*gamev1.InteractionState, error) {
		resp, err := s.interaction.MarkOOCReadyToResume(ctx, &gamev1.MarkOOCReadyToResumeRequest{CampaignId: campaignID})
		if err != nil {
			return nil, err
		}
		return resp.GetState(), nil
	})
}

func (s *Server) handleClearOOCReadyToResume(w http.ResponseWriter, r *http.Request) {
	s.handleInteractionMutation(w, r, nil, func(ctx context.Context, campaignID string) (*gamev1.InteractionState, error) {
		resp, err := s.interaction.ClearOOCReadyToResume(ctx, &gamev1.ClearOOCReadyToResumeRequest{CampaignId: campaignID})
		if err != nil {
			return nil, err
		}
		return resp.GetState(), nil
	})
}

func (s *Server) handleResumeFromOOC(w http.ResponseWriter, r *http.Request) {
	s.handleInteractionMutation(w, r, nil, func(ctx context.Context, campaignID string) (*gamev1.InteractionState, error) {
		resp, err := s.interaction.ResumeFromOOC(ctx, &gamev1.ResumeFromOOCRequest{CampaignId: campaignID})
		if err != nil {
			return nil, err
		}
		return resp.GetState(), nil
	})
}

func (s *Server) handleSetSessionGMAuthority(w http.ResponseWriter, r *http.Request) {
	var req gamev1.SetSessionGMAuthorityRequest
	s.handleInteractionMutation(w, r, &req, func(ctx context.Context, campaignID string) (*gamev1.InteractionState, error) {
		req.CampaignId = campaignID
		resp, err := s.interaction.SetSessionGMAuthority(ctx, &req)
		if err != nil {
			return nil, err
		}
		return resp.GetState(), nil
	})
}

func (s *Server) handleRetryAIGMTurn(w http.ResponseWriter, r *http.Request) {
	var req gamev1.RetryAIGMTurnRequest
	s.handleInteractionMutation(w, r, &req, func(ctx context.Context, campaignID string) (*gamev1.InteractionState, error) {
		req.CampaignId = campaignID
		resp, err := s.interaction.RetryAIGMTurn(ctx, &req)
		if err != nil {
			return nil, err
		}
		return resp.GetState(), nil
	})
}

func (s *Server) handleInteractionMutation(
	w http.ResponseWriter,
	r *http.Request,
	target any,
	call func(context.Context, string) (*gamev1.InteractionState, error),
) {
	campaignID, userID, ok := s.requireAPIUser(w, r)
	if !ok {
		return
	}
	if target != nil {
		if err := decodeStrictJSON(r, target); err != nil {
			writeJSONError(w, http.StatusBadRequest, "invalid json body")
			return
		}
	}
	state, err := call(grpcauthctx.WithUserID(r.Context(), userID), campaignID)
	if err != nil {
		writeRPCError(w, err)
		return
	}
	response, err := s.buildInteractionResponse(r.Context(), state)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, "failed to refresh play interaction state")
		return
	}
	writeJSON(w, http.StatusOK, response)
	s.realtime.broadcastCurrent(campaignID)
}

func (s *Server) requireAPIUser(w http.ResponseWriter, r *http.Request) (string, string, bool) {
	campaignID := strings.TrimSpace(r.PathValue("campaignID"))
	if campaignID == "" {
		http.NotFound(w, r)
		return "", "", false
	}
	userID, err := s.resolvePlayUserID(r.Context(), r)
	if err != nil {
		writeJSONError(w, http.StatusUnauthorized, "authentication required")
		return "", "", false
	}
	return campaignID, userID, true
}

func (s *Server) buildBootstrap(ctx context.Context, campaignID string, userID string) (playBootstrap, error) {
	state, err := s.loadInteractionState(ctx, campaignID, userID)
	if err != nil {
		return playBootstrap{}, err
	}
	system, err := s.loadSystemMetadata(ctx, campaignID, userID)
	if err != nil {
		return playBootstrap{}, err
	}
	chat, err := s.buildRecentChatSnapshot(ctx, campaignID, state)
	if err != nil {
		return playBootstrap{}, err
	}
	return playBootstrap{
		CampaignID:       strings.TrimSpace(campaignID),
		Viewer:           state.GetViewer(),
		System:           system,
		InteractionState: state,
		Chat:             chat,
		Realtime: playRealtimeConfig{
			URL:             "/realtime",
			ProtocolVersion: playRealtimeProtocolVersion,
		},
	}, nil
}

func (s *Server) buildRoomSnapshot(ctx context.Context, campaignID string, state *gamev1.InteractionState, latestGameSeq uint64) (playRoomSnapshot, error) {
	chat, err := s.buildChatCursor(ctx, campaignID, state)
	if err != nil {
		return playRoomSnapshot{}, err
	}
	return playRoomSnapshot{
		InteractionState: state,
		Chat:             chat,
		LatestGameSeq:    latestGameSeq,
	}, nil
}

func (s *Server) buildInteractionResponse(ctx context.Context, state *gamev1.InteractionState) (playRoomSnapshot, error) {
	return s.buildRoomSnapshot(ctx, strings.TrimSpace(state.GetCampaignId()), state, 0)
}

func (s *Server) buildRecentChatSnapshot(ctx context.Context, campaignID string, state *gamev1.InteractionState) (playChatSnapshot, error) {
	historyURL := pathForCampaignAPI(campaignID, "chat/history")
	sessionID := strings.TrimSpace(state.GetActiveSession().GetSessionId())
	if sessionID == "" {
		return playChatSnapshot{SessionID: "", LatestSequenceID: 0, Messages: []playChatMessage{}, HistoryURL: historyURL}, nil
	}
	latest, err := s.transcripts.LatestSequence(ctx, campaignID, sessionID)
	if err != nil {
		return playChatSnapshot{}, fmt.Errorf("load latest transcript sequence: %w", err)
	}
	messages, err := s.transcripts.HistoryBefore(ctx, campaignID, sessionID, latest+1, defaultChatHistoryLimit)
	if err != nil {
		return playChatSnapshot{}, fmt.Errorf("load recent transcript history: %w", err)
	}
	return playChatSnapshot{
		SessionID:        sessionID,
		LatestSequenceID: latest,
		Messages:         transcriptMessagesToPayload(messages),
		HistoryURL:       historyURL,
	}, nil
}

func (s *Server) buildChatCursor(ctx context.Context, campaignID string, state *gamev1.InteractionState) (playChatSnapshot, error) {
	historyURL := pathForCampaignAPI(campaignID, "chat/history")
	sessionID := strings.TrimSpace(state.GetActiveSession().GetSessionId())
	if sessionID == "" {
		return playChatSnapshot{SessionID: "", LatestSequenceID: 0, Messages: []playChatMessage{}, HistoryURL: historyURL}, nil
	}
	latest, err := s.transcripts.LatestSequence(ctx, campaignID, sessionID)
	if err != nil {
		return playChatSnapshot{}, fmt.Errorf("load latest transcript sequence: %w", err)
	}
	return playChatSnapshot{
		SessionID:        sessionID,
		LatestSequenceID: latest,
		Messages:         []playChatMessage{},
		HistoryURL:       historyURL,
	}, nil
}

func (s *Server) buildIncrementalChatMessages(ctx context.Context, campaignID string, sessionID string, afterSeq int64) ([]playChatMessage, error) {
	messages, err := s.transcripts.HistoryAfter(ctx, campaignID, sessionID, afterSeq)
	if err != nil {
		return nil, fmt.Errorf("load transcript messages after sequence: %w", err)
	}
	return transcriptMessagesToPayload(messages), nil
}

func (s *Server) loadInteractionState(ctx context.Context, campaignID string, userID string) (*gamev1.InteractionState, error) {
	resp, err := s.interaction.GetInteractionState(grpcauthctx.WithUserID(ctx, userID), &gamev1.GetInteractionStateRequest{CampaignId: campaignID})
	if err != nil {
		return nil, err
	}
	if resp == nil || resp.GetState() == nil {
		return &gamev1.InteractionState{CampaignId: campaignID}, nil
	}
	return resp.GetState(), nil
}

func (s *Server) loadSystemMetadata(ctx context.Context, campaignID string, userID string) (playSystem, error) {
	resp, err := s.campaign.GetCampaign(grpcauthctx.WithUserID(ctx, userID), &gamev1.GetCampaignRequest{CampaignId: campaignID})
	if err != nil {
		return playSystem{}, err
	}
	campaign := resp.GetCampaign()
	if campaign == nil || campaign.GetSystem() == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		return playSystem{}, nil
	}
	system := playSystem{ID: gameSystemIDString(campaign.GetSystem())}
	infoResp, err := s.system.GetGameSystem(grpcauthctx.WithUserID(ctx, userID), &gamev1.GetGameSystemRequest{Id: campaign.GetSystem()})
	if err != nil {
		return playSystem{}, err
	}
	if info := infoResp.GetSystem(); info != nil {
		system.Name = strings.TrimSpace(info.GetName())
		system.Version = strings.TrimSpace(info.GetVersion())
	}
	if system.Name == "" {
		system.Name = system.ID
	}
	return system, nil
}

func gameSystemIDString(value commonv1.GameSystem) string {
	name := strings.TrimSpace(value.String())
	if name == "" {
		return ""
	}
	name = strings.TrimPrefix(name, "GAME_SYSTEM_")
	return strings.ToLower(name)
}

func (s *Server) resolvePlayUserID(ctx context.Context, r *http.Request) (string, error) {
	sessionID, ok := readPlaySessionCookie(r)
	if !ok {
		return "", errors.New("play session cookie is required")
	}
	return s.resolvePlayUserIDFromSessionID(ctx, sessionID)
}

func (s *Server) resolvePlayUserIDFromSessionID(ctx context.Context, sessionID string) (string, error) {
	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return "", errors.New("play session cookie is required")
	}
	resp, err := s.auth.GetWebSession(ctx, &authv1.GetWebSessionRequest{SessionId: sessionID})
	if err != nil {
		return "", fmt.Errorf("lookup play session: %w", err)
	}
	if resp == nil || resp.GetSession() == nil {
		return "", errors.New("play session not found")
	}
	userID := strings.TrimSpace(resp.GetSession().GetUserId())
	if userID == "" {
		return "", errors.New("play session returned empty user id")
	}
	return userID, nil
}

func (s *Server) resolveOrCreatePlaySession(
	w http.ResponseWriter,
	r *http.Request,
	campaignID string,
	launchGrantCfg playlaunchgrant.Config,
) (string, bool) {
	grant := strings.TrimSpace(r.URL.Query().Get("launch"))
	if grant != "" {
		claims, err := playlaunchgrant.Validate(launchGrantCfg, grant)
		if err != nil || strings.TrimSpace(claims.CampaignID) != campaignID {
			clearPlaySessionCookie(w, r, s.requestSchemePolicy)
			http.Redirect(w, r, playorigin.WebURL(r, s.requestSchemePolicy, s.webFallbackPort, routepath.AppCampaignGame(campaignID)), http.StatusSeeOther)
			return "", true
		}
		resp, err := s.auth.CreateWebSession(r.Context(), &authv1.CreateWebSessionRequest{UserId: claims.UserID})
		if err != nil || resp == nil || resp.GetSession() == nil {
			writeJSONError(w, http.StatusBadGateway, "failed to create play session")
			return "", true
		}
		writePlaySessionCookie(w, r, resp.GetSession().GetId(), s.requestSchemePolicy)
		http.Redirect(w, r, stripLaunchGrant(r), http.StatusSeeOther)
		return "", true
	}

	if sessionID, ok := readPlaySessionCookie(r); ok {
		resp, err := s.auth.GetWebSession(r.Context(), &authv1.GetWebSessionRequest{SessionId: sessionID})
		if err == nil && resp != nil && resp.GetSession() != nil {
			userID := strings.TrimSpace(resp.GetSession().GetUserId())
			if userID != "" {
				return userID, false
			}
		}
		clearPlaySessionCookie(w, r, s.requestSchemePolicy)
	}
	return "", false
}

func stripLaunchGrant(r *http.Request) string {
	if r == nil || r.URL == nil {
		return "/"
	}
	cloned := new(url.URL)
	*cloned = *r.URL
	query := cloned.Query()
	query.Del("launch")
	cloned.RawQuery = query.Encode()
	return cloned.String()
}

func loggerOrDefault(logger *slog.Logger) *slog.Logger {
	if logger == nil {
		return slog.Default()
	}
	return logger
}

func writeRPCError(w http.ResponseWriter, err error) {
	if w == nil {
		return
	}
	if err == nil {
		writeJSONError(w, http.StatusInternalServerError, "unknown error")
		return
	}
	code := gogrpcstatus.Code(err)
	switch code {
	case gogrpccodes.InvalidArgument:
		writeJSONError(w, http.StatusBadRequest, gogrpcstatus.Convert(err).Message())
	case gogrpccodes.PermissionDenied:
		writeJSONError(w, http.StatusForbidden, gogrpcstatus.Convert(err).Message())
	case gogrpccodes.NotFound:
		writeJSONError(w, http.StatusNotFound, gogrpcstatus.Convert(err).Message())
	case gogrpccodes.FailedPrecondition, gogrpccodes.Aborted:
		writeJSONError(w, http.StatusConflict, gogrpcstatus.Convert(err).Message())
	case gogrpccodes.Unauthenticated:
		writeJSONError(w, http.StatusUnauthorized, gogrpcstatus.Convert(err).Message())
	default:
		writeJSONError(w, http.StatusBadGateway, "upstream request failed")
	}
}

func parseInt64(value string) (int64, error) {
	var parsed int64
	_, err := fmt.Sscan(strings.TrimSpace(value), &parsed)
	return parsed, err
}

func parseInt(value string) (int, error) {
	var parsed int
	_, err := fmt.Sscan(strings.TrimSpace(value), &parsed)
	return parsed, err
}

func pathForCampaignAPI(campaignID string, suffix string) string {
	campaignID = strings.TrimSpace(campaignID)
	suffix = strings.Trim(strings.TrimSpace(suffix), "/")
	if suffix == "" {
		return "/api/campaigns/" + campaignID
	}
	return "/api/campaigns/" + campaignID + "/" + suffix
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
	if s.transcripts != nil {
		_ = s.transcripts.Close()
	}
	if s.gameMc != nil {
		_ = s.gameMc.Close()
	}
	if s.authMc != nil {
		_ = s.authMc.Close()
	}
}
