package app

import (
	"fmt"
	"net/http"
	"strings"

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

func mountProtectedModule(root *http.ServeMux, feature module.Module, seen map[string]string, wrap func(http.Handler) http.Handler) error {
	mount, prefix, err := resolveMount(feature)
	if err != nil {
		return err
	}
	if !isProtectedPrefix(prefix) {
		return fmt.Errorf("module %q must mount under /app/, got %q", feature.ID(), prefix)
	}
	if err := mountModule(root, feature, mount, prefix, seen, wrap); err != nil {
		return err
	}
	if alias := protectedSlashlessPrefixAlias(prefix); alias != "" {
		if err := mountModule(root, feature, mount, alias, seen, wrap); err != nil {
			return err
		}
	}
	return nil
}

func isProtectedPrefix(prefix string) bool {
	return strings.HasPrefix(prefix, routepath.AppPrefix)
}

func resolveMount(feature module.Module) (module.Mount, string, error) {
	if feature == nil {
		return module.Mount{}, "", fmt.Errorf("module is nil")
	}
	mount, err := feature.Mount()
	if err != nil {
		return module.Mount{}, "", fmt.Errorf("mount module %q: %w", feature.ID(), err)
	}
	prefix := strings.TrimSpace(mount.Prefix)
	if err := validatePrefix(prefix); err != nil {
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

func validatePrefix(prefix string) error {
	if prefix == "" {
		return fmt.Errorf("prefix is required")
	}
	if strings.TrimSpace(prefix) != prefix {
		return fmt.Errorf("prefix must not include surrounding whitespace")
	}
	if !strings.HasPrefix(prefix, "/") {
		return fmt.Errorf("prefix must begin with /")
	}
	if !strings.HasSuffix(prefix, "/") {
		return fmt.Errorf("prefix must end with /")
	}
	return nil
}

func protectedSlashlessPrefixAlias(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if !isProtectedPrefix(prefix) || !strings.HasSuffix(prefix, "/") {
		return ""
	}
	alias := strings.TrimSuffix(prefix, "/")
	if alias == "" {
		return ""
	}
	return alias
}

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

func wrapProtectedModule(authenticated func(*http.Request) bool, policy requestmeta.SchemePolicy) func(http.Handler) http.Handler {
	authWrap := requireAuth(authenticated)
	csrfWrap := requireCookieSessionSameOrigin(policy)
	return func(next http.Handler) http.Handler {
		return authWrap(csrfWrap(next))
	}
}

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

func hasSameOriginProof(r *http.Request, policy requestmeta.SchemePolicy) bool {
	return requestmeta.HasSameOriginProofWithPolicy(r, policy)
}
