package scene

import "github.com/louisbranch/fracturing.space/internal/services/game/domain/session"

// SpotlightType identifies who has the spotlight within a scene.
//
// Canonical definition lives in domain/session. This alias keeps scene-package
// callers and external references (storage, projection, transport) working
// without import changes.
type SpotlightType = session.SpotlightType

const (
	SpotlightTypeGM        = session.SpotlightTypeGM
	SpotlightTypeCharacter = session.SpotlightTypeCharacter
)

// NormalizeSpotlightType validates and normalizes a spotlight type value.
//
// Delegates to session.NormalizeSpotlightType which owns the canonical logic.
func NormalizeSpotlightType(value string) (SpotlightType, error) {
	return session.NormalizeSpotlightType(value)
}

// NormalizeGateType validates and normalizes a gate type value.
//
// Delegates to session.NormalizeGateType which owns the canonical logic.
func NormalizeGateType(value string) (string, error) {
	return session.NormalizeGateType(value)
}
