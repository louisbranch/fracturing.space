// Package templates contains shared web templ layout primitives and page
// components.
//
// Keep truly cross-area shell pieces here (layout, error, image, datetime, and
// similar helpers). Module-owned full-page templates may reuse exported shell
// primitives from this package (for example layout language normalization and
// shared head includes) instead of re-implementing app-shell behavior. When
// one area's page set grows into an ownership hotspot, move those templates
// under the owning module/package instead of expanding a long-lived cross-area
// page bucket.
package templates
