package command

import (
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestNewEvent_CopiesCommandEnvelope(t *testing.T) {
	cmd := Command{
		CampaignID:    "camp-1",
		ActorType:     ActorTypeGM,
		ActorID:       "actor-1",
		SessionID:     "sess-1",
		RequestID:     "req-1",
		InvocationID:  "inv-1",
		SystemID:      "sys-1",
		SystemVersion: "v1",
		CorrelationID: "corr-1",
		CausationID:   "cause-1",
	}
	now := time.Date(2026, 2, 20, 12, 0, 0, 0, time.UTC)

	evt := NewEvent(cmd, event.Type("campaign.created"), "campaign", "camp-1", []byte(`{"name":"test"}`), now)

	if evt.CampaignID != "camp-1" {
		t.Errorf("CampaignID = %q, want camp-1", evt.CampaignID)
	}
	if evt.Type != event.Type("campaign.created") {
		t.Errorf("Type = %q, want campaign.created", evt.Type)
	}
	if evt.ActorType != event.ActorType(cmd.ActorType) {
		t.Errorf("ActorType = %q, want %q", evt.ActorType, cmd.ActorType)
	}
	if evt.ActorID != "actor-1" {
		t.Errorf("ActorID = %q, want actor-1", evt.ActorID)
	}
	if evt.SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want sess-1", evt.SessionID)
	}
	if evt.RequestID != "req-1" {
		t.Errorf("RequestID = %q, want req-1", evt.RequestID)
	}
	if evt.InvocationID != "inv-1" {
		t.Errorf("InvocationID = %q, want inv-1", evt.InvocationID)
	}
	if evt.SystemID != "sys-1" {
		t.Errorf("SystemID = %q, want sys-1", evt.SystemID)
	}
	if evt.SystemVersion != "v1" {
		t.Errorf("SystemVersion = %q, want v1", evt.SystemVersion)
	}
	if evt.CorrelationID != "corr-1" {
		t.Errorf("CorrelationID = %q, want corr-1", evt.CorrelationID)
	}
	if evt.CausationID != "cause-1" {
		t.Errorf("CausationID = %q, want cause-1", evt.CausationID)
	}
	if evt.EntityType != "campaign" {
		t.Errorf("EntityType = %q, want campaign", evt.EntityType)
	}
	if evt.EntityID != "camp-1" {
		t.Errorf("EntityID = %q, want camp-1", evt.EntityID)
	}
	if !evt.Timestamp.Equal(now) {
		t.Errorf("Timestamp = %v, want %v", evt.Timestamp, now)
	}
	if string(evt.PayloadJSON) != `{"name":"test"}` {
		t.Errorf("PayloadJSON = %s, want %s", evt.PayloadJSON, `{"name":"test"}`)
	}
}
