package campaigns

import (
	"net/http"
	"strings"

	sharedroute "github.com/louisbranch/fracturing.space/internal/services/shared/route"
	routepath "github.com/louisbranch/fracturing.space/internal/services/web/routepath"
)

type campaignDetailRouteDescriptor struct {
	length   int
	literals map[int]string
	handle   func(Service, http.ResponseWriter, *http.Request, []string)
}

func (d campaignDetailRouteDescriptor) matches(parts []string) bool {
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

var campaignDetailRouteDescriptors = []campaignDetailRouteDescriptor{
	{
		length:   2,
		literals: map[int]string{1: "sessions"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCampaignSessions(w, r, parts[0])
		},
	},
	{
		length:   3,
		literals: map[int]string{1: "sessions", 2: "start"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCampaignSessionStart(w, r, parts[0])
		},
	},
	{
		length:   3,
		literals: map[int]string{1: "sessions", 2: "end"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCampaignSessionEnd(w, r, parts[0])
		},
	},
	{
		length:   3,
		literals: map[int]string{1: "sessions"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCampaignSessionDetail(w, r, parts[0], parts[2])
		},
	},
	{
		length:   2,
		literals: map[int]string{1: "participants"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCampaignParticipants(w, r, parts[0])
		},
	},
	{
		length:   3,
		literals: map[int]string{1: "participants", 2: "update"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCampaignParticipantUpdate(w, r, parts[0])
		},
	},
	{
		length:   2,
		literals: map[int]string{1: "characters"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCampaignCharacters(w, r, parts[0])
		},
	},
	{
		length:   3,
		literals: map[int]string{1: "characters", 2: "create"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCampaignCharacterCreate(w, r, parts[0])
		},
	},
	{
		length:   3,
		literals: map[int]string{1: "characters", 2: "update"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCampaignCharacterUpdate(w, r, parts[0])
		},
	},
	{
		length:   3,
		literals: map[int]string{1: "characters", 2: "control"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCampaignCharacterControl(w, r, parts[0])
		},
	},
	{
		length:   3,
		literals: map[int]string{1: "characters"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCampaignCharacterDetail(w, r, parts[0], parts[2])
		},
	},
	{
		length:   2,
		literals: map[int]string{1: "invites"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCampaignInvites(w, r, parts[0])
		},
	},
	{
		length:   3,
		literals: map[int]string{1: "invites", 2: "create"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCampaignInviteCreate(w, r, parts[0])
		},
	},
	{
		length:   3,
		literals: map[int]string{1: "invites", 2: "revoke"},
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCampaignInviteRevoke(w, r, parts[0])
		},
	},
	{
		length: 1,
		handle: func(service Service, w http.ResponseWriter, r *http.Request, parts []string) {
			service.HandleCampaignOverview(w, r, parts[0])
		},
	},
}

func dispatchCampaignDetailPath(service Service, w http.ResponseWriter, r *http.Request, parts []string) bool {
	return dispatchMostSpecificCampaignDetailPath(campaignDetailRouteDescriptors, service, w, r, parts)
}

func dispatchMostSpecificCampaignDetailPath(
	descriptors []campaignDetailRouteDescriptor,
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

// Service is the campaign workspace transport contract consumed by the route module.
type Service interface {
	HandleCampaigns(w http.ResponseWriter, r *http.Request)
	HandleCampaignCreate(w http.ResponseWriter, r *http.Request)
	HandleCampaignOverview(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignSessions(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignSessionStart(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignSessionEnd(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignSessionDetail(w http.ResponseWriter, r *http.Request, campaignID string, sessionID string)
	HandleCampaignParticipants(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignParticipantUpdate(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignCharacters(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignCharacterCreate(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignCharacterUpdate(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignCharacterControl(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignCharacterDetail(w http.ResponseWriter, r *http.Request, campaignID string, characterID string)
	HandleCampaignInvites(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignInviteCreate(w http.ResponseWriter, r *http.Request, campaignID string)
	HandleCampaignInviteRevoke(w http.ResponseWriter, r *http.Request, campaignID string)
}

// RegisterRoutes wires campaign workspace routes into the provided mux.
func RegisterRoutes(mux *http.ServeMux, service Service) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc(routepath.AppCampaigns, service.HandleCampaigns)
	mux.HandleFunc(routepath.AppCampaignsCreate, service.HandleCampaignCreate)
	mux.HandleFunc(routepath.AppCampaignsPrefix, func(w http.ResponseWriter, r *http.Request) {
		HandleCampaignDetailPath(w, r, service)
	})
}

// HandleCampaignDetailPath parses campaign workspace subpaths and dispatches to campaign handlers.
func HandleCampaignDetailPath(w http.ResponseWriter, r *http.Request, service Service) {
	if service == nil {
		http.NotFound(w, r)
		return
	}
	if sharedroute.RedirectTrailingSlash(w, r) {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, routepath.AppCampaignsPrefix)
	if path == "" || strings.HasPrefix(path, "/") || strings.Contains(path, "//") {
		http.NotFound(w, r)
		return
	}
	rawParts := strings.Split(path, "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		parts = append(parts, part)
	}
	if !dispatchCampaignDetailPath(service, w, r, parts) {
		http.NotFound(w, r)
	}
}
