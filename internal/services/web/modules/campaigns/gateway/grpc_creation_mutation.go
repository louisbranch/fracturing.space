package gateway

import (
	"context"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
)

// ApplyCharacterCreationStep applies this package workflow transition.
func (g GRPCGateway) ApplyCharacterCreationStep(ctx context.Context, campaignID string, characterID string, step *campaignapp.CampaignCharacterCreationStepInput) error {
	if g.CharacterClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}
	if step == nil {
		return apperrors.E(apperrors.KindInvalidInput, "character creation step is required")
	}
	systemStep, err := mapCampaignCharacterCreationStepToProto(step)
	if err != nil {
		return err
	}

	_, err = g.CharacterClient.ApplyCharacterCreationStep(ctx, &statev1.ApplyCharacterCreationStepRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
		SystemStep:  &statev1.ApplyCharacterCreationStepRequest_Daggerheart{Daggerheart: systemStep},
	})
	return err
}

// mapCampaignCharacterCreationStepToProto maps values across transport and domain boundaries.
func mapCampaignCharacterCreationStepToProto(step *campaignapp.CampaignCharacterCreationStepInput) (*daggerheartv1.DaggerheartCreationStepInput, error) {
	if step == nil {
		return nil, apperrors.E(apperrors.KindInvalidInput, "character creation step is required")
	}

	active := 0
	if step.ClassSubclass != nil {
		active++
	}
	if step.Heritage != nil {
		active++
	}
	if step.Traits != nil {
		active++
	}
	if step.Details != nil {
		active++
	}
	if step.Equipment != nil {
		active++
	}
	if step.Background != nil {
		active++
	}
	if step.Experiences != nil {
		active++
	}
	if step.DomainCards != nil {
		active++
	}
	if step.Connections != nil {
		active++
	}
	if active != 1 {
		return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_step_is_not_available", "character creation step is not available")
	}

	if step.ClassSubclass != nil {
		return &daggerheartv1.DaggerheartCreationStepInput{
			Step: &daggerheartv1.DaggerheartCreationStepInput_ClassSubclassInput{ClassSubclassInput: &daggerheartv1.DaggerheartCreationStepClassSubclassInput{
				ClassId:    strings.TrimSpace(step.ClassSubclass.ClassID),
				SubclassId: strings.TrimSpace(step.ClassSubclass.SubclassID),
			}},
		}, nil
	}
	if step.Heritage != nil {
		return &daggerheartv1.DaggerheartCreationStepInput{
			Step: &daggerheartv1.DaggerheartCreationStepInput_HeritageInput{HeritageInput: &daggerheartv1.DaggerheartCreationStepHeritageInput{
				AncestryId:  strings.TrimSpace(step.Heritage.AncestryID),
				CommunityId: strings.TrimSpace(step.Heritage.CommunityID),
			}},
		}, nil
	}
	if step.Traits != nil {
		return &daggerheartv1.DaggerheartCreationStepInput{
			Step: &daggerheartv1.DaggerheartCreationStepInput_TraitsInput{TraitsInput: &daggerheartv1.DaggerheartCreationStepTraitsInput{
				Agility:   step.Traits.Agility,
				Strength:  step.Traits.Strength,
				Finesse:   step.Traits.Finesse,
				Instinct:  step.Traits.Instinct,
				Presence:  step.Traits.Presence,
				Knowledge: step.Traits.Knowledge,
			}},
		}, nil
	}
	if step.Details != nil {
		return &daggerheartv1.DaggerheartCreationStepInput{
			Step: &daggerheartv1.DaggerheartCreationStepInput_DetailsInput{DetailsInput: &daggerheartv1.DaggerheartCreationStepDetailsInput{}},
		}, nil
	}
	if step.Equipment != nil {
		weaponIDs := make([]string, 0, len(step.Equipment.WeaponIDs))
		for _, weaponID := range step.Equipment.WeaponIDs {
			trimmedWeaponID := strings.TrimSpace(weaponID)
			if trimmedWeaponID == "" {
				continue
			}
			weaponIDs = append(weaponIDs, trimmedWeaponID)
		}
		return &daggerheartv1.DaggerheartCreationStepInput{
			Step: &daggerheartv1.DaggerheartCreationStepInput_EquipmentInput{EquipmentInput: &daggerheartv1.DaggerheartCreationStepEquipmentInput{
				WeaponIds:    weaponIDs,
				ArmorId:      strings.TrimSpace(step.Equipment.ArmorID),
				PotionItemId: strings.TrimSpace(step.Equipment.PotionItemID),
			}},
		}, nil
	}
	if step.Background != nil {
		return &daggerheartv1.DaggerheartCreationStepInput{
			Step: &daggerheartv1.DaggerheartCreationStepInput_BackgroundInput{BackgroundInput: &daggerheartv1.DaggerheartCreationStepBackgroundInput{
				Background: strings.TrimSpace(step.Background.Background),
			}},
		}, nil
	}
	if step.Experiences != nil {
		experiences := make([]*daggerheartv1.DaggerheartExperience, 0, len(step.Experiences.Experiences))
		for _, experience := range step.Experiences.Experiences {
			experienceName := strings.TrimSpace(experience.Name)
			if experienceName == "" {
				continue
			}
			experiences = append(experiences, &daggerheartv1.DaggerheartExperience{
				Name:     experienceName,
				Modifier: experience.Modifier,
			})
		}
		return &daggerheartv1.DaggerheartCreationStepInput{
			Step: &daggerheartv1.DaggerheartCreationStepInput_ExperiencesInput{ExperiencesInput: &daggerheartv1.DaggerheartCreationStepExperiencesInput{
				Experiences: experiences,
			}},
		}, nil
	}
	if step.DomainCards != nil {
		domainCardIDs := make([]string, 0, len(step.DomainCards.DomainCardIDs))
		for _, domainCardID := range step.DomainCards.DomainCardIDs {
			trimmedDomainCardID := strings.TrimSpace(domainCardID)
			if trimmedDomainCardID == "" {
				continue
			}
			domainCardIDs = append(domainCardIDs, trimmedDomainCardID)
		}
		return &daggerheartv1.DaggerheartCreationStepInput{
			Step: &daggerheartv1.DaggerheartCreationStepInput_DomainCardsInput{DomainCardsInput: &daggerheartv1.DaggerheartCreationStepDomainCardsInput{
				DomainCardIds: domainCardIDs,
			}},
		}, nil
	}
	return &daggerheartv1.DaggerheartCreationStepInput{
		Step: &daggerheartv1.DaggerheartCreationStepInput_ConnectionsInput{ConnectionsInput: &daggerheartv1.DaggerheartCreationStepConnectionsInput{
			Connections: strings.TrimSpace(step.Connections.Connections),
		}},
	}, nil
}

// MapCampaignCharacterCreationStepToProto converts a domain step into the proto payload.
func MapCampaignCharacterCreationStepToProto(step *campaignapp.CampaignCharacterCreationStepInput) (*daggerheartv1.DaggerheartCreationStepInput, error) {
	return mapCampaignCharacterCreationStepToProto(step)
}

// ResetCharacterCreationWorkflow applies this package workflow transition.
func (g GRPCGateway) ResetCharacterCreationWorkflow(ctx context.Context, campaignID string, characterID string) error {
	if g.CharacterClient == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}
	_, err := g.CharacterClient.ResetCharacterCreationWorkflow(ctx, &statev1.ResetCharacterCreationWorkflowRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	return err
}
