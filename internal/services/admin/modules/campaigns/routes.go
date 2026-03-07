package campaigns

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

func newRoutes(h Handlers) *http.ServeMux {
	mux := http.NewServeMux()
	if h == nil {
		mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaigns, http.NotFound)
		mux.HandleFunc(http.MethodGet+" "+routepath.CampaignsPrefix+"{$}", http.NotFound)
		return mux
	}

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaigns, func(w http.ResponseWriter, r *http.Request) {
		if wantsRowsFragment(r) {
			h.HandleCampaignsTable(w, r)
			return
		}
		h.HandleCampaignsPage(w, r)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.CampaignsPrefix+"{$}", func(w http.ResponseWriter, r *http.Request) {
		if wantsRowsFragment(r) {
			h.HandleCampaignsTable(w, r)
			return
		}
		h.HandleCampaignsPage(w, r)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		if campaignID == "" || strings.EqualFold(campaignID, "create") {
			http.NotFound(w, r)
			return
		}
		h.HandleCampaignDetail(w, r, campaignID)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharactersPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		if campaignID == "" {
			http.NotFound(w, r)
			return
		}
		if wantsRowsFragment(r) {
			h.HandleCharactersTable(w, r, campaignID)
			return
		}
		h.HandleCharactersList(w, r, campaignID)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		characterID := strings.TrimSpace(r.PathValue("characterID"))
		if campaignID == "" || characterID == "" || isLegacyTableSegment(characterID) {
			http.NotFound(w, r)
			return
		}
		h.HandleCharacterSheet(w, r, campaignID, characterID)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterActivityPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		characterID := strings.TrimSpace(r.PathValue("characterID"))
		if campaignID == "" || characterID == "" {
			http.NotFound(w, r)
			return
		}
		h.HandleCharacterActivity(w, r, campaignID, characterID)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignParticipantsPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		if campaignID == "" {
			http.NotFound(w, r)
			return
		}
		if wantsRowsFragment(r) {
			h.HandleParticipantsTable(w, r, campaignID)
			return
		}
		h.HandleParticipantsList(w, r, campaignID)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignInvitesPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		if campaignID == "" {
			http.NotFound(w, r)
			return
		}
		if wantsRowsFragment(r) {
			h.HandleInvitesTable(w, r, campaignID)
			return
		}
		h.HandleInvitesList(w, r, campaignID)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionsPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		if campaignID == "" {
			http.NotFound(w, r)
			return
		}
		if wantsRowsFragment(r) {
			h.HandleSessionsTable(w, r, campaignID)
			return
		}
		h.HandleSessionsList(w, r, campaignID)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		sessionID := strings.TrimSpace(r.PathValue("sessionID"))
		if campaignID == "" || sessionID == "" {
			http.NotFound(w, r)
			return
		}
		h.HandleSessionDetail(w, r, campaignID, sessionID)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionEventsPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		sessionID := strings.TrimSpace(r.PathValue("sessionID"))
		if campaignID == "" || sessionID == "" {
			http.NotFound(w, r)
			return
		}
		h.HandleSessionEvents(w, r, campaignID, sessionID)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignEventsPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		if campaignID == "" {
			http.NotFound(w, r)
			return
		}
		if wantsRowsFragment(r) {
			h.HandleEventLogTable(w, r, campaignID)
			return
		}
		h.HandleEventLog(w, r, campaignID)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.CampaignsPrefix+"{campaignID}/{rest...}", http.NotFound)
	return mux
}

func wantsRowsFragment(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(r.URL.Query().Get(routepath.FragmentQueryKey)), routepath.FragmentRows)
}

func isLegacyTableSegment(segment string) bool {
	switch strings.ToLower(strings.TrimSpace(segment)) {
	case "table", "_rows":
		return true
	default:
		return false
	}
}
