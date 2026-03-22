package gametools

import (
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func rollModeToProto(value string) commonv1.RollMode {
	switch strings.ToUpper(value) {
	case "REPLAY":
		return commonv1.RollMode_REPLAY
	case "LIVE":
		return commonv1.RollMode_LIVE
	default:
		return commonv1.RollMode_ROLL_MODE_UNSPECIFIED
	}
}

func rollModeLabel(value commonv1.RollMode) string {
	switch value {
	case commonv1.RollMode_REPLAY:
		return "REPLAY"
	case commonv1.RollMode_LIVE:
		return "LIVE"
	default:
		return ""
	}
}

func scenePhaseStatusToString(status statev1.ScenePhaseStatus) string {
	switch status {
	case statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM:
		return "GM"
	case statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_PLAYERS:
		return "PLAYERS"
	case statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_GM_REVIEW:
		return "GM_REVIEW"
	default:
		return "UNSPECIFIED"
	}
}

func scenePlayerSlotReviewStatusToString(status statev1.ScenePlayerSlotReviewStatus) string {
	switch status {
	case statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_OPEN:
		return "OPEN"
	case statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_UNDER_REVIEW:
		return "UNDER_REVIEW"
	case statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_ACCEPTED:
		return "ACCEPTED"
	case statev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_CHANGES_REQUESTED:
		return "CHANGES_REQUESTED"
	default:
		return "UNSPECIFIED"
	}
}

func participantRoleToString(role statev1.ParticipantRole) string {
	switch role {
	case statev1.ParticipantRole_GM:
		return "GM"
	case statev1.ParticipantRole_PLAYER:
		return "PLAYER"
	default:
		return "UNSPECIFIED"
	}
}

func controllerToString(controller statev1.Controller) string {
	switch controller {
	case statev1.Controller_CONTROLLER_HUMAN:
		return "HUMAN"
	case statev1.Controller_CONTROLLER_AI:
		return "AI"
	default:
		return "UNSPECIFIED"
	}
}

func characterKindToString(kind statev1.CharacterKind) string {
	switch kind {
	case statev1.CharacterKind_PC:
		return "PC"
	case statev1.CharacterKind_NPC:
		return "NPC"
	default:
		return "UNSPECIFIED"
	}
}

func campaignStatusToString(status statev1.CampaignStatus) string {
	switch status {
	case statev1.CampaignStatus_DRAFT:
		return "DRAFT"
	case statev1.CampaignStatus_ACTIVE:
		return "ACTIVE"
	case statev1.CampaignStatus_COMPLETED:
		return "COMPLETED"
	case statev1.CampaignStatus_ARCHIVED:
		return "ARCHIVED"
	default:
		return "UNSPECIFIED"
	}
}

func gmModeToString(mode statev1.GmMode) string {
	switch mode {
	case statev1.GmMode_HUMAN:
		return "HUMAN"
	case statev1.GmMode_AI:
		return "AI"
	case statev1.GmMode_HYBRID:
		return "HYBRID"
	default:
		return "UNSPECIFIED"
	}
}

func campaignIntentToString(intent statev1.CampaignIntent) string {
	switch intent {
	case statev1.CampaignIntent_STANDARD:
		return "STANDARD"
	case statev1.CampaignIntent_STARTER:
		return "STARTER"
	case statev1.CampaignIntent_SANDBOX:
		return "SANDBOX"
	default:
		return "UNSPECIFIED"
	}
}

func campaignAccessPolicyToString(policy statev1.CampaignAccessPolicy) string {
	switch policy {
	case statev1.CampaignAccessPolicy_PRIVATE:
		return "PRIVATE"
	case statev1.CampaignAccessPolicy_RESTRICTED:
		return "RESTRICTED"
	case statev1.CampaignAccessPolicy_PUBLIC:
		return "PUBLIC"
	default:
		return "UNSPECIFIED"
	}
}

func sessionStatusToString(status statev1.SessionStatus) string {
	switch status {
	case statev1.SessionStatus_SESSION_ACTIVE:
		return "ACTIVE"
	case statev1.SessionStatus_SESSION_ENDED:
		return "ENDED"
	default:
		return "UNSPECIFIED"
	}
}

func formatTimestamp(ts *timestamppb.Timestamp) string {
	if ts == nil {
		return ""
	}
	return ts.AsTime().UTC().Format(time.RFC3339)
}

func intSlice(values []int32) []int {
	converted := make([]int, len(values))
	for i, v := range values {
		converted[i] = int(v)
	}
	return converted
}

func daggerheartLifeStateToString(state pb.DaggerheartLifeState) string {
	switch state {
	case pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE:
		return "ALIVE"
	case pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS:
		return "UNCONSCIOUS"
	case pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY:
		return "BLAZE_OF_GLORY"
	case pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD:
		return "DEAD"
	default:
		return "UNSPECIFIED"
	}
}

func daggerheartConditionClearTriggerToString(trigger pb.DaggerheartConditionClearTrigger) string {
	switch trigger {
	case pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST:
		return "SHORT_REST"
	case pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_LONG_REST:
		return "LONG_REST"
	case pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SESSION_END:
		return "SESSION_END"
	case pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_DAMAGE_TAKEN:
		return "DAMAGE_TAKEN"
	default:
		return "UNSPECIFIED"
	}
}

func sessionSpotlightTypeToString(value statev1.SessionSpotlightType) string {
	switch value {
	case statev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_GM:
		return "GM"
	case statev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER:
		return "CHARACTER"
	default:
		return "UNSPECIFIED"
	}
}

func daggerheartSubclassTrackOriginToString(value pb.DaggerheartSubclassTrackOrigin) string {
	switch value {
	case pb.DaggerheartSubclassTrackOrigin_DAGGERHEART_SUBCLASS_TRACK_ORIGIN_PRIMARY:
		return "PRIMARY"
	case pb.DaggerheartSubclassTrackOrigin_DAGGERHEART_SUBCLASS_TRACK_ORIGIN_MULTICLASS:
		return "MULTICLASS"
	default:
		return "UNSPECIFIED"
	}
}

func daggerheartAdversaryFeatureStateStatusToString(value pb.DaggerheartAdversaryFeatureStateStatus) string {
	switch value {
	case pb.DaggerheartAdversaryFeatureStateStatus_DAGGERHEART_ADVERSARY_FEATURE_STATE_STATUS_READY:
		return "READY"
	case pb.DaggerheartAdversaryFeatureStateStatus_DAGGERHEART_ADVERSARY_FEATURE_STATE_STATUS_ACTIVE:
		return "ACTIVE"
	case pb.DaggerheartAdversaryFeatureStateStatus_DAGGERHEART_ADVERSARY_FEATURE_STATE_STATUS_COOLDOWN:
		return "COOLDOWN"
	case pb.DaggerheartAdversaryFeatureStateStatus_DAGGERHEART_ADVERSARY_FEATURE_STATE_STATUS_SPENT:
		return "SPENT"
	case pb.DaggerheartAdversaryFeatureStateStatus_DAGGERHEART_ADVERSARY_FEATURE_STATE_STATUS_STAGED:
		return "STAGED"
	default:
		return "UNSPECIFIED"
	}
}

func countdownToneToProto(value string) pb.DaggerheartCountdownTone {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "NEUTRAL", "DAGGERHEART_COUNTDOWN_TONE_NEUTRAL":
		return pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_NEUTRAL
	case "PROGRESS", "DAGGERHEART_COUNTDOWN_TONE_PROGRESS":
		return pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS
	case "CONSEQUENCE", "DAGGERHEART_COUNTDOWN_TONE_CONSEQUENCE":
		return pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_CONSEQUENCE
	default:
		return pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_UNSPECIFIED
	}
}

func countdownToneToString(value pb.DaggerheartCountdownTone) string {
	switch value {
	case pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_NEUTRAL:
		return "NEUTRAL"
	case pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS:
		return "PROGRESS"
	case pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_CONSEQUENCE:
		return "CONSEQUENCE"
	default:
		return "UNSPECIFIED"
	}
}

func countdownPolicyToProto(value string) pb.DaggerheartCountdownAdvancementPolicy {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "MANUAL", "DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL":
		return pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL
	case "ACTION_STANDARD", "DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_STANDARD":
		return pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_STANDARD
	case "ACTION_DYNAMIC", "DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_DYNAMIC":
		return pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_DYNAMIC
	case "LONG_REST", "DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_LONG_REST":
		return pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_LONG_REST
	default:
		return pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_UNSPECIFIED
	}
}

func countdownPolicyToString(value pb.DaggerheartCountdownAdvancementPolicy) string {
	switch value {
	case pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL:
		return "MANUAL"
	case pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_STANDARD:
		return "ACTION_STANDARD"
	case pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_DYNAMIC:
		return "ACTION_DYNAMIC"
	case pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_LONG_REST:
		return "LONG_REST"
	default:
		return "UNSPECIFIED"
	}
}

func countdownLoopBehaviorToProto(value string) pb.DaggerheartCountdownLoopBehavior {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "NONE", "DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE":
		return pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE
	case "RESET", "DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET":
		return pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET
	case "RESET_INCREASE_START", "DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_INCREASE_START":
		return pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_INCREASE_START
	case "RESET_DECREASE_START", "DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_DECREASE_START":
		return pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_DECREASE_START
	default:
		return pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_UNSPECIFIED
	}
}

func countdownLoopBehaviorToString(value pb.DaggerheartCountdownLoopBehavior) string {
	switch value {
	case pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE:
		return "NONE"
	case pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET:
		return "RESET"
	case pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_INCREASE_START:
		return "RESET_INCREASE_START"
	case pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET_DECREASE_START:
		return "RESET_DECREASE_START"
	default:
		return "UNSPECIFIED"
	}
}

func countdownStatusToString(value pb.DaggerheartCountdownStatus) string {
	switch value {
	case pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE:
		return "ACTIVE"
	case pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_TRIGGER_PENDING:
		return "TRIGGER_PENDING"
	default:
		return "UNSPECIFIED"
	}
}

func actionRollContextToProto(value string) pb.ActionRollContext {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "MOVE_SILENTLY", "ACTION_ROLL_CONTEXT_MOVE_SILENTLY":
		return pb.ActionRollContext_ACTION_ROLL_CONTEXT_MOVE_SILENTLY
	default:
		return pb.ActionRollContext_ACTION_ROLL_CONTEXT_UNSPECIFIED
	}
}

func gmMoveKindToProto(value string) pb.DaggerheartGmMoveKind {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "INTERRUPT_AND_MOVE", "DAGGERHEART_GM_MOVE_KIND_INTERRUPT_AND_MOVE":
		return pb.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_INTERRUPT_AND_MOVE
	case "ADDITIONAL_MOVE", "DAGGERHEART_GM_MOVE_KIND_ADDITIONAL_MOVE":
		return pb.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_ADDITIONAL_MOVE
	default:
		return pb.DaggerheartGmMoveKind_DAGGERHEART_GM_MOVE_KIND_UNSPECIFIED
	}
}

func gmMoveShapeToProto(value string) pb.DaggerheartGmMoveShape {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "SHOW_WORLD_REACTION", "DAGGERHEART_GM_MOVE_SHAPE_SHOW_WORLD_REACTION":
		return pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SHOW_WORLD_REACTION
	case "REVEAL_DANGER", "DAGGERHEART_GM_MOVE_SHAPE_REVEAL_DANGER":
		return pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_REVEAL_DANGER
	case "FORCE_SPLIT", "DAGGERHEART_GM_MOVE_SHAPE_FORCE_SPLIT":
		return pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_FORCE_SPLIT
	case "MARK_STRESS", "DAGGERHEART_GM_MOVE_SHAPE_MARK_STRESS":
		return pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_MARK_STRESS
	case "SHIFT_ENVIRONMENT", "DAGGERHEART_GM_MOVE_SHAPE_SHIFT_ENVIRONMENT":
		return pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SHIFT_ENVIRONMENT
	case "SPOTLIGHT_ADVERSARY", "DAGGERHEART_GM_MOVE_SHAPE_SPOTLIGHT_ADVERSARY":
		return pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_SPOTLIGHT_ADVERSARY
	case "CAPTURE_IMPORTANT_TARGET", "DAGGERHEART_GM_MOVE_SHAPE_CAPTURE_IMPORTANT_TARGET":
		return pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_CAPTURE_IMPORTANT_TARGET
	case "CUSTOM", "DAGGERHEART_GM_MOVE_SHAPE_CUSTOM":
		return pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_CUSTOM
	default:
		return pb.DaggerheartGmMoveShape_DAGGERHEART_GM_MOVE_SHAPE_UNSPECIFIED
	}
}

func daggerheartAttackRangeToProto(value string) pb.DaggerheartAttackRange {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "MELEE", "DAGGERHEART_ATTACK_RANGE_MELEE":
		return pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_MELEE
	case "RANGED", "DAGGERHEART_ATTACK_RANGE_RANGED":
		return pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_RANGED
	default:
		return pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_UNSPECIFIED
	}
}

func daggerheartDamageTypeToProto(value string) pb.DaggerheartDamageType {
	switch strings.ToUpper(strings.TrimSpace(value)) {
	case "PHYSICAL", "DAGGERHEART_DAMAGE_TYPE_PHYSICAL":
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL
	case "MAGIC", "DAGGERHEART_DAMAGE_TYPE_MAGIC":
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MAGIC
	case "MIXED", "DAGGERHEART_DAMAGE_TYPE_MIXED":
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_MIXED
	default:
		return pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_UNSPECIFIED
	}
}
