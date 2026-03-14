// Package publicauth owns unauthenticated shell, auth, and passkey routes.
//
// The root package is the transport owner for the public auth surface.
// Area-local orchestration lives in `publicauth/app`, and transport depends on
// narrower page, session, passkey, and recovery seams so route ownership stays
// explicit without reintroducing wrapper packages.
package publicauth
