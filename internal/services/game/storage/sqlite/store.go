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
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/snapshot"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
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
	sqlDB   *sql.DB
	q       *db.Queries
	keyring *integrity.Keyring
}

// Open opens a SQLite projections store at the provided path.
func Open(path string) (*Store, error) {
	return OpenProjections(path)
}

// OpenEvents opens a SQLite event journal store at the provided path.
func OpenEvents(path string, keyring *integrity.Keyring) (*Store, error) {
	return openStore(path, migrations.EventsFS, "events", keyring)
}

// OpenProjections opens a SQLite projections store at the provided path.
func OpenProjections(path string) (*Store, error) {
	return openStore(path, migrations.ProjectionsFS, "projections", nil)
}

// OpenContent opens a SQLite content catalog store at the provided path.
func OpenContent(path string) (*Store, error) {
	return openStore(path, migrations.ContentFS, "content", nil)
}

// Close closes the underlying SQLite database.
func (s *Store) Close() error {
	if s == nil || s.sqlDB == nil {
		return nil
	}
	return s.sqlDB.Close()
}

func openStore(path string, migrationFS fs.FS, migrationRoot string, keyring *integrity.Keyring) (*Store, error) {
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
		sqlDB:   sqlDB,
		q:       db.New(sqlDB),
		keyring: keyring,
	}

	if err := runMigrations(sqlDB, migrationFS, migrationRoot); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	if migrationRoot == "projections" {
		if err := ensureInviteRecipientColumn(sqlDB); err != nil {
			_ = sqlDB.Close()
			return nil, fmt.Errorf("ensure invite schema: %w", err)
		}
	}

	return store, nil
}

func ensureInviteRecipientColumn(sqlDB *sql.DB) error {
	rows, err := sqlDB.Query("PRAGMA table_info(invites)")
	if err != nil {
		return fmt.Errorf("inspect invites table: %w", err)
	}
	defer rows.Close()

	var hasRecipient bool
	for rows.Next() {
		var cid int
		var name string
		var colType string
		var notNull int
		var defaultValue sql.NullString
		var pk int
		if err := rows.Scan(&cid, &name, &colType, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("scan invites table info: %w", err)
		}
		if name == "recipient_user_id" {
			hasRecipient = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read invites table info: %w", err)
	}
	if hasRecipient {
		return nil
	}

	const inviteRebuildSQL = `
DROP INDEX IF EXISTS idx_invites_recipient_status;
DROP INDEX IF EXISTS idx_invites_participant;
DROP INDEX IF EXISTS idx_invites_campaign;
DROP TABLE IF EXISTS invites;

CREATE TABLE invites (
    id TEXT PRIMARY KEY,
    campaign_id TEXT NOT NULL,
    participant_id TEXT NOT NULL,
    recipient_user_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL,
    created_by_participant_id TEXT NOT NULL DEFAULT '',
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL,
    FOREIGN KEY (campaign_id) REFERENCES campaigns(id) ON DELETE CASCADE,
    FOREIGN KEY (campaign_id, participant_id) REFERENCES participants(campaign_id, id) ON DELETE CASCADE
);

CREATE INDEX idx_invites_campaign ON invites(campaign_id);
CREATE INDEX idx_invites_participant ON invites(participant_id);
CREATE INDEX idx_invites_recipient_status ON invites(recipient_user_id, status);
`

	if _, err := sqlDB.Exec(inviteRebuildSQL); err != nil {
		return fmt.Errorf("rebuild invites table: %w", err)
	}

	return nil
}

// runMigrations runs embedded SQL migrations.
func runMigrations(sqlDB *sql.DB, migrationFS fs.FS, migrationRoot string) error {
	entries, err := fs.ReadDir(migrationFS, migrationRoot)
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
		content, err := fs.ReadFile(migrationFS, filepath.Join(migrationRoot, file))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", file, err)
		}

		upSQL := extractUpMigration(string(content))
		if upSQL == "" {
			continue
		}

		if _, err := sqlDB.Exec(upSQL); err != nil {
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
		Locale:           platformi18n.LocaleString(c.Locale),
		GameSystem:       gameSystemToString(c.System),
		Status:           campaignStatusToString(c.Status),
		GmMode:           gmModeToString(c.GmMode),
		ParticipantCount: int64(c.ParticipantCount),
		CharacterCount:   int64(c.CharacterCount),
		ThemePrompt:      c.ThemePrompt,
		CreatedAt:        toMillis(c.CreatedAt),
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
		CampaignID:     p.CampaignID,
		ID:             p.ID,
		UserID:         p.UserID,
		DisplayName:    p.DisplayName,
		Role:           participantRoleToString(p.Role),
		Controller:     participantControllerToString(p.Controller),
		CampaignAccess: participantAccessToString(p.CampaignAccess),
		CreatedAt:      toMillis(p.CreatedAt),
		UpdatedAt:      toMillis(p.UpdatedAt),
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

// PutParticipantClaim stores a user claim for a participant seat.
func (s *Store) PutParticipantClaim(ctx context.Context, campaignID, userID, participantID string, claimedAt time.Time) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("user id is required")
	}
	if strings.TrimSpace(participantID) == "" {
		return fmt.Errorf("participant id is required")
	}
	if claimedAt.IsZero() {
		claimedAt = time.Now().UTC()
	}

	_, err := s.sqlDB.ExecContext(
		ctx,
		"INSERT INTO participant_claims (campaign_id, user_id, participant_id, claimed_at) VALUES (?, ?, ?, ?)",
		campaignID,
		userID,
		participantID,
		toMillis(claimedAt),
	)
	if err == nil {
		return nil
	}
	if !isParticipantClaimConflict(err) {
		return fmt.Errorf("put participant claim: %w", err)
	}

	claim, claimErr := s.GetParticipantClaim(ctx, campaignID, userID)
	if claimErr != nil {
		return fmt.Errorf("get participant claim: %w", claimErr)
	}
	if claim.ParticipantID == participantID {
		return nil
	}
	return apperrors.WithMetadata(
		apperrors.CodeParticipantUserAlreadyClaimed,
		"participant user already claimed",
		map[string]string{
			"CampaignID": campaignID,
			"UserID":     userID,
		},
	)
}

// GetParticipantClaim returns the claim for a user in a campaign.
func (s *Store) GetParticipantClaim(ctx context.Context, campaignID, userID string) (storage.ParticipantClaim, error) {
	if err := ctx.Err(); err != nil {
		return storage.ParticipantClaim{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.ParticipantClaim{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.ParticipantClaim{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(userID) == "" {
		return storage.ParticipantClaim{}, fmt.Errorf("user id is required")
	}

	row := s.sqlDB.QueryRowContext(
		ctx,
		"SELECT participant_id, claimed_at FROM participant_claims WHERE campaign_id = ? AND user_id = ?",
		campaignID,
		userID,
	)
	var participantID string
	var claimedAt int64
	if err := row.Scan(&participantID, &claimedAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.ParticipantClaim{}, storage.ErrNotFound
		}
		return storage.ParticipantClaim{}, fmt.Errorf("get participant claim: %w", err)
	}

	return storage.ParticipantClaim{
		CampaignID:    campaignID,
		UserID:        userID,
		ParticipantID: participantID,
		ClaimedAt:     fromMillis(claimedAt),
	}, nil
}

// DeleteParticipantClaim removes a claim by user.
func (s *Store) DeleteParticipantClaim(ctx context.Context, campaignID, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(userID) == "" {
		return fmt.Errorf("user id is required")
	}

	_, err := s.sqlDB.ExecContext(
		ctx,
		"DELETE FROM participant_claims WHERE campaign_id = ? AND user_id = ?",
		campaignID,
		userID,
	)
	if err != nil {
		return fmt.Errorf("delete participant claim: %w", err)
	}
	return nil
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
		RecipientUserID:        strings.TrimSpace(inv.RecipientUserID),
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
func (s *Store) ListInvites(ctx context.Context, campaignID string, recipientUserID string, status invite.Status, pageSize int, pageToken string) (storage.InvitePage, error) {
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
	recipientUserID = strings.TrimSpace(recipientUserID)
	statusFilter := ""
	if status != invite.StatusUnspecified {
		statusFilter = invite.StatusLabel(status)
	}

	var rows []db.Invite
	var err error
	if pageToken == "" {
		rows, err = s.q.ListInvitesByCampaignPagedFirst(ctx, db.ListInvitesByCampaignPagedFirstParams{
			CampaignID:      campaignID,
			Column2:         recipientUserID,
			RecipientUserID: recipientUserID,
			Column4:         statusFilter,
			Status:          statusFilter,
			Limit:           int64(pageSize + 1),
		})
	} else {
		rows, err = s.q.ListInvitesByCampaignPaged(ctx, db.ListInvitesByCampaignPagedParams{
			CampaignID:      campaignID,
			ID:              pageToken,
			Column3:         recipientUserID,
			RecipientUserID: recipientUserID,
			Column5:         statusFilter,
			Status:          statusFilter,
			Limit:           int64(pageSize + 1),
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

// ListPendingInvites returns a page of pending invite records for a campaign.
func (s *Store) ListPendingInvites(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.InvitePage, error) {
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

	status := invite.StatusLabel(invite.StatusPending)
	var rows []db.Invite
	var err error
	if pageToken == "" {
		rows, err = s.q.ListPendingInvitesByCampaignPagedFirst(ctx, db.ListPendingInvitesByCampaignPagedFirstParams{
			CampaignID: campaignID,
			Status:     status,
			Limit:      int64(pageSize + 1),
		})
	} else {
		rows, err = s.q.ListPendingInvitesByCampaignPaged(ctx, db.ListPendingInvitesByCampaignPagedParams{
			CampaignID: campaignID,
			Status:     status,
			ID:         pageToken,
			Limit:      int64(pageSize + 1),
		})
	}
	if err != nil {
		return storage.InvitePage{}, fmt.Errorf("list pending invites: %w", err)
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

// ListPendingInvitesForRecipient returns a page of pending invite records for a user.
func (s *Store) ListPendingInvitesForRecipient(ctx context.Context, userID string, pageSize int, pageToken string) (storage.InvitePage, error) {
	if err := ctx.Err(); err != nil {
		return storage.InvitePage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.InvitePage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(userID) == "" {
		return storage.InvitePage{}, fmt.Errorf("user id is required")
	}
	if pageSize <= 0 {
		return storage.InvitePage{}, fmt.Errorf("page size must be greater than zero")
	}

	status := invite.StatusLabel(invite.StatusPending)
	var rows []db.Invite
	var err error
	if pageToken == "" {
		rows, err = s.q.ListPendingInvitesByRecipientPagedFirst(ctx, db.ListPendingInvitesByRecipientPagedFirstParams{
			RecipientUserID: userID,
			Status:          status,
			Limit:           int64(pageSize + 1),
		})
	} else {
		rows, err = s.q.ListPendingInvitesByRecipientPaged(ctx, db.ListPendingInvitesByRecipientPagedParams{
			RecipientUserID: userID,
			Status:          status,
			ID:              pageToken,
			Limit:           int64(pageSize + 1),
		})
	}
	if err != nil {
		return storage.InvitePage{}, fmt.Errorf("list pending invites for recipient: %w", err)
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
		CampaignID:              c.CampaignID,
		ID:                      c.ID,
		ControllerParticipantID: toNullString(c.ParticipantID),
		Name:                    c.Name,
		Kind:                    characterKindToString(c.Kind),
		Notes:                   c.Notes,
		CreatedAt:               toMillis(c.CreatedAt),
		UpdatedAt:               toMillis(c.UpdatedAt),
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

// Session gate methods

// PutSessionGate persists a session gate projection.
func (s *Store) PutSessionGate(ctx context.Context, gate storage.SessionGate) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(gate.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(gate.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(gate.GateID) == "" {
		return fmt.Errorf("gate id is required")
	}
	if strings.TrimSpace(gate.GateType) == "" {
		return fmt.Errorf("gate type is required")
	}
	if strings.TrimSpace(gate.Status) == "" {
		return fmt.Errorf("gate status is required")
	}

	return s.q.PutSessionGate(ctx, db.PutSessionGateParams{
		CampaignID:          gate.CampaignID,
		SessionID:           gate.SessionID,
		GateID:              gate.GateID,
		GateType:            gate.GateType,
		Status:              gate.Status,
		Reason:              gate.Reason,
		CreatedAt:           toMillis(gate.CreatedAt),
		CreatedByActorType:  gate.CreatedByActorType,
		CreatedByActorID:    gate.CreatedByActorID,
		ResolvedAt:          toNullMillis(gate.ResolvedAt),
		ResolvedByActorType: toNullString(gate.ResolvedByActorType),
		ResolvedByActorID:   toNullString(gate.ResolvedByActorID),
		MetadataJson:        gate.MetadataJSON,
		ResolutionJson:      gate.ResolutionJSON,
	})
}

// GetSessionGate retrieves a session gate by id.
func (s *Store) GetSessionGate(ctx context.Context, campaignID, sessionID, gateID string) (storage.SessionGate, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionGate{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionGate{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionGate{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return storage.SessionGate{}, fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(gateID) == "" {
		return storage.SessionGate{}, fmt.Errorf("gate id is required")
	}

	row, err := s.q.GetSessionGate(ctx, db.GetSessionGateParams{
		CampaignID: campaignID,
		SessionID:  sessionID,
		GateID:     gateID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SessionGate{}, storage.ErrNotFound
		}
		return storage.SessionGate{}, fmt.Errorf("get session gate: %w", err)
	}

	return dbSessionGateToStorage(row), nil
}

// GetOpenSessionGate retrieves the open gate for a session.
func (s *Store) GetOpenSessionGate(ctx context.Context, campaignID, sessionID string) (storage.SessionGate, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionGate{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionGate{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionGate{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return storage.SessionGate{}, fmt.Errorf("session id is required")
	}

	row, err := s.q.GetOpenSessionGate(ctx, db.GetOpenSessionGateParams{
		CampaignID: campaignID,
		SessionID:  sessionID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SessionGate{}, storage.ErrNotFound
		}
		return storage.SessionGate{}, fmt.Errorf("get open session gate: %w", err)
	}

	return dbSessionGateToStorage(row), nil
}

// Session spotlight methods

// PutSessionSpotlight persists a session spotlight projection.
func (s *Store) PutSessionSpotlight(ctx context.Context, spotlight storage.SessionSpotlight) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(spotlight.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(spotlight.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	if strings.TrimSpace(spotlight.SpotlightType) == "" {
		return fmt.Errorf("spotlight type is required")
	}

	return s.q.PutSessionSpotlight(ctx, db.PutSessionSpotlightParams{
		CampaignID:         spotlight.CampaignID,
		SessionID:          spotlight.SessionID,
		SpotlightType:      spotlight.SpotlightType,
		CharacterID:        spotlight.CharacterID,
		UpdatedAt:          toMillis(spotlight.UpdatedAt),
		UpdatedByActorType: spotlight.UpdatedByActorType,
		UpdatedByActorID:   spotlight.UpdatedByActorID,
	})
}

// GetSessionSpotlight retrieves a session spotlight by session id.
func (s *Store) GetSessionSpotlight(ctx context.Context, campaignID, sessionID string) (storage.SessionSpotlight, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionSpotlight{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionSpotlight{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionSpotlight{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return storage.SessionSpotlight{}, fmt.Errorf("session id is required")
	}

	row, err := s.q.GetSessionSpotlight(ctx, db.GetSessionSpotlightParams{
		CampaignID: campaignID,
		SessionID:  sessionID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SessionSpotlight{}, storage.ErrNotFound
		}
		return storage.SessionSpotlight{}, fmt.Errorf("get session spotlight: %w", err)
	}

	return dbSessionSpotlightToStorage(row), nil
}

// ClearSessionSpotlight removes the current spotlight for a session.
func (s *Store) ClearSessionSpotlight(ctx context.Context, campaignID, sessionID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return fmt.Errorf("session id is required")
	}

	return s.q.ClearSessionSpotlight(ctx, db.ClearSessionSpotlightParams{
		CampaignID: campaignID,
		SessionID:  sessionID,
	})
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
		shortRests := 0
		row, err := qtx.GetDaggerheartSnapshot(ctx, input.CampaignID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				fear = snapshot.GmFear{CampaignID: input.CampaignID, Value: 0}
			} else {
				return storage.RollOutcomeApplyResult{}, fmt.Errorf("get daggerheart snapshot: %w", err)
			}
		} else {
			fear = snapshot.GmFear{CampaignID: row.CampaignID, Value: int(row.GmFear)}
			shortRests = int(row.ConsecutiveShortRests)
		}

		updated, before, after, err := snapshot.ApplyGmFearGain(fear, input.GMFearDelta)
		if err != nil {
			return storage.RollOutcomeApplyResult{}, session.ErrOutcomeGMFearInvalid
		}

		if err := qtx.PutDaggerheartSnapshot(ctx, db.PutDaggerheartSnapshotParams{
			CampaignID:            updated.CampaignID,
			GmFear:                int64(updated.Value),
			ConsecutiveShortRests: int64(shortRests),
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

		payloadJSON, err := json.Marshal(daggerheart.GMFearChangedPayload{
			Before: before,
			After:  after,
		})
		if err != nil {
			return storage.RollOutcomeApplyResult{}, fmt.Errorf("marshal gm fear payload: %w", err)
		}
		if _, err := appendEventTx(ctx, qtx, s.keyring, event.Event{
			CampaignID:    input.CampaignID,
			Timestamp:     evtTimestamp,
			Type:          daggerheart.EventTypeGMFearChanged,
			SessionID:     input.SessionID,
			RequestID:     input.RequestID,
			ActorType:     event.ActorTypeSystem,
			EntityType:    "campaign",
			EntityID:      input.CampaignID,
			SystemID:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
			SystemVersion: daggerheart.SystemVersion,
			PayloadJSON:   payloadJSON,
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
		hopeMax := int(dhStateRow.HopeMax)
		beforeStress := int(dhStateRow.Stress)
		afterHope := beforeHope + delta.HopeDelta
		if afterHope > hopeMax {
			afterHope = hopeMax
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

			payload := daggerheart.CharacterStatePatchedPayload{
				CharacterID:  target,
				HopeBefore:   &beforeHope,
				HopeAfter:    &afterHope,
				StressBefore: &beforeStress,
				StressAfter:  &afterStress,
			}
			payloadJSON, err := json.Marshal(payload)
			if err != nil {
				return storage.RollOutcomeApplyResult{}, fmt.Errorf("marshal character state payload: %w", err)
			}
			if _, err := appendEventTx(ctx, qtx, s.keyring, event.Event{
				CampaignID:    input.CampaignID,
				Timestamp:     evtTimestamp,
				Type:          daggerheart.EventTypeCharacterStatePatched,
				SessionID:     input.SessionID,
				RequestID:     input.RequestID,
				ActorType:     event.ActorTypeSystem,
				EntityType:    "character",
				EntityID:      target,
				SystemID:      commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART.String(),
				SystemVersion: daggerheart.SystemVersion,
				PayloadJSON:   payloadJSON,
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
			HopeMax:     hopeMax,
			Stress:      afterStress,
			LifeState:   dhStateRow.LifeState,
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
	if _, err := appendEventTx(ctx, qtx, s.keyring, event.Event{
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

func appendEventTx(ctx context.Context, qtx *db.Queries, keyring *integrity.Keyring, evt event.Event) (event.Event, error) {
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

	if keyring == nil {
		return event.Event{}, fmt.Errorf("event integrity keyring is required")
	}

	hash, err := integrity.EventHash(evt)
	if err != nil {
		return event.Event{}, fmt.Errorf("compute event hash: %w", err)
	}
	if strings.TrimSpace(hash) == "" {
		return event.Event{}, fmt.Errorf("event hash is required")
	}
	evt.Hash = hash

	prevHash := ""
	if evt.Seq > 1 {
		prevRow, err := qtx.GetEventBySeq(ctx, db.GetEventBySeqParams{
			CampaignID: evt.CampaignID,
			Seq:        int64(evt.Seq - 1),
		})
		if err != nil {
			return event.Event{}, fmt.Errorf("load previous event: %w", err)
		}
		prevHash = prevRow.ChainHash
	}

	chainHash, err := integrity.ChainHash(evt, prevHash)
	if err != nil {
		return event.Event{}, fmt.Errorf("compute chain hash: %w", err)
	}
	if strings.TrimSpace(chainHash) == "" {
		return event.Event{}, fmt.Errorf("chain hash is required")
	}

	signature, keyID, err := keyring.SignChainHash(evt.CampaignID, chainHash)
	if err != nil {
		return event.Event{}, fmt.Errorf("sign chain hash: %w", err)
	}

	evt.PrevHash = prevHash
	evt.ChainHash = chainHash
	evt.Signature = signature
	evt.SignatureKeyID = keyID
	if err := qtx.AppendEvent(ctx, db.AppendEventParams{
		CampaignID:     evt.CampaignID,
		Seq:            seq,
		EventHash:      hash,
		PrevEventHash:  prevHash,
		ChainHash:      chainHash,
		SignatureKeyID: keyID,
		EventSignature: signature,
		Timestamp:      toMillis(evt.Timestamp),
		EventType:      string(evt.Type),
		SessionID:      evt.SessionID,
		RequestID:      evt.RequestID,
		InvocationID:   evt.InvocationID,
		ActorType:      string(evt.ActorType),
		ActorID:        evt.ActorID,
		EntityType:     evt.EntityType,
		EntityID:       evt.EntityID,
		SystemID:       evt.SystemID,
		SystemVersion:  evt.SystemVersion,
		PayloadJson:    evt.PayloadJSON,
	}); err != nil {
		return event.Event{}, fmt.Errorf("append event: %w", err)
	}

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

func participantAccessToString(access participant.CampaignAccess) string {
	switch access {
	case participant.CampaignAccessMember:
		return "MEMBER"
	case participant.CampaignAccessManager:
		return "MANAGER"
	case participant.CampaignAccessOwner:
		return "OWNER"
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

func stringToParticipantAccess(s string) participant.CampaignAccess {
	switch s {
	case "MEMBER":
		return participant.CampaignAccessMember
	case "MANAGER":
		return participant.CampaignAccessManager
	case "OWNER":
		return participant.CampaignAccessOwner
	default:
		return participant.CampaignAccessUnspecified
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
	Locale           string
	GameSystem       string
	Status           string
	GmMode           string
	ParticipantCount int64
	CharacterCount   int64
	ThemePrompt      string
	CreatedAt        int64
	UpdatedAt        int64
	CompletedAt      sql.NullInt64
	ArchivedAt       sql.NullInt64
}

func campaignRowDataToDomain(row campaignRowData) (campaign.Campaign, error) {
	locale := platformi18n.DefaultLocale()
	if parsed, ok := platformi18n.ParseLocale(row.Locale); ok {
		locale = parsed
	}
	c := campaign.Campaign{
		ID:               row.ID,
		Name:             row.Name,
		Locale:           locale,
		System:           stringToGameSystem(row.GameSystem),
		Status:           stringToCampaignStatus(row.Status),
		GmMode:           stringToGmMode(row.GmMode),
		ParticipantCount: int(row.ParticipantCount),
		CharacterCount:   int(row.CharacterCount),
		ThemePrompt:      row.ThemePrompt,
		CreatedAt:        fromMillis(row.CreatedAt),
		UpdatedAt:        fromMillis(row.UpdatedAt),
	}
	c.CompletedAt = fromNullMillis(row.CompletedAt)
	c.ArchivedAt = fromNullMillis(row.ArchivedAt)

	return c, nil
}

func dbGetCampaignRowToDomain(row db.GetCampaignRow) (campaign.Campaign, error) {
	return campaignRowDataToDomain(campaignRowData{
		ID:               row.ID,
		Name:             row.Name,
		Locale:           row.Locale,
		GameSystem:       row.GameSystem,
		Status:           row.Status,
		GmMode:           row.GmMode,
		ParticipantCount: row.ParticipantCount,
		CharacterCount:   row.CharacterCount,
		ThemePrompt:      row.ThemePrompt,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		CompletedAt:      row.CompletedAt,
		ArchivedAt:       row.ArchivedAt,
	})
}

func dbListCampaignsRowToDomain(row db.ListCampaignsRow) (campaign.Campaign, error) {
	return campaignRowDataToDomain(campaignRowData{
		ID:               row.ID,
		Name:             row.Name,
		Locale:           row.Locale,
		GameSystem:       row.GameSystem,
		Status:           row.Status,
		GmMode:           row.GmMode,
		ParticipantCount: row.ParticipantCount,
		CharacterCount:   row.CharacterCount,
		ThemePrompt:      row.ThemePrompt,
		CreatedAt:        row.CreatedAt,
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
		UpdatedAt:        row.UpdatedAt,
		CompletedAt:      row.CompletedAt,
		ArchivedAt:       row.ArchivedAt,
	})
}

func dbParticipantToDomain(row db.Participant) (participant.Participant, error) {
	return participant.Participant{
		ID:             row.ID,
		CampaignID:     row.CampaignID,
		UserID:         row.UserID,
		DisplayName:    row.DisplayName,
		Role:           stringToParticipantRole(row.Role),
		Controller:     stringToParticipantController(row.Controller),
		CampaignAccess: stringToParticipantAccess(row.CampaignAccess),
		CreatedAt:      fromMillis(row.CreatedAt),
		UpdatedAt:      fromMillis(row.UpdatedAt),
	}, nil
}

func dbInviteToDomain(row db.Invite) (invite.Invite, error) {
	return invite.Invite{
		ID:                     row.ID,
		CampaignID:             row.CampaignID,
		ParticipantID:          row.ParticipantID,
		RecipientUserID:        row.RecipientUserID,
		Status:                 invite.StatusFromLabel(row.Status),
		CreatedByParticipantID: row.CreatedByParticipantID,
		CreatedAt:              fromMillis(row.CreatedAt),
		UpdatedAt:              fromMillis(row.UpdatedAt),
	}, nil
}

func dbCharacterToDomain(row db.Character) (character.Character, error) {
	participantID := ""
	if row.ControllerParticipantID.Valid {
		participantID = row.ControllerParticipantID.String
	}
	return character.Character{
		ID:            row.ID,
		CampaignID:    row.CampaignID,
		ParticipantID: participantID,
		Name:          row.Name,
		Kind:          stringToCharacterKind(row.Kind),
		Notes:         row.Notes,
		CreatedAt:     fromMillis(row.CreatedAt),
		UpdatedAt:     fromMillis(row.UpdatedAt),
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

func dbSessionGateToStorage(row db.SessionGate) storage.SessionGate {
	gate := storage.SessionGate{
		CampaignID:         row.CampaignID,
		SessionID:          row.SessionID,
		GateID:             row.GateID,
		GateType:           row.GateType,
		Status:             row.Status,
		Reason:             row.Reason,
		CreatedAt:          fromMillis(row.CreatedAt),
		CreatedByActorType: row.CreatedByActorType,
		CreatedByActorID:   row.CreatedByActorID,
		MetadataJSON:       row.MetadataJson,
		ResolutionJSON:     row.ResolutionJson,
	}
	gate.ResolvedAt = fromNullMillis(row.ResolvedAt)
	if row.ResolvedByActorType.Valid {
		gate.ResolvedByActorType = row.ResolvedByActorType.String
	}
	if row.ResolvedByActorID.Valid {
		gate.ResolvedByActorID = row.ResolvedByActorID.String
	}
	return gate
}

func dbSessionSpotlightToStorage(row db.SessionSpotlight) storage.SessionSpotlight {
	return storage.SessionSpotlight{
		CampaignID:         row.CampaignID,
		SessionID:          row.SessionID,
		SpotlightType:      row.SpotlightType,
		CharacterID:        row.CharacterID,
		UpdatedAt:          fromMillis(row.UpdatedAt),
		UpdatedByActorType: row.UpdatedByActorType,
		UpdatedByActorID:   row.UpdatedByActorID,
	}
}

func dbDaggerheartClassToStorage(row db.DaggerheartClass) (storage.DaggerheartClass, error) {
	class := storage.DaggerheartClass{
		ID:              row.ID,
		Name:            row.Name,
		StartingEvasion: int(row.StartingEvasion),
		StartingHP:      int(row.StartingHp),
		CreatedAt:       fromMillis(row.CreatedAt),
		UpdatedAt:       fromMillis(row.UpdatedAt),
	}
	if row.StartingItemsJson != "" {
		if err := json.Unmarshal([]byte(row.StartingItemsJson), &class.StartingItems); err != nil {
			return storage.DaggerheartClass{}, fmt.Errorf("decode daggerheart class starting items: %w", err)
		}
	}
	if row.FeaturesJson != "" {
		if err := json.Unmarshal([]byte(row.FeaturesJson), &class.Features); err != nil {
			return storage.DaggerheartClass{}, fmt.Errorf("decode daggerheart class features: %w", err)
		}
	}
	if row.HopeFeatureJson != "" {
		if err := json.Unmarshal([]byte(row.HopeFeatureJson), &class.HopeFeature); err != nil {
			return storage.DaggerheartClass{}, fmt.Errorf("decode daggerheart class hope feature: %w", err)
		}
	}
	if row.DomainIdsJson != "" {
		if err := json.Unmarshal([]byte(row.DomainIdsJson), &class.DomainIDs); err != nil {
			return storage.DaggerheartClass{}, fmt.Errorf("decode daggerheart class domain ids: %w", err)
		}
	}
	return class, nil
}

func dbDaggerheartSubclassToStorage(row db.DaggerheartSubclass) (storage.DaggerheartSubclass, error) {
	subclass := storage.DaggerheartSubclass{
		ID:             row.ID,
		Name:           row.Name,
		SpellcastTrait: row.SpellcastTrait,
		CreatedAt:      fromMillis(row.CreatedAt),
		UpdatedAt:      fromMillis(row.UpdatedAt),
	}
	if row.FoundationFeaturesJson != "" {
		if err := json.Unmarshal([]byte(row.FoundationFeaturesJson), &subclass.FoundationFeatures); err != nil {
			return storage.DaggerheartSubclass{}, fmt.Errorf("decode daggerheart subclass foundation features: %w", err)
		}
	}
	if row.SpecializationFeaturesJson != "" {
		if err := json.Unmarshal([]byte(row.SpecializationFeaturesJson), &subclass.SpecializationFeatures); err != nil {
			return storage.DaggerheartSubclass{}, fmt.Errorf("decode daggerheart subclass specialization features: %w", err)
		}
	}
	if row.MasteryFeaturesJson != "" {
		if err := json.Unmarshal([]byte(row.MasteryFeaturesJson), &subclass.MasteryFeatures); err != nil {
			return storage.DaggerheartSubclass{}, fmt.Errorf("decode daggerheart subclass mastery features: %w", err)
		}
	}
	return subclass, nil
}

func dbDaggerheartHeritageToStorage(row db.DaggerheartHeritage) (storage.DaggerheartHeritage, error) {
	heritage := storage.DaggerheartHeritage{
		ID:        row.ID,
		Name:      row.Name,
		Kind:      row.Kind,
		CreatedAt: fromMillis(row.CreatedAt),
		UpdatedAt: fromMillis(row.UpdatedAt),
	}
	if row.FeaturesJson != "" {
		if err := json.Unmarshal([]byte(row.FeaturesJson), &heritage.Features); err != nil {
			return storage.DaggerheartHeritage{}, fmt.Errorf("decode daggerheart heritage features: %w", err)
		}
	}
	return heritage, nil
}

func dbDaggerheartExperienceToStorage(row db.DaggerheartExperience) storage.DaggerheartExperienceEntry {
	return storage.DaggerheartExperienceEntry{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		CreatedAt:   fromMillis(row.CreatedAt),
		UpdatedAt:   fromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartAdversaryEntryToStorage(row db.DaggerheartAdversaryEntry) (storage.DaggerheartAdversaryEntry, error) {
	entry := storage.DaggerheartAdversaryEntry{
		ID:              row.ID,
		Name:            row.Name,
		Tier:            int(row.Tier),
		Role:            row.Role,
		Description:     row.Description,
		Motives:         row.Motives,
		Difficulty:      int(row.Difficulty),
		MajorThreshold:  int(row.MajorThreshold),
		SevereThreshold: int(row.SevereThreshold),
		HP:              int(row.Hp),
		Stress:          int(row.Stress),
		Armor:           int(row.Armor),
		AttackModifier:  int(row.AttackModifier),
		CreatedAt:       fromMillis(row.CreatedAt),
		UpdatedAt:       fromMillis(row.UpdatedAt),
	}
	if row.StandardAttackJson != "" {
		if err := json.Unmarshal([]byte(row.StandardAttackJson), &entry.StandardAttack); err != nil {
			return storage.DaggerheartAdversaryEntry{}, fmt.Errorf("decode daggerheart adversary standard attack: %w", err)
		}
	}
	if row.ExperiencesJson != "" {
		if err := json.Unmarshal([]byte(row.ExperiencesJson), &entry.Experiences); err != nil {
			return storage.DaggerheartAdversaryEntry{}, fmt.Errorf("decode daggerheart adversary experiences: %w", err)
		}
	}
	if row.FeaturesJson != "" {
		if err := json.Unmarshal([]byte(row.FeaturesJson), &entry.Features); err != nil {
			return storage.DaggerheartAdversaryEntry{}, fmt.Errorf("decode daggerheart adversary features: %w", err)
		}
	}
	return entry, nil
}

func dbDaggerheartBeastformToStorage(row db.DaggerheartBeastform) (storage.DaggerheartBeastformEntry, error) {
	entry := storage.DaggerheartBeastformEntry{
		ID:           row.ID,
		Name:         row.Name,
		Tier:         int(row.Tier),
		Examples:     row.Examples,
		Trait:        row.Trait,
		TraitBonus:   int(row.TraitBonus),
		EvasionBonus: int(row.EvasionBonus),
		CreatedAt:    fromMillis(row.CreatedAt),
		UpdatedAt:    fromMillis(row.UpdatedAt),
	}
	if row.AttackJson != "" {
		if err := json.Unmarshal([]byte(row.AttackJson), &entry.Attack); err != nil {
			return storage.DaggerheartBeastformEntry{}, fmt.Errorf("decode daggerheart beastform attack: %w", err)
		}
	}
	if row.AdvantagesJson != "" {
		if err := json.Unmarshal([]byte(row.AdvantagesJson), &entry.Advantages); err != nil {
			return storage.DaggerheartBeastformEntry{}, fmt.Errorf("decode daggerheart beastform advantages: %w", err)
		}
	}
	if row.FeaturesJson != "" {
		if err := json.Unmarshal([]byte(row.FeaturesJson), &entry.Features); err != nil {
			return storage.DaggerheartBeastformEntry{}, fmt.Errorf("decode daggerheart beastform features: %w", err)
		}
	}
	return entry, nil
}

func dbDaggerheartCompanionExperienceToStorage(row db.DaggerheartCompanionExperience) storage.DaggerheartCompanionExperienceEntry {
	return storage.DaggerheartCompanionExperienceEntry{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		CreatedAt:   fromMillis(row.CreatedAt),
		UpdatedAt:   fromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartLootEntryToStorage(row db.DaggerheartLootEntry) storage.DaggerheartLootEntry {
	return storage.DaggerheartLootEntry{
		ID:          row.ID,
		Name:        row.Name,
		Roll:        int(row.Roll),
		Description: row.Description,
		CreatedAt:   fromMillis(row.CreatedAt),
		UpdatedAt:   fromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartDamageTypeToStorage(row db.DaggerheartDamageType) storage.DaggerheartDamageTypeEntry {
	return storage.DaggerheartDamageTypeEntry{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		CreatedAt:   fromMillis(row.CreatedAt),
		UpdatedAt:   fromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartDomainToStorage(row db.DaggerheartDomain) storage.DaggerheartDomain {
	return storage.DaggerheartDomain{
		ID:          row.ID,
		Name:        row.Name,
		Description: row.Description,
		CreatedAt:   fromMillis(row.CreatedAt),
		UpdatedAt:   fromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartDomainCardToStorage(row db.DaggerheartDomainCard) storage.DaggerheartDomainCard {
	return storage.DaggerheartDomainCard{
		ID:          row.ID,
		Name:        row.Name,
		DomainID:    row.DomainID,
		Level:       int(row.Level),
		Type:        row.Type,
		RecallCost:  int(row.RecallCost),
		UsageLimit:  row.UsageLimit,
		FeatureText: row.FeatureText,
		CreatedAt:   fromMillis(row.CreatedAt),
		UpdatedAt:   fromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartWeaponToStorage(row db.DaggerheartWeapon) (storage.DaggerheartWeapon, error) {
	weapon := storage.DaggerheartWeapon{
		ID:         row.ID,
		Name:       row.Name,
		Category:   row.Category,
		Tier:       int(row.Tier),
		Trait:      row.Trait,
		Range:      row.Range,
		DamageType: row.DamageType,
		Burden:     int(row.Burden),
		Feature:    row.Feature,
		CreatedAt:  fromMillis(row.CreatedAt),
		UpdatedAt:  fromMillis(row.UpdatedAt),
	}
	if row.DamageDiceJson != "" {
		if err := json.Unmarshal([]byte(row.DamageDiceJson), &weapon.DamageDice); err != nil {
			return storage.DaggerheartWeapon{}, fmt.Errorf("decode daggerheart weapon damage dice: %w", err)
		}
	}
	return weapon, nil
}

func dbDaggerheartArmorToStorage(row db.DaggerheartArmor) storage.DaggerheartArmor {
	return storage.DaggerheartArmor{
		ID:                  row.ID,
		Name:                row.Name,
		Tier:                int(row.Tier),
		BaseMajorThreshold:  int(row.BaseMajorThreshold),
		BaseSevereThreshold: int(row.BaseSevereThreshold),
		ArmorScore:          int(row.ArmorScore),
		Feature:             row.Feature,
		CreatedAt:           fromMillis(row.CreatedAt),
		UpdatedAt:           fromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartItemToStorage(row db.DaggerheartItem) storage.DaggerheartItem {
	return storage.DaggerheartItem{
		ID:          row.ID,
		Name:        row.Name,
		Rarity:      row.Rarity,
		Kind:        row.Kind,
		StackMax:    int(row.StackMax),
		Description: row.Description,
		EffectText:  row.EffectText,
		CreatedAt:   fromMillis(row.CreatedAt),
		UpdatedAt:   fromMillis(row.UpdatedAt),
	}
}

func dbDaggerheartEnvironmentToStorage(row db.DaggerheartEnvironment) (storage.DaggerheartEnvironment, error) {
	env := storage.DaggerheartEnvironment{
		ID:         row.ID,
		Name:       row.Name,
		Tier:       int(row.Tier),
		Type:       row.Type,
		Difficulty: int(row.Difficulty),
		CreatedAt:  fromMillis(row.CreatedAt),
		UpdatedAt:  fromMillis(row.UpdatedAt),
	}
	if row.ImpulsesJson != "" {
		if err := json.Unmarshal([]byte(row.ImpulsesJson), &env.Impulses); err != nil {
			return storage.DaggerheartEnvironment{}, fmt.Errorf("decode daggerheart environment impulses: %w", err)
		}
	}
	if row.PotentialAdversaryIdsJson != "" {
		if err := json.Unmarshal([]byte(row.PotentialAdversaryIdsJson), &env.PotentialAdversaryIDs); err != nil {
			return storage.DaggerheartEnvironment{}, fmt.Errorf("decode daggerheart environment adversaries: %w", err)
		}
	}
	if row.FeaturesJson != "" {
		if err := json.Unmarshal([]byte(row.FeaturesJson), &env.Features); err != nil {
			return storage.DaggerheartEnvironment{}, fmt.Errorf("decode daggerheart environment features: %w", err)
		}
	}
	if row.PromptsJson != "" {
		if err := json.Unmarshal([]byte(row.PromptsJson), &env.Prompts); err != nil {
			return storage.DaggerheartEnvironment{}, fmt.Errorf("decode daggerheart environment prompts: %w", err)
		}
	}
	return env, nil
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

	experiencesJSON, err := json.Marshal(profile.Experiences)
	if err != nil {
		return fmt.Errorf("marshal experiences: %w", err)
	}

	return s.q.PutDaggerheartCharacterProfile(ctx, db.PutDaggerheartCharacterProfileParams{
		CampaignID:      profile.CampaignID,
		CharacterID:     profile.CharacterID,
		Level:           int64(profile.Level),
		HpMax:           int64(profile.HpMax),
		StressMax:       int64(profile.StressMax),
		Evasion:         int64(profile.Evasion),
		MajorThreshold:  int64(profile.MajorThreshold),
		SevereThreshold: int64(profile.SevereThreshold),
		Proficiency:     int64(profile.Proficiency),
		ArmorScore:      int64(profile.ArmorScore),
		ArmorMax:        int64(profile.ArmorMax),
		ExperiencesJson: string(experiencesJSON),
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

	profile := storage.DaggerheartCharacterProfile{
		CampaignID:      row.CampaignID,
		CharacterID:     row.CharacterID,
		Level:           int(row.Level),
		HpMax:           int(row.HpMax),
		StressMax:       int(row.StressMax),
		Evasion:         int(row.Evasion),
		MajorThreshold:  int(row.MajorThreshold),
		SevereThreshold: int(row.SevereThreshold),
		Proficiency:     int(row.Proficiency),
		ArmorScore:      int(row.ArmorScore),
		ArmorMax:        int(row.ArmorMax),
		Agility:         int(row.Agility),
		Strength:        int(row.Strength),
		Finesse:         int(row.Finesse),
		Instinct:        int(row.Instinct),
		Presence:        int(row.Presence),
		Knowledge:       int(row.Knowledge),
	}
	if row.ExperiencesJson != "" {
		if err := json.Unmarshal([]byte(row.ExperiencesJson), &profile.Experiences); err != nil {
			return storage.DaggerheartCharacterProfile{}, fmt.Errorf("decode experiences: %w", err)
		}
	}
	return profile, nil
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

	conditions := state.Conditions
	if conditions == nil {
		conditions = []string{}
	}
	conditionsJSON, err := json.Marshal(conditions)
	if err != nil {
		return fmt.Errorf("encode conditions: %w", err)
	}

	hopeMax := state.HopeMax
	if hopeMax == 0 {
		hopeMax = daggerheart.HopeMax
	}

	lifeState := state.LifeState
	if strings.TrimSpace(lifeState) == "" {
		lifeState = daggerheart.LifeStateAlive
	}

	return s.q.PutDaggerheartCharacterState(ctx, db.PutDaggerheartCharacterStateParams{
		CampaignID:     state.CampaignID,
		CharacterID:    state.CharacterID,
		Hp:             int64(state.Hp),
		Hope:           int64(state.Hope),
		HopeMax:        int64(hopeMax),
		Stress:         int64(state.Stress),
		Armor:          int64(state.Armor),
		ConditionsJson: string(conditionsJSON),
		LifeState:      lifeState,
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

	var conditions []string
	if row.ConditionsJson != "" {
		if err := json.Unmarshal([]byte(row.ConditionsJson), &conditions); err != nil {
			return storage.DaggerheartCharacterState{}, fmt.Errorf("decode conditions: %w", err)
		}
	}

	lifeState := row.LifeState
	if strings.TrimSpace(lifeState) == "" {
		lifeState = daggerheart.LifeStateAlive
	}

	return storage.DaggerheartCharacterState{
		CampaignID:  row.CampaignID,
		CharacterID: row.CharacterID,
		Hp:          int(row.Hp),
		Hope:        int(row.Hope),
		HopeMax:     int(row.HopeMax),
		Stress:      int(row.Stress),
		Armor:       int(row.Armor),
		Conditions:  conditions,
		LifeState:   lifeState,
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
		CampaignID:            snap.CampaignID,
		GmFear:                int64(snap.GMFear),
		ConsecutiveShortRests: int64(snap.ConsecutiveShortRests),
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
			return storage.DaggerheartSnapshot{CampaignID: campaignID, GMFear: 0, ConsecutiveShortRests: 0}, nil
		}
		return storage.DaggerheartSnapshot{}, fmt.Errorf("get daggerheart snapshot: %w", err)
	}

	return storage.DaggerheartSnapshot{
		CampaignID:            row.CampaignID,
		GMFear:                int(row.GmFear),
		ConsecutiveShortRests: int(row.ConsecutiveShortRests),
	}, nil
}

// PutDaggerheartCountdown persists a Daggerheart countdown projection.
func (s *Store) PutDaggerheartCountdown(ctx context.Context, countdown storage.DaggerheartCountdown) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(countdown.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(countdown.CountdownID) == "" {
		return fmt.Errorf("countdown id is required")
	}

	looping := int64(0)
	if countdown.Looping {
		looping = 1
	}

	return s.q.PutDaggerheartCountdown(ctx, db.PutDaggerheartCountdownParams{
		CampaignID:  countdown.CampaignID,
		CountdownID: countdown.CountdownID,
		Name:        countdown.Name,
		Kind:        countdown.Kind,
		Current:     int64(countdown.Current),
		Max:         int64(countdown.Max),
		Direction:   countdown.Direction,
		Looping:     looping,
	})
}

// GetDaggerheartCountdown retrieves a Daggerheart countdown projection for a campaign.
func (s *Store) GetDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) (storage.DaggerheartCountdown, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartCountdown{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartCountdown{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.DaggerheartCountdown{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(countdownID) == "" {
		return storage.DaggerheartCountdown{}, fmt.Errorf("countdown id is required")
	}

	row, err := s.q.GetDaggerheartCountdown(ctx, db.GetDaggerheartCountdownParams{
		CampaignID:  campaignID,
		CountdownID: countdownID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartCountdown{}, storage.ErrNotFound
		}
		return storage.DaggerheartCountdown{}, fmt.Errorf("get daggerheart countdown: %w", err)
	}

	return storage.DaggerheartCountdown{
		CampaignID:  row.CampaignID,
		CountdownID: row.CountdownID,
		Name:        row.Name,
		Kind:        row.Kind,
		Current:     int(row.Current),
		Max:         int(row.Max),
		Direction:   row.Direction,
		Looping:     row.Looping != 0,
	}, nil
}

// ListDaggerheartCountdowns retrieves countdown projections for a campaign.
func (s *Store) ListDaggerheartCountdowns(ctx context.Context, campaignID string) ([]storage.DaggerheartCountdown, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}

	rows, err := s.q.ListDaggerheartCountdowns(ctx, campaignID)
	if err != nil {
		return nil, fmt.Errorf("list daggerheart countdowns: %w", err)
	}

	countdowns := make([]storage.DaggerheartCountdown, 0, len(rows))
	for _, row := range rows {
		countdowns = append(countdowns, storage.DaggerheartCountdown{
			CampaignID:  row.CampaignID,
			CountdownID: row.CountdownID,
			Name:        row.Name,
			Kind:        row.Kind,
			Current:     int(row.Current),
			Max:         int(row.Max),
			Direction:   row.Direction,
			Looping:     row.Looping != 0,
		})
	}

	return countdowns, nil
}

// DeleteDaggerheartCountdown removes a countdown projection for a campaign.
func (s *Store) DeleteDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(countdownID) == "" {
		return fmt.Errorf("countdown id is required")
	}

	return s.q.DeleteDaggerheartCountdown(ctx, db.DeleteDaggerheartCountdownParams{
		CampaignID:  campaignID,
		CountdownID: countdownID,
	})
}

// PutDaggerheartAdversary persists a Daggerheart adversary projection.
func (s *Store) PutDaggerheartAdversary(ctx context.Context, adversary storage.DaggerheartAdversary) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(adversary.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(adversary.AdversaryID) == "" {
		return fmt.Errorf("adversary id is required")
	}
	if strings.TrimSpace(adversary.Name) == "" {
		return fmt.Errorf("adversary name is required")
	}
	conditions := adversary.Conditions
	if conditions == nil {
		conditions = []string{}
	}
	conditionsJSON, err := json.Marshal(conditions)
	if err != nil {
		return fmt.Errorf("marshal adversary conditions: %w", err)
	}

	return s.q.PutDaggerheartAdversary(ctx, db.PutDaggerheartAdversaryParams{
		CampaignID:      adversary.CampaignID,
		AdversaryID:     adversary.AdversaryID,
		Name:            adversary.Name,
		Kind:            adversary.Kind,
		SessionID:       toNullString(adversary.SessionID),
		Notes:           adversary.Notes,
		Hp:              int64(adversary.HP),
		HpMax:           int64(adversary.HPMax),
		Stress:          int64(adversary.Stress),
		StressMax:       int64(adversary.StressMax),
		Evasion:         int64(adversary.Evasion),
		MajorThreshold:  int64(adversary.Major),
		SevereThreshold: int64(adversary.Severe),
		Armor:           int64(adversary.Armor),
		ConditionsJson:  string(conditionsJSON),
		CreatedAt:       toMillis(adversary.CreatedAt),
		UpdatedAt:       toMillis(adversary.UpdatedAt),
	})
}

// GetDaggerheartAdversary retrieves a Daggerheart adversary projection for a campaign.
func (s *Store) GetDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) (storage.DaggerheartAdversary, error) {
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartAdversary{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartAdversary{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.DaggerheartAdversary{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(adversaryID) == "" {
		return storage.DaggerheartAdversary{}, fmt.Errorf("adversary id is required")
	}

	row, err := s.q.GetDaggerheartAdversary(ctx, db.GetDaggerheartAdversaryParams{
		CampaignID:  campaignID,
		AdversaryID: adversaryID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.DaggerheartAdversary{}, storage.ErrNotFound
		}
		return storage.DaggerheartAdversary{}, fmt.Errorf("get daggerheart adversary: %w", err)
	}

	sessionID := ""
	if row.SessionID.Valid {
		sessionID = row.SessionID.String
	}
	conditions := []string{}
	if row.ConditionsJson != "" {
		if err := json.Unmarshal([]byte(row.ConditionsJson), &conditions); err != nil {
			return storage.DaggerheartAdversary{}, fmt.Errorf("decode daggerheart adversary conditions: %w", err)
		}
	}

	return storage.DaggerheartAdversary{
		CampaignID:  row.CampaignID,
		AdversaryID: row.AdversaryID,
		Name:        row.Name,
		Kind:        row.Kind,
		SessionID:   sessionID,
		Notes:       row.Notes,
		HP:          int(row.Hp),
		HPMax:       int(row.HpMax),
		Stress:      int(row.Stress),
		StressMax:   int(row.StressMax),
		Evasion:     int(row.Evasion),
		Major:       int(row.MajorThreshold),
		Severe:      int(row.SevereThreshold),
		Armor:       int(row.Armor),
		Conditions:  conditions,
		CreatedAt:   fromMillis(row.CreatedAt),
		UpdatedAt:   fromMillis(row.UpdatedAt),
	}, nil
}

// ListDaggerheartAdversaries retrieves adversary projections for a campaign.
func (s *Store) ListDaggerheartAdversaries(ctx context.Context, campaignID, sessionID string) ([]storage.DaggerheartAdversary, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}

	var rows []db.DaggerheartAdversary
	var err error
	if strings.TrimSpace(sessionID) == "" {
		rows, err = s.q.ListDaggerheartAdversariesByCampaign(ctx, campaignID)
	} else {
		rows, err = s.q.ListDaggerheartAdversariesBySession(ctx, db.ListDaggerheartAdversariesBySessionParams{
			CampaignID: campaignID,
			SessionID:  toNullString(sessionID),
		})
	}
	if err != nil {
		return nil, fmt.Errorf("list daggerheart adversaries: %w", err)
	}

	adversaries := make([]storage.DaggerheartAdversary, 0, len(rows))
	for _, row := range rows {
		rowSessionID := ""
		if row.SessionID.Valid {
			rowSessionID = row.SessionID.String
		}
		conditions := []string{}
		if row.ConditionsJson != "" {
			if err := json.Unmarshal([]byte(row.ConditionsJson), &conditions); err != nil {
				return nil, fmt.Errorf("decode daggerheart adversary conditions: %w", err)
			}
		}
		adversaries = append(adversaries, storage.DaggerheartAdversary{
			CampaignID:  row.CampaignID,
			AdversaryID: row.AdversaryID,
			Name:        row.Name,
			Kind:        row.Kind,
			SessionID:   rowSessionID,
			Notes:       row.Notes,
			HP:          int(row.Hp),
			HPMax:       int(row.HpMax),
			Stress:      int(row.Stress),
			StressMax:   int(row.StressMax),
			Evasion:     int(row.Evasion),
			Major:       int(row.MajorThreshold),
			Severe:      int(row.SevereThreshold),
			Armor:       int(row.Armor),
			Conditions:  conditions,
			CreatedAt:   fromMillis(row.CreatedAt),
			UpdatedAt:   fromMillis(row.UpdatedAt),
		})
	}

	return adversaries, nil
}

// DeleteDaggerheartAdversary removes an adversary projection for a campaign.
func (s *Store) DeleteDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(adversaryID) == "" {
		return fmt.Errorf("adversary id is required")
	}

	return s.q.DeleteDaggerheartAdversary(ctx, db.DeleteDaggerheartAdversaryParams{
		CampaignID:  campaignID,
		AdversaryID: adversaryID,
	})
}

// Daggerheart content catalog methods

// PutDaggerheartClass persists a Daggerheart class catalog entry.
func (s *Store) PutDaggerheartClass(ctx context.Context, class storage.DaggerheartClass) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(class.ID) == "" {
		return fmt.Errorf("class id is required")
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
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartClass{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartClass{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartClass{}, fmt.Errorf("class id is required")
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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
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
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("class id is required")
	}

	return s.q.DeleteDaggerheartClass(ctx, id)
}

// PutDaggerheartSubclass persists a Daggerheart subclass catalog entry.
func (s *Store) PutDaggerheartSubclass(ctx context.Context, subclass storage.DaggerheartSubclass) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(subclass.ID) == "" {
		return fmt.Errorf("subclass id is required")
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
	if err := ctx.Err(); err != nil {
		return storage.DaggerheartSubclass{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.DaggerheartSubclass{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return storage.DaggerheartSubclass{}, fmt.Errorf("subclass id is required")
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
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
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
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return fmt.Errorf("subclass id is required")
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

	if s.keyring == nil {
		return event.Event{}, fmt.Errorf("event integrity keyring is required")
	}

	hash, err := integrity.EventHash(evt)
	if err != nil {
		return event.Event{}, fmt.Errorf("compute event hash: %w", err)
	}
	if strings.TrimSpace(hash) == "" {
		return event.Event{}, fmt.Errorf("event hash is required")
	}
	evt.Hash = hash

	prevHash := ""
	if evt.Seq > 1 {
		prevRow, err := qtx.GetEventBySeq(ctx, db.GetEventBySeqParams{
			CampaignID: evt.CampaignID,
			Seq:        int64(evt.Seq - 1),
		})
		if err != nil {
			return event.Event{}, fmt.Errorf("load previous event: %w", err)
		}
		prevHash = prevRow.ChainHash
	}

	chainHash, err := integrity.ChainHash(evt, prevHash)
	if err != nil {
		return event.Event{}, fmt.Errorf("compute chain hash: %w", err)
	}
	if strings.TrimSpace(chainHash) == "" {
		return event.Event{}, fmt.Errorf("chain hash is required")
	}

	signature, keyID, err := s.keyring.SignChainHash(evt.CampaignID, chainHash)
	if err != nil {
		return event.Event{}, fmt.Errorf("sign chain hash: %w", err)
	}

	evt.PrevHash = prevHash
	evt.ChainHash = chainHash
	evt.Signature = signature
	evt.SignatureKeyID = keyID

	if err := qtx.AppendEvent(ctx, db.AppendEventParams{
		CampaignID:     evt.CampaignID,
		Seq:            int64(evt.Seq),
		EventHash:      evt.Hash,
		PrevEventHash:  prevHash,
		ChainHash:      chainHash,
		SignatureKeyID: keyID,
		EventSignature: signature,
		Timestamp:      toMillis(evt.Timestamp),
		EventType:      string(evt.Type),
		SessionID:      evt.SessionID,
		RequestID:      evt.RequestID,
		InvocationID:   evt.InvocationID,
		ActorType:      string(evt.ActorType),
		ActorID:        evt.ActorID,
		EntityType:     evt.EntityType,
		EntityID:       evt.EntityID,
		SystemID:       evt.SystemID,
		SystemVersion:  evt.SystemVersion,
		PayloadJson:    evt.PayloadJSON,
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

// VerifyEventIntegrity validates the event chain and signatures for all campaigns.
func (s *Store) VerifyEventIntegrity(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if s.keyring == nil {
		return fmt.Errorf("event integrity keyring is required")
	}

	campaignIDs, err := s.listEventCampaignIDs(ctx)
	if err != nil {
		return err
	}
	for _, campaignID := range campaignIDs {
		if err := s.verifyCampaignEvents(ctx, campaignID); err != nil {
			return err
		}
	}

	return nil
}

func (s *Store) listEventCampaignIDs(ctx context.Context) ([]string, error) {
	rows, err := s.sqlDB.QueryContext(ctx, "SELECT DISTINCT campaign_id FROM events ORDER BY campaign_id")
	if err != nil {
		return nil, fmt.Errorf("list campaign ids: %w", err)
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan campaign id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate campaign ids: %w", err)
	}
	return ids, nil
}

func (s *Store) verifyCampaignEvents(ctx context.Context, campaignID string) error {
	var lastSeq uint64
	prevChainHash := ""
	for {
		events, err := s.ListEvents(ctx, campaignID, lastSeq, 200)
		if err != nil {
			return fmt.Errorf("list events campaign_id=%s: %w", campaignID, err)
		}
		if len(events) == 0 {
			return nil
		}
		for _, evt := range events {
			if evt.Seq != lastSeq+1 {
				return fmt.Errorf("event sequence gap campaign_id=%s expected=%d got=%d", campaignID, lastSeq+1, evt.Seq)
			}
			if evt.Seq == 1 && evt.PrevHash != "" {
				return fmt.Errorf("first event prev hash must be empty campaign_id=%s", campaignID)
			}
			if evt.Seq > 1 && evt.PrevHash != prevChainHash {
				return fmt.Errorf("prev hash mismatch campaign_id=%s seq=%d", campaignID, evt.Seq)
			}

			hash, err := integrity.EventHash(evt)
			if err != nil {
				return fmt.Errorf("compute event hash campaign_id=%s seq=%d: %w", campaignID, evt.Seq, err)
			}
			if hash != evt.Hash {
				return fmt.Errorf("event hash mismatch campaign_id=%s seq=%d", campaignID, evt.Seq)
			}

			chainHash, err := integrity.ChainHash(evt, prevChainHash)
			if err != nil {
				return fmt.Errorf("compute chain hash campaign_id=%s seq=%d: %w", campaignID, evt.Seq, err)
			}
			if chainHash != evt.ChainHash {
				return fmt.Errorf("chain hash mismatch campaign_id=%s seq=%d", campaignID, evt.Seq)
			}

			if err := s.keyring.VerifyChainHash(campaignID, chainHash, evt.Signature, evt.SignatureKeyID); err != nil {
				return fmt.Errorf("signature mismatch campaign_id=%s seq=%d: %w", campaignID, evt.Seq, err)
			}

			prevChainHash = evt.ChainHash
			lastSeq = evt.Seq
		}
	}
}

func isConstraintError(err error) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) {
		return false
	}
	code := sqliteErr.Code()
	return code == sqlite3.SQLITE_CONSTRAINT || code == sqlite3.SQLITE_CONSTRAINT_UNIQUE || code == sqlite3.SQLITE_CONSTRAINT_PRIMARYKEY
}

func isParticipantUserConflict(err error) bool {
	if !isConstraintError(err) {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "idx_participants_campaign_user") ||
		(strings.Contains(message, "participant") && strings.Contains(message, "user_id"))
}

func isParticipantClaimConflict(err error) bool {
	if !isConstraintError(err) {
		return false
	}
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "participant_claims") ||
		strings.Contains(message, "idx_participant_claims")
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

	return eventRowDataToDomain(eventRowDataFromGetEventByHashRow(row))
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

	return eventRowDataToDomain(eventRowDataFromGetEventBySeqRow(row))
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

	return eventRowsToDomain(rows)
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

	return eventRowsBySessionToDomain(rows)
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
		"SELECT campaign_id, seq, event_hash, prev_event_hash, chain_hash, signature_key_id, event_signature, timestamp, event_type, session_id, request_id, invocation_id, actor_type, actor_id, entity_type, entity_id, payload_json FROM events WHERE %s %s %s",
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
			&row.PrevEventHash,
			&row.ChainHash,
			&row.SignatureKeyID,
			&row.EventSignature,
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

		evt, err := eventRowDataToDomain(eventRowDataFromEvent(row))
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

type eventRowData struct {
	CampaignID     string
	Seq            int64
	EventHash      string
	PrevEventHash  string
	ChainHash      string
	SignatureKeyID string
	EventSignature string
	Timestamp      int64
	EventType      string
	SessionID      string
	RequestID      string
	InvocationID   string
	ActorType      string
	ActorID        string
	EntityType     string
	EntityID       string
	SystemID       string
	SystemVersion  string
	PayloadJSON    []byte
}

func eventRowDataToDomain(row eventRowData) (event.Event, error) {
	return event.Event{
		CampaignID:     row.CampaignID,
		Seq:            uint64(row.Seq),
		Hash:           row.EventHash,
		PrevHash:       row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		Signature:      row.EventSignature,
		Timestamp:      fromMillis(row.Timestamp),
		Type:           event.Type(row.EventType),
		SessionID:      row.SessionID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      event.ActorType(row.ActorType),
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		PayloadJSON:    row.PayloadJSON,
	}, nil
}

func eventRowDataFromEvent(row db.Event) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowDataFromGetEventByHashRow(row db.GetEventByHashRow) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowDataFromGetEventBySeqRow(row db.GetEventBySeqRow) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowDataFromListEventsRow(row db.ListEventsRow) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowDataFromListEventsBySessionRow(row db.ListEventsBySessionRow) eventRowData {
	return eventRowData{
		CampaignID:     row.CampaignID,
		Seq:            row.Seq,
		EventHash:      row.EventHash,
		PrevEventHash:  row.PrevEventHash,
		ChainHash:      row.ChainHash,
		SignatureKeyID: row.SignatureKeyID,
		EventSignature: row.EventSignature,
		Timestamp:      row.Timestamp,
		EventType:      row.EventType,
		SessionID:      row.SessionID,
		RequestID:      row.RequestID,
		InvocationID:   row.InvocationID,
		ActorType:      row.ActorType,
		ActorID:        row.ActorID,
		EntityType:     row.EntityType,
		EntityID:       row.EntityID,
		SystemID:       row.SystemID,
		SystemVersion:  row.SystemVersion,
		PayloadJSON:    row.PayloadJson,
	}
}

func eventRowsToDomain(rows []db.ListEventsRow) ([]event.Event, error) {
	events := make([]event.Event, 0, len(rows))
	for _, row := range rows {
		evt, err := eventRowDataToDomain(eventRowDataFromListEventsRow(row))
		if err != nil {
			return nil, err
		}
		events = append(events, evt)
	}
	return events, nil
}

func eventRowsBySessionToDomain(rows []db.ListEventsBySessionRow) ([]event.Event, error) {
	events := make([]event.Event, 0, len(rows))
	for _, row := range rows {
		evt, err := eventRowDataToDomain(eventRowDataFromListEventsBySessionRow(row))
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
