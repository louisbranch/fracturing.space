package users

import (
	"net/http"
	"strings"

	mod "github.com/louisbranch/fracturing.space/internal/services/admin/module"
	sharedpath "github.com/louisbranch/fracturing.space/internal/services/admin/module/sharedpath"
	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
	sharedroute "github.com/louisbranch/fracturing.space/internal/services/shared/route"
)

// Service defines user routes consumed by this module.
type Service interface {
	HandleUsersPage(w http.ResponseWriter, r *http.Request)
	HandleUsersTable(w http.ResponseWriter, r *http.Request)
	HandleUserLookup(w http.ResponseWriter, r *http.Request)
	HandleUserDetail(w http.ResponseWriter, r *http.Request, userID string)
	HandleUserInvites(w http.ResponseWriter, r *http.Request, userID string)
}

// Module provides users routes.
type Module struct {
	service Service
}

// New returns a users module.
func New(service Service) Module { return Module{service: service} }

// ID returns a stable module identifier.
func (Module) ID() string { return "users" }

// Mount wires users routes.
func (m Module) Mount() (mod.Mount, error) {
	mux := http.NewServeMux()
	if m.service == nil {
		mux.HandleFunc(routepath.UsersPrefix, http.NotFound)
		return mod.Mount{Prefix: routepath.UsersPrefix, Handler: mux}, nil
	}
	mux.HandleFunc(routepath.Users, m.service.HandleUsersPage)
	mux.HandleFunc(routepath.UsersRows, m.service.HandleUsersTable)
	mux.HandleFunc(routepath.UsersLookup, m.service.HandleUserLookup)
	mux.Handle(routepath.UsersCreate, http.NotFoundHandler())
	mux.HandleFunc(routepath.UsersPrefix, func(w http.ResponseWriter, r *http.Request) {
		handleUserPath(w, r, m.service)
	})
	return mod.Mount{Prefix: routepath.UsersPrefix, Handler: mux}, nil
}

func handleUserPath(w http.ResponseWriter, r *http.Request, service Service) {
	if service == nil {
		http.NotFound(w, r)
		return
	}
	if sharedroute.RedirectTrailingSlash(w, r) {
		return
	}
	path := strings.TrimPrefix(r.URL.Path, routepath.UsersPrefix)
	parts := sharedpath.SplitPathParts(path)
	if len(parts) == 1 {
		if strings.EqualFold(parts[0], "magic-link") || strings.EqualFold(parts[0], "table") {
			http.NotFound(w, r)
			return
		}
		service.HandleUserDetail(w, r, parts[0])
		return
	}
	if len(parts) == 2 && parts[1] == "invites" {
		service.HandleUserInvites(w, r, parts[0])
		return
	}
	http.NotFound(w, r)
}
