package charactertransport

import (
	"fmt"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	sharedpronouns "github.com/louisbranch/fracturing.space/internal/services/shared/pronouns"
	"google.golang.org/protobuf/types/known/timestamppb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// CharacterToProto converts a character projection record into its protobuf
// read model.
func CharacterToProto(record storage.CharacterRecord) *campaignv1.Character {
	pb := &campaignv1.Character{
		Id:            record.ID,
		CampaignId:    record.CampaignID,
		Name:          record.Name,
		Kind:          KindToProto(record.Kind),
		Notes:         record.Notes,
		AvatarSetId:   record.AvatarSetID,
		AvatarAssetId: record.AvatarAssetID,
		Pronouns:      sharedpronouns.ToProto(record.Pronouns),
		Aliases:       append([]string(nil), record.Aliases...),
		CreatedAt:     timestamppb.New(record.CreatedAt),
		UpdatedAt:     timestamppb.New(record.UpdatedAt),
	}
	if strings.TrimSpace(record.ParticipantID) != "" {
		pb.ParticipantId = wrapperspb.String(record.ParticipantID)
	}
	return pb
}

// KindFromProto converts a protobuf character kind to the domain value.
func KindFromProto(kind campaignv1.CharacterKind) character.Kind {
	switch kind {
	case campaignv1.CharacterKind_PC:
		return character.KindPC
	case campaignv1.CharacterKind_NPC:
		return character.KindNPC
	default:
		return character.KindUnspecified
	}
}

// KindToProto converts a domain character kind to the protobuf enum.
func KindToProto(kind character.Kind) campaignv1.CharacterKind {
	switch kind {
	case character.KindPC:
		return campaignv1.CharacterKind_PC
	case character.KindNPC:
		return campaignv1.CharacterKind_NPC
	default:
		return campaignv1.CharacterKind_CHARACTER_KIND_UNSPECIFIED
	}
}

// DaggerheartProfileToProto converts a Daggerheart profile projection into the
// game protobuf read model.
func DaggerheartProfileToProto(campaignID, characterID string, profile storage.DaggerheartCharacterProfile) *campaignv1.CharacterProfile {
	return &campaignv1.CharacterProfile{
		CampaignId:  campaignID,
		CharacterId: characterID,
		SystemProfile: &campaignv1.CharacterProfile_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartProfile{
				Level:                int32(profile.Level),
				HpMax:                int32(profile.HpMax),
				StressMax:            wrapperspb.Int32(int32(profile.StressMax)),
				Evasion:              wrapperspb.Int32(int32(profile.Evasion)),
				MajorThreshold:       wrapperspb.Int32(int32(profile.MajorThreshold)),
				SevereThreshold:      wrapperspb.Int32(int32(profile.SevereThreshold)),
				Proficiency:          wrapperspb.Int32(int32(profile.Proficiency)),
				ArmorScore:           wrapperspb.Int32(int32(profile.ArmorScore)),
				ArmorMax:             wrapperspb.Int32(int32(profile.ArmorMax)),
				Agility:              wrapperspb.Int32(int32(profile.Agility)),
				Strength:             wrapperspb.Int32(int32(profile.Strength)),
				Finesse:              wrapperspb.Int32(int32(profile.Finesse)),
				Instinct:             wrapperspb.Int32(int32(profile.Instinct)),
				Presence:             wrapperspb.Int32(int32(profile.Presence)),
				Knowledge:            wrapperspb.Int32(int32(profile.Knowledge)),
				Experiences:          DaggerheartExperiencesToProto(profile.Experiences),
				ClassId:              profile.ClassID,
				SubclassId:           profile.SubclassID,
				AncestryId:           profile.AncestryID,
				CommunityId:          profile.CommunityID,
				TraitsAssigned:       wrapperspb.Bool(profile.TraitsAssigned),
				DetailsRecorded:      wrapperspb.Bool(profile.DetailsRecorded),
				StartingWeaponIds:    append([]string(nil), profile.StartingWeaponIDs...),
				StartingArmorId:      profile.StartingArmorID,
				StartingPotionItemId: profile.StartingPotionItemID,
				Background:           profile.Background,
				Description:          profile.Description,
				DomainCardIds:        append([]string(nil), profile.DomainCardIDs...),
				Connections:          profile.Connections,
			},
		},
	}
}

// DaggerheartStateToProto converts a Daggerheart state projection into the game
// protobuf read model.
func DaggerheartStateToProto(campaignID, characterID string, state storage.DaggerheartCharacterState) *campaignv1.CharacterState {
	return &campaignv1.CharacterState{
		CampaignId:  campaignID,
		CharacterId: characterID,
		SystemState: &campaignv1.CharacterState_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCharacterState{
				Hp:         int32(state.Hp),
				Hope:       int32(state.Hope),
				HopeMax:    int32(state.HopeMax),
				Stress:     int32(state.Stress),
				Armor:      int32(state.Armor),
				Conditions: DaggerheartConditionsToProto(state.Conditions),
				LifeState:  DaggerheartLifeStateToProto(state.LifeState),
			},
		},
	}
}

// DaggerheartExperiencesToProto converts Daggerheart experiences to the system
// protobuf read model.
func DaggerheartExperiencesToProto(experiences []storage.DaggerheartExperience) []*daggerheartv1.DaggerheartExperience {
	if len(experiences) == 0 {
		return nil
	}
	result := make([]*daggerheartv1.DaggerheartExperience, 0, len(experiences))
	for _, experience := range experiences {
		result = append(result, &daggerheartv1.DaggerheartExperience{
			Name:     experience.Name,
			Modifier: int32(experience.Modifier),
		})
	}
	return result
}

// DaggerheartConditionsFromProto converts system protobuf conditions to domain
// strings.
func DaggerheartConditionsFromProto(conditions []daggerheartv1.DaggerheartCondition) ([]string, error) {
	if len(conditions) == 0 {
		return []string{}, nil
	}

	result := make([]string, 0, len(conditions))
	for _, condition := range conditions {
		switch condition {
		case daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_UNSPECIFIED:
			return nil, fmt.Errorf("condition is required")
		case daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN:
			result = append(result, daggerheart.ConditionHidden)
		case daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_RESTRAINED:
			result = append(result, daggerheart.ConditionRestrained)
		case daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE:
			result = append(result, daggerheart.ConditionVulnerable)
		default:
			return nil, fmt.Errorf("condition %v is invalid", condition)
		}
	}
	return result, nil
}

// DaggerheartConditionsToProto converts domain condition strings to the system
// protobuf enum values.
func DaggerheartConditionsToProto(conditions []string) []daggerheartv1.DaggerheartCondition {
	if len(conditions) == 0 {
		return nil
	}
	result := make([]daggerheartv1.DaggerheartCondition, 0, len(conditions))
	for _, condition := range conditions {
		switch condition {
		case daggerheart.ConditionHidden:
			result = append(result, daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN)
		case daggerheart.ConditionRestrained:
			result = append(result, daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_RESTRAINED)
		case daggerheart.ConditionVulnerable:
			result = append(result, daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE)
		}
	}
	return result
}

// DaggerheartLifeStateFromProto converts a system protobuf life-state enum to
// the domain string.
func DaggerheartLifeStateFromProto(state daggerheartv1.DaggerheartLifeState) (string, error) {
	switch state {
	case daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED:
		return "", fmt.Errorf("life_state is required")
	case daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE:
		return daggerheart.LifeStateAlive, nil
	case daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS:
		return daggerheart.LifeStateUnconscious, nil
	case daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY:
		return daggerheart.LifeStateBlazeOfGlory, nil
	case daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD:
		return daggerheart.LifeStateDead, nil
	default:
		return "", fmt.Errorf("life_state %v is invalid", state)
	}
}

// DaggerheartLifeStateToProto converts a domain life-state string to the system
// protobuf enum.
func DaggerheartLifeStateToProto(state string) daggerheartv1.DaggerheartLifeState {
	switch state {
	case daggerheart.LifeStateAlive:
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE
	case daggerheart.LifeStateUnconscious:
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNCONSCIOUS
	case daggerheart.LifeStateBlazeOfGlory:
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_BLAZE_OF_GLORY
	case daggerheart.LifeStateDead:
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_DEAD
	default:
		return daggerheartv1.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_UNSPECIFIED
	}
}
