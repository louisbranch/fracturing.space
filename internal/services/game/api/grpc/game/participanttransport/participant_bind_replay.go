package participanttransport

import (
	"context"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// replayParticipantState replays one participant aggregate from the event
// journal so bind-time occupancy checks use authoritative history rather than
// potentially stale projections.
func replayParticipantState(ctx context.Context, store storage.EventHistoryStore, campaignID, participantID string) (participant.State, error) {
	return replayEntity(ctx, store, campaignID, "participant", participantID, participant.State{}, participant.Fold)
}

// replayEntity pages the event journal with entity filters and folds matching
// events into domain state.
func replayEntity[T any](
	ctx context.Context,
	store storage.EventHistoryStore,
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
