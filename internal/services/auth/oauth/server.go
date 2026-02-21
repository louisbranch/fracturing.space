package oauth

import (
	"context"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/auth/user"
)

// UserStore persists and retrieves auth user records.
type UserStore interface {
	PutUser(ctx context.Context, u user.User) error
	GetUser(ctx context.Context, userID string) (user.User, error)
}

var resolveStaticFS = func() (fs.FS, error) {
	return fs.Sub(assetsFS, "static")
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

// Server hosts OAuth endpoints and external provider flows.
type Server struct {
	config     Config
	store      *Store
	userStore  UserStore
	clock      func() time.Time
	httpClient *http.Client
}

// NewServer builds an OAuth server bound to auth config and backing stores.
func NewServer(config Config, store *Store, userStore UserStore) *Server {
	return &Server{
		config:     config,
		store:      store,
		userStore:  userStore,
		clock:      time.Now,
		httpClient: http.DefaultClient,
	}
}

// RegisterRoutes registers OAuth HTTP endpoints on the provided mux.
func (s *Server) RegisterRoutes(mux *http.ServeMux) error {
	if mux == nil {
		return nil
	}

	staticFS, err := resolveStaticFS()
	if err != nil {
		return fmt.Errorf("register static routes: %w", err)
	}
	mux.Handle(
		"/static/",
		withStaticMime(http.StripPrefix("/static/", http.FileServer(http.FS(staticFS)))),
	)

	mux.HandleFunc("/authorize", s.handleAuthorize)
	mux.HandleFunc("/authorize/consent", s.handleConsent)
	mux.HandleFunc("/token", s.handleToken)
	mux.HandleFunc("/introspect", s.handleIntrospect)
	mux.HandleFunc("/.well-known/oauth-authorization-server", s.handleMetadata)
	mux.HandleFunc("/oauth/providers/", s.handleProviderRoutes)
	mux.HandleFunc("/up", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	})
	return nil
}

// StartCleanup starts periodic expiry cleanup for transient OAuth artifacts.
//
// This keeps short-lived authorization and pending login records from
// accumulating without requiring a separate maintenance process.
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
