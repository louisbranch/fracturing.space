package engine

import (
	"context"
	"errors"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

// ReplayStateLoader replays events to build state.
type ReplayStateLoader struct {
	Events       replay.EventStore
	Checkpoints  replay.CheckpointStore
	Snapshots    StateSnapshotStore
	Applier      replay.Applier
	StateFactory func() any
	Options      replay.Options
}

// StateSnapshotStore loads and saves replay state snapshots keyed by campaign.
type StateSnapshotStore interface {
	GetState(ctx context.Context, campaignID string) (state any, lastSeq uint64, err error)
	SaveState(ctx context.Context, campaignID string, lastSeq uint64, state any) error
}

// ReplayGateStateLoader loads session state via replay.
type ReplayGateStateLoader struct {
	StateLoader ReplayStateLoader
}

// Load replays events to reconstruct state for a campaign.
func (l ReplayStateLoader) Load(ctx context.Context, cmd command.Command) (any, error) {
	if l.Events == nil {
		return nil, replay.ErrEventStoreRequired
	}
	if l.Checkpoints == nil {
		return nil, replay.ErrCheckpointStoreRequired
	}
	if l.Applier == nil {
		return nil, replay.ErrApplierRequired
	}
	var state any
	options := l.Options
	if l.Snapshots != nil {
		snapshotState, snapshotSeq, err := l.Snapshots.GetState(ctx, cmd.CampaignID)
		if err != nil {
			if !errors.Is(err, replay.ErrCheckpointNotFound) {
				return nil, err
			}
		} else {
			state = snapshotState
			if snapshotSeq > options.AfterSeq {
				options.AfterSeq = snapshotSeq
			}
		}
	}
	if l.StateFactory != nil {
		if state == nil {
			state = l.StateFactory()
		}
	}
	result, err := replay.Replay(ctx, l.Events, l.Checkpoints, l.Applier, cmd.CampaignID, state, options)
	if err != nil {
		return nil, err
	}
	return result.State, nil
}

// LoadSession returns the session state for gate checks.
func (l ReplayGateStateLoader) LoadSession(ctx context.Context, campaignID, _ string) (session.State, error) {
	state, err := l.StateLoader.Load(ctx, command.Command{CampaignID: campaignID})
	if err != nil {
		return session.State{}, err
	}
	if state == nil {
		return session.State{}, errors.New("state is required")
	}
	switch typed := state.(type) {
	case aggregate.State:
		return typed.Session, nil
	case *aggregate.State:
		if typed == nil {
			return session.State{}, errors.New("state is required")
		}
		return typed.Session, nil
	default:
		return session.State{}, errors.New("unsupported state type")
	}
}
