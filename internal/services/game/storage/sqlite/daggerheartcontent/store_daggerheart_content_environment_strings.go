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

func (s *Store) PutDaggerheartEnvironment(ctx context.Context, env contentstore.DaggerheartEnvironment) error {
	if err := s.validateContentStore(ctx); err != nil {
		return err
	}
	if err := requireCatalogEntryID(env.ID, "environment"); err != nil {
		return err
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
		CreatedAt:                 sqliteutil.ToMillis(env.CreatedAt),
		UpdatedAt:                 sqliteutil.ToMillis(env.UpdatedAt),
	})
}

// GetDaggerheartEnvironment retrieves a Daggerheart environment catalog entry.
func (s *Store) GetDaggerheartEnvironment(ctx context.Context, id string) (contentstore.DaggerheartEnvironment, error) {
	if err := s.validateContentStore(ctx); err != nil {
		return contentstore.DaggerheartEnvironment{}, err
	}
	if err := requireCatalogEntryID(id, "environment"); err != nil {
		return contentstore.DaggerheartEnvironment{}, err
	}

	row, err := s.q.GetDaggerheartEnvironment(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contentstore.DaggerheartEnvironment{}, storage.ErrNotFound
		}
		return contentstore.DaggerheartEnvironment{}, fmt.Errorf("get daggerheart environment: %w", err)
	}

	env, err := dbDaggerheartEnvironmentToStorage(row)
	if err != nil {
		return contentstore.DaggerheartEnvironment{}, err
	}
	return env, nil
}

// ListDaggerheartEnvironments lists all Daggerheart environment catalog entries.
func (s *Store) ListDaggerheartEnvironments(ctx context.Context) ([]contentstore.DaggerheartEnvironment, error) {
	if err := s.validateContentStore(ctx); err != nil {
		return nil, err
	}

	rows, err := s.q.ListDaggerheartEnvironments(ctx)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart environments: %w", err)
	}

	envs := make([]contentstore.DaggerheartEnvironment, 0, len(rows))
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
	if err := s.validateContentStore(ctx); err != nil {
		return err
	}
	if err := requireCatalogEntryID(id, "environment"); err != nil {
		return err
	}

	return s.q.DeleteDaggerheartEnvironment(ctx, id)
}

// ListDaggerheartContentStrings returns localized content strings for content ids.
func (s *Store) ListDaggerheartContentStrings(ctx context.Context, contentType string, contentIDs []string, locale string) ([]contentstore.DaggerheartContentString, error) {
	if err := s.validateContentStore(ctx); err != nil {
		return nil, err
	}
	if err := requireContentField(contentType, "content type"); err != nil {
		return nil, err
	}
	if err := requireContentField(locale, "locale"); err != nil {
		return nil, err
	}
	if len(contentIDs) == 0 {
		return []contentstore.DaggerheartContentString{}, nil
	}

	rows, err := s.q.ListDaggerheartContentStringsByIDs(ctx, db.ListDaggerheartContentStringsByIDsParams{
		ContentType: contentType,
		Locale:      locale,
		ContentIds:  contentIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("list daggerheart content strings: %w", err)
	}
	entries := make([]contentstore.DaggerheartContentString, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, dbDaggerheartContentStringToStorage(row))
	}
	return entries, nil
}

// PutDaggerheartContentString upserts a localized content string.
func (s *Store) PutDaggerheartContentString(ctx context.Context, entry contentstore.DaggerheartContentString) error {
	if err := s.validateContentStore(ctx); err != nil {
		return err
	}
	if err := requireCatalogEntryID(entry.ContentID, "content"); err != nil {
		return err
	}
	if err := requireContentField(entry.Field, "field"); err != nil {
		return err
	}
	if err := requireContentField(entry.Locale, "locale"); err != nil {
		return err
	}
	if entry.CreatedAt.IsZero() {
		return fmt.Errorf("created at is required")
	}
	if entry.UpdatedAt.IsZero() {
		return fmt.Errorf("updated at is required")
	}

	return s.q.PutDaggerheartContentString(ctx, db.PutDaggerheartContentStringParams{
		ContentID:   entry.ContentID,
		ContentType: entry.ContentType,
		Field:       entry.Field,
		Locale:      entry.Locale,
		Text:        entry.Text,
		CreatedAt:   sqliteutil.ToMillis(entry.CreatedAt),
		UpdatedAt:   sqliteutil.ToMillis(entry.UpdatedAt),
	})
}
