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
		`class="relative overflow-hidden h-full w-full"`,
		`class="skeleton absolute inset-0 z-0 pointer-events-none"`,
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

func TestCreationStepEquipmentUsesSharedSelectableCardShellForPotions(t *testing.T) {
	t.Parallel()

	view := CharacterCreationPageView{
		CampaignID:  "campaign-1",
		CharacterID: "character-1",
		Creation: CampaignCharacterCreationView{
			PrimaryWeapons: []CampaignCreationWeaponView{
				{ID: "weapon-1", Name: "Longsword"},
			},
			Armor: []CampaignCreationArmorView{
				{ID: "armor-1", Name: "Chainmail"},
			},
			PotionItems: []CampaignCreationItemView{
				{ID: "item-1", Name: "Minor Health Potion", ImageURL: "https://cdn.example.com/items/minor-health-potion.png"},
			},
		},
	}

	var buf bytes.Buffer
	err := creationStepEquipment(view, testLocalizer{}).Render(context.Background(), &buf)
	if err != nil {
		t.Fatalf("render creationStepEquipment: %v", err)
	}

	markup := strings.SplitN(buf.String(), "<script>", 2)[0]
	for _, marker := range []string{
		`data-creation-option-kind="equipment-potion"`,
		`name="potion_item_id"`,
		`type="radio"`,
		`data-image-frame="true"`,
		`data-image-skeleton="true"`,
	} {
		if !strings.Contains(markup, marker) {
			t.Fatalf("equipment output missing marker %q: %q", marker, markup)
		}
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

func TestCreationSummaryCardRendersDetailsAndBackToCampaignAction(t *testing.T) {
	t.Parallel()

	view := CharacterCreationPageView{
		CampaignID:  "campaign-1",
		CharacterID: "character-1",
		Creation: CampaignCharacterCreationView{
			Ready:           true,
			ClassID:         "class-1",
			SubclassID:      "subclass-1",
			AncestryID:      "ancestry-1",
			CommunityID:     "community-1",
			Agility:         "2",
			Strength:        "1",
			Finesse:         "1",
			Instinct:        "0",
			Presence:        "0",
			Knowledge:       "-1",
			PrimaryWeaponID: "weapon-1",
			ArmorID:         "armor-1",
			PotionItemID:    "item-1",
			Description:     "Scarred, observant, and always impeccably dressed.",
			Background:      "Former court archivist turned wanderer.",
			Connections:     "Owes the party a hard-won favor.",
			Classes:         []CampaignCreationClassView{{ID: "class-1", Name: "Warrior"}},
			Subclasses:      []CampaignCreationSubclassView{{ID: "subclass-1", Name: "Guardian"}},
			Ancestries:      []CampaignCreationHeritageView{{ID: "ancestry-1", Name: "Human"}},
			Communities:     []CampaignCreationHeritageView{{ID: "community-1", Name: "Loreborne"}},
			PrimaryWeapons:  []CampaignCreationWeaponView{{ID: "weapon-1", Name: "Longsword"}},
			Armor:           []CampaignCreationArmorView{{ID: "armor-1", Name: "Chainmail"}},
			PotionItems:     []CampaignCreationItemView{{ID: "item-1", Name: "Minor Potion"}},
		},
	}

	var buf bytes.Buffer
	if err := creationSummaryCard(view, nil).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render creationSummaryCard: %v", err)
	}

	got := buf.String()
	for _, marker := range []string{
		`data-character-creation-summary="true"`,
		`game.character_creation.step.details`,
		`Scarred, observant, and always impeccably dressed.`,
		`data-character-creation-back-to-campaign="true"`,
		`href="/app/campaigns/campaign-1/characters"`,
	} {
		if !strings.Contains(got, marker) {
			t.Fatalf("summary output missing marker %q: %q", marker, got)
		}
	}
	if strings.Contains(got, `data-character-creation-next="true"`) {
		t.Fatalf("summary output should replace next button with back-to-campaign action: %q", got)
	}
	if detailsIdx, domainIdx := strings.Index(got, `game.character_creation.step.details`), strings.Index(got, `game.character_creation.step.domain_cards`); detailsIdx != -1 && domainIdx != -1 && detailsIdx < domainIdx {
		t.Fatalf("summary output should render details after the left-column step summary blocks: %q", got)
	}
}

func TestCreationStepClassSubclassRendersDisabledNextUntilSelectionsComplete(t *testing.T) {
	t.Parallel()

	incomplete := CharacterCreationPageView{
		CampaignID:  "campaign-1",
		CharacterID: "character-1",
		Creation: CampaignCharacterCreationView{
			Classes:    []CampaignCreationClassView{{ID: "class-1", Name: "Warrior"}},
			Subclasses: []CampaignCreationSubclassView{{ID: "subclass-1", Name: "Guardian", ClassID: "class-1"}},
		},
	}

	var incompleteBuf bytes.Buffer
	if err := creationStepClassSubclass(incomplete, nil).Render(context.Background(), &incompleteBuf); err != nil {
		t.Fatalf("render incomplete creationStepClassSubclass: %v", err)
	}
	incompleteMarkup := strings.SplitN(incompleteBuf.String(), "<script>", 2)[0]
	if !strings.Contains(incompleteMarkup, `disabled data-character-creation-next="true"`) {
		t.Fatalf("expected next button to start disabled when class/subclass are incomplete: %q", incompleteMarkup)
	}

	complete := incomplete
	complete.Creation.ClassID = "class-1"
	complete.Creation.SubclassID = "subclass-1"

	var completeBuf bytes.Buffer
	if err := creationStepClassSubclass(complete, nil).Render(context.Background(), &completeBuf); err != nil {
		t.Fatalf("render complete creationStepClassSubclass: %v", err)
	}
	completeMarkup := strings.SplitN(completeBuf.String(), "<script>", 2)[0]
	if strings.Contains(completeMarkup, `disabled data-character-creation-next="true"`) {
		t.Fatalf("expected next button to start enabled when class/subclass are complete: %q", completeMarkup)
	}
}

func TestCharacterCreationPageRendersNextStepImagePrefetchHints(t *testing.T) {
	t.Parallel()

	view := CharacterCreationPageView{
		CampaignID:  "campaign-1",
		CharacterID: "character-1",
		Creation: CampaignCharacterCreationView{
			NextStepPrefetchURLs: []string{
				"https://cdn.example.com/armor-1.png",
				"https://cdn.example.com/item-1.png",
			},
		},
	}

	var buf bytes.Buffer
	if err := CharacterCreationPage(view, nil).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render CharacterCreationPage: %v", err)
	}

	got := buf.String()
	for _, marker := range []string{
		`data-image-prefetch-root="character-creation"`,
		`data-image-prefetch-url="https://cdn.example.com/armor-1.png"`,
		`data-image-prefetch-url="https://cdn.example.com/item-1.png"`,
	} {
		if !strings.Contains(got, marker) {
			t.Fatalf("CharacterCreationPage output missing marker %q: %q", marker, got)
		}
	}
}

func TestCampaignCharacterCreationSummaryBodyRendersSharedSummary(t *testing.T) {
	t.Parallel()

	creation := CampaignCharacterCreationView{
		Ready:       true,
		NextStep:    9,
		ClassID:     "class.rogue",
		SubclassID:  "subclass.night",
		AncestryID:  "ancestry.human",
		CommunityID: "community.warden",
		Classes: []CampaignCreationClassView{
			{ID: "class.rogue", Name: "Rogue"},
		},
		Subclasses: []CampaignCreationSubclassView{
			{ID: "subclass.night", Name: "Night"},
		},
		Ancestries: []CampaignCreationHeritageView{
			{ID: "ancestry.human", Name: "Human"},
		},
		Communities: []CampaignCreationHeritageView{
			{ID: "community.warden", Name: "Warden"},
		},
	}

	var buf bytes.Buffer
	if err := CampaignCharacterCreationSummaryBody(creation, nil).Render(context.Background(), &buf); err != nil {
		t.Fatalf("render CampaignCharacterCreationSummaryBody: %v", err)
	}

	got := buf.String()
	for _, marker := range []string{
		`game.character_creation.step.class_subclass`,
		`Rogue`,
		`Night`,
		`Human`,
		`Warden`,
	} {
		if !strings.Contains(got, marker) {
			t.Fatalf("CampaignCharacterCreationSummaryBody output missing marker %q: %q", marker, got)
		}
	}
}
