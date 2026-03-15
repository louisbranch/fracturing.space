package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/social/storage"
	socialusername "github.com/louisbranch/fracturing.space/internal/services/social/username"
)

// PutDirectoryUser upserts one auth-synced user directory record.
func (s *Store) PutDirectoryUser(ctx context.Context, user storage.DirectoryUser) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	userID := strings.TrimSpace(user.UserID)
	if userID == "" {
		return fmt.Errorf("user id is required")
	}
	username, err := socialusername.Canonicalize(user.Username)
	if err != nil {
		return fmt.Errorf("username is invalid: %w", err)
	}
	createdAt := user.CreatedAt.UTC()
	updatedAt := user.UpdatedAt.UTC()
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

	_, err = s.sqlDB.ExecContext(
		ctx,
		`INSERT INTO user_directory (user_id, username, created_at, updated_at)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(user_id) DO UPDATE SET
		   username = excluded.username,
		   updated_at = excluded.updated_at
		 WHERE user_directory.username <> excluded.username`,
		userID,
		username,
		sqliteutil.ToMillis(createdAt),
		sqliteutil.ToMillis(updatedAt),
	)
	if err != nil {
		return fmt.Errorf("put directory user: %w", err)
	}
	return nil
}

// SearchUsers returns ranked user directory matches for invite search.
func (s *Store) SearchUsers(ctx context.Context, viewerUserID string, query storage.SearchUsersQuery, limit int) ([]storage.SearchUser, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	viewerUserID = strings.TrimSpace(viewerUserID)
	if viewerUserID == "" {
		return nil, fmt.Errorf("viewer user id is required")
	}
	query.Username = strings.TrimSpace(query.Username)
	query.Name = strings.TrimSpace(query.Name)
	if query.Username == "" && query.Name == "" {
		return []storage.SearchUser{}, nil
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}

	namePrefix := query.Name + "%"
	usernamePrefix := query.Username + "%"
	rows, err := s.sqlDB.QueryContext(
		ctx,
		`SELECT d.user_id,
		        d.username,
		        p.name,
		        p.avatar_set_id,
		        p.avatar_asset_id,
		        CASE WHEN c.contact_user_id IS NULL THEN 0 ELSE 1 END AS is_contact
		   FROM user_directory d
		   LEFT JOIN user_profiles p
		     ON p.user_id = d.user_id
		   LEFT JOIN contacts c
		     ON c.owner_user_id = ? AND c.contact_user_id = d.user_id
		  WHERE (? <> '' AND d.username LIKE ?)
		     OR (? <> '' AND lower(COALESCE(p.name, '')) LIKE ?)
		  ORDER BY is_contact DESC,
		           CASE WHEN ? <> '' AND d.username = ? THEN 0 ELSE 1 END ASC,
		           CASE WHEN ? <> '' AND d.username LIKE ? THEN 0 ELSE 1 END ASC,
		           d.username ASC
		  LIMIT ?`,
		viewerUserID,
		query.Username,
		usernamePrefix,
		query.Name,
		namePrefix,
		query.Username,
		query.Username,
		query.Username,
		usernamePrefix,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	defer rows.Close()

	results := make([]storage.SearchUser, 0, limit)
	for rows.Next() {
		var (
			record        storage.SearchUser
			name          sql.NullString
			avatarSetID   sql.NullString
			avatarAssetID sql.NullString
			isContact     int
		)
		if err := rows.Scan(
			&record.UserID,
			&record.Username,
			&name,
			&avatarSetID,
			&avatarAssetID,
			&isContact,
		); err != nil {
			return nil, fmt.Errorf("search users: %w", err)
		}
		record.UserID = strings.TrimSpace(record.UserID)
		record.Username = strings.TrimSpace(record.Username)
		record.Name = strings.TrimSpace(name.String)
		record.AvatarSetID = strings.TrimSpace(avatarSetID.String)
		record.AvatarAssetID = strings.TrimSpace(avatarAssetID.String)
		record.IsContact = isContact == 1
		results = append(results, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("search users: %w", err)
	}
	return results, nil
}
