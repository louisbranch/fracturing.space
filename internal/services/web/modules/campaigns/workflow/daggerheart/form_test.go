package daggerheart

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

func TestParseStepInputHappyPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		step   int32
		body   string
		verify func(*testing.T, *campaigns.CampaignCharacterCreationStepInput)
	}{
		{
			name: "step 1 class and subclass",
			step: 1,
			body: "class_id=warrior&subclass_id=guardian",
			verify: func(t *testing.T, input *campaigns.CampaignCharacterCreationStepInput) {
				if input.ClassSubclass == nil {
					t.Fatal("ClassSubclass is nil")
				}
				if input.ClassSubclass.ClassID != "warrior" {
					t.Fatalf("ClassID = %q, want warrior", input.ClassSubclass.ClassID)
				}
				if input.ClassSubclass.SubclassID != "guardian" {
					t.Fatalf("SubclassID = %q, want guardian", input.ClassSubclass.SubclassID)
				}
			},
		},
		{
			name: "step 2 ancestry and community",
			step: 2,
			body: "ancestry_id=elf&community_id=loreborne",
			verify: func(t *testing.T, input *campaigns.CampaignCharacterCreationStepInput) {
				if input.Heritage == nil {
					t.Fatal("Heritage is nil")
				}
				if input.Heritage.AncestryID != "elf" {
					t.Fatalf("AncestryID = %q, want elf", input.Heritage.AncestryID)
				}
				if input.Heritage.CommunityID != "loreborne" {
					t.Fatalf("CommunityID = %q, want loreborne", input.Heritage.CommunityID)
				}
			},
		},
		{
			name: "step 3 traits",
			step: 3,
			body: "agility=2&strength=-1&finesse=1&instinct=0&presence=3&knowledge=-2",
			verify: func(t *testing.T, input *campaigns.CampaignCharacterCreationStepInput) {
				if input.Traits == nil {
					t.Fatal("Traits is nil")
				}
				if input.Traits.Agility != 2 {
					t.Fatalf("Agility = %d, want 2", input.Traits.Agility)
				}
				if input.Traits.Strength != -1 {
					t.Fatalf("Strength = %d, want -1", input.Traits.Strength)
				}
				if input.Traits.Knowledge != -2 {
					t.Fatalf("Knowledge = %d, want -2", input.Traits.Knowledge)
				}
			},
		},
		{
			name: "step 4 details",
			step: 4,
			body: "",
			verify: func(t *testing.T, input *campaigns.CampaignCharacterCreationStepInput) {
				if input.Details == nil {
					t.Fatal("Details is nil")
				}
			},
		},
		{
			name: "step 5 equipment with secondary weapon",
			step: 5,
			body: "weapon_primary_id=sword&weapon_secondary_id=dagger&armor_id=leather&potion_item_id=item.minor-health-potion",
			verify: func(t *testing.T, input *campaigns.CampaignCharacterCreationStepInput) {
				if input.Equipment == nil {
					t.Fatal("Equipment is nil")
				}
				if len(input.Equipment.WeaponIDs) != 2 {
					t.Fatalf("WeaponIDs = %v, want 2", input.Equipment.WeaponIDs)
				}
				if input.Equipment.ArmorID != "leather" {
					t.Fatalf("ArmorID = %q, want leather", input.Equipment.ArmorID)
				}
			},
		},
		{
			name: "step 5 equipment without secondary weapon",
			step: 5,
			body: "weapon_primary_id=sword&armor_id=leather&potion_item_id=item.minor-health-potion",
			verify: func(t *testing.T, input *campaigns.CampaignCharacterCreationStepInput) {
				if input.Equipment == nil {
					t.Fatal("Equipment is nil")
				}
				if len(input.Equipment.WeaponIDs) != 1 {
					t.Fatalf("WeaponIDs = %v, want 1", input.Equipment.WeaponIDs)
				}
			},
		},
		{
			name: "step 6 background",
			step: 6,
			body: "background=Noble+scholar",
			verify: func(t *testing.T, input *campaigns.CampaignCharacterCreationStepInput) {
				if input.Background == nil {
					t.Fatal("Background is nil")
				}
				if input.Background.Background != "Noble scholar" {
					t.Fatalf("Background = %q, want %q", input.Background.Background, "Noble scholar")
				}
			},
		},
		{
			name: "step 7 experience with modifier",
			step: 7,
			body: "experience_name=Outlander&experience_modifier=2",
			verify: func(t *testing.T, input *campaigns.CampaignCharacterCreationStepInput) {
				if input.Experiences == nil {
					t.Fatal("Experiences is nil")
				}
				if len(input.Experiences.Experiences) != 1 {
					t.Fatalf("Experiences = %d, want 1", len(input.Experiences.Experiences))
				}
				if input.Experiences.Experiences[0].Name != "Outlander" {
					t.Fatalf("Name = %q, want Outlander", input.Experiences.Experiences[0].Name)
				}
				if input.Experiences.Experiences[0].Modifier != 2 {
					t.Fatalf("Modifier = %d, want 2", input.Experiences.Experiences[0].Modifier)
				}
			},
		},
		{
			name: "step 7 experience without modifier",
			step: 7,
			body: "experience_name=Outlander",
			verify: func(t *testing.T, input *campaigns.CampaignCharacterCreationStepInput) {
				if input.Experiences == nil {
					t.Fatal("Experiences is nil")
				}
				if input.Experiences.Experiences[0].Modifier != 0 {
					t.Fatalf("Modifier = %d, want 0", input.Experiences.Experiences[0].Modifier)
				}
			},
		},
		{
			name: "step 8 domain cards deduplicates",
			step: 8,
			body: "domain_card_id=dc1&domain_card_id=dc2&domain_card_id=dc1",
			verify: func(t *testing.T, input *campaigns.CampaignCharacterCreationStepInput) {
				if input.DomainCards == nil {
					t.Fatal("DomainCards is nil")
				}
				if len(input.DomainCards.DomainCardIDs) != 2 {
					t.Fatalf("DomainCardIDs = %v, want 2 (deduped)", input.DomainCards.DomainCardIDs)
				}
			},
		},
		{
			name: "step 9 connections",
			step: 9,
			body: "connections=My+ally+is+the+barkeep",
			verify: func(t *testing.T, input *campaigns.CampaignCharacterCreationStepInput) {
				if input.Connections == nil {
					t.Fatal("Connections is nil")
				}
				if input.Connections.Connections != "My ally is the barkeep" {
					t.Fatalf("Connections = %q, want %q", input.Connections.Connections, "My ally is the barkeep")
				}
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			input, err := Workflow{}.ParseStepInput(req, tt.step)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			tt.verify(t, input)
		})
	}
}

func TestParseStepInputUsesLocalizationKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		step    int32
		body    string
		wantKey string
	}{
		{
			name:    "class and subclass required",
			step:    1,
			body:    "class_id=warrior",
			wantKey: "error.web.message.character_creation_class_and_subclass_are_required",
		},
		{
			name:    "ancestry and community required",
			step:    2,
			body:    "ancestry_id=elf",
			wantKey: "error.web.message.character_creation_ancestry_and_community_are_required",
		},
		{
			name:    "unknown step",
			step:    42,
			body:    "",
			wantKey: "error.web.message.character_creation_step_is_not_available",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

			_, err := Workflow{}.ParseStepInput(req, tt.step)
			if err == nil {
				t.Fatalf("expected error")
			}
			if got := apperrors.LocalizationKey(err); got != tt.wantKey {
				t.Fatalf("LocalizationKey(err) = %q, want %q", got, tt.wantKey)
			}
		})
	}
}

func TestParseRequiredInt32UsesLocalizationKeys(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		raw     string
		wantKey string
	}{
		{
			name:    "missing value",
			raw:     "   ",
			wantKey: "error.web.message.character_creation_numeric_field_is_required",
		},
		{
			name:    "invalid integer",
			raw:     "abc",
			wantKey: "error.web.message.character_creation_numeric_field_must_be_valid_integer",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			_, err := parseRequiredInt32(tt.raw, "agility")
			if err == nil {
				t.Fatalf("expected error")
			}
			if got := apperrors.LocalizationKey(err); got != tt.wantKey {
				t.Fatalf("LocalizationKey(err) = %q, want %q", got, tt.wantKey)
			}
		})
	}
}

func TestParseOptionalInt32UsesLocalizationKey(t *testing.T) {
	t.Parallel()

	_, err := parseOptionalInt32("bad")
	if err == nil {
		t.Fatalf("expected error")
	}
	if got := apperrors.LocalizationKey(err); got != "error.web.message.character_creation_modifier_must_be_valid_integer" {
		t.Fatalf("LocalizationKey(err) = %q, want %q", got, "error.web.message.character_creation_modifier_must_be_valid_integer")
	}
}
