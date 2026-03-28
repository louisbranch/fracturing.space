// Package sessions owns campaign session transport routes, including the play
// launcher handoff.
//
// It consumes shared workspace-shell support from `campaigns/detail` and keeps
// session-specific view mapping, form parsing, and play launch redirects local
// to the session surface.
package sessions
