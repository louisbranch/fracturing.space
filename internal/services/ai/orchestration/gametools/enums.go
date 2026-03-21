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
	switch strings.ToUpper(strings.TrimSpace(value)) {
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
