package creationworkflow

import (
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func creationStepSequenceFromWorkflowInput(input *daggerheartv1.DaggerheartCreationWorkflowInput) ([]*daggerheartv1.DaggerheartCreationStepInput, error) {
	if input == nil {
		return nil, status.Error(codes.InvalidArgument, "daggerheart workflow payload is required")
	}
	if input.GetClassSubclassInput() == nil {
		return nil, status.Error(codes.InvalidArgument, "class_subclass_input is required")
	}
	if input.GetHeritageInput() == nil {
		return nil, status.Error(codes.InvalidArgument, "heritage_input is required")
	}
	if input.GetTraitsInput() == nil {
		return nil, status.Error(codes.InvalidArgument, "traits_input is required")
	}
	if input.GetDetailsInput() == nil {
		return nil, status.Error(codes.InvalidArgument, "details_input is required")
	}
	if input.GetEquipmentInput() == nil {
		return nil, status.Error(codes.InvalidArgument, "equipment_input is required")
	}
	if input.GetBackgroundInput() == nil {
		return nil, status.Error(codes.InvalidArgument, "background_input is required")
	}
	if input.GetExperiencesInput() == nil {
		return nil, status.Error(codes.InvalidArgument, "experiences_input is required")
	}
	if input.GetDomainCardsInput() == nil {
		return nil, status.Error(codes.InvalidArgument, "domain_cards_input is required")
	}
	if input.GetConnectionsInput() == nil {
		return nil, status.Error(codes.InvalidArgument, "connections_input is required")
	}
	return []*daggerheartv1.DaggerheartCreationStepInput{
		{Step: &daggerheartv1.DaggerheartCreationStepInput_ClassSubclassInput{ClassSubclassInput: input.GetClassSubclassInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_HeritageInput{HeritageInput: input.GetHeritageInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_TraitsInput{TraitsInput: input.GetTraitsInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_EquipmentInput{EquipmentInput: input.GetEquipmentInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_ExperiencesInput{ExperiencesInput: input.GetExperiencesInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_DomainCardsInput{DomainCardsInput: input.GetDomainCardsInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_DetailsInput{DetailsInput: input.GetDetailsInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_BackgroundInput{BackgroundInput: input.GetBackgroundInput()}},
		{Step: &daggerheartv1.DaggerheartCreationStepInput_ConnectionsInput{ConnectionsInput: input.GetConnectionsInput()}},
	}, nil
}

func resetCreationWorkflowFields(profile projectionstore.DaggerheartCharacterProfile) projectionstore.DaggerheartCharacterProfile {
	profile.ClassID = ""
	profile.SubclassID = ""
	profile.AncestryID = ""
	profile.CommunityID = ""
	profile.TraitsAssigned = false
	profile.DetailsRecorded = false
	profile.StartingWeaponIDs = nil
	profile.StartingArmorID = ""
	profile.StartingPotionItemID = ""
	profile.Background = ""
	profile.Description = ""
	profile.Experiences = nil
	profile.DomainCardIDs = nil
	profile.Connections = ""
	profile.Agility = 0
	profile.Strength = 0
	profile.Finesse = 0
	profile.Instinct = 0
	profile.Presence = 0
	profile.Knowledge = 0
	return profile
}

func creationStepNumber(input *daggerheartv1.DaggerheartCreationStepInput) (int32, error) {
	if input == nil {
		return 0, status.Error(codes.InvalidArgument, "daggerheart step payload is required")
	}
	switch input.GetStep().(type) {
	case *daggerheartv1.DaggerheartCreationStepInput_ClassSubclassInput:
		return daggerheart.CreationStepClassSubclass, nil
	case *daggerheartv1.DaggerheartCreationStepInput_HeritageInput:
		return daggerheart.CreationStepHeritage, nil
	case *daggerheartv1.DaggerheartCreationStepInput_TraitsInput:
		return daggerheart.CreationStepTraits, nil
	case *daggerheartv1.DaggerheartCreationStepInput_DetailsInput:
		return daggerheart.CreationStepDetails, nil
	case *daggerheartv1.DaggerheartCreationStepInput_EquipmentInput:
		return daggerheart.CreationStepEquipment, nil
	case *daggerheartv1.DaggerheartCreationStepInput_BackgroundInput:
		return daggerheart.CreationStepBackground, nil
	case *daggerheartv1.DaggerheartCreationStepInput_ExperiencesInput:
		return daggerheart.CreationStepExperiences, nil
	case *daggerheartv1.DaggerheartCreationStepInput_DomainCardsInput:
		return daggerheart.CreationStepDomainCards, nil
	case *daggerheartv1.DaggerheartCreationStepInput_ConnectionsInput:
		return daggerheart.CreationStepConnections, nil
	default:
		return 0, status.Error(codes.InvalidArgument, "daggerheart creation step is required")
	}
}
