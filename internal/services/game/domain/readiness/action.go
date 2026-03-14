package readiness

import "strings"

// ResolutionKind identifies one stable UI/action mapping for a readiness blocker.
type ResolutionKind string

const (
	// ResolutionKindUnspecified indicates the blocker has no direct self-service resolution target.
	ResolutionKindUnspecified ResolutionKind = ""
	// ResolutionKindCreateCharacter asks the responsible participant to create a character.
	ResolutionKindCreateCharacter ResolutionKind = "create_character"
	// ResolutionKindCompleteCharacter asks the responsible participant to finish a specific character.
	ResolutionKindCompleteCharacter ResolutionKind = "complete_character"
	// ResolutionKindConfigureAIAgent asks the responsible owner to bind an AI agent.
	ResolutionKindConfigureAIAgent ResolutionKind = "configure_ai_agent"
	// ResolutionKindInvitePlayer asks the responsible GM/owner to invite another player.
	ResolutionKindInvitePlayer ResolutionKind = "invite_player"
	// ResolutionKindManageParticipants asks the responsible owner to manage participant seats.
	ResolutionKindManageParticipants ResolutionKind = "manage_participants"
)

// Action carries structured responsibility and resolution data for a blocker.
type Action struct {
	ResponsibleUserIDs        []string
	ResponsibleParticipantIDs []string
	ResolutionKind            ResolutionKind
	TargetParticipantID       string
	TargetCharacterID         string
}

// Actionable reports whether the blocker can be resolved through a direct, stable action.
func (a Action) Actionable() bool {
	return a.ResolutionKind != ResolutionKindUnspecified && (len(a.ResponsibleUserIDs) > 0 || len(a.ResponsibleParticipantIDs) > 0)
}

func cloneAction(input Action) Action {
	result := Action{
		ResponsibleUserIDs:        append([]string{}, input.ResponsibleUserIDs...),
		ResponsibleParticipantIDs: append([]string{}, input.ResponsibleParticipantIDs...),
		ResolutionKind:            input.ResolutionKind,
		TargetParticipantID:       strings.TrimSpace(input.TargetParticipantID),
		TargetCharacterID:         strings.TrimSpace(input.TargetCharacterID),
	}
	result.ResponsibleUserIDs = normalizeActionIDs(result.ResponsibleUserIDs)
	result.ResponsibleParticipantIDs = normalizeActionIDs(result.ResponsibleParticipantIDs)
	return result
}

func normalizeActionIDs(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := set[value]; ok {
			continue
		}
		set[value] = struct{}{}
		result = append(result, value)
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
