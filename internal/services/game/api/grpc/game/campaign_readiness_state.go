package game

import (
	"context"
	"errors"
	"strings"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	daggerheartgrpc "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/readiness"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const readinessCharacterPageSize = 200

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
			characterState.SystemProfile = daggerheartgrpc.SystemProfileMap(profile)
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
