package admin

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"

	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/composition"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	sharedhttpx "github.com/louisbranch/fracturing.space/internal/services/shared/httpx"
)

//go:embed static/*
var staticAssets embed.FS

var resolveStaticFS = func() (fs.FS, error) {
	return fs.Sub(staticAssets, "static")
}

// Handler routes admin dashboard requests.
type Handler struct {
	grpcClients *grpcClients
	// gameClientInitMu serializes on-demand game client bootstrap attempts.
	gameClientInitMu sync.Mutex
	// gameClientEnsureInProgress tracks whether a background game bootstrap is running.
	gameClientInitInProgress bool
	grpcAddr                 string
	statusClient             statusv1.StatusServiceClient
	authConfig               *AuthConfig
	introspector             TokenIntrospector
}

// NewServiceHandler builds a handler instance without attaching transport routing
// or auth middleware wrappers. Callers can reuse this service surface inside
// alternative composition layers.
func NewServiceHandler(clients *grpcClients, grpcAddr string, authCfg *AuthConfig, statusClient statusv1.StatusServiceClient) *Handler {
	handler := &Handler{
		grpcClients:  clients,
		grpcAddr:     strings.TrimSpace(grpcAddr),
		statusClient: statusClient,
		authConfig:   authCfg,
	}
	if authCfg != nil && authCfg.IntrospectURL != "" && authCfg.LoginURL != "" {
		handler.introspector = newHTTPIntrospector(authCfg.IntrospectURL, authCfg.ResourceSecret)
	}
	return handler
}

// NewHandler builds the HTTP handler for the admin server (no auth).
func NewHandler(clients *grpcClients) http.Handler {
	return NewHandlerWithConfig(clients, "", nil, nil)
}

// NewHandlerWithConfig builds the HTTP handler with explicit configuration.
// When authCfg is non-nil and fully populated, requests are guarded by
// token introspection; otherwise admin runs without authentication.
func NewHandlerWithConfig(clients *grpcClients, grpcAddr string, authCfg *AuthConfig, statusClient statusv1.StatusServiceClient) http.Handler {
	handler := NewServiceHandler(clients, grpcAddr, authCfg, statusClient)
	mux := handler.routes()
	mux = handler.withGameClientBootstrap(mux)
	root := sharedhttpx.Chain(mux, sharedhttpx.RecoverPanic(), sharedhttpx.RequestID("admin"))
	if handler.introspector == nil {
		return root
	}
	return requireAuth(root, handler.introspector, authCfg.LoginURL)
}

// moduleBuildInput extracts individual gRPC clients from the handler's grpcClients
// and assembles a BuildInput for the module registry.
func (h *Handler) moduleBuildInput() modules.BuildInput {
	input := modules.BuildInput{
		Base:         modulehandler.NewBase(),
		GRPCAddr:     h.grpcAddr,
		StatusClient: h.statusClient,
	}
	if c := h.grpcClients; c != nil {
		input.AuthClient = c.AuthClient()
		input.CampaignClient = c.CampaignClient()
		input.CharacterClient = c.CharacterClient()
		input.ParticipantClient = c.ParticipantClient()
		input.InviteClient = c.InviteClient()
		input.SessionClient = c.SessionClient()
		input.EventClient = c.EventClient()
		input.StatisticsClient = c.StatisticsClient()
		input.SystemClient = c.SystemClient()
		input.DaggerheartContentClient = c.DaggerheartContentClient()
	}
	return input
}

// routes wires the HTTP routes for the admin handler.
func (h *Handler) routes() http.Handler {
	rootMux := http.NewServeMux()
	staticFS, err := resolveStaticFS()
	if err == nil {
		staticHandler := http.StripPrefix(routepath.StaticPrefix, http.FileServer(http.FS(staticFS)))
		rootMux.Handle(routepath.StaticPrefix, staticHandler)
	} else {
		log.Printf("admin: failed to initialize static assets: %v", err)
	}
	composed, err := composition.ComposeAppHandler(composition.ComposeInput{
		Modules: h.moduleBuildInput(),
	})
	if err != nil {
		log.Printf("admin: failed to compose app handler: %v", err)
		composed = http.NotFoundHandler()
	}
	rootMux.HandleFunc(http.MethodGet+" "+routepath.Root+"{$}", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, routepath.AppDashboard, http.StatusFound)
	})
	rootMux.Handle(routepath.Root, composed)
	return rootMux
}

// withGameClientBootstrap ensures game clients are initialized lazily from the admin handler context.
func (h *Handler) withGameClientBootstrap(next http.Handler) http.Handler {
	if h == nil || next == nil {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ensureGameClients(r.Context())
		next.ServeHTTP(w, r)
	})
}

func (h *Handler) ensureGameClients(ctx context.Context) {
	if h == nil || h.grpcClients == nil {
		return
	}
	clients := h.grpcClients
	if clients.HasGameConnection() {
		return
	}

	grpcAddr := strings.TrimSpace(h.grpcAddr)
	if grpcAddr == "" {
		return
	}

	h.gameClientInitMu.Lock()
	if clients.HasGameConnection() || h.gameClientInitInProgress {
		h.gameClientInitMu.Unlock()
		return
	}
	h.gameClientInitInProgress = true
	h.gameClientInitMu.Unlock()

	go func() {
		connectGameGRPCWithRetry(context.Background(), Config{
			GRPCAddr: grpcAddr,
		}, clients)
		h.gameClientInitMu.Lock()
		h.gameClientInitInProgress = false
		h.gameClientInitMu.Unlock()
	}()
}
