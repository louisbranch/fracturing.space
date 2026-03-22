package service

import (
	"context"
	"strings"
	"sync"

	"github.com/louisbranch/fracturing.space/internal/services/ai/debugtrace"
)

const campaignDebugUpdateBuffer = 32

// CampaignDebugTurnUpdate carries one persisted turn summary plus any newly
// appended entries for realtime consumers.
type CampaignDebugTurnUpdate struct {
	Turn            debugtrace.Turn
	AppendedEntries []debugtrace.Entry
}

// CampaignDebugUpdateBroker provides in-process best-effort fanout for live AI
// debug updates. Durable trace storage remains the recovery source of truth.
type CampaignDebugUpdateBroker struct {
	mu          sync.Mutex
	nextID      uint64
	subscribers map[string]map[uint64]chan CampaignDebugTurnUpdate
}

// NewCampaignDebugUpdateBroker constructs an empty live-update broker.
func NewCampaignDebugUpdateBroker() *CampaignDebugUpdateBroker {
	return &CampaignDebugUpdateBroker{
		subscribers: make(map[string]map[uint64]chan CampaignDebugTurnUpdate),
	}
}

// Subscribe registers one future-only session-scoped subscriber.
func (b *CampaignDebugUpdateBroker) Subscribe(ctx context.Context, campaignID string, sessionID string) (<-chan CampaignDebugTurnUpdate, func()) {
	if b == nil {
		return nil, func() {}
	}
	key := campaignDebugSubscriptionKey(campaignID, sessionID)
	if key == "" {
		return nil, func() {}
	}

	ch := make(chan CampaignDebugTurnUpdate, campaignDebugUpdateBuffer)

	b.mu.Lock()
	b.nextID++
	id := b.nextID
	if b.subscribers[key] == nil {
		b.subscribers[key] = make(map[uint64]chan CampaignDebugTurnUpdate)
	}
	b.subscribers[key][id] = ch
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()
		subscribers := b.subscribers[key]
		if subscribers == nil {
			return
		}
		if existing, ok := subscribers[id]; ok {
			delete(subscribers, id)
			close(existing)
		}
		if len(subscribers) == 0 {
			delete(b.subscribers, key)
		}
	}

	go func() {
		<-ctx.Done()
		unsubscribe()
	}()

	return ch, unsubscribe
}

// Publish fans out one persisted update without blocking the caller.
func (b *CampaignDebugUpdateBroker) Publish(campaignID string, sessionID string, update CampaignDebugTurnUpdate) {
	if b == nil {
		return
	}
	key := campaignDebugSubscriptionKey(campaignID, sessionID)
	if key == "" {
		return
	}

	b.mu.Lock()
	subscribers := b.subscribers[key]
	if len(subscribers) == 0 {
		b.mu.Unlock()
		return
	}
	channels := make([]chan CampaignDebugTurnUpdate, 0, len(subscribers))
	for _, ch := range subscribers {
		channels = append(channels, ch)
	}
	b.mu.Unlock()

	for _, ch := range channels {
		cloned := cloneCampaignDebugTurnUpdate(update)
		select {
		case ch <- cloned:
		default:
			select {
			case <-ch:
			default:
			}
			select {
			case ch <- cloned:
			default:
			}
		}
	}
}

func campaignDebugSubscriptionKey(campaignID string, sessionID string) string {
	campaignID = strings.TrimSpace(campaignID)
	sessionID = strings.TrimSpace(sessionID)
	if campaignID == "" || sessionID == "" {
		return ""
	}
	return campaignID + "\x00" + sessionID
}

func cloneCampaignDebugTurnUpdate(update CampaignDebugTurnUpdate) CampaignDebugTurnUpdate {
	cloned := CampaignDebugTurnUpdate{Turn: update.Turn}
	if len(update.AppendedEntries) == 0 {
		return cloned
	}
	cloned.AppendedEntries = append([]debugtrace.Entry(nil), update.AppendedEntries...)
	return cloned
}
