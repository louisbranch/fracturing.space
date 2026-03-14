package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/social/storage"
)

// PutUserProfile upserts one social/discovery user profile record.
func (s *Store) PutUserProfile(ctx context.Context, profile storage.UserProfile) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	userID := strings.TrimSpace(profile.UserID)
	if userID == "" {
		return fmt.Errorf("user id is required")
	}
	name := strings.TrimSpace(profile.Name)
	avatarSetID := strings.TrimSpace(profile.AvatarSetID)
	avatarAssetID := strings.TrimSpace(profile.AvatarAssetID)
	bio := strings.TrimSpace(profile.Bio)
	pronouns := strings.TrimSpace(profile.Pronouns)
	createdAt := profile.CreatedAt.UTC()
	updatedAt := profile.UpdatedAt.UTC()
	if createdAt.IsZero() && updatedAt.IsZero() {
		createdAt = time.Now().UTC()
		updatedAt = createdAt
	} else {
		if createdAt.IsZero() {
			createdAt = updatedAt
		}
		if updatedAt.IsZero() {
			updatedAt = createdAt
		}
	}

	_, err := s.sqlDB.ExecContext(
		ctx,
		`INSERT INTO user_profiles (user_id, name, avatar_set_id, avatar_asset_id, bio, pronouns, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET
		   name = excluded.name,
		   avatar_set_id = excluded.avatar_set_id,
		   avatar_asset_id = excluded.avatar_asset_id,
		   bio = excluded.bio,
		   pronouns = excluded.pronouns,
		   updated_at = excluded.updated_at
		 WHERE user_profiles.name <> excluded.name
		    OR user_profiles.avatar_set_id <> excluded.avatar_set_id
		    OR user_profiles.avatar_asset_id <> excluded.avatar_asset_id
		    OR user_profiles.bio <> excluded.bio
		    OR user_profiles.pronouns <> excluded.pronouns`,
		userID,
		name,
		avatarSetID,
		avatarAssetID,
		bio,
		pronouns,
		toMillis(createdAt),
		toMillis(updatedAt),
	)
	if err != nil {
		return fmt.Errorf("put user profile: %w", err)
	}
	return nil
}

// GetUserProfileByUserID returns one social/discovery profile by owner user ID.
func (s *Store) GetUserProfileByUserID(ctx context.Context, userID string) (storage.UserProfile, error) {
	if err := ctx.Err(); err != nil {
		return storage.UserProfile{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.UserProfile{}, fmt.Errorf("storage is not configured")
	}
	userID = strings.TrimSpace(userID)
	if userID == "" {
		return storage.UserProfile{}, fmt.Errorf("user id is required")
	}

	row := s.sqlDB.QueryRowContext(
		ctx,
		`SELECT user_id, name, avatar_set_id, avatar_asset_id, bio, pronouns, created_at, updated_at
		 FROM user_profiles
		 WHERE user_id = ?`,
		userID,
	)
	var record storage.UserProfile
	var createdAt int64
	var updatedAt int64
	err := row.Scan(
		&record.UserID,
		&record.Name,
		&record.AvatarSetID,
		&record.AvatarAssetID,
		&record.Bio,
		&record.Pronouns,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.UserProfile{}, storage.ErrNotFound
		}
		return storage.UserProfile{}, fmt.Errorf("get user profile by user id: %w", err)
	}
	record.CreatedAt = fromMillis(createdAt)
	record.UpdatedAt = fromMillis(updatedAt)
	return record, nil
}
