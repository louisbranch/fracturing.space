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
	CodeUserEmptyUsername   Code = "USER_EMPTY_USERNAME"
	CodeUserInvalidUsername Code = "USER_INVALID_USERNAME"

	// Invite errors
	CodeInviteEmptyCampaignID         Code = "INVITE_EMPTY_CAMPAIGN_ID"
	CodeInviteEmptyParticipantID      Code = "INVITE_EMPTY_PARTICIPANT_ID"
	CodeInviteRecipientAlreadyInvited Code = "INVITE_RECIPIENT_ALREADY_INVITED"
	CodeInviteRecipientUserMissing    Code = "INVITE_RECIPIENT_USER_MISSING"
	CodeInviteJoinGrantInvalid        Code = "INVITE_JOIN_GRANT_INVALID"
	CodeInviteJoinGrantExpired        Code = "INVITE_JOIN_GRANT_EXPIRED"
	CodeInviteJoinGrantMismatch       Code = "INVITE_JOIN_GRANT_MISMATCH"
	CodeInviteJoinGrantUsed           Code = "INVITE_JOIN_GRANT_USED"

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
	CodeOutcomeAlreadyApplied Code = "OUTCOME_ALREADY_APPLIED"
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
	CodeForkEmptyCampaignID Code = "FORK_EMPTY_CAMPAIGN_ID"

	// AI orchestration errors
	CodeAIOrchestrationInvalidInput          Code = "AI_ORCHESTRATION_INVALID_INPUT"
	CodeAIOrchestrationUnavailable           Code = "AI_ORCHESTRATION_UNAVAILABLE"
	CodeAIOrchestrationPromptBuildFailed     Code = "AI_ORCHESTRATION_PROMPT_BUILD_FAILED"
	CodeAIOrchestrationExecutionFailed       Code = "AI_ORCHESTRATION_EXECUTION_FAILED"
	CodeAIOrchestrationStepLimitExceeded     Code = "AI_ORCHESTRATION_STEP_LIMIT_EXCEEDED"
	CodeAIOrchestrationNarrationNotCommitted Code = "AI_ORCHESTRATION_NARRATION_NOT_COMMITTED"
	CodeAIOrchestrationEmptyOutput           Code = "AI_ORCHESTRATION_EMPTY_OUTPUT"
	CodeAIOrchestrationTimedOut              Code = "AI_ORCHESTRATION_TIMED_OUT"
	CodeAIOrchestrationCanceled              Code = "AI_ORCHESTRATION_CANCELED"
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
		CodeUserEmptyUsername,
		CodeUserInvalidUsername,
		CodeInviteEmptyCampaignID,
		CodeInviteEmptyParticipantID,
		CodeInviteRecipientUserMissing,
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
		CodeAIOrchestrationInvalidInput:
		return codes.InvalidArgument

	// FailedPrecondition - state doesn't allow operation
	case CodeCampaignInvalidStatusTransition,
		CodeCampaignStatusDisallowsOp,
		CodeActiveSessionExists,
		CodeOutcomeAlreadyApplied,
		CodeSnapshotInsufficientFear,
		CodeSnapshotGMFearExceedsCap,
		CodeDaggerheartInsufficientResource,
		CodeDaggerheartResourceAtCap,
		CodeInviteJoinGrantExpired,
		CodeInviteJoinGrantUsed,
		CodeAIOrchestrationUnavailable:
		return codes.FailedPrecondition

	// NotFound - resource doesn't exist
	case CodeNotFound:
		return codes.NotFound

	// AlreadyExists - unique resource constraint
	case CodeParticipantUserAlreadyClaimed,
		CodeInviteRecipientAlreadyInvited:
		return codes.AlreadyExists

	case CodeAIOrchestrationTimedOut:
		return codes.DeadlineExceeded

	case CodeAIOrchestrationCanceled:
		return codes.Canceled

	default:
		return codes.Internal
	}
}
