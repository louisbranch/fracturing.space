// Package app owns publicauth domain contracts and orchestration.
//
// The package stays split by public surface capability:
//   - page helpers own static shell/signup/login page state,
//   - session helpers own login/signup/recovery orchestration,
//   - passkey helpers own registration ceremony behavior,
//   - recovery helpers own recovery-start validation.
//
// Keep those seams narrow so the root publicauth module can mount multiple
// route owners without rebuilding one transport-wide service sink.
package app
