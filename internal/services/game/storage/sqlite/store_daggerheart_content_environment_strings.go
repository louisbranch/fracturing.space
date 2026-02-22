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
