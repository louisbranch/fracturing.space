package session

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestDecideSessionStart_EmitsSessionStartedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("session.start"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"session_id":"sess-1","session_name":"  Chapter One  "}`),
	}

	decision := Decide(State{}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.CampaignID != "camp-1" {
		t.Fatalf("event campaign id = %s, want %s", evt.CampaignID, "camp-1")
	}
	if evt.Type != event.Type("session.started") {
		t.Fatalf("event type = %s, want %s", evt.Type, "session.started")
	}
	if evt.SessionID != "sess-1" {
		t.Fatalf("event session id = %s, want %s", evt.SessionID, "sess-1")
	}
	if evt.EntityType != "session" {
		t.Fatalf("event entity type = %s, want %s", evt.EntityType, "session")
	}
	if evt.EntityID != "sess-1" {
		t.Fatalf("event entity id = %s, want %s", evt.EntityID, "sess-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}
	if evt.ActorType != event.ActorTypeSystem {
		t.Fatalf("event actor type = %s, want %s", evt.ActorType, event.ActorTypeSystem)
	}

	var payload StartPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.SessionID != "sess-1" {
		t.Fatalf("payload session id = %s, want %s", payload.SessionID, "sess-1")
	}
	if payload.SessionName != "Chapter One" {
		t.Fatalf("payload session name = %s, want %s", payload.SessionName, "Chapter One")
	}
}

func TestDecideSessionStart_MissingSessionIDRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("session.start"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"session_id":"  "}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeSessionIDRequired {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeSessionIDRequired)
	}
}

func TestDecideSessionStart_WhenAlreadyStarted_ReturnsRejection(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("session.start"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"session_id":"sess-1"}`),
	}

	decision := Decide(State{Started: true}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeSessionAlreadyStarted {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeSessionAlreadyStarted)
	}
}

func TestDecideSessionEnd_EmitsSessionEndedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("session.end"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"session_id":"sess-1"}`),
	}

	decision := Decide(State{Started: true}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("session.ended") {
		t.Fatalf("event type = %s, want %s", evt.Type, "session.ended")
	}
	if evt.SessionID != "sess-1" {
		t.Fatalf("event session id = %s, want %s", evt.SessionID, "sess-1")
	}
	if evt.EntityType != "session" {
		t.Fatalf("event entity type = %s, want %s", evt.EntityType, "session")
	}
	if evt.EntityID != "sess-1" {
		t.Fatalf("event entity id = %s, want %s", evt.EntityID, "sess-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload EndPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.SessionID != "sess-1" {
		t.Fatalf("payload session id = %s, want %s", payload.SessionID, "sess-1")
	}
}

func TestDecideSessionEnd_WhenNotStartedRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("session.end"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"session_id":"sess-1"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeSessionNotStarted {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeSessionNotStarted)
	}
}

func TestDecideSessionEnd_MissingSessionIDRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("session.end"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"session_id":"  "}`),
	}

	decision := Decide(State{Started: true}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeSessionIDRequired {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeSessionIDRequired)
	}
}

func TestDecideSessionGateOpen_EmitsGateOpenedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("session.gate_open"),
		ActorType:   command.ActorTypeSystem,
		SessionID:   "sess-1",
		PayloadJSON: []byte(`{"gate_id":"gate-1","gate_type":"  gm_consequence ","reason":"  danger "}`),
	}

	decision := Decide(State{}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("session.gate_opened") {
		t.Fatalf("event type = %s, want %s", evt.Type, "session.gate_opened")
	}
	if evt.SessionID != "sess-1" {
		t.Fatalf("event session id = %s, want %s", evt.SessionID, "sess-1")
	}
	if evt.EntityType != "session_gate" {
		t.Fatalf("event entity type = %s, want %s", evt.EntityType, "session_gate")
	}
	if evt.EntityID != "gate-1" {
		t.Fatalf("event entity id = %s, want %s", evt.EntityID, "gate-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload GateOpenedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.GateID != "gate-1" {
		t.Fatalf("payload gate id = %s, want %s", payload.GateID, "gate-1")
	}
	if payload.GateType != "gm_consequence" {
		t.Fatalf("payload gate type = %s, want %s", payload.GateType, "gm_consequence")
	}
	if payload.Reason != "danger" {
		t.Fatalf("payload reason = %s, want %s", payload.Reason, "danger")
	}
}

func TestDecideSessionGateResolve_EmitsGateResolvedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("session.gate_resolve"),
		ActorType:   command.ActorTypeSystem,
		SessionID:   "sess-1",
		PayloadJSON: []byte(`{"gate_id":"gate-1","decision":"  approve "}`),
	}

	decision := Decide(State{}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("session.gate_resolved") {
		t.Fatalf("event type = %s, want %s", evt.Type, "session.gate_resolved")
	}
	if evt.SessionID != "sess-1" {
		t.Fatalf("event session id = %s, want %s", evt.SessionID, "sess-1")
	}
	if evt.EntityType != "session_gate" {
		t.Fatalf("event entity type = %s, want %s", evt.EntityType, "session_gate")
	}
	if evt.EntityID != "gate-1" {
		t.Fatalf("event entity id = %s, want %s", evt.EntityID, "gate-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload GateResolvedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.GateID != "gate-1" {
		t.Fatalf("payload gate id = %s, want %s", payload.GateID, "gate-1")
	}
	if payload.Decision != "approve" {
		t.Fatalf("payload decision = %s, want %s", payload.Decision, "approve")
	}
}

func TestDecideSessionGateAbandon_EmitsGateAbandonedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("session.gate_abandon"),
		ActorType:   command.ActorTypeSystem,
		SessionID:   "sess-1",
		PayloadJSON: []byte(`{"gate_id":"gate-1","reason":"  timeout "}`),
	}

	decision := Decide(State{}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("session.gate_abandoned") {
		t.Fatalf("event type = %s, want %s", evt.Type, "session.gate_abandoned")
	}
	if evt.SessionID != "sess-1" {
		t.Fatalf("event session id = %s, want %s", evt.SessionID, "sess-1")
	}
	if evt.EntityType != "session_gate" {
		t.Fatalf("event entity type = %s, want %s", evt.EntityType, "session_gate")
	}
	if evt.EntityID != "gate-1" {
		t.Fatalf("event entity id = %s, want %s", evt.EntityID, "gate-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload GateAbandonedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.GateID != "gate-1" {
		t.Fatalf("payload gate id = %s, want %s", payload.GateID, "gate-1")
	}
	if payload.Reason != "timeout" {
		t.Fatalf("payload reason = %s, want %s", payload.Reason, "timeout")
	}
}

func TestDecideSessionGateOpen_MissingGateIDRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("session.gate_open"),
		ActorType:   command.ActorTypeSystem,
		SessionID:   "sess-1",
		PayloadJSON: []byte(`{"gate_id":" ","gate_type":"gm_consequence"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeSessionGateIDRequired {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeSessionGateIDRequired)
	}
}

func TestDecideSessionGateOpen_MissingGateTypeRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("session.gate_open"),
		ActorType:   command.ActorTypeSystem,
		SessionID:   "sess-1",
		PayloadJSON: []byte(`{"gate_id":"gate-1","gate_type":"  "}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeSessionGateTypeRequired {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeSessionGateTypeRequired)
	}
}

func TestDecideSessionGateResolve_MissingGateIDRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("session.gate_resolve"),
		ActorType:   command.ActorTypeSystem,
		SessionID:   "sess-1",
		PayloadJSON: []byte(`{"gate_id":" ","decision":"approve"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeSessionGateIDRequired {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeSessionGateIDRequired)
	}
}

func TestDecideSessionGateAbandon_MissingGateIDRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("session.gate_abandon"),
		ActorType:   command.ActorTypeSystem,
		SessionID:   "sess-1",
		PayloadJSON: []byte(`{"gate_id":" ","reason":"timeout"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeSessionGateIDRequired {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeSessionGateIDRequired)
	}
}

func TestDecideSessionSpotlightSet_EmitsSpotlightSetEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("session.spotlight_set"),
		ActorType:   command.ActorTypeSystem,
		SessionID:   "sess-1",
		PayloadJSON: []byte(`{"spotlight_type":"  character ","character_id":"char-1"}`),
	}

	decision := Decide(State{}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("session.spotlight_set") {
		t.Fatalf("event type = %s, want %s", evt.Type, "session.spotlight_set")
	}
	if evt.SessionID != "sess-1" {
		t.Fatalf("event session id = %s, want %s", evt.SessionID, "sess-1")
	}
	if evt.EntityType != "session" {
		t.Fatalf("event entity type = %s, want %s", evt.EntityType, "session")
	}
	if evt.EntityID != "sess-1" {
		t.Fatalf("event entity id = %s, want %s", evt.EntityID, "sess-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload SpotlightSetPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.SpotlightType != "character" {
		t.Fatalf("spotlight type = %s, want %s", payload.SpotlightType, "character")
	}
	if payload.CharacterID != "char-1" {
		t.Fatalf("character id = %s, want %s", payload.CharacterID, "char-1")
	}
}

func TestDecideSessionSpotlightSet_MissingTypeRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("session.spotlight_set"),
		ActorType:   command.ActorTypeSystem,
		SessionID:   "sess-1",
		PayloadJSON: []byte(`{"spotlight_type":"  "}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeSessionSpotlightTypeRequired {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeSessionSpotlightTypeRequired)
	}
}

func TestDecideSessionSpotlightClear_EmitsSpotlightClearedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("session.spotlight_clear"),
		ActorType:   command.ActorTypeSystem,
		SessionID:   "sess-1",
		PayloadJSON: []byte(`{"reason":"scene change"}`),
	}

	decision := Decide(State{}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("session.spotlight_cleared") {
		t.Fatalf("event type = %s, want %s", evt.Type, "session.spotlight_cleared")
	}
	if evt.SessionID != "sess-1" {
		t.Fatalf("event session id = %s, want %s", evt.SessionID, "sess-1")
	}
	if evt.EntityType != "session" {
		t.Fatalf("event entity type = %s, want %s", evt.EntityType, "session")
	}
	if evt.EntityID != "sess-1" {
		t.Fatalf("event entity id = %s, want %s", evt.EntityID, "sess-1")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload SpotlightClearedPayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Reason != "scene change" {
		t.Fatalf("reason = %s, want %s", payload.Reason, "scene change")
	}
}
