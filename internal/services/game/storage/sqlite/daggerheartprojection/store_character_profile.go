package daggerheartprojection

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

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
	subclassTracksJSON, err := json.Marshal(profile.SubclassTracks)
	if err != nil {
		return fmt.Errorf("marshal subclass tracks: %w", err)
	}
	subclassCreationRequirementsJSON, err := json.Marshal(profile.SubclassCreationRequirements)
	if err != nil {
		return fmt.Errorf("marshal subclass creation requirements: %w", err)
	}
	heritageJSON, err := json.Marshal(profile.Heritage)
	if err != nil {
		return fmt.Errorf("marshal heritage: %w", err)
	}
	companionSheetJSON := ""
	if profile.CompanionSheet != nil {
		rawCompanionSheetJSON, err := json.Marshal(profile.CompanionSheet)
		if err != nil {
			return fmt.Errorf("marshal companion sheet: %w", err)
		}
		companionSheetJSON = string(rawCompanionSheetJSON)
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
		CampaignID:                       profile.CampaignID,
		CharacterID:                      profile.CharacterID,
		Level:                            int64(profile.Level),
		HpMax:                            int64(profile.HpMax),
		StressMax:                        int64(profile.StressMax),
		Evasion:                          int64(profile.Evasion),
		MajorThreshold:                   int64(profile.MajorThreshold),
		SevereThreshold:                  int64(profile.SevereThreshold),
		Proficiency:                      int64(profile.Proficiency),
		ArmorScore:                       int64(profile.ArmorScore),
		ArmorMax:                         int64(profile.ArmorMax),
		ExperiencesJson:                  string(experiencesJSON),
		ClassID:                          profile.ClassID,
		SubclassID:                       profile.SubclassID,
		SubclassTracksJson:               string(subclassTracksJSON),
		SubclassCreationRequirementsJson: string(subclassCreationRequirementsJSON),
		HeritageJson:                     string(heritageJSON),
		CompanionSheetJson:               companionSheetJSON,
		EquippedArmorID:                  profile.EquippedArmorID,
		SpellcastRollBonus:               int64(profile.SpellcastRollBonus),
		TraitsAssigned:                   traitsAssigned,
		DetailsRecorded:                  detailsRecorded,
		StartingWeaponIdsJson:            string(startingWeaponIDsJSON),
		StartingArmorID:                  profile.StartingArmorID,
		StartingPotionItemID:             profile.StartingPotionItemID,
		Background:                       profile.Background,
		Description:                      profile.Description,
		DomainCardIdsJson:                string(domainCardIDsJSON),
		Connections:                      profile.Connections,
		Agility:                          int64(profile.Agility),
		Strength:                         int64(profile.Strength),
		Finesse:                          int64(profile.Finesse),
		Instinct:                         int64(profile.Instinct),
		Presence:                         int64(profile.Presence),
		Knowledge:                        int64(profile.Knowledge),
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
		EquippedArmorID:      row.EquippedArmorID,
		SpellcastRollBonus:   int(row.SpellcastRollBonus),
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
	if row.SubclassCreationRequirementsJson != "" {
		if err := json.Unmarshal([]byte(row.SubclassCreationRequirementsJson), &profile.SubclassCreationRequirements); err != nil {
			return projectionstore.DaggerheartCharacterProfile{}, fmt.Errorf("decode subclass creation requirements: %w", err)
		}
	}
	if row.SubclassTracksJson != "" {
		if err := json.Unmarshal([]byte(row.SubclassTracksJson), &profile.SubclassTracks); err != nil {
			return projectionstore.DaggerheartCharacterProfile{}, fmt.Errorf("decode subclass tracks: %w", err)
		}
	}
	if row.HeritageJson != "" {
		if err := json.Unmarshal([]byte(row.HeritageJson), &profile.Heritage); err != nil {
			return projectionstore.DaggerheartCharacterProfile{}, fmt.Errorf("decode heritage: %w", err)
		}
	}
	if row.CompanionSheetJson != "" {
		var companion projectionstore.DaggerheartCompanionSheet
		if err := json.Unmarshal([]byte(row.CompanionSheetJson), &companion); err != nil {
			return projectionstore.DaggerheartCharacterProfile{}, fmt.Errorf("decode companion sheet: %w", err)
		}
		profile.CompanionSheet = &companion
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
