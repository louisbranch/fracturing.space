// Package principal owns request-scoped identity, viewer chrome, language
// resolution, and the shared request-state contracts used by the browser-facing
// web service.
//
// The root resolver delegates to smaller auth, account-profile, viewer, and
// language collaborators so request-scoped browser concerns do not accumulate
// behind one monolithic dependency bag again. Shared transport helpers embed
// the lightweight Base/PrincipalResolver contracts from this package instead of
// carrying a second request-state abstraction.
package principal
