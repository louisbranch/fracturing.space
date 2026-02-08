// Package id provides utilities for generating URL-safe identifiers.
//
// Identifiers are generated using UUIDv4 bytes encoded as base32 (RFC 4648)
// with no padding. The resulting strings are 26 characters long, lowercase,
// and safe for use in URLs and file paths.
package id
