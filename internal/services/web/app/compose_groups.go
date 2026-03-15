package app

import (
	"fmt"
	"net/http"

	module "github.com/louisbranch/fracturing.space/internal/services/web/module"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

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
