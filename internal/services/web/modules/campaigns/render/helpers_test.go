package render

import "testing"

func TestCampaignInviteStatusLabel_Declined(t *testing.T) {
	t.Parallel()

	loc := testLocalizer{
		"game.campaign_invites.value_declined": "Declined",
	}

	if got := campaignInviteStatusLabel(loc, "declined"); got != "Declined" {
		t.Fatalf("campaignInviteStatusLabel(declined) = %q, want %q", got, "Declined")
	}
}

func TestSharedHelperLabelsAndAttrs(t *testing.T) {
	t.Parallel()

	loc := testLocalizer{
		"game.campaign.system_unspecified":           "Unspecified",
		"game.campaigns.system_daggerheart":          "Daggerheart",
		"game.create.field_gm_mode_human":            "Human",
		"game.create.field_gm_mode_ai":               "AI",
		"game.campaign.overview.theme_empty":         "No theme",
		"game.campaign.overview.value_active":        "Active",
		"game.campaign.overview.value_locale_pt_br":  "Portuguese (Brazil)",
		"game.campaign.overview.value_standard":      "Standard",
		"game.campaign.overview.value_private":       "Private",
		"game.campaign.ai_binding.status_configured": "Configured",
		"game.campaign.session_status_active":        "Active Session",
		"game.participants.value.gm":                 "GM",
		"game.participants.value.owner":              "Owner",
		"game.participants.value.human":              "Human",
		"game.participants.value_they_them":          "they/them",
		"game.characters.value_pc":                   "PC",
	}

	if got := campaignParticipantCardClass(ParticipantView{IsViewer: true}); got != "card bg-base-100 border border-primary shadow-sm md:card-side" {
		t.Fatalf("campaignParticipantCardClass(viewer) = %q", got)
	}
	if got := campaignParticipantViewerAttr(ParticipantView{IsViewer: true}); got != "true" {
		t.Fatalf("campaignParticipantViewerAttr() = %q, want true", got)
	}
	if got := campaignCharacterCardClass(CharacterView{OwnedByViewer: true}); got != "card bg-base-100 border border-primary shadow-sm md:card-side" {
		t.Fatalf("campaignCharacterCardClass(owner) = %q", got)
	}
	if got := campaignCharacterOwnedByViewerAttr(CharacterView{OwnedByViewer: false}); got != "false" {
		t.Fatalf("campaignCharacterOwnedByViewerAttr() = %q, want false", got)
	}
	if got := campaignCharacterAliases([]string{"Aria", "Nyx"}); got != "Aria, Nyx" {
		t.Fatalf("campaignCharacterAliases() = %q", got)
	}
	if got := campaignSystemLabel(loc, "daggerheart"); got != "Daggerheart" {
		t.Fatalf("campaignSystemLabel() = %q", got)
	}
	if got := campaignGMModeLabel(loc, "human"); got != "Human" {
		t.Fatalf("campaignGMModeLabel() = %q", got)
	}
	if !campaignActionsLocked(true) {
		t.Fatal("campaignActionsLocked(true) = false")
	}
	if got := campaignParticipantRoleLabel(loc, "gm"); got != "GM" {
		t.Fatalf("campaignParticipantRoleLabel() = %q", got)
	}
	if got := campaignParticipantAccessLabel(loc, "owner"); got != "Owner" {
		t.Fatalf("campaignParticipantAccessLabel() = %q", got)
	}
	if got := campaignParticipantControllerLabel(loc, "human"); got != "Human" {
		t.Fatalf("campaignParticipantControllerLabel() = %q", got)
	}
	if got := participantPronounsLabel(loc, "they/them"); got != "they/them" {
		t.Fatalf("participantPronounsLabel() = %q", got)
	}
	if got := campaignCharacterKindLabel(loc, "pc"); got != "PC" {
		t.Fatalf("campaignCharacterKindLabel() = %q", got)
	}
	if got := campaignOverviewTheme(loc, ""); got != "No theme" {
		t.Fatalf("campaignOverviewTheme() = %q", got)
	}
	if got := campaignOverviewStatus(loc, "active"); got != "Active" {
		t.Fatalf("campaignOverviewStatus() = %q", got)
	}
	if got := campaignOverviewLocale(loc, "pt_br"); got != "Portuguese (Brazil)" {
		t.Fatalf("campaignOverviewLocale() = %q", got)
	}
	if got := campaignOverviewIntent(loc, "standard"); got != "Standard" {
		t.Fatalf("campaignOverviewIntent() = %q", got)
	}
	if got := campaignOverviewAccessPolicy(loc, "private"); got != "Private" {
		t.Fatalf("campaignOverviewAccessPolicy() = %q", got)
	}
	if got := campaignOverviewAIBindingStatus(loc, "configured"); got != "Configured" {
		t.Fatalf("campaignOverviewAIBindingStatus() = %q", got)
	}
	if got := campaignSessionStatusLabel(loc, "active"); got != "Active Session" {
		t.Fatalf("campaignSessionStatusLabel() = %q", got)
	}
	if !campaignSessionCanEnd("active") {
		t.Fatal("campaignSessionCanEnd(active) = false")
	}
	if !campaignSessionStartReady(SessionReadinessView{Ready: true}) {
		t.Fatal("campaignSessionStartReady() = false")
	}
	if !campaignInviteCreateReady(InviteCreatePageView{InviteSeatOptions: []InviteSeatOptionView{{ParticipantID: "p-1", Label: "Rook"}}}) {
		t.Fatal("campaignInviteCreateReady(interactive) = false")
	}
	if campaignInviteCreateReady(InviteCreatePageView{CampaignDetailBaseView: CampaignDetailBaseView{ActionsLocked: true}, InviteSeatOptions: []InviteSeatOptionView{{ParticipantID: "p-1", Label: "Rook"}}}) {
		t.Fatal("campaignInviteCreateReady(locked) = true")
	}
}

func TestRenderHelperNormalizationContracts(t *testing.T) {
	t.Parallel()

	loc := testLocalizer{
		"game.participants.value_unassigned":           "Unassigned",
		"game.participants.value_they_them":            "they/them",
		"game.participants.value_he_him":               "he/him",
		"game.participants.value_she_her":              "she/her",
		"game.participants.value_it_its":               "it/its",
		"game.character_detail.title":                  "Character",
		"game.character_detail.character_sheet_suffix": "Character Sheet",
		"game.campaigns.system_daggerheart":            "Daggerheart",
	}

	if got := campaignCharacterDetailURL("camp-1", CharacterView{ID: "char-1"}); got != "/app/campaigns/camp-1/characters/char-1" {
		t.Fatalf("campaignCharacterDetailURL() = %q", got)
	}
	if got := campaignCharacterEditURL("camp-1", "char-1"); got != "/app/campaigns/camp-1/characters/char-1/edit" {
		t.Fatalf("campaignCharacterEditURL() = %q", got)
	}
	if got := campaignCharacterSheetTitle(loc, "daggerheart"); got != "Daggerheart Character Sheet" {
		t.Fatalf("campaignCharacterSheetTitle() = %q", got)
	}
	if !campaignCharacterHasDaggerheartSummary(CharacterView{
		Daggerheart: &CharacterDaggerheartSummaryView{
			Level:         2,
			ClassName:     "Warrior",
			SubclassName:  "Guardian",
			HeritageName:  "Drakona",
			CommunityName: "Wanderborne",
		},
	}) {
		t.Fatal("campaignCharacterHasDaggerheartSummary() = false")
	}
	if got := campaignCharacterDaggerheartLevelAttr(CharacterView{
		Daggerheart: &CharacterDaggerheartSummaryView{
			Level:         2,
			ClassName:     "Warrior",
			SubclassName:  "Guardian",
			HeritageName:  "Drakona",
			CommunityName: "Wanderborne",
		},
	}); got != "2" {
		t.Fatalf("campaignCharacterDaggerheartLevelAttr() = %q", got)
	}
	if got := campaignCharacterOwnershipOptionLabel(loc, CharacterOwnershipOptionView{}); got != "Unassigned" {
		t.Fatalf("campaignCharacterOwnershipOptionLabel() = %q", got)
	}
	if got := campaignCharacterDisplayName(loc, CharacterView{}); got != "Character" {
		t.Fatalf("campaignCharacterDisplayName() = %q", got)
	}
	presets := campaignCharacterPronounPresets(loc)
	if len(presets) != 4 || presets[0] != "they/them" || presets[3] != "it/its" {
		t.Fatalf("campaignCharacterPronounPresets() = %#v", presets)
	}

	if got := campaignParticipantEditURL("camp-1", ParticipantView{ID: "p-1"}); got != "/app/campaigns/camp-1/participants/p-1/edit" {
		t.Fatalf("campaignParticipantEditURL() = %q", got)
	}
	if got := campaignParticipantControllerCanonical("controller_ai"); got != "ai" {
		t.Fatalf("campaignParticipantControllerCanonical() = %q", got)
	}
	if got := campaignParticipantRoleCanonical("participant_role_gm"); got != "gm" {
		t.Fatalf("campaignParticipantRoleCanonical() = %q", got)
	}
	if got := campaignParticipantRoleFormValue(ParticipantEditorView{}); got != "gm" {
		t.Fatalf("campaignParticipantRoleFormValue() = %q", got)
	}
	if got := campaignParticipantAccessCanonical("campaign_access_owner"); got != "owner" {
		t.Fatalf("campaignParticipantAccessCanonical() = %q", got)
	}
	if got := campaignParticipantAccessFormValue(ParticipantEditorView{}); got != "member" {
		t.Fatalf("campaignParticipantAccessFormValue() = %q", got)
	}
	presets = campaignParticipantPronounPresets(loc, ParticipantEditorView{Controller: "ai"})
	if len(presets) != 4 || presets[3] != "it/its" {
		t.Fatalf("campaignParticipantPronounPresets(ai) = %#v", presets)
	}

	if got := campaignSessionByID(nil, "s-2", []SessionView{{ID: "s-1"}, {ID: "s-2", Name: "Found"}}); got.Name != "Found" {
		t.Fatalf("campaignSessionByID() = %#v", got)
	}
	if !campaignAIBindingCurrentUnbound(AIBindingSettingsView{}) {
		t.Fatal("campaignAIBindingCurrentUnbound(empty) = false")
	}
}
