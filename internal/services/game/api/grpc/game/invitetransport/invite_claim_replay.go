package invitetransport

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// loadInviteReplayState replays the invite aggregate from the event journal so
// claim validation does not depend on potentially stale invite projections.
func loadInviteReplayState(ctx context.Context, store storage.EventStore, campaignID string, inviteID string) (invite.State, error) {
	return replayEntityState(
		ctx,
		store,
		campaignID,
		"invite",
		inviteID,
		invite.State{},
		invite.Fold,
	)
}

// loadParticipantReplayState replays one participant aggregate so occupancy
// checks can rely on authoritative history instead of projection lag windows.
func loadParticipantReplayState(ctx context.Context, store storage.EventStore, campaignID string, participantID string) (participant.State, error) {
	return replayEntityState(
		ctx,
		store,
		campaignID,
		"participant",
		participantID,
		participant.State{},
		participant.Fold,
	)
}

// loadCampaignParticipantReplayStates folds all participant events for a
// campaign so claim-time user-binding checks stay authoritative even if the
// claim index projection is missing or behind.
func loadCampaignParticipantReplayStates(ctx context.Context, store storage.EventStore, campaignID string) (map[string]participant.State, error) {
	if store == nil {
		return nil, fmt.Errorf("event store is not configured")
	}
	states := make(map[string]participant.State)
	var cursor uint64
	for {
		page, err := store.ListEventsPage(ctx, storage.ListEventsPageRequest{
			CampaignID: campaignID,
			PageSize:   200,
			CursorSeq:  cursor,
			CursorDir:  "fwd",
			Filter: storage.EventQueryFilter{
				EntityType: "participant",
			},
		})
		if err != nil {
			return nil, err
		}
		for _, evt := range page.Events {
			state := states[evt.EntityID]
			state, err = participant.Fold(state, evt)
			if err != nil {
				return nil, err
			}
			states[evt.EntityID] = state
		}
		if !page.HasNextPage || len(page.Events) == 0 {
			return states, nil
		}
		cursor = page.Events[len(page.Events)-1].Seq
	}
}

// findClaimedParticipantForUser scans replayed participant state to answer the
// conflict question directly from authoritative campaign history.
func findClaimedParticipantForUser(states map[string]participant.State, userID string) (string, bool) {
	normalizedUserID := strings.TrimSpace(userID)
	if normalizedUserID == "" {
		return "", false
	}
	for participantID, state := range states {
		if !participantStateHasActiveUserBinding(state) {
			continue
		}
		if strings.TrimSpace(state.UserID.String()) != normalizedUserID {
			continue
		}
		return participantID, true
	}
	return "", false
}

// participantStateHasActiveUserBinding narrows replayed participant history to
// active seat ownership so past leaves and explicit unbinds do not block claim.
func participantStateHasActiveUserBinding(state participant.State) bool {
	if !state.Joined || state.Left {
		return false
	}
	return strings.TrimSpace(state.UserID.String()) != ""
}

// replayEntityState pages the event journal with entity filters and folds the
// matching events into domain state for write-path preflight checks.
func replayEntityState[T any](
	ctx context.Context,
	store storage.EventStore,
	campaignID string,
	entityType string,
	entityID string,
	state T,
	fold func(T, event.Event) (T, error),
) (T, error) {
	if store == nil {
		return state, fmt.Errorf("event store is not configured")
	}
	var cursor uint64
	for {
		page, err := store.ListEventsPage(ctx, storage.ListEventsPageRequest{
			CampaignID: campaignID,
			PageSize:   200,
			CursorSeq:  cursor,
			CursorDir:  "fwd",
			Filter: storage.EventQueryFilter{
				EntityType: entityType,
				EntityID:   entityID,
			},
		})
		if err != nil {
			return state, err
		}
		for _, evt := range page.Events {
			state, err = fold(state, evt)
			if err != nil {
				return state, err
			}
		}
		if !page.HasNextPage || len(page.Events) == 0 {
			return state, nil
		}
		cursor = page.Events[len(page.Events)-1].Seq
	}
}
