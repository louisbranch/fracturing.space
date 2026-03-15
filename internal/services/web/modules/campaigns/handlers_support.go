package campaigns

import (
	"net/http"
	"time"

	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// campaignRouteSupport owns the cross-surface helpers reused by the campaigns
// transport package.
type campaignRouteSupport struct {
	modulehandler.Base
	requestMeta requestmeta.SchemePolicy
	nowFunc     func() time.Time
	sync        DashboardSync
}

// now centralizes this web behavior in one helper seam.
func (h campaignRouteSupport) now() time.Time {
	if h.nowFunc != nil {
		return h.nowFunc()
	}
	return time.Now()
}

// writeMutationError writes a flash error notice and redirects back to the
// originating page so the user stays in context and can retry.
func (h campaignRouteSupport) writeMutationError(w http.ResponseWriter, r *http.Request, err error, fallbackKey, redirectURL string) {
	notice := flash.Notice{Kind: flash.KindError}
	if key := apperrors.LocalizationKey(err); key != "" {
		notice.Key = key
	} else {
		notice.Key = fallbackKey
	}
	flash.Write(w, r, notice)
	httpx.WriteRedirect(w, r, redirectURL)
}

// writeMutationSuccess writes a success flash notice and redirects to the
// target page so the user sees confirmation feedback.
func (h campaignRouteSupport) writeMutationSuccess(w http.ResponseWriter, r *http.Request, key, redirectURL string) {
	flash.Write(w, r, flash.NoticeSuccess(key))
	httpx.WriteRedirect(w, r, redirectURL)
}
