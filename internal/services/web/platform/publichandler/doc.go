// Package publichandler provides a shared base for unauthenticated web module
// handlers.
//
// It centralizes error handling, localization, and page rendering that would
// otherwise be duplicated across public modules.
//
// Start here when a public route needs shared page-shell behavior or needs to
// branch on signed-in viewer state without becoming a second principal
// resolver. This package should stay a thin transport adapter over principal/,
// pagerender, and weberror.
package publichandler
