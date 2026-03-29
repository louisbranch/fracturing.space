package render

import "testing"

func TestCampaignRenderOverviewHelperBranches(t *testing.T) {
	t.Parallel()

	loc := testLocalizer{
		"game.campaign.system_unspecified":             "Unspecified",
		"game.campaign.overview.value_draft":           "Draft",
		"game.campaign.overview.value_completed":       "Completed",
		"game.campaign.overview.value_archived":        "Archived",
		"game.campaign.overview.value_locale_en_us":    "English (US)",
		"game.campaign.overview.value_starter":         "Starter",
		"game.campaign.overview.value_sandbox":         "Sandbox",
		"game.campaign.overview.value_restricted":      "Restricted",
		"game.campaign.overview.value_public":          "Public",
		"game.campaign.ai_binding.status_pending":      "Pending",
		"game.campaign.ai_binding.status_not_required": "Not required",
		"game.create.field_gm_mode_ai":                 "AI",
		"game.create.field_gm_mode_hybrid":             "Hybrid",
		"game.campaign.gm_mode_unspecified":            "GM unspecified",
		"game.campaign.session_status_unspecified":     "Session unspecified",
		"game.campaign.session_status_ended":           "Ended",
		"game.participants.value_unassigned":           "Unassigned",
	}

	if got := campaignOverviewStatus(loc, "draft"); got != "Draft" {
		t.Fatalf("campaignOverviewStatus(draft) = %q", got)
	}
	if got := campaignOverviewStatus(loc, "completed"); got != "Completed" {
		t.Fatalf("campaignOverviewStatus(completed) = %q", got)
	}
	if got := campaignOverviewStatus(loc, "archived"); got != "Archived" {
		t.Fatalf("campaignOverviewStatus(archived) = %q", got)
	}
	if got := campaignOverviewStatus(loc, " custom "); got != "custom" {
		t.Fatalf("campaignOverviewStatus(custom) = %q", got)
	}

	if got := campaignOverviewLocale(loc, ""); got != "Unspecified" {
		t.Fatalf("campaignOverviewLocale(empty) = %q", got)
	}
	if got := campaignOverviewLocale(loc, "en_us"); got != "English (US)" {
		t.Fatalf("campaignOverviewLocale(en_us) = %q", got)
	}
	if got := campaignOverviewLocale(loc, "es_mx"); got != "es_mx" {
		t.Fatalf("campaignOverviewLocale(es_mx) = %q", got)
	}

	if got := campaignOverviewIntent(loc, "starter"); got != "Starter" {
		t.Fatalf("campaignOverviewIntent(starter) = %q", got)
	}
	if got := campaignOverviewIntent(loc, "sandbox"); got != "Sandbox" {
		t.Fatalf("campaignOverviewIntent(sandbox) = %q", got)
	}
	if got := campaignOverviewIntent(loc, "experimental"); got != "experimental" {
		t.Fatalf("campaignOverviewIntent(experimental) = %q", got)
	}

	if got := campaignOverviewAccessPolicy(loc, "restricted"); got != "Restricted" {
		t.Fatalf("campaignOverviewAccessPolicy(restricted) = %q", got)
	}
	if got := campaignOverviewAccessPolicy(loc, "public"); got != "Public" {
		t.Fatalf("campaignOverviewAccessPolicy(public) = %q", got)
	}
	if got := campaignOverviewAccessPolicy(loc, "friends"); got != "friends" {
		t.Fatalf("campaignOverviewAccessPolicy(friends) = %q", got)
	}

	if got := campaignOverviewAIBindingStatus(loc, "pending"); got != "Pending" {
		t.Fatalf("campaignOverviewAIBindingStatus(pending) = %q", got)
	}
	if got := campaignOverviewAIBindingStatus(loc, ""); got != "Not required" {
		t.Fatalf("campaignOverviewAIBindingStatus(empty) = %q", got)
	}

	if got := campaignGMModeLabel(loc, ""); got != "GM unspecified" {
		t.Fatalf("campaignGMModeLabel(empty) = %q", got)
	}
	if got := campaignGMModeLabel(loc, "ai"); got != "AI" {
		t.Fatalf("campaignGMModeLabel(ai) = %q", got)
	}
	if got := campaignGMModeLabel(loc, "hybrid"); got != "Hybrid" {
		t.Fatalf("campaignGMModeLabel(hybrid) = %q", got)
	}
	if got := campaignGMModeLabel(loc, "guided"); got != "guided" {
		t.Fatalf("campaignGMModeLabel(guided) = %q", got)
	}

	if got := campaignSessionStatusLabel(loc, ""); got != "Session unspecified" {
		t.Fatalf("campaignSessionStatusLabel(empty) = %q", got)
	}
	if got := campaignSessionStatusLabel(loc, "ended"); got != "Ended" {
		t.Fatalf("campaignSessionStatusLabel(ended) = %q", got)
	}
	if got := campaignSessionStatusLabel(loc, "paused"); got != "paused" {
		t.Fatalf("campaignSessionStatusLabel(paused) = %q", got)
	}
}

func TestCampaignRenderSessionAndBindingHelpersFallbacks(t *testing.T) {
	t.Parallel()

	loc := testLocalizer{
		"game.participants.value_unassigned": "Unassigned",
	}

	if campaignSessionCanEnd(" ended ") {
		t.Fatal("campaignSessionCanEnd(ended) = true")
	}

	view := SessionCreatePageView{
		CharacterControllers: []SessionCreateCharacterControllerView{
			{Options: []SessionCreateControllerOptionView{{ParticipantID: "p-1", Selected: true}}},
			{Options: []SessionCreateControllerOptionView{{ParticipantID: "", Selected: true}}},
		},
	}
	if sessionCreateHasCompleteControllerAssignments(view) {
		t.Fatal("sessionCreateHasCompleteControllerAssignments() = true")
	}

	if got := campaignSessionControllerOptionLabel(loc, SessionCreateControllerOptionView{}); got != "Unassigned" {
		t.Fatalf("campaignSessionControllerOptionLabel(empty) = %q", got)
	}
	if got := campaignSessionControllerOptionLabel(loc, SessionCreateControllerOptionView{ParticipantID: "p-2"}); got != "p-2" {
		t.Fatalf("campaignSessionControllerOptionLabel(fallback) = %q", got)
	}

	if got := campaignSessionByID(nil, "", []SessionView{{ID: "s-1"}}); got.ID != "" {
		t.Fatalf("campaignSessionByID(empty) = %#v", got)
	}
	if got := campaignSessionByID(nil, "missing", []SessionView{{ID: "s-1"}}); got.ID != "" {
		t.Fatalf("campaignSessionByID(missing) = %#v", got)
	}

	if campaignAIBindingCurrentUnbound(AIBindingSettingsView{CurrentID: "agent-1"}) {
		t.Fatal("campaignAIBindingCurrentUnbound(bound) = true")
	}
}
