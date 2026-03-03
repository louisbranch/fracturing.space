package dashboard

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/admin/platform/modulehandler"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestDashboardServiceNilClients(t *testing.T) {
	svc := service{base: modulehandler.NewBase(nil)}

	rec := httptest.NewRecorder()
	svc.HandleDashboard(rec, httptest.NewRequest(http.MethodGet, "/app/dashboard", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleDashboard() status = %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	svc.HandleDashboardContent(rec, httptest.NewRequest(http.MethodGet, "/app/dashboard?fragment=rows", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("HandleDashboardContent(nil clients) status = %d", rec.Code)
	}
}

func TestDashboardHelpersFormatting(t *testing.T) {
	loc := i18n.Printer(i18n.Default())

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "campaign_created", input: "campaign.created", want: loc.Sprintf("event.campaign_created")},
		{name: "campaign_forked", input: "campaign.forked", want: loc.Sprintf("event.campaign_forked")},
		{name: "campaign_updated", input: "campaign.updated", want: loc.Sprintf("event.campaign_updated")},
		{name: "participant_joined", input: "participant.joined", want: loc.Sprintf("event.participant_joined")},
		{name: "participant_left", input: "participant.left", want: loc.Sprintf("event.participant_left")},
		{name: "participant_updated", input: "participant.updated", want: loc.Sprintf("event.participant_updated")},
		{name: "character_created", input: "character.created", want: loc.Sprintf("event.character_created")},
		{name: "character_deleted", input: "character.deleted", want: loc.Sprintf("event.character_deleted")},
		{name: "character_updated", input: "character.updated", want: loc.Sprintf("event.character_updated")},
		{name: "character_profile_updated", input: "character.profile_updated", want: loc.Sprintf("event.character_profile_updated")},
		{name: "session_started", input: "session.started", want: loc.Sprintf("event.session_started")},
		{name: "session_ended", input: "session.ended", want: loc.Sprintf("event.session_ended")},
		{name: "session_gate_opened", input: "session.gate_opened", want: loc.Sprintf("event.session_gate_opened")},
		{name: "session_gate_resolved", input: "session.gate_resolved", want: loc.Sprintf("event.session_gate_resolved")},
		{name: "session_gate_abandoned", input: "session.gate_abandoned", want: loc.Sprintf("event.session_gate_abandoned")},
		{name: "session_spotlight_set", input: "session.spotlight_set", want: loc.Sprintf("event.session_spotlight_set")},
		{name: "session_spotlight_cleared", input: "session.spotlight_cleared", want: loc.Sprintf("event.session_spotlight_cleared")},
		{name: "invite_created", input: "invite.created", want: loc.Sprintf("event.invite_created")},
		{name: "invite_updated", input: "invite.updated", want: loc.Sprintf("event.invite_updated")},
		{name: "action_roll_resolved", input: "action.roll_resolved", want: loc.Sprintf("event.action_roll_resolved")},
		{name: "action_outcome_applied", input: "action.outcome_applied", want: loc.Sprintf("event.action_outcome_applied")},
		{name: "action_outcome_rejected", input: "action.outcome_rejected", want: loc.Sprintf("event.action_outcome_rejected")},
		{name: "action_note_added", input: "action.note_added", want: loc.Sprintf("event.action_note_added")},
		{name: "action_character_state_patched", input: "action.character_state_patched", want: loc.Sprintf("event.action_character_state_patched")},
		{name: "action_gm_fear_changed", input: "action.gm_fear_changed", want: loc.Sprintf("event.action_gm_fear_changed")},
		{name: "action_death_move_resolved", input: "action.death_move_resolved", want: loc.Sprintf("event.action_death_move_resolved")},
		{name: "action_blaze_of_glory_resolved", input: "action.blaze_of_glory_resolved", want: loc.Sprintf("event.action_blaze_of_glory_resolved")},
		{name: "action_attack_resolved", input: "action.attack_resolved", want: loc.Sprintf("event.action_attack_resolved")},
		{name: "action_reaction_resolved", input: "action.reaction_resolved", want: loc.Sprintf("event.action_reaction_resolved")},
		{name: "action_damage_roll_resolved", input: "action.damage_roll_resolved", want: loc.Sprintf("event.action_damage_roll_resolved")},
		{name: "action_adversary_action_resolved", input: "action.adversary_action_resolved", want: loc.Sprintf("event.action_adversary_action_resolved")},
		{name: "fallback_underscore", input: "custom.some_event_type", want: "Some event type"},
		{name: "fallback_simple", input: "custom.hello", want: "Hello"},
		{name: "empty", input: "", want: ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := formatEventType(tc.input, loc); got != tc.want {
				t.Fatalf("formatEventType(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}

	if got := formatEventDescription(nil, loc); got != "" {
		t.Fatalf("formatEventDescription(nil) = %q", got)
	}
	if got := formatEventDescription(&statev1.Event{Type: "campaign.created"}, loc); got != loc.Sprintf("event.campaign_created") {
		t.Fatalf("formatEventDescription(event) = %q", got)
	}

	ts := timestamppb.New(time.Date(2026, time.March, 2, 15, 4, 5, 0, time.UTC))
	if got := formatTimestamp(ts); got != "2026-03-02 15:04:05" {
		t.Fatalf("formatTimestamp() = %q", got)
	}
	if got := formatTimestamp(nil); got != "" {
		t.Fatalf("formatTimestamp(nil) = %q", got)
	}
}
