package orchestration

import (
	"context"
	"fmt"
)

// DaggerheartContextSources returns Daggerheart-specific context sources.
// These inject authoritative game-system rules and character state into the
// prompt so the GM agent does not need to discover them through tool calls.
func DaggerheartContextSources() []ContextSource {
	return []ContextSource{
		ContextSourceFunc(dualityRulesContextSource),
		ContextSourceFunc(characterStateContextSource),
	}
}

// characterStateContextSource reads the campaign snapshot and returns a brief
// section with per-character HP/stress/armor/hope/conditions/life_state and
// campaign-level GM Fear. This gives the GM agent a tactical dashboard for
// informed narration decisions without tool calls.
func characterStateContextSource(ctx context.Context, sess Session, input PromptInput) (BriefContribution, error) {
	uri := fmt.Sprintf("daggerheart://campaign/%s/snapshot", input.CampaignID)
	data, err := sess.ReadResource(ctx, uri)
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read daggerheart snapshot: %w", err)
	}
	return sectionContribution(BriefSection{
		ID:       "daggerheart_character_state",
		Priority: 250,
		Label:    "Daggerheart character state",
		Content:  data,
	}), nil
}

// dualityRulesContextSource reads the Daggerheart duality dice rules from the
// session resource and returns them as a brief section. This gives the GM agent
// authoritative dice mechanics and outcome definitions in every prompt.
func dualityRulesContextSource(ctx context.Context, sess Session, _ PromptInput) (BriefContribution, error) {
	rules, err := sess.ReadResource(ctx, "daggerheart://rules/version")
	if err != nil {
		return BriefContribution{}, fmt.Errorf("read daggerheart rules: %w", err)
	}
	return sectionContribution(BriefSection{
		ID:       "daggerheart_duality_rules",
		Priority: 200,
		Label:    "Daggerheart duality rules",
		Content:  rules,
	}), nil
}
