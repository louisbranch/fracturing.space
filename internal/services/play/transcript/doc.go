// Package transcript defines play-owned human transcript storage contracts.
//
// The package owns the stable seam between the play runtime and transcript
// adapters: session scope, append idempotency input, and history pagination
// rules. Storage backends should consume these request/query types rather than
// each reimplementing trimming, defaulting, and validation rules.
package transcript
