package daggerheartcontent

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

func (s *Store) PutDaggerheartAdversaryEntry(ctx context.Context, adversary contentstore.DaggerheartAdversaryEntry) error {
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
	minionRuleJSON := ""
	if adversary.MinionRule != nil {
		rawMinionRuleJSON, err := json.Marshal(adversary.MinionRule)
		if err != nil {
			return fmt.Errorf("marshal adversary minion rule: %w", err)
		}
		minionRuleJSON = string(rawMinionRuleJSON)
	}
	hordeRuleJSON := ""
	if adversary.HordeRule != nil {
		rawHordeRuleJSON, err := json.Marshal(adversary.HordeRule)
		if err != nil {
			return fmt.Errorf("marshal adversary horde rule: %w", err)
		}
		hordeRuleJSON = string(rawHordeRuleJSON)
	}
	relentlessRuleJSON := ""
	if adversary.RelentlessRule != nil {
		rawRelentlessRuleJSON, err := json.Marshal(adversary.RelentlessRule)
		if err != nil {
			return fmt.Errorf("marshal adversary relentless rule: %w", err)
		}
		relentlessRuleJSON = string(rawRelentlessRuleJSON)
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
		MinionRuleJson:     minionRuleJSON,
		HordeRuleJson:      hordeRuleJSON,
		RelentlessRuleJson: relentlessRuleJSON,
		CreatedAt:          toMillis(adversary.CreatedAt),
		UpdatedAt:          toMillis(adversary.UpdatedAt),
	})
}

// GetDaggerheartAdversaryEntry retrieves a Daggerheart adversary catalog entry.
func (s *Store) GetDaggerheartAdversaryEntry(ctx context.Context, id string) (contentstore.DaggerheartAdversaryEntry, error) {
	if err := ctx.Err(); err != nil {
		return contentstore.DaggerheartAdversaryEntry{}, err
	}
	if s == nil || s.sqlDB == nil {
		return contentstore.DaggerheartAdversaryEntry{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return contentstore.DaggerheartAdversaryEntry{}, fmt.Errorf("adversary id is required")
	}

	row, err := s.q.GetDaggerheartAdversaryEntry(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contentstore.DaggerheartAdversaryEntry{}, storage.ErrNotFound
		}
		return contentstore.DaggerheartAdversaryEntry{}, fmt.Errorf("get daggerheart adversary: %w", err)
	}

	return dbDaggerheartAdversaryEntryToStorage(row)
}

// ListDaggerheartAdversaryEntries lists all Daggerheart adversary catalog entries.
func (s *Store) ListDaggerheartAdversaryEntries(ctx context.Context) ([]contentstore.DaggerheartAdversaryEntry, error) {
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

	adversaries := make([]contentstore.DaggerheartAdversaryEntry, 0, len(rows))
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
func (s *Store) PutDaggerheartBeastform(ctx context.Context, beastform contentstore.DaggerheartBeastformEntry) error {
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
func (s *Store) GetDaggerheartBeastform(ctx context.Context, id string) (contentstore.DaggerheartBeastformEntry, error) {
	if err := ctx.Err(); err != nil {
		return contentstore.DaggerheartBeastformEntry{}, err
	}
	if s == nil || s.sqlDB == nil {
		return contentstore.DaggerheartBeastformEntry{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return contentstore.DaggerheartBeastformEntry{}, fmt.Errorf("beastform id is required")
	}

	row, err := s.q.GetDaggerheartBeastform(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contentstore.DaggerheartBeastformEntry{}, storage.ErrNotFound
		}
		return contentstore.DaggerheartBeastformEntry{}, fmt.Errorf("get daggerheart beastform: %w", err)
	}

	return dbDaggerheartBeastformToStorage(row)
}

// ListDaggerheartBeastforms lists all Daggerheart beastform catalog entries.
func (s *Store) ListDaggerheartBeastforms(ctx context.Context) ([]contentstore.DaggerheartBeastformEntry, error) {
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

	beastforms := make([]contentstore.DaggerheartBeastformEntry, 0, len(rows))
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
func (s *Store) PutDaggerheartCompanionExperience(ctx context.Context, experience contentstore.DaggerheartCompanionExperienceEntry) error {
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
func (s *Store) GetDaggerheartCompanionExperience(ctx context.Context, id string) (contentstore.DaggerheartCompanionExperienceEntry, error) {
	if err := ctx.Err(); err != nil {
		return contentstore.DaggerheartCompanionExperienceEntry{}, err
	}
	if s == nil || s.sqlDB == nil {
		return contentstore.DaggerheartCompanionExperienceEntry{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return contentstore.DaggerheartCompanionExperienceEntry{}, fmt.Errorf("companion experience id is required")
	}

	row, err := s.q.GetDaggerheartCompanionExperience(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return contentstore.DaggerheartCompanionExperienceEntry{}, storage.ErrNotFound
		}
		return contentstore.DaggerheartCompanionExperienceEntry{}, fmt.Errorf("get daggerheart companion experience: %w", err)
	}

	return dbDaggerheartCompanionExperienceToStorage(row), nil
}

// ListDaggerheartCompanionExperiences lists all Daggerheart companion experience catalog entries.
func (s *Store) ListDaggerheartCompanionExperiences(ctx context.Context) ([]contentstore.DaggerheartCompanionExperienceEntry, error) {
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

	experiences := make([]contentstore.DaggerheartCompanionExperienceEntry, 0, len(rows))
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
