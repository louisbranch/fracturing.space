// Package credential owns the credential domain model and lifecycle.
//
// A credential stores an encrypted provider API key for a specific user and
// provider. The lifecycle supports creation (with label/provider validation),
// listing, and revocation.
//
// SecretCiphertext carries the encrypted secret as a domain field so the
// service layer can pass it through without importing storage types.
//
// Domain types (Credential, Page) flow through all layers. The storage adapter
// scans directly into domain types; there are no separate storage record types.
package credential
