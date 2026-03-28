package engine

import (
	"context"
	"errors"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

// ReplayStateLoader replays events to build state for command handling.
//
// It is intentionally thin and composable: checkpoints/snapshots and an applier
// produce deterministic state for the current command, whether from scratch or from
// a cached prefix.
type ReplayStateLoader struct {
	Events       replay.EventStore
	Checkpoints  replay.CheckpointStore
	Snapshots    StateSnapshotStore
	Folder       replay.Folder
	StateFactory func() any
	Options      replay.Options
}

// StateSnapshotStore loads and saves replay state snapshots keyed by campaign.
type StateSnapshotStore interface {
	GetState(ctx context.Context, campaignID string) (state any, lastSeq uint64, err error)
	SaveState(ctx context.Context, campaignID string, lastSeq uint64, state any) error
}

// ReplayGateStateLoader exposes session-only state for gate checks.
type ReplayGateStateLoader struct {
	StateLoader ReplayStateLoader
}

// noCheckpointStore disables replay checkpointing for fresh loads so callers
// can reconstruct authoritative state directly from the event journal.
type noCheckpointStore struct{}

func (noCheckpointStore) Get(context.Context, string) (replay.Checkpoint, error) {
	return replay.Checkpoint{}, replay.ErrCheckpointNotFound
}

func (noCheckpointStore) Save(context.Context, replay.Checkpoint) error {
	return nil
}

// Load replays events to reconstruct state for a campaign.
//
// The load flow is the same source used at runtime and during command handling,
// which makes command outcomes reproducible in replay mode.
func (l ReplayStateLoader) Load(ctx context.Context, cmd command.Command) (any, error) {
	if l.Events == nil {
		return nil, replay.ErrEventStoreRequired
	}
	if l.Checkpoints == nil {
		return nil, replay.ErrCheckpointStoreRequired
	}
	if l.Folder == nil {
		return nil, replay.ErrFolderRequired
	}
	if l.StateFactory == nil {
		return nil, ErrStateFactoryRequired
	}
	var state any
	options := l.Options
	checkpoints := l.Checkpoints
	if l.Snapshots != nil {
		snapshotState, snapshotSeq, err := l.Snapshots.GetState(ctx, string(cmd.CampaignID))
		if err != nil {
			if !errors.Is(err, replay.ErrCheckpointNotFound) {
				return nil, err
			}
		} else {
			state = snapshotState
			if snapshotSeq > options.AfterSeq {
				options.AfterSeq = snapshotSeq
			}
			// Never allow replay cursor to outrun the state represented by the
			// loaded snapshot. A stale checkpoint ahead of snapshot sequence would
			// otherwise skip events and corrupt reconstructed state.
			checkpoints = checkpointCapStore{
				base:   l.Checkpoints,
				maxSeq: snapshotSeq,
			}
		}
	}
	if state == nil {
		state = l.StateFactory()
	}
	result, err := replay.Replay(ctx, l.Events, checkpoints, l.Folder, string(cmd.CampaignID), state, options)
	if err != nil {
		return nil, err
	}
	return result.State, nil
}

// LoadFresh reconstructs state directly from the journal without using cached
// snapshots or checkpoints. Handlers use this to recover from stale cached
// state before returning a domain rejection.
func (l ReplayStateLoader) LoadFresh(ctx context.Context, cmd command.Command) (any, error) {
	fresh := l
	fresh.Checkpoints = noCheckpointStore{}
	fresh.Snapshots = nil
	return fresh.Load(ctx, cmd)
}

// checkpointCapStore forwards checkpoint writes and caps checkpoint reads to a
// maximum sequence so replay cannot skip events that are not represented by the
// in-memory state seed.
type checkpointCapStore struct {
	base   replay.CheckpointStore
	maxSeq uint64
}

func (s checkpointCapStore) Get(ctx context.Context, campaignID string) (replay.Checkpoint, error) {
	checkpoint, err := s.base.Get(ctx, campaignID)
	if err != nil {
		return replay.Checkpoint{}, err
	}
	if checkpoint.LastSeq > s.maxSeq {
		checkpoint.LastSeq = s.maxSeq
	}
	return checkpoint, nil
}

func (s checkpointCapStore) Save(ctx context.Context, checkpoint replay.Checkpoint) error {
	return s.base.Save(ctx, checkpoint)
}

// LoadSession returns the session state for gate checks.
//
// The generic aggregate is narrowed to session only because gate policy is always
// session-scoped by design.
func (l ReplayGateStateLoader) LoadSession(ctx context.Context, campaignID, _ string) (session.State, error) {
	state, err := l.StateLoader.Load(ctx, command.Command{CampaignID: ids.CampaignID(campaignID)})
	if err != nil {
		return session.State{}, err
	}
	if state == nil {
		return session.State{}, ErrStateRequired
	}
	switch typed := state.(type) {
	case aggregate.State:
		return typed.Session, nil
	case *aggregate.State:
		if typed == nil {
			return session.State{}, ErrStateRequired
		}
		return typed.Session, nil
	default:
		return session.State{}, ErrUnsupportedStateType
	}
}

// LoadScene returns the scene state for scene-scoped gate checks.
func (l ReplayGateStateLoader) LoadScene(ctx context.Context, campaignID, sceneID string) (scene.State, error) {
	state, err := l.StateLoader.Load(ctx, command.Command{CampaignID: ids.CampaignID(campaignID)})
	if err != nil {
		return scene.State{}, err
	}
	if state == nil {
		return scene.State{}, ErrStateRequired
	}
	switch typed := state.(type) {
	case aggregate.State:
		if s, ok := typed.Scenes[ids.SceneID(sceneID)]; ok {
			return s, nil
		}
		return scene.State{}, nil
	case *aggregate.State:
		if typed == nil {
			return scene.State{}, ErrStateRequired
		}
		if s, ok := typed.Scenes[ids.SceneID(sceneID)]; ok {
			return s, nil
		}
		return scene.State{}, nil
	default:
		return scene.State{}, ErrUnsupportedStateType
	}
}
