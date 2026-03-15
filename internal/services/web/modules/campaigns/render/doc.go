// Package render owns handwritten campaign render seams that sit in front of
// generated templ output.
//
// Contributors should start here, not in generated `*_templ.go` files:
//   - exported wrapper functions such as OverviewFragment and SessionsFragment
//     are the stable section entrypoints,
//   - render-owned view types keep campaign detail and creation markup local to
//     this area,
//   - shared shell primitives such as images, timestamps, and translation stay
//     delegated to `internal/services/web/templates`.
//
// This package should keep section ownership explicit and template-facing
// contracts narrow. Feature routing or workflow orchestration belongs in the
// parent campaigns package or `campaigns/workflow`, not here.
package render
