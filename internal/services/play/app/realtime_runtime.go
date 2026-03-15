package app

import (
	"context"
	"encoding/json"
	"time"
)

const (
	maxFramePayloadBytes      = 32 * 1024
	maxFramesPerSecond        = 50
	maxDecodeErrorsPerConn    = 3
	maxMessageBodyRunes       = 12000
	maxClientMessageIDLen     = 128
	defaultTypingTTL          = 3 * time.Second
	defaultProjectionRetryTTL = time.Second
)

type wsFrame struct {
	Type      string          `json:"type"`
	RequestID string          `json:"request_id,omitempty"`
	Payload   json.RawMessage `json:"payload"`
}

type typingPayload struct {
	Active bool `json:"active"`
}

type realtimeTimer interface {
	Stop() bool
}

// realtimeRuntime centralizes clock, timer, and retry hooks so realtime
// orchestration can be tested without wall-clock sleeps.
type realtimeRuntime struct {
	now                func() time.Time
	afterFunc          func(time.Duration, func()) realtimeTimer
	sleepUntilRetry    func(context.Context, time.Duration) bool
	typingTTL          time.Duration
	projectionRetryTTL time.Duration
}

type stdlibRealtimeTimer struct {
	timer *time.Timer
}

func defaultRealtimeRuntime() realtimeRuntime {
	return realtimeRuntime{
		now:                time.Now,
		afterFunc:          newStdlibRealtimeTimer,
		sleepUntilRetry:    sleepUntilRetry,
		typingTTL:          defaultTypingTTL,
		projectionRetryTTL: defaultProjectionRetryTTL,
	}
}

func newStdlibRealtimeTimer(delay time.Duration, callback func()) realtimeTimer {
	return stdlibRealtimeTimer{timer: time.AfterFunc(delay, callback)}
}

func (t stdlibRealtimeTimer) Stop() bool {
	if t.timer == nil {
		return false
	}
	return t.timer.Stop()
}

func (r realtimeRuntime) normalize() realtimeRuntime {
	if r.now == nil {
		r.now = time.Now
	}
	if r.afterFunc == nil {
		r.afterFunc = newStdlibRealtimeTimer
	}
	if r.sleepUntilRetry == nil {
		r.sleepUntilRetry = sleepUntilRetry
	}
	if r.typingTTL <= 0 {
		r.typingTTL = defaultTypingTTL
	}
	if r.projectionRetryTTL <= 0 {
		r.projectionRetryTTL = defaultProjectionRetryTTL
	}
	return r
}

func (r realtimeRuntime) nowTime() time.Time {
	return r.now().UTC()
}

func (r realtimeRuntime) newTimer(delay time.Duration, callback func()) realtimeTimer {
	return r.afterFunc(delay, callback)
}

func (r realtimeRuntime) retry(ctx context.Context) bool {
	return r.sleepUntilRetry(ctx, r.projectionRetryTTL)
}

func sleepUntilRetry(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
