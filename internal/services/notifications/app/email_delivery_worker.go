package server

import (
	"context"
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/notifications/storage"
)

const emailDeliveryBatchLimit = 50

type pendingEmailDeliveryStore interface {
	ListPendingDeliveries(ctx context.Context, channel storage.DeliveryChannel, limit int, now time.Time) ([]storage.DeliveryRecord, error)
}

// emailDeliveryWorker periodically observes pending email deliveries so runtime
// can expose backlog visibility before outbound sender wiring is enabled.
type emailDeliveryWorker struct {
	store     pendingEmailDeliveryStore
	pollEvery time.Duration
	now       func() time.Time
	logf      func(string, ...any)
}

func newEmailDeliveryWorker(
	store pendingEmailDeliveryStore,
	pollEvery time.Duration,
	now func() time.Time,
	logf func(string, ...any),
) *emailDeliveryWorker {
	if pollEvery <= 0 {
		pollEvery = defaultEmailDeliveryWorkerPollInterval
	}
	if now == nil {
		now = time.Now
	}
	if logf == nil {
		logf = func(string, ...any) {}
	}
	return &emailDeliveryWorker{
		store:     store,
		pollEvery: pollEvery,
		now:       now,
		logf:      logf,
	}
}

func (w *emailDeliveryWorker) Run(ctx context.Context) {
	if w == nil {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	ticker := time.NewTicker(w.pollEvery)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.Observe(ctx); err != nil {
				w.logf("notifications email delivery worker: %v", err)
			}
		}
	}
}

func (w *emailDeliveryWorker) Observe(ctx context.Context) error {
	if w == nil || w.store == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	pending, err := w.store.ListPendingDeliveries(
		ctx,
		storage.DeliveryChannelEmail,
		emailDeliveryBatchLimit,
		w.now().UTC(),
	)
	if err != nil {
		return fmt.Errorf("list pending email deliveries: %w", err)
	}
	if len(pending) > 0 {
		w.logf("notifications email delivery worker observed %d pending deliveries (sender scaffold not configured)", len(pending))
	}
	return nil
}
