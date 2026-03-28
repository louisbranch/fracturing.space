// Package workflowwrite owns shared Daggerheart transport write-path helpers.
//
// The package keeps two concerns out of the root Daggerheart service package:
//   - executing one domain command through the shared gRPC write path with the
//     Daggerheart-specific error-normalization policy, and
//   - defining the shared command-metadata shapes consumed by sibling
//     transport packages that reuse that write path, and
//   - constructing the shared workflow runtime used by sibling workflow
//     transport packages.
package workflowwrite
