// Package validate provides transport-level request validation helpers
// for gRPC handlers.
//
// # Proto enum UNSPECIFIED semantics
//
// Proto enums use UNSPECIFIED differently depending on context:
//   - Required fields: UNSPECIFIED is rejected (e.g. GameSystem, DaggerheartCondition)
//   - Optional/patch fields: UNSPECIFIED means "no update" (e.g. Locale, ParticipantRole)
//
// Each handler documents its own UNSPECIFIED handling; this package does not
// enforce a blanket policy.
package validate
