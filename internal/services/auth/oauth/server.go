package oauth

import (
	"context"
	"net/http"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
)

// UserWriter persists auth user records.
type UserWriter interface {
	PutUser(ctx context.Context, u user.User) error
}

// Server hosts OAuth endpoints and external provider flows.
type Server struct {
	config     Config
	store      *Store
	userStore  UserWriter
	clock      func() time.Time
	httpClient *http.Client
}

// NewServer creates a new OAuth server.
func NewServer(config Config, store *Store, userStore UserWriter) *Server {
	return &Server{
		config:     config,
		store:      store,
		userStore:  userStore,
		clock:      time.Now,
		httpClient: http.DefaultClient,
	}
}

// RegisterRoutes registers OAuth HTTP endpoints on the provided mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	if mux == nil {
		return
	}

	mux.HandleFunc("/authorize", s.handleAuthorize)
	mux.HandleFunc("/authorize/login", s.handleLogin)
	mux.HandleFunc("/authorize/consent", s.handleConsent)
	mux.HandleFunc("/token", s.handleToken)
	mux.HandleFunc("/introspect", s.handleIntrospect)
	mux.HandleFunc("/.well-known/oauth-authorization-server", s.handleMetadata)
	mux.HandleFunc("/oauth/providers/", s.handleProviderRoutes)
	mux.HandleFunc("/up", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
}

// StartCleanup runs periodic cleanup for expired OAuth entries.
func (s *Server) StartCleanup(ctx context.Context, interval time.Duration) {
	if s == nil || s.store == nil || interval <= 0 {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.store.CleanupExpired(s.clock().UTC())
			}
		}
	}()
}
