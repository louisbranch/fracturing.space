package scenarios

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestScenarioHelpersScenarioScript(t *testing.T) {
	defaultScript := defaultScenarioScript()
	if !strings.Contains(defaultScript, "Scenario.new") {
		t.Fatalf("defaultScenarioScript() missing Scenario.new: %q", defaultScript)
	}

	r := httptest.NewRequest(http.MethodGet, "/app/scenarios", nil)
	if !shouldPrefillScenarioScript(r, func(*http.Request) bool { return false }) {
		t.Fatal("expected non-HTMX requests to prefill scenario script")
	}
	if shouldPrefillScenarioScript(r, func(*http.Request) bool { return true }) {
		t.Fatal("expected HTMX request without prefill=1 to not prefill")
	}
	reqWithPrefill := httptest.NewRequest(http.MethodGet, "/app/scenarios?prefill=1", nil)
	if !shouldPrefillScenarioScript(reqWithPrefill, func(*http.Request) bool { return true }) {
		t.Fatal("expected HTMX request with prefill=1 to prefill")
	}

	logs := "line one\ncampaign created: id=camp-123 name=demo\nline three"
	if got := parseScenarioCampaignID(logs); got != "camp-123" {
		t.Fatalf("parseScenarioCampaignID() = %q", got)
	}
	if got := parseScenarioCampaignID("no campaign line"); got != "" {
		t.Fatalf("parseScenarioCampaignID(no match) = %q", got)
	}
}

func TestScenarioHelpersOriginChecks(t *testing.T) {
	loc := i18n.Printer(i18n.Default())

	req := httptest.NewRequest(http.MethodPost, "http://example.com/app/scenarios", nil)
	req.Host = "example.com"
	req.Header.Set("Origin", "http://example.com")
	rec := httptest.NewRecorder()
	if !requireSameOrigin(rec, req, loc) {
		t.Fatal("requireSameOrigin() rejected valid Origin")
	}

	req = httptest.NewRequest(http.MethodPost, "http://example.com/app/scenarios", nil)
	req.Host = "example.com"
	req.Header.Set("Referer", "http://example.com/app/scenarios")
	rec = httptest.NewRecorder()
	if !requireSameOrigin(rec, req, loc) {
		t.Fatal("requireSameOrigin() rejected valid Referer")
	}

	req = httptest.NewRequest(http.MethodPost, "http://example.com/app/scenarios", nil)
	req.Host = "example.com"
	rec = httptest.NewRecorder()
	if requireSameOrigin(rec, req, loc) {
		t.Fatal("requireSameOrigin() accepted missing Origin/Referer")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("requireSameOrigin() status = %d", rec.Code)
	}

	if requireSameOrigin(httptest.NewRecorder(), nil, loc) {
		t.Fatal("requireSameOrigin() accepted nil request")
	}

	request := httptest.NewRequest(http.MethodGet, "http://example.com/page", nil)
	request.Host = "example.com"
	if !sameOrigin("http://example.com", request) {
		t.Fatal("sameOrigin() rejected matching origin")
	}
	if !sameOrigin("http://EXAMPLE.COM", request) {
		t.Fatal("sameOrigin() should be case-insensitive")
	}
	if sameOrigin("http://other.example.com", request) {
		t.Fatal("sameOrigin() accepted different host")
	}
	if sameOrigin("null", request) {
		t.Fatal("sameOrigin() accepted null origin")
	}
	if sameOrigin("://bad", request) {
		t.Fatal("sameOrigin() accepted invalid URL")
	}
	if sameOrigin("http://example.com", nil) {
		t.Fatal("sameOrigin() accepted nil request")
	}

	defaultReq := httptest.NewRequest(http.MethodGet, "/", nil)
	if got := requestScheme(defaultReq); got != "http" {
		t.Fatalf("requestScheme(default) = %q", got)
	}
	fwdReq := httptest.NewRequest(http.MethodGet, "/", nil)
	fwdReq.Header.Set("X-Forwarded-Proto", "https, http")
	if got := requestScheme(fwdReq); got != "https" {
		t.Fatalf("requestScheme(X-Forwarded-Proto) = %q", got)
	}
	if got := requestScheme(nil); got != "http" {
		t.Fatalf("requestScheme(nil) = %q", got)
	}
}

func TestScenarioHelpersFiltersAndFormatting(t *testing.T) {
	loc := i18n.Printer(i18n.Default())
	now := timestamppb.New(time.Date(2026, time.March, 2, 15, 4, 5, 0, time.UTC))

	req := httptest.NewRequest(
		http.MethodGet,
		"/?session_id=s-1&event_type=created&actor_type=system&entity_type=character&start_date=2026-01-01&end_date=2026-12-31",
		nil,
	)
	filters := parseEventFilters(req)
	if filters.SessionID != "s-1" || filters.EventType != "created" || filters.EndDate != "2026-12-31" {
		t.Fatalf("parseEventFilters() = %#v", filters)
	}

	expression := buildEventFilterExpression(templates.EventFilterOptions{
		SessionID:  "s-1",
		EventType:  `a"b\c`,
		ActorType:  "participant",
		EntityType: "character",
		StartDate:  "2026-01-01",
		EndDate:    "2026-01-02",
	})
	if !strings.Contains(expression, `session_id = "s-1"`) || !strings.Contains(expression, `type = "a\"b\\c"`) {
		t.Fatalf("buildEventFilterExpression() = %q", expression)
	}
	if strings.Count(expression, " AND ") != 5 {
		t.Fatalf("buildEventFilterExpression() AND count = %d", strings.Count(expression, " AND "))
	}

	if escaped := escapeAIP160StringLiteral(`x"y\z`); escaped != `x\"y\\z` {
		t.Fatalf("escapeAIP160StringLiteral() = %q", escaped)
	}

	pushURL := eventFilterPushURL("/app/scenarios/camp-1/events", filters, "next-page")
	if !strings.Contains(pushURL, "page_token=next-page") || !strings.Contains(pushURL, "session_id=s-1") {
		t.Fatalf("eventFilterPushURL() = %q", pushURL)
	}

	if got := formatTimestamp(now); got != "2026-03-02 15:04:05" {
		t.Fatalf("formatTimestamp() = %q", got)
	}
	if got := formatTimestamp(nil); got != "" {
		t.Fatalf("formatTimestamp(nil) = %q", got)
	}

	if got := formatActorType("system", loc); got != loc.Sprintf("filter.actor.system") {
		t.Fatalf("formatActorType(system) = %q", got)
	}
	if got := formatActorType("other", loc); got != "other" {
		t.Fatalf("formatActorType(other) = %q", got)
	}
	if got := formatActorType("", loc); got != "" {
		t.Fatalf("formatActorType(empty) = %q", got)
	}

	if got := formatEventDescription(nil, loc); got != "" {
		t.Fatalf("formatEventDescription(nil) = %q", got)
	}
	if got := formatEventDescription(&statev1.Event{Type: "campaign.created"}, loc); got != loc.Sprintf("event.campaign_created") {
		t.Fatalf("formatEventDescription(event) = %q", got)
	}

	eventRows := buildEventRows([]*statev1.Event{
		nil,
		{
			CampaignId:  "camp-1",
			Seq:         9,
			Hash:        "hash-1",
			Type:        "campaign.created",
			Ts:          now,
			SessionId:   "session-1",
			ActorType:   "participant",
			EntityType:  "character",
			EntityId:    "char-1",
			PayloadJson: []byte(`{"k":"v"}`),
		},
	}, loc)
	if len(eventRows) != 1 || eventRows[0].Seq != 9 || eventRows[0].EntityID != "char-1" {
		t.Fatalf("buildEventRows() = %#v", eventRows)
	}
}

func TestScenarioHelpersTimeline(t *testing.T) {
	loc := i18n.Printer(i18n.Default())
	now := timestamppb.New(time.Date(2026, time.March, 2, 15, 4, 5, 0, time.UTC))

	fields := buildScenarioTimelineFields([]*statev1.ProjectionField{
		nil,
		{Label: "", Value: ""},
		{Label: "HP", Value: "5"},
	})
	if len(fields) != 1 || fields[0].Label != "HP" {
		t.Fatalf("buildScenarioTimelineFields() = %#v", fields)
	}
	if out := buildScenarioTimelineFields(nil); out != nil {
		t.Fatalf("buildScenarioTimelineFields(nil) = %#v", out)
	}

	entries := buildScenarioTimelineEntries([]*statev1.TimelineEntry{
		nil,
		{
			Seq:              7,
			EventType:        "session.started",
			EventTime:        now,
			IconId:           commonv1.IconId_ICON_ID_UNSPECIFIED,
			EventPayloadJson: `{"a":1}`,
			Projection: &statev1.ProjectionDisplay{
				Title:    "",
				Subtitle: "sub",
				Status:   "ACTIVE",
				Fields:   []*statev1.ProjectionField{{Label: "HP", Value: "5"}},
			},
		},
	}, loc)
	if len(entries) != 1 || entries[0].IconID != commonv1.IconId_ICON_ID_GENERIC {
		t.Fatalf("buildScenarioTimelineEntries() = %#v", entries)
	}
	if entries[0].Title == "" || entries[0].StatusBadge != "success" {
		t.Fatalf("buildScenarioTimelineEntries() unexpected entry: %#v", entries[0])
	}

	tests := []struct {
		status string
		want   string
	}{
		{status: "", want: "secondary"},
		{status: "ACTIVE", want: "success"},
		{status: "DRAFT", want: "warning"},
		{status: "COMPLETED", want: "success"},
		{status: "ARCHIVED", want: "neutral"},
		{status: "ENDED", want: "neutral"},
		{status: "UNKNOWN", want: "secondary"},
	}
	for _, tc := range tests {
		if got := timelineStatusBadgeVariant(tc.status); got != tc.want {
			t.Fatalf("timelineStatusBadgeVariant(%q) = %q, want %q", tc.status, got, tc.want)
		}
	}
}

func TestScenarioHelpersEventTypeFormatting(t *testing.T) {
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
}

func TestScenarioServiceGRPCAddr(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_ADDR", "")
	if got := (&service{}).scenarioGRPCAddr(); got != "localhost:8080" {
		t.Fatalf("scenarioGRPCAddr(default) = %q", got)
	}

	t.Setenv("FRACTURING_SPACE_GAME_ADDR", "env:9090")
	if got := (&service{}).scenarioGRPCAddr(); got != "env:9090" {
		t.Fatalf("scenarioGRPCAddr(env) = %q", got)
	}

	if got := (&service{grpcAddr: " configured:8080 "}).scenarioGRPCAddr(); got != " configured:8080 " {
		t.Fatalf("scenarioGRPCAddr(explicit) = %q", got)
	}

	var nilService *service
	if got := nilService.scenarioGRPCAddr(); got != "localhost:8080" {
		t.Fatalf("scenarioGRPCAddr(nil service) = %q", got)
	}
}
