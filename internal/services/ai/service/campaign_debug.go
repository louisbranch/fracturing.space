package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/ai/debugtrace"
	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

// CampaignDebugTurn expands one persisted turn summary with ordered entries.
type CampaignDebugTurn struct {
	Turn    debugtrace.Turn
	Entries []debugtrace.Entry
}

// CampaignDebugTurnsPage returns one paginated slice of campaign debug turns.
type CampaignDebugTurnsPage struct {
	Turns         []debugtrace.Turn
	NextPageToken string
}

// CampaignDebugService handles read-only access to AI campaign debug traces.
type CampaignDebugService struct {
	debugTraceStore debugtrace.Store
	updateBroker    *CampaignDebugUpdateBroker
}

// CampaignDebugServiceConfig declares dependencies for debug-trace reads.
type CampaignDebugServiceConfig struct {
	DebugTraceStore debugtrace.Store
	UpdateBroker    *CampaignDebugUpdateBroker
}

// NewCampaignDebugService builds a campaign debug read service.
func NewCampaignDebugService(cfg CampaignDebugServiceConfig) (*CampaignDebugService, error) {
	if cfg.DebugTraceStore == nil {
		return nil, fmt.Errorf("ai: NewCampaignDebugService: debug trace store is required")
	}
	return &CampaignDebugService{
		debugTraceStore: cfg.DebugTraceStore,
		updateBroker:    cfg.UpdateBroker,
	}, nil
}

// ListCampaignDebugTurnsInput contains the filters for session-scoped turn history.
type ListCampaignDebugTurnsInput struct {
	CampaignID string
	SessionID  string
	PageSize   int
	PageToken  string
}

// ListCampaignDebugTurns returns newest-first turn summaries for one active session.
func (s *CampaignDebugService) ListCampaignDebugTurns(ctx context.Context, input ListCampaignDebugTurnsInput) (CampaignDebugTurnsPage, error) {
	if strings.TrimSpace(input.CampaignID) == "" {
		return CampaignDebugTurnsPage{}, Errorf(ErrKindInvalidArgument, "campaign_id is required")
	}
	if strings.TrimSpace(input.SessionID) == "" {
		return CampaignDebugTurnsPage{}, Errorf(ErrKindInvalidArgument, "session_id is required")
	}
	if input.PageSize <= 0 {
		return CampaignDebugTurnsPage{}, Errorf(ErrKindInvalidArgument, "page_size must be greater than zero")
	}
	page, err := s.debugTraceStore.ListCampaignDebugTurns(ctx, input.CampaignID, input.SessionID, input.PageSize, input.PageToken)
	if err != nil {
		return CampaignDebugTurnsPage{}, Wrapf(ErrKindInternal, err, "list campaign debug turns")
	}
	return CampaignDebugTurnsPage{
		Turns:         page.Turns,
		NextPageToken: page.NextPageToken,
	}, nil
}

// GetCampaignDebugTurnInput selects one persisted turn by campaign and turn id.
type GetCampaignDebugTurnInput struct {
	CampaignID string
	TurnID     string
}

// SubscribeCampaignDebugUpdatesInput selects one future-only session stream.
type SubscribeCampaignDebugUpdatesInput struct {
	CampaignID string
	SessionID  string
}

// GetCampaignDebugTurn returns one turn plus its ordered trace entries.
func (s *CampaignDebugService) GetCampaignDebugTurn(ctx context.Context, input GetCampaignDebugTurnInput) (CampaignDebugTurn, error) {
	if strings.TrimSpace(input.CampaignID) == "" {
		return CampaignDebugTurn{}, Errorf(ErrKindInvalidArgument, "campaign_id is required")
	}
	if strings.TrimSpace(input.TurnID) == "" {
		return CampaignDebugTurn{}, Errorf(ErrKindInvalidArgument, "turn_id is required")
	}

	turn, err := s.debugTraceStore.GetCampaignDebugTurn(ctx, input.CampaignID, input.TurnID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return CampaignDebugTurn{}, Errorf(ErrKindNotFound, "campaign debug turn not found")
		}
		return CampaignDebugTurn{}, Wrapf(ErrKindInternal, err, "get campaign debug turn")
	}
	entries, err := s.debugTraceStore.ListCampaignDebugTurnEntries(ctx, turn.ID)
	if err != nil {
		return CampaignDebugTurn{}, Wrapf(ErrKindInternal, err, "list campaign debug turn entries")
	}
	return CampaignDebugTurn{
		Turn:    turn,
		Entries: entries,
	}, nil
}

// SubscribeCampaignDebugUpdates registers one future-only session-scoped
// update stream for realtime consumers.
func (s *CampaignDebugService) SubscribeCampaignDebugUpdates(ctx context.Context, input SubscribeCampaignDebugUpdatesInput) (<-chan CampaignDebugTurnUpdate, func(), error) {
	if strings.TrimSpace(input.CampaignID) == "" {
		return nil, nil, Errorf(ErrKindInvalidArgument, "campaign_id is required")
	}
	if strings.TrimSpace(input.SessionID) == "" {
		return nil, nil, Errorf(ErrKindInvalidArgument, "session_id is required")
	}
	if s.updateBroker == nil {
		return nil, nil, Errorf(ErrKindFailedPrecondition, "campaign debug live updates are unavailable")
	}
	ch, unsubscribe := s.updateBroker.Subscribe(ctx, input.CampaignID, input.SessionID)
	if ch == nil {
		return nil, nil, Errorf(ErrKindFailedPrecondition, "campaign debug live updates are unavailable")
	}
	return ch, unsubscribe, nil
}
