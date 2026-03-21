package daggerheartcontent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

func (s *Store) PutDaggerheartClass(ctx context.Context, class contentstore.DaggerheartClass) error {
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
		CreatedAt:         sqliteutil.ToMillis(class.CreatedAt),
		UpdatedAt:         sqliteutil.ToMillis(class.UpdatedAt),
	})
}

// GetDaggerheartClass retrieves a Daggerheart class catalog entry.
func (s *Store) GetDaggerheartClass(ctx context.Context, id string) (contentstore.DaggerheartClass, error) {
	if err := s.validateContentStore(ctx); err != nil {
		return contentstore.DaggerheartClass{}, err
	}
	if err := requireCatalogEntryID(id, "class"); err != nil {
		return contentstore.DaggerheartClass{}, err
	}

	row, err := s.q.GetDaggerheartClass(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contentstore.DaggerheartClass{}, storage.ErrNotFound
		}
		return contentstore.DaggerheartClass{}, fmt.Errorf("get daggerheart class: %w", err)
	}

	class, err := dbDaggerheartClassToStorage(row)
	if err != nil {
		return contentstore.DaggerheartClass{}, err
	}
	return class, nil
}

// ListDaggerheartClasses lists all Daggerheart class catalog entries.
func (s *Store) ListDaggerheartClasses(ctx context.Context) ([]contentstore.DaggerheartClass, error) {
	if err := s.validateContentStore(ctx); err != nil {
		return nil, err
	}

	rows, err := s.q.ListDaggerheartClasses(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart classes: %w", err)
	}

	classes := make([]contentstore.DaggerheartClass, 0, len(rows))
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
func (s *Store) PutDaggerheartSubclass(ctx context.Context, subclass contentstore.DaggerheartSubclass) error {
	if err := s.validateContentStore(ctx); err != nil {
		return err
	}
	if err := requireCatalogEntryID(subclass.ID, "subclass"); err != nil {
		return err
	}
	if err := requireCatalogEntryID(subclass.ClassID, "subclass class"); err != nil {
		return err
	}

	foundationJSON, err := json.Marshal(subclass.FoundationFeatures)
	if err != nil {
		return fmt.Errorf("marshal subclass foundation features: %w", err)
	}
	requirementsJSON, err := json.Marshal(subclass.CreationRequirements)
	if err != nil {
		return fmt.Errorf("marshal subclass creation requirements: %w", err)
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
		ClassID:                    subclass.ClassID,
		SpellcastTrait:             subclass.SpellcastTrait,
		CreationRequirementsJson:   string(requirementsJSON),
		FoundationFeaturesJson:     string(foundationJSON),
		SpecializationFeaturesJson: string(specializationJSON),
		MasteryFeaturesJson:        string(masteryJSON),
		CreatedAt:                  sqliteutil.ToMillis(subclass.CreatedAt),
		UpdatedAt:                  sqliteutil.ToMillis(subclass.UpdatedAt),
	})
}

// GetDaggerheartSubclass retrieves a Daggerheart subclass catalog entry.
func (s *Store) GetDaggerheartSubclass(ctx context.Context, id string) (contentstore.DaggerheartSubclass, error) {
	if err := s.validateContentStore(ctx); err != nil {
		return contentstore.DaggerheartSubclass{}, err
	}
	if err := requireCatalogEntryID(id, "subclass"); err != nil {
		return contentstore.DaggerheartSubclass{}, err
	}

	row, err := s.q.GetDaggerheartSubclass(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contentstore.DaggerheartSubclass{}, storage.ErrNotFound
		}
		return contentstore.DaggerheartSubclass{}, fmt.Errorf("get daggerheart subclass: %w", err)
	}

	subclass, err := dbDaggerheartSubclassToStorage(row)
	if err != nil {
		return contentstore.DaggerheartSubclass{}, err
	}
	return subclass, nil
}

// ListDaggerheartSubclasses lists all Daggerheart subclass catalog entries.
func (s *Store) ListDaggerheartSubclasses(ctx context.Context) ([]contentstore.DaggerheartSubclass, error) {
	if err := s.validateContentStore(ctx); err != nil {
		return nil, err
	}

	rows, err := s.q.ListDaggerheartSubclasses(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart subclasses: %w", err)
	}

	subclasses := make([]contentstore.DaggerheartSubclass, 0, len(rows))
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
