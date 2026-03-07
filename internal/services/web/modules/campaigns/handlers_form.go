package campaigns

import (
	"net/http"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// requireParsedForm parses POST form values and writes a localized invalid-input
// error response when parsing fails.
func (h handlers) requireParsedForm(
	w http.ResponseWriter,
	r *http.Request,
	localizationKey string,
	fallbackMessage string,
) bool {
	if err := r.ParseForm(); err != nil {
		h.WriteError(w, r, apperrors.EK(apperrors.KindInvalidInput, localizationKey, fallbackMessage))
		return false
	}
	return true
}
