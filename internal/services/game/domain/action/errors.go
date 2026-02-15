package action

import apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"

var (
	// ErrOutcomeAlreadyApplied indicates the outcome was already applied for a roll.
	ErrOutcomeAlreadyApplied = apperrors.New(apperrors.CodeOutcomeAlreadyApplied, "outcome already applied")
	// ErrOutcomeCharacterNotFound indicates a character state is missing.
	ErrOutcomeCharacterNotFound = apperrors.New(apperrors.CodeOutcomeCharacterNotFound, "character state not found")
	// ErrOutcomeGMFearInvalid indicates a GM fear mutation is invalid.
	ErrOutcomeGMFearInvalid = apperrors.New(apperrors.CodeOutcomeGMFearInvalid, "gm fear update invalid")
)

const (
	// OutcomeFieldHope indicates a hope mutation.
	OutcomeFieldHope = "hope"
	// OutcomeFieldStress indicates a stress mutation.
	OutcomeFieldStress = "stress"
	// OutcomeFieldGMFear indicates a GM fear mutation.
	OutcomeFieldGMFear = "gm_fear"
)
