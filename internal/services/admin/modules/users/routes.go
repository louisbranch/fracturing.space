package users

import (
	"net/http"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/admin/routepath"
)

func newRoutes(service Service) *http.ServeMux {
	mux := http.NewServeMux()
	if service == nil {
		mux.HandleFunc(http.MethodGet+" "+routepath.AppUsers, http.NotFound)
		mux.HandleFunc(http.MethodGet+" "+routepath.UsersPrefix+"{$}", http.NotFound)
		return mux
	}
	mux.HandleFunc(http.MethodGet+" "+routepath.AppUsers, func(w http.ResponseWriter, r *http.Request) {
		if wantsRowsFragment(r) {
			service.HandleUsersTable(w, r)
			return
		}
		service.HandleUsersPage(w, r)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.UsersPrefix+"{$}", func(w http.ResponseWriter, r *http.Request) {
		if wantsRowsFragment(r) {
			service.HandleUsersTable(w, r)
			return
		}
		service.HandleUsersPage(w, r)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.UsersLookup, service.HandleUserLookup)
	mux.HandleFunc(http.MethodGet+" "+routepath.UsersCreate, http.NotFound)
	mux.HandleFunc(http.MethodGet+" "+routepath.AppUserInvitesPattern, func(w http.ResponseWriter, r *http.Request) {
		userID := strings.TrimSpace(r.PathValue("userID"))
		if userID == "" || isReservedUserID(userID) {
			http.NotFound(w, r)
			return
		}
		service.HandleUserInvites(w, r, userID)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.AppUserPattern, func(w http.ResponseWriter, r *http.Request) {
		userID := strings.TrimSpace(r.PathValue("userID"))
		if userID == "" || isReservedUserID(userID) {
			http.NotFound(w, r)
			return
		}
		service.HandleUserDetail(w, r, userID)
	})
	mux.HandleFunc(http.MethodGet+" "+routepath.UsersPrefix+"{userID}/{rest...}", http.NotFound)
	return mux
}

func isReservedUserID(userID string) bool {
	switch strings.ToLower(strings.TrimSpace(userID)) {
	case "lookup", "create", "magic-link", "table":
		return true
	default:
		return false
	}
}

func wantsRowsFragment(r *http.Request) bool {
	if r == nil || r.URL == nil {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(r.URL.Query().Get(routepath.FragmentQueryKey)), routepath.FragmentRows)
}
