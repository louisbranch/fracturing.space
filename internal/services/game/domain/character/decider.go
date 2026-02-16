package character

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

const (
	commandTypeCreate        command.Type = "character.create"
	commandTypeUpdate        command.Type = "character.update"
	commandTypeDelete        command.Type = "character.delete"
	commandTypeProfileUpdate command.Type = "character.profile_update"
	eventTypeCreated         event.Type   = "character.created"
	eventTypeUpdated         event.Type   = "character.updated"
	eventTypeDeleted         event.Type   = "character.deleted"
	eventTypeProfileUpdated  event.Type   = "character.profile_updated"

	rejectionCodeCharacterAlreadyExists      = "CHARACTER_ALREADY_EXISTS"
	rejectionCodeCharacterIDRequired         = "CHARACTER_ID_REQUIRED"
	rejectionCodeCharacterNameEmpty          = "CHARACTER_NAME_EMPTY"
	rejectionCodeCharacterKindInvalid        = "CHARACTER_KIND_INVALID"
	rejectionCodeCharacterNotCreated         = "CHARACTER_NOT_CREATED"
	rejectionCodeCharacterUpdateEmpty        = "CHARACTER_UPDATE_EMPTY"
	rejectionCodeCharacterUpdateFieldInvalid = "CHARACTER_UPDATE_FIELD_INVALID"
)

// Decide returns the decision for a character command against current state.
func Decide(state State, cmd command.Command, now func() time.Time) command.Decision {
	if cmd.Type == commandTypeCreate {
		if state.Created {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterAlreadyExists,
				Message: "character already exists",
			})
		}
		var payload CreatePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		characterID := strings.TrimSpace(payload.CharacterID)
		if characterID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterIDRequired,
				Message: "character id is required",
			})
		}
		name := strings.TrimSpace(payload.Name)
		if name == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterNameEmpty,
				Message: "character name is required",
			})
		}
		kind, ok := normalizeCharacterKindLabel(payload.Kind)
		if !ok {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterKindInvalid,
				Message: "character kind is invalid",
			})
		}
		notes := strings.TrimSpace(payload.Notes)
		if now == nil {
			now = time.Now
		}

		normalizedPayload := CreatePayload{
			CharacterID: characterID,
			Name:        name,
			Kind:        kind,
			Notes:       notes,
		}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeCreated,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "character",
			EntityID:      characterID,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	}

	if cmd.Type == commandTypeUpdate {
		if !state.Created || state.Deleted {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterNotCreated,
				Message: "character not created",
			})
		}
		var payload UpdatePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		characterID := strings.TrimSpace(payload.CharacterID)
		if characterID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterIDRequired,
				Message: "character id is required",
			})
		}
		if len(payload.Fields) == 0 {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterUpdateEmpty,
				Message: "character update requires fields",
			})
		}
		normalizedFields := make(map[string]string, len(payload.Fields))
		for key, value := range payload.Fields {
			switch key {
			case "name":
				trimmed := strings.TrimSpace(value)
				if trimmed == "" {
					return command.Reject(command.Rejection{
						Code:    rejectionCodeCharacterNameEmpty,
						Message: "character name is required",
					})
				}
				normalizedFields[key] = trimmed
			case "kind":
				kind, ok := normalizeCharacterKindLabel(value)
				if !ok {
					return command.Reject(command.Rejection{
						Code:    rejectionCodeCharacterKindInvalid,
						Message: "character kind is invalid",
					})
				}
				normalizedFields[key] = kind
			case "notes":
				normalizedFields[key] = strings.TrimSpace(value)
			case "participant_id":
				normalizedFields[key] = strings.TrimSpace(value)
			default:
				return command.Reject(command.Rejection{
					Code:    rejectionCodeCharacterUpdateFieldInvalid,
					Message: "character update field is invalid",
				})
			}
		}
		if now == nil {
			now = time.Now
		}

		normalizedPayload := UpdatePayload{CharacterID: characterID, Fields: normalizedFields}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeUpdated,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "character",
			EntityID:      characterID,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	}

	if cmd.Type == commandTypeDelete {
		if !state.Created || state.Deleted {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterNotCreated,
				Message: "character not created",
			})
		}
		var payload DeletePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		characterID := strings.TrimSpace(payload.CharacterID)
		if characterID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterIDRequired,
				Message: "character id is required",
			})
		}
		reason := strings.TrimSpace(payload.Reason)
		if now == nil {
			now = time.Now
		}

		normalizedPayload := DeletePayload{CharacterID: characterID, Reason: reason}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeDeleted,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "character",
			EntityID:      characterID,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	}

	if cmd.Type == commandTypeProfileUpdate {
		if !state.Created || state.Deleted {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterNotCreated,
				Message: "character not created",
			})
		}
		var payload ProfileUpdatePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		characterID := strings.TrimSpace(payload.CharacterID)
		if characterID == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCharacterIDRequired,
				Message: "character id is required",
			})
		}
		if now == nil {
			now = time.Now
		}

		normalizedPayload := ProfileUpdatePayload{
			CharacterID:   characterID,
			SystemProfile: payload.SystemProfile,
		}
		payloadJSON, _ := json.Marshal(normalizedPayload)
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeProfileUpdated,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "character",
			EntityID:      characterID,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	}

	return command.Decision{}
}

// normalizeCharacterKindLabel returns a canonical character kind label.
func normalizeCharacterKindLabel(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	upper := strings.ToUpper(trimmed)
	switch upper {
	case "PC", "CHARACTER_KIND_PC":
		return "pc", true
	case "NPC", "CHARACTER_KIND_NPC":
		return "npc", true
	default:
		return "", false
	}
}
