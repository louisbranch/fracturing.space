package campaigns

import (
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	campaignrender "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/render"
)

// mapCharactersView converts domain characters to template view items.
func mapCharactersView(items []campaignapp.CampaignCharacter) []campaignrender.CharacterView {
	result := make([]campaignrender.CharacterView, 0, len(items))
	for _, c := range items {
		result = append(result, campaignrender.CharacterView{
			ID:                      c.ID,
			Name:                    c.Name,
			Kind:                    c.Kind,
			Controller:              c.Controller,
			ControllerParticipantID: c.ControllerParticipantID,
			Pronouns:                c.Pronouns,
			Aliases:                 append([]string(nil), c.Aliases...),
			AvatarURL:               c.AvatarURL,
			CanEdit:                 c.CanEdit,
			EditReasonCode:          c.EditReasonCode,
			Daggerheart:             mapCharacterDaggerheartSummaryView(c.Daggerheart),
		})
	}
	return result
}

// mapCharacterDaggerheartSummaryView copies the optional Daggerheart card
// summary into the template-facing view model.
func mapCharacterDaggerheartSummaryView(summary *campaignapp.CampaignCharacterDaggerheartSummary) *campaignrender.CharacterDaggerheartSummaryView {
	if summary == nil {
		return nil
	}
	return &campaignrender.CharacterDaggerheartSummaryView{
		Level:         summary.Level,
		ClassName:     summary.ClassName,
		SubclassName:  summary.SubclassName,
		AncestryName:  summary.AncestryName,
		CommunityName: summary.CommunityName,
	}
}

// mapCharacterEditorView converts domain character editor state to template view state.
func mapCharacterEditorView(editor campaignapp.CampaignCharacterEditor) campaignrender.CharacterEditorView {
	return campaignrender.CharacterEditorView{
		ID:       editor.Character.ID,
		Name:     editor.Character.Name,
		Pronouns: editor.Character.Pronouns,
		Kind:     editor.Character.Kind,
	}
}

// mapCharacterControlView converts domain control state to template view state.
func mapCharacterControlView(control campaignapp.CampaignCharacterControl) campaignrender.CharacterControlView {
	options := make([]campaignrender.CharacterControlOptionView, 0, len(control.Options))
	for _, option := range control.Options {
		options = append(options, campaignrender.CharacterControlOptionView{
			ParticipantID: option.ParticipantID,
			Label:         option.Label,
			Selected:      option.Selected,
		})
	}
	return campaignrender.CharacterControlView{
		CurrentParticipantName: control.CurrentParticipantName,
		CanSelfClaim:           control.CanSelfClaim,
		CanSelfRelease:         control.CanSelfRelease,
		CanManageControl:       control.CanManageControl,
		Options:                options,
	}
}
