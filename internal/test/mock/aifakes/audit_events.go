package aifakes

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/auditevent"
)

// AuditEventStore is an in-memory audit-event repository fake.
type AuditEventStore struct {
	AuditEvents     []auditevent.Event
	AuditEventNames []string
	PutErr          error
	ListErr         error
}

// NewAuditEventStore creates an initialized audit-event fake.
func NewAuditEventStore() *AuditEventStore {
	return &AuditEventStore{}
}

// PutAuditEvent appends an audit event record.
func (s *AuditEventStore) PutAuditEvent(_ context.Context, record auditevent.Event) error {
	if s.PutErr != nil {
		return s.PutErr
	}
	if strings.TrimSpace(record.ID) == "" {
		record.ID = fmt.Sprintf("%d", len(s.AuditEvents)+1)
	}
	s.AuditEvents = append(s.AuditEvents, record)
	s.AuditEventNames = append(s.AuditEventNames, string(record.EventName))
	return nil
}

// ListAuditEventsByOwner returns paginated audit events matching the filter.
func (s *AuditEventStore) ListAuditEventsByOwner(_ context.Context, ownerUserID string, pageSize int, pageToken string, filter auditevent.Filter) (auditevent.Page, error) {
	if s.ListErr != nil {
		return auditevent.Page{}, s.ListErr
	}
	if pageSize <= 0 {
		return auditevent.Page{}, errors.New("page size must be greater than zero")
	}
	eventName := strings.TrimSpace(string(filter.EventName))
	agentID := strings.TrimSpace(filter.AgentID)
	var (
		createdAfter  *time.Time
		createdBefore *time.Time
	)
	if filter.CreatedAfter != nil {
		timestamp := filter.CreatedAfter.UTC()
		createdAfter = &timestamp
	}
	if filter.CreatedBefore != nil {
		timestamp := filter.CreatedBefore.UTC()
		createdBefore = &timestamp
	}
	items := make([]auditevent.Event, 0, len(s.AuditEvents))
	for _, rec := range s.AuditEvents {
		if rec.OwnerUserID != ownerUserID {
			continue
		}
		if eventName != "" && string(rec.EventName) != eventName {
			continue
		}
		if agentID != "" && rec.AgentID != agentID {
			continue
		}
		if createdAfter != nil && rec.CreatedAt.Before(*createdAfter) {
			continue
		}
		if createdBefore != nil && rec.CreatedAt.After(*createdBefore) {
			continue
		}
		items = append(items, rec)
	}
	sort.Slice(items, func(i int, j int) bool {
		return compareAuditEventID(items[i].ID, items[j].ID) < 0
	})
	start := 0
	pageToken = strings.TrimSpace(pageToken)
	if pageToken != "" {
		start = len(items)
		for idx, rec := range items {
			if compareAuditEventID(rec.ID, pageToken) > 0 {
				start = idx
				break
			}
		}
	}
	if start >= len(items) {
		return auditevent.Page{AuditEvents: []auditevent.Event{}}, nil
	}

	end := start + pageSize
	nextPageToken := ""
	if end < len(items) {
		nextPageToken = items[end-1].ID
	} else {
		end = len(items)
	}
	return auditevent.Page{
		AuditEvents:   items[start:end],
		NextPageToken: nextPageToken,
	}, nil
}

func compareAuditEventID(left string, right string) int {
	leftID, leftErr := strconv.ParseInt(strings.TrimSpace(left), 10, 64)
	rightID, rightErr := strconv.ParseInt(strings.TrimSpace(right), 10, 64)
	if leftErr == nil && rightErr == nil {
		if leftID < rightID {
			return -1
		}
		if leftID > rightID {
			return 1
		}
		return 0
	}
	return strings.Compare(strings.TrimSpace(left), strings.TrimSpace(right))
}
