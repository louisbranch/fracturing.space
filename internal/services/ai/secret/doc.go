// Package secret defines the encryption seam for AI service secrets.
//
// The Sealer interface provides Seal/Open operations used by the service layer
// to encrypt credential API keys and provider-grant OAuth tokens before
// persistence. Concrete implementations live outside this package.
package secret
