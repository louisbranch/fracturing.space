// Package command defines the canonical command envelope, registration, and
// decision types used by the game domain write path.
//
// Commands represent intent. They are validated and normalized before deciders
// run. Deciders return Decisions containing either accepted events or
// rejections.
package command
