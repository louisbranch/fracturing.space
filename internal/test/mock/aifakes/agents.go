package aifakes

import (
	"context"

	"github.com/louisbranch/fracturing.space/internal/services/ai/agent"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// AgentStore is an in-memory AI agent repository fake.
type AgentStore struct {
	Agents    map[string]agent.Agent
	PutErr    error
	GetErr    error
	ListErr   error
	DeleteErr error
}

// NewAgentStore creates an initialized agent fake.
func NewAgentStore() *AgentStore {
	return &AgentStore{Agents: make(map[string]agent.Agent)}
}

// PutAgent stores an agent.
func (s *AgentStore) PutAgent(_ context.Context, a agent.Agent) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	for id, existing := range s.Agents {
		if id == a.ID {
			continue
		}
		if !sameNormalizedOwner(a.OwnerUserID, existing.OwnerUserID) {
			continue
		}
		if normalizedLabel(a.Label) == normalizedLabel(existing.Label) {
			return storage.ErrConflict
		}
	}
	s.Agents[a.ID] = a
	return nil
}

// GetAgent returns an agent by ID.
func (s *AgentStore) GetAgent(_ context.Context, agentID string) (agent.Agent, error) {
	if s.GetErr != nil {
		return agent.Agent{}, s.GetErr
	}
	a, ok := s.Agents[agentID]
	if !ok {
		return agent.Agent{}, storage.ErrNotFound
	}
	return a, nil
}

// ListAgentsByOwner lists agents for an owner.
func (s *AgentStore) ListAgentsByOwner(_ context.Context, ownerUserID string, _ int, _ string) (agent.Page, error) {
	if s.ListErr != nil {
		return agent.Page{}, s.ListErr
	}
	items := make([]agent.Agent, 0)
	for _, a := range s.Agents {
		if a.OwnerUserID == ownerUserID {
			items = append(items, a)
		}
	}
	return agent.Page{Agents: items}, nil
}

// ListAccessibleAgents returns all agents the user can invoke (owned + shared).
// This fake returns all owned agents; shared access is not modeled.
func (s *AgentStore) ListAccessibleAgents(_ context.Context, userID string, _ int, _ string) (agent.Page, error) {
	if s.ListErr != nil {
		return agent.Page{}, s.ListErr
	}
	items := make([]agent.Agent, 0)
	for _, a := range s.Agents {
		if a.OwnerUserID == userID {
			items = append(items, a)
		}
	}
	return agent.Page{Agents: items}, nil
}

// DeleteAgent removes an owned agent.
func (s *AgentStore) DeleteAgent(_ context.Context, ownerUserID string, agentID string) error {
	if s.DeleteErr != nil {
		return s.DeleteErr
	}
	a, ok := s.Agents[agentID]
	if !ok || a.OwnerUserID != ownerUserID {
		return storage.ErrNotFound
	}
	delete(s.Agents, agentID)
	return nil
}
