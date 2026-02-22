package admin

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"strings"
	"sync"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/requestctx"
	"github.com/louisbranch/fracturing.space/internal/platform/timeouts"
	"github.com/louisbranch/fracturing.space/internal/services/admin/i18n"
	campaignsmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/campaigns"
	catalogmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/catalog"
	dashboardmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/dashboard"
	iconsmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/icons"
	scenariosmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/scenarios"
	systemsmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/systems"
	usersmodule "github.com/louisbranch/fracturing.space/internal/services/admin/module/users"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	httpmux "github.com/louisbranch/fracturing.space/internal/services/admin/transport/httpmux"
	"github.com/louisbranch/fracturing.space/internal/services/shared/grpcauthctx"
	"golang.org/x/text/message"
)

const (
	// grpcRequestTimeout caps the gRPC request time for admin requests.
	grpcRequestTimeout = timeouts.GRPCRequest
	// campaignThemePromptLimit caps the number of characters shown in the table.
	campaignThemePromptLimit = 80
	// sessionListPageSize caps the number of sessions shown in the UI.
	sessionListPageSize = 10
	// eventListPageSize caps the number of events shown per page.
	eventListPageSize = 50
	// inviteListPageSize caps the number of invites shown per page.
	inviteListPageSize = 50
	// catalogListPageSize caps the number of catalog entries shown per page.
	catalogListPageSize = 25
	// catalogDescriptionLimit caps the number of characters shown in catalog tables.
	catalogDescriptionLimit = 80
	// maxScenarioScriptSize caps scenario scripts to limit resource usage.
	maxScenarioScriptSize = 100 * 1024
	// scenarioTempDirEnv configures the temp directory for scenario scripts.
	scenarioTempDirEnv = "FRACTURING_SPACE_SCENARIO_TMPDIR"
)

//go:embed static/*
var staticAssets embed.FS

var resolveStaticFS = func() (fs.FS, error) {
	return fs.Sub(staticAssets, "static")
}

// GRPCClientProvider supplies gRPC clients for request handling.
type GRPCClientProvider interface {
	AuthClient() authv1.AuthServiceClient
	AccountClient() authv1.AccountServiceClient
	CampaignClient() statev1.CampaignServiceClient
	SessionClient() statev1.SessionServiceClient
	CharacterClient() statev1.CharacterServiceClient
	ParticipantClient() statev1.ParticipantServiceClient
	InviteClient() statev1.InviteServiceClient
	SnapshotClient() statev1.SnapshotServiceClient
	EventClient() statev1.EventServiceClient
	StatisticsClient() statev1.StatisticsServiceClient
	SystemClient() statev1.SystemServiceClient
	DaggerheartContentClient() daggerheartv1.DaggerheartContentServiceClient
}

// Handler routes admin dashboard requests.
type Handler struct {
	clientProvider GRPCClientProvider
	// gameClientInitMu serializes on-demand game client bootstrap attempts.
	gameClientInitMu sync.Mutex
	// gameClientEnsureInProgress tracks whether a background game bootstrap is running.
	gameClientInitInProgress bool
	grpcAddr                 string
	authConfig               *AuthConfig
	introspector             TokenIntrospector
}

// NewHandler builds the HTTP handler for the admin server (no auth).
func NewHandler(clientProvider GRPCClientProvider) http.Handler {
	return NewHandlerWithConfig(clientProvider, "", nil)
}

// NewHandlerWithConfig builds the HTTP handler with explicit configuration.
// When authCfg is non-nil and fully populated, requests are guarded by
// token introspection; otherwise admin runs without authentication.
func NewHandlerWithConfig(clientProvider GRPCClientProvider, grpcAddr string, authCfg *AuthConfig) http.Handler {
	handler := &Handler{
		clientProvider: clientProvider,
		grpcAddr:       strings.TrimSpace(grpcAddr),
		authConfig:     authCfg,
	}
	if authCfg != nil && authCfg.IntrospectURL != "" && authCfg.LoginURL != "" {
		handler.introspector = newHTTPIntrospector(authCfg.IntrospectURL, authCfg.ResourceSecret)
	}
	mux := handler.routes()
	mux = handler.withGameClientBootstrap(mux)
	if handler.introspector == nil {
		return mux
	}
	return requireAuth(mux, handler.introspector, authCfg.LoginURL)
}

func (h *Handler) localizer(w http.ResponseWriter, r *http.Request) (*message.Printer, string) {
	tag, persist := i18n.ResolveTag(r)
	if persist {
		i18n.SetLanguageCookie(w, tag)
	}
	return i18n.Printer(tag), tag.String()
}

func (h *Handler) pageContext(lang string, loc *message.Printer, r *http.Request) templates.PageContext {
	path := ""
	query := ""
	if r != nil && r.URL != nil {
		path = r.URL.Path
		query = r.URL.RawQuery
	}
	return templates.PageContext{
		Lang:         lang,
		Loc:          loc,
		CurrentPath:  path,
		CurrentQuery: query,
	}
}

// gameGRPCCallContext creates a bounded game RPC context with user identity.
// Admin override is injected by connection-level interceptors, not per-call.
func (h *Handler) gameGRPCCallContext(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	ctx, cancel := context.WithTimeout(parent, grpcRequestTimeout)
	if userID := strings.TrimSpace(requestctx.UserIDFromContext(parent)); userID != "" {
		ctx = grpcauthctx.WithUserID(ctx, userID)
	}
	return ctx, cancel
}

func withStaticMime(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch path := strings.ToLower(r.URL.Path); {
		case strings.HasSuffix(path, ".css"):
			w.Header().Set("Content-Type", "text/css")
		case strings.HasSuffix(path, ".js"):
			w.Header().Set("Content-Type", "application/javascript")
		case strings.HasSuffix(path, ".svg"):
			w.Header().Set("Content-Type", "image/svg+xml")
		}
		next.ServeHTTP(w, r)
	})
}

// routes wires the HTTP routes for the admin handler.
func (h *Handler) routes() http.Handler {
	rootMux := http.NewServeMux()
	adminMux := http.NewServeMux()
	staticFS, err := resolveStaticFS()
	if err == nil {
		httpmux.MountStatic(rootMux, staticFS, withStaticMime)
	} else {
		log.Printf("admin: failed to initialize static assets: %v", err)
	}
	dashboardmodule.RegisterRoutes(adminMux, h)
	campaignsmodule.RegisterRoutes(adminMux, h)
	systemsmodule.RegisterRoutes(adminMux, h)
	catalogmodule.RegisterRoutes(adminMux, h)
	iconsmodule.RegisterRoutes(adminMux, h)
	usersmodule.RegisterRoutes(adminMux, h)
	scenariosmodule.RegisterRoutes(adminMux, h)

	httpmux.MountAdminRoutes(rootMux, adminMux)
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
	if h == nil || h.clientProvider == nil {
		return
	}
	clients, ok := h.clientProvider.(*grpcClients)
	if !ok || clients == nil {
		return
	}
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

// handleCampaignsTable returns the first page of campaign rows for HTMX.
