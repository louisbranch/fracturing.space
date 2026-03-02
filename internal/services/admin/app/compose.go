package app

import (
	"fmt"
	"net/http"
	"strings"

	mod "github.com/louisbranch/fracturing.space/internal/services/admin/module"
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
		if err := validatePrefix(prefix); err != nil {
			return nil, fmt.Errorf("mount module %q has invalid prefix %q: %w", feature.ID(), prefix, err)
		}
		if prev, ok := seen[prefix]; ok {
			return nil, fmt.Errorf("module %q duplicates prefix %q owned by module %q", feature.ID(), prefix, prev)
		}
		seen[prefix] = feature.ID()
		root.Handle(prefix, mount.Handler)

		if prefix != "/" && strings.HasSuffix(prefix, "/") {
			alias := strings.TrimSuffix(prefix, "/")
			if alias != "" {
				if _, ok := seen[alias]; !ok {
					seen[alias] = feature.ID()
					root.Handle(alias, rewritePath(alias, mount.Handler))
				}
			}
		}
	}

	return root, nil
}

func rewritePath(path string, next http.Handler) http.Handler {
	if next == nil {
		next = http.NotFoundHandler()
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r == nil {
			next.ServeHTTP(w, r)
			return
		}
		clone := r.Clone(r.Context())
		if clone.URL != nil {
			urlCopy := *clone.URL
			urlCopy.Path = path
			clone.URL = &urlCopy
		}
		next.ServeHTTP(w, clone)
	})
}

func validatePrefix(prefix string) error {
	if !strings.HasPrefix(prefix, "/") {
		return fmt.Errorf("prefix must begin with /")
	}
	if strings.TrimSpace(prefix) != prefix {
		return fmt.Errorf("prefix must not include surrounding whitespace")
	}
	return nil
}
