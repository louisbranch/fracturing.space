package event

import "testing"

func TestType_IsValid(t *testing.T) {
	tests := []struct {
		eventType Type
		want      bool
	}{
		// Campaign events
		{TypeCampaignCreated, true},
		{TypeCampaignForked, true},
		{TypeCampaignStatusChanged, true},
		{TypeCampaignUpdated, true},
		// Participant events
		{TypeParticipantJoined, true},
		{TypeParticipantLeft, true},
		{TypeParticipantUpdated, true},
		// Character events
		{TypeCharacterCreated, true},
		{TypeCharacterDeleted, true},
		{TypeCharacterUpdated, true},
		{TypeProfileUpdated, true},
		{TypeControllerAssigned, true},
		// Snapshot-related events
		{TypeCharacterStateChanged, true},
		{TypeGMFearChanged, true},
		// Session events
		{TypeSessionStarted, true},
		{TypeSessionEnded, true},
		// Action events (facts, not commands)
		{TypeRollResolved, true},
		{TypeOutcomeApplied, true},
		{TypeOutcomeRejected, true},
		{TypeNoteAdded, true},
		// Empty type
		{"", false},
		// Custom types are allowed
		{"invalid", true},
		{"campaign.invalid", true},
		{"unknown.event", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			if got := tt.eventType.IsValid(); got != tt.want {
				t.Errorf("Type(%q).IsValid() = %v, want %v", tt.eventType, got, tt.want)
			}
		})
	}
}

func TestType_Domain(t *testing.T) {
	tests := []struct {
		eventType Type
		want      string
	}{
		{TypeCampaignCreated, "campaign"},
		{TypeCampaignForked, "campaign"},
		{TypeParticipantJoined, "participant"},
		{TypeCharacterCreated, "character"},
		{TypeCharacterStateChanged, "snapshot"},
		{TypeSessionStarted, "session"},
		{TypeRollResolved, "action"},
		{Type("nodot"), "nodot"},
		{Type(""), ""},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			if got := tt.eventType.Domain(); got != tt.want {
				t.Errorf("Type(%q).Domain() = %q, want %q", tt.eventType, got, tt.want)
			}
		})
	}
}
