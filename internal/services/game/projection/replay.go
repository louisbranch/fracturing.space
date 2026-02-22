package projection

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

const replayPageSize = 200

// ReplayOptions configures the projection rebuild cursor and payload scope.
type ReplayOptions struct {
	// AfterSeq starts replay after this campaign sequence.
	AfterSeq uint64
	// UntilSeq stops replay when events exceed this sequence.
	UntilSeq uint64
	// Filter returns true for events that should be projected.
	Filter func(event.Event) bool
}

// ReplayCampaign rebuilds campaign-level projections from event 0 onward.
func ReplayCampaign(ctx context.Context, eventStore storage.EventStore, applier Applier, campaignID string) (uint64, error) {
	return ReplayCampaignWith(ctx, eventStore, applier, campaignID, ReplayOptions{})
}

// ReplaySnapshot replays snapshot-bearing events only, useful for state reconstruction
// paths that depend on system snapshots.
func ReplaySnapshot(ctx context.Context, eventStore storage.EventStore, applier Applier, campaignID string, untilSeq uint64) (uint64, error) {
	return ReplayCampaignWith(ctx, eventStore, applier, campaignID, ReplayOptions{
		UntilSeq: untilSeq,
		Filter: func(evt event.Event) bool {
			return evt.SystemID != ""
		},
	})
}

// ReplayCampaignWith applies a bounded replay stream into the projection applier.
//
// It exists for operational recovery: one code path for full rebuild and one for
// bounded/specialized replay flows.
func ReplayCampaignWith(ctx context.Context, eventStore storage.EventStore, applier Applier, campaignID string, options ReplayOptions) (uint64, error) {
	if eventStore == nil {
		return 0, fmt.Errorf("event store is not configured")
	}
	if strings.TrimSpace(campaignID) == "" {
		return 0, fmt.Errorf("campaign id is required")
	}

	lastSeq := options.AfterSeq
	for {
		events, err := eventStore.ListEvents(ctx, campaignID, lastSeq, replayPageSize)
		if err != nil {
			return lastSeq, err
		}
		if len(events) == 0 {
			return lastSeq, nil
		}
		for _, evt := range events {
			if err := ctx.Err(); err != nil {
				return lastSeq, err
			}
			if options.UntilSeq > 0 && evt.Seq > options.UntilSeq {
				return lastSeq, nil
			}
			// Detect journal gaps: projection replay must see every event
			// in order. A missing sequence indicates storage corruption or
			// a non-contiguous ListEvents result.
			expectedSeq := lastSeq + 1
			if evt.Seq != expectedSeq {
				return lastSeq, fmt.Errorf("projection replay sequence gap: expected %d got %d", expectedSeq, evt.Seq)
			}
			lastSeq = evt.Seq
			if options.Filter != nil && !options.Filter(evt) {
				continue
			}
			if err := applier.Apply(ctx, evt); err != nil {
				return lastSeq, err
			}
		}
	}
}
