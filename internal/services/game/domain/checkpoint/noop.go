package checkpoint

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
)

// Noop ignores stored checkpoints for replay.
type Noop struct{}

// NewNoop creates a checkpoint store that never reuses checkpoints.
func NewNoop() *Noop {
	return &Noop{}
}

// Get always reports that no checkpoint exists.
func (n *Noop) Get(ctx context.Context, _ string) (replay.Checkpoint, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return replay.Checkpoint{}, err
		}
	}
	return replay.Checkpoint{}, replay.ErrCheckpointNotFound
}

// Save is a no-op.
func (n *Noop) Save(ctx context.Context, _ replay.Checkpoint) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return nil
}

// GetState always reports that no snapshot exists.
func (n *Noop) GetState(ctx context.Context, _ string) (any, uint64, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
	}
	return nil, 0, replay.ErrCheckpointNotFound
}

// SaveState is a no-op.
func (n *Noop) SaveState(ctx context.Context, _ string, _ uint64, _ any) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	return nil
}
