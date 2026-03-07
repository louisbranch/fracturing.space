package templates

import (
	"bytes"
	"context"
	"regexp"
	"strings"
	"testing"
)

func TestCreationStepClassSubclassRendersClassDomainWatermarks(t *testing.T) {
	t.Parallel()

	view := CharacterCreationPageView{
		CampaignID:  "campaign-1",
		CharacterID: "character-1",
		Creation: CampaignCharacterCreationView{
			Classes: []CampaignCreationClassView{
				{
					ID:              "class.druid",
					Name:            "Druid",
					ImageURL:        "https://cdn.example.com/class/druid.png",
					StartingHP:      6,
					StartingEvasion: 10,
					DomainNames:     []string{"Sage", "Arcana"},
					DomainWatermarks: []CampaignCreationDomainWatermarkView{
						{ID: "domain.sage", Name: "Sage", IconURL: "https://cdn.example.com/domain/sage.png"},
						{ID: "domain.arcana", Name: "Arcana", IconURL: "https://cdn.example.com/domain/arcana.png"},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := creationStepClassSubclass(view, testLocalizer{}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render creationStepClassSubclass: %v", err)
	}

	got := buf.String()
	for _, marker := range []string{
		`data-class-domain-watermarks="true"`,
		`data-class-domain-watermark="domain.sage"`,
		`data-class-domain-watermark="domain.arcana"`,
		`mask-image:url(https://cdn.example.com/domain/sage.png)`,
		`mask-image:url(https://cdn.example.com/domain/arcana.png)`,
		`bg-base-content opacity-50`,
	} {
		if !strings.Contains(got, marker) {
			t.Fatalf("class-subclass output missing marker %q: %q", marker, got)
		}
	}
}

func TestCreationStepClassSubclassSkipsWatermarkWithoutIconURL(t *testing.T) {
	t.Parallel()

	view := CharacterCreationPageView{
		CampaignID:  "campaign-1",
		CharacterID: "character-1",
		Creation: CampaignCharacterCreationView{
			Classes: []CampaignCreationClassView{
				{
					ID:              "class.druid",
					Name:            "Druid",
					ImageURL:        "https://cdn.example.com/class/druid.png",
					StartingHP:      6,
					StartingEvasion: 10,
					DomainWatermarks: []CampaignCreationDomainWatermarkView{
						{ID: "domain.sage", Name: "Sage", IconURL: "https://cdn.example.com/domain/sage.png"},
						{ID: "domain.arcana", Name: "Arcana", IconURL: "   "},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := creationStepClassSubclass(view, testLocalizer{}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render creationStepClassSubclass: %v", err)
	}

	got := buf.String()
	if !strings.Contains(got, `data-class-domain-watermark="domain.sage"`) {
		t.Fatalf("class-subclass output missing rendered watermark icon for mapped domain: %q", got)
	}
	// Invariant: watermark entries with no icon URL must not render a placeholder element.
	if strings.Contains(got, `data-class-domain-watermark="domain.arcana"`) {
		t.Fatalf("class-subclass output unexpectedly rendered watermark without icon url: %q", got)
	}
}

func TestCreationStepClassSubclassSkipsWatermarkContainerWhenAllIconsMissing(t *testing.T) {
	t.Parallel()

	view := CharacterCreationPageView{
		CampaignID:  "campaign-1",
		CharacterID: "character-1",
		Creation: CampaignCharacterCreationView{
			Classes: []CampaignCreationClassView{
				{
					ID:              "class.druid",
					Name:            "Druid",
					ImageURL:        "https://cdn.example.com/class/druid.png",
					StartingHP:      6,
					StartingEvasion: 10,
					DomainWatermarks: []CampaignCreationDomainWatermarkView{
						{ID: "domain.sage", Name: "Sage", IconURL: "   "},
						{ID: "domain.arcana", Name: "Arcana", IconURL: ""},
					},
				},
			},
		},
	}

	var buf bytes.Buffer
	err := creationStepClassSubclass(view, testLocalizer{}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render creationStepClassSubclass: %v", err)
	}

	got := buf.String()
	if strings.Contains(got, `data-class-domain-watermarks="true"`) {
		t.Fatalf("class-subclass output unexpectedly rendered empty watermark container: %q", got)
	}
}

func TestCreationStepDomainCardsUsesSharedSelectableCardShell(t *testing.T) {
	t.Parallel()

	view := CharacterCreationPageView{
		CampaignID:  "campaign-1",
		CharacterID: "character-1",
		Creation: CampaignCharacterCreationView{
			DomainCardIDs: []string{"dc1"},
			DomainCards: []CampaignCreationDomainCardView{
				{ID: "dc1", Name: "Runeward", ImageURL: "https://cdn.example.com/domain-cards/runeward.png", DomainName: "Arcana", Level: 1},
				{ID: "dc2", Name: "Wallwalk", DomainName: "Arcana", Level: 1},
			},
		},
	}

	var buf bytes.Buffer
	err := creationStepDomainCards(view, testLocalizer{}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render creationStepDomainCards: %v", err)
	}

	markup := strings.SplitN(buf.String(), "<script>", 2)[0]
	if strings.Count(markup, `data-creation-option-kind="domain-card"`) != 2 {
		t.Fatalf("domain-card selectable cards = %d, want 2", strings.Count(markup, `data-creation-option-kind="domain-card"`))
	}
	for _, marker := range []string{
		`type="checkbox"`,
		`data-image-frame="true"`,
		`data-image-skeleton="true"`,
		`border-primary ring-2 ring-primary/20`,
	} {
		if !strings.Contains(markup, marker) {
			t.Fatalf("domain-cards output missing marker %q: %q", marker, markup)
		}
	}
}

func TestCreationStepDomainCardsDisablesUnselectedWhenLimitReached(t *testing.T) {
	t.Parallel()

	view := CharacterCreationPageView{
		CampaignID:  "campaign-1",
		CharacterID: "character-1",
		Creation: CampaignCharacterCreationView{
			DomainCardIDs: []string{"dc1", "dc2"},
			DomainCards: []CampaignCreationDomainCardView{
				{ID: "dc1", Name: "Runeward", Level: 1},
				{ID: "dc2", Name: "Wallwalk", Level: 1},
				{ID: "dc3", Name: "Whirlwind", Level: 1},
			},
		},
	}

	var buf bytes.Buffer
	err := creationStepDomainCards(view, testLocalizer{}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render creationStepDomainCards: %v", err)
	}

	markup := strings.SplitN(buf.String(), "<script>", 2)[0]
	if !regexp.MustCompile(`value="dc3"[^>]*disabled`).MatchString(markup) {
		t.Fatalf("expected unselected dc3 checkbox to be disabled when two cards are already selected: %q", markup)
	}
}

func TestCreationIconMaskStyleEscapesSpecialURLCharacters(t *testing.T) {
	t.Parallel()

	style := creationIconMaskStyle("https://cdn.example.com/domain/sage(1).png?sig=a b'c\"d\\")
	for _, marker := range []string{
		`mask-image:url(https://cdn.example.com/domain/sage%281%29.png?sig=a%20b%27c%22d%5C)`,
		`-webkit-mask-image:url(https://cdn.example.com/domain/sage%281%29.png?sig=a%20b%27c%22d%5C)`,
	} {
		if !strings.Contains(style, marker) {
			t.Fatalf("mask style missing escaped marker %q: %q", marker, style)
		}
	}
}
