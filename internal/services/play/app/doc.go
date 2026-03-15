// Package app hosts the browser-facing play transport for the play service.
//
// The package owns HTTP and websocket request handling, play-shell rendering,
// authenticated browser request mapping, request-context resolution,
// route catalogs, interaction-route descriptors, and browser-state assembly
// from injected service dependencies. Runtime infrastructure creation belongs
// in internal/cmd/play so this package stays focused on transport
// orchestration, request/application seams, and route ownership that are
// easier to test in isolation.
package app
