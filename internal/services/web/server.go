package web

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"sync"
	"time"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	connectionsv1 "github.com/louisbranch/fracturing.space/api/gen/go/connections/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	notificationsv1 "github.com/louisbranch/fracturing.space/api/gen/go/notifications/v1"
	webstorage "github.com/louisbranch/fracturing.space/internal/services/web/storage"
	websqlite "github.com/louisbranch/fracturing.space/internal/services/web/storage/sqlite"
	websupport "github.com/louisbranch/fracturing.space/internal/services/web/support"
	webhttp "github.com/louisbranch/fracturing.space/internal/services/web/transport/http"
	httpmux "github.com/louisbranch/fracturing.space/internal/services/web/transport/httpmux"
	"golang.org/x/text/message"
	"google.golang.org/grpc"
)

var subStaticFS = func() (fs.FS, error) {
	return fs.Sub(assetsFS, "static")
}

// Config defines the inputs for the web login server.
type Config struct {
	HTTPAddr             string
	ChatHTTPAddr         string
	AuthBaseURL          string
	AuthAddr             string
	ConnectionsAddr      string
	GameAddr             string
	NotificationsAddr    string
	AIAddr               string
	ListingAddr          string
	CacheDBPath          string
	AssetBaseURL         string
	AssetManifestVersion string
	AppName              string
	GRPCDialTimeout      time.Duration
	// OAuthClientID is the first-party OAuth client ID for web login.
	OAuthClientID string
	// CallbackURL is the public URL for the OAuth callback endpoint.
	CallbackURL string
	// AuthTokenURL is the internal auth token endpoint for code exchange.
	AuthTokenURL string
	// Domain is the parent domain used for cross-subdomain cookie scoping.
	Domain string
	// OAuthResourceSecret is used by web service to introspect access tokens.
	OAuthResourceSecret string
}

// Server hosts the web login HTTP server.
type Server struct {
	httpAddr                       string
	httpServer                     *http.Server
	authConn                       *grpc.ClientConn
	connectionsConn                *grpc.ClientConn
	gameConn                       *grpc.ClientConn
	notificationsConn              *grpc.ClientConn
	aiConn                         *grpc.ClientConn
	listingConn                    *grpc.ClientConn
	cacheStore                     *websqlite.Store
	cacheInvalidationDone          chan struct{}
	cacheInvalidationStop          context.CancelFunc
	campaignUpdateSubscriptionDone chan struct{}
	campaignUpdateSubscriptionStop context.CancelFunc
}

type handler struct {
	config              Config
	authClient          authv1.AuthServiceClient
	connectionsClient   connectionsv1.ConnectionsServiceClient
	accountClient       authv1.AccountServiceClient
	credentialClient    aiv1.CredentialServiceClient
	sessions            *sessionStore
	pendingFlows        *pendingFlowStore
	cacheStore          webstorage.Store
	clientInitMu        sync.Mutex
	campaignNameCacheMu sync.RWMutex
	campaignNameCache   map[string]campaignNameCache
	campaignClient      statev1.CampaignServiceClient
	eventClient         statev1.EventServiceClient
	sessionClient       statev1.SessionServiceClient
	participantClient   statev1.ParticipantServiceClient
	characterClient     statev1.CharacterServiceClient
	inviteClient        statev1.InviteServiceClient
	notificationClient  notificationsv1.NotificationServiceClient
	listingClient       listingv1.CampaignListingServiceClient
	campaignAccess      campaignAccessChecker
}

type handlerDependencies struct {
	campaignAccess     campaignAccessChecker
	cacheStore         webstorage.Store
	accountClient      authv1.AccountServiceClient
	connectionsClient  connectionsv1.ConnectionsServiceClient
	credentialClient   aiv1.CredentialServiceClient
	campaignClient     statev1.CampaignServiceClient
	eventClient        statev1.EventServiceClient
	sessionClient      statev1.SessionServiceClient
	participantClient  statev1.ParticipantServiceClient
	characterClient    statev1.CharacterServiceClient
	inviteClient       statev1.InviteServiceClient
	notificationClient notificationsv1.NotificationServiceClient
	listingClient      listingv1.CampaignListingServiceClient
}

// localizer resolves the request locale, optionally persists a cookie,
// and returns a message printer with the resolved language tag string.
func localizer(w http.ResponseWriter, r *http.Request) (*message.Printer, string) {
	return websupport.ResolveLocalizer(w, r)
}

// NewHandler creates the HTTP handler for the login UX.
//
// This function is the test-oriented entrypoint that assembles route handlers
// while keeping gRPC dependencies injectable via NewHandlerWithCampaignAccess.
func NewHandler(config Config, authClient authv1.AuthServiceClient) http.Handler {
	handler, err := NewHandlerWithCampaignAccess(config, authClient, handlerDependencies{})
	if err != nil {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			localizeHTTPError(w, r, http.StatusInternalServerError, "error.http.web_handler_unavailable")
		})
	}
	return handler
}

// NewHandlerWithCampaignAccess creates the HTTP handler with campaign access checks.
func NewHandlerWithCampaignAccess(config Config, authClient authv1.AuthServiceClient, deps handlerDependencies) (http.Handler, error) {
	rootMux := http.NewServeMux()
	staticFS, err := subStaticFS()
	if err != nil {
		return nil, fmt.Errorf("resolve static assets: %w", err)
	}
	var sessionPersistence sessionPersistence
	if webSessionStore, ok := deps.cacheStore.(*websqlite.Store); ok && webSessionStore != nil {
		sessionPersistence = webSessionStore
	}
	httpmux.MountStatic(rootMux, staticFS, webhttp.WithStaticMime)

	h := &handler{
		config:             config,
		authClient:         authClient,
		connectionsClient:  deps.connectionsClient,
		accountClient:      deps.accountClient,
		credentialClient:   deps.credentialClient,
		sessions:           newSessionStore(sessionPersistence),
		pendingFlows:       newPendingFlowStore(),
		cacheStore:         deps.cacheStore,
		campaignNameCache:  make(map[string]campaignNameCache),
		campaignClient:     deps.campaignClient,
		eventClient:        deps.eventClient,
		sessionClient:      deps.sessionClient,
		participantClient:  deps.participantClient,
		characterClient:    deps.characterClient,
		inviteClient:       deps.inviteClient,
		notificationClient: deps.notificationClient,
		listingClient:      deps.listingClient,
		campaignAccess:     deps.campaignAccess,
	}

	gameMux := http.NewServeMux()
	h.registerGameRoutes(gameMux)

	publicMux := http.NewServeMux()
	h.registerPublicRoutes(publicMux)
	httpmux.MountAppAndPublicRoutes(rootMux, gameMux, publicMux)

	return rootMux, nil
}

// NewServer builds a configured web server.
func NewServer(config Config) (*Server, error) {
	return NewServerWithContext(context.Background(), config)
}

// NewServerWithContext builds a configured web server.
func NewServerWithContext(ctx context.Context, config Config) (*Server, error) {
	return newServerWithContext(ctx, config)
}

// ListenAndServe runs the HTTP server until the context ends.
//
// On cancellation, it performs a bounded shutdown so in-flight requests
// are drained before hard close.
func (s *Server) ListenAndServe(ctx context.Context) error {
	return s.listenAndServe(ctx)
}

// Close releases any gRPC resources held by the server.
func (s *Server) Close() {
	s.close()
}
