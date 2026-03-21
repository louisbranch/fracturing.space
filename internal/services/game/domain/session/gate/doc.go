// Package gate implements the session gate workflow engine: workflow types,
// progress computation, and projection helpers.
//
// Gate types are consumed by projection and transport layers outside the
// session aggregate. The aggregate behavior (commands, events, state, fold,
// deciders) remains in the parent session package because it operates on
// session.State directly.
package gate
