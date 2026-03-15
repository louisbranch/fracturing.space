// Package validate provides transport-level request validation helpers
// for gRPC handlers.
//
// # Handler pipeline position
//
// validate runs first in every handler. It rejects malformed requests
// with codes.InvalidArgument before any domain state is loaded. This
// keeps domain logic free of shape/size concerns.
//
//	Request → validate (shape) → load state → domain checks → commandbuild → domainwrite → grpcerror
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
