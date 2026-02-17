// Package replay defines deterministic reconstruction boundaries for event-sourced flows.
//
// Replay is how write-path state is rebuilt from immutable history and how
// projection rebuilds are repaired consistently after partial failures.
package replay

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

const defaultPageSize = 200

var (
	// ErrEventStoreRequired indicates a missing event store.
	ErrEventStoreRequired = errors.New("event store is required")
	// ErrCheckpointStoreRequired indicates a missing checkpoint store.
	ErrCheckpointStoreRequired = errors.New("checkpoint store is required")
	// ErrApplierRequired indicates a missing applier.
	ErrApplierRequired = errors.New("applier is required")
	// ErrCampaignIDRequired indicates a missing campaign id.
	ErrCampaignIDRequired = errors.New("campaign id is required")
	// ErrCheckpointNotFound indicates no checkpoint exists yet.
	ErrCheckpointNotFound = errors.New("checkpoint not found")
)

// EventStore exposes read access to campaign event history for deterministic rebuild.
type EventStore interface {
	ListEvents(ctx context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error)
}

// CheckpointStore stores replay progress so reconstruction can resume from a cursor.
type CheckpointStore interface {
	Get(ctx context.Context, campaignID string) (Checkpoint, error)
	Save(ctx context.Context, checkpoint Checkpoint) error
}

// Applier applies a domain event into the in-memory replay target.
type Applier interface {
	Apply(state any, evt event.Event) (any, error)
}

// Checkpoint captures the last applied sequence for a campaign.
type Checkpoint struct {
	CampaignID string
	LastSeq    uint64
	UpdatedAt  time.Time
}

// Options constrains replay work for maintenance windows or partial repair.
type Options struct {
	AfterSeq uint64
	UntilSeq uint64
	PageSize int
}

// Result captures replay outcomes and the new cursor for checkpoint updates.
type Result struct {
	State   any
	LastSeq uint64
	Applied int
}

// Replay rebuilds aggregate state from ordered events and persists checkpoints as it goes.
//
// It is the shared safety net used by both startup recovery and projection rebuilds:
// sequence gaps fail fast, and each checkpoint represents the last known-correct seq.
func Replay(ctx context.Context, store EventStore, checkpoints CheckpointStore, applier Applier, campaignID string, state any, options Options) (Result, error) {
	if store == nil {
		return Result{}, ErrEventStoreRequired
	}
	if checkpoints == nil {
		return Result{}, ErrCheckpointStoreRequired
	}
	if applier == nil {
		return Result{}, ErrApplierRequired
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return Result{}, ErrCampaignIDRequired
	}

	checkpointSeq := uint64(0)
	checkpoint, err := checkpoints.Get(ctx, campaignID)
	if err != nil {
		if !errors.Is(err, ErrCheckpointNotFound) {
			return Result{}, err
		}
	} else {
		checkpointSeq = checkpoint.LastSeq
	}

	lastSeq := options.AfterSeq
	if checkpointSeq > lastSeq {
		lastSeq = checkpointSeq
	}
	pageSize := options.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}

	result := Result{State: state, LastSeq: lastSeq}
	for {
		events, err := store.ListEvents(ctx, campaignID, result.LastSeq, pageSize)
		if err != nil {
			return result, err
		}
		if len(events) == 0 {
			return result, nil
		}
		for _, evt := range events {
			if options.UntilSeq > 0 && evt.Seq > options.UntilSeq {
				return result, nil
			}
			expectedSeq := result.LastSeq + 1
			if evt.Seq != expectedSeq {
				return result, fmt.Errorf("event sequence gap: expected %d got %d", expectedSeq, evt.Seq)
			}
			nextState, err := applier.Apply(result.State, evt)
			if err != nil {
				return result, err
			}
			result.State = nextState
			result.LastSeq = evt.Seq
			result.Applied++
			if err := checkpoints.Save(ctx, Checkpoint{CampaignID: campaignID, LastSeq: result.LastSeq, UpdatedAt: time.Now().UTC()}); err != nil {
				return result, err
			}
		}
	}
}
