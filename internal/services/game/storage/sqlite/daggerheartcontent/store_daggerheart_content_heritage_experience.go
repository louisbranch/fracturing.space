package daggerheartcontent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

func (s *Store) PutDaggerheartHeritage(ctx context.Context, heritage contentstore.DaggerheartHeritage) error {
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
		CreatedAt:    sqliteutil.ToMillis(heritage.CreatedAt),
		UpdatedAt:    sqliteutil.ToMillis(heritage.UpdatedAt),
	})
}

// GetDaggerheartHeritage retrieves a Daggerheart heritage catalog entry.
func (s *Store) GetDaggerheartHeritage(ctx context.Context, id string) (contentstore.DaggerheartHeritage, error) {
	if err := ctx.Err(); err != nil {
		return contentstore.DaggerheartHeritage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return contentstore.DaggerheartHeritage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return contentstore.DaggerheartHeritage{}, fmt.Errorf("heritage id is required")
	}

	row, err := s.q.GetDaggerheartHeritage(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contentstore.DaggerheartHeritage{}, storage.ErrNotFound
		}
		return contentstore.DaggerheartHeritage{}, fmt.Errorf("get daggerheart heritage: %w", err)
	}

	heritage, err := dbDaggerheartHeritageToStorage(row)
	if err != nil {
		return contentstore.DaggerheartHeritage{}, err
	}
	return heritage, nil
}

// ListDaggerheartHeritages lists all Daggerheart heritage catalog entries.
func (s *Store) ListDaggerheartHeritages(ctx context.Context) ([]contentstore.DaggerheartHeritage, error) {
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

	heritages := make([]contentstore.DaggerheartHeritage, 0, len(rows))
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
func (s *Store) PutDaggerheartExperience(ctx context.Context, experience contentstore.DaggerheartExperienceEntry) error {
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
		CreatedAt:   sqliteutil.ToMillis(experience.CreatedAt),
		UpdatedAt:   sqliteutil.ToMillis(experience.UpdatedAt),
	})
}

// GetDaggerheartExperience retrieves a Daggerheart experience catalog entry.
func (s *Store) GetDaggerheartExperience(ctx context.Context, id string) (contentstore.DaggerheartExperienceEntry, error) {
	if err := ctx.Err(); err != nil {
		return contentstore.DaggerheartExperienceEntry{}, err
	}
	if s == nil || s.sqlDB == nil {
		return contentstore.DaggerheartExperienceEntry{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return contentstore.DaggerheartExperienceEntry{}, fmt.Errorf("experience id is required")
	}

	row, err := s.q.GetDaggerheartExperience(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contentstore.DaggerheartExperienceEntry{}, storage.ErrNotFound
		}
		return contentstore.DaggerheartExperienceEntry{}, fmt.Errorf("get daggerheart experience: %w", err)
	}

	return dbDaggerheartExperienceToStorage(row), nil
}

// ListDaggerheartExperiences lists all Daggerheart experience catalog entries.
func (s *Store) ListDaggerheartExperiences(ctx context.Context) ([]contentstore.DaggerheartExperienceEntry, error) {
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

	experiences := make([]contentstore.DaggerheartExperienceEntry, 0, len(rows))
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
