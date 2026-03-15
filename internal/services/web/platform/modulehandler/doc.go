// Package modulehandler provides a composable base for protected web module
// handlers.
//
// Protected modules (those mounted under /app/) share common handler
// infrastructure for user resolution, localization, page rendering, and error
// handling. This package extracts that shared scaffold so modules embed it
// rather than duplicating it.
package modulehandler
