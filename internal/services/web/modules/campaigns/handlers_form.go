package campaigns

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
)

// requireParsedForm parses POST form values and writes a flash error notice
// with redirect when parsing fails.
func (h handlers) requireParsedForm(
	w http.ResponseWriter,
	r *http.Request,
	localizationKey string,
	redirectURL string,
) bool {
	if err := r.ParseForm(); err != nil {
		flash.Write(w, r, flash.Notice{Kind: flash.KindError, Key: localizationKey})
		httpx.WriteRedirect(w, r, redirectURL)
		return false
	}
	return true
}
