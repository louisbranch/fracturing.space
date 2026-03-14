// Package forminput provides shared form-parsing helpers for web handlers.
package forminput

import (
	"net/http"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
)

// ParseInvalidInput parses form values and maps malformed bodies to the
// standard invalid-input transport error used by web modules.
func ParseInvalidInput(r *http.Request, localizationKey, message string) error {
	if err := parse(r); err != nil {
		return apperrors.EK(apperrors.KindInvalidInput, localizationKey, message)
	}
	return nil
}

// ParseOrRedirectErrorNotice parses form values and redirects back to the
// current module page with one flash error notice when parsing fails.
func ParseOrRedirectErrorNotice(w http.ResponseWriter, r *http.Request, localizationKey, redirectURL string) bool {
	if err := parse(r); err != nil {
		flash.Write(w, r, flash.Notice{Kind: flash.KindError, Key: localizationKey})
		httpx.WriteRedirect(w, r, redirectURL)
		return false
	}
	return true
}

// parse centralizes nil-safe form parsing for the package transport policies.
func parse(r *http.Request) error {
	if r == nil {
		return http.ErrMissingFile
	}
	return r.ParseForm()
}
