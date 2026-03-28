// Package grpcmeta provides shared gRPC metadata headers, context helpers, and
// request-correlation interceptors for internal services.
//
// This package is platform-owned because the header vocabulary is shared across
// AI, game, worker, web, and other transports. Service packages should depend
// on this shared surface instead of importing another service's transport tree.
package grpcmeta
