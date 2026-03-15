package checkpoint

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/aggregate"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/module"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/replay"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/scene"
)

var (
	// ErrCampaignIDRequired indicates a missing campaign id.
	// Canonical definition in domain/ids; re-exported for caller compatibility.
	ErrCampaignIDRequired = ids.ErrCampaignIDRequired
)

// Memory stores checkpoints in memory.
type Memory struct {
	mu          sync.Mutex
	checkpoints map[string]replay.Checkpoint
	states      map[string]any

	// Clock returns the current time. Defaults to time.Now in NewMemory.
	// Callers may override this for deterministic tests.
	Clock func() time.Time
}

// NewMemory creates a new in-memory checkpoint store.
func NewMemory() *Memory {
	return &Memory{
		checkpoints: make(map[string]replay.Checkpoint),
		states:      make(map[string]any),
		Clock:       time.Now,
	}
}

// Get retrieves a checkpoint by campaign id.
func (m *Memory) Get(ctx context.Context, campaignID string) (replay.Checkpoint, error) {
	if err := ctx.Err(); err != nil {
		return replay.Checkpoint{}, err
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
	if err := ctx.Err(); err != nil {
		return err
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
	if err := ctx.Err(); err != nil {
		return nil, 0, err
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

	cloned, err := cloneSnapshotState(snapshot)
	if err != nil {
		return nil, 0, err
	}
	return cloned, checkpoint.LastSeq, nil
}

// SaveState persists a replay state snapshot.
func (m *Memory) SaveState(ctx context.Context, campaignID string, lastSeq uint64, state any) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if m == nil {
		return errors.New("checkpoint store is required")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return ErrCampaignIDRequired
	}

	cloned, err := cloneSnapshotState(state)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.states[campaignID] = cloned
	m.checkpoints[campaignID] = replay.Checkpoint{
		CampaignID: campaignID,
		LastSeq:    lastSeq,
		UpdatedAt:  m.Clock().UTC(),
	}
	return nil
}

func cloneSnapshotState(state any) (any, error) {
	switch typed := state.(type) {
	case aggregate.State:
		return cloneAggregateState(typed), nil
	case *aggregate.State:
		if typed == nil {
			return aggregate.State{}, nil
		}
		return cloneAggregateState(*typed), nil
	default:
		return nil, fmt.Errorf("checkpoint: unhandled state type %T — add a clone case", state)
	}
}

func cloneAggregateState(source aggregate.State) aggregate.State {
	cloned := source
	if source.Participants != nil {
		cloned.Participants = make(map[ids.ParticipantID]participant.State, len(source.Participants))
		for key, value := range source.Participants {
			cloned.Participants[key] = value
		}
	}
	if source.Characters != nil {
		cloned.Characters = make(map[ids.CharacterID]character.State, len(source.Characters))
		for key, value := range source.Characters {
			cloned.Characters[key] = value
		}
	}
	if source.Invites != nil {
		cloned.Invites = make(map[ids.InviteID]invite.State, len(source.Invites))
		for key, value := range source.Invites {
			cloned.Invites[key] = value
		}
	}
	if source.Scenes != nil {
		cloned.Scenes = make(map[ids.SceneID]scene.State, len(source.Scenes))
		for key, value := range source.Scenes {
			cloned.Scenes[key] = cloneSceneState(value)
		}
	}
	if source.Systems != nil {
		cloned.Systems = make(map[module.Key]any, len(source.Systems))
		for key, value := range source.Systems {
			cloned.Systems[key] = value
		}
	}
	return cloned
}

func cloneSceneState(source scene.State) scene.State {
	cloned := source
	if source.Characters != nil {
		cloned.Characters = make(map[ids.CharacterID]bool, len(source.Characters))
		for k, v := range source.Characters {
			cloned.Characters[k] = v
		}
	}
	if source.PlayerPhaseActingCharacters != nil {
		cloned.PlayerPhaseActingCharacters = append([]ids.CharacterID(nil), source.PlayerPhaseActingCharacters...)
	}
	if source.PlayerPhaseActingParticipants != nil {
		cloned.PlayerPhaseActingParticipants = make(map[ids.ParticipantID]bool, len(source.PlayerPhaseActingParticipants))
		for k, v := range source.PlayerPhaseActingParticipants {
			cloned.PlayerPhaseActingParticipants[k] = v
		}
	}
	if source.PlayerPhaseSlots != nil {
		cloned.PlayerPhaseSlots = make(map[ids.ParticipantID]scene.PlayerPhaseSlot, len(source.PlayerPhaseSlots))
		for k, v := range source.PlayerPhaseSlots {
			slot := v
			if v.CharacterIDs != nil {
				slot.CharacterIDs = append([]ids.CharacterID(nil), v.CharacterIDs...)
			}
			if v.ReviewCharacterIDs != nil {
				slot.ReviewCharacterIDs = append([]ids.CharacterID(nil), v.ReviewCharacterIDs...)
			}
			cloned.PlayerPhaseSlots[k] = slot
		}
	}
	return cloned
}
