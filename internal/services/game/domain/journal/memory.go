package journal

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

var (
	// ErrCampaignIDRequired indicates a missing campaign id.
	ErrCampaignIDRequired = errors.New("campaign id is required")
)

// Memory stores events in memory for tests and local use.
type Memory struct {
	mu       sync.Mutex
	registry *event.Registry
	streams  map[string][]event.Event
}

// NewMemory creates a new in-memory journal.
func NewMemory(registry *event.Registry) *Memory {
	return &Memory{
		registry: registry,
		streams:  make(map[string][]event.Event),
	}
}

// Append adds an event to the in-memory journal and assigns sequence and hashes.
func (m *Memory) Append(ctx context.Context, evt event.Event) (event.Event, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return event.Event{}, err
		}
	}
	if m == nil {
		return event.Event{}, errors.New("journal is required")
	}
	campaignID := strings.TrimSpace(evt.CampaignID)
	if campaignID == "" {
		return event.Event{}, ErrCampaignIDRequired
	}
	if m.registry != nil {
		validated, err := m.registry.ValidateForAppend(evt)
		if err != nil {
			return event.Event{}, err
		}
		evt = validated
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	stream := m.streams[campaignID]
	seq := uint64(len(stream) + 1)
	evt.Seq = seq
	hash, err := event.EventHash(evt)
	if err != nil {
		return event.Event{}, err
	}
	evt.Hash = hash
	prevHash := ""
	if len(stream) > 0 {
		prevHash = stream[len(stream)-1].ChainHash
	}
	evt.PrevHash = prevHash
	chainHash, err := event.ChainHash(evt, prevHash)
	if err != nil {
		return event.Event{}, err
	}
	evt.ChainHash = chainHash

	m.streams[campaignID] = append(stream, evt)
	return evt, nil
}

// ListEvents returns events ordered by sequence for a campaign.
func (m *Memory) ListEvents(ctx context.Context, campaignID string, afterSeq uint64, limit int) ([]event.Event, error) {
	if ctx != nil {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
	}
	if m == nil {
		return nil, errors.New("journal is required")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, ErrCampaignIDRequired
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	stream := m.streams[campaignID]
	if len(stream) == 0 {
		return nil, nil
	}
	start := 0
	if afterSeq > 0 {
		if afterSeq >= uint64(len(stream)) {
			return nil, nil
		}
		start = int(afterSeq)
	}
	end := len(stream)
	if limit > 0 && start+limit < end {
		end = start + limit
	}
	page := make([]event.Event, 0, end-start)
	for _, evt := range stream[start:end] {
		page = append(page, evt)
	}
	return page, nil
}
