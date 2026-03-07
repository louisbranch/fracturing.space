package projection

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a Applier) saveProjectionWatermark(ctx context.Context, evt event.Event) error {
	if !a.shouldSaveProjectionWatermark(evt) {
		return nil
	}

	// Gap detection: load current watermark to check if events were skipped.
	// A gap means replay or re-projection is needed to fill missing events.
	existing, err := a.Watermarks.GetProjectionWatermark(ctx, evt.CampaignID)
	expectedNext := evt.Seq + 1
	if err == nil && existing.ExpectedNextSeq > 0 && evt.Seq > existing.ExpectedNextSeq {
		// Preserve the gap boundary instead of advancing past it. This keeps
		// the mid-stream gap visible to DetectProjectionGaps so it can trigger
		// repair without manual CLI invocation.
		log.Printf("projection gap detected for campaign %s: expected seq %d but got %d",
			evt.CampaignID, existing.ExpectedNextSeq, evt.Seq)
		expectedNext = existing.ExpectedNextSeq
	}

	return a.Watermarks.SaveProjectionWatermark(ctx, storage.ProjectionWatermark{
		CampaignID:      evt.CampaignID,
		AppliedSeq:      evt.Seq,
		ExpectedNextSeq: expectedNext,
		UpdatedAt:       a.nowUTC(),
	})
}

func (a Applier) shouldSaveProjectionWatermark(evt event.Event) bool {
	return a.Watermarks != nil && evt.Seq > 0 && strings.TrimSpace(evt.CampaignID) != ""
}

func (a Applier) nowUTC() time.Time {
	if a.Now == nil {
		return time.Now().UTC()
	}
	return a.Now().UTC()
}
