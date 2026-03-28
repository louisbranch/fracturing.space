// Package providercatalog owns provider runtime bundle registration and
// capability lookup for the AI service.
//
// Provider identity lives in `provider/`, while this package answers the
// runtime question of which providers are actually wired into the current
// process and which capabilities each registered provider exposes.
package providercatalog
