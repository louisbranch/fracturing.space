package charactertransport

import (
	"context"
	"fmt"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
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
func DaggerheartProfileToProto(campaignID, characterID string, profile projectionstore.DaggerheartCharacterProfile, content contentstore.DaggerheartContentReadStore) *campaignv1.CharacterProfile {
	activeFeatures := daggerheartActiveSubclassFeaturesToProto(content, profile)
	activeClassFeatures := daggerheartActiveClassFeaturesToProto(content, profile)
	return &campaignv1.CharacterProfile{
		CampaignId:  campaignID,
		CharacterId: characterID,
		SystemProfile: &campaignv1.CharacterProfile_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartProfile{
				Level:                        int32(profile.Level),
				HpMax:                        int32(profile.HpMax),
				StressMax:                    wrapperspb.Int32(int32(profile.StressMax)),
				Evasion:                      wrapperspb.Int32(int32(profile.Evasion)),
				MajorThreshold:               wrapperspb.Int32(int32(profile.MajorThreshold)),
				SevereThreshold:              wrapperspb.Int32(int32(profile.SevereThreshold)),
				Proficiency:                  wrapperspb.Int32(int32(profile.Proficiency)),
				ArmorScore:                   wrapperspb.Int32(int32(profile.ArmorScore)),
				ArmorMax:                     wrapperspb.Int32(int32(profile.ArmorMax)),
				Agility:                      wrapperspb.Int32(int32(profile.Agility)),
				Strength:                     wrapperspb.Int32(int32(profile.Strength)),
				Finesse:                      wrapperspb.Int32(int32(profile.Finesse)),
				Instinct:                     wrapperspb.Int32(int32(profile.Instinct)),
				Presence:                     wrapperspb.Int32(int32(profile.Presence)),
				Knowledge:                    wrapperspb.Int32(int32(profile.Knowledge)),
				Experiences:                  DaggerheartExperiencesToProto(profile.Experiences),
				ClassId:                      profile.ClassID,
				SubclassId:                   profile.SubclassID,
				SubclassCreationRequirements: daggerheartSubclassCreationRequirementsToProto(profile.SubclassCreationRequirements),
				Heritage:                     daggerheartHeritageToProto(profile.Heritage),
				CompanionSheet:               daggerheartCompanionSheetToProto(profile.CompanionSheet, content),
				EquippedArmorId:              profile.EquippedArmorID,
				SpellcastRollBonus:           wrapperspb.Int32(int32(profile.SpellcastRollBonus)),
				SubclassTracks:               daggerheartSubclassTracksToProto(profile.SubclassTracks),
				ActiveSubclassFeatures:       activeFeatures,
				ActiveClassFeatures:          activeClassFeatures,
				TraitsAssigned:               wrapperspb.Bool(profile.TraitsAssigned),
				DetailsRecorded:              wrapperspb.Bool(profile.DetailsRecorded),
				StartingWeaponIds:            append([]string(nil), profile.StartingWeaponIDs...),
				StartingArmorId:              profile.StartingArmorID,
				StartingPotionItemId:         profile.StartingPotionItemID,
				Background:                   profile.Background,
				Description:                  profile.Description,
				DomainCardIds:                append([]string(nil), profile.DomainCardIDs...),
				Connections:                  profile.Connections,
			},
		},
	}
}

func daggerheartActiveClassFeaturesToProto(content contentstore.DaggerheartContentReadStore, profile projectionstore.DaggerheartCharacterProfile) []*daggerheartv1.DaggerheartActiveClassFeature {
	if content == nil {
		return nil
	}
	classID := strings.TrimSpace(profile.ClassID)
	if classID == "" {
		return nil
	}
	classEntry, err := content.GetDaggerheartClass(context.Background(), classID)
	if err != nil {
		return nil
	}
	items := make([]*daggerheartv1.DaggerheartActiveClassFeature, 0, len(classEntry.Features)+1)
	for _, feature := range classEntry.Features {
		items = append(items, &daggerheartv1.DaggerheartActiveClassFeature{
			Id:          feature.ID,
			Name:        feature.Name,
			Description: feature.Description,
			Level:       int32(feature.Level),
		})
	}
	if strings.TrimSpace(classEntry.HopeFeature.Name) != "" {
		items = append(items, &daggerheartv1.DaggerheartActiveClassFeature{
			Id:          "hope_feature:" + classEntry.ID,
			Name:        classEntry.HopeFeature.Name,
			Description: classEntry.HopeFeature.Description,
			Level:       1,
			HopeFeature: true,
		})
	}
	if len(items) == 0 {
		return nil
	}
	return items
}

func daggerheartSubclassTracksToProto(tracks []projectionstore.DaggerheartSubclassTrack) []*daggerheartv1.DaggerheartSubclassTrack {
	if len(tracks) == 0 {
		return nil
	}
	items := make([]*daggerheartv1.DaggerheartSubclassTrack, 0, len(tracks))
	for _, track := range tracks {
		items = append(items, &daggerheartv1.DaggerheartSubclassTrack{
			Origin:     daggerheartSubclassTrackOriginToProto(track.Origin),
			ClassId:    track.ClassID,
			SubclassId: track.SubclassID,
			Rank:       daggerheartSubclassTrackRankToProto(track.Rank),
			DomainId:   track.DomainID,
		})
	}
	return items
}

func daggerheartActiveSubclassFeaturesToProto(content contentstore.DaggerheartContentReadStore, profile projectionstore.DaggerheartCharacterProfile) []*daggerheartv1.DaggerheartActiveSubclassTrackFeatures {
	if content == nil || len(profile.SubclassTracks) == 0 {
		return nil
	}
	typed := daggerheart.CharacterProfileFromStorage(profile)
	sets, err := daggerheart.ActiveSubclassTrackFeaturesFromStore(context.Background(), content, typed.SubclassTracks)
	if err != nil {
		return nil
	}
	items := make([]*daggerheartv1.DaggerheartActiveSubclassTrackFeatures, 0, len(sets))
	for _, set := range sets {
		items = append(items, &daggerheartv1.DaggerheartActiveSubclassTrackFeatures{
			Track: &daggerheartv1.DaggerheartSubclassTrack{
				Origin:     daggerheartSubclassTrackOriginToProto(projectionstore.DaggerheartSubclassTrackOrigin(set.Track.Origin)),
				ClassId:    set.Track.ClassID,
				SubclassId: set.Track.SubclassID,
				Rank:       daggerheartSubclassTrackRankToProto(projectionstore.DaggerheartSubclassTrackRank(set.Track.Rank)),
				DomainId:   set.Track.DomainID,
			},
			FoundationFeatures:     daggerheartActiveSubclassFeaturesListToProto(set.FoundationFeatures),
			SpecializationFeatures: daggerheartActiveSubclassFeaturesListToProto(set.SpecializationFeatures),
			MasteryFeatures:        daggerheartActiveSubclassFeaturesListToProto(set.MasteryFeatures),
		})
	}
	return items
}

func daggerheartActiveSubclassFeaturesListToProto(features []contentstore.DaggerheartFeature) []*daggerheartv1.DaggerheartActiveSubclassFeature {
	if len(features) == 0 {
		return nil
	}
	items := make([]*daggerheartv1.DaggerheartActiveSubclassFeature, 0, len(features))
	for _, feature := range features {
		items = append(items, &daggerheartv1.DaggerheartActiveSubclassFeature{
			Id:          feature.ID,
			Name:        feature.Name,
			Description: feature.Description,
			Level:       int32(feature.Level),
		})
	}
	return items
}

func daggerheartFeatureAutomationStatusToProto(status contentstore.DaggerheartFeatureAutomationStatus) daggerheartv1.DaggerheartFeatureAutomationStatus {
	switch status {
	case contentstore.DaggerheartFeatureAutomationStatusSupported:
		return daggerheartv1.DaggerheartFeatureAutomationStatus_DAGGERHEART_FEATURE_AUTOMATION_STATUS_SUPPORTED
	case contentstore.DaggerheartFeatureAutomationStatusUnsupported:
		return daggerheartv1.DaggerheartFeatureAutomationStatus_DAGGERHEART_FEATURE_AUTOMATION_STATUS_UNSUPPORTED
	default:
		return daggerheartv1.DaggerheartFeatureAutomationStatus_DAGGERHEART_FEATURE_AUTOMATION_STATUS_UNSPECIFIED
	}
}

func daggerheartSubclassFeatureRuleToProto(rule *contentstore.DaggerheartSubclassFeatureRule) *daggerheartv1.DaggerheartSubclassFeatureRule {
	if rule == nil {
		return nil
	}
	return &daggerheartv1.DaggerheartSubclassFeatureRule{
		Kind:              daggerheartSubclassFeatureRuleKindToProto(rule.Kind),
		Bonus:             int32(rule.Bonus),
		RequiredHopeMin:   int32(rule.RequiredHopeMin),
		DamageDiceCount:   int32(rule.DamageDiceCount),
		DamageDieSides:    int32(rule.DamageDieSides),
		UseCharacterLevel: rule.UseCharacterLevel,
		ThresholdScope:    daggerheartSubclassThresholdScopeToProto(rule.ThresholdScope),
	}
}

func daggerheartSubclassFeatureRuleKindToProto(kind contentstore.DaggerheartSubclassFeatureRuleKind) daggerheartv1.DaggerheartSubclassFeatureRuleKind {
	switch kind {
	case contentstore.DaggerheartSubclassFeatureRuleKindThresholdBonus:
		return daggerheartv1.DaggerheartSubclassFeatureRuleKind_DAGGERHEART_SUBCLASS_FEATURE_RULE_KIND_THRESHOLD_BONUS
	case contentstore.DaggerheartSubclassFeatureRuleKindHPSlotBonus:
		return daggerheartv1.DaggerheartSubclassFeatureRuleKind_DAGGERHEART_SUBCLASS_FEATURE_RULE_KIND_HP_SLOT_BONUS
	case contentstore.DaggerheartSubclassFeatureRuleKindStressSlotBonus:
		return daggerheartv1.DaggerheartSubclassFeatureRuleKind_DAGGERHEART_SUBCLASS_FEATURE_RULE_KIND_STRESS_SLOT_BONUS
	case contentstore.DaggerheartSubclassFeatureRuleKindEvasionBonus:
		return daggerheartv1.DaggerheartSubclassFeatureRuleKind_DAGGERHEART_SUBCLASS_FEATURE_RULE_KIND_EVASION_BONUS
	case contentstore.DaggerheartSubclassFeatureRuleKindEvasionBonusWhileHopeAtLeast:
		return daggerheartv1.DaggerheartSubclassFeatureRuleKind_DAGGERHEART_SUBCLASS_FEATURE_RULE_KIND_EVASION_BONUS_WHILE_HOPE_AT_LEAST
	case contentstore.DaggerheartSubclassFeatureRuleKindGainHopeOnFailureWithFear:
		return daggerheartv1.DaggerheartSubclassFeatureRuleKind_DAGGERHEART_SUBCLASS_FEATURE_RULE_KIND_GAIN_HOPE_ON_FAILURE_WITH_FEAR
	case contentstore.DaggerheartSubclassFeatureRuleKindBonusMagicDamageOnSuccessWithFear:
		return daggerheartv1.DaggerheartSubclassFeatureRuleKind_DAGGERHEART_SUBCLASS_FEATURE_RULE_KIND_BONUS_MAGIC_DAMAGE_ON_SUCCESS_WITH_FEAR
	case contentstore.DaggerheartSubclassFeatureRuleKindBonusDamageWhileVulnerable:
		return daggerheartv1.DaggerheartSubclassFeatureRuleKind_DAGGERHEART_SUBCLASS_FEATURE_RULE_KIND_BONUS_DAMAGE_WHILE_VULNERABLE
	default:
		return daggerheartv1.DaggerheartSubclassFeatureRuleKind_DAGGERHEART_SUBCLASS_FEATURE_RULE_KIND_UNSPECIFIED
	}
}

func daggerheartSubclassThresholdScopeToProto(scope contentstore.DaggerheartSubclassThresholdScope) daggerheartv1.DaggerheartSubclassThresholdScope {
	switch scope {
	case contentstore.DaggerheartSubclassThresholdScopeAll:
		return daggerheartv1.DaggerheartSubclassThresholdScope_DAGGERHEART_SUBCLASS_THRESHOLD_SCOPE_ALL
	case contentstore.DaggerheartSubclassThresholdScopeSevereOnly:
		return daggerheartv1.DaggerheartSubclassThresholdScope_DAGGERHEART_SUBCLASS_THRESHOLD_SCOPE_SEVERE_ONLY
	default:
		return daggerheartv1.DaggerheartSubclassThresholdScope_DAGGERHEART_SUBCLASS_THRESHOLD_SCOPE_UNSPECIFIED
	}
}

func daggerheartSubclassTrackOriginToProto(origin projectionstore.DaggerheartSubclassTrackOrigin) daggerheartv1.DaggerheartSubclassTrackOrigin {
	switch origin {
	case projectionstore.DaggerheartSubclassTrackOriginPrimary:
		return daggerheartv1.DaggerheartSubclassTrackOrigin_DAGGERHEART_SUBCLASS_TRACK_ORIGIN_PRIMARY
	case projectionstore.DaggerheartSubclassTrackOriginMulticlass:
		return daggerheartv1.DaggerheartSubclassTrackOrigin_DAGGERHEART_SUBCLASS_TRACK_ORIGIN_MULTICLASS
	default:
		return daggerheartv1.DaggerheartSubclassTrackOrigin_DAGGERHEART_SUBCLASS_TRACK_ORIGIN_UNSPECIFIED
	}
}

func daggerheartSubclassTrackRankToProto(rank projectionstore.DaggerheartSubclassTrackRank) daggerheartv1.DaggerheartSubclassTrackRank {
	switch rank {
	case projectionstore.DaggerheartSubclassTrackRankFoundation:
		return daggerheartv1.DaggerheartSubclassTrackRank_DAGGERHEART_SUBCLASS_TRACK_RANK_FOUNDATION
	case projectionstore.DaggerheartSubclassTrackRankSpecialization:
		return daggerheartv1.DaggerheartSubclassTrackRank_DAGGERHEART_SUBCLASS_TRACK_RANK_SPECIALIZATION
	case projectionstore.DaggerheartSubclassTrackRankMastery:
		return daggerheartv1.DaggerheartSubclassTrackRank_DAGGERHEART_SUBCLASS_TRACK_RANK_MASTERY
	default:
		return daggerheartv1.DaggerheartSubclassTrackRank_DAGGERHEART_SUBCLASS_TRACK_RANK_UNSPECIFIED
	}
}

func daggerheartSubclassCreationRequirementsToProto(requirements []projectionstore.DaggerheartSubclassCreationRequirement) []string {
	if len(requirements) == 0 {
		return nil
	}
	items := make([]string, 0, len(requirements))
	for _, requirement := range requirements {
		items = append(items, string(requirement))
	}
	return items
}

func daggerheartHeritageToProto(heritage projectionstore.DaggerheartHeritageSelection) *daggerheartv1.DaggerheartHeritageSelection {
	if heritage == (projectionstore.DaggerheartHeritageSelection{}) {
		return nil
	}
	return &daggerheartv1.DaggerheartHeritageSelection{
		AncestryLabel:           heritage.AncestryLabel,
		FirstFeatureAncestryId:  heritage.FirstFeatureAncestryID,
		FirstFeatureId:          heritage.FirstFeatureID,
		SecondFeatureAncestryId: heritage.SecondFeatureAncestryID,
		SecondFeatureId:         heritage.SecondFeatureID,
		CommunityId:             heritage.CommunityID,
	}
}

func daggerheartCompanionSheetToProto(companion *projectionstore.DaggerheartCompanionSheet, content contentstore.DaggerheartContentReadStore) *daggerheartv1.DaggerheartCompanionSheet {
	if companion == nil {
		return nil
	}
	experiences := make([]*daggerheartv1.DaggerheartCompanionExperience, 0, len(companion.Experiences))
	for _, experience := range companion.Experiences {
		name := experience.Name
		if content != nil && experience.ExperienceID != "" {
			entry, err := content.GetDaggerheartCompanionExperience(context.Background(), experience.ExperienceID)
			if err == nil {
				name = entry.Name
			}
		}
		experiences = append(experiences, &daggerheartv1.DaggerheartCompanionExperience{
			ExperienceId: experience.ExperienceID,
			Name:         name,
			Modifier:     int32(experience.Modifier),
		})
	}
	return &daggerheartv1.DaggerheartCompanionSheet{
		AnimalKind:        companion.AnimalKind,
		Name:              companion.Name,
		Evasion:           int32(companion.Evasion),
		Experiences:       experiences,
		AttackDescription: companion.AttackDescription,
		AttackRange:       companion.AttackRange,
		DamageDieSides:    int32(companion.DamageDieSides),
		DamageType:        companion.DamageType,
	}
}

// DaggerheartStateToProto converts a Daggerheart state projection into the game
// protobuf read model.
func DaggerheartStateToProto(campaignID, characterID string, state projectionstore.DaggerheartCharacterState) *campaignv1.CharacterState {
	return &campaignv1.CharacterState{
		CampaignId:  campaignID,
		CharacterId: characterID,
		SystemState: &campaignv1.CharacterState_Daggerheart{
			Daggerheart: &daggerheartv1.DaggerheartCharacterState{
				Hp:                            int32(state.Hp),
				Hope:                          int32(state.Hope),
				HopeMax:                       int32(state.HopeMax),
				Stress:                        int32(state.Stress),
				Armor:                         int32(state.Armor),
				ConditionStates:               DaggerheartProjectionConditionStatesToProto(state.Conditions),
				LifeState:                     DaggerheartLifeStateToProto(state.LifeState),
				ImpenetrableUsedThisShortRest: state.ImpenetrableUsedThisShortRest,
			},
		},
	}
}

// DaggerheartExperiencesToProto converts Daggerheart experiences to the system
// protobuf read model.
func DaggerheartExperiencesToProto(experiences []projectionstore.DaggerheartExperience) []*daggerheartv1.DaggerheartExperience {
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

func DaggerheartProjectionConditionStatesToProto(conditions []projectionstore.DaggerheartConditionState) []*daggerheartv1.DaggerheartConditionState {
	if len(conditions) == 0 {
		return nil
	}
	result := make([]*daggerheartv1.DaggerheartConditionState, 0, len(conditions))
	for _, condition := range conditions {
		entry := &daggerheartv1.DaggerheartConditionState{
			Id:       condition.ID,
			Code:     condition.Code,
			Label:    condition.Label,
			Source:   condition.Source,
			SourceId: condition.SourceID,
		}
		switch strings.TrimSpace(condition.Class) {
		case "standard":
			entry.Class = daggerheartv1.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_STANDARD
		case "tag":
			entry.Class = daggerheartv1.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_TAG
		case "special":
			entry.Class = daggerheartv1.DaggerheartConditionClass_DAGGERHEART_CONDITION_CLASS_SPECIAL
		}
		switch strings.TrimSpace(condition.Standard) {
		case daggerheart.ConditionHidden:
			entry.Standard = daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_HIDDEN
		case daggerheart.ConditionRestrained:
			entry.Standard = daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_RESTRAINED
		case daggerheart.ConditionVulnerable:
			entry.Standard = daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_VULNERABLE
		case daggerheart.ConditionCloaked:
			entry.Standard = daggerheartv1.DaggerheartCondition_DAGGERHEART_CONDITION_CLOAKED
		}
		for _, trigger := range condition.ClearTriggers {
			switch strings.TrimSpace(trigger) {
			case "short_rest":
				entry.ClearTriggers = append(entry.ClearTriggers, daggerheartv1.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SHORT_REST)
			case "long_rest":
				entry.ClearTriggers = append(entry.ClearTriggers, daggerheartv1.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_LONG_REST)
			case "session_end":
				entry.ClearTriggers = append(entry.ClearTriggers, daggerheartv1.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_SESSION_END)
			case "damage_taken":
				entry.ClearTriggers = append(entry.ClearTriggers, daggerheartv1.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_DAMAGE_TAKEN)
			}
		}
		result = append(result, entry)
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
