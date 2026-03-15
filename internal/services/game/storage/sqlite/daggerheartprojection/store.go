package daggerheartprojection

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

// Store provides the SQLite-backed Daggerheart projection backend.
//
// This backend binds to the root projections store's shared `sql.DB` and
// `db.Queries` handles so system adapters can rebind inside exact-once
// projection transactions without opening a second database.
type Store struct {
	sqlDB *sql.DB
	q     *db.Queries
}

var _ projectionstore.Store = (*Store)(nil)

// Bind creates a Daggerheart projection backend from an existing projections DB
// handle and query bundle.
func Bind(sqlDB *sql.DB, q *db.Queries) *Store {
	if sqlDB == nil || q == nil {
		return nil
	}
	return &Store{
		sqlDB: sqlDB,
		q:     q,
	}
}

// PutDaggerheartCharacterProfile persists a Daggerheart character profile extension.
func (s *Store) PutDaggerheartCharacterProfile(ctx context.Context, profile projectionstore.DaggerheartCharacterProfile) error {
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
	domainCardIDsJSON, err := json.Marshal(profile.DomainCardIDs)
	if err != nil {
		return fmt.Errorf("marshal domain card ids: %w", err)
	}
	startingWeaponIDsJSON, err := json.Marshal(profile.StartingWeaponIDs)
	if err != nil {
		return fmt.Errorf("marshal starting weapon ids: %w", err)
	}
	traitsAssigned := int64(0)
	if profile.TraitsAssigned {
		traitsAssigned = 1
	}
	detailsRecorded := int64(0)
	if profile.DetailsRecorded {
		detailsRecorded = 1
	}

	return s.q.PutDaggerheartCharacterProfile(ctx, db.PutDaggerheartCharacterProfileParams{
		CampaignID:            profile.CampaignID,
		CharacterID:           profile.CharacterID,
		Level:                 int64(profile.Level),
		HpMax:                 int64(profile.HpMax),
		StressMax:             int64(profile.StressMax),
		Evasion:               int64(profile.Evasion),
		MajorThreshold:        int64(profile.MajorThreshold),
		SevereThreshold:       int64(profile.SevereThreshold),
		Proficiency:           int64(profile.Proficiency),
		ArmorScore:            int64(profile.ArmorScore),
		ArmorMax:              int64(profile.ArmorMax),
		ExperiencesJson:       string(experiencesJSON),
		ClassID:               profile.ClassID,
		SubclassID:            profile.SubclassID,
		AncestryID:            profile.AncestryID,
		CommunityID:           profile.CommunityID,
		TraitsAssigned:        traitsAssigned,
		DetailsRecorded:       detailsRecorded,
		StartingWeaponIdsJson: string(startingWeaponIDsJSON),
		StartingArmorID:       profile.StartingArmorID,
		StartingPotionItemID:  profile.StartingPotionItemID,
		Background:            profile.Background,
		Description:           profile.Description,
		DomainCardIdsJson:     string(domainCardIDsJSON),
		Connections:           profile.Connections,
		Agility:               int64(profile.Agility),
		Strength:              int64(profile.Strength),
		Finesse:               int64(profile.Finesse),
		Instinct:              int64(profile.Instinct),
		Presence:              int64(profile.Presence),
		Knowledge:             int64(profile.Knowledge),
	})
}

// GetDaggerheartCharacterProfile retrieves a Daggerheart character profile extension.
func (s *Store) GetDaggerheartCharacterProfile(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterProfile, error) {
	if err := ctx.Err(); err != nil {
		return projectionstore.DaggerheartCharacterProfile{}, err
	}
	if s == nil || s.sqlDB == nil {
		return projectionstore.DaggerheartCharacterProfile{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return projectionstore.DaggerheartCharacterProfile{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(characterID) == "" {
		return projectionstore.DaggerheartCharacterProfile{}, fmt.Errorf("character id is required")
	}

	row, err := s.q.GetDaggerheartCharacterProfile(ctx, db.GetDaggerheartCharacterProfileParams{
		CampaignID:  campaignID,
		CharacterID: characterID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return projectionstore.DaggerheartCharacterProfile{}, storage.ErrNotFound
		}
		return projectionstore.DaggerheartCharacterProfile{}, fmt.Errorf("get daggerheart character profile: %w", err)
	}

	return dbDaggerheartCharacterProfileToDomain(row)
}

// ListDaggerheartCharacterProfiles retrieves a page of Daggerheart character profiles.
func (s *Store) ListDaggerheartCharacterProfiles(ctx context.Context, campaignID string, pageSize int, pageToken string) (projectionstore.DaggerheartCharacterProfilePage, error) {
	if err := ctx.Err(); err != nil {
		return projectionstore.DaggerheartCharacterProfilePage{}, err
	}
	if s == nil || s.sqlDB == nil {
		return projectionstore.DaggerheartCharacterProfilePage{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return projectionstore.DaggerheartCharacterProfilePage{}, fmt.Errorf("campaign id is required")
	}
	if pageSize <= 0 {
		return projectionstore.DaggerheartCharacterProfilePage{}, fmt.Errorf("page size must be greater than zero")
	}

	var rows []db.DaggerheartCharacterProfile
	var err error
	if pageToken == "" {
		rows, err = s.q.ListDaggerheartCharacterProfilesPagedFirst(ctx, db.ListDaggerheartCharacterProfilesPagedFirstParams{
			CampaignID: campaignID,
			Limit:      int64(pageSize + 1),
		})
	} else {
		rows, err = s.q.ListDaggerheartCharacterProfilesPaged(ctx, db.ListDaggerheartCharacterProfilesPagedParams{
			CampaignID:  campaignID,
			CharacterID: pageToken,
			Limit:       int64(pageSize + 1),
		})
	}
	if err != nil {
		return projectionstore.DaggerheartCharacterProfilePage{}, fmt.Errorf("list daggerheart character profiles: %w", err)
	}

	profiles, nextPageToken, err := mapPageRows(rows, pageSize, func(row db.DaggerheartCharacterProfile) string {
		return row.CharacterID
	}, dbDaggerheartCharacterProfileToDomain)
	if err != nil {
		return projectionstore.DaggerheartCharacterProfilePage{}, err
	}

	return projectionstore.DaggerheartCharacterProfilePage{
		Profiles:      profiles,
		NextPageToken: nextPageToken,
	}, nil
}

// DeleteDaggerheartCharacterProfile deletes a Daggerheart character profile extension.
func (s *Store) DeleteDaggerheartCharacterProfile(ctx context.Context, campaignID, characterID string) error {
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

	return s.q.DeleteDaggerheartCharacterProfile(ctx, db.DeleteDaggerheartCharacterProfileParams{
		CampaignID:  campaignID,
		CharacterID: characterID,
	})
}

func dbDaggerheartCharacterProfileToDomain(row db.DaggerheartCharacterProfile) (projectionstore.DaggerheartCharacterProfile, error) {
	profile := projectionstore.DaggerheartCharacterProfile{
		CampaignID:           row.CampaignID,
		CharacterID:          row.CharacterID,
		Level:                int(row.Level),
		HpMax:                int(row.HpMax),
		StressMax:            int(row.StressMax),
		Evasion:              int(row.Evasion),
		MajorThreshold:       int(row.MajorThreshold),
		SevereThreshold:      int(row.SevereThreshold),
		Proficiency:          int(row.Proficiency),
		ArmorScore:           int(row.ArmorScore),
		ArmorMax:             int(row.ArmorMax),
		ClassID:              row.ClassID,
		SubclassID:           row.SubclassID,
		AncestryID:           row.AncestryID,
		CommunityID:          row.CommunityID,
		TraitsAssigned:       row.TraitsAssigned != 0,
		DetailsRecorded:      row.DetailsRecorded != 0,
		StartingArmorID:      row.StartingArmorID,
		StartingPotionItemID: row.StartingPotionItemID,
		Background:           row.Background,
		Description:          row.Description,
		Connections:          row.Connections,
		Agility:              int(row.Agility),
		Strength:             int(row.Strength),
		Finesse:              int(row.Finesse),
		Instinct:             int(row.Instinct),
		Presence:             int(row.Presence),
		Knowledge:            int(row.Knowledge),
	}
	if row.ExperiencesJson != "" {
		if err := json.Unmarshal([]byte(row.ExperiencesJson), &profile.Experiences); err != nil {
			return projectionstore.DaggerheartCharacterProfile{}, fmt.Errorf("decode experiences: %w", err)
		}
	}
	if row.DomainCardIdsJson != "" {
		if err := json.Unmarshal([]byte(row.DomainCardIdsJson), &profile.DomainCardIDs); err != nil {
			return projectionstore.DaggerheartCharacterProfile{}, fmt.Errorf("decode domain card ids: %w", err)
		}
	}
	if row.StartingWeaponIdsJson != "" {
		if err := json.Unmarshal([]byte(row.StartingWeaponIdsJson), &profile.StartingWeaponIDs); err != nil {
			return projectionstore.DaggerheartCharacterProfile{}, fmt.Errorf("decode starting weapon ids: %w", err)
		}
	}
	return profile, nil
}

// PutDaggerheartCharacterState persists a Daggerheart character state extension.
func (s *Store) PutDaggerheartCharacterState(ctx context.Context, state projectionstore.DaggerheartCharacterState) error {
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
	temporaryArmorJSON, err := json.Marshal(state.TemporaryArmor)
	if err != nil {
		return fmt.Errorf("encode temporary armor: %w", err)
	}

	hopeMax := state.HopeMax
	if hopeMax == 0 {
		hopeMax = daggerheart.HopeMaxDefault
	}

	lifeState := state.LifeState
	if strings.TrimSpace(lifeState) == "" {
		lifeState = daggerheart.LifeStateAlive
	}

	return s.q.PutDaggerheartCharacterState(ctx, db.PutDaggerheartCharacterStateParams{
		CampaignID:         state.CampaignID,
		CharacterID:        state.CharacterID,
		Hp:                 int64(state.Hp),
		Hope:               int64(state.Hope),
		HopeMax:            int64(hopeMax),
		Stress:             int64(state.Stress),
		Armor:              int64(state.Armor),
		ConditionsJson:     string(conditionsJSON),
		TemporaryArmorJson: string(temporaryArmorJSON),
		LifeState:          lifeState,
	})
}

// GetDaggerheartCharacterState retrieves a Daggerheart character state extension.
func (s *Store) GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error) {
	if err := ctx.Err(); err != nil {
		return projectionstore.DaggerheartCharacterState{}, err
	}
	if s == nil || s.sqlDB == nil {
		return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(characterID) == "" {
		return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("character id is required")
	}

	row, err := s.q.GetDaggerheartCharacterState(ctx, db.GetDaggerheartCharacterStateParams{
		CampaignID:  campaignID,
		CharacterID: characterID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return projectionstore.DaggerheartCharacterState{}, storage.ErrNotFound
		}
		return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("get daggerheart character state: %w", err)
	}

	var conditions []string
	if row.ConditionsJson != "" {
		if err := json.Unmarshal([]byte(row.ConditionsJson), &conditions); err != nil {
			return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("decode conditions: %w", err)
		}
	}
	var temporaryArmor []projectionstore.DaggerheartTemporaryArmor
	if row.TemporaryArmorJson != "" {
		if err := json.Unmarshal([]byte(row.TemporaryArmorJson), &temporaryArmor); err != nil {
			return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("decode temporary armor: %w", err)
		}
	}

	lifeState := row.LifeState
	if strings.TrimSpace(lifeState) == "" {
		lifeState = daggerheart.LifeStateAlive
	}

	return projectionstore.DaggerheartCharacterState{
		CampaignID:     row.CampaignID,
		CharacterID:    row.CharacterID,
		Hp:             int(row.Hp),
		Hope:           int(row.Hope),
		HopeMax:        int(row.HopeMax),
		Stress:         int(row.Stress),
		Armor:          int(row.Armor),
		TemporaryArmor: temporaryArmor,
		Conditions:     conditions,
		LifeState:      lifeState,
	}, nil
}

// PutDaggerheartSnapshot persists a Daggerheart snapshot projection.
func (s *Store) PutDaggerheartSnapshot(ctx context.Context, snap projectionstore.DaggerheartSnapshot) error {
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
func (s *Store) GetDaggerheartSnapshot(ctx context.Context, campaignID string) (projectionstore.DaggerheartSnapshot, error) {
	if err := ctx.Err(); err != nil {
		return projectionstore.DaggerheartSnapshot{}, err
	}
	if s == nil || s.sqlDB == nil {
		return projectionstore.DaggerheartSnapshot{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return projectionstore.DaggerheartSnapshot{}, fmt.Errorf("campaign id is required")
	}

	row, err := s.q.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// Return zero-value for not found (consistent with GetGmFear behavior)
			return projectionstore.DaggerheartSnapshot{CampaignID: campaignID, GMFear: 0, ConsecutiveShortRests: 0}, nil
		}
		return projectionstore.DaggerheartSnapshot{}, fmt.Errorf("get daggerheart snapshot: %w", err)
	}

	return projectionstore.DaggerheartSnapshot{
		CampaignID:            row.CampaignID,
		GMFear:                int(row.GmFear),
		ConsecutiveShortRests: int(row.ConsecutiveShortRests),
	}, nil
}

// PutDaggerheartCountdown persists a Daggerheart countdown projection.
func (s *Store) PutDaggerheartCountdown(ctx context.Context, countdown projectionstore.DaggerheartCountdown) error {
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
func (s *Store) GetDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) (projectionstore.DaggerheartCountdown, error) {
	if err := ctx.Err(); err != nil {
		return projectionstore.DaggerheartCountdown{}, err
	}
	if s == nil || s.sqlDB == nil {
		return projectionstore.DaggerheartCountdown{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return projectionstore.DaggerheartCountdown{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(countdownID) == "" {
		return projectionstore.DaggerheartCountdown{}, fmt.Errorf("countdown id is required")
	}

	row, err := s.q.GetDaggerheartCountdown(ctx, db.GetDaggerheartCountdownParams{
		CampaignID:  campaignID,
		CountdownID: countdownID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return projectionstore.DaggerheartCountdown{}, storage.ErrNotFound
		}
		return projectionstore.DaggerheartCountdown{}, fmt.Errorf("get daggerheart countdown: %w", err)
	}

	return projectionstore.DaggerheartCountdown{
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
func (s *Store) ListDaggerheartCountdowns(ctx context.Context, campaignID string) ([]projectionstore.DaggerheartCountdown, error) {
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

	countdowns := make([]projectionstore.DaggerheartCountdown, 0, len(rows))
	for _, row := range rows {
		countdowns = append(countdowns, projectionstore.DaggerheartCountdown{
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
func (s *Store) PutDaggerheartAdversary(ctx context.Context, adversary projectionstore.DaggerheartAdversary) error {
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
func (s *Store) GetDaggerheartAdversary(ctx context.Context, campaignID, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	if err := ctx.Err(); err != nil {
		return projectionstore.DaggerheartAdversary{}, err
	}
	if s == nil || s.sqlDB == nil {
		return projectionstore.DaggerheartAdversary{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return projectionstore.DaggerheartAdversary{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(adversaryID) == "" {
		return projectionstore.DaggerheartAdversary{}, fmt.Errorf("adversary id is required")
	}

	row, err := s.q.GetDaggerheartAdversary(ctx, db.GetDaggerheartAdversaryParams{
		CampaignID:  campaignID,
		AdversaryID: adversaryID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return projectionstore.DaggerheartAdversary{}, storage.ErrNotFound
		}
		return projectionstore.DaggerheartAdversary{}, fmt.Errorf("get daggerheart adversary: %w", err)
	}

	sessionID := ""
	if row.SessionID.Valid {
		sessionID = row.SessionID.String
	}
	conditions := []string{}
	if row.ConditionsJson != "" {
		if err := json.Unmarshal([]byte(row.ConditionsJson), &conditions); err != nil {
			return projectionstore.DaggerheartAdversary{}, fmt.Errorf("decode daggerheart adversary conditions: %w", err)
		}
	}

	return projectionstore.DaggerheartAdversary{
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
func (s *Store) ListDaggerheartAdversaries(ctx context.Context, campaignID, sessionID string) ([]projectionstore.DaggerheartAdversary, error) {
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

	adversaries := make([]projectionstore.DaggerheartAdversary, 0, len(rows))
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
		adversaries = append(adversaries, projectionstore.DaggerheartAdversary{
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

func toMillis(value time.Time) int64 {
	return value.UTC().UnixMilli()
}

func fromMillis(value int64) time.Time {
	return time.UnixMilli(value).UTC()
}

func toNullString(value string) sql.NullString {
	if strings.TrimSpace(value) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: value, Valid: true}
}

func mapPageRows[Row any, Item any](
	rows []Row,
	pageSize int,
	rowID func(Row) string,
	mapRow func(Row) (Item, error),
) ([]Item, string, error) {
	capHint := pageSize
	if capHint > len(rows) {
		capHint = len(rows)
	}
	items := make([]Item, 0, capHint)

	for i, row := range rows {
		if i >= pageSize {
			return items, rowID(rows[pageSize-1]), nil
		}
		item, err := mapRow(row)
		if err != nil {
			return nil, "", err
		}
		items = append(items, item)
	}

	return items, "", nil
}
