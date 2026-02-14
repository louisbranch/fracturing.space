package i18n

// Error codes must match the codes defined in internal/platform/errors/codes.go.
// These are duplicated as strings to avoid an import cycle.
const (
	CodeCampaignNameEmpty               = "CAMPAIGN_NAME_EMPTY"
	CodeCampaignInvalidGmMode           = "CAMPAIGN_INVALID_GM_MODE"
	CodeCampaignInvalidGameSystem       = "CAMPAIGN_INVALID_GAME_SYSTEM"
	CodeCampaignInvalidStatusTransition = "CAMPAIGN_INVALID_STATUS_TRANSITION"
	CodeCampaignStatusDisallowsOp       = "CAMPAIGN_STATUS_DISALLOWS_OPERATION"
	CodeCampaignCreatorUserMissing      = "CAMPAIGN_CREATOR_USER_MISSING"
	CodeParticipantEmptyDisplayName     = "PARTICIPANT_EMPTY_DISPLAY_NAME"
	CodeParticipantInvalidRole          = "PARTICIPANT_INVALID_ROLE"
	CodeParticipantEmptyCampaignID      = "PARTICIPANT_EMPTY_CAMPAIGN_ID"
	CodeParticipantUserAlreadyClaimed   = "PARTICIPANT_USER_ALREADY_CLAIMED"
	CodeUserEmptyDisplayName            = "USER_EMPTY_DISPLAY_NAME"
	CodeInviteEmptyCampaignID           = "INVITE_EMPTY_CAMPAIGN_ID"
	CodeInviteEmptyParticipantID        = "INVITE_EMPTY_PARTICIPANT_ID"
	CodeInviteRecipientUserMissing      = "INVITE_RECIPIENT_USER_MISSING"
	CodeInviteJoinGrantInvalid          = "INVITE_JOIN_GRANT_INVALID"
	CodeInviteJoinGrantExpired          = "INVITE_JOIN_GRANT_EXPIRED"
	CodeInviteJoinGrantMismatch         = "INVITE_JOIN_GRANT_MISMATCH"
	CodeInviteJoinGrantUsed             = "INVITE_JOIN_GRANT_USED"
	CodeSessionEmptyCampaignID          = "SESSION_EMPTY_CAMPAIGN_ID"
	CodeCharacterEmptyCampaignID        = "CHARACTER_EMPTY_CAMPAIGN_ID"
	CodeCharacterEmptyName              = "CHARACTER_EMPTY_NAME"
	CodeCharacterInvalidKind            = "CHARACTER_INVALID_KIND"
	CodeCharacterInvalidProfileHp       = "CHARACTER_INVALID_PROFILE_HP"
	CodeSnapshotInvalidHp               = "SNAPSHOT_INVALID_HP"
	CodeSnapshotInvalidGMFear           = "SNAPSHOT_INVALID_GM_FEAR_AMOUNT"
	CodeSnapshotInsufficientFear        = "SNAPSHOT_INSUFFICIENT_GM_FEAR"
	CodeSnapshotGMFearExceedsCap        = "SNAPSHOT_GM_FEAR_EXCEEDS_CAP"
	CodeOutcomeAlreadyApplied           = "OUTCOME_ALREADY_APPLIED"
	CodeOutcomeCharacterNotFound        = "OUTCOME_CHARACTER_NOT_FOUND"
	CodeOutcomeGMFearInvalid            = "OUTCOME_GM_FEAR_INVALID"
	CodeNotFound                        = "NOT_FOUND"
	CodeActiveSessionExists             = "ACTIVE_SESSION_EXISTS"
	CodeDiceMissing                     = "DICE_MISSING"
	CodeDiceInvalidSpec                 = "DICE_INVALID_SPEC"
	CodeSeedOutOfRange                  = "SEED_OUT_OF_RANGE"
	CodeDaggerheartInvalidDifficulty    = "DAGGERHEART_INVALID_DIFFICULTY"
	CodeDaggerheartInvalidDualityDie    = "DAGGERHEART_INVALID_DUALITY_DIE"
	CodeDaggerheartInvalidLevel         = "DAGGERHEART_INVALID_LEVEL"
	CodeDaggerheartInvalidTraitValue    = "DAGGERHEART_INVALID_TRAIT_VALUE"
	CodeDaggerheartInvalidStressMax     = "DAGGERHEART_INVALID_STRESS_MAX"
	CodeDaggerheartInvalidHpMax         = "DAGGERHEART_INVALID_HP_MAX"
	CodeDaggerheartInvalidHp            = "DAGGERHEART_INVALID_HP"
	CodeDaggerheartInvalidEvasion       = "DAGGERHEART_INVALID_EVASION"
	CodeDaggerheartInvalidThresholds    = "DAGGERHEART_INVALID_THRESHOLDS"
	CodeDaggerheartInvalidProficiency   = "DAGGERHEART_INVALID_PROFICIENCY"
	CodeDaggerheartInvalidArmorMax      = "DAGGERHEART_INVALID_ARMOR_MAX"
	CodeDaggerheartInvalidArmorScore    = "DAGGERHEART_INVALID_ARMOR_SCORE"
	CodeDaggerheartInvalidExperience    = "DAGGERHEART_INVALID_EXPERIENCE"
	CodeDaggerheartInvalidRestSequence  = "DAGGERHEART_INVALID_REST_SEQUENCE"
	CodeDaggerheartUnknownResource      = "DAGGERHEART_UNKNOWN_RESOURCE"
	CodeDaggerheartInsufficientResource = "DAGGERHEART_INSUFFICIENT_RESOURCE"
	CodeDaggerheartResourceAtCap        = "DAGGERHEART_RESOURCE_AT_CAP"
	CodeForkEmptyCampaignID             = "FORK_EMPTY_CAMPAIGN_ID"
	CodeForkInvalidForkPoint            = "FORK_INVALID_FORK_POINT"
	CodeForkPointInFuture               = "FORK_POINT_IN_FUTURE"
)

var enUSCatalog = &Catalog{
	locale: "en-US",
	messages: map[Code]string{
		// Campaign errors
		CodeCampaignNameEmpty:               "Campaign name cannot be empty",
		CodeCampaignInvalidGmMode:           "Invalid GM mode specified",
		CodeCampaignInvalidGameSystem:       "Invalid game system specified",
		CodeCampaignInvalidStatusTransition: "Cannot transition campaign from {{.FromStatus}} to {{.ToStatus}}",
		CodeCampaignStatusDisallowsOp:       "Campaign status {{.Status}} does not allow {{.Operation}}",
		CodeCampaignCreatorUserMissing:      "Creator user is required to create a campaign",

		// Participant errors
		CodeParticipantEmptyDisplayName:   "Participant display name cannot be empty",
		CodeParticipantInvalidRole:        "Invalid participant role specified",
		CodeParticipantEmptyCampaignID:    "Campaign ID is required for participant",
		CodeParticipantUserAlreadyClaimed: "User is already assigned to a participant in this campaign",

		// User errors
		CodeUserEmptyDisplayName: "User display name cannot be empty",

		// Invite errors
		CodeInviteEmptyCampaignID:      "Campaign ID is required for invite",
		CodeInviteEmptyParticipantID:   "Participant ID is required for invite",
		CodeInviteRecipientUserMissing: "Invite recipient user was not found",
		CodeInviteJoinGrantInvalid:     "Join grant is invalid",
		CodeInviteJoinGrantExpired:     "Join grant has expired",
		CodeInviteJoinGrantMismatch:    "Join grant {{.Field}} does not match",
		CodeInviteJoinGrantUsed:        "Join grant has already been used",

		// Session errors
		CodeSessionEmptyCampaignID: "Campaign ID is required for session",

		// Character errors
		CodeCharacterEmptyCampaignID:  "Campaign ID is required for character",
		CodeCharacterEmptyName:        "Character name cannot be empty",
		CodeCharacterInvalidKind:      "Invalid character kind specified",
		CodeCharacterInvalidProfileHp: "HP maximum must be at least 1",

		// Snapshot errors
		CodeSnapshotInvalidHp:        "HP {{.HP}} exceeds maximum {{.HPMax}}",
		CodeSnapshotInvalidGMFear:    "GM Fear amount must be greater than zero",
		CodeSnapshotInsufficientFear: "Insufficient GM Fear to spend",
		CodeSnapshotGMFearExceedsCap: "GM Fear would exceed maximum cap",

		// Outcome errors
		CodeOutcomeAlreadyApplied:    "Outcome has already been applied for this roll",
		CodeOutcomeCharacterNotFound: "Character state not found for outcome application",
		CodeOutcomeGMFearInvalid:     "GM Fear update is invalid",

		// Storage errors
		CodeNotFound:            "The requested resource was not found",
		CodeActiveSessionExists: "An active session already exists for this campaign",

		// Dice/mechanics errors
		CodeDiceMissing:     "At least one die must be specified",
		CodeDiceInvalidSpec: "Dice must have positive sides and count",

		// Random/seed errors
		CodeSeedOutOfRange: "Random seed is out of valid range",

		// Daggerheart-specific errors
		CodeDaggerheartInvalidDifficulty:    "Difficulty must be non-negative",
		CodeDaggerheartInvalidDualityDie:    "Duality dice must be between 1 and 12",
		CodeDaggerheartInvalidLevel:         "Level must be in range 1..10",
		CodeDaggerheartInvalidTraitValue:    "Trait {{.Trait}} value {{.Value}} must be in range -2 to +4",
		CodeDaggerheartInvalidStressMax:     "Stress maximum must be in range 0..12",
		CodeDaggerheartInvalidHpMax:         "HP maximum must be in range 1..12",
		CodeDaggerheartInvalidHp:            "HP {{.HP}} exceeds maximum {{.HPMax}}",
		CodeDaggerheartInvalidEvasion:       "Evasion must be non-negative",
		CodeDaggerheartInvalidThresholds:    "Severe threshold must be >= major threshold >= 0",
		CodeDaggerheartInvalidProficiency:   "Proficiency must be non-negative",
		CodeDaggerheartInvalidArmorMax:      "Armor max must be in range 0..12",
		CodeDaggerheartInvalidArmorScore:    "Armor score must be non-negative",
		CodeDaggerheartInvalidExperience:    "Experience name must be set",
		CodeDaggerheartInvalidRestSequence:  "Too many short rests in a row",
		CodeDaggerheartUnknownResource:      "Unknown resource: {{.Resource}}",
		CodeDaggerheartInsufficientResource: "Insufficient {{.Resource}}: have {{.Have}}, need {{.Need}}",
		CodeDaggerheartResourceAtCap:        "Resource {{.Resource}} is already at maximum",

		// Fork errors
		CodeForkEmptyCampaignID:  "Source campaign ID is required for fork",
		CodeForkInvalidForkPoint: "Invalid fork point specified",
		CodeForkPointInFuture:    "Fork point is beyond the current campaign state",
	},
}
