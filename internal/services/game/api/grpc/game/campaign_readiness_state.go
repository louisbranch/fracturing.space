package game

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/game/handler"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	daggerheartdomain "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/readiness"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const readinessCharacterPageSize = handler.PageLarge

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
			return nil, grpcerror.Internal("list characters", err)
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
	return false, grpcerror.Internal("check active session", err)
}

func campaignReadinessAggregateState(
	ctx context.Context,
	daggerheartStore projectionstore.Store,
	campaignRecord storage.CampaignRecord,
	participantRecords []storage.ParticipantRecord,
	characterRecords []storage.CharacterRecord,
) (aggregate.State, error) {
	state := aggregate.State{
		Campaign: campaign.State{
			Status:    campaignRecord.Status,
			GmMode:    campaignRecord.GmMode,
			AIAgentID: strings.TrimSpace(campaignRecord.AIAgentID),
		},
		Participants: make(map[ids.ParticipantID]participant.State, len(participantRecords)),
		Characters:   make(map[ids.CharacterID]character.State, len(characterRecords)),
		Systems:      make(map[module.Key]any),
	}

	for _, participantRecord := range participantRecords {
		participantID := strings.TrimSpace(participantRecord.ID)
		if participantID == "" {
			continue
		}
		state.Participants[ids.ParticipantID(participantID)] = participant.State{
			Joined:         true,
			ParticipantID:  ids.ParticipantID(participantID),
			UserID:         ids.UserID(strings.TrimSpace(participantRecord.UserID)),
			Name:           strings.TrimSpace(participantRecord.Name),
			Role:           participantRecord.Role,
			Controller:     participantRecord.Controller,
			CampaignAccess: participantRecord.CampaignAccess,
		}
	}

	for _, characterRecord := range characterRecords {
		characterID := strings.TrimSpace(characterRecord.ID)
		if characterID == "" {
			continue
		}
		state.Characters[ids.CharacterID(characterID)] = character.State{
			Created:       true,
			CharacterID:   ids.CharacterID(characterID),
			Name:          strings.TrimSpace(characterRecord.Name),
			ParticipantID: ids.ParticipantID(strings.TrimSpace(characterRecord.ParticipantID)),
		}
	}

	if handler.SystemIDFromCampaignRecord(campaignRecord) == bridge.SystemIDDaggerheart {
		if daggerheartStore == nil {
			return aggregate.State{}, status.Error(codes.Internal, "daggerheart projection store is not configured")
		}
		snapshot := daggerheartdomain.SnapshotState{
			CampaignID:        ids.CampaignID(campaignRecord.ID),
			GMFear:            daggerheartdomain.GMFearDefault,
			CharacterProfiles: make(map[ids.CharacterID]daggerheartdomain.CharacterProfile),
			CharacterStates:   make(map[ids.CharacterID]daggerheartdomain.CharacterState),
			AdversaryStates:   make(map[ids.AdversaryID]daggerheartdomain.AdversaryState),
			CountdownStates:   make(map[ids.CountdownID]daggerheartdomain.CountdownState),
		}
		for characterID, characterState := range state.Characters {
			profile, err := daggerheartStore.GetDaggerheartCharacterProfile(ctx, campaignRecord.ID, string(characterID))
			if err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					continue
				}
				return aggregate.State{}, grpcerror.Internal(fmt.Sprintf("get daggerheart character profile %s", characterID), err)
			}
			snapshot.CharacterProfiles[characterID] = daggerheartdomain.CharacterProfileFromStorage(profile)
			state.Characters[characterID] = characterState
		}
		state.Systems[module.Key{ID: daggerheartdomain.SystemID, Version: daggerheartdomain.SystemVersion}] = snapshot
	}

	return state, nil
}

func systemReadinessChecker(system bridge.SystemID, state aggregate.State) readiness.CharacterSystemReadiness {
	switch system {
	case bridge.SystemIDDaggerheart:
		systemKey := module.Key{ID: daggerheartdomain.SystemID, Version: daggerheartdomain.SystemVersion}
		return func(characterID string) (bool, string) {
			snapshotAny := state.Systems[systemKey]
			ch, ok := state.Characters[ids.CharacterID(characterID)]
			if !ok {
				return false, "character is missing"
			}
			mod := daggerheartdomain.NewModule()
			return mod.CharacterReady(snapshotAny, ch)
		}
	default:
		return nil
	}
}
