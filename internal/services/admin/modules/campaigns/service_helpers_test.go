package campaigns

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	"github.com/louisbranch/fracturing.space/internal/services/admin/modules/eventview"
	"github.com/louisbranch/fracturing.space/internal/services/admin/templates"
	"github.com/louisbranch/fracturing.space/internal/services/shared/i18nhttp"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

func TestCampaignHelpersFormatters(t *testing.T) {
	loc := i18nhttp.Printer(i18nhttp.Default())

	if got := formatGmMode(statev1.GmMode_HUMAN, loc); got != loc.Sprintf("label.human") {
		t.Fatalf("formatGmMode(HUMAN) = %q", got)
	}
	if got := formatGmMode(statev1.GmMode_GM_MODE_UNSPECIFIED, loc); got != loc.Sprintf("label.unspecified") {
		t.Fatalf("formatGmMode(unspecified) = %q", got)
	}

	if got := formatGameSystem(commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART, loc); got != loc.Sprintf("label.daggerheart") {
		t.Fatalf("formatGameSystem(daggerheart) = %q", got)
	}
	if got := formatGameSystem(commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED, loc); got != loc.Sprintf("label.unspecified") {
		t.Fatalf("formatGameSystem(unspecified) = %q", got)
	}

	if got := formatSessionStatus(statev1.SessionStatus_SESSION_ACTIVE, loc); got != loc.Sprintf("label.active") {
		t.Fatalf("formatSessionStatus(active) = %q", got)
	}
	if got := formatSessionStatus(statev1.SessionStatus_SESSION_STATUS_UNSPECIFIED, loc); got != loc.Sprintf("label.unspecified") {
		t.Fatalf("formatSessionStatus(unspecified) = %q", got)
	}

	label, variant := formatInviteStatus(invitev1.InviteStatus_PENDING, loc)
	if label != loc.Sprintf("label.invite_pending") || variant != "warning" {
		t.Fatalf("formatInviteStatus(pending) = (%q,%q)", label, variant)
	}
	label, variant = formatInviteStatus(invitev1.InviteStatus_INVITE_STATUS_UNSPECIFIED, loc)
	if label != loc.Sprintf("label.unspecified") || variant != "secondary" {
		t.Fatalf("formatInviteStatus(unspecified) = (%q,%q)", label, variant)
	}

	ts := timestamppb.New(time.Date(2026, time.March, 2, 15, 4, 5, 0, time.UTC))
	if got := formatCreatedDate(ts); got != "2026-03-02" {
		t.Fatalf("formatCreatedDate() = %q", got)
	}
	if got := formatCreatedDate(nil); got != "" {
		t.Fatalf("formatCreatedDate(nil) = %q", got)
	}
	if got := eventview.FormatTimestamp(ts); got != "2026-03-02 15:04:05" {
		t.Fatalf("eventview.FormatTimestamp() = %q", got)
	}
	if got := eventview.FormatTimestamp(nil); got != "" {
		t.Fatalf("eventview.FormatTimestamp(nil) = %q", got)
	}

	if got := truncateText("hello world", 5); got != "hello..." {
		t.Fatalf("truncateText() = %q", got)
	}
	if got := truncateText("hello", 0); got != "" {
		t.Fatalf("truncateText(limit=0) = %q", got)
	}

	label, variant = formatParticipantRole(statev1.ParticipantRole_GM, loc)
	if label != loc.Sprintf("label.gm") || variant != "info" {
		t.Fatalf("formatParticipantRole(GM) = (%q,%q)", label, variant)
	}
	label, variant = formatParticipantRole(statev1.ParticipantRole_ROLE_UNSPECIFIED, loc)
	if label != loc.Sprintf("label.unspecified") || variant != "secondary" {
		t.Fatalf("formatParticipantRole(unspecified) = (%q,%q)", label, variant)
	}

	label, variant = formatParticipantController(statev1.Controller_CONTROLLER_HUMAN, loc)
	if label != loc.Sprintf("label.human") || variant != "success" {
		t.Fatalf("formatParticipantController(human) = (%q,%q)", label, variant)
	}
	label, variant = formatParticipantController(statev1.Controller_CONTROLLER_UNSPECIFIED, loc)
	if label != loc.Sprintf("label.unspecified") || variant != "secondary" {
		t.Fatalf("formatParticipantController(unspecified) = (%q,%q)", label, variant)
	}

	label, variant = formatParticipantAccess(statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER, loc)
	if label != loc.Sprintf("label.owner") || variant != "warning" {
		t.Fatalf("formatParticipantAccess(owner) = (%q,%q)", label, variant)
	}
	label, variant = formatParticipantAccess(statev1.CampaignAccess_CAMPAIGN_ACCESS_UNSPECIFIED, loc)
	if label != loc.Sprintf("label.unspecified") || variant != "secondary" {
		t.Fatalf("formatParticipantAccess(unspecified) = (%q,%q)", label, variant)
	}

	if got := formatCharacterController(nil, nil, loc); got != loc.Sprintf("label.unassigned") {
		t.Fatalf("formatCharacterController(nil) = %q", got)
	}
	if got := formatCharacterController(&statev1.Character{}, nil, loc); got != loc.Sprintf("label.unassigned") {
		t.Fatalf("formatCharacterController(no participant) = %q", got)
	}
	if got := formatCharacterController(
		&statev1.Character{OwnerParticipantId: wrapperspb.String("p-1")},
		map[string]string{"p-1": "Alice"},
		loc,
	); got != "Alice" {
		t.Fatalf("formatCharacterController(found) = %q", got)
	}
	if got := formatCharacterController(
		&statev1.Character{OwnerParticipantId: wrapperspb.String("p-x")},
		map[string]string{},
		loc,
	); got != loc.Sprintf("label.unknown") {
		t.Fatalf("formatCharacterController(unknown) = %q", got)
	}

	if got := formatCharacterKind(statev1.CharacterKind_PC, loc); got != loc.Sprintf("label.pc") {
		t.Fatalf("formatCharacterKind(pc) = %q", got)
	}
	if got := formatCharacterKind(statev1.CharacterKind_CHARACTER_KIND_UNSPECIFIED, loc); got != loc.Sprintf("label.unspecified") {
		t.Fatalf("formatCharacterKind(unspecified) = %q", got)
	}
}

func TestCampaignHelpersBuilders(t *testing.T) {
	loc := i18nhttp.Printer(i18nhttp.Default())
	now := timestamppb.New(time.Date(2026, time.March, 2, 15, 4, 5, 0, time.UTC))

	rows := buildCampaignRows([]*statev1.Campaign{
		nil,
		{
			Id:               "camp-1",
			Name:             "Campaign",
			System:           commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			GmMode:           statev1.GmMode_HUMAN,
			ParticipantCount: 2,
			CharacterCount:   3,
			ThemePrompt:      "A very long prompt that will be truncated in compact tables",
			CreatedAt:        now,
		},
	}, loc)
	if len(rows) != 1 || rows[0].ID != "camp-1" || rows[0].ParticipantCount != "2" {
		t.Fatalf("buildCampaignRows() unexpected rows: %#v", rows)
	}

	detail := buildCampaignDetail(&statev1.Campaign{
		Id:               "camp-1",
		Name:             "Campaign",
		System:           commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		GmMode:           statev1.GmMode_AI,
		ParticipantCount: 1,
		CharacterCount:   2,
		ThemePrompt:      "full prompt",
		CreatedAt:        now,
		UpdatedAt:        now,
	}, loc)
	if detail.ID != "camp-1" || detail.ThemePrompt != "full prompt" {
		t.Fatalf("buildCampaignDetail() unexpected detail: %#v", detail)
	}
	if empty := buildCampaignDetail(nil, loc); empty.ID != "" {
		t.Fatalf("buildCampaignDetail(nil) = %#v", empty)
	}

	sessionRows := buildCampaignSessionRows([]*statev1.Session{
		{Id: "s-1", CampaignId: "camp-1", Status: statev1.SessionStatus_SESSION_ACTIVE, StartedAt: now},
		{Id: "s-2", CampaignId: "camp-1", Status: statev1.SessionStatus_SESSION_ENDED, StartedAt: now, EndedAt: now},
	}, loc)
	if len(sessionRows) != 2 || sessionRows[0].StatusBadge != "success" || sessionRows[1].StatusBadge != "secondary" {
		t.Fatalf("buildCampaignSessionRows() unexpected rows: %#v", sessionRows)
	}

	charRows := buildCharacterRows([]*statev1.Character{
		{
			Id:                 "char-1",
			CampaignId:         "camp-1",
			Name:               "Hero",
			Kind:               statev1.CharacterKind_PC,
			OwnerParticipantId: wrapperspb.String("p-1"),
		},
	}, map[string]string{"p-1": "Alice"}, loc)
	if len(charRows) != 1 || charRows[0].Controller != "Alice" {
		t.Fatalf("buildCharacterRows() unexpected rows: %#v", charRows)
	}

	sheet := buildCharacterSheet(
		"camp-1",
		"Campaign",
		&statev1.Character{Id: "char-1", Name: "Hero", CreatedAt: now, UpdatedAt: now},
		[]templates.EventRow{{Seq: 7}},
		"Alice",
		loc,
	)
	if sheet.CampaignID != "camp-1" || len(sheet.RecentEvents) != 1 {
		t.Fatalf("buildCharacterSheet() unexpected sheet: %#v", sheet)
	}

	inviteRows := buildInviteRows([]*invitev1.Invite{
		{
			Id:              "inv-1",
			CampaignId:      "camp-1",
			ParticipantId:   "p-1",
			RecipientUserId: "u-1",
			Status:          invitev1.InviteStatus_CLAIMED,
			CreatedAt:       now,
			UpdatedAt:       now,
		},
		{
			Id:            "inv-2",
			ParticipantId: "missing",
			Status:        invitev1.InviteStatus_REVOKED,
		},
	}, map[string]string{"p-1": "Alice"}, map[string]string{"u-1": "Bob"}, loc)
	if len(inviteRows) != 2 || inviteRows[0].Participant != "Alice" || inviteRows[0].Recipient != "Bob" {
		t.Fatalf("buildInviteRows() unexpected rows: %#v", inviteRows)
	}
	if inviteRows[1].Participant != loc.Sprintf("label.unknown") || inviteRows[1].Recipient != loc.Sprintf("label.unassigned") {
		t.Fatalf("buildInviteRows() fallback mismatch: %#v", inviteRows[1])
	}

	participantRows := buildParticipantRows([]*statev1.Participant{
		{
			Id:             "p-1",
			Name:           "Alice",
			Role:           statev1.ParticipantRole_GM,
			CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_OWNER,
			Controller:     statev1.Controller_CONTROLLER_HUMAN,
			CreatedAt:      now,
		},
	}, loc)
	if len(participantRows) != 1 || participantRows[0].Name != "Alice" {
		t.Fatalf("buildParticipantRows() unexpected rows: %#v", participantRows)
	}

	sessionDetail := buildSessionDetail(
		"camp-1",
		"Campaign",
		&statev1.Session{Id: "s-1", Name: "Session", Status: statev1.SessionStatus_SESSION_ACTIVE, StartedAt: now},
		9,
		loc,
	)
	if sessionDetail.StatusBadge != "success" || sessionDetail.EventCount != 9 {
		t.Fatalf("buildSessionDetail(active) = %#v", sessionDetail)
	}
	if empty := buildSessionDetail("camp-1", "Campaign", nil, 0, loc); empty.ID != "" {
		t.Fatalf("buildSessionDetail(nil) = %#v", empty)
	}

	eventRows := eventview.BuildEventRows([]*statev1.Event{
		nil,
		{
			CampaignId:  "camp-1",
			Seq:         5,
			Hash:        "hash-1",
			Type:        "campaign.created",
			Ts:          now,
			SessionId:   "s-1",
			ActorType:   "participant",
			EntityType:  "character",
			EntityId:    "char-1",
			PayloadJson: []byte(`{"k":"v"}`),
		},
	}, loc)
	if len(eventRows) != 1 || eventRows[0].Seq != 5 || eventRows[0].EntityID != "char-1" {
		t.Fatalf("eventview.BuildEventRows() unexpected rows: %#v", eventRows)
	}
}

func TestCampaignHelpersEventFilters(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodGet,
		"/?session_id=s-1&event_type=created&actor_type=system&entity_type=character&start_date=2026-01-01&end_date=2026-12-31",
		nil,
	)
	filters := eventview.ParseEventFilters(req)
	if filters.SessionID != "s-1" || filters.EventType != "created" || filters.EndDate != "2026-12-31" {
		t.Fatalf("eventview.ParseEventFilters() = %#v", filters)
	}

	got := eventview.BuildEventFilterExpression(templates.EventFilterOptions{
		SessionID:  "s-1",
		EventType:  `a"b\c`,
		ActorType:  "participant",
		EntityType: "character",
		StartDate:  "2026-01-01",
		EndDate:    "2026-01-02",
	})
	if !strings.Contains(got, `session_id = "s-1"`) || !strings.Contains(got, `type = "a\"b\\c"`) {
		t.Fatalf("eventview.BuildEventFilterExpression() = %q", got)
	}
	if strings.Count(got, " AND ") != 5 {
		t.Fatalf("eventview.BuildEventFilterExpression() AND count = %d", strings.Count(got, " AND "))
	}

	if escaped := eventview.EscapeAIP160StringLiteral(`x"y\z`); escaped != `x\"y\\z` {
		t.Fatalf("eventview.EscapeAIP160StringLiteral() = %q", escaped)
	}

	pushURL := eventview.EventFilterPushURL("/app/campaigns/camp-1/events", filters, "p-1")
	if !strings.Contains(pushURL, "page_token=p-1") || !strings.Contains(pushURL, "session_id=s-1") {
		t.Fatalf("eventview.EventFilterPushURL() = %q", pushURL)
	}
}

func TestCampaignHelpersEventTypeFormatting(t *testing.T) {
	loc := i18nhttp.Printer(i18nhttp.Default())
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
			if got := eventview.FormatEventType(tc.input, loc); got != tc.want {
				t.Fatalf("eventview.FormatEventType(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}

	actorTests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "empty", input: "", want: ""},
		{name: "system", input: "system", want: loc.Sprintf("filter.actor.system")},
		{name: "participant", input: "participant", want: loc.Sprintf("filter.actor.participant")},
		{name: "gm", input: "gm", want: loc.Sprintf("filter.actor.gm")},
		{name: "unknown", input: "other", want: "other"},
	}
	for _, tc := range actorTests {
		t.Run("actor_"+tc.name, func(t *testing.T) {
			if got := eventview.FormatActorType(tc.input, loc); got != tc.want {
				t.Fatalf("eventview.FormatActorType(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}

	if got := eventview.FormatEventDescription(nil, loc); got != "" {
		t.Fatalf("eventview.FormatEventDescription(nil) = %q", got)
	}
	if got := eventview.FormatEventDescription(&statev1.Event{Type: "campaign.created"}, loc); got != loc.Sprintf("event.campaign_created") {
		t.Fatalf("eventview.FormatEventDescription(event) = %q", got)
	}
}
