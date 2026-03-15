package app

import (
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
