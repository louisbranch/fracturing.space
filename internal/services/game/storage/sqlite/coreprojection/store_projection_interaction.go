package coreprojection

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// PutSessionInteraction persists session-level interaction state for the game surface.
func (s *Store) PutSessionInteraction(ctx context.Context, interaction storage.SessionInteraction) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(interaction.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(interaction.SessionID) == "" {
		return fmt.Errorf("session id is required")
	}

	oocPostsJSON, err := json.Marshal(interaction.OOCPosts)
	if err != nil {
		return fmt.Errorf("encode session ooc posts: %w", err)
	}
	characterControllersJSON, err := json.Marshal(interaction.CharacterControllers)
	if err != nil {
		return fmt.Errorf("encode session character controllers: %w", err)
	}
	readyJSON, err := json.Marshal(interaction.ReadyToResumeParticipantIDs)
	if err != nil {
		return fmt.Errorf("encode ready to resume: %w", err)
	}

	_, err = s.projectionQueryable().ExecContext(ctx,
		`INSERT INTO session_interactions (
			campaign_id, session_id, active_scene_id, gm_authority_participant_id,
			character_controllers_json,
			ooc_opened, ooc_requested_by_participant_id, ooc_reason,
			ooc_interrupted_scene_id, ooc_interrupted_phase_id, ooc_interrupted_phase_status,
			ooc_resolution_pending, ooc_posts_json, ready_to_resume_json,
			ai_turn_status, ai_turn_token, ai_turn_owner_participant_id,
			ai_turn_source_event_type, ai_turn_source_scene_id, ai_turn_source_phase_id, ai_turn_last_error,
			updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (campaign_id, session_id) DO UPDATE SET
			active_scene_id = excluded.active_scene_id,
			gm_authority_participant_id = excluded.gm_authority_participant_id,
			character_controllers_json = excluded.character_controllers_json,
			ooc_opened = excluded.ooc_opened,
			ooc_requested_by_participant_id = excluded.ooc_requested_by_participant_id,
			ooc_reason = excluded.ooc_reason,
			ooc_interrupted_scene_id = excluded.ooc_interrupted_scene_id,
			ooc_interrupted_phase_id = excluded.ooc_interrupted_phase_id,
			ooc_interrupted_phase_status = excluded.ooc_interrupted_phase_status,
			ooc_resolution_pending = excluded.ooc_resolution_pending,
			ooc_posts_json = excluded.ooc_posts_json,
			ready_to_resume_json = excluded.ready_to_resume_json,
			ai_turn_status = excluded.ai_turn_status,
			ai_turn_token = excluded.ai_turn_token,
			ai_turn_owner_participant_id = excluded.ai_turn_owner_participant_id,
			ai_turn_source_event_type = excluded.ai_turn_source_event_type,
			ai_turn_source_scene_id = excluded.ai_turn_source_scene_id,
			ai_turn_source_phase_id = excluded.ai_turn_source_phase_id,
			ai_turn_last_error = excluded.ai_turn_last_error,
			updated_at = excluded.updated_at`,
		interaction.CampaignID,
		interaction.SessionID,
		interaction.ActiveSceneID,
		interaction.GMAuthorityParticipantID,
		characterControllersJSON,
		boolToInt(interaction.OOCPaused),
		interaction.OOCRequestedByParticipantID,
		interaction.OOCReason,
		interaction.OOCInterruptedSceneID,
		interaction.OOCInterruptedPhaseID,
		interaction.OOCInterruptedPhaseStatus,
		boolToInt(interaction.OOCResolutionPending),
		oocPostsJSON,
		readyJSON,
		string(interaction.AITurn.Status),
		interaction.AITurn.TurnToken,
		interaction.AITurn.OwnerParticipantID,
		interaction.AITurn.SourceEventType,
		interaction.AITurn.SourceSceneID,
		interaction.AITurn.SourcePhaseID,
		interaction.AITurn.LastError,
		sqliteutil.ToMillis(interaction.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("put session interaction: %w", err)
	}
	return nil
}

// GetSessionInteraction retrieves session-level interaction state for one session.
func (s *Store) GetSessionInteraction(ctx context.Context, campaignID, sessionID string) (storage.SessionInteraction, error) {
	if err := ctx.Err(); err != nil {
		return storage.SessionInteraction{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SessionInteraction{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SessionInteraction{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return storage.SessionInteraction{}, fmt.Errorf("session id is required")
	}

	var (
		activeSceneID               string
		gmAuthorityParticipantID    string
		characterControllersJSON    []byte
		oocPaused                   int64
		oocRequestedByParticipantID string
		oocReason                   string
		oocInterruptedSceneID       string
		oocInterruptedPhaseID       string
		oocInterruptedPhaseStatus   string
		oocResolutionPending        int64
		oocPostsJSON                []byte
		readyJSON                   []byte
		aiTurnStatus                string
		aiTurnToken                 string
		aiTurnOwnerParticipantID    string
		aiTurnSourceEventType       string
		aiTurnSourceSceneID         string
		aiTurnSourcePhaseID         string
		aiTurnLastError             string
		updatedAt                   int64
	)
	err := s.projectionQueryable().QueryRowContext(ctx,
		`SELECT active_scene_id, gm_authority_participant_id, character_controllers_json, ooc_opened,
		        ooc_requested_by_participant_id, ooc_reason, ooc_interrupted_scene_id,
		        ooc_interrupted_phase_id, ooc_interrupted_phase_status, ooc_resolution_pending,
		        ooc_posts_json, ready_to_resume_json,
		        ai_turn_status, ai_turn_token, ai_turn_owner_participant_id, ai_turn_source_event_type,
		        ai_turn_source_scene_id, ai_turn_source_phase_id, ai_turn_last_error, updated_at
		 FROM session_interactions
		 WHERE campaign_id = ? AND session_id = ?`,
		campaignID,
		sessionID,
	).Scan(
		&activeSceneID,
		&gmAuthorityParticipantID,
		&characterControllersJSON,
		&oocPaused,
		&oocRequestedByParticipantID,
		&oocReason,
		&oocInterruptedSceneID,
		&oocInterruptedPhaseID,
		&oocInterruptedPhaseStatus,
		&oocResolutionPending,
		&oocPostsJSON,
		&readyJSON,
		&aiTurnStatus,
		&aiTurnToken,
		&aiTurnOwnerParticipantID,
		&aiTurnSourceEventType,
		&aiTurnSourceSceneID,
		&aiTurnSourcePhaseID,
		&aiTurnLastError,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SessionInteraction{}, storage.ErrNotFound
		}
		return storage.SessionInteraction{}, fmt.Errorf("get session interaction: %w", err)
	}

	var posts []storage.SessionOOCPost
	if len(oocPostsJSON) != 0 {
		if err := json.Unmarshal(oocPostsJSON, &posts); err != nil {
			return storage.SessionInteraction{}, fmt.Errorf("decode session ooc posts: %w", err)
		}
	}
	var characterControllers []storage.SessionCharacterController
	if len(characterControllersJSON) != 0 {
		if err := json.Unmarshal(characterControllersJSON, &characterControllers); err != nil {
			return storage.SessionInteraction{}, fmt.Errorf("decode session character controllers: %w", err)
		}
	}
	var ready []string
	if len(readyJSON) != 0 {
		if err := json.Unmarshal(readyJSON, &ready); err != nil {
			return storage.SessionInteraction{}, fmt.Errorf("decode session ready list: %w", err)
		}
	}
	return storage.SessionInteraction{
		CampaignID:                  campaignID,
		SessionID:                   sessionID,
		CharacterControllers:        characterControllers,
		ActiveSceneID:               activeSceneID,
		GMAuthorityParticipantID:    gmAuthorityParticipantID,
		OOCPaused:                   intToBool(oocPaused),
		OOCRequestedByParticipantID: oocRequestedByParticipantID,
		OOCReason:                   oocReason,
		OOCInterruptedSceneID:       oocInterruptedSceneID,
		OOCInterruptedPhaseID:       oocInterruptedPhaseID,
		OOCInterruptedPhaseStatus:   oocInterruptedPhaseStatus,
		OOCResolutionPending:        intToBool(oocResolutionPending),
		OOCPosts:                    posts,
		ReadyToResumeParticipantIDs: ready,
		AITurn: storage.SessionAITurn{
			Status:             normalizeAITurnStatus(aiTurnStatus),
			TurnToken:          aiTurnToken,
			OwnerParticipantID: aiTurnOwnerParticipantID,
			SourceEventType:    aiTurnSourceEventType,
			SourceSceneID:      aiTurnSourceSceneID,
			SourcePhaseID:      aiTurnSourcePhaseID,
			LastError:          aiTurnLastError,
		},
		UpdatedAt: sqliteutil.FromMillis(updatedAt),
	}, nil
}

// PutSceneInteraction persists scene-level player-phase state for the game surface.
func (s *Store) PutSceneInteraction(ctx context.Context, interaction storage.SceneInteraction) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(interaction.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(interaction.SceneID) == "" {
		return fmt.Errorf("scene id is required")
	}

	actingCharacterIDsJSON, err := json.Marshal(interaction.ActingCharacterIDs)
	if err != nil {
		return fmt.Errorf("encode acting character ids: %w", err)
	}
	actingParticipantIDsJSON, err := json.Marshal(interaction.ActingParticipantIDs)
	if err != nil {
		return fmt.Errorf("encode acting participant ids: %w", err)
	}
	slotsJSON, err := json.Marshal(interaction.Slots)
	if err != nil {
		return fmt.Errorf("encode scene slots: %w", err)
	}

	_, err = s.projectionQueryable().ExecContext(ctx,
		`INSERT INTO scene_interactions (
			campaign_id, scene_id, session_id, phase_open, phase_id,
			phase_status, acting_character_ids_json, acting_participant_ids_json, slots_json, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (campaign_id, scene_id) DO UPDATE SET
			session_id = excluded.session_id,
			phase_open = excluded.phase_open,
			phase_id = excluded.phase_id,
			phase_status = excluded.phase_status,
			acting_character_ids_json = excluded.acting_character_ids_json,
			acting_participant_ids_json = excluded.acting_participant_ids_json,
			slots_json = excluded.slots_json,
			updated_at = excluded.updated_at`,
		interaction.CampaignID,
		interaction.SceneID,
		interaction.SessionID,
		boolToInt(interaction.PhaseOpen),
		interaction.PhaseID,
		string(interaction.PhaseStatus),
		actingCharacterIDsJSON,
		actingParticipantIDsJSON,
		slotsJSON,
		sqliteutil.ToMillis(interaction.UpdatedAt),
	)
	if err != nil {
		return fmt.Errorf("put scene interaction: %w", err)
	}
	return nil
}

// GetSceneInteraction retrieves scene-level player-phase state for one scene.
func (s *Store) GetSceneInteraction(ctx context.Context, campaignID, sceneID string) (storage.SceneInteraction, error) {
	if err := ctx.Err(); err != nil {
		return storage.SceneInteraction{}, err
	}
	if s == nil || s.sqlDB == nil {
		return storage.SceneInteraction{}, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return storage.SceneInteraction{}, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sceneID) == "" {
		return storage.SceneInteraction{}, fmt.Errorf("scene id is required")
	}

	var (
		sessionID                string
		phaseOpen                int64
		phaseID                  string
		phaseStatus              string
		actingCharacterIDsJSON   []byte
		actingParticipantIDsJSON []byte
		slotsJSON                []byte
		updatedAt                int64
	)
	err := s.projectionQueryable().QueryRowContext(ctx,
		`SELECT session_id, phase_open, phase_id, phase_status, acting_character_ids_json,
		        acting_participant_ids_json, slots_json, updated_at
		 FROM scene_interactions
		 WHERE campaign_id = ? AND scene_id = ?`,
		campaignID,
		sceneID,
	).Scan(
		&sessionID,
		&phaseOpen,
		&phaseID,
		&phaseStatus,
		&actingCharacterIDsJSON,
		&actingParticipantIDsJSON,
		&slotsJSON,
		&updatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return storage.SceneInteraction{}, storage.ErrNotFound
		}
		return storage.SceneInteraction{}, fmt.Errorf("get scene interaction: %w", err)
	}

	var actingCharacterIDs []string
	if len(actingCharacterIDsJSON) != 0 {
		if err := json.Unmarshal(actingCharacterIDsJSON, &actingCharacterIDs); err != nil {
			return storage.SceneInteraction{}, fmt.Errorf("decode acting character ids: %w", err)
		}
	}
	var actingParticipantIDs []string
	if len(actingParticipantIDsJSON) != 0 {
		if err := json.Unmarshal(actingParticipantIDsJSON, &actingParticipantIDs); err != nil {
			return storage.SceneInteraction{}, fmt.Errorf("decode acting participant ids: %w", err)
		}
	}
	var slots []storage.ScenePlayerSlot
	if len(slotsJSON) != 0 {
		if err := json.Unmarshal(slotsJSON, &slots); err != nil {
			return storage.SceneInteraction{}, fmt.Errorf("decode scene slots: %w", err)
		}
	}
	return storage.SceneInteraction{
		CampaignID:           campaignID,
		SceneID:              sceneID,
		SessionID:            sessionID,
		PhaseOpen:            intToBool(phaseOpen),
		PhaseID:              phaseID,
		PhaseStatus:          normalizeScenePhaseStatus(phaseStatus),
		ActingCharacterIDs:   actingCharacterIDs,
		ActingParticipantIDs: actingParticipantIDs,
		Slots:                slots,
		UpdatedAt:            sqliteutil.FromMillis(updatedAt),
	}, nil
}

// PutSceneGMInteraction persists one immutable GM-authored interaction record for a scene.
func (s *Store) PutSceneGMInteraction(ctx context.Context, interaction storage.SceneGMInteraction) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if s == nil || s.sqlDB == nil {
		return fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(interaction.CampaignID) == "" {
		return fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(interaction.SceneID) == "" {
		return fmt.Errorf("scene id is required")
	}
	if strings.TrimSpace(interaction.InteractionID) == "" {
		return fmt.Errorf("interaction id is required")
	}

	characterIDsJSON, err := json.Marshal(interaction.CharacterIDs)
	if err != nil {
		return fmt.Errorf("encode interaction character ids: %w", err)
	}
	beatsJSON, err := json.Marshal(interaction.Beats)
	if err != nil {
		return fmt.Errorf("encode interaction beats: %w", err)
	}
	illustrationJSON, err := json.Marshal(interaction.Illustration)
	if err != nil {
		return fmt.Errorf("encode interaction illustration: %w", err)
	}

	_, err = s.projectionQueryable().ExecContext(ctx,
		`INSERT INTO scene_gm_interactions (
			campaign_id, scene_id, session_id, interaction_id, phase_id, participant_id,
			title, character_ids_json, illustration_json, beats_json, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (campaign_id, interaction_id) DO UPDATE SET
			scene_id = excluded.scene_id,
			session_id = excluded.session_id,
			phase_id = excluded.phase_id,
			participant_id = excluded.participant_id,
			title = excluded.title,
			character_ids_json = excluded.character_ids_json,
			illustration_json = excluded.illustration_json,
			beats_json = excluded.beats_json,
			created_at = excluded.created_at`,
		interaction.CampaignID,
		interaction.SceneID,
		interaction.SessionID,
		interaction.InteractionID,
		interaction.PhaseID,
		interaction.ParticipantID,
		interaction.Title,
		characterIDsJSON,
		illustrationJSON,
		beatsJSON,
		sqliteutil.ToMillis(interaction.CreatedAt),
	)
	if err != nil {
		return fmt.Errorf("put scene gm interaction: %w", err)
	}
	return nil
}

// ListSceneGMInteractions retrieves immutable GM interactions for one scene ordered newest first.
func (s *Store) ListSceneGMInteractions(ctx context.Context, campaignID, sceneID string) ([]storage.SceneGMInteraction, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s == nil || s.sqlDB == nil {
		return nil, fmt.Errorf("storage is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if strings.TrimSpace(sceneID) == "" {
		return nil, fmt.Errorf("scene id is required")
	}

	rows, err := s.projectionQueryable().QueryContext(ctx,
		`SELECT session_id, interaction_id, phase_id, participant_id, title,
		        character_ids_json, illustration_json, beats_json, created_at
		 FROM scene_gm_interactions
		 WHERE campaign_id = ? AND scene_id = ?
		 ORDER BY created_at DESC, interaction_id DESC`,
		campaignID,
		sceneID,
	)
	if err != nil {
		return nil, fmt.Errorf("list scene gm interactions: %w", err)
	}
	defer rows.Close()

	interactions := []storage.SceneGMInteraction{}
	for rows.Next() {
		var (
			sessionID        string
			interactionID    string
			phaseID          string
			participantID    string
			title            string
			characterIDsJSON []byte
			illustrationJSON []byte
			beatsJSON        []byte
			createdAt        int64
		)
		if err := rows.Scan(
			&sessionID,
			&interactionID,
			&phaseID,
			&participantID,
			&title,
			&characterIDsJSON,
			&illustrationJSON,
			&beatsJSON,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan scene gm interaction: %w", err)
		}

		var characterIDs []string
		if len(characterIDsJSON) != 0 {
			if err := json.Unmarshal(characterIDsJSON, &characterIDs); err != nil {
				return nil, fmt.Errorf("decode interaction character ids: %w", err)
			}
		}
		var beats []storage.SceneGMInteractionBeat
		if len(beatsJSON) != 0 {
			if err := json.Unmarshal(beatsJSON, &beats); err != nil {
				return nil, fmt.Errorf("decode interaction beats: %w", err)
			}
		}
		var illustration *storage.SceneGMInteractionIllustration
		if len(illustrationJSON) != 0 && string(illustrationJSON) != "null" {
			var value storage.SceneGMInteractionIllustration
			if err := json.Unmarshal(illustrationJSON, &value); err != nil {
				return nil, fmt.Errorf("decode interaction illustration: %w", err)
			}
			illustration = &value
		}
		interactions = append(interactions, storage.SceneGMInteraction{
			CampaignID:    campaignID,
			SceneID:       sceneID,
			SessionID:     sessionID,
			InteractionID: interactionID,
			PhaseID:       phaseID,
			ParticipantID: participantID,
			Title:         title,
			CharacterIDs:  characterIDs,
			Illustration:  illustration,
			Beats:         beats,
			CreatedAt:     sqliteutil.FromMillis(createdAt),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate scene gm interactions: %w", err)
	}
	return interactions, nil
}

func normalizeAITurnStatus(value string) session.AITurnStatus {
	status, err := session.NormalizeAITurnStatus(value)
	if err == nil {
		return status
	}
	return session.AITurnStatusIdle
}

func normalizeScenePhaseStatus(value string) scene.PlayerPhaseStatus {
	switch scene.PlayerPhaseStatus(strings.TrimSpace(value)) {
	case scene.PlayerPhaseStatusPlayers:
		return scene.PlayerPhaseStatusPlayers
	case scene.PlayerPhaseStatusGMReview:
		return scene.PlayerPhaseStatusGMReview
	default:
		return ""
	}
}
