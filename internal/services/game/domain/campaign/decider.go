package campaign

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
)

const (
	CommandTypeCreate                 command.Type = "campaign.create"
	CommandTypeCreateWithParticipants command.Type = "campaign.create_with_participants"
	CommandTypeUpdate                 command.Type = "campaign.update"
	CommandTypeAIBind                 command.Type = "campaign.ai_bind"
	CommandTypeAIUnbind               command.Type = "campaign.ai_unbind"
	CommandTypeAIAuthRotate           command.Type = "campaign.ai_auth_rotate"
	CommandTypeFork                   command.Type = "campaign.fork"
	CommandTypeEnd                    command.Type = "campaign.end"
	CommandTypeArchive                command.Type = "campaign.archive"
	CommandTypeRestore                command.Type = "campaign.restore"
	EventTypeCreated                  event.Type   = "campaign.created"
	EventTypeUpdated                  event.Type   = "campaign.updated"
	EventTypeAIBound                  event.Type   = "campaign.ai_bound"
	EventTypeAIUnbound                event.Type   = "campaign.ai_unbound"
	EventTypeAIAuthRotated            event.Type   = "campaign.ai_auth_rotated"
	EventTypeForked                   event.Type   = "campaign.forked"

	rejectionCodeCampaignAlreadyExists        = "CAMPAIGN_ALREADY_EXISTS"
	rejectionCodeCampaignNotCreated           = "CAMPAIGN_NOT_CREATED"
	rejectionCodeCampaignNameEmpty            = "CAMPAIGN_NAME_EMPTY"
	rejectionCodeCampaignGameSystemInvalid    = "CAMPAIGN_INVALID_GAME_SYSTEM"
	rejectionCodeCampaignGmModeInvalid        = "CAMPAIGN_INVALID_GM_MODE"
	rejectionCodeCampaignUpdateEmpty          = "CAMPAIGN_UPDATE_EMPTY"
	rejectionCodeCampaignStatusInvalid        = "CAMPAIGN_INVALID_STATUS"
	rejectionCodeCampaignStatusTransition     = "CAMPAIGN_INVALID_STATUS_TRANSITION"
	rejectionCodeCampaignUpdateFieldInvalid   = "CAMPAIGN_UPDATE_FIELD_INVALID"
	rejectionCodeCampaignLocaleInvalid        = "CAMPAIGN_LOCALE_INVALID"
	rejectionCodeCampaignCoverAssetInvalid    = "CAMPAIGN_COVER_ASSET_INVALID"
	rejectionCodeCampaignCoverSetInvalid      = "CAMPAIGN_COVER_SET_INVALID"
	rejectionCodeCampaignAIAgentIDRequired    = "CAMPAIGN_AI_AGENT_ID_REQUIRED"
	rejectionCodeCampaignParticipantsEmpty    = "CAMPAIGN_PARTICIPANTS_REQUIRED"
	rejectionCodeCampaignParticipantDuplicate = "CAMPAIGN_PARTICIPANT_DUPLICATE"
	rejectionCodeCommandTypeUnsupported       = "COMMAND_TYPE_UNSUPPORTED"
	rejectionCodePayloadDecodeFailed          = "PAYLOAD_DECODE_FAILED"
)

var lifecycleCommandTargets = map[command.Type]Status{
	CommandTypeEnd:     StatusCompleted,
	CommandTypeArchive: StatusArchived,
	CommandTypeRestore: StatusDraft,
}

type updateFieldNormalizer func(currentStatus Status, value string) (string, *command.Rejection)

var campaignUpdateFieldNormalizers = map[string]updateFieldNormalizer{
	"status": func(currentStatus Status, value string) (string, *command.Rejection) {
		normalizedStatus, ok := normalizeStatusLabel(value)
		if !ok {
			return "", &command.Rejection{Code: rejectionCodeCampaignStatusInvalid, Message: "campaign status is invalid"}
		}
		if !isStatusTransitionAllowed(currentStatus, normalizedStatus) {
			return "", &command.Rejection{Code: rejectionCodeCampaignStatusTransition, Message: "campaign status transition is not allowed"}
		}
		return string(normalizedStatus), nil
	},
	"name": func(_ Status, value string) (string, *command.Rejection) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return "", &command.Rejection{Code: rejectionCodeCampaignNameEmpty, Message: "campaign name is required"}
		}
		return trimmed, nil
	},
	"theme_prompt": func(_ Status, value string) (string, *command.Rejection) {
		return strings.TrimSpace(value), nil
	},
	"locale": func(_ Status, value string) (string, *command.Rejection) {
		locale, ok := platformi18n.ParseLocale(value)
		if !ok {
			return "", &command.Rejection{Code: rejectionCodeCampaignLocaleInvalid, Message: "campaign locale is invalid"}
		}
		return platformi18n.LocaleString(locale), nil
	},
	"cover_asset_id": func(_ Status, value string) (string, *command.Rejection) {
		normalizedCoverAssetID, ok := normalizeCampaignCoverAssetID(value)
		if !ok {
			return "", &command.Rejection{Code: rejectionCodeCampaignCoverAssetInvalid, Message: "campaign cover asset is invalid"}
		}
		return normalizedCoverAssetID, nil
	},
	"cover_set_id": func(_ Status, value string) (string, *command.Rejection) {
		normalizedCoverSetID, ok := normalizeCampaignCoverSetID(value)
		if !ok {
			return "", &command.Rejection{Code: rejectionCodeCampaignCoverSetInvalid, Message: "campaign cover set is invalid"}
		}
		return normalizedCoverSetID, nil
	},
}

// Decide returns the decision for a campaign command against current state.
//
// This function is the campaign policy hub: it normalizes command payloads,
// enforces legal transitions, and emits immutable events that can be replayed
// to reproduce the same campaign state.
func Decide(state State, cmd command.Command, now func() time.Time) command.Decision {
	switch cmd.Type {
	case CommandTypeCreate:
		return decideCreate(state, cmd, now)
	case CommandTypeCreateWithParticipants:
		return decideCreateWithParticipants(state, cmd, now)
	case CommandTypeUpdate:
		return decideUpdate(state, cmd, now)
	case CommandTypeAIBind:
		return decideAIBind(state, cmd, now)
	case CommandTypeAIUnbind:
		return decideAIUnbind(state, cmd, now)
	case CommandTypeAIAuthRotate:
		return decideAIAuthRotate(state, cmd, now)
	case CommandTypeFork:
		return decideFork(state, cmd, now)
	case CommandTypeEnd, CommandTypeArchive, CommandTypeRestore:
		return decideLifecycleStatus(state, cmd, now)
	default:
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCommandTypeUnsupported,
			Message: fmt.Sprintf("command type %s is not supported by campaign decider", cmd.Type),
		})
	}
}

func decideCreate(state State, cmd command.Command, now func() time.Time) command.Decision {
	if state.Created {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCampaignAlreadyExists,
			Message: "campaign already exists",
		})
	}
	var payload CreatePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	normalizedPayload, rejection := normalizeCreatePayload(payload, cmd.CampaignID)
	if rejection != nil {
		return command.Reject(*rejection)
	}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeCreated, "campaign", cmd.CampaignID, payloadJSON, nowFunc(now)().UTC())
	return command.Accept(evt)
}

func decideCreateWithParticipants(state State, cmd command.Command, now func() time.Time) command.Decision {
	if state.Created {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCampaignAlreadyExists,
			Message: "campaign already exists",
		})
	}

	var payload CreateWithParticipantsPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}

	normalizedCampaign, rejection := normalizeCreatePayload(payload.Campaign, cmd.CampaignID)
	if rejection != nil {
		return command.Reject(*rejection)
	}
	if len(payload.Participants) == 0 {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCampaignParticipantsEmpty,
			Message: "at least one participant is required",
		})
	}

	decisionTime := nowFunc(now)().UTC()
	campaignPayloadJSON, _ := json.Marshal(normalizedCampaign)
	campaignCreated := command.NewEvent(cmd, EventTypeCreated, "campaign", cmd.CampaignID, campaignPayloadJSON, decisionTime)

	seenParticipantIDs := make(map[string]struct{}, len(payload.Participants))
	participantEvents := make([]event.Event, 0, len(payload.Participants))
	for _, joinPayload := range payload.Participants {
		participantID := strings.TrimSpace(joinPayload.ParticipantID)
		if _, exists := seenParticipantIDs[participantID]; exists {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCampaignParticipantDuplicate,
				Message: "participant ids must be unique",
			})
		}
		seenParticipantIDs[participantID] = struct{}{}

		joinPayloadJSON, _ := json.Marshal(joinPayload)
		joinDecision := participant.Decide(
			participant.State{},
			command.Command{
				CampaignID:   cmd.CampaignID,
				Type:         participant.CommandTypeJoin,
				ActorType:    command.ActorTypeSystem,
				RequestID:    cmd.RequestID,
				InvocationID: cmd.InvocationID,
				EntityType:   "participant",
				EntityID:     participantID,
				PayloadJSON:  joinPayloadJSON,
			},
			func() time.Time { return decisionTime },
		)
		if len(joinDecision.Rejections) > 0 {
			rej := joinDecision.Rejections[0]
			return command.Reject(command.Rejection{Code: rej.Code, Message: rej.Message})
		}
		if len(joinDecision.Events) != 1 {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCommandTypeUnsupported,
				Message: "participant bootstrap did not emit join event",
			})
		}
		participantEvents = append(participantEvents, joinDecision.Events[0])
	}

	events := make([]event.Event, 0, 1+len(participantEvents))
	events = append(events, campaignCreated)
	events = append(events, participantEvents...)
	return command.Accept(events...)
}

func normalizeCreatePayload(payload CreatePayload, campaignID string) (CreatePayload, *command.Rejection) {
	normalizedName := strings.TrimSpace(payload.Name)
	if normalizedName == "" {
		return CreatePayload{}, &command.Rejection{
			Code:    rejectionCodeCampaignNameEmpty,
			Message: "campaign name is required",
		}
	}
	normalizedGameSystem, ok := normalizeGameSystemLabel(payload.GameSystem)
	if !ok {
		return CreatePayload{}, &command.Rejection{
			Code:    rejectionCodeCampaignGameSystemInvalid,
			Message: "game system is required",
		}
	}
	normalizedGmMode, ok := normalizeGmModeLabel(payload.GmMode)
	if !ok {
		return CreatePayload{}, &command.Rejection{
			Code:    rejectionCodeCampaignGmModeInvalid,
			Message: "gm mode is required",
		}
	}

	coverAssetID := strings.TrimSpace(payload.CoverAssetID)
	if coverAssetID == "" {
		coverAssetID = defaultCampaignCoverAssetID(campaignID)
	}
	normalizedCoverAssetID, ok := normalizeCampaignCoverAssetID(coverAssetID)
	if !ok {
		return CreatePayload{}, &command.Rejection{
			Code:    rejectionCodeCampaignCoverAssetInvalid,
			Message: "campaign cover asset is invalid",
		}
	}
	coverSetID := strings.TrimSpace(payload.CoverSetID)
	if coverSetID == "" {
		coverSetID = defaultCampaignCoverSetID
	}
	normalizedCoverSetID, ok := normalizeCampaignCoverSetID(coverSetID)
	if !ok {
		return CreatePayload{}, &command.Rejection{
			Code:    rejectionCodeCampaignCoverSetInvalid,
			Message: "campaign cover set is invalid",
		}
	}

	return CreatePayload{
		Name:         normalizedName,
		Locale:       normalizeCampaignLocale(payload.Locale),
		GameSystem:   normalizedGameSystem,
		GmMode:       normalizedGmMode,
		Intent:       strings.TrimSpace(payload.Intent),
		AccessPolicy: strings.TrimSpace(payload.AccessPolicy),
		ThemePrompt:  payload.ThemePrompt,
		CoverAssetID: normalizedCoverAssetID,
		CoverSetID:   normalizedCoverSetID,
	}, nil
}

// normalizeCampaignLocale canonicalizes known locale labels/tags and falls
// back to the platform default so create-event payloads remain replay-safe.
func normalizeCampaignLocale(value string) string {
	locale, ok := platformi18n.ParseLocale(value)
	if !ok {
		return platformi18n.LocaleString(platformi18n.DefaultLocale())
	}
	return platformi18n.LocaleString(locale)
}

func decideUpdate(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Created {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCampaignNotCreated,
			Message: "campaign does not exist",
		})
	}
	var payload UpdatePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
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
		normalizeField, ok := campaignUpdateFieldNormalizers[key]
		if !ok {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeCampaignUpdateFieldInvalid,
				Message: "campaign update field is invalid",
			})
		}
		normalized, rejection := normalizeField(currentStatus, value)
		if rejection != nil {
			return command.Reject(*rejection)
		}
		normalizedFields[key] = normalized
	}

	normalizedPayload := UpdatePayload{Fields: normalizedFields}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeUpdated, "campaign", cmd.CampaignID, payloadJSON, nowFunc(now)().UTC())
	return command.Accept(evt)
}

func decideAIBind(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Created {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCampaignNotCreated,
			Message: "campaign does not exist",
		})
	}
	var payload AIBindPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	agentID := strings.TrimSpace(payload.AIAgentID)
	if agentID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCampaignAIAgentIDRequired,
			Message: "ai agent id is required",
		})
	}

	normalizedPayload := AIBindPayload{AIAgentID: agentID}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeAIBound, "campaign", cmd.CampaignID, payloadJSON, nowFunc(now)().UTC())
	return command.Accept(evt)
}

func decideAIUnbind(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Created {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCampaignNotCreated,
			Message: "campaign does not exist",
		})
	}
	payloadJSON, _ := json.Marshal(AIUnbindPayload{})
	evt := command.NewEvent(cmd, EventTypeAIUnbound, "campaign", cmd.CampaignID, payloadJSON, nowFunc(now)().UTC())
	return command.Accept(evt)
}

func decideAIAuthRotate(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Created {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCampaignNotCreated,
			Message: "campaign does not exist",
		})
	}

	var payload AIAuthRotatePayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}

	payload.EpochAfter = state.AIAuthEpoch + 1
	payload.Reason = strings.TrimSpace(payload.Reason)
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypeAIAuthRotated, "campaign", cmd.CampaignID, payloadJSON, nowFunc(now)().UTC())
	return command.Accept(evt)
}

func decideFork(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Created {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCampaignNotCreated,
			Message: "campaign does not exist",
		})
	}
	var payload ForkPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}
	payload.ParentCampaignID = strings.TrimSpace(payload.ParentCampaignID)
	payload.OriginCampaignID = strings.TrimSpace(payload.OriginCampaignID)
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypeForked, "campaign", cmd.CampaignID, payloadJSON, nowFunc(now)().UTC())
	return command.Accept(evt)
}

func decideLifecycleStatus(state State, cmd command.Command, now func() time.Time) command.Decision {
	targetStatus, _ := statusCommandTarget(cmd.Type)
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

	normalizedPayload := UpdatePayload{Fields: map[string]string{"status": string(targetStatus)}}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeUpdated, "campaign", cmd.CampaignID, payloadJSON, nowFunc(now)().UTC())
	return command.Accept(evt)
}

func nowFunc(now func() time.Time) func() time.Time {
	if now == nil {
		return time.Now
	}
	return now
}

// statusCommandTarget maps lifecycle command names to their destination status.
//
// Centralizing lifecycle transition targets prevents duplicate status-mapping logic
// in every handler and keeps command intent readable.
func statusCommandTarget(cmdType command.Type) (Status, bool) {
	target, ok := lifecycleCommandTargets[cmdType]
	return target, ok
}

// normalizeGameSystemLabel returns a canonical label for stable payload hashes.
//
// Campaign behavior depends on a stable system identity, not caller-specific casing
// or enum prefix variants.
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

// gameSystemFromLabel accepts enum labels with or without the GAME_SYSTEM_ prefix.
//
// This keeps campaign creation tolerant of payload shape differences while keeping
// canonical values internally.
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

// canonicalGameSystemLabel strips transport enum prefixes for stable, compact state.
func canonicalGameSystemLabel(system commonv1.GameSystem) string {
	label := system.String()
	label = strings.TrimPrefix(label, "GAME_SYSTEM_")
	return strings.ToLower(label)
}

// normalizeGmModeLabel returns a canonical label for stable payload hashes.
//
// The canonical GM mode value is the shared contract used later by permission and
// command-policy checks.
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
