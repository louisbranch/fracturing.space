// Package modulehandler provides a composable base for protected web module
// handlers.
//
// Protected modules (those mounted under /app/) share common handler
// infrastructure for user resolution, localization, page rendering, and error
// handling. This package extracts that shared scaffold so modules embed it
// rather than duplicating it.
//
// Start here when a protected route needs shared request-scoped transport
// behavior. The package should stay a thin adapter over principal/, pagerender,
// and weberror rather than growing feature policy of its own.
package modulehandler
