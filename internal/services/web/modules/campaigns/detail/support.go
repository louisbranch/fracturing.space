package detail

import (
	"net/http"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/dashboardsync"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/httpx"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/modulehandler"
	"github.com/louisbranch/fracturing.space/internal/services/web/platform/requestmeta"
)

// DashboardSync keeps campaign mutations aligned with dashboard freshness.
type DashboardSync = dashboardsync.Service

// Support owns the cross-surface helpers reused by campaign workspace detail
// routes.
type Support struct {
	modulehandler.Base
	requestMeta requestmeta.SchemePolicy
	nowFunc     func() time.Time
	sync        DashboardSync
}

// NewSupport keeps shared transport defaults in one constructor instead of
// repeating them across route-owner assembly.
func NewSupport(base modulehandler.Base, meta requestmeta.SchemePolicy, sync DashboardSync) Support {
	if sync == nil {
		sync = dashboardsync.Noop{}
	}
	return Support{
		Base:        base,
		requestMeta: meta,
		nowFunc:     time.Now,
		sync:        sync,
	}
}

// Now centralizes this web behavior in one helper seam.
func (h Support) Now() time.Time {
	if h.nowFunc != nil {
		return h.nowFunc()
	}
	return time.Now()
}

// Sync returns the dashboard freshness notifier used by mutation surfaces.
func (h Support) Sync() DashboardSync {
	return h.sync
}

// RequestMeta returns the request scheme policy used by transport helpers that
// need to build absolute same-origin links.
func (h Support) RequestMeta() requestmeta.SchemePolicy {
	return h.requestMeta
}

// RouteCampaignID extracts the canonical campaign route parameter.
func (h Support) RouteCampaignID(r *http.Request) (string, bool) {
	return httpx.ReadRouteParam(r, "campaignID")
}

// RouteCharacterID centralizes campaign character route-param extraction.
func (h Support) RouteCharacterID(r *http.Request) (string, bool) {
	return httpx.ReadRouteParam(r, "characterID")
}

// RouteParticipantID centralizes campaign participant route-param extraction.
func (h Support) RouteParticipantID(r *http.Request) (string, bool) {
	return httpx.ReadRouteParam(r, "participantID")
}

// RouteSessionID centralizes campaign session route-param extraction.
func (h Support) RouteSessionID(r *http.Request) (string, bool) {
	return httpx.ReadRouteParam(r, "sessionID")
}

// RouteStarterKey centralizes protected starter route-param extraction.
func (h Support) RouteStarterKey(r *http.Request) (string, bool) {
	return httpx.ReadRouteParam(r, "starterKey")
}

// WithCampaignID extracts the campaign ID path param and delegates to fn,
// returning 404 when the param is missing.
func (h Support) WithCampaignID(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		campaignID, ok := h.RouteCampaignID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		fn(w, r, campaignID)
	}
}

// WithCampaignAndParticipantID extracts campaign/participant IDs and delegates
// to fn, returning 404 when either route parameter is missing.
func (h Support) WithCampaignAndParticipantID(fn func(http.ResponseWriter, *http.Request, string, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		campaignID, ok := h.RouteCampaignID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		participantID, ok := h.RouteParticipantID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		fn(w, r, campaignID, participantID)
	}
}

// WithCampaignAndCharacterID extracts campaign/character IDs and delegates to
// fn, returning 404 when either route parameter is missing.
func (h Support) WithCampaignAndCharacterID(fn func(http.ResponseWriter, *http.Request, string, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		campaignID, ok := h.RouteCampaignID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		characterID, ok := h.RouteCharacterID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		fn(w, r, campaignID, characterID)
	}
}

// WithCampaignAndSessionID extracts campaign/session IDs and delegates to fn,
// returning 404 when either route parameter is missing.
func (h Support) WithCampaignAndSessionID(fn func(http.ResponseWriter, *http.Request, string, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		campaignID, ok := h.RouteCampaignID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		sessionID, ok := h.RouteSessionID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		fn(w, r, campaignID, sessionID)
	}
}
