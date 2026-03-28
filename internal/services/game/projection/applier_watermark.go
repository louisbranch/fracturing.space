package projection

// Watermark concurrency model
//
// Under concurrent projection (e.g. multiple outbox consumers), the watermark
// may temporarily regress because SaveProjectionWatermark uses last-write-wins
// semantics. This is safe because:
//
//  1. Gap detection on each write checks ExpectedNextSeq against the incoming
//     event sequence. If events were skipped, the watermark preserves the gap
//     boundary instead of advancing past it.
//
//  2. DetectProjectionGaps periodically scans for campaigns whose watermark
//     indicates missing events and triggers targeted replay to fill them.
//
// The result is eventual consistency: watermarks may drift briefly under
// concurrency, but gap repair always converges to the correct high-water mark.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	auditevents "github.com/louisbranch/fracturing.space/internal/services/game/observability/audit/events"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

func (a Applier) saveProjectionWatermark(ctx context.Context, evt event.Event) error {
	if !a.shouldSaveProjectionWatermark(evt) {
		return nil
	}

	// Gap detection: load current watermark to check if events were skipped.
	// A gap means replay or re-projection is needed to fill missing events.
	cid := string(evt.CampaignID)
	existing, err := a.Watermarks.GetProjectionWatermark(ctx, cid)
	if err != nil && !errors.Is(err, storage.ErrNotFound) {
		return fmt.Errorf("load projection watermark for gap detection: %w", err)
	}
	expectedNext := evt.Seq + 1
	if err == nil && existing.ExpectedNextSeq > 0 && evt.Seq > existing.ExpectedNextSeq {
		// Preserve the gap boundary instead of advancing past it. This keeps
		// the mid-stream gap visible to DetectProjectionGaps so it can trigger
		// repair without manual CLI invocation.
		a.emitProjectionGapAudit(ctx, cid, existing.ExpectedNextSeq, evt.Seq)
		expectedNext = existing.ExpectedNextSeq
	}

	return a.Watermarks.SaveProjectionWatermark(ctx, storage.ProjectionWatermark{
		CampaignID:      cid,
		AppliedSeq:      evt.Seq,
		ExpectedNextSeq: expectedNext,
		UpdatedAt:       a.nowUTC(),
	})
}

func (a Applier) shouldSaveProjectionWatermark(evt event.Event) bool {
	return a.Watermarks != nil && evt.Seq > 0 && strings.TrimSpace(string(evt.CampaignID)) != ""
}

func (a Applier) nowUTC() time.Time {
	if a.Now == nil {
		return time.Now().UTC()
	}
	return a.Now().UTC()
}

// emitProjectionGapAudit records a projection gap as both a log line and an
// audit event. The log line is retained for backward compatibility with
// existing log-based alerting.
func (a Applier) emitProjectionGapAudit(ctx context.Context, campaignID string, expectedSeq, actualSeq uint64) {
	slog.Warn("projection gap detected",
		"campaign_id", campaignID,
		"expected_seq", expectedSeq,
		"actual_seq", actualSeq,
	)

	if a.Auditor == nil {
		return
	}
	if err := a.Auditor.Emit(ctx, storage.AuditEvent{
		EventName:  auditevents.ProjectionGapDetected,
		Severity:   "WARN",
		CampaignID: campaignID,
		Attributes: map[string]any{
			"expected_seq": expectedSeq,
			"actual_seq":   actualSeq,
		},
	}); err != nil {
		slog.Error("audit emit projection gap", "error", err)
	}
}
