package daggerheart

import (
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
		return &campaignapp.CampaignCharacterCreationStepInput{
			Details: &campaignapp.CampaignCharacterCreationStepDetails{},
		}, nil
	case 5:
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
	case 6:
		background := strings.TrimSpace(form.Get("background"))
		if background == "" {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_background_is_required", "background is required")
		}
		return &campaignapp.CampaignCharacterCreationStepInput{
			Background: &campaignapp.CampaignCharacterCreationStepBackground{
				Background: background,
			},
		}, nil
	case 7:
		experienceName := strings.TrimSpace(form.Get("experience_name"))
		if experienceName == "" {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_experience_name_is_required", "experience name is required")
		}
		experienceModifier, err := parseOptionalInt32(form.Get("experience_modifier"))
		if err != nil {
			return nil, err
		}
		return &campaignapp.CampaignCharacterCreationStepInput{
			Experiences: &campaignapp.CampaignCharacterCreationStepExperiences{
				Experiences: []campaignapp.CampaignCharacterCreationStepExperience{
					{
						Name:     experienceName,
						Modifier: experienceModifier,
					},
				},
			},
		}, nil
	case 8:
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
		if len(domainCardIDs) == 0 {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_at_least_one_domain_card_is_required", "at least one domain card is required")
		}
		return &campaignapp.CampaignCharacterCreationStepInput{
			DomainCards: &campaignapp.CampaignCharacterCreationStepDomainCards{
				DomainCardIDs: domainCardIDs,
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

// parseOptionalInt32 parses inbound values into package-safe forms.
func parseOptionalInt32(raw string) (int32, error) {
	trimmedRaw := strings.TrimSpace(raw)
	if trimmedRaw == "" {
		return 0, nil
	}
	value, err := strconv.Atoi(trimmedRaw)
	if err != nil {
		return 0, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_modifier_must_be_valid_integer", "modifier must be a valid integer")
	}
	return int32(value), nil
}
