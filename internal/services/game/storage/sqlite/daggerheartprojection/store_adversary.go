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
