// Package provideroauth owns provider-side OAuth handshake contracts used by
// provider-grant connect, exchange, refresh, and optional upstream revoke
// workflows.
//
// This package exists so the generic `provider/` package can stay focused on
// provider identity plus invocation/model-listing seams, while provider-grant
// flows depend on a narrower OAuth-specific support boundary. It also owns
// small shared OAuth vocabulary such as scope normalization and structured
// token-payload encoding so provider adapters do not depend on provider-grant
// domain helpers.
package provideroauth
