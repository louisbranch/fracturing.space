package campaigns

import (
	"net/http"
	"strings"

	sharedpath "github.com/louisbranch/fracturing.space/internal/services/admin/module/sharedpath"
	routepath "github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	sharedroute "github.com/louisbranch/fracturing.space/internal/services/shared/route"
)

type campaignRouteDescriptor struct {
	length   int
	literals map[int]string
	handle   func(Service, http.ResponseWriter, *http.Request, []string)
}

func (d campaignRouteDescriptor) matches(parts []string) bool {
	if len(parts) != d.length {
		return false
	}
	for index, value := range d.literals {
		if parts[index] != value {
			return false
		}
	}
	return true
}

var campaignRouteDescriptors = []campaignRouteDescriptor{
	{
		length:   2,
		literals: map[int]string{1: "characters"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCharactersList(w, r, parts[0])
		},
	},
	{
		length:   3,
		literals: map[int]string{1: "characters", 2: "table"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCharactersTable(w, r, parts[0])
		},
	},
	{
		length:   4,
		literals: map[int]string{1: "characters", 3: "activity"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCharacterActivity(w, r, parts[0], parts[2])
		},
	},
	{
		length:   3,
		literals: map[int]string{1: "characters"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCharacterSheet(w, r, parts[0], parts[2])
		},
	},
	{
		length:   2,
		literals: map[int]string{1: "participants"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleParticipantsList(w, r, parts[0])
		},
	},
	{
		length:   3,
		literals: map[int]string{1: "participants", 2: "table"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleParticipantsTable(w, r, parts[0])
		},
	},
	{
		length:   2,
		literals: map[int]string{1: "invites"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleInvitesList(w, r, parts[0])
		},
	},
	{
		length:   3,
		literals: map[int]string{1: "invites", 2: "table"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleInvitesTable(w, r, parts[0])
		},
	},
	{
		length:   2,
		literals: map[int]string{1: "sessions"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleSessionsList(w, r, parts[0])
		},
	},
	{
		length:   3,
		literals: map[int]string{1: "sessions", 2: "table"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleSessionsTable(w, r, parts[0])
		},
	},
	{
		length:   4,
		literals: map[int]string{1: "sessions", 3: "events"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleSessionEvents(w, r, parts[0], parts[2])
		},
	},
	{
		length:   3,
		literals: map[int]string{1: "sessions"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleSessionDetail(w, r, parts[0], parts[2])
		},
	},
	{
		length:   2,
		literals: map[int]string{1: "events"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleEventLog(w, r, parts[0])
		},
	},
	{
		length:   3,
		literals: map[int]string{1: "events", 2: "table"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleEventLogTable(w, r, parts[0])
		},
	},
	{
		length: 1,
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCampaignDetail(w, r, parts[0])
		},
	},
}

func dispatchCampaignPath(service Service, w http.ResponseWriter, r *http.Request, parts []string) bool {
	return dispatchMostSpecificCampaignPath(campaignRouteDescriptors, service, w, r, parts)
}

func dispatchMostSpecificCampaignPath(
	descriptors []campaignRouteDescriptor,
	service Service,
	w http.ResponseWriter,
	r *http.Request,
	parts []string,
) bool {
	bestIndex := -1
	bestSpecificity := -1
	for index, descriptor := range descriptors {
		if !descriptor.matches(parts) {
			continue
		}
		specificity := len(descriptor.literals)
		if specificity > bestSpecificity {
			bestSpecificity = specificity
			bestIndex = index
		}
	}
	if bestIndex < 0 {
		return false
	}
	descriptors[bestIndex].handle(service, w, r, parts)
	return true
}

// Service defines campaign route handlers consumed by this route module.
type Service interface {
	HandleCampaignsPage(w http.ResponseWriter, r *http.Request)
	HandleCampaignsTable(w http.ResponseWriter, r *http.Request)
	HandleCampaignDetail(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCharactersList(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCharactersTable(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCharacterSheet(w http.ResponseWriter, r *http.Request, campaignID string, characterID string)
	HandleCharacterActivity(w http.ResponseWriter, r *http.Request, campaignID string, characterID string)
	HandleParticipantsList(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleParticipantsTable(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleInvitesList(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleInvitesTable(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleSessionsList(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleSessionsTable(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleSessionDetail(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string)
	HandleSessionEvents(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string)
	HandleEventLog(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleEventLogTable(w http.ResponseWriter, r *http.Request, campaignID string)
}

// RegisterRoutes wires campaign routes into the provided mux.
func RegisterRoutes(mux *http.ServeMux, service Service) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc(routepath.Campaigns, service.HandleCampaignsPage)
	mux.HandleFunc(routepath.CampaignsTable, service.HandleCampaignsTable)
	mux.HandleFunc(routepath.CampaignsPrefix, func(w http.ResponseWriter, r *http.Request) {
		HandleCampaignPath(w, r, service)
	})
}

// HandleCampaignPath parses campaign subroutes and dispatches to service handlers.
func HandleCampaignPath(w http.ResponseWriter, r *http.Request, service Service) {
	if service == nil {
		http.NotFound(w, r)
		return
	}
	if sharedroute.RedirectTrailingSlash(w, r) {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, routepath.CampaignsPrefix)
	parts := sharedpath.SplitPathParts(path)
	if len(parts) == 1 && strings.EqualFold(parts[0], "create") {
		http.NotFound(w, r)
		return
	}
	if !dispatchCampaignPath(service, w, r, parts) {
		http.NotFound(w, r)
	}
}
