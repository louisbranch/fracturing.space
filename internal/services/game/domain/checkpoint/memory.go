package checkpoint

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/system"
)

var (
	// ErrCampaignIDRequired indicates a missing campaign id.
	ErrCampaignIDRequired = errors.New("campaign id is required")
)

// Memory stores checkpoints in memory.
type Memory struct {
	mu          sync.Mutex
	checkpoints map[string]replay.Checkpoint
	states      map[string]any
}

// NewMemory creates a new in-memory checkpoint store.
func NewMemory() *Memory {
	return &Memory{
		checkpoints: make(map[string]replay.Checkpoint),
		states:      make(map[string]any),
	}
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

// GetState retrieves a replay state snapshot and its sequence.
func (m *Memory) GetState(ctx context.Context, campaignID string) (any, uint64, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
	}
	if m == nil {
		return nil, 0, errors.New("checkpoint store is required")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, 0, ErrCampaignIDRequired
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	snapshot, ok := m.states[campaignID]
	if !ok {
		return nil, 0, replay.ErrCheckpointNotFound
	}
	checkpoint, ok := m.checkpoints[campaignID]
	if !ok {
		return nil, 0, replay.ErrCheckpointNotFound
	}

	return cloneSnapshotState(snapshot), checkpoint.LastSeq, nil
}

// SaveState persists a replay state snapshot.
func (m *Memory) SaveState(ctx context.Context, campaignID string, lastSeq uint64, state any) error {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return err
		}
	}
	if m == nil {
		return errors.New("checkpoint store is required")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return ErrCampaignIDRequired
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.states[campaignID] = cloneSnapshotState(state)
	m.checkpoints[campaignID] = replay.Checkpoint{
		CampaignID: campaignID,
		LastSeq:    lastSeq,
		UpdatedAt:  time.Now().UTC(),
	}
	return nil
}

func cloneSnapshotState(state any) any {
	switch typed := state.(type) {
	case aggregate.State:
		return cloneAggregateState(typed)
	case *aggregate.State:
		if typed == nil {
			return aggregate.State{}
		}
		return cloneAggregateState(*typed)
	default:
		return state
	}
}

func cloneAggregateState(source aggregate.State) aggregate.State {
	cloned := source
	if source.Participants != nil {
		cloned.Participants = make(map[string]participant.State, len(source.Participants))
		for key, value := range source.Participants {
			cloned.Participants[key] = value
		}
	}
	if source.Characters != nil {
		cloned.Characters = make(map[string]character.State, len(source.Characters))
		for key, value := range source.Characters {
			cloned.Characters[key] = value
		}
	}
	if source.Invites != nil {
		cloned.Invites = make(map[string]invite.State, len(source.Invites))
		for key, value := range source.Invites {
			cloned.Invites[key] = value
		}
	}
	if source.Systems != nil {
		cloned.Systems = make(map[system.Key]any, len(source.Systems))
		for key, value := range source.Systems {
			cloned.Systems[key] = value
		}
	}
	return cloned
}
