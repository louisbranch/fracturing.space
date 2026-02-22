package campaign

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

func TestDecideCampaignCreate_EmitsCampaignCreatedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"game_system":"daggerheart","gm_mode":"human","name":"Sunfall"}`),
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
	if evt.Type != event.Type("campaign.created") {
		t.Fatalf("event type = %s, want %s", evt.Type, "campaign.created")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}
	if evt.ActorType != event.ActorTypeSystem {
		t.Fatalf("event actor type = %s, want %s", evt.ActorType, event.ActorTypeSystem)
	}

	var payload CreatePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Name != "Sunfall" {
		t.Fatalf("payload name = %s, want %s", payload.Name, "Sunfall")
	}
	if payload.GameSystem != "daggerheart" {
		t.Fatalf("payload game system = %s, want %s", payload.GameSystem, "daggerheart")
	}
	if payload.GmMode != "human" {
		t.Fatalf("payload gm mode = %s, want %s", payload.GmMode, "human")
	}
}

func TestDecideCampaignCreate_NormalizesPayloadValues(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"  Sunfall  ","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"GM_MODE_HUMAN"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	var payload CreatePayload
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Name != "Sunfall" {
		t.Fatalf("payload name = %s, want %s", payload.Name, "Sunfall")
	}
	if payload.GameSystem != "daggerheart" {
		t.Fatalf("payload game system = %s, want %s", payload.GameSystem, "daggerheart")
	}
	if payload.GmMode != "human" {
		t.Fatalf("payload gm mode = %s, want %s", payload.GmMode, "human")
	}
}

func TestDecideCampaignCreate_DefaultCoverAssetAssigned(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"Sunfall","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"GM_MODE_HUMAN"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	var payload CreatePayload
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.CoverAssetID == "" {
		t.Fatal("expected cover_asset_id to be assigned")
	}
}

func TestDecideCampaignCreate_DefaultCoverAssetDeterministic(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-deterministic",
		Type:        command.Type("campaign.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"Sunfall","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"GM_MODE_HUMAN"}`),
	}

	first := Decide(State{}, cmd, nil)
	second := Decide(State{}, cmd, nil)
	if len(first.Events) != 1 || len(second.Events) != 1 {
		t.Fatal("expected one event per decision")
	}

	var firstPayload CreatePayload
	if err := json.Unmarshal(first.Events[0].PayloadJSON, &firstPayload); err != nil {
		t.Fatalf("decode first payload: %v", err)
	}
	var secondPayload CreatePayload
	if err := json.Unmarshal(second.Events[0].PayloadJSON, &secondPayload); err != nil {
		t.Fatalf("decode second payload: %v", err)
	}
	if firstPayload.CoverAssetID == "" {
		t.Fatal("expected non-empty default cover asset id")
	}
	if firstPayload.CoverAssetID != secondPayload.CoverAssetID {
		t.Fatalf("cover asset ids differ: %q vs %q", firstPayload.CoverAssetID, secondPayload.CoverAssetID)
	}
}

func TestDecideCampaignCreate_InvalidCoverAssetRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"Sunfall","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"GM_MODE_HUMAN","cover_asset_id":"unknown-cover"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCampaignCoverAssetInvalid {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCampaignCoverAssetInvalid)
	}
}

func TestDecideCampaignCreate_IncludesMetadata(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"Sunfall","game_system":"GAME_SYSTEM_DAGGERHEART","gm_mode":"GM_MODE_HUMAN","locale":"en-US","intent":"STANDARD","access_policy":"PUBLIC","theme_prompt":"A dark fantasy adventure"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	var payload map[string]any
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if got, ok := payload["locale"].(string); !ok || got != "en-US" {
		t.Fatalf("payload locale = %v, want %s", payload["locale"], "en-US")
	}
	if got, ok := payload["intent"].(string); !ok || got != "STANDARD" {
		t.Fatalf("payload intent = %v, want %s", payload["intent"], "STANDARD")
	}
	if got, ok := payload["access_policy"].(string); !ok || got != "PUBLIC" {
		t.Fatalf("payload access_policy = %v, want %s", payload["access_policy"], "PUBLIC")
	}
	if got, ok := payload["theme_prompt"].(string); !ok || got != "A dark fantasy adventure" {
		t.Fatalf("payload theme_prompt = %v, want %s", payload["theme_prompt"], "A dark fantasy adventure")
	}
}

func TestDecideCampaignCreate_WhenAlreadyCreated_ReturnsRejection(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"game_system":"daggerheart","gm_mode":"human","name":"Sunfall"}`),
	}

	decision := Decide(State{Created: true}, cmd, func() time.Time { return now })
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCampaignAlreadyExists {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCampaignAlreadyExists)
	}
}

func TestDecideCampaignCreate_MissingName_ReturnsRejection(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"game_system":"daggerheart","gm_mode":"human"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCampaignNameEmpty {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCampaignNameEmpty)
	}
}

func TestDecideCampaignCreate_WhitespaceName_ReturnsRejection(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"   ","game_system":"daggerheart","gm_mode":"human"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCampaignNameEmpty {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCampaignNameEmpty)
	}
}

func TestDecideCampaignCreate_MissingGameSystem_ReturnsRejection(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"Sunfall","gm_mode":"human"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCampaignGameSystemInvalid {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCampaignGameSystemInvalid)
	}
}

func TestDecideCampaignCreate_InvalidGameSystem_ReturnsRejection(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"Sunfall","game_system":"unknown","gm_mode":"human"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCampaignGameSystemInvalid {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCampaignGameSystemInvalid)
	}
}

func TestDecideCampaignCreate_MissingGmMode_ReturnsRejection(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"Sunfall","game_system":"daggerheart"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCampaignGmModeInvalid {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCampaignGmModeInvalid)
	}
}

func TestDecideCampaignCreate_InvalidGmMode_ReturnsRejection(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.create"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"name":"Sunfall","game_system":"daggerheart","gm_mode":"unknown"}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCampaignGmModeInvalid {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCampaignGmModeInvalid)
	}
}

func TestDecideCampaignUpdate_EmitsCampaignUpdatedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"fields":{"name":"  Sunfall  ","status":"CAMPAIGN_STATUS_ACTIVE","theme_prompt":"  new theme  "}}`),
	}

	decision := Decide(State{Created: true, Status: StatusDraft}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	evt := decision.Events[0]
	if evt.Type != event.Type("campaign.updated") {
		t.Fatalf("event type = %s, want %s", evt.Type, "campaign.updated")
	}
	if !evt.Timestamp.Equal(now) {
		t.Fatalf("event timestamp = %s, want %s", evt.Timestamp, now)
	}

	var payload updatePayload
	if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Fields["name"] != "Sunfall" {
		t.Fatalf("payload name = %s, want %s", payload.Fields["name"], "Sunfall")
	}
	if payload.Fields["status"] != "active" {
		t.Fatalf("payload status = %s, want %s", payload.Fields["status"], "active")
	}
	if payload.Fields["theme_prompt"] != "new theme" {
		t.Fatalf("payload theme_prompt = %s, want %s", payload.Fields["theme_prompt"], "new theme")
	}
}

func TestDecideCampaignUpdate_UpdatesCoverAssetID(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"fields":{"cover_asset_id":"  abandoned_castle_courtyard  "}}`),
	}

	decision := Decide(State{Created: true, Status: StatusDraft}, cmd, nil)
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	var payload updatePayload
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Fields["cover_asset_id"] != "abandoned_castle_courtyard" {
		t.Fatalf("payload cover_asset_id = %s, want %s", payload.Fields["cover_asset_id"], "abandoned_castle_courtyard")
	}
}

func TestDecideCampaignUpdate_InvalidCoverAssetIDRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"fields":{"cover_asset_id":"unknown-cover"}}`),
	}

	decision := Decide(State{Created: true, Status: StatusDraft}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCampaignCoverAssetInvalid {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCampaignCoverAssetInvalid)
	}
}

func TestDecideCampaignUpdate_InvalidStatusRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"fields":{"status":"UNKNOWN"}}`),
	}

	decision := Decide(State{Created: true, Status: StatusDraft}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "CAMPAIGN_INVALID_STATUS" {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, "CAMPAIGN_INVALID_STATUS")
	}
}

func TestDecideCampaignUpdate_InvalidStatusTransitionRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"fields":{"status":"ARCHIVED"}}`),
	}

	decision := Decide(State{Created: true, Status: StatusDraft}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "CAMPAIGN_INVALID_STATUS_TRANSITION" {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, "CAMPAIGN_INVALID_STATUS_TRANSITION")
	}
}

func TestDecideCampaignUpdate_EmptyFieldsRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.update"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"fields":{}}`),
	}

	decision := Decide(State{Created: true, Status: StatusDraft}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "CAMPAIGN_UPDATE_EMPTY" {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, "CAMPAIGN_UPDATE_EMPTY")
	}
}

func TestDecideCampaignEnd_EmitsCompletedStatus(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.end"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{}`),
	}

	decision := Decide(State{Created: true, Status: StatusActive}, cmd, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	var payload updatePayload
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Fields["status"] != "completed" {
		t.Fatalf("payload status = %s, want %s", payload.Fields["status"], "completed")
	}
}

func TestDecideCampaignArchive_EmitsArchivedStatus(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.archive"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{}`),
	}

	decision := Decide(State{Created: true, Status: StatusActive}, cmd, nil)
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	var payload updatePayload
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Fields["status"] != "archived" {
		t.Fatalf("payload status = %s, want %s", payload.Fields["status"], "archived")
	}
}

func TestDecideCampaignRestore_EmitsDraftStatus(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.restore"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{}`),
	}

	decision := Decide(State{Created: true, Status: StatusArchived}, cmd, nil)
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}

	var payload updatePayload
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.Fields["status"] != "draft" {
		t.Fatalf("payload status = %s, want %s", payload.Fields["status"], "draft")
	}
}

func TestDecideCampaignStatusCommand_NotCreatedRejected(t *testing.T) {
	cmd := command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.end"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{}`),
	}

	decision := Decide(State{}, cmd, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCampaignNotCreated {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCampaignNotCreated)
	}
}

func TestDecideCampaignStatusCommand_InvalidTransitionRejected(t *testing.T) {
	tests := []struct {
		name   string
		status Status
		cmd    command.Type
	}{
		{name: "end from draft", status: StatusDraft, cmd: command.Type("campaign.end")},
		{name: "archive from draft", status: StatusDraft, cmd: command.Type("campaign.archive")},
		{name: "restore from active", status: StatusActive, cmd: command.Type("campaign.restore")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := Decide(State{Created: true, Status: tt.status}, command.Command{
				CampaignID:  "camp-1",
				Type:        tt.cmd,
				ActorType:   command.ActorTypeSystem,
				PayloadJSON: []byte(`{}`),
			}, nil)
			if len(decision.Events) != 0 {
				t.Fatalf("expected no events, got %d", len(decision.Events))
			}
			if len(decision.Rejections) != 1 {
				t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
			}
			if decision.Rejections[0].Code != rejectionCodeCampaignStatusTransition {
				t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCampaignStatusTransition)
			}
		})
	}
}

func TestDecideCampaignFork_EmitsForkedEvent(t *testing.T) {
	now := time.Date(2026, 2, 14, 0, 0, 0, 0, time.UTC)
	decision := Decide(State{Created: true}, command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.fork"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"parent_campaign_id":"camp-0","fork_event_seq":3,"origin_campaign_id":"camp-root","copy_participants":true}`),
	}, func() time.Time { return now })
	if len(decision.Rejections) != 0 {
		t.Fatalf("expected no rejections, got %d", len(decision.Rejections))
	}
	if len(decision.Events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(decision.Events))
	}
	if decision.Events[0].Type != event.Type("campaign.forked") {
		t.Fatalf("event type = %s, want %s", decision.Events[0].Type, "campaign.forked")
	}
	if decision.Events[0].EntityType != "campaign" {
		t.Fatalf("entity type = %s, want %s", decision.Events[0].EntityType, "campaign")
	}
	if decision.Events[0].EntityID != "camp-1" {
		t.Fatalf("entity id = %s, want %s", decision.Events[0].EntityID, "camp-1")
	}
	if !decision.Events[0].Timestamp.Equal(now) {
		t.Fatalf("timestamp = %s, want %s", decision.Events[0].Timestamp, now)
	}
	var payload struct {
		ParentCampaignID string `json:"parent_campaign_id"`
		ForkEventSeq     uint64 `json:"fork_event_seq"`
		OriginCampaignID string `json:"origin_campaign_id"`
		CopyParticipants bool   `json:"copy_participants"`
	}
	if err := json.Unmarshal(decision.Events[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if payload.ParentCampaignID != "camp-0" {
		t.Fatalf("parent campaign id = %s, want %s", payload.ParentCampaignID, "camp-0")
	}
	if payload.ForkEventSeq != 3 {
		t.Fatalf("fork event seq = %d, want %d", payload.ForkEventSeq, 3)
	}
	if payload.OriginCampaignID != "camp-root" {
		t.Fatalf("origin campaign id = %s, want %s", payload.OriginCampaignID, "camp-root")
	}
	if !payload.CopyParticipants {
		t.Fatal("expected copy participants to be true")
	}
}

func TestDecideCampaignFork_NotCreatedRejected(t *testing.T) {
	decision := Decide(State{}, command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.fork"),
		ActorType:   command.ActorTypeSystem,
		PayloadJSON: []byte(`{"parent_campaign_id":"camp-0","fork_event_seq":1,"origin_campaign_id":"camp-root"}`),
	}, nil)
	if len(decision.Events) != 0 {
		t.Fatalf("expected no events, got %d", len(decision.Events))
	}
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != rejectionCodeCampaignNotCreated {
		t.Fatalf("rejection code = %s, want %s", decision.Rejections[0].Code, rejectionCodeCampaignNotCreated)
	}
}

func TestDecide_UnrecognizedCommandTypeRejected(t *testing.T) {
	decision := Decide(State{}, command.Command{
		CampaignID: "camp-1",
		Type:       command.Type("campaign.nonexistent"),
	}, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "COMMAND_TYPE_UNSUPPORTED" {
		t.Fatalf("rejection code = %s, want COMMAND_TYPE_UNSUPPORTED", decision.Rejections[0].Code)
	}
}

func TestDecide_MalformedCreatePayloadRejected(t *testing.T) {
	decision := Decide(State{}, command.Command{
		CampaignID:  "camp-1",
		Type:        command.Type("campaign.create"),
		PayloadJSON: []byte(`{corrupt`),
	}, nil)
	if len(decision.Rejections) != 1 {
		t.Fatalf("expected 1 rejection, got %d", len(decision.Rejections))
	}
	if decision.Rejections[0].Code != "PAYLOAD_DECODE_FAILED" {
		t.Fatalf("rejection code = %s, want PAYLOAD_DECODE_FAILED", decision.Rejections[0].Code)
	}
}

type updatePayload struct {
	Fields map[string]string `json:"fields"`
}
