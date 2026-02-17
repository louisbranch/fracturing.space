// Package server composes and runs the auth process boundary.
//
// It hosts the gRPC API plus optional OAuth HTTP endpoints that share the same
// SQLite store so user identity decisions are made from one source of truth.
package server
