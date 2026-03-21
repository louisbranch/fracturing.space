// Package settings owns authenticated user settings transport routes.
//
// Start here when changing account profile, locale, security, or AI settings
// pages. The root package owns route registration and handler assembly; area
// orchestration lives in `settings/app`, and backend protocol mapping lives in
// `settings/gateway`.
//
// Composition should stay area-owned:
//   - `composition.go` builds account and AI service graphs,
//   - `module.go` mounts routes with ready handler services,
//   - shared protected-request concerns stay in `modulehandler.Base`.
package settings
