package daggerheart

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// characterCreationStepParser parses one step-specific form payload into domain input.
type characterCreationStepParser func(url.Values) (*campaignapp.CampaignCharacterCreationStepInput, error)

var characterCreationStepParsers = map[int32]characterCreationStepParser{
	1: parseClassSubclassStepInput,
	2: parseHeritageStepInput,
	3: parseTraitsStepInput,
	4: parseEquipmentStepInput,
	5: parseExperiencesStepInput,
	6: parseDomainCardsStepInput,
	7: parseDetailsStepInput,
	8: parseBackgroundStepInput,
	9: parseConnectionsStepInput,
}

// ParseStepInput parses a character creation step form submission into domain
// step input based on the current step number.
func (Workflow) ParseStepInput(form url.Values, nextStep int32) (*campaignapp.CampaignCharacterCreationStepInput, error) {
	parser, ok := characterCreationStepParsers[nextStep]
	if !ok {
		return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_step_is_not_available", "character creation step is not available")
	}
	return parser(form)
}

// parseClassSubclassStepInput parses and validates the class/subclass selection step.
func parseClassSubclassStepInput(form url.Values) (*campaignapp.CampaignCharacterCreationStepInput, error) {
	classID := strings.TrimSpace(form.Get("class_id"))
	subclassID := strings.TrimSpace(form.Get("subclass_id"))
	if classID == "" || subclassID == "" {
		return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_class_and_subclass_are_required", "class and subclass are required")
	}
	return &campaignapp.CampaignCharacterCreationStepInput{
		ClassSubclass: &campaignapp.CampaignCharacterCreationStepClassSubclass{
			ClassID:    classID,
			SubclassID: subclassID,
		},
	}, nil
}

// parseHeritageStepInput parses and validates ancestry/community selections.
func parseHeritageStepInput(form url.Values) (*campaignapp.CampaignCharacterCreationStepInput, error) {
	ancestryID := strings.TrimSpace(form.Get("ancestry_id"))
	communityID := strings.TrimSpace(form.Get("community_id"))
	if ancestryID == "" || communityID == "" {
		return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_ancestry_and_community_are_required", "ancestry and community are required")
	}
	return &campaignapp.CampaignCharacterCreationStepInput{
		Heritage: &campaignapp.CampaignCharacterCreationStepHeritage{
			AncestryID:  ancestryID,
			CommunityID: communityID,
		},
	}, nil
}

// parseTraitsStepInput parses and validates numeric trait allocations.
func parseTraitsStepInput(form url.Values) (*campaignapp.CampaignCharacterCreationStepInput, error) {
	agility, err := parseRequiredInt32(form.Get("agility"), "agility")
	if err != nil {
		return nil, err
	}
	strength, err := parseRequiredInt32(form.Get("strength"), "strength")
	if err != nil {
		return nil, err
	}
	finesse, err := parseRequiredInt32(form.Get("finesse"), "finesse")
	if err != nil {
		return nil, err
	}
	instinct, err := parseRequiredInt32(form.Get("instinct"), "instinct")
	if err != nil {
		return nil, err
	}
	presence, err := parseRequiredInt32(form.Get("presence"), "presence")
	if err != nil {
		return nil, err
	}
	knowledge, err := parseRequiredInt32(form.Get("knowledge"), "knowledge")
	if err != nil {
		return nil, err
	}
	return &campaignapp.CampaignCharacterCreationStepInput{
		Traits: &campaignapp.CampaignCharacterCreationStepTraits{
			Agility:   agility,
			Strength:  strength,
			Finesse:   finesse,
			Instinct:  instinct,
			Presence:  presence,
			Knowledge: knowledge,
		},
	}, nil
}

// parseEquipmentStepInput parses and validates equipment choices for creation.
func parseEquipmentStepInput(form url.Values) (*campaignapp.CampaignCharacterCreationStepInput, error) {
	primaryWeaponID := strings.TrimSpace(form.Get("weapon_primary_id"))
	secondaryWeaponID := strings.TrimSpace(form.Get("weapon_secondary_id"))
	armorID := strings.TrimSpace(form.Get("armor_id"))
	potionItemID := strings.TrimSpace(form.Get("potion_item_id"))
	if primaryWeaponID == "" || armorID == "" || potionItemID == "" {
		return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_primary_weapon_armor_and_potion_are_required", "primary weapon, armor, and potion are required")
	}
	weaponIDs := []string{primaryWeaponID}
	if secondaryWeaponID != "" {
		weaponIDs = append(weaponIDs, secondaryWeaponID)
	}
	return &campaignapp.CampaignCharacterCreationStepInput{
		Equipment: &campaignapp.CampaignCharacterCreationStepEquipment{
			WeaponIDs:    weaponIDs,
			ArmorID:      armorID,
			PotionItemID: potionItemID,
		},
	}, nil
}

// parseExperiencesStepInput parses and validates the two required experiences.
func parseExperiencesStepInput(form url.Values) (*campaignapp.CampaignCharacterCreationStepInput, error) {
	experiences := make([]campaignapp.CampaignCharacterCreationStepExperience, 0, 2)
	for i := 0; i < 2; i++ {
		name := strings.TrimSpace(form.Get(fmt.Sprintf("experience_%d_name", i)))
		if name == "" {
			continue
		}
		experiences = append(experiences, campaignapp.CampaignCharacterCreationStepExperience{
			Name:     name,
			Modifier: 2,
		})
	}
	if len(experiences) != 2 {
		return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_two_experiences_required", "two experiences are required")
	}
	return &campaignapp.CampaignCharacterCreationStepInput{
		Experiences: &campaignapp.CampaignCharacterCreationStepExperiences{
			Experiences: experiences,
		},
	}, nil
}

// parseDomainCardsStepInput parses and validates unique domain-card selections.
func parseDomainCardsStepInput(form url.Values) (*campaignapp.CampaignCharacterCreationStepInput, error) {
	rawDomainCardIDs := form["domain_card_id"]
	domainCardIDs := make([]string, 0, len(rawDomainCardIDs))
	seen := map[string]struct{}{}
	for _, rawDomainCardID := range rawDomainCardIDs {
		trimmedDomainCardID := strings.TrimSpace(rawDomainCardID)
		if trimmedDomainCardID == "" {
			continue
		}
		if _, ok := seen[trimmedDomainCardID]; ok {
			continue
		}
		seen[trimmedDomainCardID] = struct{}{}
		domainCardIDs = append(domainCardIDs, trimmedDomainCardID)
	}
	if len(domainCardIDs) != 2 {
		return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_exactly_two_domain_cards_required", "exactly two domain cards are required")
	}
	return &campaignapp.CampaignCharacterCreationStepInput{
		DomainCards: &campaignapp.CampaignCharacterCreationStepDomainCards{
			DomainCardIDs: domainCardIDs,
		},
	}, nil
}

// parseDetailsStepInput parses and validates the character details step.
func parseDetailsStepInput(form url.Values) (*campaignapp.CampaignCharacterCreationStepInput, error) {
	description := strings.TrimSpace(form.Get("description"))
	if description == "" {
		return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_description_is_required", "description is required")
	}
	return &campaignapp.CampaignCharacterCreationStepInput{
		Details: &campaignapp.CampaignCharacterCreationStepDetails{
			Description: description,
		},
	}, nil
}

// parseBackgroundStepInput parses and validates background text.
func parseBackgroundStepInput(form url.Values) (*campaignapp.CampaignCharacterCreationStepInput, error) {
	background := strings.TrimSpace(form.Get("background"))
	if background == "" {
		return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_background_is_required", "background is required")
	}
	return &campaignapp.CampaignCharacterCreationStepInput{
		Background: &campaignapp.CampaignCharacterCreationStepBackground{
			Background: background,
		},
	}, nil
}

// parseConnectionsStepInput parses and validates player connections text.
func parseConnectionsStepInput(form url.Values) (*campaignapp.CampaignCharacterCreationStepInput, error) {
	connections := strings.TrimSpace(form.Get("connections"))
	if connections == "" {
		return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_connections_are_required", "connections are required")
	}
	return &campaignapp.CampaignCharacterCreationStepInput{
		Connections: &campaignapp.CampaignCharacterCreationStepConnections{
			Connections: connections,
		},
	}, nil
}

// parseRequiredInt32 parses inbound values into package-safe forms.
func parseRequiredInt32(raw string, field string) (int32, error) {
	trimmedRaw := strings.TrimSpace(raw)
	if trimmedRaw == "" {
		return 0, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_numeric_field_is_required", field+" is required")
	}
	value, err := strconv.Atoi(trimmedRaw)
	if err != nil {
		return 0, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_numeric_field_must_be_valid_integer", field+" must be a valid integer")
	}
	return int32(value), nil
}
