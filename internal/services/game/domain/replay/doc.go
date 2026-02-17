// Package replay provides deterministic state reconstruction from the event log.
//
// Every write-path decision can be re-executed from history. Replay is the
// guarantee that aggregate state is deterministic and portable across process
// restarts and projection rebuilds.
package replay
