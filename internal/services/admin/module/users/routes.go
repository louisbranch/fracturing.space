package users

import (
	"net/http"
	"strings"

	sharedpath "github.com/louisbranch/fracturing.space/internal/services/admin/module/sharedpath"
	routepath "github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	sharedroute "github.com/louisbranch/fracturing.space/internal/services/shared/route"
)

// Service defines users route handlers consumed by this route module.
type Service interface {
	HandleUsersPage(w http.ResponseWriter, r *http.Request)
	HandleUsersTable(w http.ResponseWriter, r *http.Request)
	HandleUserLookup(w http.ResponseWriter, r *http.Request)
	HandleMagicLink(w http.ResponseWriter, r *http.Request)
	HandleUserDetail(w http.ResponseWriter, r *http.Request, userID string)
	HandleUserInvites(w http.ResponseWriter, r *http.Request, userID string)
}

// RegisterRoutes wires user routes into the provided mux.
func RegisterRoutes(mux *http.ServeMux, service Service) {
	if mux == nil || service == nil {
		return
	}
	mux.HandleFunc(routepath.Users, service.HandleUsersPage)
	mux.HandleFunc(routepath.UsersTable, service.HandleUsersTable)
	mux.HandleFunc(routepath.UsersLookup, service.HandleUserLookup)
	mux.Handle(routepath.UsersCreate, http.NotFoundHandler())
	mux.HandleFunc(routepath.UsersMagicLink, service.HandleMagicLink)
	mux.HandleFunc(routepath.UsersPrefix, func(w http.ResponseWriter, r *http.Request) {
		HandleUserPath(w, r, service)
	})
}

// HandleUserPath parses user detail subroutes and dispatches to service handlers.
func HandleUserPath(w http.ResponseWriter, r *http.Request, service Service) {
	if service == nil {
		http.NotFound(w, r)
		return
	}
	if sharedroute.RedirectTrailingSlash(w, r) {
		return
	}

	path := strings.TrimPrefix(r.URL.Path, routepath.UsersPrefix)
	parts := sharedpath.SplitPathParts(path)
	if len(parts) == 2 && parts[1] == "invites" {
		service.HandleUserInvites(w, r, parts[0])
		return
	}
	if len(parts) == 1 {
		service.HandleUserDetail(w, r, parts[0])
		return
	}
	http.NotFound(w, r)
}
