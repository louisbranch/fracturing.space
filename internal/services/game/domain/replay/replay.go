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

// EventStore lists events for replay.
type EventStore interface {
	ListEvents(ctx context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error)
}

// CheckpointStore manages replay checkpoints.
type CheckpointStore interface {
	Get(ctx context.Context, campaignID string) (Checkpoint, error)
	Save(ctx context.Context, checkpoint Checkpoint) error
}

// Applier applies a domain event to projection state.
type Applier interface {
	Apply(state any, evt event.Event) (any, error)
}

// Checkpoint captures the last applied sequence for a campaign.
type Checkpoint struct {
	CampaignID string
	LastSeq    uint64
	UpdatedAt  time.Time
}

// Options configures replay behavior.
type Options struct {
	AfterSeq uint64
	UntilSeq uint64
	PageSize int
}

// Result captures replay outcomes.
type Result struct {
	State   any
	LastSeq uint64
	Applied int
}

// Replay replays events in order and updates checkpoints after each apply.
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
