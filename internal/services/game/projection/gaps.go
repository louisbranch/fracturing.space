package projection

import (
	"context"
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// GapRepairResult reports the outcome of replaying a single projection gap.
type GapRepairResult struct {
	CampaignID     string
	EventsReplayed int
}

// RepairProjectionGaps detects campaigns where projections are behind the
// event journal and replays the missing events to close the gap.
func RepairProjectionGaps(ctx context.Context, watermarks storage.ProjectionWatermarkStore, eventStore storage.EventStore, applier Applier) ([]GapRepairResult, error) {
	gaps, err := DetectProjectionGaps(ctx, watermarks, eventStore)
	if err != nil {
		return nil, fmt.Errorf("detect gaps: %w", err)
	}
	var results []GapRepairResult
	for _, gap := range gaps {
		lastSeq, err := ReplayCampaignWith(ctx, eventStore, applier, gap.CampaignID, ReplayOptions{
			AfterSeq: gap.WatermarkSeq,
		})
		if err != nil {
			return results, fmt.Errorf("replay campaign %s: %w", gap.CampaignID, err)
		}
		replayed := int(lastSeq - gap.WatermarkSeq)
		if replayed > 0 {
			results = append(results, GapRepairResult{
				CampaignID:     gap.CampaignID,
				EventsReplayed: replayed,
			})
		}
	}
	return results, nil
}

// EventHighWaterStore provides read access to the latest event sequence per campaign.
type EventHighWaterStore interface {
	GetLatestEventSeq(ctx context.Context, campaignID string) (uint64, error)
}

// ProjectionGap describes a campaign whose projection watermark is behind
// the event journal high-water mark.
type ProjectionGap struct {
	CampaignID   string
	WatermarkSeq uint64
	JournalSeq   uint64
}

// DetectProjectionGaps compares projection watermarks against event journal
// high-water marks and returns campaigns that have unapplied events.
func DetectProjectionGaps(ctx context.Context, watermarks storage.ProjectionWatermarkStore, events EventHighWaterStore) ([]ProjectionGap, error) {
	if watermarks == nil {
		return nil, fmt.Errorf("watermark store is required")
	}
	if events == nil {
		return nil, fmt.Errorf("event store is required")
	}
	wms, err := watermarks.ListProjectionWatermarks(ctx)
	if err != nil {
		return nil, fmt.Errorf("list projection watermarks: %w", err)
	}
	var gaps []ProjectionGap
	for _, wm := range wms {
		journalSeq, err := events.GetLatestEventSeq(ctx, wm.CampaignID)
		if err != nil {
			return nil, fmt.Errorf("get latest event seq for %s: %w", wm.CampaignID, err)
		}
		if journalSeq > wm.AppliedSeq {
			gaps = append(gaps, ProjectionGap{
				CampaignID:   wm.CampaignID,
				WatermarkSeq: wm.AppliedSeq,
				JournalSeq:   journalSeq,
			})
		}
	}
	return gaps, nil
}
