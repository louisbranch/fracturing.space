package admin

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"

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
	server       *Server
	grpcAddr     string
	statusClient statusv1.StatusServiceClient
	authConfig   *AuthConfig
	introspector TokenIntrospector
}

// NewServiceHandler builds a handler instance without attaching transport routing
// or auth middleware wrappers. Callers can reuse this service surface inside
// alternative composition layers.
func NewServiceHandler(server *Server, grpcAddr string, authCfg *AuthConfig, statusClient statusv1.StatusServiceClient) *Handler {
	handler := &Handler{
		server:       server,
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
func NewHandler(server *Server) http.Handler {
	return NewHandlerWithConfig(server, "", nil, nil)
}

// NewHandlerWithConfig builds the HTTP handler with explicit configuration.
// When authCfg is non-nil and fully populated, requests are guarded by
// token introspection; otherwise admin runs without authentication.
func NewHandlerWithConfig(server *Server, grpcAddr string, authCfg *AuthConfig, statusClient statusv1.StatusServiceClient) http.Handler {
	handler := NewServiceHandler(server, grpcAddr, authCfg, statusClient)
	mux := handler.routes()
	root := sharedhttpx.Chain(mux, sharedhttpx.RecoverPanic(), sharedhttpx.RequestID("admin"))
	if handler.introspector == nil {
		return root
	}
	return requireAuth(root, handler.introspector, authCfg.LoginURL)
}

// moduleBuildInput extracts individual gRPC clients from the server and
// assembles a BuildInput for the module registry.
func (h *Handler) moduleBuildInput() modules.BuildInput {
	input := modules.BuildInput{
		Base:         modulehandler.NewBase(),
		GRPCAddr:     h.grpcAddr,
		StatusClient: h.statusClient,
	}
	if s := h.server; s != nil {
		input.AuthClient = s.AuthClient()
		input.CampaignClient = s.CampaignClient()
		input.CharacterClient = s.CharacterClient()
		input.ParticipantClient = s.ParticipantClient()
		input.InviteClient = s.InviteClient()
		input.SessionClient = s.SessionClient()
		input.EventClient = s.EventClient()
		input.StatisticsClient = s.StatisticsClient()
		input.SystemClient = s.SystemClient()
		input.DaggerheartContentClient = s.DaggerheartContentClient()
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
