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
func (g characterCreationMutationGateway) ApplyCharacterCreationStep(ctx context.Context, campaignID string, characterID string, step *campaignapp.CampaignCharacterCreationStepInput) error {
	if g.mutation.Character == nil {
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
	mappedStep, err := mapCampaignCharacterCreationStepToProto(step)
	if err != nil {
		return err
	}

	_, err = g.mutation.Character.ApplyCharacterCreationStep(ctx, &statev1.ApplyCharacterCreationStepRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
		SystemStep:  &statev1.ApplyCharacterCreationStepRequest_Daggerheart{Daggerheart: mappedStep},
	})
	return err
}

// mapCampaignCharacterCreationStepToProto maps values across transport and domain boundaries.
func mapCampaignCharacterCreationStepToProto(step *campaignapp.CampaignCharacterCreationStepInput) (*daggerheartv1.DaggerheartCreationStepInput, error) {
	if step == nil {
		return nil, apperrors.E(apperrors.KindInvalidInput, "character creation step is required")
	}
	if countActiveCreationStepInputs(step) != 1 {
		return nil, apperrors.EK(apperrors.KindInvalidInput, "error.web.message.character_creation_step_is_not_available", "character creation step is not available")
	}

	switch {
	case step.ClassSubclass != nil:
		return mapClassSubclassStep(step.ClassSubclass), nil
	case step.Heritage != nil:
		return mapHeritageStep(step.Heritage), nil
	case step.Traits != nil:
		return mapTraitsStep(step.Traits), nil
	case step.Details != nil:
		return mapDetailsStep(step.Details), nil
	case step.Equipment != nil:
		return mapEquipmentStep(step.Equipment), nil
	case step.Background != nil:
		return mapBackgroundStep(step.Background), nil
	case step.Experiences != nil:
		return mapExperiencesStep(step.Experiences), nil
	case step.DomainCards != nil:
		return mapDomainCardsStep(step.DomainCards), nil
	default:
		return mapConnectionsStep(step.Connections), nil
	}
}

// countActiveCreationStepInputs returns how many oneof-style step inputs are set.
func countActiveCreationStepInputs(step *campaignapp.CampaignCharacterCreationStepInput) int {
	if step == nil {
		return 0
	}
	active := 0
	for _, present := range []bool{
		step.ClassSubclass != nil,
		step.Heritage != nil,
		step.Traits != nil,
		step.Details != nil,
		step.Equipment != nil,
		step.Background != nil,
		step.Experiences != nil,
		step.DomainCards != nil,
		step.Connections != nil,
	} {
		if present {
			active++
		}
	}
	return active
}

// mapClassSubclassStep maps class/subclass selection to proto payload.
func mapClassSubclassStep(step *campaignapp.CampaignCharacterCreationStepClassSubclass) *daggerheartv1.DaggerheartCreationStepInput {
	var companion *daggerheartv1.DaggerheartCreationCompanionInput
	if step.Companion != nil {
		companion = &daggerheartv1.DaggerheartCreationCompanionInput{
			AnimalKind:        strings.TrimSpace(step.Companion.AnimalKind),
			Name:              strings.TrimSpace(step.Companion.Name),
			ExperienceIds:     trimNonEmptyStepValues(step.Companion.ExperienceIDs),
			AttackDescription: strings.TrimSpace(step.Companion.AttackDescription),
			DamageType:        strings.TrimSpace(step.Companion.DamageType),
		}
	}
	return &daggerheartv1.DaggerheartCreationStepInput{
		Step: &daggerheartv1.DaggerheartCreationStepInput_ClassSubclassInput{ClassSubclassInput: &daggerheartv1.DaggerheartCreationStepClassSubclassInput{
			ClassId:    strings.TrimSpace(step.ClassID),
			SubclassId: strings.TrimSpace(step.SubclassID),
			Companion:  companion,
		}},
	}
}

// mapHeritageStep maps heritage selection to proto payload.
func mapHeritageStep(step *campaignapp.CampaignCharacterCreationStepHeritage) *daggerheartv1.DaggerheartCreationStepInput {
	return &daggerheartv1.DaggerheartCreationStepInput{
		Step: &daggerheartv1.DaggerheartCreationStepInput_HeritageInput{HeritageInput: &daggerheartv1.DaggerheartCreationStepHeritageInput{
			Heritage: &daggerheartv1.DaggerheartCreationStepHeritageSelectionInput{
				AncestryLabel:           strings.TrimSpace(step.Heritage.AncestryLabel),
				FirstFeatureAncestryId:  strings.TrimSpace(step.Heritage.FirstFeatureAncestryID),
				SecondFeatureAncestryId: strings.TrimSpace(step.Heritage.SecondFeatureAncestryID),
				CommunityId:             strings.TrimSpace(step.Heritage.CommunityID),
			},
		}},
	}
}

// mapTraitsStep maps trait allocation to proto payload.
func mapTraitsStep(step *campaignapp.CampaignCharacterCreationStepTraits) *daggerheartv1.DaggerheartCreationStepInput {
	return &daggerheartv1.DaggerheartCreationStepInput{
		Step: &daggerheartv1.DaggerheartCreationStepInput_TraitsInput{TraitsInput: &daggerheartv1.DaggerheartCreationStepTraitsInput{
			Agility:   step.Agility,
			Strength:  step.Strength,
			Finesse:   step.Finesse,
			Instinct:  step.Instinct,
			Presence:  step.Presence,
			Knowledge: step.Knowledge,
		}},
	}
}

// mapDetailsStep maps freeform character details to proto payload.
func mapDetailsStep(step *campaignapp.CampaignCharacterCreationStepDetails) *daggerheartv1.DaggerheartCreationStepInput {
	return &daggerheartv1.DaggerheartCreationStepInput{
		Step: &daggerheartv1.DaggerheartCreationStepInput_DetailsInput{DetailsInput: &daggerheartv1.DaggerheartCreationStepDetailsInput{
			Description: strings.TrimSpace(step.Description),
		}},
	}
}

// mapEquipmentStep maps equipment selection to proto payload.
func mapEquipmentStep(step *campaignapp.CampaignCharacterCreationStepEquipment) *daggerheartv1.DaggerheartCreationStepInput {
	return &daggerheartv1.DaggerheartCreationStepInput{
		Step: &daggerheartv1.DaggerheartCreationStepInput_EquipmentInput{EquipmentInput: &daggerheartv1.DaggerheartCreationStepEquipmentInput{
			WeaponIds:    trimNonEmptyStepValues(step.WeaponIDs),
			ArmorId:      strings.TrimSpace(step.ArmorID),
			PotionItemId: strings.TrimSpace(step.PotionItemID),
		}},
	}
}

// mapBackgroundStep maps character background text to proto payload.
func mapBackgroundStep(step *campaignapp.CampaignCharacterCreationStepBackground) *daggerheartv1.DaggerheartCreationStepInput {
	return &daggerheartv1.DaggerheartCreationStepInput{
		Step: &daggerheartv1.DaggerheartCreationStepInput_BackgroundInput{BackgroundInput: &daggerheartv1.DaggerheartCreationStepBackgroundInput{
			Background: strings.TrimSpace(step.Background),
		}},
	}
}

// mapExperiencesStep maps experience entries to proto payload.
func mapExperiencesStep(step *campaignapp.CampaignCharacterCreationStepExperiences) *daggerheartv1.DaggerheartCreationStepInput {
	experiences := make([]*daggerheartv1.DaggerheartExperience, 0, len(step.Experiences))
	for _, experience := range step.Experiences {
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
	}
}

// mapDomainCardsStep maps selected domain cards to proto payload.
func mapDomainCardsStep(step *campaignapp.CampaignCharacterCreationStepDomainCards) *daggerheartv1.DaggerheartCreationStepInput {
	return &daggerheartv1.DaggerheartCreationStepInput{
		Step: &daggerheartv1.DaggerheartCreationStepInput_DomainCardsInput{DomainCardsInput: &daggerheartv1.DaggerheartCreationStepDomainCardsInput{
			DomainCardIds: trimNonEmptyStepValues(step.DomainCardIDs),
		}},
	}
}

// mapConnectionsStep maps relationship text to proto payload.
func mapConnectionsStep(step *campaignapp.CampaignCharacterCreationStepConnections) *daggerheartv1.DaggerheartCreationStepInput {
	return &daggerheartv1.DaggerheartCreationStepInput{
		Step: &daggerheartv1.DaggerheartCreationStepInput_ConnectionsInput{ConnectionsInput: &daggerheartv1.DaggerheartCreationStepConnectionsInput{
			Connections: strings.TrimSpace(step.Connections),
		}},
	}
}

// trimNonEmptyStepValues trims whitespace and drops empty values while preserving order.
func trimNonEmptyStepValues(values []string) []string {
	mapped := make([]string, 0, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			continue
		}
		mapped = append(mapped, trimmed)
	}
	return mapped
}

// MapCampaignCharacterCreationStepToProto converts a domain step into the proto payload.
func MapCampaignCharacterCreationStepToProto(step *campaignapp.CampaignCharacterCreationStepInput) (*daggerheartv1.DaggerheartCreationStepInput, error) {
	return mapCampaignCharacterCreationStepToProto(step)
}

// ResetCharacterCreationWorkflow applies this package workflow transition.
func (g characterCreationMutationGateway) ResetCharacterCreationWorkflow(ctx context.Context, campaignID string, characterID string) error {
	if g.mutation.Character == nil {
		return apperrors.EK(apperrors.KindUnavailable, "error.web.message.character_service_client_is_not_configured", "character service client is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	characterID = strings.TrimSpace(characterID)
	if campaignID == "" || characterID == "" {
		return apperrors.E(apperrors.KindInvalidInput, "campaign id and character id are required")
	}
	_, err := g.mutation.Character.ResetCharacterCreationWorkflow(ctx, &statev1.ResetCharacterCreationWorkflowRequest{
		CampaignId:  campaignID,
		CharacterId: characterID,
	})
	return err
}
