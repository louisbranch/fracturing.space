// Package orchestration owns the runtime policy for campaign AI turns.
//
// It is responsible for opening game sessions with fixed authority, collecting
// a typed session brief from authoritative resources, rendering the
// provider-facing prompt, enforcing tool and runtime safety policies, and
// executing the provider/tool loop for one GM turn.
package orchestration
