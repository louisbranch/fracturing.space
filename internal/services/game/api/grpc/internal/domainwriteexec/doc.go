// Package domainwriteexec orchestrates shared execute-and-apply flows
// for gRPC domain writes.
//
// It composes domainwrite.Runtime with grpcerror defaults so individual
// gRPC handlers call a single function instead of repeating option
// normalization. This is the primary entry point for handler write paths.
package domainwriteexec
