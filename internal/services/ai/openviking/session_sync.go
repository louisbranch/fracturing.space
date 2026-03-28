package openviking

import (
	"context"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

type sessionClient interface {
	AddMessage(ctx context.Context, sessionID string, input AddMessageInput) error
	Used(ctx context.Context, sessionID string, input UsedInput) error
	Commit(ctx context.Context, sessionID string, wait bool) (CommitResult, error)
}

// TurnSyncInput carries the turn data mirrored into an OpenViking session.
type TurnSyncInput struct {
	CampaignID        string
	SessionID         string
	ParticipantID     string
	UserText          string
	AssistantText     string
	RetrievedContexts []orchestration.RetrievedContext
}

// SessionSync mirrors completed turns into OpenViking conversation sessions.
type SessionSync struct {
	client sessionClient
	mode   IntegrationMode
}

// NewSessionSync builds one session sync helper from an OpenViking client.
func NewSessionSync(client sessionClient, mode IntegrationMode) (*SessionSync, error) {
	if client == nil {
		return nil, fmt.Errorf("openviking session client is required")
	}
	normalized, err := ParseIntegrationMode(string(mode))
	if err != nil {
		return nil, err
	}
	return &SessionSync{client: client, mode: normalized}, nil
}

// SyncTurn writes one user/assistant turn pair plus retrieved-context
// references to OpenViking and commits the session inline.
func (s *SessionSync) SyncTurn(ctx context.Context, input TurnSyncInput) error {
	if s == nil || s.client == nil {
		return nil
	}
	sessionID := StableSessionID(input.CampaignID, input.SessionID, input.ParticipantID)
	if text := strings.TrimSpace(input.UserText); text != "" {
		if err := s.client.AddMessage(ctx, sessionID, AddMessageInput{
			Role:    "user",
			Content: text,
		}); err != nil {
			return err
		}
	}

	parts := make([]MessagePart, 0, len(input.RetrievedContexts)+1)
	if text := strings.TrimSpace(input.AssistantText); text != "" {
		parts = append(parts, MessagePart{Type: "text", Text: text})
	}
	for _, item := range input.RetrievedContexts {
		if strings.TrimSpace(item.URI) == "" {
			continue
		}
		parts = append(parts, MessagePart{
			Type:        "context",
			URI:         strings.TrimSpace(item.URI),
			ContextType: strings.TrimSpace(item.ContextType),
			Abstract:    strings.TrimSpace(item.Abstract),
		})
	}
	if len(parts) > 0 {
		if err := s.client.AddMessage(ctx, sessionID, AddMessageInput{
			Role:  "assistant",
			Parts: parts,
		}); err != nil {
			return err
		}
	}
	if s.mode.RecordsUsedContexts() {
		contexts := uniqueRetrievedContextURIs(input.RetrievedContexts)
		if len(contexts) > 0 {
			if err := s.client.Used(ctx, sessionID, UsedInput{Contexts: contexts}); err != nil {
				return err
			}
		}
	}
	_, err := s.client.Commit(ctx, sessionID, true)
	return err
}

// StableSessionID returns the deterministic OpenViking session identifier used
// for one Viking campaign turn stream.
func StableSessionID(campaignID, sessionID, participantID string) string {
	return fmt.Sprintf(
		"campaign:%s/session:%s/participant:%s",
		strings.TrimSpace(campaignID),
		strings.TrimSpace(sessionID),
		strings.TrimSpace(participantID),
	)
}

func uniqueRetrievedContextURIs(items []orchestration.RetrievedContext) []string {
	seen := map[string]struct{}{}
	uris := make([]string, 0, len(items))
	for _, item := range items {
		uri := strings.TrimSpace(item.URI)
		if uri == "" {
			continue
		}
		if _, ok := seen[uri]; ok {
			continue
		}
		seen[uri] = struct{}{}
		uris = append(uris, uri)
	}
	return uris
}
