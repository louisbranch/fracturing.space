package campaign

import (
	"encoding/json"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

const (
	commandTypeCreate  command.Type = "campaign.create"
	commandTypeUpdate  command.Type = "campaign.update"
	commandTypeFork    command.Type = "campaign.fork"
	commandTypeEnd     command.Type = "campaign.end"
	commandTypeArchive command.Type = "campaign.archive"
	commandTypeRestore command.Type = "campaign.restore"
	eventTypeCreated   event.Type   = "campaign.created"
	eventTypeUpdated   event.Type   = "campaign.updated"
	eventTypeForked    event.Type   = "campaign.forked"

	rejectionCodeCampaignAlreadyExists      = "CAMPAIGN_ALREADY_EXISTS"
	rejectionCodeCampaignNotCreated         = "CAMPAIGN_NOT_CREATED"
	rejectionCodeCampaignNameEmpty          = "CAMPAIGN_NAME_EMPTY"
	rejectionCodeCampaignGameSystemInvalid  = "CAMPAIGN_INVALID_GAME_SYSTEM"
	rejectionCodeCampaignGmModeInvalid      = "CAMPAIGN_INVALID_GM_MODE"
	rejectionCodeCampaignUpdateEmpty        = "CAMPAIGN_UPDATE_EMPTY"
	rejectionCodeCampaignStatusInvalid      = "CAMPAIGN_INVALID_STATUS"
	rejectionCodeCampaignStatusTransition   = "CAMPAIGN_INVALID_STATUS_TRANSITION"
	rejectionCodeCampaignUpdateFieldInvalid = "CAMPAIGN_UPDATE_FIELD_INVALID"
)

// Decide returns the decision for a campaign command against current state.
func Decide(state State, cmd command.Command, now func() time.Time) command.Decision {
	if cmd.Type == commandTypeCreate {
		if state.Created {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCampaignAlreadyExists,
				Message: "campaign already exists",
			})
		}
		var payload CreatePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		normalizedName := strings.TrimSpace(payload.Name)
		if normalizedName == "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCampaignNameEmpty,
				Message: "campaign name is required",
			})
		}
		normalizedGameSystem, ok := normalizeGameSystemLabel(payload.GameSystem)
		if !ok {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCampaignGameSystemInvalid,
				Message: "game system is required",
			})
		}
		normalizedGmMode, ok := normalizeGmModeLabel(payload.GmMode)
		if !ok {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCampaignGmModeInvalid,
				Message: "gm mode is required",
			})
		}
		if now == nil {
			now = time.Now
		}

		normalizedPayload := CreatePayload{
			Name:         normalizedName,
			Locale:       strings.TrimSpace(payload.Locale),
			GameSystem:   normalizedGameSystem,
			GmMode:       normalizedGmMode,
			Intent:       strings.TrimSpace(payload.Intent),
			AccessPolicy: strings.TrimSpace(payload.AccessPolicy),
			ThemePrompt:  payload.ThemePrompt,
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
			EntityType:    "campaign",
			EntityID:      cmd.CampaignID,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	}

	if cmd.Type == commandTypeUpdate {
		if !state.Created {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCampaignNotCreated,
				Message: "campaign does not exist",
			})
		}
		var payload UpdatePayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		if len(payload.Fields) == 0 {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCampaignUpdateEmpty,
				Message: "campaign update requires fields",
			})
		}

		normalizedFields := make(map[string]string, len(payload.Fields))
		currentStatus := state.Status
		if currentStatus == "" {
			currentStatus = StatusDraft
		}
		for key, value := range payload.Fields {
			switch key {
			case "status":
				normalizedStatus, ok := normalizeStatusLabel(value)
				if !ok {
					return command.Reject(command.Rejection{
						Code:    rejectionCodeCampaignStatusInvalid,
						Message: "campaign status is invalid",
					})
				}
				if !isStatusTransitionAllowed(currentStatus, normalizedStatus) {
					return command.Reject(command.Rejection{
						Code:    rejectionCodeCampaignStatusTransition,
						Message: "campaign status transition is not allowed",
					})
				}
				normalizedFields[key] = string(normalizedStatus)
			case "name":
				trimmed := strings.TrimSpace(value)
				if trimmed == "" {
					return command.Reject(command.Rejection{
						Code:    rejectionCodeCampaignNameEmpty,
						Message: "campaign name is required",
					})
				}
				normalizedFields[key] = trimmed
			case "theme_prompt":
				normalizedFields[key] = strings.TrimSpace(value)
			default:
				return command.Reject(command.Rejection{
					Code:    rejectionCodeCampaignUpdateFieldInvalid,
					Message: "campaign update field is invalid",
				})
			}
		}
		if now == nil {
			now = time.Now
		}

		normalizedPayload := UpdatePayload{Fields: normalizedFields}
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
			EntityType:    "campaign",
			EntityID:      cmd.CampaignID,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	}

	if cmd.Type == commandTypeFork {
		if !state.Created {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCampaignNotCreated,
				Message: "campaign does not exist",
			})
		}
		var payload ForkPayload
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		payload.ParentCampaignID = strings.TrimSpace(payload.ParentCampaignID)
		payload.OriginCampaignID = strings.TrimSpace(payload.OriginCampaignID)
		if now == nil {
			now = time.Now
		}
		payloadJSON, _ := json.Marshal(payload)
		evt := event.Event{
			CampaignID:    cmd.CampaignID,
			Type:          eventTypeForked,
			Timestamp:     now().UTC(),
			ActorType:     event.ActorType(cmd.ActorType),
			ActorID:       cmd.ActorID,
			SessionID:     cmd.SessionID,
			RequestID:     cmd.RequestID,
			InvocationID:  cmd.InvocationID,
			EntityType:    "campaign",
			EntityID:      cmd.CampaignID,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	}

	if targetStatus, ok := statusCommandTarget(cmd.Type); ok {
		if !state.Created {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCampaignNotCreated,
				Message: "campaign does not exist",
			})
		}
		currentStatus := state.Status
		if currentStatus == "" {
			currentStatus = StatusDraft
		}
		if !isStatusTransitionAllowed(currentStatus, targetStatus) {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCampaignStatusTransition,
				Message: "campaign status transition is not allowed",
			})
		}
		if now == nil {
			now = time.Now
		}

		normalizedPayload := UpdatePayload{Fields: map[string]string{"status": string(targetStatus)}}
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
			EntityType:    "campaign",
			EntityID:      cmd.CampaignID,
			CorrelationID: cmd.CorrelationID,
			CausationID:   cmd.CausationID,
			PayloadJSON:   payloadJSON,
		}

		return command.Accept(evt)
	}

	return command.Decision{}
}

// statusCommandTarget maps lifecycle commands to their target status.
func statusCommandTarget(cmdType command.Type) (Status, bool) {
	switch cmdType {
	case commandTypeEnd:
		return StatusCompleted, true
	case commandTypeArchive:
		return StatusArchived, true
	case commandTypeRestore:
		return StatusDraft, true
	default:
		return "", false
	}
}

// normalizeGameSystemLabel returns a canonical label for stable payload hashes.
func normalizeGameSystemLabel(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	system, ok := gameSystemFromLabel(trimmed)
	if !ok || system == commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED {
		return "", false
	}
	return canonicalGameSystemLabel(system), true
}

// gameSystemFromLabel accepts enum labels with or without the prefix.
func gameSystemFromLabel(value string) (commonv1.GameSystem, bool) {
	if system, ok := commonv1.GameSystem_value[value]; ok {
		return commonv1.GameSystem(system), true
	}
	upper := strings.ToUpper(value)
	if system, ok := commonv1.GameSystem_value["GAME_SYSTEM_"+upper]; ok {
		return commonv1.GameSystem(system), true
	}
	return commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, false
}

// canonicalGameSystemLabel prefers lowercase short labels.
func canonicalGameSystemLabel(system commonv1.GameSystem) string {
	label := system.String()
	label = strings.TrimPrefix(label, "GAME_SYSTEM_")
	return strings.ToLower(label)
}

// normalizeGmModeLabel returns a canonical label for stable payload hashes.
func normalizeGmModeLabel(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", false
	}
	switch strings.ToUpper(trimmed) {
	case "HUMAN", "GM_MODE_HUMAN":
		return "human", true
	case "AI", "GM_MODE_AI":
		return "ai", true
	case "HYBRID", "GM_MODE_HYBRID":
		return "hybrid", true
	default:
		return "", false
	}
}
