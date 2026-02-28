package daggerheart

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// ParseStepInput parses a character creation step form submission into domain
// step input based on the current step number.
func (Workflow) ParseStepInput(r *http.Request, nextStep int32) (*campaigns.CampaignCharacterCreationStepInput, error) {
	switch nextStep {
	case 1:
		classID := strings.TrimSpace(r.FormValue("class_id"))
		subclassID := strings.TrimSpace(r.FormValue("subclass_id"))
		if classID == "" || subclassID == "" {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_class_and_subclass_are_required", "class and subclass are required")
		}
		return &campaigns.CampaignCharacterCreationStepInput{
			ClassSubclass: &campaigns.CampaignCharacterCreationStepClassSubclass{
				ClassID:    classID,
				SubclassID: subclassID,
			},
		}, nil
	case 2:
		ancestryID := strings.TrimSpace(r.FormValue("ancestry_id"))
		communityID := strings.TrimSpace(r.FormValue("community_id"))
		if ancestryID == "" || communityID == "" {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_ancestry_and_community_are_required", "ancestry and community are required")
		}
		return &campaigns.CampaignCharacterCreationStepInput{
			Heritage: &campaigns.CampaignCharacterCreationStepHeritage{
				AncestryID:  ancestryID,
				CommunityID: communityID,
			},
		}, nil
	case 3:
		agility, err := parseRequiredInt32(r.FormValue("agility"), "agility")
		if err != nil {
			return nil, err
		}
		strength, err := parseRequiredInt32(r.FormValue("strength"), "strength")
		if err != nil {
			return nil, err
		}
		finesse, err := parseRequiredInt32(r.FormValue("finesse"), "finesse")
		if err != nil {
			return nil, err
		}
		instinct, err := parseRequiredInt32(r.FormValue("instinct"), "instinct")
		if err != nil {
			return nil, err
		}
		presence, err := parseRequiredInt32(r.FormValue("presence"), "presence")
		if err != nil {
			return nil, err
		}
		knowledge, err := parseRequiredInt32(r.FormValue("knowledge"), "knowledge")
		if err != nil {
			return nil, err
		}
		return &campaigns.CampaignCharacterCreationStepInput{
			Traits: &campaigns.CampaignCharacterCreationStepTraits{
				Agility:   agility,
				Strength:  strength,
				Finesse:   finesse,
				Instinct:  instinct,
				Presence:  presence,
				Knowledge: knowledge,
			},
		}, nil
	case 4:
		return &campaigns.CampaignCharacterCreationStepInput{
			Details: &campaigns.CampaignCharacterCreationStepDetails{},
		}, nil
	case 5:
		primaryWeaponID := strings.TrimSpace(r.FormValue("weapon_primary_id"))
		secondaryWeaponID := strings.TrimSpace(r.FormValue("weapon_secondary_id"))
		armorID := strings.TrimSpace(r.FormValue("armor_id"))
		potionItemID := strings.TrimSpace(r.FormValue("potion_item_id"))
		if primaryWeaponID == "" || armorID == "" || potionItemID == "" {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_primary_weapon_armor_and_potion_are_required", "primary weapon, armor, and potion are required")
		}
		weaponIDs := []string{primaryWeaponID}
		if secondaryWeaponID != "" {
			weaponIDs = append(weaponIDs, secondaryWeaponID)
		}
		return &campaigns.CampaignCharacterCreationStepInput{
			Equipment: &campaigns.CampaignCharacterCreationStepEquipment{
				WeaponIDs:    weaponIDs,
				ArmorID:      armorID,
				PotionItemID: potionItemID,
			},
		}, nil
	case 6:
		background := strings.TrimSpace(r.FormValue("background"))
		if background == "" {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_background_is_required", "background is required")
		}
		return &campaigns.CampaignCharacterCreationStepInput{
			Background: &campaigns.CampaignCharacterCreationStepBackground{
				Background: background,
			},
		}, nil
	case 7:
		experienceName := strings.TrimSpace(r.FormValue("experience_name"))
		if experienceName == "" {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_experience_name_is_required", "experience name is required")
		}
		experienceModifier, err := parseOptionalInt32(r.FormValue("experience_modifier"))
		if err != nil {
			return nil, err
		}
		return &campaigns.CampaignCharacterCreationStepInput{
			Experiences: &campaigns.CampaignCharacterCreationStepExperiences{
				Experiences: []campaigns.CampaignCharacterCreationStepExperience{
					{
						Name:     experienceName,
						Modifier: experienceModifier,
					},
				},
			},
		}, nil
	case 8:
		if r.Form == nil {
			r.ParseForm()
		}
		rawDomainCardIDs := r.Form["domain_card_id"]
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
		return &campaigns.CampaignCharacterCreationStepInput{
			DomainCards: &campaigns.CampaignCharacterCreationStepDomainCards{
				DomainCardIDs: domainCardIDs,
			},
		}, nil
	case 9:
		connections := strings.TrimSpace(r.FormValue("connections"))
		if connections == "" {
			return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_connections_are_required", "connections are required")
		}
		return &campaigns.CampaignCharacterCreationStepInput{
			Connections: &campaigns.CampaignCharacterCreationStepConnections{
				Connections: connections,
			},
		}, nil
	default:
		return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_step_is_not_available", "character creation step is not available")
	}
}

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
