package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

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
