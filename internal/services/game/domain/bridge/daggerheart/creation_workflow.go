package daggerheart

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const (
	// CreationStepClassSubclass selects class and subclass.
	CreationStepClassSubclass int32 = 1
	// CreationStepHeritage selects ancestry and community.
	CreationStepHeritage int32 = 2
	// CreationStepTraits assigns the six trait modifiers.
	CreationStepTraits int32 = 3
	// CreationStepDetails records additional starting character details.
	CreationStepDetails int32 = 4
	// CreationStepEquipment records starting equipment choices.
	CreationStepEquipment int32 = 5
	// CreationStepBackground records free-form background.
	CreationStepBackground int32 = 6
	// CreationStepExperiences records starting experiences.
	CreationStepExperiences int32 = 7
	// CreationStepDomainCards records starting domain cards.
	CreationStepDomainCards int32 = 8
	// CreationStepConnections records party connections.
	CreationStepConnections int32 = 9

	// StartingPotionMinorHealthID is the valid minor health potion item id.
	StartingPotionMinorHealthID = "item.minor-health-potion"
	// StartingPotionMinorStaminaID is the valid minor stamina potion item id.
	StartingPotionMinorStaminaID = "item.minor-stamina-potion"
)

var creationStepKeys = []string{
	"class_subclass",
	"heritage",
	"traits",
	"details",
	"equipment",
	"background",
	"experiences",
	"domain_cards",
	"connections",
}

// CreationProfile captures Daggerheart-specific character-creation choices.
type CreationProfile struct {
	ClassID              string
	SubclassID           string
	AncestryID           string
	CommunityID          string
	TraitsAssigned       bool
	Traits               Traits
	DetailsRecorded      bool
	Level                int
	HpMax                int
	StressMax            int
	Evasion              int
	StartingWeaponIDs    []string
	StartingArmorID      string
	StartingPotionItemID string
	Background           string
	Experiences          []Experience
	DomainCardIDs        []string
	Connections          string
}

// CreationStepProgress represents completion state for one creation step.
type CreationStepProgress struct {
	Step     int32
	Key      string
	Complete bool
}

// CreationProgress describes end-to-end creation workflow readiness.
type CreationProgress struct {
	Steps        []CreationStepProgress
	NextStep     int32
	Ready        bool
	UnmetReasons []string
}

// EvaluateCreationProgress evaluates Daggerheart's SRD-aligned 9-step creation
// workflow.
//
// Steps complete strictly in order. A later step is not considered complete
// until all earlier steps are complete, even if its own fields are present.
func EvaluateCreationProgress(profile CreationProfile) CreationProgress {
	rawChecks := []bool{
		hasClassAndSubclass(profile),
		hasHeritage(profile),
		hasTraitAssignment(profile),
		hasRecordedDetails(profile),
		hasStartingEquipment(profile),
		strings.TrimSpace(profile.Background) != "",
		hasExperiences(profile.Experiences),
		hasDomainCardIDs(profile.DomainCardIDs),
		strings.TrimSpace(profile.Connections) != "",
	}
	reasons := []string{
		"class and subclass selection is required",
		"ancestry and community selection are required",
		"trait assignment must match +2,+1,+1,+0,+0,-1",
		"additional character details must be recorded",
		"starting equipment selection is required",
		"background is required",
		"at least one experience is required",
		"at least one domain card is required",
		"connections are required",
	}

	steps := make([]CreationStepProgress, 0, len(creationStepKeys))
	allowComplete := true
	for i := range creationStepKeys {
		complete := rawChecks[i] && allowComplete
		steps = append(steps, CreationStepProgress{
			Step:     int32(i + 1),
			Key:      creationStepKeys[i],
			Complete: complete,
		})
		if !complete {
			allowComplete = false
		}
	}

	nextStep := int32(0)
	for _, step := range steps {
		if !step.Complete {
			nextStep = step.Step
			break
		}
	}

	unmet := make([]string, 0, len(reasons))
	for i, ok := range rawChecks {
		if !ok {
			unmet = append(unmet, reasons[i])
		}
	}

	return CreationProgress{
		Steps:        steps,
		NextStep:     nextStep,
		Ready:        nextStep == 0,
		UnmetReasons: unmet,
	}
}

// ValidateCreationTraitDistribution validates the SRD-required starting
// distribution (+2,+1,+1,+0,+0,-1).
func ValidateCreationTraitDistribution(traits Traits) error {
	values := []int{
		traits.Agility,
		traits.Strength,
		traits.Finesse,
		traits.Instinct,
		traits.Presence,
		traits.Knowledge,
	}
	for _, value := range values {
		if value < TraitMin || value > TraitMax {
			return fmt.Errorf("traits must be assigned as +2,+1,+1,+0,+0,-1")
		}
	}
	sort.Ints(values)
	expected := []int{-1, 0, 0, 1, 1, 2}
	for i := range values {
		if values[i] != expected[i] {
			return fmt.Errorf("traits must be assigned as +2,+1,+1,+0,+0,-1")
		}
	}
	return nil
}

// IsValidStartingPotionItemID reports whether the item id is one of the SRD
// starting potion choices.
func IsValidStartingPotionItemID(potionItemID string) bool {
	switch strings.TrimSpace(potionItemID) {
	case StartingPotionMinorHealthID, StartingPotionMinorStaminaID:
		return true
	default:
		return false
	}
}

// EvaluateCreationReadinessFromSystemProfile evaluates readiness directly from
// a core aggregate system_profile map.
func EvaluateCreationReadinessFromSystemProfile(systemProfile map[string]any) (bool, string) {
	rawProfile, ok := systemProfile[SystemID]
	if !ok || rawProfile == nil {
		return false, "daggerheart profile is missing"
	}

	payloadJSON, err := json.Marshal(rawProfile)
	if err != nil {
		return false, "daggerheart profile is invalid"
	}

	var payload profilePayload
	if err := json.Unmarshal(payloadJSON, &payload); err != nil {
		return false, "daggerheart profile is invalid"
	}
	if payload.Reset {
		return false, "character creation workflow is reset"
	}

	progress := EvaluateCreationProgress(creationProfileFromPayload(payload))
	if progress.Ready {
		return true, ""
	}
	if len(progress.UnmetReasons) == 0 {
		return false, "character creation workflow is incomplete"
	}
	return false, progress.UnmetReasons[0]
}

func creationProfileFromPayload(payload profilePayload) CreationProfile {
	experiences := make([]Experience, 0, len(payload.Experiences))
	for _, exp := range payload.Experiences {
		experiences = append(experiences, Experience{
			Name:     exp.Name,
			Modifier: exp.Modifier,
		})
	}
	return CreationProfile{
		ClassID:        payload.ClassID,
		SubclassID:     payload.SubclassID,
		AncestryID:     payload.AncestryID,
		CommunityID:    payload.CommunityID,
		TraitsAssigned: payload.TraitsAssigned,
		Traits: Traits{
			Agility:   payload.Agility,
			Strength:  payload.Strength,
			Finesse:   payload.Finesse,
			Instinct:  payload.Instinct,
			Presence:  payload.Presence,
			Knowledge: payload.Knowledge,
		},
		DetailsRecorded:      payload.DetailsRecorded,
		Level:                payload.Level,
		HpMax:                payload.HpMax,
		StressMax:            payload.StressMax,
		Evasion:              payload.Evasion,
		StartingWeaponIDs:    append([]string(nil), payload.StartingWeaponIDs...),
		StartingArmorID:      payload.StartingArmorID,
		StartingPotionItemID: payload.StartingPotionItemID,
		Background:           payload.Background,
		Experiences:          experiences,
		DomainCardIDs:        append([]string(nil), payload.DomainCardIDs...),
		Connections:          payload.Connections,
	}
}

func hasClassAndSubclass(profile CreationProfile) bool {
	return strings.TrimSpace(profile.ClassID) != "" && strings.TrimSpace(profile.SubclassID) != ""
}

func hasHeritage(profile CreationProfile) bool {
	return strings.TrimSpace(profile.AncestryID) != "" && strings.TrimSpace(profile.CommunityID) != ""
}

func hasTraitAssignment(profile CreationProfile) bool {
	if !profile.TraitsAssigned {
		return false
	}
	return ValidateCreationTraitDistribution(profile.Traits) == nil
}

func hasRecordedDetails(profile CreationProfile) bool {
	if !profile.DetailsRecorded {
		return false
	}
	return profile.Level > 0 && profile.HpMax > 0 && profile.StressMax > 0 && profile.Evasion > 0
}

func hasStartingEquipment(profile CreationProfile) bool {
	if len(profile.StartingWeaponIDs) == 0 {
		return false
	}
	for _, weaponID := range profile.StartingWeaponIDs {
		if strings.TrimSpace(weaponID) == "" {
			return false
		}
	}
	if strings.TrimSpace(profile.StartingArmorID) == "" {
		return false
	}
	return IsValidStartingPotionItemID(profile.StartingPotionItemID)
}

func hasExperiences(experiences []Experience) bool {
	if len(experiences) == 0 {
		return false
	}
	for _, experience := range experiences {
		if strings.TrimSpace(experience.Name) == "" {
			return false
		}
	}
	return true
}

func hasDomainCardIDs(domainCardIDs []string) bool {
	if len(domainCardIDs) == 0 {
		return false
	}
	for _, domainCardID := range domainCardIDs {
		if strings.TrimSpace(domainCardID) == "" {
			return false
		}
	}
	return true
}
