// Package contenttransport owns Daggerheart content and catalog read handlers.
//
// The root Daggerheart gRPC package keeps the public service constructors and
// registration surface stable. This package owns the actual content transport
// implementation: request validation, pagination, localization, descriptor
// tables, protobuf mapping, and asset-map assembly.
package contenttransport
