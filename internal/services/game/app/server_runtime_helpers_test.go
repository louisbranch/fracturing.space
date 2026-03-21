package app

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestComposeRuntimeStops_ReverseOrderAndSkipNil(t *testing.T) {
	order := make([]int, 0, 3)
	stop := composeRuntimeStops(
		func() { order = append(order, 1) },
		nil,
		func() { order = append(order, 3) },
		func() { order = append(order, 4) },
	)

	stop()

	want := []int{4, 3, 1}
	if len(order) != len(want) {
		t.Fatalf("stop order len = %d, want %d", len(order), len(want))
	}
	for index := range want {
		if order[index] != want[index] {
			t.Fatalf("stop order[%d] = %d, want %d (full=%v)", index, order[index], want[index], order)
		}
	}
}

func TestStartCancelableLoop_StopWaitsForExit(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	exited := make(chan struct{})
	stop := startCancelableLoop(ctx, func(workerCtx context.Context) {
		<-workerCtx.Done()
		close(exited)
	})

	stop()

	select {
	case <-exited:
	case <-time.After(time.Second):
		t.Fatal("expected background loop to exit after stop")
	}
}

func TestRunBatchedPollingLoop_DrainsBacklogThenTicks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	const limit = 10
	var calls atomic.Int32
	done := make(chan struct{})
	go func() {
		defer close(done)
		runBatchedPollingLoop(ctx, 5*time.Millisecond, limit, func() int {
			call := int(calls.Add(1))
			switch call {
			case 1, 2:
				return limit // backlog still full
			default:
				return 0
			}
		})
	}()

	deadline := time.Now().Add(200 * time.Millisecond)
	for time.Now().Before(deadline) {
		if calls.Load() >= 4 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}
	if calls.Load() < 4 {
		t.Fatalf("expected backlog passes and at least one ticker pass, got %d", calls.Load())
	}

	cancel()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected polling loop to stop on context cancellation")
	}
}

func TestRunBatchedPollingLoop_NoopOnInvalidConfig(t *testing.T) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		runBatchedPollingLoop(context.Background(), time.Second, 0, func() int { return 1 })
	}()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("expected invalid config to return immediately")
	}
}
