package daggerheartprojection

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/rules"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
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

func domainConditionStatesToProjection(values []rules.ConditionState) []projectionstore.DaggerheartConditionState {
	if len(values) == 0 {
		return []projectionstore.DaggerheartConditionState{}
	}
	items := make([]projectionstore.DaggerheartConditionState, 0, len(values))
	for _, value := range values {
		triggers := make([]string, 0, len(value.ClearTriggers))
		for _, trigger := range value.ClearTriggers {
			triggers = append(triggers, string(trigger))
		}
		items = append(items, projectionstore.DaggerheartConditionState{
			ID:            value.ID,
			Class:         string(value.Class),
			Standard:      value.Standard,
			Code:          value.Code,
			Label:         value.Label,
			Source:        value.Source,
			SourceID:      value.SourceID,
			ClearTriggers: triggers,
		})
	}
	return items
}

func projectionConditionStatesToDomain(values []projectionstore.DaggerheartConditionState) []rules.ConditionState {
	if len(values) == 0 {
		return []rules.ConditionState{}
	}
	items := make([]rules.ConditionState, 0, len(values))
	for _, value := range values {
		triggers := make([]rules.ConditionClearTrigger, 0, len(value.ClearTriggers))
		for _, trigger := range value.ClearTriggers {
			triggers = append(triggers, rules.ConditionClearTrigger(trigger))
		}
		items = append(items, rules.ConditionState{
			ID:            value.ID,
			Class:         rules.ConditionClass(value.Class),
			Standard:      value.Standard,
			Code:          value.Code,
			Label:         value.Label,
			Source:        value.Source,
			SourceID:      value.SourceID,
			ClearTriggers: triggers,
		})
	}
	return items
}

func decodeProjectionConditionStates(raw string) ([]projectionstore.DaggerheartConditionState, error) {
	if strings.TrimSpace(raw) == "" {
		return []projectionstore.DaggerheartConditionState{}, nil
	}
	var structured []projectionstore.DaggerheartConditionState
	if err := json.Unmarshal([]byte(raw), &structured); err == nil {
		return structured, nil
	}
	var legacy []string
	if err := json.Unmarshal([]byte(raw), &legacy); err != nil {
		return nil, err
	}
	items := make([]projectionstore.DaggerheartConditionState, 0, len(legacy))
	for _, code := range legacy {
		state, err := rules.StandardConditionState(code)
		if err != nil {
			return nil, err
		}
		items = append(items, domainConditionStatesToProjection([]rules.ConditionState{state})...)
	}
	return items, nil
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
		conditions = []projectionstore.DaggerheartConditionState{}
	}
	conditionsJSON, err := json.Marshal(conditions)
	if err != nil {
		return fmt.Errorf("encode conditions: %w", err)
	}
	temporaryArmorJSON, err := json.Marshal(state.TemporaryArmor)
	if err != nil {
		return fmt.Errorf("encode temporary armor: %w", err)
	}
	classStateJSON, err := json.Marshal(state.ClassState)
	if err != nil {
		return fmt.Errorf("encode class state: %w", err)
	}
	subclassStateJSON, err := json.Marshal(state.SubclassState)
	if err != nil {
		return fmt.Errorf("encode subclass state: %w", err)
	}
	companionStateJSON, err := json.Marshal(state.CompanionState)
	if err != nil {
		return fmt.Errorf("encode companion state: %w", err)
	}
	statModifiers := state.StatModifiers
	if statModifiers == nil {
		statModifiers = []projectionstore.DaggerheartStatModifier{}
	}
	statModifiersJSON, err := json.Marshal(statModifiers)
	if err != nil {
		return fmt.Errorf("encode stat modifiers: %w", err)
	}

	hopeMax := state.HopeMax
	if hopeMax == 0 {
		hopeMax = daggerheartstate.HopeMaxDefault
	}

	lifeState := state.LifeState
	if strings.TrimSpace(lifeState) == "" {
		lifeState = daggerheartstate.LifeStateAlive
	}

	return s.q.PutDaggerheartCharacterState(ctx, db.PutDaggerheartCharacterStateParams{
		CampaignID:                    state.CampaignID,
		CharacterID:                   state.CharacterID,
		Hp:                            int64(state.Hp),
		Hope:                          int64(state.Hope),
		HopeMax:                       int64(hopeMax),
		Stress:                        int64(state.Stress),
		Armor:                         int64(state.Armor),
		ConditionsJson:                string(conditionsJSON),
		TemporaryArmorJson:            string(temporaryArmorJSON),
		LifeState:                     lifeState,
		ClassStateJson:                string(classStateJSON),
		SubclassStateJson:             string(subclassStateJSON),
		CompanionStateJson:            string(companionStateJSON),
		ImpenetrableUsedThisShortRest: boolToInt64(state.ImpenetrableUsedThisShortRest),
		StatModifiersJson:             string(statModifiersJSON),
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

	var conditions []projectionstore.DaggerheartConditionState
	if row.ConditionsJson != "" {
		conditions, err = decodeProjectionConditionStates(row.ConditionsJson)
		if err != nil {
			return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("decode conditions: %w", err)
		}
	}
	var temporaryArmor []projectionstore.DaggerheartTemporaryArmor
	if row.TemporaryArmorJson != "" {
		if err := json.Unmarshal([]byte(row.TemporaryArmorJson), &temporaryArmor); err != nil {
			return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("decode temporary armor: %w", err)
		}
	}
	var classState projectionstore.DaggerheartClassState
	if row.ClassStateJson != "" {
		if err := json.Unmarshal([]byte(row.ClassStateJson), &classState); err != nil {
			return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("decode class state: %w", err)
		}
	}
	var subclassState *projectionstore.DaggerheartSubclassState
	if row.SubclassStateJson != "" && row.SubclassStateJson != "null" && row.SubclassStateJson != "{}" {
		var decoded projectionstore.DaggerheartSubclassState
		if err := json.Unmarshal([]byte(row.SubclassStateJson), &decoded); err != nil {
			return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("decode subclass state: %w", err)
		}
		subclassState = &decoded
	}
	var companionState *projectionstore.DaggerheartCompanionState
	if row.CompanionStateJson != "" && row.CompanionStateJson != "null" && row.CompanionStateJson != "{}" {
		var decoded projectionstore.DaggerheartCompanionState
		if err := json.Unmarshal([]byte(row.CompanionStateJson), &decoded); err != nil {
			return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("decode companion state: %w", err)
		}
		companionState = &decoded
	}

	lifeState := row.LifeState
	if strings.TrimSpace(lifeState) == "" {
		lifeState = daggerheartstate.LifeStateAlive
	}
	var statModifiers []projectionstore.DaggerheartStatModifier
	if row.StatModifiersJson != "" && row.StatModifiersJson != "[]" {
		if err := json.Unmarshal([]byte(row.StatModifiersJson), &statModifiers); err != nil {
			return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("decode stat modifiers: %w", err)
		}
	}

	return projectionstore.DaggerheartCharacterState{
		CampaignID:                    row.CampaignID,
		CharacterID:                   row.CharacterID,
		Hp:                            int(row.Hp),
		Hope:                          int(row.Hope),
		HopeMax:                       int(row.HopeMax),
		Stress:                        int(row.Stress),
		Armor:                         int(row.Armor),
		TemporaryArmor:                temporaryArmor,
		Conditions:                    conditions,
		LifeState:                     lifeState,
		ClassState:                    classState,
		SubclassState:                 subclassState,
		CompanionState:                companionState,
		ImpenetrableUsedThisShortRest: row.ImpenetrableUsedThisShortRest != 0,
		StatModifiers:                 statModifiers,
	}, nil
}

func boolToInt64(value bool) int64 {
	if value {
		return 1
	}
	return 0
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
	if strings.TrimSpace(adversary.AdversaryEntryID) == "" {
		return fmt.Errorf("adversary entry id is required")
	}
	if strings.TrimSpace(adversary.Name) == "" {
		return fmt.Errorf("adversary name is required")
	}
	if strings.TrimSpace(adversary.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}
	conditions := adversary.Conditions
	if conditions == nil {
		conditions = []projectionstore.DaggerheartConditionState{}
	}
	conditionsJSON, err := json.Marshal(conditions)
	if err != nil {
		return fmt.Errorf("marshal adversary conditions: %w", err)
	}
	featureStates := adversary.FeatureStates
	if featureStates == nil {
		featureStates = []projectionstore.DaggerheartAdversaryFeatureState{}
	}
	featureStatesJSON, err := json.Marshal(featureStates)
	if err != nil {
		return fmt.Errorf("marshal adversary feature states: %w", err)
	}
	pendingExperienceJSON := ""
	if adversary.PendingExperience != nil {
		payloadJSON, err := json.Marshal(adversary.PendingExperience)
		if err != nil {
			return fmt.Errorf("marshal adversary pending experience: %w", err)
		}
		pendingExperienceJSON = string(payloadJSON)
	}

	return s.q.PutDaggerheartAdversary(ctx, db.PutDaggerheartAdversaryParams{
		CampaignID:            adversary.CampaignID,
		AdversaryID:           adversary.AdversaryID,
		AdversaryEntryID:      adversary.AdversaryEntryID,
		Name:                  adversary.Name,
		Kind:                  adversary.Kind,
		SessionID:             adversary.SessionID,
		SceneID:               adversary.SceneID,
		Notes:                 adversary.Notes,
		Hp:                    int64(adversary.HP),
		HpMax:                 int64(adversary.HPMax),
		Stress:                int64(adversary.Stress),
		StressMax:             int64(adversary.StressMax),
		Evasion:               int64(adversary.Evasion),
		MajorThreshold:        int64(adversary.Major),
		SevereThreshold:       int64(adversary.Severe),
		Armor:                 int64(adversary.Armor),
		ConditionsJson:        string(conditionsJSON),
		FeatureStateJson:      string(featureStatesJSON),
		PendingExperienceJson: pendingExperienceJSON,
		SpotlightGateID:       adversary.SpotlightGateID,
		SpotlightCount:        int64(adversary.SpotlightCount),
		CreatedAt:             toMillis(adversary.CreatedAt),
		UpdatedAt:             toMillis(adversary.UpdatedAt),
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

	conditions := []projectionstore.DaggerheartConditionState{}
	if row.ConditionsJson != "" {
		conditions, err = decodeProjectionConditionStates(row.ConditionsJson)
		if err != nil {
			return projectionstore.DaggerheartAdversary{}, fmt.Errorf("decode daggerheart adversary conditions: %w", err)
		}
	}
	featureStates, err := decodeAdversaryFeatureStates(row.FeatureStateJson)
	if err != nil {
		return projectionstore.DaggerheartAdversary{}, fmt.Errorf("decode daggerheart adversary feature states: %w", err)
	}
	pendingExperience, err := decodeAdversaryPendingExperience(row.PendingExperienceJson)
	if err != nil {
		return projectionstore.DaggerheartAdversary{}, fmt.Errorf("decode daggerheart adversary pending experience: %w", err)
	}

	return projectionstore.DaggerheartAdversary{
		CampaignID:        row.CampaignID,
		AdversaryID:       row.AdversaryID,
		AdversaryEntryID:  row.AdversaryEntryID,
		Name:              row.Name,
		Kind:              row.Kind,
		SessionID:         row.SessionID,
		SceneID:           row.SceneID,
		Notes:             row.Notes,
		HP:                int(row.Hp),
		HPMax:             int(row.HpMax),
		Stress:            int(row.Stress),
		StressMax:         int(row.StressMax),
		Evasion:           int(row.Evasion),
		Major:             int(row.MajorThreshold),
		Severe:            int(row.SevereThreshold),
		Armor:             int(row.Armor),
		Conditions:        conditions,
		FeatureStates:     featureStates,
		PendingExperience: pendingExperience,
		SpotlightGateID:   row.SpotlightGateID,
		SpotlightCount:    int(row.SpotlightCount),
		CreatedAt:         fromMillis(row.CreatedAt),
		UpdatedAt:         fromMillis(row.UpdatedAt),
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
			SessionID:  sessionID,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("list daggerheart adversaries: %w", err)
	}

	adversaries := make([]projectionstore.DaggerheartAdversary, 0, len(rows))
	for _, row := range rows {
		conditions := []projectionstore.DaggerheartConditionState{}
		if row.ConditionsJson != "" {
			conditions, err = decodeProjectionConditionStates(row.ConditionsJson)
			if err != nil {
				return nil, fmt.Errorf("decode daggerheart adversary conditions: %w", err)
			}
		}
		featureStates, err := decodeAdversaryFeatureStates(row.FeatureStateJson)
		if err != nil {
			return nil, fmt.Errorf("decode daggerheart adversary feature states: %w", err)
		}
		pendingExperience, err := decodeAdversaryPendingExperience(row.PendingExperienceJson)
		if err != nil {
			return nil, fmt.Errorf("decode daggerheart adversary pending experience: %w", err)
		}
		adversaries = append(adversaries, projectionstore.DaggerheartAdversary{
			CampaignID:        row.CampaignID,
			AdversaryID:       row.AdversaryID,
			AdversaryEntryID:  row.AdversaryEntryID,
			Name:              row.Name,
			Kind:              row.Kind,
			SessionID:         row.SessionID,
			SceneID:           row.SceneID,
			Notes:             row.Notes,
			HP:                int(row.Hp),
			HPMax:             int(row.HpMax),
			Stress:            int(row.Stress),
			StressMax:         int(row.StressMax),
			Evasion:           int(row.Evasion),
			Major:             int(row.MajorThreshold),
			Severe:            int(row.SevereThreshold),
			Armor:             int(row.Armor),
			Conditions:        conditions,
			FeatureStates:     featureStates,
			PendingExperience: pendingExperience,
			SpotlightGateID:   row.SpotlightGateID,
			SpotlightCount:    int(row.SpotlightCount),
			CreatedAt:         fromMillis(row.CreatedAt),
			UpdatedAt:         fromMillis(row.UpdatedAt),
		})
	}

	return adversaries, nil
}

func decodeAdversaryFeatureStates(raw string) ([]projectionstore.DaggerheartAdversaryFeatureState, error) {
	if strings.TrimSpace(raw) == "" {
		return []projectionstore.DaggerheartAdversaryFeatureState{}, nil
	}
	var featureStates []projectionstore.DaggerheartAdversaryFeatureState
	if err := json.Unmarshal([]byte(raw), &featureStates); err != nil {
		return nil, err
	}
	if featureStates == nil {
		return []projectionstore.DaggerheartAdversaryFeatureState{}, nil
	}
	return featureStates, nil
}

func decodeAdversaryPendingExperience(raw string) (*projectionstore.DaggerheartAdversaryPendingExperience, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	var pending projectionstore.DaggerheartAdversaryPendingExperience
	if err := json.Unmarshal([]byte(raw), &pending); err != nil {
		return nil, err
	}
	return &pending, nil
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

// PutDaggerheartEnvironmentEntity persists a Daggerheart environment entity projection.
func (s *Store) PutDaggerheartEnvironmentEntity(ctx context.Context, environmentEntity projectionstore.DaggerheartEnvironmentEntity) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(environmentEntity.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(environmentEntity.EnvironmentEntityID) == "" {
		return fmt.Errorf("environment entity id is required")
	}
	if strings.TrimSpace(environmentEntity.EnvironmentID) == "" {
		return fmt.Errorf("environment id is required")
	}
	if strings.TrimSpace(environmentEntity.Name) == "" {
		return fmt.Errorf("environment name is required")
	}
	if strings.TrimSpace(environmentEntity.Type) == "" {
		return fmt.Errorf("environment type is required")
	}
	if strings.TrimSpace(environmentEntity.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}

	return s.q.PutDaggerheartEnvironmentEntity(ctx, db.PutDaggerheartEnvironmentEntityParams{
		CampaignID:          environmentEntity.CampaignID,
		EnvironmentEntityID: environmentEntity.EnvironmentEntityID,
		EnvironmentID:       environmentEntity.EnvironmentID,
		Name:                environmentEntity.Name,
		Type:                environmentEntity.Type,
		Tier:                int64(environmentEntity.Tier),
		Difficulty:          int64(environmentEntity.Difficulty),
		SessionID:           environmentEntity.SessionID,
		SceneID:             environmentEntity.SceneID,
		Notes:               environmentEntity.Notes,
		CreatedAt:           toMillis(environmentEntity.CreatedAt),
		UpdatedAt:           toMillis(environmentEntity.UpdatedAt),
	})
}

// GetDaggerheartEnvironmentEntity retrieves a Daggerheart environment entity projection for a campaign.
func (s *Store) GetDaggerheartEnvironmentEntity(ctx context.Context, campaignID, environmentEntityID string) (projectionstore.DaggerheartEnvironmentEntity, error) {
	if err := ctx.Err(); err != nil {
		return projectionstore.DaggerheartEnvironmentEntity{}, err
	}
	if s == nil || s.sqlDB == nil {
		return projectionstore.DaggerheartEnvironmentEntity{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return projectionstore.DaggerheartEnvironmentEntity{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(environmentEntityID) == "" {
		return projectionstore.DaggerheartEnvironmentEntity{}, fmt.Errorf("environment entity id is required")
	}

	row, err := s.q.GetDaggerheartEnvironmentEntity(ctx, db.GetDaggerheartEnvironmentEntityParams{
		CampaignID:          campaignID,
		EnvironmentEntityID: environmentEntityID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return projectionstore.DaggerheartEnvironmentEntity{}, storage.ErrNotFound
		}
		return projectionstore.DaggerheartEnvironmentEntity{}, fmt.Errorf("get daggerheart environment entity: %w", err)
	}

	return projectionstore.DaggerheartEnvironmentEntity{
		CampaignID:          row.CampaignID,
		EnvironmentEntityID: row.EnvironmentEntityID,
		EnvironmentID:       row.EnvironmentID,
		Name:                row.Name,
		Type:                row.Type,
		Tier:                int(row.Tier),
		Difficulty:          int(row.Difficulty),
		SessionID:           row.SessionID,
		SceneID:             row.SceneID,
		Notes:               row.Notes,
		CreatedAt:           fromMillis(row.CreatedAt),
		UpdatedAt:           fromMillis(row.UpdatedAt),
	}, nil
}

// ListDaggerheartEnvironmentEntities retrieves environment entity projections for a campaign session.
func (s *Store) ListDaggerheartEnvironmentEntities(ctx context.Context, campaignID, sessionID, sceneID string) ([]projectionstore.DaggerheartEnvironmentEntity, error) {
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

	var (
		rows []db.DaggerheartEnvironmentEntity
		err  error
	)
	if strings.TrimSpace(sceneID) == "" {
		rows, err = s.q.ListDaggerheartEnvironmentEntitiesBySession(ctx, db.ListDaggerheartEnvironmentEntitiesBySessionParams{
			CampaignID: campaignID,
			SessionID:  sessionID,
		})
	} else {
		rows, err = s.q.ListDaggerheartEnvironmentEntitiesByScene(ctx, db.ListDaggerheartEnvironmentEntitiesBySceneParams{
			CampaignID: campaignID,
			SessionID:  sessionID,
			SceneID:    sceneID,
		})
	}
	if err != nil {
		return nil, fmt.Errorf("list daggerheart environment entities: %w", err)
	}

	environmentEntities := make([]projectionstore.DaggerheartEnvironmentEntity, 0, len(rows))
	for _, row := range rows {
		environmentEntities = append(environmentEntities, projectionstore.DaggerheartEnvironmentEntity{
			CampaignID:          row.CampaignID,
			EnvironmentEntityID: row.EnvironmentEntityID,
			EnvironmentID:       row.EnvironmentID,
			Name:                row.Name,
			Type:                row.Type,
			Tier:                int(row.Tier),
			Difficulty:          int(row.Difficulty),
			SessionID:           row.SessionID,
			SceneID:             row.SceneID,
			Notes:               row.Notes,
			CreatedAt:           fromMillis(row.CreatedAt),
			UpdatedAt:           fromMillis(row.UpdatedAt),
		})
	}
	return environmentEntities, nil
}

// DeleteDaggerheartEnvironmentEntity removes an environment entity projection for a campaign.
func (s *Store) DeleteDaggerheartEnvironmentEntity(ctx context.Context, campaignID, environmentEntityID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(environmentEntityID) == "" {
		return fmt.Errorf("environment entity id is required")
	}

	return s.q.DeleteDaggerheartEnvironmentEntity(ctx, db.DeleteDaggerheartEnvironmentEntityParams{
		CampaignID:          campaignID,
		EnvironmentEntityID: environmentEntityID,
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
