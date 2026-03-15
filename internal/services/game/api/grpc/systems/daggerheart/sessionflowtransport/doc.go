// Package sessionflowtransport owns the high-level Daggerheart session gameplay
// flow handlers that compose lower-level roll, outcome, and damage operations.
//
// It keeps orchestration out of the root package while intentionally leaving
// the lower-level roll and write helpers rooted until a later phase can move
// them without mixing concerns.
package sessionflowtransport
