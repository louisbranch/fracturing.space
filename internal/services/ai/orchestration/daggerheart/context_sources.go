// Package daggerheart provides Daggerheart game-system context sources for
// campaign turn orchestration prompts.
package daggerheart

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

// ContextSources returns Daggerheart-specific context sources.
// These inject authoritative game-system rules and character state into the
// prompt so the GM agent does not need to discover them through tool calls.
func ContextSources() []orchestration.ContextSource {
	return []orchestration.ContextSource{
		orchestration.ContextSourceFunc(dualityRulesContextSource),
		orchestration.ContextSourceFunc(activeCharacterCapabilitiesContextSource),
		orchestration.ContextSourceFunc(combatBoardContextSource),
		orchestration.ContextSourceFunc(characterStateContextSource),
	}
}

// characterStateContextSource reads the campaign snapshot and returns a brief
// section with per-character HP/stress/armor/hope/conditions/life_state and
// campaign-level GM Fear. This gives the GM agent a tactical dashboard for
// informed narration decisions without tool calls.
func characterStateContextSource(ctx context.Context, sess orchestration.Session, input orchestration.PromptInput) (orchestration.BriefContribution, error) {
	uri := fmt.Sprintf("daggerheart://campaign/%s/snapshot", input.CampaignID)
	data, err := sess.ReadResource(ctx, uri)
	if err != nil {
		return orchestration.BriefContribution{}, fmt.Errorf("read daggerheart snapshot: %w", err)
	}
	return orchestration.SectionContribution(orchestration.BriefSection{
		ID:       "daggerheart_character_state",
		Priority: 250,
		Label:    "Daggerheart character state",
		Content:  data,
	}), nil
}

// activeCharacterCapabilitiesContextSource injects a compact digest of what
// the active-scene characters can currently do so the GM has always-on access
// to traits, equipment, Hope, domain cards, and active features.
func activeCharacterCapabilitiesContextSource(ctx context.Context, sess orchestration.Session, input orchestration.PromptInput) (orchestration.BriefContribution, error) {
	interactionRaw, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/interaction", input.CampaignID))
	if err != nil {
		return orchestration.BriefContribution{}, fmt.Errorf("read interaction state: %w", err)
	}
	sceneID, err := activeSceneIDFromInteraction(interactionRaw)
	if err != nil {
		return orchestration.BriefContribution{}, fmt.Errorf("decode interaction state: %w", err)
	}
	if strings.TrimSpace(sceneID) == "" {
		return orchestration.BriefContribution{}, nil
	}

	scenesRaw, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/sessions/%s/scenes", input.CampaignID, input.SessionID))
	if err != nil {
		return orchestration.BriefContribution{}, fmt.Errorf("read scenes: %w", err)
	}
	characterIDs, err := activeSceneCharacterIDs(scenesRaw, sceneID)
	if err != nil {
		return orchestration.BriefContribution{}, fmt.Errorf("decode scenes: %w", err)
	}
	if len(characterIDs) == 0 {
		return orchestration.BriefContribution{}, nil
	}

	payload := activeCharacterCapabilitiesPayload{
		ActiveSceneID: sceneID,
		Characters:    make([]activeCharacterCapabilityEntry, 0, len(characterIDs)),
	}
	for _, characterID := range characterIDs {
		raw, err := sess.ReadResource(ctx, fmt.Sprintf("campaign://%s/characters/%s/sheet", input.CampaignID, characterID))
		if err != nil {
			return orchestration.BriefContribution{}, fmt.Errorf("read character sheet %s: %w", characterID, err)
		}
		var sheet characterSheetSummary
		if err := json.Unmarshal([]byte(raw), &sheet); err != nil {
			return orchestration.BriefContribution{}, fmt.Errorf("decode character sheet %s: %w", characterID, err)
		}
		payload.Characters = append(payload.Characters, digestCharacterSheet(sheet))
	}
	if len(payload.Characters) == 0 {
		return orchestration.BriefContribution{}, nil
	}
	content, err := marshalJSON(payload)
	if err != nil {
		return orchestration.BriefContribution{}, fmt.Errorf("marshal character capabilities: %w", err)
	}
	return orchestration.SectionContribution(orchestration.BriefSection{
		ID:       "daggerheart_active_character_capabilities",
		Priority: 225,
		Label:    "Daggerheart active character capabilities",
		Content:  content,
	}), nil
}

// combatBoardContextSource injects the Daggerheart board state the GM needs for
// spotlight, countdown, and Fear-aware adjudication without an extra lookup.
func combatBoardContextSource(ctx context.Context, sess orchestration.Session, input orchestration.PromptInput) (orchestration.BriefContribution, error) {
	uri := fmt.Sprintf("daggerheart://campaign/%s/sessions/%s/combat_board", input.CampaignID, input.SessionID)
	data, err := sess.ReadResource(ctx, uri)
	if err != nil {
		return orchestration.BriefContribution{}, fmt.Errorf("read daggerheart combat board: %w", err)
	}
	return orchestration.SectionContribution(orchestration.BriefSection{
		ID:       "daggerheart_combat_board",
		Priority: 240,
		Label:    "Daggerheart combat board",
		Content:  data,
	}), nil
}

// dualityRulesContextSource reads the Daggerheart duality dice rules from the
// session resource and returns them as a brief section. This gives the GM agent
// authoritative dice mechanics and outcome definitions in every prompt.
func dualityRulesContextSource(ctx context.Context, sess orchestration.Session, _ orchestration.PromptInput) (orchestration.BriefContribution, error) {
	rules, err := sess.ReadResource(ctx, "daggerheart://rules/version")
	if err != nil {
		return orchestration.BriefContribution{}, fmt.Errorf("read daggerheart rules: %w", err)
	}
	return orchestration.SectionContribution(orchestration.BriefSection{
		ID:       "daggerheart_duality_rules",
		Priority: 200,
		Label:    "Daggerheart duality rules",
		Content:  rules,
	}), nil
}

type activeCharacterCapabilitiesPayload struct {
	ActiveSceneID string                           `json:"active_scene_id"`
	Characters    []activeCharacterCapabilityEntry `json:"characters"`
}

type activeCharacterCapabilityEntry struct {
	CharacterID            string                   `json:"character_id,omitempty"`
	Name                   string                   `json:"name,omitempty"`
	Class                  string                   `json:"class,omitempty"`
	Subclass               string                   `json:"subclass,omitempty"`
	Heritage               string                   `json:"heritage,omitempty"`
	Level                  int                      `json:"level,omitempty"`
	Traits                 *sheetTraitsSummary      `json:"traits,omitempty"`
	Resources              *sheetResourcesSummary   `json:"resources,omitempty"`
	Experiences            []sheetExperienceSummary `json:"experiences,omitempty"`
	Weapons                []sheetWeaponSummary     `json:"weapons,omitempty"`
	Armor                  *sheetArmorSummary       `json:"armor,omitempty"`
	DomainCards            []sheetDomainCardSummary `json:"domain_cards,omitempty"`
	ActiveClassFeatures    []string                 `json:"active_class_features,omitempty"`
	ActiveSubclassFeatures []string                 `json:"active_subclass_features,omitempty"`
	Conditions             []string                 `json:"conditions,omitempty"`
	Companion              *sheetCompanionSummary   `json:"companion,omitempty"`
	ActiveBeastform        *sheetBeastformSummary   `json:"active_beastform,omitempty"`
}

type characterSheetSummary struct {
	Character struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"character"`
	Daggerheart struct {
		Level int `json:"level"`
		Class struct {
			Name string `json:"name"`
		} `json:"class"`
		Subclass struct {
			Name string `json:"name"`
		} `json:"subclass"`
		Heritage struct {
			Ancestry  string `json:"ancestry"`
			Community string `json:"community"`
		} `json:"heritage"`
		Traits    *sheetTraitsSummary    `json:"traits"`
		Resources *sheetResourcesSummary `json:"resources"`
		Equipment struct {
			PrimaryWeapon   *sheetWeaponSummary `json:"primary_weapon"`
			SecondaryWeapon *sheetWeaponSummary `json:"secondary_weapon"`
			ActiveArmor     *sheetArmorSummary  `json:"active_armor"`
		} `json:"equipment"`
		Experiences         []sheetExperienceSummary `json:"experiences"`
		DomainCards         []sheetDomainCardSummary `json:"domain_cards"`
		ActiveClassFeatures []struct {
			Name string `json:"name"`
		} `json:"active_class_features"`
		ActiveSubclassFeatures []struct {
			Name string `json:"name"`
		} `json:"active_subclass_features"`
		Conditions []struct {
			Label string `json:"label"`
		} `json:"conditions"`
		Companion  *sheetCompanionSummary `json:"companion"`
		ClassState *struct {
			ActiveBeastform *sheetBeastformSummary `json:"active_beastform"`
		} `json:"class_state"`
	} `json:"daggerheart"`
}

type sheetTraitsSummary struct {
	Agility   int `json:"agility,omitempty"`
	Strength  int `json:"strength,omitempty"`
	Finesse   int `json:"finesse,omitempty"`
	Instinct  int `json:"instinct,omitempty"`
	Presence  int `json:"presence,omitempty"`
	Knowledge int `json:"knowledge,omitempty"`
}

type sheetResourcesSummary struct {
	HP        int    `json:"hp,omitempty"`
	HPMax     int    `json:"hp_max,omitempty"`
	Hope      int    `json:"hope,omitempty"`
	HopeMax   int    `json:"hope_max,omitempty"`
	Stress    int    `json:"stress,omitempty"`
	Armor     int    `json:"armor,omitempty"`
	LifeState string `json:"life_state,omitempty"`
}

type sheetWeaponSummary struct {
	Name       string `json:"name,omitempty"`
	Trait      string `json:"trait,omitempty"`
	Range      string `json:"range,omitempty"`
	DamageDice string `json:"damage_dice,omitempty"`
	DamageType string `json:"damage_type,omitempty"`
	Feature    string `json:"feature,omitempty"`
}

type sheetArmorSummary struct {
	Name      string `json:"name,omitempty"`
	BaseScore *int   `json:"base_score,omitempty"`
	Feature   string `json:"feature,omitempty"`
}

type sheetExperienceSummary struct {
	Name     string `json:"name,omitempty"`
	Modifier int    `json:"modifier,omitempty"`
}

type sheetDomainCardSummary struct {
	Name   string `json:"name,omitempty"`
	Domain string `json:"domain,omitempty"`
}

type sheetCompanionSummary struct {
	Name       string `json:"name,omitempty"`
	AnimalKind string `json:"animal_kind,omitempty"`
	Status     string `json:"status,omitempty"`
}

type sheetBeastformSummary struct {
	BaseTrait   string `json:"base_trait,omitempty"`
	AttackTrait string `json:"attack_trait,omitempty"`
	DamageType  string `json:"damage_type,omitempty"`
}

func activeSceneIDFromInteraction(raw string) (string, error) {
	var value struct {
		ActiveScene struct {
			SceneID string `json:"scene_id"`
		} `json:"active_scene"`
	}
	if err := json.Unmarshal([]byte(raw), &value); err != nil {
		return "", err
	}
	return strings.TrimSpace(value.ActiveScene.SceneID), nil
}

func activeSceneCharacterIDs(raw, activeSceneID string) ([]string, error) {
	var payload struct {
		Scenes []struct {
			SceneID      string   `json:"scene_id"`
			CharacterIDs []string `json:"character_ids"`
		} `json:"scenes"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return nil, err
	}
	for _, scene := range payload.Scenes {
		if strings.TrimSpace(scene.SceneID) != strings.TrimSpace(activeSceneID) {
			continue
		}
		result := make([]string, 0, len(scene.CharacterIDs))
		seen := make(map[string]struct{}, len(scene.CharacterIDs))
		for _, characterID := range scene.CharacterIDs {
			characterID = strings.TrimSpace(characterID)
			if characterID == "" {
				continue
			}
			if _, ok := seen[characterID]; ok {
				continue
			}
			seen[characterID] = struct{}{}
			result = append(result, characterID)
		}
		return result, nil
	}
	return nil, nil
}

func digestCharacterSheet(sheet characterSheetSummary) activeCharacterCapabilityEntry {
	entry := activeCharacterCapabilityEntry{
		CharacterID: sheet.Character.ID,
		Name:        sheet.Character.Name,
		Class:       strings.TrimSpace(sheet.Daggerheart.Class.Name),
		Subclass:    strings.TrimSpace(sheet.Daggerheart.Subclass.Name),
		Level:       sheet.Daggerheart.Level,
		Traits:      sheet.Daggerheart.Traits,
		Resources:   sheet.Daggerheart.Resources,
		Armor:       sheet.Daggerheart.Equipment.ActiveArmor,
		DomainCards: append([]sheetDomainCardSummary(nil), sheet.Daggerheart.DomainCards...),
		Companion:   sheet.Daggerheart.Companion,
	}
	entry.Heritage = heritageLabel(sheet.Daggerheart.Heritage.Ancestry, sheet.Daggerheart.Heritage.Community)
	for _, exp := range sheet.Daggerheart.Experiences {
		if name := strings.TrimSpace(exp.Name); name != "" {
			entry.Experiences = append(entry.Experiences, sheetExperienceSummary{
				Name:     name,
				Modifier: exp.Modifier,
			})
		}
	}
	if weapon := sheet.Daggerheart.Equipment.PrimaryWeapon; weapon != nil {
		entry.Weapons = append(entry.Weapons, *weapon)
	}
	if weapon := sheet.Daggerheart.Equipment.SecondaryWeapon; weapon != nil {
		entry.Weapons = append(entry.Weapons, *weapon)
	}
	for _, feature := range sheet.Daggerheart.ActiveClassFeatures {
		if name := strings.TrimSpace(feature.Name); name != "" {
			entry.ActiveClassFeatures = append(entry.ActiveClassFeatures, name)
		}
	}
	for _, feature := range sheet.Daggerheart.ActiveSubclassFeatures {
		if name := strings.TrimSpace(feature.Name); name != "" {
			entry.ActiveSubclassFeatures = append(entry.ActiveSubclassFeatures, name)
		}
	}
	for _, condition := range sheet.Daggerheart.Conditions {
		if label := strings.TrimSpace(condition.Label); label != "" {
			entry.Conditions = append(entry.Conditions, label)
		}
	}
	if sheet.Daggerheart.ClassState != nil {
		entry.ActiveBeastform = sheet.Daggerheart.ClassState.ActiveBeastform
	}
	return entry
}

func heritageLabel(ancestry, community string) string {
	ancestry = strings.TrimSpace(ancestry)
	community = strings.TrimSpace(community)
	switch {
	case ancestry != "" && community != "":
		return ancestry + " / " + community
	case ancestry != "":
		return ancestry
	default:
		return community
	}
}

func marshalJSON(value any) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
