package app

import (
	"fmt"
	"net/http"
	"strings"

	module "github.com/louisbranch/fracturing.space/internal/services/web2/module"
	"github.com/louisbranch/fracturing.space/internal/services/web2/platform/requestmeta"
	"github.com/louisbranch/fracturing.space/internal/services/web2/platform/sessioncookie"
	"github.com/louisbranch/fracturing.space/internal/services/web2/routepath"
)

const defaultLoginPath = routepath.Login

// ComposeInput carries module groups and shared composition contracts.
type ComposeInput struct {
	Dependencies     module.Dependencies
	AuthRequired     func(*http.Request) bool
	PublicModules    []module.Module
	ProtectedModules []module.Module
}

// Composer wires root mux mounts and route-group auth behavior.
type Composer struct{}

// Compose builds a root HTTP handler from module groups.
func (Composer) Compose(input ComposeInput) (http.Handler, error) {
	root := http.NewServeMux()
	if input.AuthRequired == nil {
		input.AuthRequired = func(*http.Request) bool { return false }
	}
	seen := make(map[string]string)

	for _, feature := range input.PublicModules {
		if feature == nil {
			return nil, fmt.Errorf("public module is nil")
		}
		if err := mountPublicModule(root, feature, input.Dependencies, seen); err != nil {
			return nil, err
		}
	}

	for _, feature := range input.ProtectedModules {
		if feature == nil {
			return nil, fmt.Errorf("protected module is nil")
		}
		if err := mountProtectedModule(root, feature, input.Dependencies, seen, wrapProtectedModule(input.AuthRequired)); err != nil {
			return nil, err
		}
	}

	return root, nil
}

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

func mountPublicModule(root *http.ServeMux, feature module.Module, deps module.Dependencies, seen map[string]string) error {
	mount, prefix, err := resolveMount(feature, deps)
	if err != nil {
		return err
	}
	if isProtectedPrefix(prefix) {
		return fmt.Errorf("module %q has protected prefix %q in public group", feature.ID(), prefix)
	}
	return mountModule(root, feature, mount, prefix, seen, nil)
}

func mountProtectedModule(root *http.ServeMux, feature module.Module, deps module.Dependencies, seen map[string]string, wrap func(http.Handler) http.Handler) error {
	mount, prefix, err := resolveMount(feature, deps)
	if err != nil {
		return err
	}
	if !isProtectedPrefix(prefix) {
		return fmt.Errorf("module %q must mount under /app/, got %q", feature.ID(), prefix)
	}
	return mountModule(root, feature, mount, prefix, seen, wrap)
}

func isProtectedPrefix(prefix string) bool {
	return strings.HasPrefix(prefix, routepath.AppPrefix)
}

func resolveMount(feature module.Module, deps module.Dependencies) (module.Mount, string, error) {
	if feature == nil {
		return module.Mount{}, "", fmt.Errorf("module is nil")
	}
	mount, err := feature.Mount(deps)
	if err != nil {
		return module.Mount{}, "", fmt.Errorf("mount module %q: %w", feature.ID(), err)
	}
	prefix := normalizePrefix(mount.Prefix)
	if prefix == "" {
		return module.Mount{}, "", fmt.Errorf("mount module %q: prefix is required", feature.ID())
	}
	if mount.Handler == nil {
		return module.Mount{}, "", fmt.Errorf("mount module %q: handler is required", feature.ID())
	}
	return mount, prefix, nil
}

func normalizePrefix(prefix string) string {
	// TODO(web2-composition): consider rejecting non-canonical prefixes instead of silently normalizing them.
	prefix = strings.TrimSpace(prefix)
	if prefix == "" {
		return ""
	}
	if !strings.HasPrefix(prefix, "/") {
		prefix = "/" + prefix
	}
	if !strings.HasSuffix(prefix, "/") {
		prefix += "/"
	}
	return prefix
}

func requireAuth(authenticated func(*http.Request) bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			return http.NotFoundHandler()
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !authenticated(r) {
				http.Redirect(w, r, defaultLoginPath, http.StatusFound)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func wrapProtectedModule(authenticated func(*http.Request) bool) func(http.Handler) http.Handler {
	authWrap := requireAuth(authenticated)
	csrfWrap := requireCookieSessionSameOrigin()
	return func(next http.Handler) http.Handler {
		return authWrap(csrfWrap(next))
	}
}

func requireCookieSessionSameOrigin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		if next == nil {
			next = http.NotFoundHandler()
		}
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !isMutationMethod(r) || !hasSessionCookie(r) {
				next.ServeHTTP(w, r)
				return
			}
			if !hasSameOriginProof(r) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

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

func hasSessionCookie(r *http.Request) bool {
	_, ok := sessioncookie.Read(r)
	return ok
}

func hasSameOriginProof(r *http.Request) bool {
	return requestmeta.HasSameOriginProof(r)
}
