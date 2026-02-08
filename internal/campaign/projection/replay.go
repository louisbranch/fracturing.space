package projection

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/storage"
)

const replayPageSize = 200

// ReplayOptions configures event replay behavior.
type ReplayOptions struct {
	AfterSeq uint64
	UntilSeq uint64
	Filter   func(event.Event) bool
}

// ReplayCampaign replays events for a campaign and applies projections in order.
func ReplayCampaign(ctx context.Context, eventStore storage.EventStore, applier Applier, campaignID string) (uint64, error) {
	return ReplayCampaignWith(ctx, eventStore, applier, campaignID, ReplayOptions{})
}

// ReplaySnapshot replays snapshot-related events for a campaign and applies projections.
func ReplaySnapshot(ctx context.Context, eventStore storage.EventStore, applier Applier, campaignID string, untilSeq uint64) (uint64, error) {
	return ReplayCampaignWith(ctx, eventStore, applier, campaignID, ReplayOptions{
		UntilSeq: untilSeq,
		Filter: func(evt event.Event) bool {
			return evt.Type == event.TypeCharacterStateChanged || evt.Type == event.TypeGMFearChanged
		},
	})
}

// ReplayCampaignWith replays events with additional filtering and bounds.
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
			if options.UntilSeq > 0 && evt.Seq > options.UntilSeq {
				return lastSeq, nil
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
