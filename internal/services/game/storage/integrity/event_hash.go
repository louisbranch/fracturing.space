package integrity

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/core/encoding"
)

// EventHash computes the content hash for an event.
func EventHash(evt event.Event) (string, error) {
	envelope := map[string]any{
		"campaign_id": evt.CampaignID,
		"event_type":  string(evt.Type),
		"timestamp":   evt.Timestamp.Format(time.RFC3339Nano),
		"actor_type":  string(evt.ActorType),
		"payload":     json.RawMessage(evt.PayloadJSON),
	}
	if evt.SessionID != "" {
		envelope["session_id"] = evt.SessionID
	}
	if evt.RequestID != "" {
		envelope["request_id"] = evt.RequestID
	}
	if evt.InvocationID != "" {
		envelope["invocation_id"] = evt.InvocationID
	}
	if evt.ActorID != "" {
		envelope["actor_id"] = evt.ActorID
	}
	if evt.EntityType != "" {
		envelope["entity_type"] = evt.EntityType
	}
	if evt.EntityID != "" {
		envelope["entity_id"] = evt.EntityID
	}
	return encoding.ContentHash(envelope)
}

// ChainHash computes the SHA-256 hash that links an event to the previous hash.
func ChainHash(evt event.Event, prevHash string) (string, error) {
	if evt.Hash == "" {
		return "", fmt.Errorf("event hash is required")
	}
	envelope := map[string]any{
		"campaign_id":     evt.CampaignID,
		"seq":             evt.Seq,
		"event_hash":      evt.Hash,
		"timestamp":       evt.Timestamp.Format(time.RFC3339Nano),
		"event_type":      string(evt.Type),
		"actor_type":      string(evt.ActorType),
		"payload":         json.RawMessage(evt.PayloadJSON),
		"prev_event_hash": prevHash,
	}
	if evt.SessionID != "" {
		envelope["session_id"] = evt.SessionID
	}
	if evt.RequestID != "" {
		envelope["request_id"] = evt.RequestID
	}
	if evt.InvocationID != "" {
		envelope["invocation_id"] = evt.InvocationID
	}
	if evt.ActorID != "" {
		envelope["actor_id"] = evt.ActorID
	}
	if evt.EntityType != "" {
		envelope["entity_type"] = evt.EntityType
	}
	if evt.EntityID != "" {
		envelope["entity_id"] = evt.EntityID
	}

	canonical, err := encoding.CanonicalJSON(envelope)
	if err != nil {
		return "", fmt.Errorf("canonical json: %w", err)
	}
	hash := sha256.Sum256(canonical)
	return hex.EncodeToString(hash[:]), nil
}
