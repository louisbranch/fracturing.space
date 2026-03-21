package aifakes

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// AgentStore is an in-memory AI agent repository fake.
type AgentStore struct {
	Agents map[string]storage.AgentRecord
}

// NewAgentStore creates an initialized agent fake.
func NewAgentStore() *AgentStore {
	return &AgentStore{Agents: make(map[string]storage.AgentRecord)}
}

// PutAgent stores an agent record.
func (s *AgentStore) PutAgent(_ context.Context, record storage.AgentRecord) error {
	for id, existing := range s.Agents {
		if id == record.ID {
			continue
		}
		if !sameNormalizedOwner(record.OwnerUserID, existing.OwnerUserID) {
			continue
		}
		if normalizedLabel(record.Label) == normalizedLabel(existing.Label) {
			return storage.ErrConflict
		}
	}
	s.Agents[record.ID] = record
	return nil
}

// GetAgent returns an agent by ID.
func (s *AgentStore) GetAgent(_ context.Context, agentID string) (storage.AgentRecord, error) {
	rec, ok := s.Agents[agentID]
	if !ok {
		return storage.AgentRecord{}, storage.ErrNotFound
	}
	return rec, nil
}

// ListAgentsByOwner lists agents for an owner.
func (s *AgentStore) ListAgentsByOwner(_ context.Context, ownerUserID string, _ int, _ string) (storage.AgentPage, error) {
	items := make([]storage.AgentRecord, 0)
	for _, rec := range s.Agents {
		if rec.OwnerUserID == ownerUserID {
			items = append(items, rec)
		}
	}
	return storage.AgentPage{Agents: items}, nil
}

// DeleteAgent removes an owned agent.
func (s *AgentStore) DeleteAgent(_ context.Context, ownerUserID string, agentID string) error {
	rec, ok := s.Agents[agentID]
	if !ok || rec.OwnerUserID != ownerUserID {
		return storage.ErrNotFound
	}
	delete(s.Agents, agentID)
	return nil
}
