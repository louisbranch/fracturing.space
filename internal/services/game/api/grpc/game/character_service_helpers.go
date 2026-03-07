package game

import (
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// daggerheartProfileToProto converts a Daggerheart profile to proto.
func daggerheartProfileToProto(campaignID, characterID string, dh storage.DaggerheartCharacterProfile) *campaignv1.CharacterProfile {
	return &campaignv1.CharacterProfile{
		CampaignId:  campaignID,
		CharacterId: characterID,
		SystemProfile: &campaignv1.CharacterProfile_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartProfile{
				Level:                int32(dh.Level),
				HpMax:                int32(dh.HpMax),
				StressMax:            wrapperspb.Int32(int32(dh.StressMax)),
				Evasion:              wrapperspb.Int32(int32(dh.Evasion)),
				MajorThreshold:       wrapperspb.Int32(int32(dh.MajorThreshold)),
				SevereThreshold:      wrapperspb.Int32(int32(dh.SevereThreshold)),
				Proficiency:          wrapperspb.Int32(int32(dh.Proficiency)),
				ArmorScore:           wrapperspb.Int32(int32(dh.ArmorScore)),
				ArmorMax:             wrapperspb.Int32(int32(dh.ArmorMax)),
				Agility:              wrapperspb.Int32(int32(dh.Agility)),
				Strength:             wrapperspb.Int32(int32(dh.Strength)),
				Finesse:              wrapperspb.Int32(int32(dh.Finesse)),
				Instinct:             wrapperspb.Int32(int32(dh.Instinct)),
				Presence:             wrapperspb.Int32(int32(dh.Presence)),
				Knowledge:            wrapperspb.Int32(int32(dh.Knowledge)),
				Experiences:          daggerheartExperiencesToProto(dh.Experiences),
				ClassId:              dh.ClassID,
				SubclassId:           dh.SubclassID,
				AncestryId:           dh.AncestryID,
				CommunityId:          dh.CommunityID,
				TraitsAssigned:       wrapperspb.Bool(dh.TraitsAssigned),
				DetailsRecorded:      wrapperspb.Bool(dh.DetailsRecorded),
				StartingWeaponIds:    append([]string(nil), dh.StartingWeaponIDs...),
				StartingArmorId:      dh.StartingArmorID,
				StartingPotionItemId: dh.StartingPotionItemID,
				Background:           dh.Background,
				Description:          dh.Description,
				DomainCardIds:        append([]string(nil), dh.DomainCardIDs...),
				Connections:          dh.Connections,
			},
		},
	}
}

// daggerheartStateToProto converts a Daggerheart state to proto.
func daggerheartStateToProto(campaignID, characterID string, dh storage.DaggerheartCharacterState) *campaignv1.CharacterState {
	return &campaignv1.CharacterState{
		CampaignId:  campaignID,
		CharacterId: characterID,
		SystemState: &campaignv1.CharacterState_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCharacterState{
				Hp:         int32(dh.Hp),
				Hope:       int32(dh.Hope),
				HopeMax:    int32(dh.HopeMax),
				Stress:     int32(dh.Stress),
				Armor:      int32(dh.Armor),
				Conditions: daggerheartConditionsToProto(dh.Conditions),
				LifeState:  daggerheartLifeStateToProto(dh.LifeState),
			},
		},
	}
}

func daggerheartExperiencesToProto(experiences []storage.DaggerheartExperience) []*daggerheartv1.DaggerheartExperience {
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
