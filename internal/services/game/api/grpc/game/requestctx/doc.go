// Package requestctx provides request-metadata test helpers for the game gRPC
// transport. It owns synthetic incoming metadata contexts such as participant,
// user, and admin-override callers so store fakes and record fixtures do not
// need to carry transport-specific helpers.
package requestctx
