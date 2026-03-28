// Package platform provides shared transport and rendering utilities for the
// web service. Sub-packages are grouped by concern:
//
// # Handler Bases
//
// These are the primary entry points for module handlers:
//
//   - modulehandler: authenticated handler base with user resolution, page
//     rendering, mutation flash helpers, and route-param extraction.
//   - publichandler: unauthenticated handler base with optional viewer resolution
//     and public page rendering.
//
// # Rendering
//
//   - pagerender: full-page and HTMX-aware rendering for module and public pages.
//   - weberror: localized error response rendering for both app-shell and public
//     chrome contexts.
//
// # Transport Utilities
//
//   - httpx: HTTP request/response helpers (method gating, JSON, redirects, route
//     params, form parsing).
//   - requestmeta: scheme resolution, same-origin policy, and proxy header trust.
//   - grpcpaging: gRPC pagination helpers for list views.
//
// # Authentication Plumbing
//
//   - sessioncookie: session cookie read/write with scheme-aware policy.
//   - authctx: gRPC auth context helpers for downstream calls.
//   - userid: user ID normalization and validation.
//
// # User Experience
//
//   - flash: one-time flash notices persisted in cookies across redirects.
//   - i18n: language tag resolution, localizer construction, and language cookie
//     synchronization.
//
// # Cross-Cutting
//
//   - errors: web-specific error types and gRPC-to-HTTP status mapping. Re-exports
//     shared httperrors types for convenience.
//   - dashboardsync: cross-module dashboard freshness coordination after mutations.
//   - observability: request logging and tracing middleware.
package platform
