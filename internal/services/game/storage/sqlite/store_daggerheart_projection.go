package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

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
	var temporaryArmor []storage.DaggerheartTemporaryArmor
	if row.TemporaryArmorJson != "" {
		if err := json.Unmarshal([]byte(row.TemporaryArmorJson), &temporaryArmor); err != nil {
			return storage.DaggerheartCharacterState{}, fmt.Errorf("decode temporary armor: %w", err)
		}
	}

	lifeState := row.LifeState
	if strings.TrimSpace(lifeState) == "" {
		lifeState = daggerheart.LifeStateAlive
	}

	return storage.DaggerheartCharacterState{
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
