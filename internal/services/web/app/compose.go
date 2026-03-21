package app

import (
	"fmt"
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// ComposeInput carries module groups and shared composition contracts.
type ComposeInput struct {
	AuthRequired        func(*http.Request) bool
	PublicModules       []module.Module
	ProtectedModules    []module.Module
	RequestSchemePolicy requestmeta.SchemePolicy
}

// Compose builds a root HTTP handler from module groups.
func Compose(input ComposeInput) (http.Handler, error) {
	input = normalizeComposeInput(input)
	root := http.NewServeMux()
	seen := make(map[string]string)

	if err := mountPublicModules(root, input.PublicModules, seen); err != nil {
		return nil, err
	}
	if err := mountProtectedModules(root, input.ProtectedModules, seen, input.AuthRequired, input.RequestSchemePolicy); err != nil {
		return nil, err
	}

	return root, nil
}

// normalizeComposeInput fills nil-safe root composition defaults so tests and
// production wiring both traverse the same assembly path.
func normalizeComposeInput(input ComposeInput) ComposeInput {
	if input.AuthRequired == nil {
		input.AuthRequired = func(*http.Request) bool { return false }
	}
	return input
}

// mountPublicModules applies public-prefix policy to the public module group.
func mountPublicModules(root *http.ServeMux, features []module.Module, seen map[string]string) error {
	for _, feature := range features {
		if feature == nil {
			return fmt.Errorf("public module is nil")
		}
		if err := mountPublicModule(root, feature, seen); err != nil {
			return err
		}
	}
	return nil
}

// mountProtectedModules applies auth and same-origin policy uniformly to the
// protected module group.
func mountProtectedModules(
	root *http.ServeMux,
	features []module.Module,
	seen map[string]string,
	authRequired func(*http.Request) bool,
	policy requestmeta.SchemePolicy,
) error {
	wrap := wrapProtectedModule(authRequired, policy)
	for _, feature := range features {
		if feature == nil {
			return fmt.Errorf("protected module is nil")
		}
		if err := mountProtectedModule(root, feature, seen, wrap); err != nil {
			return err
		}
	}
	return nil
}
