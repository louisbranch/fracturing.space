package app

import (
	"context"
	"encoding/json"
	"time"
)

const (
	maxFramePayloadBytes        = 32 * 1024
	maxFramesPerSecond          = 50
	maxDecodeErrorsPerConn      = 3
	maxMessageBodyRunes         = 12000
	maxClientMessageIDLen       = 128
	defaultTypingTTL            = 3 * time.Second
	defaultProjectionRetryTTL   = time.Second
	maxProjectionRetryTTL       = 30 * time.Second
	projectionBackoffMultiplier = 2.0
)

// wsRateLimiter enforces a fixed-window frame rate per WebSocket connection.
type wsRateLimiter struct {
	now         func() time.Time
	windowStart time.Time
	count       int
}

func newWSRateLimiter(now func() time.Time) wsRateLimiter {
	return wsRateLimiter{now: now, windowStart: now()}
}

// allow returns true if the next frame is within the rate limit.
func (r *wsRateLimiter) allow() bool {
	now := r.now()
	if now.Sub(r.windowStart) >= time.Second {
		r.windowStart = now
		r.count = 0
	}
	r.count++
	return r.count <= maxFramesPerSecond
}

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

func (r realtimeRuntime) retryWithDelay(ctx context.Context, delay time.Duration) bool {
	return r.sleepUntilRetry(ctx, delay)
}

// backoff doubles the delay up to maxProjectionRetryTTL for exponential backoff
// on projection subscription reconnects.
func (r realtimeRuntime) backoff(current time.Duration) time.Duration {
	next := time.Duration(float64(current) * projectionBackoffMultiplier)
	if next > maxProjectionRetryTTL {
		next = maxProjectionRetryTTL
	}
	return next
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
