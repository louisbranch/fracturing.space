package daggerheart

import "strings"

// GMMoveKind identifies the supported direct GM fear-buy action.
type GMMoveKind string

const (
	GMMoveKindUnspecified      GMMoveKind = ""
	GMMoveKindInterruptAndMove GMMoveKind = "interrupt_and_move"
	GMMoveKindAdditionalMove   GMMoveKind = "additional_move"
)

// GMMoveShape identifies the narrative shape of a GM move.
type GMMoveShape string

const (
	GMMoveShapeUnspecified            GMMoveShape = ""
	GMMoveShapeShowWorldReaction      GMMoveShape = "show_world_reaction"
	GMMoveShapeRevealDanger           GMMoveShape = "reveal_danger"
	GMMoveShapeForceSplit             GMMoveShape = "force_split"
	GMMoveShapeMarkStress             GMMoveShape = "mark_stress"
	GMMoveShapeShiftEnvironment       GMMoveShape = "shift_environment"
	GMMoveShapeSpotlightAdversary     GMMoveShape = "spotlight_adversary"
	GMMoveShapeCaptureImportantTarget GMMoveShape = "capture_important_target"
	GMMoveShapeCustom                 GMMoveShape = "custom"
)

func NormalizeGMMoveKind(value string) (GMMoveKind, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(GMMoveKindInterruptAndMove):
		return GMMoveKindInterruptAndMove, true
	case string(GMMoveKindAdditionalMove):
		return GMMoveKindAdditionalMove, true
	default:
		return GMMoveKindUnspecified, false
	}
}

func NormalizeGMMoveShape(value string) (GMMoveShape, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(GMMoveShapeShowWorldReaction):
		return GMMoveShapeShowWorldReaction, true
	case string(GMMoveShapeRevealDanger):
		return GMMoveShapeRevealDanger, true
	case string(GMMoveShapeForceSplit):
		return GMMoveShapeForceSplit, true
	case string(GMMoveShapeMarkStress):
		return GMMoveShapeMarkStress, true
	case string(GMMoveShapeShiftEnvironment):
		return GMMoveShapeShiftEnvironment, true
	case string(GMMoveShapeSpotlightAdversary):
		return GMMoveShapeSpotlightAdversary, true
	case string(GMMoveShapeCaptureImportantTarget):
		return GMMoveShapeCaptureImportantTarget, true
	case string(GMMoveShapeCustom):
		return GMMoveShapeCustom, true
	default:
		return GMMoveShapeUnspecified, false
	}
}

// GMMoveTargetType identifies the supported Fear-spend target families.
type GMMoveTargetType string

const (
	GMMoveTargetTypeUnspecified         GMMoveTargetType = ""
	GMMoveTargetTypeDirectMove          GMMoveTargetType = "direct_move"
	GMMoveTargetTypeAdversaryFeature    GMMoveTargetType = "adversary_feature"
	GMMoveTargetTypeEnvironmentFeature  GMMoveTargetType = "environment_feature"
	GMMoveTargetTypeAdversaryExperience GMMoveTargetType = "adversary_experience"
)

func NormalizeGMMoveTargetType(value string) (GMMoveTargetType, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(GMMoveTargetTypeDirectMove):
		return GMMoveTargetTypeDirectMove, true
	case string(GMMoveTargetTypeAdversaryFeature):
		return GMMoveTargetTypeAdversaryFeature, true
	case string(GMMoveTargetTypeEnvironmentFeature):
		return GMMoveTargetTypeEnvironmentFeature, true
	case string(GMMoveTargetTypeAdversaryExperience):
		return GMMoveTargetTypeAdversaryExperience, true
	default:
		return GMMoveTargetTypeUnspecified, false
	}
}
