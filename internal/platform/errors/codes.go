// Package errors provides structured error handling with i18n support.
package errors

import "google.golang.org/grpc/codes"

// Code is a machine-readable error code.
type Code string

const (
	// CodeUnknown represents an unknown error.
	CodeUnknown Code = "UNKNOWN"

	// Campaign errors
	CodeCampaignNameEmpty               Code = "CAMPAIGN_NAME_EMPTY"
	CodeCampaignInvalidGmMode           Code = "CAMPAIGN_INVALID_GM_MODE"
	CodeCampaignInvalidGameSystem       Code = "CAMPAIGN_INVALID_GAME_SYSTEM"
	CodeCampaignInvalidStatusTransition Code = "CAMPAIGN_INVALID_STATUS_TRANSITION"
	CodeCampaignStatusDisallowsOp       Code = "CAMPAIGN_STATUS_DISALLOWS_OPERATION"
	CodeCampaignCreatorUserMissing      Code = "CAMPAIGN_CREATOR_USER_MISSING"

	// Participant errors
	CodeParticipantEmptyDisplayName   Code = "PARTICIPANT_EMPTY_DISPLAY_NAME"
	CodeParticipantInvalidRole        Code = "PARTICIPANT_INVALID_ROLE"
	CodeParticipantEmptyCampaignID    Code = "PARTICIPANT_EMPTY_CAMPAIGN_ID"
	CodeParticipantUserAlreadyClaimed Code = "PARTICIPANT_USER_ALREADY_CLAIMED"

	// User errors
	CodeUserEmptyDisplayName Code = "USER_EMPTY_DISPLAY_NAME"

	// Invite errors
	CodeInviteEmptyCampaignID    Code = "INVITE_EMPTY_CAMPAIGN_ID"
	CodeInviteEmptyParticipantID Code = "INVITE_EMPTY_PARTICIPANT_ID"
	CodeInviteJoinGrantInvalid   Code = "INVITE_JOIN_GRANT_INVALID"
	CodeInviteJoinGrantExpired   Code = "INVITE_JOIN_GRANT_EXPIRED"
	CodeInviteJoinGrantMismatch  Code = "INVITE_JOIN_GRANT_MISMATCH"
	CodeInviteJoinGrantUsed      Code = "INVITE_JOIN_GRANT_USED"

	// Session errors
	CodeSessionEmptyCampaignID Code = "SESSION_EMPTY_CAMPAIGN_ID"

	// Character errors
	CodeCharacterEmptyCampaignID  Code = "CHARACTER_EMPTY_CAMPAIGN_ID"
	CodeCharacterEmptyName        Code = "CHARACTER_EMPTY_NAME"
	CodeCharacterInvalidKind      Code = "CHARACTER_INVALID_KIND"
	CodeCharacterInvalidProfileHp Code = "CHARACTER_INVALID_PROFILE_HP"

	// Snapshot errors
	CodeSnapshotInvalidHp        Code = "SNAPSHOT_INVALID_HP"
	CodeSnapshotInvalidGMFear    Code = "SNAPSHOT_INVALID_GM_FEAR_AMOUNT"
	CodeSnapshotInsufficientFear Code = "SNAPSHOT_INSUFFICIENT_GM_FEAR"
	CodeSnapshotGMFearExceedsCap Code = "SNAPSHOT_GM_FEAR_EXCEEDS_CAP"

	// Outcome errors
	CodeOutcomeAlreadyApplied    Code = "OUTCOME_ALREADY_APPLIED"
	CodeOutcomeCharacterNotFound Code = "OUTCOME_CHARACTER_NOT_FOUND"
	CodeOutcomeGMFearInvalid     Code = "OUTCOME_GM_FEAR_INVALID"

	// Storage errors
	CodeNotFound            Code = "NOT_FOUND"
	CodeActiveSessionExists Code = "ACTIVE_SESSION_EXISTS"

	// Dice/mechanics errors
	CodeDiceMissing     Code = "DICE_MISSING"
	CodeDiceInvalidSpec Code = "DICE_INVALID_SPEC"

	// Random/seed errors
	CodeSeedOutOfRange Code = "SEED_OUT_OF_RANGE"

	// Daggerheart-specific errors
	CodeDaggerheartInvalidDifficulty    Code = "DAGGERHEART_INVALID_DIFFICULTY"
	CodeDaggerheartInvalidDualityDie    Code = "DAGGERHEART_INVALID_DUALITY_DIE"
	CodeDaggerheartInvalidLevel         Code = "DAGGERHEART_INVALID_LEVEL"
	CodeDaggerheartInvalidTraitValue    Code = "DAGGERHEART_INVALID_TRAIT_VALUE"
	CodeDaggerheartInvalidStressMax     Code = "DAGGERHEART_INVALID_STRESS_MAX"
	CodeDaggerheartInvalidHpMax         Code = "DAGGERHEART_INVALID_HP_MAX"
	CodeDaggerheartInvalidHp            Code = "DAGGERHEART_INVALID_HP"
	CodeDaggerheartInvalidEvasion       Code = "DAGGERHEART_INVALID_EVASION"
	CodeDaggerheartInvalidThresholds    Code = "DAGGERHEART_INVALID_THRESHOLDS"
	CodeDaggerheartInvalidProficiency   Code = "DAGGERHEART_INVALID_PROFICIENCY"
	CodeDaggerheartInvalidArmorMax      Code = "DAGGERHEART_INVALID_ARMOR_MAX"
	CodeDaggerheartInvalidArmorScore    Code = "DAGGERHEART_INVALID_ARMOR_SCORE"
	CodeDaggerheartInvalidExperience    Code = "DAGGERHEART_INVALID_EXPERIENCE"
	CodeDaggerheartInvalidRestSequence  Code = "DAGGERHEART_INVALID_REST_SEQUENCE"
	CodeDaggerheartUnknownResource      Code = "DAGGERHEART_UNKNOWN_RESOURCE"
	CodeDaggerheartInsufficientResource Code = "DAGGERHEART_INSUFFICIENT_RESOURCE"
	CodeDaggerheartResourceAtCap        Code = "DAGGERHEART_RESOURCE_AT_CAP"

	// Fork errors
	CodeForkEmptyCampaignID  Code = "FORK_EMPTY_CAMPAIGN_ID"
	CodeForkInvalidForkPoint Code = "FORK_INVALID_FORK_POINT"
	CodeForkPointInFuture    Code = "FORK_POINT_IN_FUTURE"
)

// GRPCCode maps domain codes to gRPC status codes.
func (c Code) GRPCCode() codes.Code {
	switch c {
	// InvalidArgument - validation failures, bad input
	case CodeCampaignNameEmpty,
		CodeCampaignInvalidGmMode,
		CodeCampaignInvalidGameSystem,
		CodeCampaignCreatorUserMissing,
		CodeParticipantEmptyDisplayName,
		CodeParticipantInvalidRole,
		CodeParticipantEmptyCampaignID,
		CodeUserEmptyDisplayName,
		CodeInviteEmptyCampaignID,
		CodeInviteEmptyParticipantID,
		CodeInviteJoinGrantInvalid,
		CodeInviteJoinGrantMismatch,
		CodeSessionEmptyCampaignID,
		CodeCharacterEmptyCampaignID,
		CodeCharacterEmptyName,
		CodeCharacterInvalidKind,
		CodeCharacterInvalidProfileHp,
		CodeSnapshotInvalidHp,
		CodeSnapshotInvalidGMFear,
		CodeDiceMissing,
		CodeDiceInvalidSpec,
		CodeSeedOutOfRange,
		CodeDaggerheartInvalidDifficulty,
		CodeDaggerheartInvalidDualityDie,
		CodeDaggerheartInvalidLevel,
		CodeDaggerheartInvalidTraitValue,
		CodeDaggerheartInvalidStressMax,
		CodeDaggerheartInvalidHpMax,
		CodeDaggerheartInvalidHp,
		CodeDaggerheartInvalidEvasion,
		CodeDaggerheartInvalidThresholds,
		CodeDaggerheartInvalidProficiency,
		CodeDaggerheartInvalidArmorMax,
		CodeDaggerheartInvalidArmorScore,
		CodeDaggerheartInvalidExperience,
		CodeDaggerheartInvalidRestSequence,
		CodeDaggerheartUnknownResource,
		CodeForkEmptyCampaignID,
		CodeForkInvalidForkPoint:
		return codes.InvalidArgument

	// FailedPrecondition - state doesn't allow operation
	case CodeCampaignInvalidStatusTransition,
		CodeCampaignStatusDisallowsOp,
		CodeActiveSessionExists,
		CodeOutcomeAlreadyApplied,
		CodeOutcomeGMFearInvalid,
		CodeSnapshotInsufficientFear,
		CodeSnapshotGMFearExceedsCap,
		CodeDaggerheartInsufficientResource,
		CodeDaggerheartResourceAtCap,
		CodeInviteJoinGrantExpired,
		CodeInviteJoinGrantUsed,
		CodeForkPointInFuture:
		return codes.FailedPrecondition

	// NotFound - resource doesn't exist
	case CodeNotFound,
		CodeOutcomeCharacterNotFound:
		return codes.NotFound

	// AlreadyExists - unique resource constraint
	case CodeParticipantUserAlreadyClaimed:
		return codes.AlreadyExists

	default:
		return codes.Internal
	}
}
