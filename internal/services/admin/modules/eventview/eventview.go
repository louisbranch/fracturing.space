// Package eventview provides shared helpers for formatting and filtering
// domain events across admin modules.
package eventview

import (
	"net/http"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"golang.org/x/text/message"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FormatTimestamp renders a protobuf timestamp as "2006-01-02 15:04:05".
func FormatTimestamp(value *timestamppb.Timestamp) string {
	if value == nil {
		return ""
	}
	return value.AsTime().Format("2006-01-02 15:04:05")
}

// FormatEventType translates event type keys into localized display strings.
func FormatEventType(eventType string, loc *message.Printer) string {
	switch eventType {
	case "campaign.created":
		return loc.Sprintf("event.campaign_created")
	case "campaign.forked":
		return loc.Sprintf("event.campaign_forked")
	case "campaign.updated":
		return loc.Sprintf("event.campaign_updated")
	case "participant.joined":
		return loc.Sprintf("event.participant_joined")
	case "participant.left":
		return loc.Sprintf("event.participant_left")
	case "participant.updated":
		return loc.Sprintf("event.participant_updated")
	case "character.created":
		return loc.Sprintf("event.character_created")
	case "character.deleted":
		return loc.Sprintf("event.character_deleted")
	case "character.updated":
		return loc.Sprintf("event.character_updated")
	case "character.profile_updated":
		return loc.Sprintf("event.character_profile_updated")
	case "session.started":
		return loc.Sprintf("event.session_started")
	case "session.ended":
		return loc.Sprintf("event.session_ended")
	case "session.gate_opened":
		return loc.Sprintf("event.session_gate_opened")
	case "session.gate_resolved":
		return loc.Sprintf("event.session_gate_resolved")
	case "session.gate_abandoned":
		return loc.Sprintf("event.session_gate_abandoned")
	case "session.spotlight_set":
		return loc.Sprintf("event.session_spotlight_set")
	case "session.spotlight_cleared":
		return loc.Sprintf("event.session_spotlight_cleared")
	case "invite.created":
		return loc.Sprintf("event.invite_created")
	case "invite.updated":
		return loc.Sprintf("event.invite_updated")
	case "action.roll_resolved":
		return loc.Sprintf("event.action_roll_resolved")
	case "action.outcome_applied":
		return loc.Sprintf("event.action_outcome_applied")
	case "action.outcome_rejected":
		return loc.Sprintf("event.action_outcome_rejected")
	case "action.note_added":
		return loc.Sprintf("event.action_note_added")
	case "action.character_state_patched":
		return loc.Sprintf("event.action_character_state_patched")
	case "action.gm_fear_changed":
		return loc.Sprintf("event.action_gm_fear_changed")
	case "action.death_move_resolved":
		return loc.Sprintf("event.action_death_move_resolved")
	case "action.blaze_of_glory_resolved":
		return loc.Sprintf("event.action_blaze_of_glory_resolved")
	case "action.attack_resolved":
		return loc.Sprintf("event.action_attack_resolved")
	case "action.reaction_resolved":
		return loc.Sprintf("event.action_reaction_resolved")
	case "action.damage_roll_resolved":
		return loc.Sprintf("event.action_damage_roll_resolved")
	case "action.adversary_action_resolved":
		return loc.Sprintf("event.action_adversary_action_resolved")
	default:
		parts := strings.Split(eventType, ".")
		if len(parts) > 0 {
			last := parts[len(parts)-1]
			if len(last) > 0 {
				formatted := strings.ReplaceAll(last, "_", " ")
				return strings.ToUpper(formatted[:1]) + formatted[1:]
			}
		}
		return eventType
	}
}

// FormatActorType translates actor type keys into localized display strings.
func FormatActorType(actorType string, loc *message.Printer) string {
	if actorType == "" {
		return ""
	}
	switch actorType {
	case "system":
		return loc.Sprintf("filter.actor.system")
	case "participant":
		return loc.Sprintf("filter.actor.participant")
	case "gm":
		return loc.Sprintf("filter.actor.gm")
	default:
		return actorType
	}
}

// FormatEventDescription returns a display description for an event.
func FormatEventDescription(event *statev1.Event, loc *message.Printer) string {
	if event == nil {
		return ""
	}
	return FormatEventType(event.GetType(), loc)
}

// BuildEventRows converts proto events into template-ready rows.
func BuildEventRows(events []*statev1.Event, loc *message.Printer) []templates.EventRow {
	rows := make([]templates.EventRow, 0, len(events))
	for _, event := range events {
		if event == nil {
			continue
		}
		rows = append(rows, templates.EventRow{
			CampaignID:       event.GetCampaignId(),
			Seq:              event.GetSeq(),
			Hash:             event.GetHash(),
			Type:             event.GetType(),
			TypeDisplay:      FormatEventType(event.GetType(), loc),
			Timestamp:        FormatTimestamp(event.GetTs()),
			SessionID:        event.GetSessionId(),
			ActorType:        event.GetActorType(),
			ActorTypeDisplay: FormatActorType(event.GetActorType(), loc),
			ActorName:        "",
			EntityType:       event.GetEntityType(),
			EntityID:         event.GetEntityId(),
			EntityName:       event.GetEntityId(),
			Description:      FormatEventDescription(event, loc),
			PayloadJSON:      string(event.GetPayloadJson()),
		})
	}
	return rows
}

// ParseEventFilters extracts event filter options from query parameters.
func ParseEventFilters(r *http.Request) templates.EventFilterOptions {
	return templates.EventFilterOptions{
		SessionID:  r.URL.Query().Get("session_id"),
		EventType:  r.URL.Query().Get("event_type"),
		ActorType:  r.URL.Query().Get("actor_type"),
		EntityType: r.URL.Query().Get("entity_type"),
		StartDate:  r.URL.Query().Get("start_date"),
		EndDate:    r.URL.Query().Get("end_date"),
	}
}

// EscapeAIP160StringLiteral escapes backslash and double-quote for AIP-160
// filter expressions.
func EscapeAIP160StringLiteral(v string) string {
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `"`, `\"`)
	return v
}

// BuildEventFilterExpression builds an AIP-160 filter string from filter options.
func BuildEventFilterExpression(filters templates.EventFilterOptions) string {
	var parts []string

	if filters.SessionID != "" {
		parts = append(parts, "session_id = \""+EscapeAIP160StringLiteral(filters.SessionID)+"\"")
	}
	if filters.EventType != "" {
		parts = append(parts, "type = \""+EscapeAIP160StringLiteral(filters.EventType)+"\"")
	}
	if filters.ActorType != "" {
		parts = append(parts, "actor_type = \""+EscapeAIP160StringLiteral(filters.ActorType)+"\"")
	}
	if filters.EntityType != "" {
		parts = append(parts, "entity_type = \""+EscapeAIP160StringLiteral(filters.EntityType)+"\"")
	}
	if filters.StartDate != "" {
		parts = append(parts, "ts >= timestamp(\""+EscapeAIP160StringLiteral(filters.StartDate)+"T00:00:00Z\")")
	}
	if filters.EndDate != "" {
		parts = append(parts, "ts <= timestamp(\""+EscapeAIP160StringLiteral(filters.EndDate)+"T23:59:59Z\")")
	}

	return strings.Join(parts, " AND ")
}

// EventFilterPushURL builds an HTMX push URL for event filter pagination.
func EventFilterPushURL(basePath string, filters templates.EventFilterOptions, pageToken string) string {
	pushURL := templates.EventFilterBaseURL(basePath, filters)
	if pageToken != "" {
		return templates.AppendQueryParam(pushURL, "page_token", pageToken)
	}
	return pushURL
}
