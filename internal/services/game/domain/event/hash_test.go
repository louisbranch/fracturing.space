package event

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func TestEventHashDeterministic(t *testing.T) {
	ts := time.Date(2024, 2, 1, 10, 30, 0, 0, time.UTC)
	evt := Event{
		CampaignID:  "c1",
		Timestamp:   ts,
		Type:        Type("campaign.created"),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"demo"}`),
	}

	first, err := EventHash(evt)
	if err != nil {
		t.Fatalf("event hash: %v", err)
	}

	second, err := EventHash(evt)
	if err != nil {
		t.Fatalf("event hash: %v", err)
	}

	if first != second {
		t.Fatalf("expected deterministic hash, got %s and %s", first, second)
	}
}

func TestEventHashChangesWithOptionalFields(t *testing.T) {
	ts := time.Date(2024, 2, 1, 10, 30, 0, 0, time.UTC)
	base := Event{
		CampaignID:  "c1",
		Timestamp:   ts,
		Type:        Type("campaign.created"),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"demo"}`),
	}

	baseline, err := EventHash(base)
	if err != nil {
		t.Fatalf("event hash: %v", err)
	}

	withCorrelation := base
	withCorrelation.CorrelationID = "corr-1"
	hashCorrelation, err := EventHash(withCorrelation)
	if err != nil {
		t.Fatalf("event hash: %v", err)
	}

	if baseline == hashCorrelation {
		t.Fatal("expected hash to change when optional fields change")
	}
}

func TestChainHashRequiresEventHash(t *testing.T) {
	evt := Event{
		CampaignID:  "c1",
		Seq:         10,
		Timestamp:   time.Date(2024, 2, 1, 10, 30, 0, 0, time.UTC),
		Type:        Type("campaign.created"),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"demo"}`),
	}

	_, err := ChainHash(evt, "prev")
	if err == nil {
		t.Fatal("expected error when event hash is missing")
	}
}

func TestChainHashDeterministic(t *testing.T) {
	evt := Event{
		CampaignID:  "c1",
		Seq:         10,
		Hash:        "eventhash",
		Timestamp:   time.Date(2024, 2, 1, 10, 30, 0, 0, time.UTC),
		Type:        Type("campaign.created"),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"demo"}`),
	}

	first, err := ChainHash(evt, "prev")
	if err != nil {
		t.Fatalf("chain hash: %v", err)
	}
	second, err := ChainHash(evt, "prev")
	if err != nil {
		t.Fatalf("chain hash: %v", err)
	}
	if first != second {
		t.Fatalf("expected deterministic chain hash, got %s and %s", first, second)
	}
}

func TestContentEnvelope_OptionalFieldsBranchCoverage(t *testing.T) {
	ts := time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC)
	withAll := Event{
		CampaignID:    "camp-1",
		Timestamp:     ts,
		Type:          Type("campaign.created"),
		ActorType:     ActorTypeSystem,
		PayloadJSON:   []byte(`{"ok":true}`),
		SessionID:     ids.SessionID("sess-1"),
		SceneID:       ids.SceneID("scene-1"),
		RequestID:     "req-1",
		InvocationID:  "inv-1",
		ActorID:       "actor-1",
		EntityType:    "campaign",
		EntityID:      "camp-1",
		SystemID:      "daggerheart",
		SystemVersion: "v1",
		CorrelationID: "corr-1",
		CausationID:   "cause-1",
	}
	envelope := contentEnvelope(withAll)
	for key := range map[string]string{
		"session_id":     withAll.SessionID.String(),
		"scene_id":       withAll.SceneID.String(),
		"request_id":     withAll.RequestID,
		"invocation_id":  withAll.InvocationID,
		"actor_id":       withAll.ActorID,
		"entity_type":    withAll.EntityType,
		"entity_id":      withAll.EntityID,
		"system_id":      withAll.SystemID,
		"system_version": withAll.SystemVersion,
		"correlation_id": withAll.CorrelationID,
		"causation_id":   withAll.CausationID,
	} {
		if _, ok := envelope[key]; !ok {
			t.Fatalf("contentEnvelope() missing optional key %q", key)
		}
	}

	withNone := Event{
		CampaignID:  "camp-1",
		Timestamp:   ts,
		Type:        Type("campaign.created"),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte(`{"ok":true}`),
	}
	envelope = contentEnvelope(withNone)
	for _, key := range []string{
		"session_id",
		"scene_id",
		"request_id",
		"invocation_id",
		"actor_id",
		"entity_type",
		"entity_id",
		"system_id",
		"system_version",
		"correlation_id",
		"causation_id",
	} {
		if _, ok := envelope[key]; ok {
			t.Fatalf("contentEnvelope() included empty optional key %q", key)
		}
	}
}

func TestChainEnvelope_IncludesPrevHashAndOptionalFields(t *testing.T) {
	ts := time.Date(2026, 3, 1, 12, 5, 0, 0, time.UTC)
	evt := Event{
		CampaignID:   "camp-1",
		Seq:          12,
		Hash:         "event-hash-1",
		Timestamp:    ts,
		Type:         Type("campaign.created"),
		ActorType:    ActorTypeSystem,
		PayloadJSON:  []byte(`{"ok":true}`),
		SessionID:    ids.SessionID("sess-1"),
		RequestID:    "req-1",
		InvocationID: "inv-1",
		ActorID:      "actor-1",
	}
	envelope := chainEnvelope(evt, "prev-hash-1")
	if envelope["prev_event_hash"] != "prev-hash-1" {
		t.Fatalf("prev_event_hash = %v, want prev-hash-1", envelope["prev_event_hash"])
	}
	if envelope["seq"] != uint64(12) {
		t.Fatalf("seq = %v, want 12", envelope["seq"])
	}
	for _, key := range []string{"session_id", "request_id", "invocation_id", "actor_id"} {
		if _, ok := envelope[key]; !ok {
			t.Fatalf("chainEnvelope() missing optional key %q", key)
		}
	}
}

func TestChainHash_ReturnsCanonicalJSONError(t *testing.T) {
	restore := canonicalJSON
	canonicalJSON = func(any) ([]byte, error) {
		return nil, errors.New("forced chain canonical failure")
	}
	t.Cleanup(func() { canonicalJSON = restore })

	evt := Event{
		CampaignID:  "c1",
		Seq:         10,
		Hash:        "eventhash",
		Timestamp:   time.Date(2024, 2, 1, 10, 30, 0, 0, time.UTC),
		Type:        Type("campaign.created"),
		ActorType:   ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"demo"}`),
	}

	_, err := ChainHash(evt, "prev")
	if err == nil {
		t.Fatal("expected canonical json error")
	}
	if !strings.Contains(err.Error(), "canonical json") {
		t.Fatalf("ChainHash() error = %v, want canonical json context", err)
	}
}
