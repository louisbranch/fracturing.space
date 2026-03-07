package app

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/shared/modulecompose"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/sessioncookie"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

const defaultLoginPath = routepath.Login

// ComposeInput carries module groups and shared composition contracts.
type ComposeInput struct {
	AuthRequired        func(*http.Request) bool
	PublicModules       []module.Module
	ProtectedModules    []module.Module
	RequestSchemePolicy requestmeta.SchemePolicy
}

// Compose builds a root HTTP handler from module groups.
func Compose(input ComposeInput) (http.Handler, error) {
	root := http.NewServeMux()
	if input.AuthRequired == nil {
		input.AuthRequired = func(*http.Request) bool { return false }
	}
	seen := make(map[string]string)

	for _, feature := range input.PublicModules {
		if feature == nil {
			return nil, fmt.Errorf("public module is nil")
		}
		if err := mountPublicModule(root, feature, seen); err != nil {
			return nil, err
		}
	}

	for _, feature := range input.ProtectedModules {
		if feature == nil {
			return nil, fmt.Errorf("protected module is nil")
		}
		if err := mountProtectedModule(root, feature, seen, wrapProtectedModule(input.AuthRequired, input.RequestSchemePolicy)); err != nil {
			return nil, err
		}
	}

	return root, nil
}

// mountModule performs shared mount bookkeeping so public/protected mounting
// paths enforce one prefix owner and optional wrapper behavior consistently.
func mountModule(
	root *http.ServeMux,
	feature module.Module,
	mount module.Mount,
	prefix string,
	seen map[string]string,
	wrap func(http.Handler) http.Handler,
) error {
	if root == nil || feature == nil {
		return nil
	}
	if previous, ok := seen[prefix]; ok {
		return fmt.Errorf("module %q duplicates prefix %q owned by module %q", feature.ID(), prefix, previous)
	}
	seen[prefix] = feature.ID()

	handler := mount.Handler
	if wrap != nil {
		handler = wrap(handler)
	}
	root.Handle(prefix, handler)
	return nil
}

// mountPublicModule enforces that public modules never claim protected prefixes.
func mountPublicModule(root *http.ServeMux, feature module.Module, seen map[string]string) error {
	mount, prefix, err := resolveMount(feature)
	if err != nil {
		return err
	}
	if isProtectedPrefix(prefix) {
		return fmt.Errorf("module %q has protected prefix %q in public group", feature.ID(), prefix)
	}
	return mountModule(root, feature, mount, prefix, seen, nil)
}

// mountProtectedModule enforces protected-prefix ownership for canonical
// `/app/*` module roots.
func mountProtectedModule(root *http.ServeMux, feature module.Module, seen map[string]string, wrap func(http.Handler) http.Handler) error {
	mount, prefix, err := resolveMount(feature)
	if err != nil {
		return err
	}
	if !isProtectedPrefix(prefix) {
		return fmt.Errorf("module %q must mount under /app/, got %q", feature.ID(), prefix)
	}
	return mountModule(root, feature, mount, prefix, seen, wrap)
}

// isProtectedPrefix centralizes the `/app/` ownership rule used during compose.
func isProtectedPrefix(prefix string) bool {
	return strings.HasPrefix(prefix, routepath.AppPrefix)
}

// resolveMount validates module mount contracts once so callers can share
// canonical prefix and handler checks.
func resolveMount(feature module.Module) (module.Mount, string, error) {
	if feature == nil {
		return module.Mount{}, "", fmt.Errorf("module is nil")
	}
	mount, err := feature.Mount()
	if err != nil {
		return module.Mount{}, "", fmt.Errorf("mount module %q: %w", feature.ID(), err)
	}
	prefix := strings.TrimSpace(mount.Prefix)
	if err := modulecompose.ValidatePrefix(prefix); err != nil {
		return module.Mount{}, "", fmt.Errorf("mount module %q has invalid prefix %q: %w", feature.ID(), mount.Prefix, err)
	}
	if prefix == "" {
		return module.Mount{}, "", fmt.Errorf("mount module %q: prefix is required", feature.ID())
	}
	if mount.Handler == nil {
		return module.Mount{}, "", fmt.Errorf("mount module %q: handler is required", feature.ID())
	}
	return mount, prefix, nil
}

// requireAuth wraps handlers with session-backed auth checks and redirects to
// the shared login path when auth is missing.
func requireAuth(authenticated func(*http.Request) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			return http.NotFoundHandler()
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authenticated(r) {
				httpx.WriteRedirect(w, r, defaultLoginPath)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// wrapProtectedModule composes auth and same-origin protections for protected
// modules so each module mount receives identical guardrails.
func wrapProtectedModule(authenticated func(*http.Request) bool, policy requestmeta.SchemePolicy) func(http.Handler) http.Handler {
	authWrap := requireAuth(authenticated)
	csrfWrap := requireCookieSessionSameOrigin(policy)
	return func(next http.Handler) http.Handler {
		return authWrap(csrfWrap(next))
	}
}

// requireCookieSessionSameOrigin enforces same-origin proof for cookie-backed
// mutation requests and leaves non-mutation reads untouched.
func requireCookieSessionSameOrigin(policy requestmeta.SchemePolicy) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isMutationMethod(r) || !hasSessionCookie(r) {
				next.ServeHTTP(w, r)
				return
			}
			if !hasSameOriginProof(r, policy) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// isMutationMethod identifies state-changing HTTP verbs for same-origin checks.
func isMutationMethod(r *http.Request) bool {
	if r == nil {
		return false
	}
	switch r.Method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

// hasSessionCookie reports whether the request carries an authenticated web
// session cookie and therefore requires same-origin mutation proof.
func hasSessionCookie(r *http.Request) bool {
	_, ok := sessioncookie.Read(r)
	return ok
}

// hasSameOriginProof delegates proof validation to shared requestmeta helpers.
func hasSameOriginProof(r *http.Request, policy requestmeta.SchemePolicy) bool {
	return requestmeta.HasSameOriginProofWithPolicy(r, policy)
}
