package campaigns

import (
	"net/http"

	"github.com/louisbranch/fracturing.space/internal/services/web/platform/routeparam"
)

// routeCampaignID extracts the canonical campaign route parameter.
func (h handlers) routeCampaignID(r *http.Request) (string, bool) {
	return routeparam.Read(r, "campaignID")
}

// routeCharacterID centralizes campaign character route-param extraction.
func (h handlers) routeCharacterID(r *http.Request) (string, bool) {
	return routeparam.Read(r, "characterID")
}

// routeParticipantID centralizes campaign participant route-param extraction.
func (h handlers) routeParticipantID(r *http.Request) (string, bool) {
	return routeparam.Read(r, "participantID")
}

// routeSessionID centralizes campaign session route-param extraction.
func (h handlers) routeSessionID(r *http.Request) (string, bool) {
	return routeparam.Read(r, "sessionID")
}

// withCampaignID extracts the campaign ID path param and delegates to fn,
// returning 404 when the param is missing.
func (h handlers) withCampaignID(fn func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		campaignID, ok := h.routeCampaignID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		fn(w, r, campaignID)
	}
}

// withCampaignAndParticipantID extracts campaign/participant IDs and delegates
// to fn, returning 404 when either route parameter is missing.
func (h handlers) withCampaignAndParticipantID(fn func(http.ResponseWriter, *http.Request, string, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		campaignID, ok := h.routeCampaignID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		participantID, ok := h.routeParticipantID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		fn(w, r, campaignID, participantID)
	}
}

// withCampaignAndCharacterID extracts campaign/character IDs and delegates to
// fn, returning 404 when either route parameter is missing.
func (h handlers) withCampaignAndCharacterID(fn func(http.ResponseWriter, *http.Request, string, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		campaignID, ok := h.routeCampaignID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		characterID, ok := h.routeCharacterID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		fn(w, r, campaignID, characterID)
	}
}

// withCampaignAndSessionID extracts campaign/session IDs and delegates to fn,
// returning 404 when either route parameter is missing.
func (h handlers) withCampaignAndSessionID(fn func(http.ResponseWriter, *http.Request, string, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		campaignID, ok := h.routeCampaignID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		sessionID, ok := h.routeSessionID(r)
		if !ok {
			h.WriteNotFound(w, r)
			return
		}
		fn(w, r, campaignID, sessionID)
	}
}
