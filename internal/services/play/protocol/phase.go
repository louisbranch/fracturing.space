package protocol

import (
	"strings"

	gamev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

// ScenePlayerPhase represents the current player phase in a scene.
type ScenePlayerPhase struct {
	PhaseID              string            `json:"phase_id"`
	Status               string            `json:"status,omitempty"`
	ActingCharacterIDs   []string          `json:"acting_character_ids"`
	ActingParticipantIDs []string          `json:"acting_participant_ids"`
	Slots                []ScenePlayerSlot `json:"slots"`
}

// ScenePlayerSlot represents one player's submission slot.
type ScenePlayerSlot struct {
	ParticipantID      string   `json:"participant_id"`
	SummaryText        string   `json:"summary_text,omitempty"`
	CharacterIDs       []string `json:"character_ids"`
	UpdatedAt          string   `json:"updated_at,omitempty"`
	Yielded            bool     `json:"yielded"`
	ReviewStatus       string   `json:"review_status,omitempty"`
	ReviewReason       string   `json:"review_reason,omitempty"`
	ReviewCharacterIDs []string `json:"review_character_ids"`
}

// PlayerPhaseFromGamePhase maps a proto ScenePlayerPhase to protocol.
func PlayerPhaseFromGamePhase(phase *gamev1.ScenePlayerPhase) *ScenePlayerPhase {
	if phase == nil {
		return nil
	}
	phaseID := strings.TrimSpace(phase.GetPhaseId())
	if phaseID == "" {
		return nil
	}
	slots := make([]ScenePlayerSlot, 0, len(phase.GetSlots()))
	for _, s := range phase.GetSlots() {
		slots = append(slots, ScenePlayerSlot{
			ParticipantID:      strings.TrimSpace(s.GetParticipantId()),
			SummaryText:        strings.TrimSpace(s.GetSummaryText()),
			CharacterIDs:       TrimStringSlice(s.GetCharacterIds()),
			UpdatedAt:          FormatTimestamp(s.GetUpdatedAt()),
			Yielded:            s.GetYielded(),
			ReviewStatus:       slotReviewStatusString(s.GetReviewStatus()),
			ReviewReason:       strings.TrimSpace(s.GetReviewReason()),
			ReviewCharacterIDs: TrimStringSlice(s.GetReviewCharacterIds()),
		})
	}
	return &ScenePlayerPhase{
		PhaseID:              phaseID,
		Status:               scenePhaseStatusString(phase.GetStatus()),
		ActingCharacterIDs:   TrimStringSlice(phase.GetActingCharacterIds()),
		ActingParticipantIDs: TrimStringSlice(phase.GetActingParticipantIds()),
		Slots:                slots,
	}
}

func scenePhaseStatusString(value gamev1.ScenePhaseStatus) string {
	return ProtoEnumToLower(value, gamev1.ScenePhaseStatus_SCENE_PHASE_STATUS_UNSPECIFIED, "SCENE_PHASE_STATUS_")
}

func slotReviewStatusString(value gamev1.ScenePlayerSlotReviewStatus) string {
	return ProtoEnumToLower(value, gamev1.ScenePlayerSlotReviewStatus_SCENE_PLAYER_SLOT_REVIEW_STATUS_UNSPECIFIED, "SCENE_PLAYER_SLOT_REVIEW_STATUS_")
}
