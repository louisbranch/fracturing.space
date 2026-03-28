package daggerheartprojection

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/db"
)

// PutDaggerheartCharacterState persists a Daggerheart character state extension.
func (s *Store) PutDaggerheartCharacterState(ctx context.Context, state projectionstore.DaggerheartCharacterState) error {
	if err := s.validateProjectionStore(ctx); err != nil {
		return err
	}
	if err := requireProjectionField(state.CampaignID, "campaign id"); err != nil {
		return err
	}
	if err := requireProjectionField(state.CharacterID, "character id"); err != nil {
		return err
	}
	if state.HopeMax <= 0 {
		return fmt.Errorf("hope max must be greater than zero")
	}
	if err := requireProjectionField(state.LifeState, "life state"); err != nil {
		return err
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

	return s.q.PutDaggerheartCharacterState(ctx, db.PutDaggerheartCharacterStateParams{
		CampaignID:                    state.CampaignID,
		CharacterID:                   state.CharacterID,
		Hp:                            int64(state.Hp),
		Hope:                          int64(state.Hope),
		HopeMax:                       int64(state.HopeMax),
		Stress:                        int64(state.Stress),
		Armor:                         int64(state.Armor),
		ConditionsJson:                string(conditionsJSON),
		TemporaryArmorJson:            string(temporaryArmorJSON),
		LifeState:                     state.LifeState,
		ClassStateJson:                string(classStateJSON),
		SubclassStateJson:             string(subclassStateJSON),
		CompanionStateJson:            string(companionStateJSON),
		ImpenetrableUsedThisShortRest: boolToInt64(state.ImpenetrableUsedThisShortRest),
		StatModifiersJson:             string(statModifiersJSON),
	})
}

// GetDaggerheartCharacterState retrieves a Daggerheart character state extension.
func (s *Store) GetDaggerheartCharacterState(ctx context.Context, campaignID, characterID string) (projectionstore.DaggerheartCharacterState, error) {
	if err := s.validateProjectionStore(ctx); err != nil {
		return projectionstore.DaggerheartCharacterState{}, err
	}
	if err := requireProjectionField(campaignID, "campaign id"); err != nil {
		return projectionstore.DaggerheartCharacterState{}, err
	}
	if err := requireProjectionField(characterID, "character id"); err != nil {
		return projectionstore.DaggerheartCharacterState{}, err
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

	return dbDaggerheartCharacterStateToDomain(row)
}

func dbDaggerheartCharacterStateToDomain(row db.DaggerheartCharacterState) (projectionstore.DaggerheartCharacterState, error) {
	var conditions []projectionstore.DaggerheartConditionState
	if row.ConditionsJson != "" {
		decodedConditions, err := decodeProjectionConditionStates(row.ConditionsJson)
		if err != nil {
			return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("decode conditions: %w", err)
		}
		conditions = decodedConditions
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
	subclassState, err := decodeProjectionOptionalState[projectionstore.DaggerheartSubclassState](row.SubclassStateJson)
	if err != nil {
		return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("decode subclass state: %w", err)
	}
	companionState, err := decodeProjectionOptionalState[projectionstore.DaggerheartCompanionState](row.CompanionStateJson)
	if err != nil {
		return projectionstore.DaggerheartCharacterState{}, fmt.Errorf("decode companion state: %w", err)
	}

	lifeState := row.LifeState
	if requireProjectionField(lifeState, "life state") != nil {
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

func decodeProjectionOptionalState[T any](raw string) (*T, error) {
	if raw == "" || raw == "null" || raw == "{}" {
		return nil, nil
	}
	var decoded T
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		return nil, err
	}
	return &decoded, nil
}
