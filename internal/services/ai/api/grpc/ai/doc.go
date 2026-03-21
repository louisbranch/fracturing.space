// Package ai provides the AI service gRPC transport layer.
//
// The package is organized around dedicated handler roots plus a small set of
// shared seams for auth-token resolution, shared-access policy, and audit writes
// so contributors do not have to recover behavior from one catch-all server type.
package ai
