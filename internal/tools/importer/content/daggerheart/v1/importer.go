package catalogimporter

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

const (
	contentTypeClass               = "class"
	contentTypeSubclass            = "subclass"
	contentTypeHeritage            = "heritage"
	contentTypeExperience          = "experience"
	contentTypeAdversary           = "adversary"
	contentTypeBeastform           = "beastform"
	contentTypeCompanionExperience = "companion_experience"
	contentTypeLootEntry           = "loot_entry"
	contentTypeDamageType          = "damage_type"
	contentTypeDomain              = "domain"
	contentTypeDomainCard          = "domain_card"
	contentTypeWeapon              = "weapon"
	contentTypeArmor               = "armor"
	contentTypeItem                = "item"
	contentTypeEnvironment         = "environment"
	contentTypeFeature             = "feature"
	contentTypeHopeFeature         = "hope_feature"
	contentTypeAdversaryFeature    = "adversary_feature"
	contentTypeBeastformFeature    = "beastform_feature"
)

func upsertClasses(ctx context.Context, store storage.DaggerheartContentStore, items []classRecord, locale string, isBase bool, now time.Time) error {
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("class id is required")
		}
		if isBase {
			class := storage.DaggerheartClass{
				ID:              item.ID,
				Name:            item.Name,
				StartingEvasion: item.StartingEvasion,
				StartingHP:      item.StartingHP,
				StartingItems:   append([]string{}, item.StartingItems...),
				Features:        toStorageFeatures(item.Features),
				HopeFeature:     toStorageHopeFeature(item.HopeFeature),
				DomainIDs:       append([]string{}, item.DomainIDs...),
				CreatedAt:       now,
				UpdatedAt:       now,
			}
			if err := store.PutDaggerheartClass(ctx, class); err != nil {
				return fmt.Errorf("put class %s: %w", item.ID, err)
			}
		}
		if err := upsertClassStrings(ctx, store, item, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertSubclasses(ctx context.Context, store storage.DaggerheartContentStore, items []subclassRecord, locale string, isBase bool, now time.Time) error {
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("subclass id is required")
		}
		if isBase && strings.TrimSpace(item.ClassID) == "" {
			return fmt.Errorf("subclass class_id is required")
		}
		if isBase {
			subclass := storage.DaggerheartSubclass{
				ID:                     item.ID,
				Name:                   item.Name,
				ClassID:                item.ClassID,
				SpellcastTrait:         item.SpellcastTrait,
				FoundationFeatures:     toStorageFeatures(item.FoundationFeatures),
				SpecializationFeatures: toStorageFeatures(item.SpecializationFeatures),
				MasteryFeatures:        toStorageFeatures(item.MasteryFeatures),
				CreatedAt:              now,
				UpdatedAt:              now,
			}
			if err := store.PutDaggerheartSubclass(ctx, subclass); err != nil {
				return fmt.Errorf("put subclass %s: %w", item.ID, err)
			}
		}
		if err := upsertSubclassStrings(ctx, store, item, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertHeritages(ctx context.Context, store storage.DaggerheartContentStore, items []heritageRecord, locale string, isBase bool, now time.Time) error {
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("heritage id is required")
		}
		if isBase {
			heritage := storage.DaggerheartHeritage{
				ID:        item.ID,
				Name:      item.Name,
				Kind:      item.Kind,
				Features:  toStorageFeatures(item.Features),
				CreatedAt: now,
				UpdatedAt: now,
			}
			if err := store.PutDaggerheartHeritage(ctx, heritage); err != nil {
				return fmt.Errorf("put heritage %s: %w", item.ID, err)
			}
		}
		if err := upsertHeritageStrings(ctx, store, item, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertExperiences(ctx context.Context, store storage.DaggerheartContentStore, items []experienceRecord, locale string, isBase bool, now time.Time) error {
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("experience id is required")
		}
		if isBase {
			experience := storage.DaggerheartExperienceEntry{
				ID:          item.ID,
				Name:        item.Name,
				Description: item.Description,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := store.PutDaggerheartExperience(ctx, experience); err != nil {
				return fmt.Errorf("put experience %s: %w", item.ID, err)
			}
		}
		if err := upsertExperienceStrings(ctx, store, item, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertAdversaries(ctx context.Context, store storage.DaggerheartContentStore, items []adversaryRecord, locale string, isBase bool, now time.Time) error {
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("adversary id is required")
		}
		if isBase {
			entry := storage.DaggerheartAdversaryEntry{
				ID:              item.ID,
				Name:            item.Name,
				Tier:            item.Tier,
				Role:            item.Role,
				Description:     item.Description,
				Motives:         item.Motives,
				Difficulty:      item.Difficulty,
				MajorThreshold:  item.MajorThreshold,
				SevereThreshold: item.SevereThreshold,
				HP:              item.HP,
				Stress:          item.Stress,
				Armor:           item.Armor,
				AttackModifier:  item.AttackModifier,
				StandardAttack:  toStorageAdversaryAttack(item.StandardAttack),
				Experiences:     toStorageAdversaryExperiences(item.Experiences),
				Features:        toStorageAdversaryFeatures(item.Features),
				CreatedAt:       now,
				UpdatedAt:       now,
			}
			if err := store.PutDaggerheartAdversaryEntry(ctx, entry); err != nil {
				return fmt.Errorf("put adversary %s: %w", item.ID, err)
			}
		}
		if err := upsertAdversaryStrings(ctx, store, item, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertBeastforms(ctx context.Context, store storage.DaggerheartContentStore, items []beastformRecord, locale string, isBase bool, now time.Time) error {
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("beastform id is required")
		}
		if isBase {
			entry := storage.DaggerheartBeastformEntry{
				ID:           item.ID,
				Name:         item.Name,
				Tier:         item.Tier,
				Examples:     item.Examples,
				Trait:        item.Trait,
				TraitBonus:   item.TraitBonus,
				EvasionBonus: item.EvasionBonus,
				Attack:       toStorageBeastformAttack(item.Attack),
				Advantages:   append([]string{}, item.Advantages...),
				Features:     toStorageBeastformFeatures(item.Features),
				CreatedAt:    now,
				UpdatedAt:    now,
			}
			if err := store.PutDaggerheartBeastform(ctx, entry); err != nil {
				return fmt.Errorf("put beastform %s: %w", item.ID, err)
			}
		}
		if err := upsertBeastformStrings(ctx, store, item, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertCompanionExperiences(ctx context.Context, store storage.DaggerheartContentStore, items []companionExperienceRecord, locale string, isBase bool, now time.Time) error {
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("companion experience id is required")
		}
		if isBase {
			experience := storage.DaggerheartCompanionExperienceEntry{
				ID:          item.ID,
				Name:        item.Name,
				Description: item.Description,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := store.PutDaggerheartCompanionExperience(ctx, experience); err != nil {
				return fmt.Errorf("put companion experience %s: %w", item.ID, err)
			}
		}
		if err := upsertCompanionExperienceStrings(ctx, store, item, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertLootEntries(ctx context.Context, store storage.DaggerheartContentStore, items []lootEntryRecord, locale string, isBase bool, now time.Time) error {
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("loot entry id is required")
		}
		if isBase {
			entry := storage.DaggerheartLootEntry{
				ID:          item.ID,
				Name:        item.Name,
				Roll:        item.Roll,
				Description: item.Description,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := store.PutDaggerheartLootEntry(ctx, entry); err != nil {
				return fmt.Errorf("put loot entry %s: %w", item.ID, err)
			}
		}
		if err := upsertLootEntryStrings(ctx, store, item, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertDamageTypes(ctx context.Context, store storage.DaggerheartContentStore, items []damageTypeRecord, locale string, isBase bool, now time.Time) error {
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("damage type id is required")
		}
		if isBase {
			entry := storage.DaggerheartDamageTypeEntry{
				ID:          item.ID,
				Name:        item.Name,
				Description: item.Description,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := store.PutDaggerheartDamageType(ctx, entry); err != nil {
				return fmt.Errorf("put damage type %s: %w", item.ID, err)
			}
		}
		if err := upsertDamageTypeStrings(ctx, store, item, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertDomains(ctx context.Context, store storage.DaggerheartContentStore, items []domainRecord, locale string, isBase bool, now time.Time) error {
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("domain id is required")
		}
		if isBase {
			domain := storage.DaggerheartDomain{
				ID:          item.ID,
				Name:        item.Name,
				Description: item.Description,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := store.PutDaggerheartDomain(ctx, domain); err != nil {
				return fmt.Errorf("put domain %s: %w", item.ID, err)
			}
		}
		if err := upsertDomainStrings(ctx, store, item, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertDomainCards(ctx context.Context, store storage.DaggerheartContentStore, items []domainCardRecord, locale string, isBase bool, now time.Time) error {
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("domain card id is required")
		}
		if isBase {
			card := storage.DaggerheartDomainCard{
				ID:          item.ID,
				Name:        item.Name,
				DomainID:    item.DomainID,
				Level:       item.Level,
				Type:        item.Type,
				RecallCost:  item.RecallCost,
				UsageLimit:  item.UsageLimit,
				FeatureText: item.FeatureText,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := store.PutDaggerheartDomainCard(ctx, card); err != nil {
				return fmt.Errorf("put domain card %s: %w", item.ID, err)
			}
		}
		if err := upsertDomainCardStrings(ctx, store, item, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertWeapons(ctx context.Context, store storage.DaggerheartContentStore, items []weaponRecord, locale string, isBase bool, now time.Time) error {
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("weapon id is required")
		}
		if isBase {
			weapon := storage.DaggerheartWeapon{
				ID:         item.ID,
				Name:       item.Name,
				Category:   item.Category,
				Tier:       item.Tier,
				Trait:      item.Trait,
				Range:      item.Range,
				DamageDice: toStorageDamageDice(item.DamageDice),
				DamageType: item.DamageType,
				Burden:     item.Burden,
				Feature:    item.Feature,
				CreatedAt:  now,
				UpdatedAt:  now,
			}
			if err := store.PutDaggerheartWeapon(ctx, weapon); err != nil {
				return fmt.Errorf("put weapon %s: %w", item.ID, err)
			}
		}
		if err := upsertWeaponStrings(ctx, store, item, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertArmor(ctx context.Context, store storage.DaggerheartContentStore, items []armorRecord, locale string, isBase bool, now time.Time) error {
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("armor id is required")
		}
		if isBase {
			armor := storage.DaggerheartArmor{
				ID:                  item.ID,
				Name:                item.Name,
				Tier:                item.Tier,
				BaseMajorThreshold:  item.BaseMajorThreshold,
				BaseSevereThreshold: item.BaseSevereThreshold,
				ArmorScore:          item.ArmorScore,
				Feature:             item.Feature,
				CreatedAt:           now,
				UpdatedAt:           now,
			}
			if err := store.PutDaggerheartArmor(ctx, armor); err != nil {
				return fmt.Errorf("put armor %s: %w", item.ID, err)
			}
		}
		if err := upsertArmorStrings(ctx, store, item, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertItems(ctx context.Context, store storage.DaggerheartContentStore, items []itemRecord, locale string, isBase bool, now time.Time) error {
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("item id is required")
		}
		if isBase {
			entry := storage.DaggerheartItem{
				ID:          item.ID,
				Name:        item.Name,
				Rarity:      item.Rarity,
				Kind:        item.Kind,
				StackMax:    item.StackMax,
				Description: item.Description,
				EffectText:  item.EffectText,
				CreatedAt:   now,
				UpdatedAt:   now,
			}
			if err := store.PutDaggerheartItem(ctx, entry); err != nil {
				return fmt.Errorf("put item %s: %w", item.ID, err)
			}
		}
		if err := upsertItemStrings(ctx, store, item, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertEnvironments(ctx context.Context, store storage.DaggerheartContentStore, items []environmentRecord, locale string, isBase bool, now time.Time) error {
	for _, item := range items {
		if strings.TrimSpace(item.ID) == "" {
			return fmt.Errorf("environment id is required")
		}
		if isBase {
			env := storage.DaggerheartEnvironment{
				ID:                    item.ID,
				Name:                  item.Name,
				Tier:                  item.Tier,
				Type:                  item.Type,
				Difficulty:            item.Difficulty,
				Impulses:              append([]string{}, item.Impulses...),
				PotentialAdversaryIDs: append([]string{}, item.PotentialAdversaryIDs...),
				Features:              toStorageFeatures(item.Features),
				Prompts:               append([]string{}, item.Prompts...),
				CreatedAt:             now,
				UpdatedAt:             now,
			}
			if err := store.PutDaggerheartEnvironment(ctx, env); err != nil {
				return fmt.Errorf("put environment %s: %w", item.ID, err)
			}
		}
		if err := upsertEnvironmentStrings(ctx, store, item, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func toStorageFeatures(features []featureRecord) []storage.DaggerheartFeature {
	items := make([]storage.DaggerheartFeature, 0, len(features))
	for _, feature := range features {
		items = append(items, storage.DaggerheartFeature{
			ID:          feature.ID,
			Name:        feature.Name,
			Description: feature.Description,
			Level:       feature.Level,
		})
	}
	return items
}

func toStorageHopeFeature(feature hopeFeatureRecord) storage.DaggerheartHopeFeature {
	return storage.DaggerheartHopeFeature{
		Name:        feature.Name,
		Description: feature.Description,
		HopeCost:    feature.HopeCost,
	}
}

func toStorageDamageDice(dice []damageDieRecord) []storage.DaggerheartDamageDie {
	items := make([]storage.DaggerheartDamageDie, 0, len(dice))
	for _, die := range dice {
		items = append(items, storage.DaggerheartDamageDie{
			Sides: die.Sides,
			Count: die.Count,
		})
	}
	return items
}

func toStorageAdversaryAttack(attack adversaryAttackRecord) storage.DaggerheartAdversaryAttack {
	return storage.DaggerheartAdversaryAttack{
		Name:        attack.Name,
		Range:       attack.Range,
		DamageDice:  toStorageDamageDice(attack.DamageDice),
		DamageBonus: attack.DamageBonus,
		DamageType:  attack.DamageType,
	}
}

func toStorageAdversaryExperiences(experiences []adversaryExperienceRecord) []storage.DaggerheartAdversaryExperience {
	items := make([]storage.DaggerheartAdversaryExperience, 0, len(experiences))
	for _, experience := range experiences {
		items = append(items, storage.DaggerheartAdversaryExperience{
			Name:     experience.Name,
			Modifier: experience.Modifier,
		})
	}
	return items
}

func toStorageAdversaryFeatures(features []adversaryFeatureRecord) []storage.DaggerheartAdversaryFeature {
	items := make([]storage.DaggerheartAdversaryFeature, 0, len(features))
	for _, feature := range features {
		items = append(items, storage.DaggerheartAdversaryFeature{
			ID:          feature.ID,
			Name:        feature.Name,
			Kind:        feature.Kind,
			Description: feature.Description,
			CostType:    feature.CostType,
			Cost:        feature.Cost,
		})
	}
	return items
}

func toStorageBeastformAttack(attack beastformAttackRecord) storage.DaggerheartBeastformAttack {
	return storage.DaggerheartBeastformAttack{
		Range:       attack.Range,
		Trait:       attack.Trait,
		DamageDice:  toStorageDamageDice(attack.DamageDice),
		DamageBonus: attack.DamageBonus,
		DamageType:  attack.DamageType,
	}
}

func toStorageBeastformFeatures(features []beastformFeatureRecord) []storage.DaggerheartBeastformFeature {
	items := make([]storage.DaggerheartBeastformFeature, 0, len(features))
	for _, feature := range features {
		items = append(items, storage.DaggerheartBeastformFeature{
			ID:          feature.ID,
			Name:        feature.Name,
			Description: feature.Description,
		})
	}
	return items
}

func upsertClassStrings(ctx context.Context, store storage.DaggerheartContentStore, item classRecord, locale string, now time.Time) error {
	if err := putContentString(ctx, store, item.ID, contentTypeClass, "name", locale, item.Name, now); err != nil {
		return err
	}
	for _, feature := range item.Features {
		if err := upsertFeatureStrings(ctx, store, feature, locale, now); err != nil {
			return err
		}
	}
	if strings.TrimSpace(item.HopeFeature.Name) != "" || strings.TrimSpace(item.HopeFeature.Description) != "" {
		hopeID := fmt.Sprintf("hope_feature:%s", item.ID)
		if err := putContentString(ctx, store, hopeID, contentTypeHopeFeature, "name", locale, item.HopeFeature.Name, now); err != nil {
			return err
		}
		if err := putContentString(ctx, store, hopeID, contentTypeHopeFeature, "description", locale, item.HopeFeature.Description, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertSubclassStrings(ctx context.Context, store storage.DaggerheartContentStore, item subclassRecord, locale string, now time.Time) error {
	if err := putContentString(ctx, store, item.ID, contentTypeSubclass, "name", locale, item.Name, now); err != nil {
		return err
	}
	for _, feature := range item.FoundationFeatures {
		if err := upsertFeatureStrings(ctx, store, feature, locale, now); err != nil {
			return err
		}
	}
	for _, feature := range item.SpecializationFeatures {
		if err := upsertFeatureStrings(ctx, store, feature, locale, now); err != nil {
			return err
		}
	}
	for _, feature := range item.MasteryFeatures {
		if err := upsertFeatureStrings(ctx, store, feature, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertHeritageStrings(ctx context.Context, store storage.DaggerheartContentStore, item heritageRecord, locale string, now time.Time) error {
	if err := putContentString(ctx, store, item.ID, contentTypeHeritage, "name", locale, item.Name, now); err != nil {
		return err
	}
	for _, feature := range item.Features {
		if err := upsertFeatureStrings(ctx, store, feature, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertExperienceStrings(ctx context.Context, store storage.DaggerheartContentStore, item experienceRecord, locale string, now time.Time) error {
	if err := putContentString(ctx, store, item.ID, contentTypeExperience, "name", locale, item.Name, now); err != nil {
		return err
	}
	if err := putContentString(ctx, store, item.ID, contentTypeExperience, "description", locale, item.Description, now); err != nil {
		return err
	}
	return nil
}

func upsertAdversaryStrings(ctx context.Context, store storage.DaggerheartContentStore, item adversaryRecord, locale string, now time.Time) error {
	if err := putContentString(ctx, store, item.ID, contentTypeAdversary, "name", locale, item.Name, now); err != nil {
		return err
	}
	if err := putContentString(ctx, store, item.ID, contentTypeAdversary, "description", locale, item.Description, now); err != nil {
		return err
	}
	if err := putContentString(ctx, store, item.ID, contentTypeAdversary, "motives", locale, item.Motives, now); err != nil {
		return err
	}
	if err := putContentString(ctx, store, item.ID, contentTypeAdversary, "attack_name", locale, item.StandardAttack.Name, now); err != nil {
		return err
	}
	if err := putContentString(ctx, store, item.ID, contentTypeAdversary, "attack_range", locale, item.StandardAttack.Range, now); err != nil {
		return err
	}
	for _, feature := range item.Features {
		if err := upsertAdversaryFeatureStrings(ctx, store, feature, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertBeastformStrings(ctx context.Context, store storage.DaggerheartContentStore, item beastformRecord, locale string, now time.Time) error {
	if err := putContentString(ctx, store, item.ID, contentTypeBeastform, "name", locale, item.Name, now); err != nil {
		return err
	}
	if err := putContentString(ctx, store, item.ID, contentTypeBeastform, "examples", locale, item.Examples, now); err != nil {
		return err
	}
	for idx, advantage := range item.Advantages {
		field := fmt.Sprintf("advantage.%d", idx)
		if err := putContentString(ctx, store, item.ID, contentTypeBeastform, field, locale, advantage, now); err != nil {
			return err
		}
	}
	for _, feature := range item.Features {
		if err := upsertBeastformFeatureStrings(ctx, store, feature, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertCompanionExperienceStrings(ctx context.Context, store storage.DaggerheartContentStore, item companionExperienceRecord, locale string, now time.Time) error {
	if err := putContentString(ctx, store, item.ID, contentTypeCompanionExperience, "name", locale, item.Name, now); err != nil {
		return err
	}
	if err := putContentString(ctx, store, item.ID, contentTypeCompanionExperience, "description", locale, item.Description, now); err != nil {
		return err
	}
	return nil
}

func upsertLootEntryStrings(ctx context.Context, store storage.DaggerheartContentStore, item lootEntryRecord, locale string, now time.Time) error {
	if err := putContentString(ctx, store, item.ID, contentTypeLootEntry, "name", locale, item.Name, now); err != nil {
		return err
	}
	if err := putContentString(ctx, store, item.ID, contentTypeLootEntry, "description", locale, item.Description, now); err != nil {
		return err
	}
	return nil
}

func upsertDamageTypeStrings(ctx context.Context, store storage.DaggerheartContentStore, item damageTypeRecord, locale string, now time.Time) error {
	if err := putContentString(ctx, store, item.ID, contentTypeDamageType, "name", locale, item.Name, now); err != nil {
		return err
	}
	if err := putContentString(ctx, store, item.ID, contentTypeDamageType, "description", locale, item.Description, now); err != nil {
		return err
	}
	return nil
}

func upsertDomainStrings(ctx context.Context, store storage.DaggerheartContentStore, item domainRecord, locale string, now time.Time) error {
	if err := putContentString(ctx, store, item.ID, contentTypeDomain, "name", locale, item.Name, now); err != nil {
		return err
	}
	if err := putContentString(ctx, store, item.ID, contentTypeDomain, "description", locale, item.Description, now); err != nil {
		return err
	}
	return nil
}

func upsertDomainCardStrings(ctx context.Context, store storage.DaggerheartContentStore, item domainCardRecord, locale string, now time.Time) error {
	if err := putContentString(ctx, store, item.ID, contentTypeDomainCard, "name", locale, item.Name, now); err != nil {
		return err
	}
	if err := putContentString(ctx, store, item.ID, contentTypeDomainCard, "usage_limit", locale, item.UsageLimit, now); err != nil {
		return err
	}
	if err := putContentString(ctx, store, item.ID, contentTypeDomainCard, "feature_text", locale, item.FeatureText, now); err != nil {
		return err
	}
	return nil
}

func upsertWeaponStrings(ctx context.Context, store storage.DaggerheartContentStore, item weaponRecord, locale string, now time.Time) error {
	if err := putContentString(ctx, store, item.ID, contentTypeWeapon, "name", locale, item.Name, now); err != nil {
		return err
	}
	if err := putContentString(ctx, store, item.ID, contentTypeWeapon, "feature", locale, item.Feature, now); err != nil {
		return err
	}
	return nil
}

func upsertArmorStrings(ctx context.Context, store storage.DaggerheartContentStore, item armorRecord, locale string, now time.Time) error {
	if err := putContentString(ctx, store, item.ID, contentTypeArmor, "name", locale, item.Name, now); err != nil {
		return err
	}
	if err := putContentString(ctx, store, item.ID, contentTypeArmor, "feature", locale, item.Feature, now); err != nil {
		return err
	}
	return nil
}

func upsertItemStrings(ctx context.Context, store storage.DaggerheartContentStore, item itemRecord, locale string, now time.Time) error {
	if err := putContentString(ctx, store, item.ID, contentTypeItem, "name", locale, item.Name, now); err != nil {
		return err
	}
	if err := putContentString(ctx, store, item.ID, contentTypeItem, "description", locale, item.Description, now); err != nil {
		return err
	}
	if err := putContentString(ctx, store, item.ID, contentTypeItem, "effect_text", locale, item.EffectText, now); err != nil {
		return err
	}
	return nil
}

func upsertEnvironmentStrings(ctx context.Context, store storage.DaggerheartContentStore, item environmentRecord, locale string, now time.Time) error {
	if err := putContentString(ctx, store, item.ID, contentTypeEnvironment, "name", locale, item.Name, now); err != nil {
		return err
	}
	for idx, impulse := range item.Impulses {
		field := fmt.Sprintf("impulse.%d", idx)
		if err := putContentString(ctx, store, item.ID, contentTypeEnvironment, field, locale, impulse, now); err != nil {
			return err
		}
	}
	for idx, prompt := range item.Prompts {
		field := fmt.Sprintf("prompt.%d", idx)
		if err := putContentString(ctx, store, item.ID, contentTypeEnvironment, field, locale, prompt, now); err != nil {
			return err
		}
	}
	for _, feature := range item.Features {
		if err := upsertFeatureStrings(ctx, store, feature, locale, now); err != nil {
			return err
		}
	}
	return nil
}

func upsertFeatureStrings(ctx context.Context, store storage.DaggerheartContentStore, feature featureRecord, locale string, now time.Time) error {
	if strings.TrimSpace(feature.ID) == "" {
		return fmt.Errorf("feature id is required")
	}
	if err := putContentString(ctx, store, feature.ID, contentTypeFeature, "name", locale, feature.Name, now); err != nil {
		return err
	}
	if err := putContentString(ctx, store, feature.ID, contentTypeFeature, "description", locale, feature.Description, now); err != nil {
		return err
	}
	return nil
}

func upsertAdversaryFeatureStrings(ctx context.Context, store storage.DaggerheartContentStore, feature adversaryFeatureRecord, locale string, now time.Time) error {
	if strings.TrimSpace(feature.ID) == "" {
		return fmt.Errorf("adversary feature id is required")
	}
	if err := putContentString(ctx, store, feature.ID, contentTypeAdversaryFeature, "name", locale, feature.Name, now); err != nil {
		return err
	}
	if err := putContentString(ctx, store, feature.ID, contentTypeAdversaryFeature, "description", locale, feature.Description, now); err != nil {
		return err
	}
	return nil
}

func upsertBeastformFeatureStrings(ctx context.Context, store storage.DaggerheartContentStore, feature beastformFeatureRecord, locale string, now time.Time) error {
	if strings.TrimSpace(feature.ID) == "" {
		return fmt.Errorf("beastform feature id is required")
	}
	if err := putContentString(ctx, store, feature.ID, contentTypeBeastformFeature, "name", locale, feature.Name, now); err != nil {
		return err
	}
	if err := putContentString(ctx, store, feature.ID, contentTypeBeastformFeature, "description", locale, feature.Description, now); err != nil {
		return err
	}
	return nil
}

func putContentString(ctx context.Context, store storage.DaggerheartContentStore, contentID, contentType, field, locale, text string, now time.Time) error {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	entry := storage.DaggerheartContentString{
		ContentID:   contentID,
		ContentType: contentType,
		Field:       field,
		Locale:      locale,
		Text:        text,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if err := store.PutDaggerheartContentString(ctx, entry); err != nil {
		return fmt.Errorf("put content string %s.%s: %w", contentID, field, err)
	}
	return nil
}
