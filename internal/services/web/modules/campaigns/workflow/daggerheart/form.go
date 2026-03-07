package daggerheart

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"

	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// ParseStepInput parses a character creation step form submission into domain
// step input based on the current step number.
func (Workflow) ParseStepInput(form url.Values, nextStep int32) (*campaignapp.CampaignCharacterCreationStepInput, error) {
	switch nextStep {
	case 1:
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
	case 2:
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
	case 3:
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
	case 4:
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
	case 5:
		var experiences []campaignapp.CampaignCharacterCreationStepExperience
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
	case 6:
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
	case 7:
		description := strings.TrimSpace(form.Get("description"))
		if description == "" {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_description_is_required", "description is required")
		}
		return &campaignapp.CampaignCharacterCreationStepInput{
			Details: &campaignapp.CampaignCharacterCreationStepDetails{
				Description: description,
			},
		}, nil
	case 8:
		background := strings.TrimSpace(form.Get("background"))
		if background == "" {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_background_is_required", "background is required")
		}
		return &campaignapp.CampaignCharacterCreationStepInput{
			Background: &campaignapp.CampaignCharacterCreationStepBackground{
				Background: background,
			},
		}, nil
	case 9:
		connections := strings.TrimSpace(form.Get("connections"))
		if connections == "" {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_connections_are_required", "connections are required")
		}
		return &campaignapp.CampaignCharacterCreationStepInput{
			Connections: &campaignapp.CampaignCharacterCreationStepConnections{
				Connections: connections,
			},
		}, nil
	default:
		return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_step_is_not_available", "character creation step is not available")
	}
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
