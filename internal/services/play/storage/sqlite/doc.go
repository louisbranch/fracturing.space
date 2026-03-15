// Package sqlite provides SQLite-backed transcript storage for the play service.
//
// The adapter consumes the canonical transcript request/query types and retries
// retryable SQLite write conflicts so concurrent chat writers still produce one
// gapless per-session sequence.
package sqlite
