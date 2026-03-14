// Package sessiontransport owns gRPC-facing session protobuf conversions.
//
// Keeping session record/gate/spotlight mapping here gives other transport
// features, such as communication, a clear session-owned import boundary instead
// of reaching into package-local helpers inside the root game transport package.
package sessiontransport
