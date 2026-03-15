package server

import (
	"context"
	"time"
)

type runtimeStopFunc func()

func noopStop() {}

// composeRuntimeStops combines stop funcs into a single stop closure that runs
// in reverse startup order.
func composeRuntimeStops(stops ...runtimeStopFunc) runtimeStopFunc {
	return func() {
		for index := len(stops) - 1; index >= 0; index-- {
			if stops[index] == nil {
				continue
			}
			stops[index]()
		}
	}
}

// startCancelableLoop launches a cancellable background goroutine and returns
// a stop function that blocks until the goroutine exits.
func startCancelableLoop(ctx context.Context, run func(context.Context)) runtimeStopFunc {
	if run == nil {
		return noopStop
	}
	if ctx == nil {
		ctx = context.Background()
	}
	workerCtx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})
	go func() {
		defer close(done)
		run(workerCtx)
	}()
	return func() {
		cancel()
		<-done
	}
}

// runBatchedPollingLoop drains work in immediate batches, then polls on an
// interval until context cancellation.
func runBatchedPollingLoop(
	ctx context.Context,
	interval time.Duration,
	batchLimit int,
	runPass func() int,
) {
	if runPass == nil || interval <= 0 || batchLimit <= 0 {
		return
	}
	if ctx == nil {
		ctx = context.Background()
	}

	for {
		if runPass() < batchLimit {
			break
		}
		select {
		case <-ctx.Done():
			return
		default:
		}
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			runPass()
		}
	}
}
