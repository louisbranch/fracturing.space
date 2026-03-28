// Package testclients provides shared narrow RPC client doubles for game
// transport tests. These fakes model only the auth and social calls the game
// transport actually consumes, without broadening the main `gametest`
// store-fixture package or mirroring whole generated client surfaces.
package testclients
