package campaigns

import (
	"net/http"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/flash"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// DashboardSync keeps campaign mutations aligned with dashboard freshness.
type DashboardSync = dashboardsync.Service

// campaignRouteSupport owns the cross-surface helpers reused by the campaigns
// transport package.
type campaignRouteSupport struct {
	modulehandler.Base
	requestMeta requestmeta.SchemePolicy
	nowFunc     func() time.Time
	sync        DashboardSync
}

// newCampaignRouteSupport keeps shared transport defaults in one constructor
// instead of repeating them across route-owner assembly.
func newCampaignRouteSupport(base modulehandler.Base, meta requestmeta.SchemePolicy, sync DashboardSync) campaignRouteSupport {
	if sync == nil {
		sync = dashboardsync.Noop{}
	}
	return campaignRouteSupport{
		Base:        base,
		requestMeta: meta,
		nowFunc:     time.Now,
		sync:        sync,
	}
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
