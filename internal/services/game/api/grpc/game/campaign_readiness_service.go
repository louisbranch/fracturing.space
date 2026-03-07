package game

import (
	"context"
	"errors"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	platformi18n "github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/readiness"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"golang.org/x/text/message"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const readinessCharacterPageSize = 200

// GetCampaignSessionReadiness returns deterministic readiness blockers for session start.
func (s *CampaignService) GetCampaignSessionReadiness(ctx context.Context, in *campaignv1.GetCampaignSessionReadinessRequest) (*campaignv1.GetCampaignSessionReadinessResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get campaign session readiness request is required")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	record, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		return nil, handleDomainError(err)
	}
	if err := campaign.ValidateCampaignOperation(record.Status, campaign.CampaignOpRead); err != nil {
		return nil, handleDomainError(err)
	}
	if err := requireReadPolicy(ctx, s.stores, record); err != nil {
		return nil, err
	}

	participantsByCampaign, err := s.stores.Participant.ListParticipantsByCampaign(ctx, campaignID)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list participants by campaign: %v", err)
	}

	charactersByCampaign, err := listAllCharactersByCampaign(ctx, s.stores.Character, campaignID)
	if err != nil {
		return nil, err
	}

	hasActiveSession, err := campaignHasActiveSession(ctx, s.stores.Session, campaignID)
	if err != nil {
		return nil, err
	}

	state, err := campaignReadinessAggregateState(ctx, s.stores, record, participantsByCampaign, charactersByCampaign)
	if err != nil {
		return nil, err
	}

	report := readiness.EvaluateSessionStartReport(state, readiness.ReportOptions{
		SystemReadiness:        systemReadinessChecker(record.System),
		IncludeSessionBoundary: true,
		HasActiveSession:       hasActiveSession,
	})
	locale := resolveReadinessLocale(in.GetLocale(), record.Locale)
	readinessProto := &campaignv1.CampaignSessionReadiness{
		Ready: report.Ready(),
	}
	if len(report.Blockers) > 0 {
		readinessProto.Blockers = make([]*campaignv1.CampaignSessionReadinessBlocker, 0, len(report.Blockers))
		for _, blocker := range report.Blockers {
			readinessProto.Blockers = append(readinessProto.Blockers, readinessBlockerToProto(locale, blocker))
		}
	}

	return &campaignv1.GetCampaignSessionReadinessResponse{
		Readiness: readinessProto,
	}, nil
}

func listAllCharactersByCampaign(ctx context.Context, store storage.CharacterStore, campaignID string) ([]storage.CharacterRecord, error) {
	if store == nil {
		return nil, status.Error(codes.Internal, "character store is not configured")
	}

	characters := make([]storage.CharacterRecord, 0, readinessCharacterPageSize)
	pageToken := ""
	seenTokens := map[string]struct{}{"": {}}
	for {
		page, err := store.ListCharacters(ctx, campaignID, readinessCharacterPageSize, pageToken)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "list characters: %v", err)
		}
		characters = append(characters, page.Characters...)

		nextPageToken := strings.TrimSpace(page.NextPageToken)
		if nextPageToken == "" {
			return characters, nil
		}
		if _, exists := seenTokens[nextPageToken]; exists {
			return nil, status.Error(codes.Internal, "list characters returned a repeated page token")
		}
		seenTokens[nextPageToken] = struct{}{}
		pageToken = nextPageToken
	}
}

func campaignHasActiveSession(ctx context.Context, store storage.SessionStore, campaignID string) (bool, error) {
	if store == nil {
		return false, status.Error(codes.Internal, "session store is not configured")
	}
	_, err := store.GetActiveSession(ctx, campaignID)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, storage.ErrNotFound) {
		return false, nil
	}
	return false, status.Errorf(codes.Internal, "check active session: %v", err)
}

func campaignReadinessAggregateState(
	ctx context.Context,
	stores Stores,
	campaignRecord storage.CampaignRecord,
	participantRecords []storage.ParticipantRecord,
	characterRecords []storage.CharacterRecord,
) (aggregate.State, error) {
	state := aggregate.State{
		Campaign: campaign.State{
			Status:    campaignRecord.Status,
			GmMode:    string(campaignRecord.GmMode),
			AIAgentID: strings.TrimSpace(campaignRecord.AIAgentID),
		},
		Participants: make(map[string]participant.State, len(participantRecords)),
		Characters:   make(map[string]character.State, len(characterRecords)),
	}

	for _, participantRecord := range participantRecords {
		participantID := strings.TrimSpace(participantRecord.ID)
		if participantID == "" {
			continue
		}
		state.Participants[participantID] = participant.State{
			Joined:         true,
			ParticipantID:  participantID,
			UserID:         strings.TrimSpace(participantRecord.UserID),
			Name:           strings.TrimSpace(participantRecord.Name),
			Role:           string(participantRecord.Role),
			Controller:     string(participantRecord.Controller),
			CampaignAccess: string(participantRecord.CampaignAccess),
		}
	}

	for _, characterRecord := range characterRecords {
		characterID := strings.TrimSpace(characterRecord.ID)
		if characterID == "" {
			continue
		}
		state.Characters[characterID] = character.State{
			Created:       true,
			CharacterID:   characterID,
			ParticipantID: strings.TrimSpace(characterRecord.ParticipantID),
			SystemProfile: map[string]any{},
		}
	}

	if campaignRecord.System == commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		if stores.SystemStores.Daggerheart == nil {
			return aggregate.State{}, status.Error(codes.Internal, "daggerheart projection store is not configured")
		}
		for characterID, characterState := range state.Characters {
			profile, err := stores.SystemStores.Daggerheart.GetDaggerheartCharacterProfile(ctx, campaignRecord.ID, characterID)
			if err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					continue
				}
				return aggregate.State{}, status.Errorf(codes.Internal, "get daggerheart character profile %s: %v", characterID, err)
			}
			characterState.SystemProfile = daggerheartSystemProfileMap(profile)
			state.Characters[characterID] = characterState
		}
	}

	return state, nil
}

func systemReadinessChecker(system commonv1.GameSystem) readiness.CharacterSystemReadiness {
	switch system {
	case commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART:
		return daggerheartdomain.EvaluateCreationReadinessFromSystemProfile
	default:
		return nil
	}
}

func resolveReadinessLocale(requested commonv1.Locale, campaignLocale commonv1.Locale) commonv1.Locale {
	if requested != commonv1.Locale_LOCALE_UNSPECIFIED {
		return platformi18n.NormalizeLocale(requested)
	}
	if campaignLocale != commonv1.Locale_LOCALE_UNSPECIFIED {
		return platformi18n.NormalizeLocale(campaignLocale)
	}
	return commonv1.Locale_LOCALE_EN_US
}

func readinessBlockerToProto(locale commonv1.Locale, blocker readiness.Blocker) *campaignv1.CampaignSessionReadinessBlocker {
	metadata := make(map[string]string, len(blocker.Metadata))
	for key, value := range blocker.Metadata {
		metadata[key] = value
	}
	return &campaignv1.CampaignSessionReadinessBlocker{
		Code:     strings.TrimSpace(blocker.Code),
		Message:  localizeReadinessBlockerMessage(locale, blocker),
		Metadata: metadata,
	}
}

func localizeReadinessBlockerMessage(locale commonv1.Locale, blocker readiness.Blocker) string {
	printer := message.NewPrinter(platformi18n.TagForLocale(locale))
	switch blocker.Code {
	case readiness.RejectionCodeSessionReadinessCampaignStatusDisallowsStart:
		return printer.Sprintf("game.session_readiness.campaign_status_disallows_start", readinessBlockerMetadataValue(blocker.Metadata, "status"))
	case readiness.RejectionCodeSessionReadinessActiveSessionExists:
		return printer.Sprintf("game.session_readiness.active_session_exists")
	case readiness.RejectionCodeSessionReadinessAIAgentRequired:
		return printer.Sprintf("game.session_readiness.ai_agent_required")
	case readiness.RejectionCodeSessionReadinessAIGMParticipantRequired:
		return printer.Sprintf("game.session_readiness.ai_gm_participant_required")
	case readiness.RejectionCodeSessionReadinessGMRequired:
		return printer.Sprintf("game.session_readiness.gm_required")
	case readiness.RejectionCodeSessionReadinessPlayerRequired:
		return printer.Sprintf("game.session_readiness.player_required")
	case readiness.RejectionCodeSessionReadinessCharacterControllerRequired:
		return printer.Sprintf("game.session_readiness.character_controller_required", readinessBlockerMetadataValue(blocker.Metadata, "character_id"))
	case readiness.RejectionCodeSessionReadinessPlayerCharacterRequired:
		return printer.Sprintf("game.session_readiness.player_character_required", readinessBlockerMetadataValue(blocker.Metadata, "participant_id"))
	case readiness.RejectionCodeSessionReadinessCharacterSystemRequired:
		reason := readinessBlockerOptionalMetadataValue(blocker.Metadata, "reason")
		if reason == "" {
			return printer.Sprintf("game.session_readiness.character_system_required", readinessBlockerMetadataValue(blocker.Metadata, "character_id"))
		}
		return printer.Sprintf("game.session_readiness.character_system_required_with_reason", readinessBlockerMetadataValue(blocker.Metadata, "character_id"), reason)
	default:
		return strings.TrimSpace(blocker.Message)
	}
}

func readinessBlockerOptionalMetadataValue(metadata map[string]string, key string) string {
	return readinessBlockerMetadataValueOrDefault(metadata, key, "")
}

func readinessBlockerMetadataValue(metadata map[string]string, key string) string {
	return readinessBlockerMetadataValueOrDefault(metadata, key, "unspecified")
}

func readinessBlockerMetadataValueOrDefault(metadata map[string]string, key, fallback string) string {
	value := strings.TrimSpace(metadata[key])
	if value != "" {
		return value
	}
	return fallback
}
