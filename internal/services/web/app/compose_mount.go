package app

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/shared/modulecompose"
	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

// mountModule performs shared mount bookkeeping so public and protected
// mounting paths enforce one prefix owner and optional wrapper behavior
// consistently.
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
	if err := claimRoute(seen, prefix, feature.ID()); err != nil {
		return err
	}

	handler := mount.Handler
	if wrap != nil {
		handler = wrap(handler)
	}
	handler = canonicalizeTrailingSlash(prefix, mount.CanonicalRoot, handler)

	if mount.CanonicalRoot {
		rootPath := strings.TrimSuffix(prefix, "/")
		if rootPath == "" || rootPath == routepath.Root {
			return fmt.Errorf("module %q has invalid canonical root for prefix %q", feature.ID(), prefix)
		}
		if err := claimRoute(seen, rootPath, feature.ID()); err != nil {
			return err
		}
		root.Handle(rootPath, handler)
	}

	root.Handle(prefix, handler)
	return nil
}

// claimRoute preserves one-owner route claims during app composition so module
// collisions fail fast.
func claimRoute(seen map[string]string, pattern string, owner string) error {
	if previous, ok := seen[pattern]; ok {
		return fmt.Errorf("module %q duplicates prefix %q owned by module %q", owner, pattern, previous)
	}
	seen[pattern] = owner
	return nil
}

// mountPublicModule enforces that public modules never claim protected
// prefixes.
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

// isProtectedPrefix centralizes the `/app/` ownership rule used during
// composition.
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
	if prefix == routepath.Root && mount.CanonicalRoot {
		return module.Mount{}, "", fmt.Errorf("mount module %q: root mount cannot claim canonical root", feature.ID())
	}
	if mount.Handler == nil {
		return module.Mount{}, "", fmt.Errorf("mount module %q: handler is required", feature.ID())
	}
	return mount, prefix, nil
}
