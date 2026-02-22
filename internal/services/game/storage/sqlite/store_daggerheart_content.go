package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

// Daggerheart content catalog methods

func (s *Store) validateContentStore(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	return nil
}

func requireCatalogEntryID(id string, label string) error {
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("%s id is required", label)
	}
	return nil
}

// PutDaggerheartClass persists a Daggerheart class catalog entry.
func (s *Store) PutDaggerheartClass(ctx context.Context, class storage.DaggerheartClass) error {
	if err := s.validateContentStore(ctx); err != nil {
		return err
	}
	if err := requireCatalogEntryID(class.ID, "class"); err != nil {
		return err
	}

	startingItemsJSON, err := json.Marshal(class.StartingItems)
	if err != nil {
		return fmt.Errorf("marshal class starting items: %w", err)
	}
	featuresJSON, err := json.Marshal(class.Features)
	if err != nil {
		return fmt.Errorf("marshal class features: %w", err)
	}
	hopeFeatureJSON, err := json.Marshal(class.HopeFeature)
	if err != nil {
		return fmt.Errorf("marshal class hope feature: %w", err)
	}
	domainIDsJSON, err := json.Marshal(class.DomainIDs)
	if err != nil {
		return fmt.Errorf("marshal class domain ids: %w", err)
	}

	return s.q.PutDaggerheartClass(ctx, db.PutDaggerheartClassParams{
		ID:                class.ID,
		Name:              class.Name,
		StartingEvasion:   int64(class.StartingEvasion),
		StartingHp:        int64(class.StartingHP),
		StartingItemsJson: string(startingItemsJSON),
		FeaturesJson:      string(featuresJSON),
		HopeFeatureJson:   string(hopeFeatureJSON),
		DomainIdsJson:     string(domainIDsJSON),
		CreatedAt:         toMillis(class.CreatedAt),
		UpdatedAt:         toMillis(class.UpdatedAt),
	})
}

// GetDaggerheartClass retrieves a Daggerheart class catalog entry.
func (s *Store) GetDaggerheartClass(ctx context.Context, id string) (storage.DaggerheartClass, error) {
	if err := s.validateContentStore(ctx); err != nil {
		return storage.DaggerheartClass{}, err
	}
	if err := requireCatalogEntryID(id, "class"); err != nil {
		return storage.DaggerheartClass{}, err
	}

	row, err := s.q.GetDaggerheartClass(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartClass{}, storage.ErrNotFound
		}
		return storage.DaggerheartClass{}, fmt.Errorf("get daggerheart class: %w", err)
	}

	class, err := dbDaggerheartClassToStorage(row)
	if err != nil {
		return storage.DaggerheartClass{}, err
	}
	return class, nil
}

// ListDaggerheartClasses lists all Daggerheart class catalog entries.
func (s *Store) ListDaggerheartClasses(ctx context.Context) ([]storage.DaggerheartClass, error) {
	if err := s.validateContentStore(ctx); err != nil {
		return nil, err
	}

	rows, err := s.q.ListDaggerheartClasses(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart classes: %w", err)
	}

	classes := make([]storage.DaggerheartClass, 0, len(rows))
	for _, row := range rows {
		class, err := dbDaggerheartClassToStorage(row)
		if err != nil {
			return nil, err
		}
		classes = append(classes, class)
	}
	return classes, nil
}

// DeleteDaggerheartClass removes a Daggerheart class catalog entry.
func (s *Store) DeleteDaggerheartClass(ctx context.Context, id string) error {
	if err := s.validateContentStore(ctx); err != nil {
		return err
	}
	if err := requireCatalogEntryID(id, "class"); err != nil {
		return err
	}

	return s.q.DeleteDaggerheartClass(ctx, id)
}

// PutDaggerheartSubclass persists a Daggerheart subclass catalog entry.
func (s *Store) PutDaggerheartSubclass(ctx context.Context, subclass storage.DaggerheartSubclass) error {
	if err := s.validateContentStore(ctx); err != nil {
		return err
	}
	if err := requireCatalogEntryID(subclass.ID, "subclass"); err != nil {
		return err
	}

	foundationJSON, err := json.Marshal(subclass.FoundationFeatures)
	if err != nil {
		return fmt.Errorf("marshal subclass foundation features: %w", err)
	}
	specializationJSON, err := json.Marshal(subclass.SpecializationFeatures)
	if err != nil {
		return fmt.Errorf("marshal subclass specialization features: %w", err)
	}
	masteryJSON, err := json.Marshal(subclass.MasteryFeatures)
	if err != nil {
		return fmt.Errorf("marshal subclass mastery features: %w", err)
	}

	return s.q.PutDaggerheartSubclass(ctx, db.PutDaggerheartSubclassParams{
		ID:                         subclass.ID,
		Name:                       subclass.Name,
		SpellcastTrait:             subclass.SpellcastTrait,
		FoundationFeaturesJson:     string(foundationJSON),
		SpecializationFeaturesJson: string(specializationJSON),
		MasteryFeaturesJson:        string(masteryJSON),
		CreatedAt:                  toMillis(subclass.CreatedAt),
		UpdatedAt:                  toMillis(subclass.UpdatedAt),
	})
}

// GetDaggerheartSubclass retrieves a Daggerheart subclass catalog entry.
func (s *Store) GetDaggerheartSubclass(ctx context.Context, id string) (storage.DaggerheartSubclass, error) {
	if err := s.validateContentStore(ctx); err != nil {
		return storage.DaggerheartSubclass{}, err
	}
	if err := requireCatalogEntryID(id, "subclass"); err != nil {
		return storage.DaggerheartSubclass{}, err
	}

	row, err := s.q.GetDaggerheartSubclass(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartSubclass{}, storage.ErrNotFound
		}
		return storage.DaggerheartSubclass{}, fmt.Errorf("get daggerheart subclass: %w", err)
	}

	subclass, err := dbDaggerheartSubclassToStorage(row)
	if err != nil {
		return storage.DaggerheartSubclass{}, err
	}
	return subclass, nil
}

// ListDaggerheartSubclasses lists all Daggerheart subclass catalog entries.
func (s *Store) ListDaggerheartSubclasses(ctx context.Context) ([]storage.DaggerheartSubclass, error) {
	if err := s.validateContentStore(ctx); err != nil {
		return nil, err
	}

	rows, err := s.q.ListDaggerheartSubclasses(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart subclasses: %w", err)
	}

	subclasses := make([]storage.DaggerheartSubclass, 0, len(rows))
	for _, row := range rows {
		subclass, err := dbDaggerheartSubclassToStorage(row)
		if err != nil {
			return nil, err
		}
		subclasses = append(subclasses, subclass)
	}
	return subclasses, nil
}

// DeleteDaggerheartSubclass removes a Daggerheart subclass catalog entry.
func (s *Store) DeleteDaggerheartSubclass(ctx context.Context, id string) error {
	if err := s.validateContentStore(ctx); err != nil {
		return err
	}
	if err := requireCatalogEntryID(id, "subclass"); err != nil {
		return err
	}

	return s.q.DeleteDaggerheartSubclass(ctx, id)
}

// PutDaggerheartHeritage persists a Daggerheart heritage catalog entry.
func (s *Store) PutDaggerheartHeritage(ctx context.Context, heritage storage.DaggerheartHeritage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(heritage.ID) == "" {
		return fmt.Errorf("heritage id is required")
	}

	featuresJSON, err := json.Marshal(heritage.Features)
	if err != nil {
		return fmt.Errorf("marshal heritage features: %w", err)
	}

	return s.q.PutDaggerheartHeritage(ctx, db.PutDaggerheartHeritageParams{
		ID:           heritage.ID,
		Name:         heritage.Name,
		Kind:         heritage.Kind,
		FeaturesJson: string(featuresJSON),
		CreatedAt:    toMillis(heritage.CreatedAt),
		UpdatedAt:    toMillis(heritage.UpdatedAt),
	})
}

// GetDaggerheartHeritage retrieves a Daggerheart heritage catalog entry.
func (s *Store) GetDaggerheartHeritage(ctx context.Context, id string) (storage.DaggerheartHeritage, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartHeritage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartHeritage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartHeritage{}, fmt.Errorf("heritage id is required")
	}

	row, err := s.q.GetDaggerheartHeritage(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartHeritage{}, storage.ErrNotFound
		}
		return storage.DaggerheartHeritage{}, fmt.Errorf("get daggerheart heritage: %w", err)
	}

	heritage, err := dbDaggerheartHeritageToStorage(row)
	if err != nil {
		return storage.DaggerheartHeritage{}, err
	}
	return heritage, nil
}

// ListDaggerheartHeritages lists all Daggerheart heritage catalog entries.
func (s *Store) ListDaggerheartHeritages(ctx context.Context) ([]storage.DaggerheartHeritage, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartHeritages(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart heritages: %w", err)
	}

	heritages := make([]storage.DaggerheartHeritage, 0, len(rows))
	for _, row := range rows {
		heritage, err := dbDaggerheartHeritageToStorage(row)
		if err != nil {
			return nil, err
		}
		heritages = append(heritages, heritage)
	}
	return heritages, nil
}

// DeleteDaggerheartHeritage removes a Daggerheart heritage catalog entry.
func (s *Store) DeleteDaggerheartHeritage(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("heritage id is required")
	}

	return s.q.DeleteDaggerheartHeritage(ctx, id)
}

// PutDaggerheartExperience persists a Daggerheart experience catalog entry.
func (s *Store) PutDaggerheartExperience(ctx context.Context, experience storage.DaggerheartExperienceEntry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(experience.ID) == "" {
		return fmt.Errorf("experience id is required")
	}

	return s.q.PutDaggerheartExperience(ctx, db.PutDaggerheartExperienceParams{
		ID:          experience.ID,
		Name:        experience.Name,
		Description: experience.Description,
		CreatedAt:   toMillis(experience.CreatedAt),
		UpdatedAt:   toMillis(experience.UpdatedAt),
	})
}

// GetDaggerheartExperience retrieves a Daggerheart experience catalog entry.
func (s *Store) GetDaggerheartExperience(ctx context.Context, id string) (storage.DaggerheartExperienceEntry, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartExperienceEntry{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartExperienceEntry{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartExperienceEntry{}, fmt.Errorf("experience id is required")
	}

	row, err := s.q.GetDaggerheartExperience(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartExperienceEntry{}, storage.ErrNotFound
		}
		return storage.DaggerheartExperienceEntry{}, fmt.Errorf("get daggerheart experience: %w", err)
	}

	return dbDaggerheartExperienceToStorage(row), nil
}

// ListDaggerheartExperiences lists all Daggerheart experience catalog entries.
func (s *Store) ListDaggerheartExperiences(ctx context.Context) ([]storage.DaggerheartExperienceEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartExperiences(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart experiences: %w", err)
	}

	experiences := make([]storage.DaggerheartExperienceEntry, 0, len(rows))
	for _, row := range rows {
		experiences = append(experiences, dbDaggerheartExperienceToStorage(row))
	}
	return experiences, nil
}

// DeleteDaggerheartExperience removes a Daggerheart experience catalog entry.
func (s *Store) DeleteDaggerheartExperience(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("experience id is required")
	}

	return s.q.DeleteDaggerheartExperience(ctx, id)
}

// PutDaggerheartAdversaryEntry persists a Daggerheart adversary catalog entry.
func (s *Store) PutDaggerheartAdversaryEntry(ctx context.Context, adversary storage.DaggerheartAdversaryEntry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(adversary.ID) == "" {
		return fmt.Errorf("adversary id is required")
	}

	attackJSON, err := json.Marshal(adversary.StandardAttack)
	if err != nil {
		return fmt.Errorf("marshal adversary standard attack: %w", err)
	}
	experiencesJSON, err := json.Marshal(adversary.Experiences)
	if err != nil {
		return fmt.Errorf("marshal adversary experiences: %w", err)
	}
	featuresJSON, err := json.Marshal(adversary.Features)
	if err != nil {
		return fmt.Errorf("marshal adversary features: %w", err)
	}

	return s.q.PutDaggerheartAdversaryEntry(ctx, db.PutDaggerheartAdversaryEntryParams{
		ID:                 adversary.ID,
		Name:               adversary.Name,
		Tier:               int64(adversary.Tier),
		Role:               adversary.Role,
		Description:        adversary.Description,
		Motives:            adversary.Motives,
		Difficulty:         int64(adversary.Difficulty),
		MajorThreshold:     int64(adversary.MajorThreshold),
		SevereThreshold:    int64(adversary.SevereThreshold),
		Hp:                 int64(adversary.HP),
		Stress:             int64(adversary.Stress),
		Armor:              int64(adversary.Armor),
		AttackModifier:     int64(adversary.AttackModifier),
		StandardAttackJson: string(attackJSON),
		ExperiencesJson:    string(experiencesJSON),
		FeaturesJson:       string(featuresJSON),
		CreatedAt:          toMillis(adversary.CreatedAt),
		UpdatedAt:          toMillis(adversary.UpdatedAt),
	})
}

// GetDaggerheartAdversaryEntry retrieves a Daggerheart adversary catalog entry.
func (s *Store) GetDaggerheartAdversaryEntry(ctx context.Context, id string) (storage.DaggerheartAdversaryEntry, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartAdversaryEntry{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartAdversaryEntry{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartAdversaryEntry{}, fmt.Errorf("adversary id is required")
	}

	row, err := s.q.GetDaggerheartAdversaryEntry(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartAdversaryEntry{}, storage.ErrNotFound
		}
		return storage.DaggerheartAdversaryEntry{}, fmt.Errorf("get daggerheart adversary: %w", err)
	}

	return dbDaggerheartAdversaryEntryToStorage(row)
}

// ListDaggerheartAdversaryEntries lists all Daggerheart adversary catalog entries.
func (s *Store) ListDaggerheartAdversaryEntries(ctx context.Context) ([]storage.DaggerheartAdversaryEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartAdversaryEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart adversaries: %w", err)
	}

	adversaries := make([]storage.DaggerheartAdversaryEntry, 0, len(rows))
	for _, row := range rows {
		entry, err := dbDaggerheartAdversaryEntryToStorage(row)
		if err != nil {
			return nil, err
		}
		adversaries = append(adversaries, entry)
	}
	return adversaries, nil
}

// DeleteDaggerheartAdversaryEntry removes a Daggerheart adversary catalog entry.
func (s *Store) DeleteDaggerheartAdversaryEntry(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("adversary id is required")
	}

	return s.q.DeleteDaggerheartAdversaryEntry(ctx, id)
}

// PutDaggerheartBeastform persists a Daggerheart beastform catalog entry.
func (s *Store) PutDaggerheartBeastform(ctx context.Context, beastform storage.DaggerheartBeastformEntry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(beastform.ID) == "" {
		return fmt.Errorf("beastform id is required")
	}

	attackJSON, err := json.Marshal(beastform.Attack)
	if err != nil {
		return fmt.Errorf("marshal beastform attack: %w", err)
	}
	advantagesJSON, err := json.Marshal(beastform.Advantages)
	if err != nil {
		return fmt.Errorf("marshal beastform advantages: %w", err)
	}
	featuresJSON, err := json.Marshal(beastform.Features)
	if err != nil {
		return fmt.Errorf("marshal beastform features: %w", err)
	}

	return s.q.PutDaggerheartBeastform(ctx, db.PutDaggerheartBeastformParams{
		ID:             beastform.ID,
		Name:           beastform.Name,
		Tier:           int64(beastform.Tier),
		Examples:       beastform.Examples,
		Trait:          beastform.Trait,
		TraitBonus:     int64(beastform.TraitBonus),
		EvasionBonus:   int64(beastform.EvasionBonus),
		AttackJson:     string(attackJSON),
		AdvantagesJson: string(advantagesJSON),
		FeaturesJson:   string(featuresJSON),
		CreatedAt:      toMillis(beastform.CreatedAt),
		UpdatedAt:      toMillis(beastform.UpdatedAt),
	})
}

// GetDaggerheartBeastform retrieves a Daggerheart beastform catalog entry.
func (s *Store) GetDaggerheartBeastform(ctx context.Context, id string) (storage.DaggerheartBeastformEntry, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartBeastformEntry{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartBeastformEntry{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartBeastformEntry{}, fmt.Errorf("beastform id is required")
	}

	row, err := s.q.GetDaggerheartBeastform(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartBeastformEntry{}, storage.ErrNotFound
		}
		return storage.DaggerheartBeastformEntry{}, fmt.Errorf("get daggerheart beastform: %w", err)
	}

	return dbDaggerheartBeastformToStorage(row)
}

// ListDaggerheartBeastforms lists all Daggerheart beastform catalog entries.
func (s *Store) ListDaggerheartBeastforms(ctx context.Context) ([]storage.DaggerheartBeastformEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartBeastforms(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart beastforms: %w", err)
	}

	beastforms := make([]storage.DaggerheartBeastformEntry, 0, len(rows))
	for _, row := range rows {
		entry, err := dbDaggerheartBeastformToStorage(row)
		if err != nil {
			return nil, err
		}
		beastforms = append(beastforms, entry)
	}
	return beastforms, nil
}

// DeleteDaggerheartBeastform removes a Daggerheart beastform catalog entry.
func (s *Store) DeleteDaggerheartBeastform(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("beastform id is required")
	}

	return s.q.DeleteDaggerheartBeastform(ctx, id)
}

// PutDaggerheartCompanionExperience persists a Daggerheart companion experience catalog entry.
func (s *Store) PutDaggerheartCompanionExperience(ctx context.Context, experience storage.DaggerheartCompanionExperienceEntry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(experience.ID) == "" {
		return fmt.Errorf("companion experience id is required")
	}

	return s.q.PutDaggerheartCompanionExperience(ctx, db.PutDaggerheartCompanionExperienceParams{
		ID:          experience.ID,
		Name:        experience.Name,
		Description: experience.Description,
		CreatedAt:   toMillis(experience.CreatedAt),
		UpdatedAt:   toMillis(experience.UpdatedAt),
	})
}

// GetDaggerheartCompanionExperience retrieves a Daggerheart companion experience catalog entry.
func (s *Store) GetDaggerheartCompanionExperience(ctx context.Context, id string) (storage.DaggerheartCompanionExperienceEntry, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartCompanionExperienceEntry{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartCompanionExperienceEntry{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartCompanionExperienceEntry{}, fmt.Errorf("companion experience id is required")
	}

	row, err := s.q.GetDaggerheartCompanionExperience(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartCompanionExperienceEntry{}, storage.ErrNotFound
		}
		return storage.DaggerheartCompanionExperienceEntry{}, fmt.Errorf("get daggerheart companion experience: %w", err)
	}

	return dbDaggerheartCompanionExperienceToStorage(row), nil
}

// ListDaggerheartCompanionExperiences lists all Daggerheart companion experience catalog entries.
func (s *Store) ListDaggerheartCompanionExperiences(ctx context.Context) ([]storage.DaggerheartCompanionExperienceEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartCompanionExperiences(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart companion experiences: %w", err)
	}

	experiences := make([]storage.DaggerheartCompanionExperienceEntry, 0, len(rows))
	for _, row := range rows {
		experiences = append(experiences, dbDaggerheartCompanionExperienceToStorage(row))
	}
	return experiences, nil
}

// DeleteDaggerheartCompanionExperience removes a Daggerheart companion experience catalog entry.
func (s *Store) DeleteDaggerheartCompanionExperience(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("companion experience id is required")
	}

	return s.q.DeleteDaggerheartCompanionExperience(ctx, id)
}

// PutDaggerheartLootEntry persists a Daggerheart loot catalog entry.
func (s *Store) PutDaggerheartLootEntry(ctx context.Context, entry storage.DaggerheartLootEntry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(entry.ID) == "" {
		return fmt.Errorf("loot entry id is required")
	}

	return s.q.PutDaggerheartLootEntry(ctx, db.PutDaggerheartLootEntryParams{
		ID:          entry.ID,
		Name:        entry.Name,
		Roll:        int64(entry.Roll),
		Description: entry.Description,
		CreatedAt:   toMillis(entry.CreatedAt),
		UpdatedAt:   toMillis(entry.UpdatedAt),
	})
}

// GetDaggerheartLootEntry retrieves a Daggerheart loot catalog entry.
func (s *Store) GetDaggerheartLootEntry(ctx context.Context, id string) (storage.DaggerheartLootEntry, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartLootEntry{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartLootEntry{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartLootEntry{}, fmt.Errorf("loot entry id is required")
	}

	row, err := s.q.GetDaggerheartLootEntry(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartLootEntry{}, storage.ErrNotFound
		}
		return storage.DaggerheartLootEntry{}, fmt.Errorf("get daggerheart loot entry: %w", err)
	}

	return dbDaggerheartLootEntryToStorage(row), nil
}

// ListDaggerheartLootEntries lists all Daggerheart loot catalog entries.
func (s *Store) ListDaggerheartLootEntries(ctx context.Context) ([]storage.DaggerheartLootEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartLootEntries(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart loot entries: %w", err)
	}

	entries := make([]storage.DaggerheartLootEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, dbDaggerheartLootEntryToStorage(row))
	}
	return entries, nil
}

// DeleteDaggerheartLootEntry removes a Daggerheart loot catalog entry.
func (s *Store) DeleteDaggerheartLootEntry(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("loot entry id is required")
	}

	return s.q.DeleteDaggerheartLootEntry(ctx, id)
}

// PutDaggerheartDamageType persists a Daggerheart damage type catalog entry.
func (s *Store) PutDaggerheartDamageType(ctx context.Context, entry storage.DaggerheartDamageTypeEntry) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(entry.ID) == "" {
		return fmt.Errorf("damage type id is required")
	}

	return s.q.PutDaggerheartDamageType(ctx, db.PutDaggerheartDamageTypeParams{
		ID:          entry.ID,
		Name:        entry.Name,
		Description: entry.Description,
		CreatedAt:   toMillis(entry.CreatedAt),
		UpdatedAt:   toMillis(entry.UpdatedAt),
	})
}

// GetDaggerheartDamageType retrieves a Daggerheart damage type catalog entry.
func (s *Store) GetDaggerheartDamageType(ctx context.Context, id string) (storage.DaggerheartDamageTypeEntry, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartDamageTypeEntry{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartDamageTypeEntry{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartDamageTypeEntry{}, fmt.Errorf("damage type id is required")
	}

	row, err := s.q.GetDaggerheartDamageType(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartDamageTypeEntry{}, storage.ErrNotFound
		}
		return storage.DaggerheartDamageTypeEntry{}, fmt.Errorf("get daggerheart damage type: %w", err)
	}

	return dbDaggerheartDamageTypeToStorage(row), nil
}

// ListDaggerheartDamageTypes lists all Daggerheart damage type catalog entries.
func (s *Store) ListDaggerheartDamageTypes(ctx context.Context) ([]storage.DaggerheartDamageTypeEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartDamageTypes(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart damage types: %w", err)
	}

	entries := make([]storage.DaggerheartDamageTypeEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, dbDaggerheartDamageTypeToStorage(row))
	}
	return entries, nil
}

// DeleteDaggerheartDamageType removes a Daggerheart damage type catalog entry.
func (s *Store) DeleteDaggerheartDamageType(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("damage type id is required")
	}

	return s.q.DeleteDaggerheartDamageType(ctx, id)
}

// PutDaggerheartDomain persists a Daggerheart domain catalog entry.
func (s *Store) PutDaggerheartDomain(ctx context.Context, domain storage.DaggerheartDomain) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(domain.ID) == "" {
		return fmt.Errorf("domain id is required")
	}

	return s.q.PutDaggerheartDomain(ctx, db.PutDaggerheartDomainParams{
		ID:          domain.ID,
		Name:        domain.Name,
		Description: domain.Description,
		CreatedAt:   toMillis(domain.CreatedAt),
		UpdatedAt:   toMillis(domain.UpdatedAt),
	})
}

// GetDaggerheartDomain retrieves a Daggerheart domain catalog entry.
func (s *Store) GetDaggerheartDomain(ctx context.Context, id string) (storage.DaggerheartDomain, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartDomain{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartDomain{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartDomain{}, fmt.Errorf("domain id is required")
	}

	row, err := s.q.GetDaggerheartDomain(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartDomain{}, storage.ErrNotFound
		}
		return storage.DaggerheartDomain{}, fmt.Errorf("get daggerheart domain: %w", err)
	}

	return dbDaggerheartDomainToStorage(row), nil
}

// ListDaggerheartDomains lists all Daggerheart domain catalog entries.
func (s *Store) ListDaggerheartDomains(ctx context.Context) ([]storage.DaggerheartDomain, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartDomains(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart domains: %w", err)
	}

	domains := make([]storage.DaggerheartDomain, 0, len(rows))
	for _, row := range rows {
		domains = append(domains, dbDaggerheartDomainToStorage(row))
	}
	return domains, nil
}

// DeleteDaggerheartDomain removes a Daggerheart domain catalog entry.
func (s *Store) DeleteDaggerheartDomain(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("domain id is required")
	}

	return s.q.DeleteDaggerheartDomain(ctx, id)
}

// PutDaggerheartDomainCard persists a Daggerheart domain card catalog entry.
func (s *Store) PutDaggerheartDomainCard(ctx context.Context, card storage.DaggerheartDomainCard) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(card.ID) == "" {
		return fmt.Errorf("domain card id is required")
	}

	return s.q.PutDaggerheartDomainCard(ctx, db.PutDaggerheartDomainCardParams{
		ID:          card.ID,
		Name:        card.Name,
		DomainID:    card.DomainID,
		Level:       int64(card.Level),
		Type:        card.Type,
		RecallCost:  int64(card.RecallCost),
		UsageLimit:  card.UsageLimit,
		FeatureText: card.FeatureText,
		CreatedAt:   toMillis(card.CreatedAt),
		UpdatedAt:   toMillis(card.UpdatedAt),
	})
}

// GetDaggerheartDomainCard retrieves a Daggerheart domain card catalog entry.
func (s *Store) GetDaggerheartDomainCard(ctx context.Context, id string) (storage.DaggerheartDomainCard, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartDomainCard{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartDomainCard{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartDomainCard{}, fmt.Errorf("domain card id is required")
	}

	row, err := s.q.GetDaggerheartDomainCard(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartDomainCard{}, storage.ErrNotFound
		}
		return storage.DaggerheartDomainCard{}, fmt.Errorf("get daggerheart domain card: %w", err)
	}

	return dbDaggerheartDomainCardToStorage(row), nil
}

// ListDaggerheartDomainCards lists all Daggerheart domain card catalog entries.
func (s *Store) ListDaggerheartDomainCards(ctx context.Context) ([]storage.DaggerheartDomainCard, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartDomainCards(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart domain cards: %w", err)
	}

	cards := make([]storage.DaggerheartDomainCard, 0, len(rows))
	for _, row := range rows {
		cards = append(cards, dbDaggerheartDomainCardToStorage(row))
	}
	return cards, nil
}

// ListDaggerheartDomainCardsByDomain lists Daggerheart domain cards for a domain.
func (s *Store) ListDaggerheartDomainCardsByDomain(ctx context.Context, domainID string) ([]storage.DaggerheartDomainCard, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(domainID) == "" {
		return nil, fmt.Errorf("domain id is required")
	}

	rows, err := s.q.ListDaggerheartDomainCardsByDomain(ctx, domainID)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart domain cards by domain: %w", err)
	}

	cards := make([]storage.DaggerheartDomainCard, 0, len(rows))
	for _, row := range rows {
		cards = append(cards, dbDaggerheartDomainCardToStorage(row))
	}
	return cards, nil
}

// DeleteDaggerheartDomainCard removes a Daggerheart domain card catalog entry.
func (s *Store) DeleteDaggerheartDomainCard(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("domain card id is required")
	}

	return s.q.DeleteDaggerheartDomainCard(ctx, id)
}

// PutDaggerheartWeapon persists a Daggerheart weapon catalog entry.
func (s *Store) PutDaggerheartWeapon(ctx context.Context, weapon storage.DaggerheartWeapon) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(weapon.ID) == "" {
		return fmt.Errorf("weapon id is required")
	}

	damageDiceJSON, err := json.Marshal(weapon.DamageDice)
	if err != nil {
		return fmt.Errorf("marshal weapon damage dice: %w", err)
	}

	return s.q.PutDaggerheartWeapon(ctx, db.PutDaggerheartWeaponParams{
		ID:             weapon.ID,
		Name:           weapon.Name,
		Category:       weapon.Category,
		Tier:           int64(weapon.Tier),
		Trait:          weapon.Trait,
		Range:          weapon.Range,
		DamageDiceJson: string(damageDiceJSON),
		DamageType:     weapon.DamageType,
		Burden:         int64(weapon.Burden),
		Feature:        weapon.Feature,
		CreatedAt:      toMillis(weapon.CreatedAt),
		UpdatedAt:      toMillis(weapon.UpdatedAt),
	})
}

// GetDaggerheartWeapon retrieves a Daggerheart weapon catalog entry.
func (s *Store) GetDaggerheartWeapon(ctx context.Context, id string) (storage.DaggerheartWeapon, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartWeapon{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartWeapon{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartWeapon{}, fmt.Errorf("weapon id is required")
	}

	row, err := s.q.GetDaggerheartWeapon(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartWeapon{}, storage.ErrNotFound
		}
		return storage.DaggerheartWeapon{}, fmt.Errorf("get daggerheart weapon: %w", err)
	}

	weapon, err := dbDaggerheartWeaponToStorage(row)
	if err != nil {
		return storage.DaggerheartWeapon{}, err
	}
	return weapon, nil
}

// ListDaggerheartWeapons lists all Daggerheart weapon catalog entries.
func (s *Store) ListDaggerheartWeapons(ctx context.Context) ([]storage.DaggerheartWeapon, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartWeapons(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart weapons: %w", err)
	}

	weapons := make([]storage.DaggerheartWeapon, 0, len(rows))
	for _, row := range rows {
		weapon, err := dbDaggerheartWeaponToStorage(row)
		if err != nil {
			return nil, err
		}
		weapons = append(weapons, weapon)
	}
	return weapons, nil
}

// DeleteDaggerheartWeapon removes a Daggerheart weapon catalog entry.
func (s *Store) DeleteDaggerheartWeapon(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("weapon id is required")
	}

	return s.q.DeleteDaggerheartWeapon(ctx, id)
}

// PutDaggerheartArmor persists a Daggerheart armor catalog entry.
func (s *Store) PutDaggerheartArmor(ctx context.Context, armor storage.DaggerheartArmor) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(armor.ID) == "" {
		return fmt.Errorf("armor id is required")
	}

	return s.q.PutDaggerheartArmor(ctx, db.PutDaggerheartArmorParams{
		ID:                  armor.ID,
		Name:                armor.Name,
		Tier:                int64(armor.Tier),
		BaseMajorThreshold:  int64(armor.BaseMajorThreshold),
		BaseSevereThreshold: int64(armor.BaseSevereThreshold),
		ArmorScore:          int64(armor.ArmorScore),
		Feature:             armor.Feature,
		CreatedAt:           toMillis(armor.CreatedAt),
		UpdatedAt:           toMillis(armor.UpdatedAt),
	})
}

// GetDaggerheartArmor retrieves a Daggerheart armor catalog entry.
func (s *Store) GetDaggerheartArmor(ctx context.Context, id string) (storage.DaggerheartArmor, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartArmor{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartArmor{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartArmor{}, fmt.Errorf("armor id is required")
	}

	row, err := s.q.GetDaggerheartArmor(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartArmor{}, storage.ErrNotFound
		}
		return storage.DaggerheartArmor{}, fmt.Errorf("get daggerheart armor: %w", err)
	}

	return dbDaggerheartArmorToStorage(row), nil
}

// ListDaggerheartArmor lists all Daggerheart armor catalog entries.
func (s *Store) ListDaggerheartArmor(ctx context.Context) ([]storage.DaggerheartArmor, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartArmor(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart armor: %w", err)
	}

	armor := make([]storage.DaggerheartArmor, 0, len(rows))
	for _, row := range rows {
		armor = append(armor, dbDaggerheartArmorToStorage(row))
	}
	return armor, nil
}

// DeleteDaggerheartArmor removes a Daggerheart armor catalog entry.
func (s *Store) DeleteDaggerheartArmor(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("armor id is required")
	}

	return s.q.DeleteDaggerheartArmor(ctx, id)
}

// PutDaggerheartItem persists a Daggerheart item catalog entry.
func (s *Store) PutDaggerheartItem(ctx context.Context, item storage.DaggerheartItem) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(item.ID) == "" {
		return fmt.Errorf("item id is required")
	}

	return s.q.PutDaggerheartItem(ctx, db.PutDaggerheartItemParams{
		ID:          item.ID,
		Name:        item.Name,
		Rarity:      item.Rarity,
		Kind:        item.Kind,
		StackMax:    int64(item.StackMax),
		Description: item.Description,
		EffectText:  item.EffectText,
		CreatedAt:   toMillis(item.CreatedAt),
		UpdatedAt:   toMillis(item.UpdatedAt),
	})
}

// GetDaggerheartItem retrieves a Daggerheart item catalog entry.
func (s *Store) GetDaggerheartItem(ctx context.Context, id string) (storage.DaggerheartItem, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartItem{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartItem{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartItem{}, fmt.Errorf("item id is required")
	}

	row, err := s.q.GetDaggerheartItem(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartItem{}, storage.ErrNotFound
		}
		return storage.DaggerheartItem{}, fmt.Errorf("get daggerheart item: %w", err)
	}

	return dbDaggerheartItemToStorage(row), nil
}

// ListDaggerheartItems lists all Daggerheart item catalog entries.
func (s *Store) ListDaggerheartItems(ctx context.Context) ([]storage.DaggerheartItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart items: %w", err)
	}

	items := make([]storage.DaggerheartItem, 0, len(rows))
	for _, row := range rows {
		items = append(items, dbDaggerheartItemToStorage(row))
	}
	return items, nil
}

// DeleteDaggerheartItem removes a Daggerheart item catalog entry.
func (s *Store) DeleteDaggerheartItem(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("item id is required")
	}

	return s.q.DeleteDaggerheartItem(ctx, id)
}

// PutDaggerheartEnvironment persists a Daggerheart environment catalog entry.
func (s *Store) PutDaggerheartEnvironment(ctx context.Context, env storage.DaggerheartEnvironment) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(env.ID) == "" {
		return fmt.Errorf("environment id is required")
	}

	impulsesJSON, err := json.Marshal(env.Impulses)
	if err != nil {
		return fmt.Errorf("marshal environment impulses: %w", err)
	}
	adversariesJSON, err := json.Marshal(env.PotentialAdversaryIDs)
	if err != nil {
		return fmt.Errorf("marshal environment adversaries: %w", err)
	}
	featuresJSON, err := json.Marshal(env.Features)
	if err != nil {
		return fmt.Errorf("marshal environment features: %w", err)
	}
	promptsJSON, err := json.Marshal(env.Prompts)
	if err != nil {
		return fmt.Errorf("marshal environment prompts: %w", err)
	}

	return s.q.PutDaggerheartEnvironment(ctx, db.PutDaggerheartEnvironmentParams{
		ID:                        env.ID,
		Name:                      env.Name,
		Tier:                      int64(env.Tier),
		Type:                      env.Type,
		Difficulty:                int64(env.Difficulty),
		ImpulsesJson:              string(impulsesJSON),
		PotentialAdversaryIdsJson: string(adversariesJSON),
		FeaturesJson:              string(featuresJSON),
		PromptsJson:               string(promptsJSON),
		CreatedAt:                 toMillis(env.CreatedAt),
		UpdatedAt:                 toMillis(env.UpdatedAt),
	})
}

// GetDaggerheartEnvironment retrieves a Daggerheart environment catalog entry.
func (s *Store) GetDaggerheartEnvironment(ctx context.Context, id string) (storage.DaggerheartEnvironment, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartEnvironment{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartEnvironment{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartEnvironment{}, fmt.Errorf("environment id is required")
	}

	row, err := s.q.GetDaggerheartEnvironment(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartEnvironment{}, storage.ErrNotFound
		}
		return storage.DaggerheartEnvironment{}, fmt.Errorf("get daggerheart environment: %w", err)
	}

	env, err := dbDaggerheartEnvironmentToStorage(row)
	if err != nil {
		return storage.DaggerheartEnvironment{}, err
	}
	return env, nil
}

// ListDaggerheartEnvironments lists all Daggerheart environment catalog entries.
func (s *Store) ListDaggerheartEnvironments(ctx context.Context) ([]storage.DaggerheartEnvironment, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}

	rows, err := s.q.ListDaggerheartEnvironments(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart environments: %w", err)
	}

	envs := make([]storage.DaggerheartEnvironment, 0, len(rows))
	for _, row := range rows {
		env, err := dbDaggerheartEnvironmentToStorage(row)
		if err != nil {
			return nil, err
		}
		envs = append(envs, env)
	}
	return envs, nil
}

// DeleteDaggerheartEnvironment removes a Daggerheart environment catalog entry.
func (s *Store) DeleteDaggerheartEnvironment(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("environment id is required")
	}

	return s.q.DeleteDaggerheartEnvironment(ctx, id)
}

// ListDaggerheartContentStrings returns localized content strings for content ids.
func (s *Store) ListDaggerheartContentStrings(ctx context.Context, contentType string, contentIDs []string, locale string) ([]storage.DaggerheartContentString, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(contentType) == "" {
		return nil, fmt.Errorf("content type is required")
	}
	if strings.TrimSpace(locale) == "" {
		return nil, fmt.Errorf("locale is required")
	}
	if len(contentIDs) == 0 {
		return []storage.DaggerheartContentString{}, nil
	}

	rows, err := s.q.ListDaggerheartContentStringsByIDs(ctx, db.ListDaggerheartContentStringsByIDsParams{
		ContentType: contentType,
		Locale:      locale,
		ContentIds:  contentIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("list daggerheart content strings: %w", err)
	}
	entries := make([]storage.DaggerheartContentString, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, dbDaggerheartContentStringToStorage(row))
	}
	return entries, nil
}

// PutDaggerheartContentString upserts a localized content string.
func (s *Store) PutDaggerheartContentString(ctx context.Context, entry storage.DaggerheartContentString) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(entry.ContentID) == "" {
		return fmt.Errorf("content id is required")
	}
	if strings.TrimSpace(entry.Field) == "" {
		return fmt.Errorf("field is required")
	}
	if strings.TrimSpace(entry.Locale) == "" {
		return fmt.Errorf("locale is required")
	}

	createdAt := entry.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}
	updatedAt := entry.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}

	return s.q.PutDaggerheartContentString(ctx, db.PutDaggerheartContentStringParams{
		ContentID:   entry.ContentID,
		ContentType: entry.ContentType,
		Field:       entry.Field,
		Locale:      entry.Locale,
		Text:        entry.Text,
		CreatedAt:   toMillis(createdAt),
		UpdatedAt:   toMillis(updatedAt),
	})
}
