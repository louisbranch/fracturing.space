// Package principal owns request-scoped identity, viewer chrome, and language
// resolution for the browser-facing web service.
//
// The root resolver delegates to smaller auth, account-profile, viewer, and
// language collaborators so request-scoped browser concerns do not accumulate
// behind one monolithic dependency bag again.
package principal
