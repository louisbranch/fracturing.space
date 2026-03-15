// Package protocol defines the browser-facing HTTP and websocket contracts for
// the play service.
//
// The app package owns runtime orchestration, but it should import these types
// instead of redefining browser payloads locally. That keeps the play browser
// contract visible at one package boundary and makes transport drift easier to
// detect during refactors.
package protocol
