package event

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	coreencoding "github.com/louisbranch/fracturing.space/internal/services/game/core/encoding"
)

// contentEnvelope builds the canonical field map used for content hashing.
// This is the single source of truth for which fields participate in the hash
// and how they are serialized, preventing drift between call sites.
func contentEnvelope(evt Event) map[string]any {
	envelope := map[string]any{
		"campaign_id": evt.CampaignID,
		"event_type":  string(evt.Type),
		"timestamp":   evt.Timestamp.Format(time.RFC3339Nano),
		"actor_type":  string(evt.ActorType),
		"payload":     json.RawMessage(evt.PayloadJSON),
	}
	addOptionalFields(envelope, evt)
	return envelope
}

// chainEnvelope builds the canonical field map used for chain integrity hashing.
// Extends the content envelope with chain-linking fields (seq, event hash,
// previous event hash).
func chainEnvelope(evt Event, prevHash string) map[string]any {
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
	addOptionalFields(envelope, evt)
	return envelope
}

// addOptionalFields adds non-empty optional envelope fields to the map.
// Centralizes the conditional-inclusion logic so content and chain envelopes
// stay in sync when new fields are added.
func addOptionalFields(envelope map[string]any, evt Event) {
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
	if evt.SystemID != "" {
		envelope["system_id"] = evt.SystemID
	}
	if evt.SystemVersion != "" {
		envelope["system_version"] = evt.SystemVersion
	}
	if evt.CorrelationID != "" {
		envelope["correlation_id"] = evt.CorrelationID
	}
	if evt.CausationID != "" {
		envelope["causation_id"] = evt.CausationID
	}
}

// EventHash computes the content hash for an event.
func EventHash(evt Event) (string, error) {
	return coreencoding.ContentHash(contentEnvelope(evt))
}

// ChainHash computes the SHA-256 hash that links an event to the previous hash.
func ChainHash(evt Event, prevHash string) (string, error) {
	if evt.Hash == "" {
		return "", fmt.Errorf("event hash is required")
	}
	canonical, err := coreencoding.CanonicalJSON(chainEnvelope(evt, prevHash))
	if err != nil {
		return "", fmt.Errorf("canonical json: %w", err)
	}
	hash := sha256.Sum256(canonical)
	return hex.EncodeToString(hash[:]), nil
}
