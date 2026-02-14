package templates

import "net/url"

// EventRow represents an event in the timeline (enhanced version).
type EventRow struct {
	CampaignID       string
	Seq              uint64
	Hash             string
	Type             string
	TypeDisplay      string
	Timestamp        string
	SessionID        string
	ActorType        string
	ActorTypeDisplay string
	ActorName        string
	EntityType       string
	EntityID         string
	EntityName       string
	Description      string
	PayloadJSON      string
	Expanded         bool
}

// EventFilterOptions holds the current filter state for event lists.
type EventFilterOptions struct {
	SessionID  string
	EventType  string
	ActorType  string
	EntityType string
	StartDate  string
	EndDate    string
}

// EventFilterBaseURL returns a base URL with filter query parameters applied.
func EventFilterBaseURL(baseURL string, filters EventFilterOptions) string {
	query := url.Values{}
	if filters.SessionID != "" {
		query.Set("session_id", filters.SessionID)
	}
	if filters.EventType != "" {
		query.Set("event_type", filters.EventType)
	}
	if filters.ActorType != "" {
		query.Set("actor_type", filters.ActorType)
	}
	if filters.EntityType != "" {
		query.Set("entity_type", filters.EntityType)
	}
	if filters.StartDate != "" {
		query.Set("start_date", filters.StartDate)
	}
	if filters.EndDate != "" {
		query.Set("end_date", filters.EndDate)
	}
	encoded := query.Encode()
	if encoded == "" {
		return baseURL
	}
	return baseURL + "?" + encoded
}

// EventLogView holds data for rendering the event log page.
type EventLogView struct {
	CampaignID   string
	CampaignName string
	SessionID    string
	SessionName  string
	Events       []EventRow
	Filters      EventFilterOptions
	NextToken    string
	PrevToken    string
	TotalCount   int32
}

// EventTypeOption represents an option in the event type filter dropdown.
type EventTypeOption struct {
	Value   string
	Label   string
	Current bool
}

// GetEventTypeOptions returns the available event type filter options.
func GetEventTypeOptions(current string, loc Localizer) []EventTypeOption {
	types := []struct {
		Value string
		Label string
	}{
		{"", T(loc, "filter.all_types")},
		{"campaign.created", T(loc, "event.campaign_created")},
		{"session.started", T(loc, "event.session_started")},
		{"session.ended", T(loc, "event.session_ended")},
		{"session.gate_opened", T(loc, "event.session_gate_opened")},
		{"session.gate_resolved", T(loc, "event.session_gate_resolved")},
		{"session.gate_abandoned", T(loc, "event.session_gate_abandoned")},
		{"session.spotlight_set", T(loc, "event.session_spotlight_set")},
		{"session.spotlight_cleared", T(loc, "event.session_spotlight_cleared")},
		{"character.created", T(loc, "event.character_created")},
		{"participant.joined", T(loc, "event.participant_joined")},
		{"action.roll_resolved", T(loc, "event.action_roll_resolved")},
		{"action.outcome_applied", T(loc, "event.action_outcome_applied")},
	}

	options := make([]EventTypeOption, len(types))
	for i, t := range types {
		options[i] = EventTypeOption{
			Value:   t.Value,
			Label:   t.Label,
			Current: t.Value == current,
		}
	}
	return options
}

// GetActorTypeOptions returns the available actor type filter options.
func GetActorTypeOptions(current string, loc Localizer) []EventTypeOption {
	types := []struct {
		Value string
		Label string
	}{
		{"", T(loc, "filter.all_actors")},
		{"system", T(loc, "filter.actor.system")},
		{"participant", T(loc, "filter.actor.participant")},
		{"gm", T(loc, "filter.actor.gm")},
	}

	options := make([]EventTypeOption, len(types))
	for i, t := range types {
		options[i] = EventTypeOption{
			Value:   t.Value,
			Label:   t.Label,
			Current: t.Value == current,
		}
	}
	return options
}

// GetEntityTypeOptions returns the available entity type filter options.
func GetEntityTypeOptions(current string, loc Localizer) []EventTypeOption {
	types := []struct {
		Value string
		Label string
	}{
		{"", T(loc, "filter.all_entities")},
		{"character", T(loc, "filter.entity.character")},
		{"session", T(loc, "filter.entity.session")},
		{"campaign", T(loc, "filter.entity.campaign")},
		{"participant", T(loc, "filter.entity.participant")},
	}

	options := make([]EventTypeOption, len(types))
	for i, t := range types {
		options[i] = EventTypeOption{
			Value:   t.Value,
			Label:   t.Label,
			Current: t.Value == current,
		}
	}
	return options
}
