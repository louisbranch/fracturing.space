// Package campaigns owns authenticated campaign workspace transport routes.
//
// The root package is the transport owner for route registration, request
// parsing, and render assembly. Area-local orchestration and gateway contracts
// live under `campaigns/app` and `campaigns/gateway`; transport depends on
// narrow app-facing service groups rather than treating the app layer as one
// monolithic seam.
package campaigns
