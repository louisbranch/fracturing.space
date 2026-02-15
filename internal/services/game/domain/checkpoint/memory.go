package checkpoint

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
)

var (
	// ErrCampaignIDRequired indicates a missing campaign id.
	ErrCampaignIDRequired = errors.New("campaign id is required")
)

// Memory stores checkpoints in memory.
type Memory struct {
	mu          sync.Mutex
	checkpoints map[string]replay.Checkpoint
}

// NewMemory creates a new in-memory checkpoint store.
func NewMemory() *Memory {
	return &Memory{checkpoints: make(map[string]replay.Checkpoint)}
}

// Get retrieves a checkpoint by campaign id.
func (m *Memory) Get(ctx context.Context, campaignID string) (replay.Checkpoint, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return replay.Checkpoint{}, err
		}
	}
	if m == nil {
		return replay.Checkpoint{}, errors.New("checkpoint store is required")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return replay.Checkpoint{}, ErrCampaignIDRequired
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	checkpoint, ok := m.checkpoints[campaignID]
	if !ok {
		return replay.Checkpoint{}, replay.ErrCheckpointNotFound
	}
	return checkpoint, nil
}

// Save persists a checkpoint.
func (m *Memory) Save(ctx context.Context, checkpoint replay.Checkpoint) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if m == nil {
		return errors.New("checkpoint store is required")
	}
	campaignID := strings.TrimSpace(checkpoint.CampaignID)
	if campaignID == "" {
		return ErrCampaignIDRequired
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	checkpoint.CampaignID = campaignID
	m.checkpoints[campaignID] = checkpoint
	return nil
}
