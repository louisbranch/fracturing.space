package campaigns

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

func newRoutes(service Service) *http.ServeMux {
	mux := http.NewServeMux()
	if service == nil {
		mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaigns, http.NotFound)
		mux.HandleFunc(http.MethodGet+" "+routepath.CampaignsPrefix+"{$}", http.NotFound)
		return mux
	}

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaigns, func(w http.ResponseWriter, r *http.Request) {
		if wantsRowsFragment(r) {
			service.HandleCampaignsTable(w, r)
			return
		}
		service.HandleCampaignsPage(w, r)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.CampaignsPrefix+"{$}", func(w http.ResponseWriter, r *http.Request) {
		if wantsRowsFragment(r) {
			service.HandleCampaignsTable(w, r)
			return
		}
		service.HandleCampaignsPage(w, r)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		if campaignID == "" || strings.EqualFold(campaignID, "create") {
			http.NotFound(w, r)
			return
		}
		service.HandleCampaignDetail(w, r, campaignID)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharactersPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		if campaignID == "" {
			http.NotFound(w, r)
			return
		}
		if wantsRowsFragment(r) {
			service.HandleCharactersTable(w, r, campaignID)
			return
		}
		service.HandleCharactersList(w, r, campaignID)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		characterID := strings.TrimSpace(r.PathValue("characterID"))
		if campaignID == "" || characterID == "" || isLegacyTableSegment(characterID) {
			http.NotFound(w, r)
			return
		}
		service.HandleCharacterSheet(w, r, campaignID, characterID)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignCharacterActivityPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		characterID := strings.TrimSpace(r.PathValue("characterID"))
		if campaignID == "" || characterID == "" {
			http.NotFound(w, r)
			return
		}
		service.HandleCharacterActivity(w, r, campaignID, characterID)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignParticipantsPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		if campaignID == "" {
			http.NotFound(w, r)
			return
		}
		if wantsRowsFragment(r) {
			service.HandleParticipantsTable(w, r, campaignID)
			return
		}
		service.HandleParticipantsList(w, r, campaignID)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignInvitesPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		if campaignID == "" {
			http.NotFound(w, r)
			return
		}
		if wantsRowsFragment(r) {
			service.HandleInvitesTable(w, r, campaignID)
			return
		}
		service.HandleInvitesList(w, r, campaignID)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionsPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		if campaignID == "" {
			http.NotFound(w, r)
			return
		}
		if wantsRowsFragment(r) {
			service.HandleSessionsTable(w, r, campaignID)
			return
		}
		service.HandleSessionsList(w, r, campaignID)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		sessionID := strings.TrimSpace(r.PathValue("sessionID"))
		if campaignID == "" || sessionID == "" {
			http.NotFound(w, r)
			return
		}
		service.HandleSessionDetail(w, r, campaignID, sessionID)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignSessionEventsPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		sessionID := strings.TrimSpace(r.PathValue("sessionID"))
		if campaignID == "" || sessionID == "" {
			http.NotFound(w, r)
			return
		}
		service.HandleSessionEvents(w, r, campaignID, sessionID)
	})

	mux.HandleFunc(http.MethodGet+" "+routepath.AppCampaignEventsPattern, func(w http.ResponseWriter, r *http.Request) {
		campaignID := strings.TrimSpace(r.PathValue("campaignID"))
		if campaignID == "" {
			http.NotFound(w, r)
			return
		}
		if wantsRowsFragment(r) {
			service.HandleEventLogTable(w, r, campaignID)
			return
		}
		service.HandleEventLog(w, r, campaignID)
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
