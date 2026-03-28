package render

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/a-h/templ"
)

func TestCampaignCreationHelperContracts(t *testing.T) {
	t.Parallel()

	loc := testLocalizer{
		"game.character_creation.step.heritage": "Heritage",
		"error.workflow.blocked":                "Blocked by requirements",
	}

	view := CampaignCharacterCreationView{
		Heritage: CampaignCreationHeritageSelectionView{
			FirstFeatureAncestryID:  "ancestry.drake",
			SecondFeatureAncestryID: "ancestry.elf",
			CommunityID:             "community.wanderborne",
		},
		Ancestries: []CampaignCreationHeritageView{
			{
				ID:   "ancestry.drake",
				Name: "Drakona",
				Features: []CampaignCreationClassFeatureView{
					{Name: "Claws"},
					{Name: "Scales"},
				},
			},
			{
				ID:   "ancestry.elf",
				Name: "Elf",
				Features: []CampaignCreationClassFeatureView{
					{Name: "Grace"},
					{Name: "Wings"},
				},
			},
		},
		Communities: []CampaignCreationHeritageView{
			{
				ID:   "community.wanderborne",
				Name: "Wanderborne",
			},
		},
	}

	companion := &CampaignCreationCompanionView{
		AnimalKind:        " wolf ",
		Name:              " Ash ",
		AttackDescription: " Pack strike ",
		DamageType:        " physical ",
		Experiences: []CampaignCreationExperienceView{
			{ID: "exp-1", Name: " Scout "},
			{ID: "exp-2", Name: " Guard "},
		},
	}

	if got := campaignCreationStepLabel(loc, " heritage "); got != "Heritage" {
		t.Fatalf("campaignCreationStepLabel() = %q, want %q", got, "Heritage")
	}
	if got := campaignCreationStepLabel(loc, "unknown_step"); got != "unknown_step" {
		t.Fatalf("campaignCreationStepLabel(unknown) = %q", got)
	}
	if got := campaignCreationNumericValue("  "); got != "0" {
		t.Fatalf("campaignCreationNumericValue(blank) = %q", got)
	}
	if got := campaignCreationNumericValue(" 12 "); got != "12" {
		t.Fatalf("campaignCreationNumericValue(value) = %q", got)
	}
	if got := campaignCreationUnmetReason(loc, " error.workflow.blocked "); got != "Blocked by requirements" {
		t.Fatalf("campaignCreationUnmetReason(message key) = %q", got)
	}
	if got := campaignCreationUnmetReason(loc, "Need one more choice"); got != "Need one more choice" {
		t.Fatalf("campaignCreationUnmetReason(literal) = %q", got)
	}
	if got := campaignCreationHeritageFeatureSummary(view); got != "Claws + Wings" {
		t.Fatalf("campaignCreationHeritageFeatureSummary() = %q", got)
	}
	if got := campaignCreationHeritageStepSummary(view); got != "Drakona / Elf, Wanderborne · Claws + Wings" {
		t.Fatalf("campaignCreationHeritageStepSummary() = %q", got)
	}
	if got := campaignCreationCompanionExperienceName(companion, 0); got != "Scout" {
		t.Fatalf("campaignCreationCompanionExperienceName(0) = %q", got)
	}
	if got := campaignCreationCompanionExperienceName(companion, 3); got != "" {
		t.Fatalf("campaignCreationCompanionExperienceName(out of range) = %q", got)
	}
	if got := campaignCreationCompanionText(companion, "animal_kind"); got != "wolf" {
		t.Fatalf("campaignCreationCompanionText(animal_kind) = %q", got)
	}
	if got := campaignCreationCompanionText(companion, "attack_description"); got != "Pack strike" {
		t.Fatalf("campaignCreationCompanionText(attack_description) = %q", got)
	}
	if got := campaignCreationCompanionText(companion, "missing"); got != "" {
		t.Fatalf("campaignCreationCompanionText(missing) = %q", got)
	}
}

func TestCreationStepReadinessHelpers(t *testing.T) {
	t.Parallel()

	if creationStepExperiencesReady(CampaignCharacterCreationView{
		Experiences: []CampaignCreationExperienceView{
			{Name: "Sailor"},
			{Name: "  "},
		},
	}) {
		t.Fatal("creationStepExperiencesReady(incomplete) = true")
	}
	if !creationStepExperiencesReady(CampaignCharacterCreationView{
		Experiences: []CampaignCreationExperienceView{
			{Name: "Sailor"},
			{Name: "Tracker"},
		},
	}) {
		t.Fatal("creationStepExperiencesReady(complete) = false")
	}
	if creationStepTextareaReady("  ") {
		t.Fatal("creationStepTextareaReady(blank) = true")
	}
	if !creationStepTextareaReady("Ready text") {
		t.Fatal("creationStepTextareaReady(text) = false")
	}
}

func TestCreationStepExperiencesRendersOwnedMarkers(t *testing.T) {
	t.Parallel()

	view := CharacterCreationPageView{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Creation: CampaignCharacterCreationView{
			Experiences: []CampaignCreationExperienceView{
				{Name: "Sailor"},
				{Name: "Tracker"},
			},
		},
	}

	got := renderCreationComponent(t, creationStepExperiences(view, testLocalizer{
		"game.character_creation.step.experiences":        "Experiences",
		"game.character_creation.experiences_guidance":    "Choose two backgrounds.",
		"game.character_creation.experience_label_1":      "Experience 1",
		"game.character_creation.experience_label_2":      "Experience 2",
		"game.character_creation.field.experience_name":   "Experience name",
		"game.character_creation.action_next":             "Next",
		"game.character_creation.action_back":             "Back",
		"game.character_creation.action_cancel":           "Cancel",
		"game.character_creation.action_finish":           "Finish",
		"game.character_creation.action_back_to_campaign": "Back to campaign",
	}))

	for _, marker := range []string{
		`action="/app/campaigns/camp-1/characters/char-1/creation/step"`,
		`data-character-creation-form-step="5"`,
		`name="experience_0_name"`,
		`value="Sailor"`,
		`name="experience_1_name"`,
		`value="Tracker"`,
		`<button class="btn btn-primary btn-wide" type="submit" data-character-creation-next="true">`,
		`form.addEventListener('input', updateNextButton);`,
	} {
		if !strings.Contains(got, marker) {
			t.Fatalf("creationStepExperiences output missing marker %q: %q", marker, got)
		}
	}
}

func TestCreationTextareaStepWrappersRenderOwnedMarkers(t *testing.T) {
	t.Parallel()

	loc := testLocalizer{
		"game.character_creation.step.details":            "Details",
		"game.character_creation.details_guidance_1":      "Describe the first impression.",
		"game.character_creation.details_guidance_2":      "Include a notable habit.",
		"game.character_creation.details_guidance_3":      "Mention a visible feature.",
		"game.character_creation.step.background":         "Background",
		"game.character_creation.background_guidance_1":   "Name a place from your past.",
		"game.character_creation.background_guidance_2":   "Describe a formative event.",
		"game.character_creation.background_guidance_3":   "Explain why it still matters.",
		"game.character_creation.step.connections":        "Connections",
		"game.character_creation.connections_guidance_1":  "Name one bond.",
		"game.character_creation.connections_guidance_2":  "Explain one tension.",
		"game.character_creation.connections_guidance_3":  "Note one unresolved question.",
		"game.character_creation.action_next":             "Next",
		"game.character_creation.action_back":             "Back",
		"game.character_creation.action_cancel":           "Cancel",
		"game.character_creation.action_finish":           "Finish",
		"game.character_creation.action_back_to_campaign": "Back to campaign",
	}

	tests := []struct {
		name       string
		component  func(*testing.T, CharacterCreationPageView, testLocalizer) string
		stepNumber string
		fieldName  string
		heading    string
		value      string
		guidance   string
	}{
		{
			name: "details",
			component: func(t *testing.T, view CharacterCreationPageView, loc testLocalizer) string {
				return renderCreationComponent(t, creationStepDetails(view, loc))
			},
			stepNumber: "7",
			fieldName:  "description",
			heading:    "Details",
			value:      "Quiet, observant, always cataloguing exits.",
			guidance:   "Describe the first impression.",
		},
		{
			name: "background",
			component: func(t *testing.T, view CharacterCreationPageView, loc testLocalizer) string {
				return renderCreationComponent(t, creationStepBackground(view, loc))
			},
			stepNumber: "8",
			fieldName:  "background",
			heading:    "Background",
			value:      "Raised in a lighthouse after the war.",
			guidance:   "Name a place from your past.",
		},
		{
			name: "connections",
			component: func(t *testing.T, view CharacterCreationPageView, loc testLocalizer) string {
				return renderCreationComponent(t, creationStepConnections(view, loc))
			},
			stepNumber: "9",
			fieldName:  "connections",
			heading:    "Connections",
			value:      "Owes Rowan a favor and distrusts the council.",
			guidance:   "Name one bond.",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			view := CharacterCreationPageView{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				Creation: CampaignCharacterCreationView{
					Description: tt.value,
					Background:  tt.value,
					Connections: tt.value,
				},
			}

			got := tt.component(t, view, loc)
			for _, marker := range []string{
				`action="/app/campaigns/camp-1/characters/char-1/creation/step"`,
				`data-character-creation-form-step="` + tt.stepNumber + `"`,
				`<h3 class="text-lg font-semibold">` + tt.heading + `</h3>`,
				`<textarea name="` + tt.fieldName + `" rows="6" required class="textarea textarea-bordered w-full">` + tt.value + `</textarea>`,
				tt.guidance,
				`<button class="btn btn-primary btn-wide" type="submit" data-character-creation-next="true">`,
				`textarea.value.trim() === ''`,
			} {
				if !strings.Contains(got, marker) {
					t.Fatalf("%s output missing marker %q: %q", tt.name, marker, got)
				}
			}
		})
	}
}

func renderCreationComponent(t *testing.T, component templ.Component) string {
	t.Helper()

	var buf bytes.Buffer
	if err := component.Render(context.Background(), &buf); err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	return buf.String()
}
