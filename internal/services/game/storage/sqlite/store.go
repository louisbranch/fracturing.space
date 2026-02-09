// Package sqlite provides a SQLite-backed implementation of storage interfaces.
package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	apperrors "github.com/louisbranch/fracturing.space/internal/platform/errors"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/snapshot"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/core/encoding"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/migrations"
	sqlite "modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

func toMillis(value time.Time) int64 {
	return value.UTC().UnixMilli()
}

func fromMillis(value int64) time.Time {
	return time.UnixMilli(value).UTC()
}

func toNullMillis(value *time.Time) sql.NullInt64 {
	if value == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: toMillis(*value), Valid: true}
}

func fromNullMillis(value sql.NullInt64) *time.Time {
	if !value.Valid {
		return nil
	}
	t := fromMillis(value.Int64)
	return &t
}

// Store provides a SQLite-backed store implementing all storage interfaces.
type Store struct {
	sqlDB *sql.DB
	q     *db.Queries
}

// Open opens a SQLite store at the provided path.
func Open(path string) (*Store, error) {
	if strings.TrimSpace(path) == "" {
		return nil, fmt.Errorf("storage path is required")
	}

	cleanPath := filepath.Clean(path)
	dsn := cleanPath + "?_journal_mode=WAL&_foreign_keys=ON&_busy_timeout=5000&_synchronous=NORMAL"
	sqlDB, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite db: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping sqlite db: %w", err)
	}

	store := &Store{
		sqlDB: sqlDB,
		q:     db.New(sqlDB),
	}

	if err := store.runMigrations(); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	return store, nil
}

// Close closes the underlying SQLite database.
func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

// runMigrations runs embedded SQL migrations.
func (s *Store) runMigrations() error {
	entries, err := fs.ReadDir(migrations.FS, ".")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var sqlFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".sql") {
			sqlFiles = append(sqlFiles, entry.Name())
		}
	}
	sort.Strings(sqlFiles)

	for _, file := range sqlFiles {
		content, err := fs.ReadFile(migrations.FS, file)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}

		upSQL := extractUpMigration(string(content))
		if upSQL == "" {
			continue
		}

		if _, err := s.sqlDB.Exec(upSQL); err != nil {
			if !isAlreadyExistsError(err) {
				return fmt.Errorf("exec migration %s: %w", file, err)
			}
		}
	}

	return nil
}

// extractUpMigration extracts the Up migration portion from a migration file.
func extractUpMigration(content string) string {
	upIdx := strings.Index(content, "-- +migrate Up")
	if upIdx == -1 {
		return content
	}
	downIdx := strings.Index(content, "-- +migrate Down")
	if downIdx == -1 {
		return content[upIdx+len("-- +migrate Up"):]
	}
	return content[upIdx+len("-- +migrate Up") : downIdx]
}

// isAlreadyExistsError checks if the error is a table/index already exists error.
func isAlreadyExistsError(err error) bool {
	return strings.Contains(err.Error(), "already exists")
}

// Campaign methods

// Put persists a campaign record.
func (s *Store) Put(ctx context.Context, c campaign.Campaign) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(c.ID) == "" {
		return fmt.Errorf("campaign id is required")
	}

	completedAt := toNullMillis(c.CompletedAt)
	archivedAt := toNullMillis(c.ArchivedAt)

	return s.q.PutCampaign(ctx, db.PutCampaignParams{
		ID:               c.ID,
		Name:             c.Name,
		GameSystem:       gameSystemToString(c.System),
		Status:           campaignStatusToString(c.Status),
		GmMode:           gmModeToString(c.GmMode),
		ParticipantCount: int64(c.ParticipantCount),
		CharacterCount:   int64(c.CharacterCount),
		ThemePrompt:      c.ThemePrompt,
		CreatedAt:        toMillis(c.CreatedAt),
		LastActivityAt:   toMillis(c.LastActivityAt),
		UpdatedAt:        toMillis(c.UpdatedAt),
		CompletedAt:      completedAt,
		ArchivedAt:       archivedAt,
	})
}

// Get fetches a campaign record by ID.
func (s *Store) Get(ctx context.Context, id string) (campaign.Campaign, error) {
	if err := ctx.Err(); err != nil {
		return campaign.Campaign{}, err
	}
	if s == nil || s.sqlDB == nil {
		return campaign.Campaign{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return campaign.Campaign{}, fmt.Errorf("campaign id is required")
	}

	row, err := s.q.GetCampaign(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return campaign.Campaign{}, storage.ErrNotFound
		}
		return campaign.Campaign{}, fmt.Errorf("get campaign: %w", err)
	}

	return dbGetCampaignRowToDomain(row)
}

// List returns a page of campaign records ordered by storage key.
func (s *Store) List(ctx context.Context, pageSize int, pageToken string) (storage.CampaignPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.CampaignPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.CampaignPage{}, fmt.Errorf("storage is not configured")
	}
	if pageSize <= 0 {
		return storage.CampaignPage{}, fmt.Errorf("page size must be greater than zero")
	}

	page := storage.CampaignPage{
		Campaigns: make([]campaign.Campaign, 0, pageSize),
	}

	if pageToken == "" {
		rows, err := s.q.ListAllCampaigns(ctx, int64(pageSize+1))
		if err != nil {
			return storage.CampaignPage{}, fmt.Errorf("list campaigns: %w", err)
		}
		for i, row := range rows {
			if i >= pageSize {
				page.NextPageToken = rows[pageSize-1].ID
				break
			}
			c, err := dbListAllCampaignsRowToDomain(row)
			if err != nil {
				return storage.CampaignPage{}, err
			}
			page.Campaigns = append(page.Campaigns, c)
		}
	} else {
		rows, err := s.q.ListCampaigns(ctx, db.ListCampaignsParams{
			ID:    pageToken,
			Limit: int64(pageSize + 1),
		})
		if err != nil {
			return storage.CampaignPage{}, fmt.Errorf("list campaigns: %w", err)
		}
		for i, row := range rows {
			if i >= pageSize {
				page.NextPageToken = rows[pageSize-1].ID
				break
			}
			c, err := dbListCampaignsRowToDomain(row)
			if err != nil {
				return storage.CampaignPage{}, err
			}
			page.Campaigns = append(page.Campaigns, c)
		}
	}

	return page, nil
}

// Participant methods

// PutParticipant persists a participant record.
func (s *Store) PutParticipant(ctx context.Context, p participant.Participant) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(p.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(p.ID) == "" {
		return fmt.Errorf("participant id is required")
	}

	if err := s.q.PutParticipant(ctx, db.PutParticipantParams{
		CampaignID:  p.CampaignID,
		ID:          p.ID,
		UserID:      p.UserID,
		DisplayName: p.DisplayName,
		Role:        participantRoleToString(p.Role),
		Controller:  participantControllerToString(p.Controller),
		IsOwner:     boolToInt(p.IsOwner),
		CreatedAt:   toMillis(p.CreatedAt),
		UpdatedAt:   toMillis(p.UpdatedAt),
	}); err != nil {
		if isParticipantUserConflict(err) {
			return apperrors.WithMetadata(
				apperrors.CodeParticipantUserAlreadyClaimed,
				"participant user already claimed",
				map[string]string{
					"CampaignID": p.CampaignID,
					"UserID":     p.UserID,
				},
			)
		}
		return err
	}
	return nil
}

// DeleteParticipant deletes a participant record by IDs.
func (s *Store) DeleteParticipant(ctx context.Context, campaignID, participantID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(participantID) == "" {
		return fmt.Errorf("participant id is required")
	}

	return s.q.DeleteParticipant(ctx, db.DeleteParticipantParams{
		CampaignID: campaignID,
		ID:         participantID,
	})
}

// GetParticipant fetches a participant record by IDs.
func (s *Store) GetParticipant(ctx context.Context, campaignID, participantID string) (participant.Participant, error) {
	if err := ctx.Err(); err != nil {
		return participant.Participant{}, err
	}
	if s == nil || s.sqlDB == nil {
		return participant.Participant{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return participant.Participant{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(participantID) == "" {
		return participant.Participant{}, fmt.Errorf("participant id is required")
	}

	row, err := s.q.GetParticipant(ctx, db.GetParticipantParams{
		CampaignID: campaignID,
		ID:         participantID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return participant.Participant{}, storage.ErrNotFound
		}
		return participant.Participant{}, fmt.Errorf("get participant: %w", err)
	}

	return dbParticipantToDomain(row)
}

// ListParticipantsByCampaign returns all participants for a campaign.
func (s *Store) ListParticipantsByCampaign(ctx context.Context, campaignID string) ([]participant.Participant, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}

	rows, err := s.q.ListParticipantsByCampaign(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list participants: %w", err)
	}

	participants := make([]participant.Participant, 0, len(rows))
	for _, row := range rows {
		p, err := dbParticipantToDomain(row)
		if err != nil {
			return nil, err
		}
		participants = append(participants, p)
	}

	return participants, nil
}

// ListParticipants returns a page of participant records.
func (s *Store) ListParticipants(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.ParticipantPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.ParticipantPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ParticipantPage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.ParticipantPage{}, fmt.Errorf("campaign id is required")
	}
	if pageSize <= 0 {
		return storage.ParticipantPage{}, fmt.Errorf("page size must be greater than zero")
	}

	var rows []db.Participant
	var err error

	if pageToken == "" {
		rows, err = s.q.ListParticipantsByCampaignPagedFirst(ctx, db.ListParticipantsByCampaignPagedFirstParams{
			CampaignID: campaignID,
			Limit:      int64(pageSize + 1),
		})
	} else {
		rows, err = s.q.ListParticipantsByCampaignPaged(ctx, db.ListParticipantsByCampaignPagedParams{
			CampaignID: campaignID,
			ID:         pageToken,
			Limit:      int64(pageSize + 1),
		})
	}
	if err != nil {
		return storage.ParticipantPage{}, fmt.Errorf("list participants: %w", err)
	}

	page := storage.ParticipantPage{
		Participants: make([]participant.Participant, 0, pageSize),
	}

	for i, row := range rows {
		if i >= pageSize {
			page.NextPageToken = rows[pageSize-1].ID
			break
		}
		p, err := dbParticipantToDomain(row)
		if err != nil {
			return storage.ParticipantPage{}, err
		}
		page.Participants = append(page.Participants, p)
	}

	return page, nil
}

// Invite methods

// PutInvite persists an invite record.
func (s *Store) PutInvite(ctx context.Context, inv invite.Invite) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(inv.ID) == "" {
		return fmt.Errorf("invite id is required")
	}
	if strings.TrimSpace(inv.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(inv.ParticipantID) == "" {
		return fmt.Errorf("participant id is required")
	}

	return s.q.PutInvite(ctx, db.PutInviteParams{
		ID:                     inv.ID,
		CampaignID:             inv.CampaignID,
		ParticipantID:          inv.ParticipantID,
		Status:                 invite.StatusLabel(inv.Status),
		CreatedByParticipantID: inv.CreatedByParticipantID,
		CreatedAt:              toMillis(inv.CreatedAt),
		UpdatedAt:              toMillis(inv.UpdatedAt),
	})
}

// GetInvite fetches an invite record by ID.
func (s *Store) GetInvite(ctx context.Context, inviteID string) (invite.Invite, error) {
	if err := ctx.Err(); err != nil {
		return invite.Invite{}, err
	}
	if s == nil || s.sqlDB == nil {
		return invite.Invite{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(inviteID) == "" {
		return invite.Invite{}, fmt.Errorf("invite id is required")
	}

	row, err := s.q.GetInvite(ctx, inviteID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return invite.Invite{}, storage.ErrNotFound
		}
		return invite.Invite{}, fmt.Errorf("get invite: %w", err)
	}

	return dbInviteToDomain(row)
}

// ListInvites returns a page of invite records for a campaign.
func (s *Store) ListInvites(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.InvitePage, error) {
	if err := ctx.Err(); err != nil {
		return storage.InvitePage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.InvitePage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.InvitePage{}, fmt.Errorf("campaign id is required")
	}
	if pageSize <= 0 {
		return storage.InvitePage{}, fmt.Errorf("page size must be greater than zero")
	}

	var rows []db.Invite
	var err error
	if pageToken == "" {
		rows, err = s.q.ListInvitesByCampaignPagedFirst(ctx, db.ListInvitesByCampaignPagedFirstParams{
			CampaignID: campaignID,
			Limit:      int64(pageSize + 1),
		})
	} else {
		rows, err = s.q.ListInvitesByCampaignPaged(ctx, db.ListInvitesByCampaignPagedParams{
			CampaignID: campaignID,
			ID:         pageToken,
			Limit:      int64(pageSize + 1),
		})
	}
	if err != nil {
		return storage.InvitePage{}, fmt.Errorf("list invites: %w", err)
	}

	page := storage.InvitePage{Invites: make([]invite.Invite, 0, pageSize)}
	for i, row := range rows {
		if i >= pageSize {
			page.NextPageToken = rows[pageSize-1].ID
			break
		}
		inv, err := dbInviteToDomain(row)
		if err != nil {
			return storage.InvitePage{}, err
		}
		page.Invites = append(page.Invites, inv)
	}

	return page, nil
}

// UpdateInviteStatus updates the status for an invite.
func (s *Store) UpdateInviteStatus(ctx context.Context, inviteID string, status invite.Status, updatedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(inviteID) == "" {
		return fmt.Errorf("invite id is required")
	}
	if updatedAt.IsZero() {
		updatedAt = time.Now().UTC()
	}

	return s.q.UpdateInviteStatus(ctx, db.UpdateInviteStatusParams{
		Status:    invite.StatusLabel(status),
		UpdatedAt: toMillis(updatedAt),
		ID:        inviteID,
	})
}

// Character methods

// PutCharacter persists a character record.
func (s *Store) PutCharacter(ctx context.Context, c character.Character) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(c.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(c.ID) == "" {
		return fmt.Errorf("character id is required")
	}

	return s.q.PutCharacter(ctx, db.PutCharacterParams{
		CampaignID: c.CampaignID,
		ID:         c.ID,
		Name:       c.Name,
		Kind:       characterKindToString(c.Kind),
		Notes:      c.Notes,
		CreatedAt:  toMillis(c.CreatedAt),
		UpdatedAt:  toMillis(c.UpdatedAt),
	})
}

// DeleteCharacter deletes a character record by IDs.
func (s *Store) DeleteCharacter(ctx context.Context, campaignID, characterID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(characterID) == "" {
		return fmt.Errorf("character id is required")
	}

	return s.q.DeleteCharacter(ctx, db.DeleteCharacterParams{
		CampaignID: campaignID,
		ID:         characterID,
	})
}

// GetCharacter fetches a character record by IDs.
func (s *Store) GetCharacter(ctx context.Context, campaignID, characterID string) (character.Character, error) {
	if err := ctx.Err(); err != nil {
		return character.Character{}, err
	}
	if s == nil || s.sqlDB == nil {
		return character.Character{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return character.Character{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(characterID) == "" {
		return character.Character{}, fmt.Errorf("character id is required")
	}

	row, err := s.q.GetCharacter(ctx, db.GetCharacterParams{
		CampaignID: campaignID,
		ID:         characterID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return character.Character{}, storage.ErrNotFound
		}
		return character.Character{}, fmt.Errorf("get character: %w", err)
	}

	return dbCharacterToDomain(row)
}

// ListCharacters returns a page of character records.
func (s *Store) ListCharacters(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.CharacterPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.CharacterPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.CharacterPage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.CharacterPage{}, fmt.Errorf("campaign id is required")
	}
	if pageSize <= 0 {
		return storage.CharacterPage{}, fmt.Errorf("page size must be greater than zero")
	}

	var rows []db.Character
	var err error

	if pageToken == "" {
		rows, err = s.q.ListCharactersByCampaignPagedFirst(ctx, db.ListCharactersByCampaignPagedFirstParams{
			CampaignID: campaignID,
			Limit:      int64(pageSize + 1),
		})
	} else {
		rows, err = s.q.ListCharactersByCampaignPaged(ctx, db.ListCharactersByCampaignPagedParams{
			CampaignID: campaignID,
			ID:         pageToken,
			Limit:      int64(pageSize + 1),
		})
	}
	if err != nil {
		return storage.CharacterPage{}, fmt.Errorf("list characters: %w", err)
	}

	page := storage.CharacterPage{
		Characters: make([]character.Character, 0, pageSize),
	}

	for i, row := range rows {
		if i >= pageSize {
			page.NextPageToken = rows[pageSize-1].ID
			break
		}
		c, err := dbCharacterToDomain(row)
		if err != nil {
			return storage.CharacterPage{}, err
		}
		page.Characters = append(page.Characters, c)
	}

	return page, nil
}

// Control Default methods

// PutControlDefault sets the default controller for a character.
func (s *Store) PutControlDefault(ctx context.Context, campaignID, characterID string, controller character.CharacterController) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(characterID) == "" {
		return fmt.Errorf("character id is required")
	}

	isGM := int64(0)
	if controller.IsGM {
		isGM = 1
	}

	return s.q.PutControlDefault(ctx, db.PutControlDefaultParams{
		CampaignID:    campaignID,
		CharacterID:   characterID,
		IsGm:          isGM,
		ParticipantID: controller.ParticipantID,
	})
}

// GetControlDefault retrieves the default controller for a character.
func (s *Store) GetControlDefault(ctx context.Context, campaignID, characterID string) (character.CharacterController, error) {
	if err := ctx.Err(); err != nil {
		return character.CharacterController{}, err
	}
	if s == nil || s.sqlDB == nil {
		return character.CharacterController{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return character.CharacterController{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(characterID) == "" {
		return character.CharacterController{}, fmt.Errorf("character id is required")
	}

	row, err := s.q.GetControlDefault(ctx, db.GetControlDefaultParams{
		CampaignID:  campaignID,
		CharacterID: characterID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return character.CharacterController{}, storage.ErrNotFound
		}
		return character.CharacterController{}, fmt.Errorf("get control default: %w", err)
	}

	return character.CharacterController{
		IsGM:          row.IsGm != 0,
		ParticipantID: row.ParticipantID,
	}, nil
}

// Session methods

// PutSession atomically stores a session and sets it as the active session for the campaign.
func (s *Store) PutSession(ctx context.Context, sess session.Session) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(sess.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sess.ID) == "" {
		return fmt.Errorf("session id is required")
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)

	if sess.Status == session.SessionStatusActive {
		hasActive, err := qtx.HasActiveSession(ctx, sess.CampaignID)
		if err != nil {
			return fmt.Errorf("check active session: %w", err)
		}
		if hasActive != 0 {
			return storage.ErrActiveSessionExists
		}
	}

	endedAt := toNullMillis(sess.EndedAt)

	if err := qtx.PutSession(ctx, db.PutSessionParams{
		CampaignID: sess.CampaignID,
		ID:         sess.ID,
		Name:       sess.Name,
		Status:     sessionStatusToString(sess.Status),
		StartedAt:  toMillis(sess.StartedAt),
		UpdatedAt:  toMillis(sess.UpdatedAt),
		EndedAt:    endedAt,
	}); err != nil {
		return fmt.Errorf("put session: %w", err)
	}

	if sess.Status == session.SessionStatusActive {
		if err := qtx.SetActiveSession(ctx, db.SetActiveSessionParams{
			CampaignID: sess.CampaignID,
			SessionID:  sess.ID,
		}); err != nil {
			return fmt.Errorf("set active session: %w", err)
		}
	}

	return tx.Commit()
}

// EndSession marks a session as ended and clears it as active for the campaign.
func (s *Store) EndSession(ctx context.Context, campaignID, sessionID string, endedAt time.Time) (session.Session, bool, error) {
	if err := ctx.Err(); err != nil {
		return session.Session{}, false, err
	}
	if s == nil || s.sqlDB == nil {
		return session.Session{}, false, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return session.Session{}, false, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return session.Session{}, false, fmt.Errorf("session id is required")
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return session.Session{}, false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)

	row, err := qtx.GetSession(ctx, db.GetSessionParams{
		CampaignID: campaignID,
		ID:         sessionID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return session.Session{}, false, storage.ErrNotFound
		}
		return session.Session{}, false, fmt.Errorf("get session: %w", err)
	}

	sess, err := dbSessionToDomain(row)
	if err != nil {
		return session.Session{}, false, err
	}

	transitioned := false
	if sess.Status == session.SessionStatusActive {
		transitioned = true
		sess.Status = session.SessionStatusEnded
		sess.UpdatedAt = endedAt.UTC()
		sess.EndedAt = &sess.UpdatedAt

		if err := qtx.UpdateSessionStatus(ctx, db.UpdateSessionStatusParams{
			Status:     sessionStatusToString(sess.Status),
			UpdatedAt:  toMillis(sess.UpdatedAt),
			EndedAt:    toNullMillis(sess.EndedAt),
			CampaignID: campaignID,
			ID:         sessionID,
		}); err != nil {
			return session.Session{}, false, fmt.Errorf("update session status: %w", err)
		}
	}

	if err := qtx.ClearActiveSession(ctx, campaignID); err != nil {
		return session.Session{}, false, fmt.Errorf("clear active session: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return session.Session{}, false, fmt.Errorf("commit: %w", err)
	}

	return sess, transitioned, nil
}

// GetSession retrieves a session by campaign ID and session ID.
func (s *Store) GetSession(ctx context.Context, campaignID, sessionID string) (session.Session, error) {
	if err := ctx.Err(); err != nil {
		return session.Session{}, err
	}
	if s == nil || s.sqlDB == nil {
		return session.Session{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return session.Session{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return session.Session{}, fmt.Errorf("session id is required")
	}

	row, err := s.q.GetSession(ctx, db.GetSessionParams{
		CampaignID: campaignID,
		ID:         sessionID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return session.Session{}, storage.ErrNotFound
		}
		return session.Session{}, fmt.Errorf("get session: %w", err)
	}

	return dbSessionToDomain(row)
}

// GetActiveSession retrieves the active session for a campaign.
func (s *Store) GetActiveSession(ctx context.Context, campaignID string) (session.Session, error) {
	if err := ctx.Err(); err != nil {
		return session.Session{}, err
	}
	if s == nil || s.sqlDB == nil {
		return session.Session{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return session.Session{}, fmt.Errorf("campaign id is required")
	}

	row, err := s.q.GetActiveSession(ctx, campaignID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return session.Session{}, storage.ErrNotFound
		}
		return session.Session{}, fmt.Errorf("get active session: %w", err)
	}

	return dbSessionToDomain(row)
}

// ListSessions returns a page of session records.
func (s *Store) ListSessions(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.SessionPage, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionPage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionPage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionPage{}, fmt.Errorf("campaign id is required")
	}
	if pageSize <= 0 {
		return storage.SessionPage{}, fmt.Errorf("page size must be greater than zero")
	}

	var rows []db.Session
	var err error

	if pageToken == "" {
		rows, err = s.q.ListSessionsByCampaignPagedFirst(ctx, db.ListSessionsByCampaignPagedFirstParams{
			CampaignID: campaignID,
			Limit:      int64(pageSize + 1),
		})
	} else {
		rows, err = s.q.ListSessionsByCampaignPaged(ctx, db.ListSessionsByCampaignPagedParams{
			CampaignID: campaignID,
			ID:         pageToken,
			Limit:      int64(pageSize + 1),
		})
	}
	if err != nil {
		return storage.SessionPage{}, fmt.Errorf("list sessions: %w", err)
	}

	page := storage.SessionPage{
		Sessions: make([]session.Session, 0, pageSize),
	}

	for i, row := range rows {
		if i >= pageSize {
			page.NextPageToken = rows[pageSize-1].ID
			break
		}
		sess, err := dbSessionToDomain(row)
		if err != nil {
			return storage.SessionPage{}, err
		}
		page.Sessions = append(page.Sessions, sess)
	}

	return page, nil
}

// Roll Outcome methods

// ApplyRollOutcome atomically applies a roll outcome and appends the applied event.
func (s *Store) ApplyRollOutcome(ctx context.Context, input storage.RollOutcomeApplyInput) (storage.RollOutcomeApplyResult, error) {
	if err := ctx.Err(); err != nil {
		return storage.RollOutcomeApplyResult{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.RollOutcomeApplyResult{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(input.CampaignID) == "" {
		return storage.RollOutcomeApplyResult{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(input.SessionID) == "" {
		return storage.RollOutcomeApplyResult{}, fmt.Errorf("session id is required")
	}
	if input.RollSeq == 0 {
		return storage.RollOutcomeApplyResult{}, fmt.Errorf("roll seq is required")
	}
	if len(input.Targets) == 0 {
		return storage.RollOutcomeApplyResult{}, fmt.Errorf("targets are required")
	}

	targetSet := make(map[string]struct{}, len(input.Targets))
	for _, target := range input.Targets {
		if strings.TrimSpace(target) == "" {
			return storage.RollOutcomeApplyResult{}, fmt.Errorf("target id is required")
		}
		targetSet[target] = struct{}{}
	}

	perTargetDelta := make(map[string]storage.RollOutcomeDelta, len(input.CharacterDeltas))
	for _, delta := range input.CharacterDeltas {
		if _, ok := targetSet[delta.CharacterID]; !ok {
			return storage.RollOutcomeApplyResult{}, fmt.Errorf("character delta target mismatch")
		}
		current := perTargetDelta[delta.CharacterID]
		current.CharacterID = delta.CharacterID
		current.HopeDelta += delta.HopeDelta
		current.StressDelta += delta.StressDelta
		perTargetDelta[delta.CharacterID] = current
	}

	result := storage.RollOutcomeApplyResult{
		UpdatedCharacterStates: make([]storage.DaggerheartCharacterState, 0, len(input.Targets)),
		AppliedChanges:         make([]session.OutcomeAppliedChange, 0),
	}

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return storage.RollOutcomeApplyResult{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)

	evtTimestamp := input.EventTimestamp
	if evtTimestamp.IsZero() {
		evtTimestamp = time.Now().UTC()
	}

	applied, err := qtx.CheckOutcomeApplied(ctx, db.CheckOutcomeAppliedParams{
		CampaignID: input.CampaignID,
		SessionID:  input.SessionID,
		RequestID:  input.RequestID,
	})
	if err != nil {
		return storage.RollOutcomeApplyResult{}, fmt.Errorf("check outcome applied: %w", err)
	}
	if applied != 0 {
		return storage.RollOutcomeApplyResult{}, session.ErrOutcomeAlreadyApplied
	}

	if input.GMFearDelta != 0 {
		if input.GMFearDelta < 0 {
			return storage.RollOutcomeApplyResult{}, session.ErrOutcomeGMFearInvalid
		}

		var fear snapshot.GmFear
		row, err := qtx.GetDaggerheartSnapshot(ctx, input.CampaignID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				fear = snapshot.GmFear{CampaignID: input.CampaignID, Value: 0}
			} else {
				return storage.RollOutcomeApplyResult{}, fmt.Errorf("get daggerheart snapshot: %w", err)
			}
		} else {
			fear = snapshot.GmFear{CampaignID: row.CampaignID, Value: int(row.GmFear)}
		}

		updated, before, after, err := snapshot.ApplyGmFearGain(fear, input.GMFearDelta)
		if err != nil {
			return storage.RollOutcomeApplyResult{}, session.ErrOutcomeGMFearInvalid
		}

		if err := qtx.PutDaggerheartSnapshot(ctx, db.PutDaggerheartSnapshotParams{
			CampaignID: updated.CampaignID,
			GmFear:     int64(updated.Value),
		}); err != nil {
			return storage.RollOutcomeApplyResult{}, fmt.Errorf("put daggerheart snapshot: %w", err)
		}

		result.GMFearChanged = true
		result.GMFearBefore = before
		result.GMFearAfter = after
		result.AppliedChanges = append(result.AppliedChanges, session.OutcomeAppliedChange{
			Field:  session.OutcomeFieldGMFear,
			Before: before,
			After:  after,
		})

		payloadJSON, err := json.Marshal(event.GMFearChangedPayload{
			Before: before,
			After:  after,
		})
		if err != nil {
			return storage.RollOutcomeApplyResult{}, fmt.Errorf("marshal gm fear payload: %w", err)
		}
		if _, err := appendEventTx(ctx, qtx, event.Event{
			CampaignID:  input.CampaignID,
			Timestamp:   evtTimestamp,
			Type:        event.TypeGMFearChanged,
			SessionID:   input.SessionID,
			RequestID:   input.RequestID,
			ActorType:   event.ActorTypeSystem,
			EntityType:  "snapshot",
			EntityID:    input.CampaignID,
			PayloadJSON: payloadJSON,
		}); err != nil {
			return storage.RollOutcomeApplyResult{}, fmt.Errorf("append gm fear event: %w", err)
		}
	}

	for _, target := range input.Targets {
		// Get Daggerheart-specific state (HP, Hope, Stress)
		dhStateRow, err := qtx.GetDaggerheartCharacterState(ctx, db.GetDaggerheartCharacterStateParams{
			CampaignID:  input.CampaignID,
			CharacterID: target,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return storage.RollOutcomeApplyResult{}, session.ErrOutcomeCharacterNotFound
			}
			return storage.RollOutcomeApplyResult{}, fmt.Errorf("get daggerheart character state: %w", err)
		}

		dhProfileRow, err := qtx.GetDaggerheartCharacterProfile(ctx, db.GetDaggerheartCharacterProfileParams{
			CampaignID:  input.CampaignID,
			CharacterID: target,
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return storage.RollOutcomeApplyResult{}, session.ErrOutcomeCharacterNotFound
			}
			return storage.RollOutcomeApplyResult{}, fmt.Errorf("get daggerheart character profile: %w", err)
		}

		delta := perTargetDelta[target]
		beforeHope := int(dhStateRow.Hope)
		beforeStress := int(dhStateRow.Stress)
		afterHope := beforeHope + delta.HopeDelta
		if afterHope > 6 {
			afterHope = 6
		}
		if afterHope < 0 {
			afterHope = 0
		}
		afterStress := beforeStress + delta.StressDelta
		if afterStress < 0 {
			afterStress = 0
		}
		if afterStress > int(dhProfileRow.StressMax) {
			afterStress = int(dhProfileRow.StressMax)
		}

		if afterHope != beforeHope {
			result.AppliedChanges = append(result.AppliedChanges, session.OutcomeAppliedChange{
				CharacterID: target,
				Field:       session.OutcomeFieldHope,
				Before:      beforeHope,
				After:       afterHope,
			})
		}
		if afterStress != beforeStress {
			result.AppliedChanges = append(result.AppliedChanges, session.OutcomeAppliedChange{
				CharacterID: target,
				Field:       session.OutcomeFieldStress,
				Before:      beforeStress,
				After:       afterStress,
			})
		}

		if afterHope != beforeHope || afterStress != beforeStress {
			if err := qtx.UpdateDaggerheartCharacterStateHopeStress(ctx, db.UpdateDaggerheartCharacterStateHopeStressParams{
				Hope:        int64(afterHope),
				Stress:      int64(afterStress),
				CampaignID:  input.CampaignID,
				CharacterID: target,
			}); err != nil {
				return storage.RollOutcomeApplyResult{}, fmt.Errorf("update daggerheart character state: %w", err)
			}

			payload := event.CharacterStateChangedPayload{
				CharacterID: target,
				SystemState: map[string]any{
					"daggerheart": map[string]any{
						"hope_before":   beforeHope,
						"hope_after":    afterHope,
						"stress_before": beforeStress,
						"stress_after":  afterStress,
					},
				},
			}
			payloadJSON, err := json.Marshal(payload)
			if err != nil {
				return storage.RollOutcomeApplyResult{}, fmt.Errorf("marshal character state payload: %w", err)
			}
			if _, err := appendEventTx(ctx, qtx, event.Event{
				CampaignID:  input.CampaignID,
				Timestamp:   evtTimestamp,
				Type:        event.TypeCharacterStateChanged,
				SessionID:   input.SessionID,
				RequestID:   input.RequestID,
				ActorType:   event.ActorTypeSystem,
				EntityType:  "character",
				EntityID:    target,
				PayloadJSON: payloadJSON,
			}); err != nil {
				return storage.RollOutcomeApplyResult{}, fmt.Errorf("append character state event: %w", err)
			}
		}

		// Build result state with updated values
		result.UpdatedCharacterStates = append(result.UpdatedCharacterStates, storage.DaggerheartCharacterState{
			CampaignID:  input.CampaignID,
			CharacterID: target,
			Hp:          int(dhStateRow.Hp),
			Hope:        afterHope,
			Stress:      afterStress,
		})
	}

	// Convert applied changes for event payload
	payloadChanges := make([]event.OutcomeAppliedChange, len(result.AppliedChanges))
	for i, ch := range result.AppliedChanges {
		payloadChanges[i] = event.OutcomeAppliedChange{
			CharacterID: ch.CharacterID,
			Field:       string(ch.Field),
			Before:      ch.Before,
			After:       ch.After,
		}
	}

	payload, err := json.Marshal(event.OutcomeAppliedPayload{
		RequestID:            input.RequestID,
		RollSeq:              input.RollSeq,
		Targets:              input.Targets,
		RequiresComplication: input.RequiresComplication,
		AppliedChanges:       payloadChanges,
	})
	if err != nil {
		return storage.RollOutcomeApplyResult{}, fmt.Errorf("marshal outcome applied payload: %w", err)
	}

	// Use unified event table
	if _, err := appendEventTx(ctx, qtx, event.Event{
		CampaignID:   input.CampaignID,
		Timestamp:    evtTimestamp,
		Type:         event.TypeOutcomeApplied,
		SessionID:    input.SessionID,
		RequestID:    input.RequestID,
		InvocationID: input.InvocationID,
		ActorType:    event.ActorTypeSystem,
		EntityType:   "outcome",
		EntityID:     input.RequestID,
		PayloadJSON:  payload,
	}); err != nil {
		return storage.RollOutcomeApplyResult{}, fmt.Errorf("append outcome applied event: %w", err)
	}

	if err := qtx.MarkOutcomeApplied(ctx, db.MarkOutcomeAppliedParams{
		CampaignID: input.CampaignID,
		SessionID:  input.SessionID,
		RequestID:  input.RequestID,
	}); err != nil {
		return storage.RollOutcomeApplyResult{}, fmt.Errorf("mark outcome applied: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return storage.RollOutcomeApplyResult{}, fmt.Errorf("commit: %w", err)
	}

	return result, nil
}

func appendEventTx(ctx context.Context, qtx *db.Queries, evt event.Event) (event.Event, error) {
	if qtx == nil {
		return event.Event{}, fmt.Errorf("event store is not configured")
	}

	normalized, err := event.NormalizeForAppend(evt)
	if err != nil {
		return event.Event{}, err
	}
	evt = normalized
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}

	if err := qtx.InitEventSeq(ctx, evt.CampaignID); err != nil {
		return event.Event{}, fmt.Errorf("init event seq: %w", err)
	}

	seq, err := qtx.GetEventSeq(ctx, evt.CampaignID)
	if err != nil {
		return event.Event{}, fmt.Errorf("get event seq: %w", err)
	}
	if err := qtx.IncrementEventSeq(ctx, evt.CampaignID); err != nil {
		return event.Event{}, fmt.Errorf("increment event seq: %w", err)
	}
	if seq <= 0 {
		return event.Event{}, fmt.Errorf("event seq is required")
	}
	evt.Seq = uint64(seq)

	envelope := map[string]any{
		"campaign_id": evt.CampaignID,
		"event_type":  string(evt.Type),
		"timestamp":   evt.Timestamp.Format(time.RFC3339Nano),
		"actor_type":  string(evt.ActorType),
		"payload":     json.RawMessage(evt.PayloadJSON),
	}
	if evt.SessionID != "" {
		envelope["session_id"] = evt.SessionID
	}
	if evt.RequestID != "" {
		envelope["request_id"] = evt.RequestID
	}
	if evt.ActorID != "" {
		envelope["actor_id"] = evt.ActorID
	}
	if evt.EntityType != "" {
		envelope["entity_type"] = evt.EntityType
	}
	if evt.EntityID != "" {
		envelope["entity_id"] = evt.EntityID
	}

	hash, err := encoding.ContentHash(envelope)
	if err != nil {
		return event.Event{}, fmt.Errorf("compute event hash: %w", err)
	}
	if strings.TrimSpace(hash) == "" {
		return event.Event{}, fmt.Errorf("event hash is required")
	}
	if err := qtx.AppendEvent(ctx, db.AppendEventParams{
		CampaignID:   evt.CampaignID,
		Seq:          seq,
		EventHash:    hash,
		Timestamp:    toMillis(evt.Timestamp),
		EventType:    string(evt.Type),
		SessionID:    evt.SessionID,
		RequestID:    evt.RequestID,
		InvocationID: evt.InvocationID,
		ActorType:    string(evt.ActorType),
		ActorID:      evt.ActorID,
		EntityType:   evt.EntityType,
		EntityID:     evt.EntityID,
		PayloadJson:  evt.PayloadJSON,
	}); err != nil {
		return event.Event{}, fmt.Errorf("append event: %w", err)
	}
	evt.Hash = hash

	return evt, nil
}

// Conversion helpers

func gameSystemToString(gs commonv1.GameSystem) string {
	switch gs {
	case commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART:
		return "DAGGERHEART"
	default:
		return "UNSPECIFIED"
	}
}

func stringToGameSystem(s string) commonv1.GameSystem {
	switch s {
	case "DAGGERHEART":
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
	default:
		return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
	}
}

func campaignStatusToString(cs campaign.CampaignStatus) string {
	switch cs {
	case campaign.CampaignStatusDraft:
		return "DRAFT"
	case campaign.CampaignStatusActive:
		return "ACTIVE"
	case campaign.CampaignStatusCompleted:
		return "COMPLETED"
	case campaign.CampaignStatusArchived:
		return "ARCHIVED"
	default:
		return "UNSPECIFIED"
	}
}

func stringToCampaignStatus(s string) campaign.CampaignStatus {
	switch s {
	case "DRAFT":
		return campaign.CampaignStatusDraft
	case "ACTIVE":
		return campaign.CampaignStatusActive
	case "COMPLETED":
		return campaign.CampaignStatusCompleted
	case "ARCHIVED":
		return campaign.CampaignStatusArchived
	default:
		return campaign.CampaignStatusUnspecified
	}
}

func gmModeToString(gm campaign.GmMode) string {
	switch gm {
	case campaign.GmModeHuman:
		return "HUMAN"
	case campaign.GmModeAI:
		return "AI"
	case campaign.GmModeHybrid:
		return "HYBRID"
	default:
		return "UNSPECIFIED"
	}
}

func stringToGmMode(s string) campaign.GmMode {
	switch s {
	case "HUMAN":
		return campaign.GmModeHuman
	case "AI":
		return campaign.GmModeAI
	case "HYBRID":
		return campaign.GmModeHybrid
	default:
		return campaign.GmModeUnspecified
	}
}

func participantRoleToString(pr participant.ParticipantRole) string {
	switch pr {
	case participant.ParticipantRoleGM:
		return "GM"
	case participant.ParticipantRolePlayer:
		return "PLAYER"
	default:
		return "UNSPECIFIED"
	}
}

func stringToParticipantRole(s string) participant.ParticipantRole {
	switch s {
	case "GM":
		return participant.ParticipantRoleGM
	case "PLAYER":
		return participant.ParticipantRolePlayer
	default:
		return participant.ParticipantRoleUnspecified
	}
}

func participantControllerToString(pc participant.Controller) string {
	switch pc {
	case participant.ControllerHuman:
		return "HUMAN"
	case participant.ControllerAI:
		return "AI"
	default:
		return "UNSPECIFIED"
	}
}

func boolToInt(value bool) int64 {
	if value {
		return 1
	}
	return 0
}

func intToBool(value int64) bool {
	return value != 0
}

func stringToParticipantController(s string) participant.Controller {
	switch s {
	case "HUMAN":
		return participant.ControllerHuman
	case "AI":
		return participant.ControllerAI
	default:
		return participant.ControllerUnspecified
	}
}

func characterKindToString(ck character.CharacterKind) string {
	switch ck {
	case character.CharacterKindPC:
		return "PC"
	case character.CharacterKindNPC:
		return "NPC"
	default:
		return "UNSPECIFIED"
	}
}

func stringToCharacterKind(s string) character.CharacterKind {
	switch s {
	case "PC":
		return character.CharacterKindPC
	case "NPC":
		return character.CharacterKindNPC
	default:
		return character.CharacterKindUnspecified
	}
}

func sessionStatusToString(ss session.SessionStatus) string {
	switch ss {
	case session.SessionStatusActive:
		return "ACTIVE"
	case session.SessionStatusEnded:
		return "ENDED"
	default:
		return "UNSPECIFIED"
	}
}

func stringToSessionStatus(s string) session.SessionStatus {
	switch s {
	case "ACTIVE":
		return session.SessionStatusActive
	case "ENDED":
		return session.SessionStatusEnded
	default:
		return session.SessionStatusUnspecified
	}
}

// Domain conversion helpers

// campaignRowData holds the common fields from campaign row types.
type campaignRowData struct {
	ID               string
	Name             string
	GameSystem       string
	Status           string
	GmMode           string
	ParticipantCount int64
	CharacterCount   int64
	ThemePrompt      string
	CreatedAt        int64
	LastActivityAt   int64
	UpdatedAt        int64
	CompletedAt      sql.NullInt64
	ArchivedAt       sql.NullInt64
}

func campaignRowDataToDomain(row campaignRowData) (campaign.Campaign, error) {
	c := campaign.Campaign{
		ID:               row.ID,
		Name:             row.Name,
		System:           stringToGameSystem(row.GameSystem),
		Status:           stringToCampaignStatus(row.Status),
		GmMode:           stringToGmMode(row.GmMode),
		ParticipantCount: int(row.ParticipantCount),
		CharacterCount:   int(row.CharacterCount),
		ThemePrompt:      row.ThemePrompt,
		CreatedAt:        fromMillis(row.CreatedAt),
		LastActivityAt:   fromMillis(row.LastActivityAt),
		UpdatedAt:        fromMillis(row.UpdatedAt),
	}
	c.CompletedAt = fromNullMillis(row.CompletedAt)
	c.ArchivedAt = fromNullMillis(row.ArchivedAt)

	return c, nil
}

func dbCampaignToDomain(row db.Campaign) (campaign.Campaign, error) {
	return campaignRowDataToDomain(campaignRowData{
		ID:               row.ID,
		Name:             row.Name,
		GameSystem:       row.GameSystem,
		Status:           row.Status,
		GmMode:           row.GmMode,
		ParticipantCount: row.ParticipantCount,
		CharacterCount:   row.CharacterCount,
		ThemePrompt:      row.ThemePrompt,
		CreatedAt:        row.CreatedAt,
		LastActivityAt:   row.LastActivityAt,
		UpdatedAt:        row.UpdatedAt,
		CompletedAt:      row.CompletedAt,
		ArchivedAt:       row.ArchivedAt,
	})
}

func dbGetCampaignRowToDomain(row db.GetCampaignRow) (campaign.Campaign, error) {
	return campaignRowDataToDomain(campaignRowData{
		ID:               row.ID,
		Name:             row.Name,
		GameSystem:       row.GameSystem,
		Status:           row.Status,
		GmMode:           row.GmMode,
		ParticipantCount: row.ParticipantCount,
		CharacterCount:   row.CharacterCount,
		ThemePrompt:      row.ThemePrompt,
		CreatedAt:        row.CreatedAt,
		LastActivityAt:   row.LastActivityAt,
		UpdatedAt:        row.UpdatedAt,
		CompletedAt:      row.CompletedAt,
		ArchivedAt:       row.ArchivedAt,
	})
}

func dbListCampaignsRowToDomain(row db.ListCampaignsRow) (campaign.Campaign, error) {
	return campaignRowDataToDomain(campaignRowData{
		ID:               row.ID,
		Name:             row.Name,
		GameSystem:       row.GameSystem,
		Status:           row.Status,
		GmMode:           row.GmMode,
		ParticipantCount: row.ParticipantCount,
		CharacterCount:   row.CharacterCount,
		ThemePrompt:      row.ThemePrompt,
		CreatedAt:        row.CreatedAt,
		LastActivityAt:   row.LastActivityAt,
		UpdatedAt:        row.UpdatedAt,
		CompletedAt:      row.CompletedAt,
		ArchivedAt:       row.ArchivedAt,
	})
}

func dbListAllCampaignsRowToDomain(row db.ListAllCampaignsRow) (campaign.Campaign, error) {
	return campaignRowDataToDomain(campaignRowData{
		ID:               row.ID,
		Name:             row.Name,
		GameSystem:       row.GameSystem,
		Status:           row.Status,
		GmMode:           row.GmMode,
		ParticipantCount: row.ParticipantCount,
		CharacterCount:   row.CharacterCount,
		ThemePrompt:      row.ThemePrompt,
		CreatedAt:        row.CreatedAt,
		LastActivityAt:   row.LastActivityAt,
		UpdatedAt:        row.UpdatedAt,
		CompletedAt:      row.CompletedAt,
		ArchivedAt:       row.ArchivedAt,
	})
}

func dbParticipantToDomain(row db.Participant) (participant.Participant, error) {
	return participant.Participant{
		ID:          row.ID,
		CampaignID:  row.CampaignID,
		UserID:      row.UserID,
		DisplayName: row.DisplayName,
		Role:        stringToParticipantRole(row.Role),
		Controller:  stringToParticipantController(row.Controller),
		IsOwner:     intToBool(row.IsOwner),
		CreatedAt:   fromMillis(row.CreatedAt),
		UpdatedAt:   fromMillis(row.UpdatedAt),
	}, nil
}

func dbInviteToDomain(row db.Invite) (invite.Invite, error) {
	return invite.Invite{
		ID:                     row.ID,
		CampaignID:             row.CampaignID,
		ParticipantID:          row.ParticipantID,
		Status:                 invite.StatusFromLabel(row.Status),
		CreatedByParticipantID: row.CreatedByParticipantID,
		CreatedAt:              fromMillis(row.CreatedAt),
		UpdatedAt:              fromMillis(row.UpdatedAt),
	}, nil
}

func dbCharacterToDomain(row db.Character) (character.Character, error) {
	return character.Character{
		ID:         row.ID,
		CampaignID: row.CampaignID,
		Name:       row.Name,
		Kind:       stringToCharacterKind(row.Kind),
		Notes:      row.Notes,
		CreatedAt:  fromMillis(row.CreatedAt),
		UpdatedAt:  fromMillis(row.UpdatedAt),
	}, nil
}

func dbSessionToDomain(row db.Session) (session.Session, error) {
	sess := session.Session{
		ID:         row.ID,
		CampaignID: row.CampaignID,
		Name:       row.Name,
		Status:     stringToSessionStatus(row.Status),
		StartedAt:  fromMillis(row.StartedAt),
		UpdatedAt:  fromMillis(row.UpdatedAt),
	}
	sess.EndedAt = fromNullMillis(row.EndedAt)

	return sess, nil
}

// Daggerheart-specific storage methods

// PutDaggerheartCharacterProfile persists a Daggerheart character profile extension.
func (s *Store) PutDaggerheartCharacterProfile(ctx context.Context, profile storage.DaggerheartCharacterProfile) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(profile.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(profile.CharacterID) == "" {
		return fmt.Errorf("character id is required")
	}

	return s.q.PutDaggerheartCharacterProfile(ctx, db.PutDaggerheartCharacterProfileParams{
		CampaignID:      profile.CampaignID,
		CharacterID:     profile.CharacterID,
		HpMax:           int64(profile.HpMax),
		StressMax:       int64(profile.StressMax),
		Evasion:         int64(profile.Evasion),
		MajorThreshold:  int64(profile.MajorThreshold),
		SevereThreshold: int64(profile.SevereThreshold),
		Agility:         int64(profile.Agility),
		Strength:        int64(profile.Strength),
		Finesse:         int64(profile.Finesse),
		Instinct:        int64(profile.Instinct),
		Presence:        int64(profile.Presence),
		Knowledge:       int64(profile.Knowledge),
	})
}

// GetDaggerheartCharacterProfile retrieves a Daggerheart character profile extension.
func (s *Store) GetDaggerheartCharacterProfile(ctx context.Context, campaignID, characterID string) (storage.DaggerheartCharacterProfile, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartCharacterProfile{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartCharacterProfile{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.DaggerheartCharacterProfile{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(characterID) == "" {
		return storage.DaggerheartCharacterProfile{}, fmt.Errorf("character id is required")
	}

	row, err := s.q.GetDaggerheartCharacterProfile(ctx, db.GetDaggerheartCharacterProfileParams{
		CampaignID:  campaignID,
		CharacterID: characterID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartCharacterProfile{}, storage.ErrNotFound
		}
		return storage.DaggerheartCharacterProfile{}, fmt.Errorf("get daggerheart character profile: %w", err)
	}

	return storage.DaggerheartCharacterProfile{
		CampaignID:      row.CampaignID,
		CharacterID:     row.CharacterID,
		HpMax:           int(row.HpMax),
		StressMax:       int(row.StressMax),
		Evasion:         int(row.Evasion),
		MajorThreshold:  int(row.MajorThreshold),
		SevereThreshold: int(row.SevereThreshold),
		Agility:         int(row.Agility),
		Strength:        int(row.Strength),
		Finesse:         int(row.Finesse),
		Instinct:        int(row.Instinct),
		Presence:        int(row.Presence),
		Knowledge:       int(row.Knowledge),
	}, nil
}

// PutDaggerheartCharacterState persists a Daggerheart character state extension.
func (s *Store) PutDaggerheartCharacterState(ctx context.Context, state storage.DaggerheartCharacterState) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(state.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(state.CharacterID) == "" {
		return fmt.Errorf("character id is required")
	}

	return s.q.PutDaggerheartCharacterState(ctx, db.PutDaggerheartCharacterStateParams{
		CampaignID:  state.CampaignID,
		CharacterID: state.CharacterID,
		Hp:          int64(state.Hp),
		Hope:        int64(state.Hope),
		Stress:      int64(state.Stress),
	})
}

// GetDaggerheartCharacterState retrieves a Daggerheart character state extension.
func (s *Store) GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (storage.DaggerheartCharacterState, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartCharacterState{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartCharacterState{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.DaggerheartCharacterState{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(characterID) == "" {
		return storage.DaggerheartCharacterState{}, fmt.Errorf("character id is required")
	}

	row, err := s.q.GetDaggerheartCharacterState(ctx, db.GetDaggerheartCharacterStateParams{
		CampaignID:  campaignID,
		CharacterID: characterID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartCharacterState{}, storage.ErrNotFound
		}
		return storage.DaggerheartCharacterState{}, fmt.Errorf("get daggerheart character state: %w", err)
	}

	return storage.DaggerheartCharacterState{
		CampaignID:  row.CampaignID,
		CharacterID: row.CharacterID,
		Hp:          int(row.Hp),
		Hope:        int(row.Hope),
		Stress:      int(row.Stress),
	}, nil
}

// PutDaggerheartSnapshot persists a Daggerheart snapshot projection.
func (s *Store) PutDaggerheartSnapshot(ctx context.Context, snap storage.DaggerheartSnapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(snap.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}

	return s.q.PutDaggerheartSnapshot(ctx, db.PutDaggerheartSnapshotParams{
		CampaignID: snap.CampaignID,
		GmFear:     int64(snap.GMFear),
	})
}

// GetDaggerheartSnapshot retrieves the Daggerheart snapshot projection for a campaign.
func (s *Store) GetDaggerheartSnapshot(ctx context.Context, campaignID string) (storage.DaggerheartSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartSnapshot{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartSnapshot{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.DaggerheartSnapshot{}, fmt.Errorf("campaign id is required")
	}

	row, err := s.q.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return zero-value for not found (consistent with GetGmFear behavior)
			return storage.DaggerheartSnapshot{CampaignID: campaignID, GMFear: 0}, nil
		}
		return storage.DaggerheartSnapshot{}, fmt.Errorf("get daggerheart snapshot: %w", err)
	}

	return storage.DaggerheartSnapshot{
		CampaignID: row.CampaignID,
		GMFear:     int(row.GmFear),
	}, nil
}

// EventStore methods (unified event journal)

// AppendEvent atomically appends an event and returns it with sequence and hash set.
func (s *Store) AppendEvent(ctx context.Context, evt event.Event) (event.Event, error) {
	if err := ctx.Err(); err != nil {
		return event.Event{}, err
	}
	if s == nil || s.sqlDB == nil {
		return event.Event{}, fmt.Errorf("storage is not configured")
	}

	validated, err := event.NormalizeForAppend(evt)
	if err != nil {
		return event.Event{}, err
	}
	evt = validated

	tx, err := s.sqlDB.BeginTx(ctx, nil)
	if err != nil {
		return event.Event{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	qtx := s.q.WithTx(tx)

	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}

	if err := qtx.InitEventSeq(ctx, evt.CampaignID); err != nil {
		return event.Event{}, fmt.Errorf("init event seq: %w", err)
	}

	seq, err := qtx.GetEventSeq(ctx, evt.CampaignID)
	if err != nil {
		return event.Event{}, fmt.Errorf("get event seq: %w", err)
	}
	evt.Seq = uint64(seq)

	if err := qtx.IncrementEventSeq(ctx, evt.CampaignID); err != nil {
		return event.Event{}, fmt.Errorf("increment event seq: %w", err)
	}

	// Compute content hash for the event (excludes seq which is assigned by storage)
	envelope := map[string]any{
		"campaign_id": evt.CampaignID,
		"event_type":  string(evt.Type),
		"timestamp":   evt.Timestamp.Format(time.RFC3339Nano),
		"actor_type":  string(evt.ActorType),
		"payload":     json.RawMessage(evt.PayloadJSON),
	}
	if evt.SessionID != "" {
		envelope["session_id"] = evt.SessionID
	}
	if evt.RequestID != "" {
		envelope["request_id"] = evt.RequestID
	}
	if evt.ActorID != "" {
		envelope["actor_id"] = evt.ActorID
	}
	if evt.EntityType != "" {
		envelope["entity_type"] = evt.EntityType
	}
	if evt.EntityID != "" {
		envelope["entity_id"] = evt.EntityID
	}

	hash, err := encoding.ContentHash(envelope)
	if err != nil {
		return event.Event{}, fmt.Errorf("compute event hash: %w", err)
	}
	evt.Hash = hash

	if err := qtx.AppendEvent(ctx, db.AppendEventParams{
		CampaignID:   evt.CampaignID,
		Seq:          int64(evt.Seq),
		EventHash:    evt.Hash,
		Timestamp:    toMillis(evt.Timestamp),
		EventType:    string(evt.Type),
		SessionID:    evt.SessionID,
		RequestID:    evt.RequestID,
		InvocationID: evt.InvocationID,
		ActorType:    string(evt.ActorType),
		ActorID:      evt.ActorID,
		EntityType:   evt.EntityType,
		EntityID:     evt.EntityID,
		PayloadJson:  evt.PayloadJSON,
	}); err != nil {
		if isConstraintError(err) {
			stored, lookupErr := s.GetEventByHash(ctx, evt.Hash)
			if lookupErr == nil {
				return stored, nil
			}
		}
		return event.Event{}, fmt.Errorf("append event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return event.Event{}, fmt.Errorf("commit: %w", err)
	}

	return evt, nil
}

func isConstraintError(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	code := sqliteErr.Code()
	return code == sqlite3.SQLITE_CONSTRAINT || code == sqlite3.SQLITE_CONSTRAINT_UNIQUE
}

func isParticipantUserConflict(err error) bool {
	if !isConstraintError(err) {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "idx_participants_campaign_user") ||
		(strings.Contains(message, "participant") && strings.Contains(message, "user_id"))
}

// AppendTelemetryEvent records an operational telemetry event.
func (s *Store) AppendTelemetryEvent(ctx context.Context, evt storage.TelemetryEvent) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(evt.EventName) == "" {
		return fmt.Errorf("event name is required")
	}
	if strings.TrimSpace(evt.Severity) == "" {
		return fmt.Errorf("severity is required")
	}
	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now().UTC()
	}
	if len(evt.AttributesJSON) == 0 && len(evt.Attributes) > 0 {
		payload, err := json.Marshal(evt.Attributes)
		if err != nil {
			return fmt.Errorf("marshal telemetry attributes: %w", err)
		}
		evt.AttributesJSON = payload
	}

	return s.q.AppendTelemetryEvent(ctx, db.AppendTelemetryEventParams{
		Timestamp:      toMillis(evt.Timestamp),
		EventName:      evt.EventName,
		Severity:       evt.Severity,
		CampaignID:     toNullString(evt.CampaignID),
		SessionID:      toNullString(evt.SessionID),
		ActorType:      toNullString(evt.ActorType),
		ActorID:        toNullString(evt.ActorID),
		RequestID:      toNullString(evt.RequestID),
		InvocationID:   toNullString(evt.InvocationID),
		TraceID:        toNullString(evt.TraceID),
		SpanID:         toNullString(evt.SpanID),
		AttributesJson: evt.AttributesJSON,
	})
}

// GetGameStatistics returns aggregate counts across the game data set.
func (s *Store) GetGameStatistics(ctx context.Context, since *time.Time) (storage.GameStatistics, error) {
	if err := ctx.Err(); err != nil {
		return storage.GameStatistics{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.GameStatistics{}, fmt.Errorf("storage is not configured")
	}

	sinceValue := toNullMillis(since)

	row, err := s.q.GetGameStatistics(ctx, sinceValue)
	if err != nil {
		return storage.GameStatistics{}, fmt.Errorf("get game statistics: %w", err)
	}

	return storage.GameStatistics{
		CampaignCount:    row.CampaignCount,
		SessionCount:     row.SessionCount,
		CharacterCount:   row.CharacterCount,
		ParticipantCount: row.ParticipantCount,
	}, nil
}

func toNullString(value string) sql.NullString {
	if strings.TrimSpace(value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

// GetEventByHash retrieves an event by its content hash.
func (s *Store) GetEventByHash(ctx context.Context, hash string) (event.Event, error) {
	if err := ctx.Err(); err != nil {
		return event.Event{}, err
	}
	if s == nil || s.sqlDB == nil {
		return event.Event{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(hash) == "" {
		return event.Event{}, fmt.Errorf("event hash is required")
	}

	row, err := s.q.GetEventByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return event.Event{}, storage.ErrNotFound
		}
		return event.Event{}, fmt.Errorf("get event by hash: %w", err)
	}

	return dbEventToDomain(row)
}

// GetEventBySeq retrieves a specific event by sequence number.
func (s *Store) GetEventBySeq(ctx context.Context, campaignID string, seq uint64) (event.Event, error) {
	if err := ctx.Err(); err != nil {
		return event.Event{}, err
	}
	if s == nil || s.sqlDB == nil {
		return event.Event{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return event.Event{}, fmt.Errorf("campaign id is required")
	}

	row, err := s.q.GetEventBySeq(ctx, db.GetEventBySeqParams{
		CampaignID: campaignID,
		Seq:        int64(seq),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return event.Event{}, storage.ErrNotFound
		}
		return event.Event{}, fmt.Errorf("get event by seq: %w", err)
	}

	return dbEventToDomain(row)
}

// ListEvents returns events ordered by sequence ascending.
func (s *Store) ListEvents(ctx context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}

	rows, err := s.q.ListEvents(ctx, db.ListEventsParams{
		CampaignID: campaignID,
		Seq:        int64(afterSeq),
		Limit:      int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}

	return dbEventsToDomain(rows)
}

// ListEventsBySession returns events for a specific session.
func (s *Store) ListEventsBySession(ctx context.Context, campaignID, sessionID string, afterSeq uint64, limit int) ([]event.Event, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return nil, fmt.Errorf("session id is required")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}

	rows, err := s.q.ListEventsBySession(ctx, db.ListEventsBySessionParams{
		CampaignID: campaignID,
		SessionID:  sessionID,
		Seq:        int64(afterSeq),
		Limit:      int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list events by session: %w", err)
	}

	return dbEventsToDomain(rows)
}

// GetLatestEventSeq returns the latest event sequence number for a campaign.
func (s *Store) GetLatestEventSeq(ctx context.Context, campaignID string) (uint64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if s == nil || s.sqlDB == nil {
		return 0, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return 0, fmt.Errorf("campaign id is required")
	}

	seq, err := s.q.GetLatestEventSeq(ctx, campaignID)
	if err != nil {
		return 0, fmt.Errorf("get latest event seq: %w", err)
	}

	return uint64(seq), nil
}

// ListEventsPage returns a paginated, filtered, and sorted list of events.
func (s *Store) ListEventsPage(ctx context.Context, req storage.ListEventsPageRequest) (storage.ListEventsPageResult, error) {
	if err := ctx.Err(); err != nil {
		return storage.ListEventsPageResult{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ListEventsPageResult{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(req.CampaignID) == "" {
		return storage.ListEventsPageResult{}, fmt.Errorf("campaign id is required")
	}
	if req.PageSize <= 0 {
		req.PageSize = 50
	}
	if req.PageSize > 200 {
		req.PageSize = 200
	}

	// Build the base WHERE clause
	whereClause := "campaign_id = ?"
	params := []any{req.CampaignID}

	// Add cursor condition if paginating
	// The cursor direction directly determines the comparison operator:
	// - Forward (fwd): seq > cursor
	// - Backward (bwd): seq < cursor
	// The Descending flag only affects ORDER BY, not the cursor comparison.
	// The event_service.go is responsible for choosing the right cursor direction
	// based on both the sort order and whether we're going to next/prev page.
	if req.CursorSeq > 0 {
		if req.CursorDir == "bwd" {
			whereClause += " AND seq < ?"
		} else {
			whereClause += " AND seq > ?"
		}
		params = append(params, req.CursorSeq)
	}

	// Add filter clause if provided
	if req.FilterClause != "" {
		whereClause += " AND " + req.FilterClause
		params = append(params, req.FilterParams...)
	}

	// Determine sort order
	orderClause := "ORDER BY seq ASC"
	if req.Descending {
		orderClause = "ORDER BY seq DESC"
	}
	// For "previous page" navigation, temporarily reverse the sort to fetch
	// items nearest to the cursor first, then reverse the results afterward.
	if req.CursorReverse {
		if req.Descending {
			orderClause = "ORDER BY seq ASC"
		} else {
			orderClause = "ORDER BY seq DESC"
		}
	}

	// Fetch one extra row to detect if there are more pages
	limitClause := fmt.Sprintf("LIMIT %d", req.PageSize+1)

	// Build and execute the query
	query := fmt.Sprintf(
		"SELECT campaign_id, seq, event_hash, timestamp, event_type, session_id, request_id, invocation_id, actor_type, actor_id, entity_type, entity_id, payload_json FROM events WHERE %s %s %s",
		whereClause,
		orderClause,
		limitClause,
	)

	rows, err := s.sqlDB.QueryContext(ctx, query, params...)
	if err != nil {
		return storage.ListEventsPageResult{}, fmt.Errorf("query events: %w", err)
	}
	defer rows.Close()

	events := make([]event.Event, 0, req.PageSize)
	for rows.Next() {
		var row db.Event
		if err := rows.Scan(
			&row.CampaignID,
			&row.Seq,
			&row.EventHash,
			&row.Timestamp,
			&row.EventType,
			&row.SessionID,
			&row.RequestID,
			&row.InvocationID,
			&row.ActorType,
			&row.ActorID,
			&row.EntityType,
			&row.EntityID,
			&row.PayloadJson,
		); err != nil {
			return storage.ListEventsPageResult{}, fmt.Errorf("scan event: %w", err)
		}

		evt, err := dbEventToDomain(row)
		if err != nil {
			return storage.ListEventsPageResult{}, err
		}
		events = append(events, evt)
	}
	if err := rows.Err(); err != nil {
		return storage.ListEventsPageResult{}, fmt.Errorf("iterate events: %w", err)
	}

	// Determine if there are more pages
	hasMore := len(events) > req.PageSize
	if hasMore {
		events = events[:req.PageSize]
	}

	// For "previous page" navigation, reverse the results to maintain consistent order
	if req.CursorReverse {
		for i, j := 0, len(events)-1; i < j; i, j = i+1, j-1 {
			events[i], events[j] = events[j], events[i]
		}
	}

	// Build count query for total
	countWhereClause := "campaign_id = ?"
	countParams := []any{req.CampaignID}
	if req.FilterClause != "" {
		countWhereClause += " AND " + req.FilterClause
		countParams = append(countParams, req.FilterParams...)
	}

	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM events WHERE %s", countWhereClause)
	var totalCount int
	if err := s.sqlDB.QueryRowContext(ctx, countQuery, countParams...).Scan(&totalCount); err != nil {
		return storage.ListEventsPageResult{}, fmt.Errorf("count events: %w", err)
	}

	// Determine hasPrev/hasNext based on pagination direction
	result := storage.ListEventsPageResult{
		Events:     events,
		TotalCount: totalCount,
	}

	if req.CursorReverse {
		result.HasNextPage = true // We came from next, so there is a next
		result.HasPrevPage = hasMore
	} else {
		result.HasNextPage = hasMore
		result.HasPrevPage = req.CursorSeq > 0
	}

	return result, nil
}

// Snapshot Store methods

// PutSnapshot stores a snapshot.
func (s *Store) PutSnapshot(ctx context.Context, snapshot storage.Snapshot) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(snapshot.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(snapshot.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}

	return s.q.PutSnapshot(ctx, db.PutSnapshotParams{
		CampaignID:          snapshot.CampaignID,
		SessionID:           snapshot.SessionID,
		EventSeq:            int64(snapshot.EventSeq),
		CharacterStatesJson: snapshot.CharacterStatesJSON,
		GmStateJson:         snapshot.GMStateJSON,
		SystemStateJson:     snapshot.SystemStateJSON,
		CreatedAt:           toMillis(snapshot.CreatedAt),
	})
}

// GetSnapshot retrieves a snapshot by campaign and session ID.
func (s *Store) GetSnapshot(ctx context.Context, campaignID, sessionID string) (storage.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return storage.Snapshot{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.Snapshot{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.Snapshot{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return storage.Snapshot{}, fmt.Errorf("session id is required")
	}

	row, err := s.q.GetSnapshot(ctx, db.GetSnapshotParams{
		CampaignID: campaignID,
		SessionID:  sessionID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.Snapshot{}, storage.ErrNotFound
		}
		return storage.Snapshot{}, fmt.Errorf("get snapshot: %w", err)
	}

	return dbSnapshotToDomain(row)
}

// GetLatestSnapshot retrieves the most recent snapshot for a campaign.
func (s *Store) GetLatestSnapshot(ctx context.Context, campaignID string) (storage.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return storage.Snapshot{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.Snapshot{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.Snapshot{}, fmt.Errorf("campaign id is required")
	}

	row, err := s.q.GetLatestSnapshot(ctx, campaignID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.Snapshot{}, storage.ErrNotFound
		}
		return storage.Snapshot{}, fmt.Errorf("get latest snapshot: %w", err)
	}

	return dbSnapshotToDomain(row)
}

// ListSnapshots returns snapshots ordered by event sequence descending.
func (s *Store) ListSnapshots(ctx context.Context, campaignID string, limit int) ([]storage.Snapshot, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if limit <= 0 {
		return nil, fmt.Errorf("limit must be greater than zero")
	}

	rows, err := s.q.ListSnapshots(ctx, db.ListSnapshotsParams{
		CampaignID: campaignID,
		Limit:      int64(limit),
	})
	if err != nil {
		return nil, fmt.Errorf("list snapshots: %w", err)
	}

	snapshots := make([]storage.Snapshot, 0, len(rows))
	for _, row := range rows {
		snapshot, err := dbSnapshotToDomain(row)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, nil
}

// Campaign Fork Store methods

// GetCampaignForkMetadata retrieves fork metadata for a campaign.
func (s *Store) GetCampaignForkMetadata(ctx context.Context, campaignID string) (storage.ForkMetadata, error) {
	if err := ctx.Err(); err != nil {
		return storage.ForkMetadata{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ForkMetadata{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.ForkMetadata{}, fmt.Errorf("campaign id is required")
	}

	row, err := s.q.GetCampaignForkMetadata(ctx, campaignID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ForkMetadata{}, storage.ErrNotFound
		}
		return storage.ForkMetadata{}, fmt.Errorf("get campaign fork metadata: %w", err)
	}

	metadata := storage.ForkMetadata{}
	if row.ParentCampaignID.Valid {
		metadata.ParentCampaignID = row.ParentCampaignID.String
	}
	if row.ForkEventSeq.Valid {
		metadata.ForkEventSeq = uint64(row.ForkEventSeq.Int64)
	}
	if row.OriginCampaignID.Valid {
		metadata.OriginCampaignID = row.OriginCampaignID.String
	}

	return metadata, nil
}

// SetCampaignForkMetadata sets fork metadata for a campaign.
func (s *Store) SetCampaignForkMetadata(ctx context.Context, campaignID string, metadata storage.ForkMetadata) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}

	var parentCampaignID sql.NullString
	if metadata.ParentCampaignID != "" {
		parentCampaignID = sql.NullString{String: metadata.ParentCampaignID, Valid: true}
	}

	var forkEventSeq sql.NullInt64
	if metadata.ForkEventSeq > 0 {
		forkEventSeq = sql.NullInt64{Int64: int64(metadata.ForkEventSeq), Valid: true}
	}

	var originCampaignID sql.NullString
	if metadata.OriginCampaignID != "" {
		originCampaignID = sql.NullString{String: metadata.OriginCampaignID, Valid: true}
	}

	return s.q.SetCampaignForkMetadata(ctx, db.SetCampaignForkMetadataParams{
		ParentCampaignID: parentCampaignID,
		ForkEventSeq:     forkEventSeq,
		OriginCampaignID: originCampaignID,
		ID:               campaignID,
	})
}

// Domain conversion helpers for events

func dbEventToDomain(row db.Event) (event.Event, error) {
	return event.Event{
		CampaignID:   row.CampaignID,
		Seq:          uint64(row.Seq),
		Hash:         row.EventHash,
		Timestamp:    fromMillis(row.Timestamp),
		Type:         event.Type(row.EventType),
		SessionID:    row.SessionID,
		RequestID:    row.RequestID,
		InvocationID: row.InvocationID,
		ActorType:    event.ActorType(row.ActorType),
		ActorID:      row.ActorID,
		EntityType:   row.EntityType,
		EntityID:     row.EntityID,
		PayloadJSON:  row.PayloadJson,
	}, nil
}

func dbEventsToDomain(rows []db.Event) ([]event.Event, error) {
	events := make([]event.Event, 0, len(rows))
	for _, row := range rows {
		evt, err := dbEventToDomain(row)
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	return events, nil
}

func dbSnapshotToDomain(row db.Snapshot) (storage.Snapshot, error) {
	return storage.Snapshot{
		CampaignID:          row.CampaignID,
		SessionID:           row.SessionID,
		EventSeq:            uint64(row.EventSeq),
		CharacterStatesJSON: row.CharacterStatesJson,
		GMStateJSON:         row.GmStateJson,
		SystemStateJSON:     row.SystemStateJson,
		CreatedAt:           fromMillis(row.CreatedAt),
	}, nil
}
