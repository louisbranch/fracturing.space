package app

import (
	"fmt"
	"net/http"
	"strings"

	mod "github.com/louisbranch/fracturing.space/internal/services/admin/module"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	"github.com/louisbranch/fracturing.space/internal/services/shared/modulecompose"
)

// ComposeInput carries module groups and shared composition contracts.
type ComposeInput struct {
	Modules []mod.Module
}

// Compose builds a root HTTP handler from module mounts.
func Compose(input ComposeInput) (http.Handler, error) {
	root := http.NewServeMux()
	seen := map[string]string{}

	for _, feature := range input.Modules {
		if feature == nil {
			return nil, fmt.Errorf("module is nil")
		}
		mount, err := feature.Mount()
		if err != nil {
			return nil, fmt.Errorf("mount module %q: %w", feature.ID(), err)
		}
		prefix := strings.TrimSpace(mount.Prefix)
		if prefix == "" {
			return nil, fmt.Errorf("mount module %q: prefix is required", feature.ID())
		}
		if mount.Handler == nil {
			return nil, fmt.Errorf("mount module %q: handler is required", feature.ID())
		}
		if err := modulecompose.ValidatePrefix(prefix); err != nil {
			return nil, fmt.Errorf("mount module %q has invalid prefix %q: %w", feature.ID(), prefix, err)
		}
		if !isProtectedPrefix(prefix) {
			return nil, fmt.Errorf("module %q must mount under %s, got %q", feature.ID(), routepath.AppPrefix, prefix)
		}
		if prev, ok := seen[prefix]; ok {
			return nil, fmt.Errorf("module %q duplicates prefix %q owned by module %q", feature.ID(), prefix, prev)
		}
		seen[prefix] = feature.ID()
		root.Handle(prefix, mount.Handler)
	}

	return root, nil
}

func isProtectedPrefix(prefix string) bool {
	return strings.HasPrefix(prefix, routepath.AppPrefix)
}
