// Package damagetransport owns the Daggerheart damage-application transport
// surface.
//
// It handles character damage and adversary damage behind explicit read-store
// contracts and a system-command callback so the root Daggerheart package can
// stay a thin facade over this gameplay mutation slice.
package damagetransport
