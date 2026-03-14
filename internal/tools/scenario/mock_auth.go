package scenario

import (
	"fmt"
	"sync"
)

// MockAuth provides permissive, in-memory auth behaviors for scenario runs.
type MockAuth struct {
	mu   sync.Mutex
	next int
}

// NewMockAuth returns a permissive auth helper for scenarios.
func NewMockAuth() *MockAuth {
	return &MockAuth{}
}

// CreateUser returns a synthetic user id for standalone scenario runs.
func (m *MockAuth) CreateUser(_ string) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.next++
	return fmt.Sprintf("user_mock_%d", m.next), nil
}

// IssueJoinGrant returns a synthetic join grant token.
func (m *MockAuth) IssueJoinGrant(_ string, _ string, _ string, _ string) string {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.next++
	return fmt.Sprintf("grant_mock_%d", m.next)
}

// ClaimInvite is a permissive no-op for scenario runs.
func (m *MockAuth) ClaimInvite(_ string, _ string, _ string) error {
	return nil
}
