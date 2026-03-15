package daggerheartcontent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

func unmarshalOptionalJSON[T any](raw string, dest *T, label string) error {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(raw), dest); err != nil {
		return fmt.Errorf("decode %s: %w", label, err)
	}
	return nil
}

func dbDaggerheartClassToStorage(row db.DaggerheartClass) (contentstore.DaggerheartClass, error) {
	class := contentstore.DaggerheartClass{
		ID:              row.ID,
		Name:            row.Name,
		StartingEvasion: int(row.StartingEvasion),
		StartingHP:      int(row.StartingHp),
		CreatedAt:       sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt:       sqliteutil.FromMillis(row.UpdatedAt),
	}
	if err := unmarshalOptionalJSON(row.StartingItemsJson, &class.StartingItems, "daggerheart class starting items"); err != nil {
		return contentstore.DaggerheartClass{}, err
	}
	if err := unmarshalOptionalJSON(row.FeaturesJson, &class.Features, "daggerheart class features"); err != nil {
		return contentstore.DaggerheartClass{}, err
	}
	if err := unmarshalOptionalJSON(row.HopeFeatureJson, &class.HopeFeature, "daggerheart class hope feature"); err != nil {
		return contentstore.DaggerheartClass{}, err
	}
	if err := unmarshalOptionalJSON(row.DomainIdsJson, &class.DomainIDs, "daggerheart class domain ids"); err != nil {
		return contentstore.DaggerheartClass{}, err
	}
	return class, nil
}

func dbDaggerheartSubclassToStorage(row db.DaggerheartSubclass) (contentstore.DaggerheartSubclass, error) {
	subclass := contentstore.DaggerheartSubclass{
		ID:             row.ID,
		Name:           row.Name,
		ClassID:        row.ClassID,
		SpellcastTrait: row.SpellcastTrait,
		CreatedAt:      sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt:      sqliteutil.FromMillis(row.UpdatedAt),
	}
	if err := unmarshalOptionalJSON(row.FoundationFeaturesJson, &subclass.FoundationFeatures, "daggerheart subclass foundation features"); err != nil {
		return contentstore.DaggerheartSubclass{}, err
	}
	if err := unmarshalOptionalJSON(row.SpecializationFeaturesJson, &subclass.SpecializationFeatures, "daggerheart subclass specialization features"); err != nil {
		return contentstore.DaggerheartSubclass{}, err
	}
	if err := unmarshalOptionalJSON(row.MasteryFeaturesJson, &subclass.MasteryFeatures, "daggerheart subclass mastery features"); err != nil {
		return contentstore.DaggerheartSubclass{}, err
	}
	return subclass, nil
}

func dbDaggerheartHeritageToStorage(row db.DaggerheartHeritage) (contentstore.DaggerheartHeritage, error) {
	heritage := contentstore.DaggerheartHeritage{
		ID:        row.ID,
		Name:      row.Name,
		Kind:      row.Kind,
		CreatedAt: sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt: sqliteutil.FromMillis(row.UpdatedAt),
	}
	if err := unmarshalOptionalJSON(row.FeaturesJson, &heritage.Features, "daggerheart heritage features"); err != nil {
		return contentstore.DaggerheartHeritage{}, err
	}
	return heritage, nil
}

func dbDaggerheartExperienceToStorage(row db.DaggerheartExperience) contentstore.DaggerheartExperienceEntry {
	return contentstore.DaggerheartExperienceEntry{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		CreatedAt:   sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt:   sqliteutil.FromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartAdversaryEntryToStorage(row db.DaggerheartAdversaryEntry) (contentstore.DaggerheartAdversaryEntry, error) {
	entry := contentstore.DaggerheartAdversaryEntry{
		ID:              row.ID,
		Name:            row.Name,
		Tier:            int(row.Tier),
		Role:            row.Role,
		Description:     row.Description,
		Motives:         row.Motives,
		Difficulty:      int(row.Difficulty),
		MajorThreshold:  int(row.MajorThreshold),
		SevereThreshold: int(row.SevereThreshold),
		HP:              int(row.Hp),
		Stress:          int(row.Stress),
		Armor:           int(row.Armor),
		AttackModifier:  int(row.AttackModifier),
		CreatedAt:       sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt:       sqliteutil.FromMillis(row.UpdatedAt),
	}
	if err := unmarshalOptionalJSON(row.StandardAttackJson, &entry.StandardAttack, "daggerheart adversary standard attack"); err != nil {
		return contentstore.DaggerheartAdversaryEntry{}, err
	}
	if err := unmarshalOptionalJSON(row.ExperiencesJson, &entry.Experiences, "daggerheart adversary experiences"); err != nil {
		return contentstore.DaggerheartAdversaryEntry{}, err
	}
	if err := unmarshalOptionalJSON(row.FeaturesJson, &entry.Features, "daggerheart adversary features"); err != nil {
		return contentstore.DaggerheartAdversaryEntry{}, err
	}
	return entry, nil
}

func dbDaggerheartBeastformToStorage(row db.DaggerheartBeastform) (contentstore.DaggerheartBeastformEntry, error) {
	entry := contentstore.DaggerheartBeastformEntry{
		ID:           row.ID,
		Name:         row.Name,
		Tier:         int(row.Tier),
		Examples:     row.Examples,
		Trait:        row.Trait,
		TraitBonus:   int(row.TraitBonus),
		EvasionBonus: int(row.EvasionBonus),
		CreatedAt:    sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt:    sqliteutil.FromMillis(row.UpdatedAt),
	}
	if err := unmarshalOptionalJSON(row.AttackJson, &entry.Attack, "daggerheart beastform attack"); err != nil {
		return contentstore.DaggerheartBeastformEntry{}, err
	}
	if err := unmarshalOptionalJSON(row.AdvantagesJson, &entry.Advantages, "daggerheart beastform advantages"); err != nil {
		return contentstore.DaggerheartBeastformEntry{}, err
	}
	if err := unmarshalOptionalJSON(row.FeaturesJson, &entry.Features, "daggerheart beastform features"); err != nil {
		return contentstore.DaggerheartBeastformEntry{}, err
	}
	return entry, nil
}

func dbDaggerheartCompanionExperienceToStorage(row db.DaggerheartCompanionExperience) contentstore.DaggerheartCompanionExperienceEntry {
	return contentstore.DaggerheartCompanionExperienceEntry{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		CreatedAt:   sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt:   sqliteutil.FromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartLootEntryToStorage(row db.DaggerheartLootEntry) contentstore.DaggerheartLootEntry {
	return contentstore.DaggerheartLootEntry{
		ID:          row.ID,
		Name:        row.Name,
		Roll:        int(row.Roll),
		Description: row.Description,
		CreatedAt:   sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt:   sqliteutil.FromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartDamageTypeToStorage(row db.DaggerheartDamageType) contentstore.DaggerheartDamageTypeEntry {
	return contentstore.DaggerheartDamageTypeEntry{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		CreatedAt:   sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt:   sqliteutil.FromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartDomainToStorage(row db.DaggerheartDomain) contentstore.DaggerheartDomain {
	return contentstore.DaggerheartDomain{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		CreatedAt:   sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt:   sqliteutil.FromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartDomainCardToStorage(row db.DaggerheartDomainCard) contentstore.DaggerheartDomainCard {
	return contentstore.DaggerheartDomainCard{
		ID:          row.ID,
		Name:        row.Name,
		DomainID:    row.DomainID,
		Level:       int(row.Level),
		Type:        row.Type,
		RecallCost:  int(row.RecallCost),
		UsageLimit:  row.UsageLimit,
		FeatureText: row.FeatureText,
		CreatedAt:   sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt:   sqliteutil.FromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartWeaponToStorage(row db.DaggerheartWeapon) (contentstore.DaggerheartWeapon, error) {
	weapon := contentstore.DaggerheartWeapon{
		ID:         row.ID,
		Name:       row.Name,
		Category:   row.Category,
		Tier:       int(row.Tier),
		Trait:      row.Trait,
		Range:      row.Range,
		DamageType: row.DamageType,
		Burden:     int(row.Burden),
		Feature:    row.Feature,
		CreatedAt:  sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt:  sqliteutil.FromMillis(row.UpdatedAt),
	}
	if err := unmarshalOptionalJSON(row.DamageDiceJson, &weapon.DamageDice, "daggerheart weapon damage dice"); err != nil {
		return contentstore.DaggerheartWeapon{}, err
	}
	return weapon, nil
}

func dbDaggerheartArmorToStorage(row db.DaggerheartArmor) contentstore.DaggerheartArmor {
	return contentstore.DaggerheartArmor{
		ID:                  row.ID,
		Name:                row.Name,
		Tier:                int(row.Tier),
		BaseMajorThreshold:  int(row.BaseMajorThreshold),
		BaseSevereThreshold: int(row.BaseSevereThreshold),
		ArmorScore:          int(row.ArmorScore),
		Feature:             row.Feature,
		CreatedAt:           sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt:           sqliteutil.FromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartItemToStorage(row db.DaggerheartItem) contentstore.DaggerheartItem {
	return contentstore.DaggerheartItem{
		ID:          row.ID,
		Name:        row.Name,
		Rarity:      row.Rarity,
		Kind:        row.Kind,
		StackMax:    int(row.StackMax),
		Description: row.Description,
		EffectText:  row.EffectText,
		CreatedAt:   sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt:   sqliteutil.FromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartEnvironmentToStorage(row db.DaggerheartEnvironment) (contentstore.DaggerheartEnvironment, error) {
	env := contentstore.DaggerheartEnvironment{
		ID:         row.ID,
		Name:       row.Name,
		Tier:       int(row.Tier),
		Type:       row.Type,
		Difficulty: int(row.Difficulty),
		CreatedAt:  sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt:  sqliteutil.FromMillis(row.UpdatedAt),
	}
	if err := unmarshalOptionalJSON(row.ImpulsesJson, &env.Impulses, "daggerheart environment impulses"); err != nil {
		return contentstore.DaggerheartEnvironment{}, err
	}
	if err := unmarshalOptionalJSON(row.PotentialAdversaryIdsJson, &env.PotentialAdversaryIDs, "daggerheart environment adversaries"); err != nil {
		return contentstore.DaggerheartEnvironment{}, err
	}
	if err := unmarshalOptionalJSON(row.FeaturesJson, &env.Features, "daggerheart environment features"); err != nil {
		return contentstore.DaggerheartEnvironment{}, err
	}
	if err := unmarshalOptionalJSON(row.PromptsJson, &env.Prompts, "daggerheart environment prompts"); err != nil {
		return contentstore.DaggerheartEnvironment{}, err
	}
	return env, nil
}

func dbDaggerheartContentStringToStorage(row db.DaggerheartContentString) contentstore.DaggerheartContentString {
	return contentstore.DaggerheartContentString{
		ContentID:   row.ContentID,
		ContentType: row.ContentType,
		Field:       row.Field,
		Locale:      row.Locale,
		Text:        row.Text,
		CreatedAt:   sqliteutil.FromMillis(row.CreatedAt),
		UpdatedAt:   sqliteutil.FromMillis(row.UpdatedAt),
	}
}
