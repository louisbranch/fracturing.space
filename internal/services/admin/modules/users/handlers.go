package users

import "net/http"

// Service defines user handlers consumed by this module's routes.
type Service interface {
	HandleUsersPage(w http.ResponseWriter, r *http.Request)
	HandleUsersTable(w http.ResponseWriter, r *http.Request)
	HandleUserLookup(w http.ResponseWriter, r *http.Request)
	HandleUserDetail(w http.ResponseWriter, r *http.Request, userID string)
	HandleUserInvites(w http.ResponseWriter, r *http.Request, userID string)
}
