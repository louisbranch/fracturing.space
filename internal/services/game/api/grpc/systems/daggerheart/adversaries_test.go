package daggerheart

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

// fakeDaggerheartAdversaryStore extends the fake store to support adversary CRUD.
type fakeDaggerheartAdversaryStore struct {
	fakeDaggerheartStore
	adversaries map[string]storage.DaggerheartAdversary
}

func newFakeDaggerheartAdversaryStore() *fakeDaggerheartAdversaryStore {
	return &fakeDaggerheartAdversaryStore{
		fakeDaggerheartStore: *newFakeDaggerheartStore(),
		adversaries:          make(map[string]storage.DaggerheartAdversary),
	}
}

func (s *fakeDaggerheartAdversaryStore) PutDaggerheartAdversary(_ context.Context, a storage.DaggerheartAdversary) error {
	s.adversaries[a.CampaignID+":"+a.AdversaryID] = a
	return nil
}

func (s *fakeDaggerheartAdversaryStore) GetDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) (storage.DaggerheartAdversary, error) {
	a, ok := s.adversaries[campaignID+":"+adversaryID]
	if !ok {
		return storage.DaggerheartAdversary{}, storage.ErrNotFound
	}
	return a, nil
}

func (s *fakeDaggerheartAdversaryStore) ListDaggerheartAdversaries(_ context.Context, campaignID, sessionID string) ([]storage.DaggerheartAdversary, error) {
	var result []storage.DaggerheartAdversary
	for _, a := range s.adversaries {
		if a.CampaignID != campaignID {
			continue
		}
		if sessionID != "" && a.SessionID != sessionID {
			continue
		}
		result = append(result, a)
	}
	return result, nil
}

func (s *fakeDaggerheartAdversaryStore) DeleteDaggerheartAdversary(_ context.Context, campaignID, adversaryID string) error {
	delete(s.adversaries, campaignID+":"+adversaryID)
	return nil
}

type dynamicDomainEngine struct {
	store       storage.EventStore
	calls       int
	lastCommand command.Command
}

func (d *dynamicDomainEngine) Execute(ctx context.Context, cmd command.Command) (engine.Result, error) {
	d.calls++
	d.lastCommand = cmd

	var eventType event.Type
	switch cmd.Type {
	case command.Type("action.adversary.create"):
		eventType = event.Type("action.adversary_created")
	case command.Type("action.adversary.update"):
		eventType = event.Type("action.adversary_updated")
	case command.Type("action.adversary.delete"):
		eventType = event.Type("action.adversary_deleted")
	case command.Type("action.adversary_damage.apply"):
		eventType = event.Type("action.adversary_damage_applied")
	case command.Type("action.adversary_condition.change"):
		eventType = event.Type("action.adversary_condition_changed")
	case command.Type("action.adversary_attack.resolve"):
		eventType = event.Type("action.adversary_attack_resolved")
	case command.Type("action.adversary_action.resolve"):
		eventType = event.Type("action.adversary_action_resolved")
	case command.Type("action.adversary_roll.resolve"):
		eventType = event.Type("action.adversary_roll_resolved")
	default:
		return engine.Result{}, nil
	}

	entityID := strings.TrimSpace(cmd.EntityID)
	if entityID == "" {
		var payload struct {
			AdversaryID string `json:"adversary_id"`
		}
		_ = json.Unmarshal(cmd.PayloadJSON, &payload)
		entityID = strings.TrimSpace(payload.AdversaryID)
	}

	evt := event.Event{
		CampaignID:    cmd.CampaignID,
		Type:          eventType,
		Timestamp:     time.Now().UTC(),
		ActorType:     event.ActorType(cmd.ActorType),
		ActorID:       cmd.ActorID,
		SessionID:     cmd.SessionID,
		RequestID:     cmd.RequestID,
		InvocationID:  cmd.InvocationID,
		EntityType:    "adversary",
		EntityID:      entityID,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		PayloadJSON:   cmd.PayloadJSON,
	}

	result := engine.Result{Decision: command.Accept(evt)}
	if d.store == nil || len(result.Decision.Events) == 0 {
		return result, nil
	}
	stored := make([]event.Event, 0, len(result.Decision.Events))
	for _, evt := range result.Decision.Events {
		storedEvent, err := d.store.AppendEvent(ctx, evt)
		if err != nil {
			return engine.Result{}, err
		}
		stored = append(stored, storedEvent)
	}
	result.Decision.Events = stored
	return result, nil
}

func newAdversaryTestService() *DaggerheartService {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["camp-1"] = storage.CampaignRecord{
		ID:     "camp-1",
		Status: campaign.StatusActive,
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	}
	campaignStore.campaigns["camp-non-dh"] = storage.CampaignRecord{
		ID:     "camp-non-dh",
		Status: campaign.StatusActive,
		System: commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED,
	}

	sessStore := newFakeSessionStore()
	sessStore.sessions["camp-1:sess-1"] = storage.SessionRecord{
		ID:         "sess-1",
		CampaignID: "camp-1",
		Status:     session.StatusActive,
	}

	dhStore := newFakeDaggerheartAdversaryStore()
	eventStore := newFakeActionEventStore()

	return &DaggerheartService{
		stores: Stores{
			Campaign:    campaignStore,
			Daggerheart: dhStore,
			Event:       eventStore,
			Domain:      &dynamicDomainEngine{store: eventStore},
			SessionGate: &fakeSessionGateStore{},
			Session:     sessStore,
		},
		seedFunc: func() (int64, error) { return 42, nil },
	}
}

// --- CreateAdversary tests ---

func TestCreateAdversary_NilRequest(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.CreateAdversary(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateAdversary_MissingStores(t *testing.T) {
	svc := &DaggerheartService{}
	_, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1", Name: "Goblin",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateAdversary_MissingCampaignID(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		Name: "Goblin",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateAdversary_MissingName(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestCreateAdversary_CampaignNotFound(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "nonexistent", Name: "Goblin",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateAdversary_NonDaggerheartCampaign(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-non-dh", Name: "Goblin",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestCreateAdversary_Success(t *testing.T) {
	svc := newAdversaryTestService()
	resp, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1",
		Name:       "Goblin",
		Kind:       "bruiser",
		Notes:      "A test goblin",
		Hp:         wrapperspb.Int32(6),
		HpMax:      wrapperspb.Int32(6),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Adversary == nil {
		t.Fatal("expected adversary in response")
	}
	if resp.Adversary.Name != "Goblin" {
		t.Errorf("name = %q, want Goblin", resp.Adversary.Name)
	}
	if resp.Adversary.Kind != "bruiser" {
		t.Errorf("kind = %q, want bruiser", resp.Adversary.Kind)
	}
	if resp.Adversary.Id == "" {
		t.Error("expected non-empty adversary ID")
	}
}

func TestCreateAdversary_WithSession(t *testing.T) {
	svc := newAdversaryTestService()
	resp, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1",
		Name:       "Goblin",
		SessionId:  wrapperspb.String("sess-1"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Adversary.SessionId == nil || resp.Adversary.SessionId.Value != "sess-1" {
		t.Errorf("expected session_id = sess-1, got %v", resp.Adversary.SessionId)
	}
}

func TestCreateAdversary_UsesDomainEngine(t *testing.T) {
	svc := newAdversaryTestService()
	engine := &dynamicDomainEngine{store: svc.stores.Event}
	svc.stores.Domain = engine

	_, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1",
		Name:       " Goblin ",
		Kind:       "bruiser",
		Notes:      " note ",
	})
	if err != nil {
		t.Fatalf("create adversary: %v", err)
	}
	if engine.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", engine.calls)
	}
	if engine.lastCommand.Type != command.Type("action.adversary.create") {
		t.Fatalf("command type = %s, want %s", engine.lastCommand.Type, "action.adversary.create")
	}
	if engine.lastCommand.SystemID != daggerheart.SystemID {
		t.Fatalf("command system id = %s, want %s", engine.lastCommand.SystemID, daggerheart.SystemID)
	}
	if engine.lastCommand.SystemVersion != daggerheart.SystemVersion {
		t.Fatalf("command system version = %s, want %s", engine.lastCommand.SystemVersion, daggerheart.SystemVersion)
	}

	var payload struct {
		AdversaryID string `json:"adversary_id"`
		Name        string `json:"name"`
		Kind        string `json:"kind"`
		Notes       string `json:"notes"`
	}
	if err := json.Unmarshal(engine.lastCommand.PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload.AdversaryID == "" {
		t.Fatal("expected adversary_id in command payload")
	}
	if payload.Name != "Goblin" {
		t.Fatalf("name = %s, want %s", payload.Name, "Goblin")
	}
	if payload.Kind != "bruiser" {
		t.Fatalf("kind = %s, want %s", payload.Kind, "bruiser")
	}
	if payload.Notes != "note" {
		t.Fatalf("notes = %s, want %s", payload.Notes, "note")
	}
}

func TestCreateAdversary_SessionNotFound(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1",
		Name:       "Goblin",
		SessionId:  wrapperspb.String("nonexistent"),
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestCreateAdversary_InvalidStats(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1",
		Name:       "Goblin",
		HpMax:      wrapperspb.Int32(0),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

// --- GetAdversary tests ---

func TestGetAdversary_NilRequest(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.GetAdversary(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetAdversary_MissingCampaignID(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.GetAdversary(context.Background(), &pb.DaggerheartGetAdversaryRequest{
		AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetAdversary_MissingAdversaryID(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.GetAdversary(context.Background(), &pb.DaggerheartGetAdversaryRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetAdversary_NotFound(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.GetAdversary(context.Background(), &pb.DaggerheartGetAdversaryRequest{
		CampaignId: "camp-1", AdversaryId: "nonexistent",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestGetAdversary_NonDaggerheartCampaign(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.GetAdversary(context.Background(), &pb.DaggerheartGetAdversaryRequest{
		CampaignId: "camp-non-dh", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestGetAdversary_Success(t *testing.T) {
	svc := newAdversaryTestService()

	// Create an adversary first.
	createResp, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1", Name: "Goblin",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	getResp, err := svc.GetAdversary(context.Background(), &pb.DaggerheartGetAdversaryRequest{
		CampaignId: "camp-1", AdversaryId: createResp.Adversary.Id,
	})
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if getResp.Adversary.Name != "Goblin" {
		t.Errorf("name = %q, want Goblin", getResp.Adversary.Name)
	}
}

// --- ListAdversaries tests ---

func TestListAdversaries_NilRequest(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.ListAdversaries(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListAdversaries_MissingCampaignID(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.ListAdversaries(context.Background(), &pb.DaggerheartListAdversariesRequest{})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListAdversaries_EmptyResult(t *testing.T) {
	svc := newAdversaryTestService()
	resp, err := svc.ListAdversaries(context.Background(), &pb.DaggerheartListAdversariesRequest{
		CampaignId: "camp-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Adversaries) != 0 {
		t.Errorf("expected 0 adversaries, got %d", len(resp.Adversaries))
	}
}

func TestListAdversaries_WithResults(t *testing.T) {
	svc := newAdversaryTestService()

	// Create two adversaries.
	for _, name := range []string{"Goblin", "Orc"} {
		_, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
			CampaignId: "camp-1", Name: name,
		})
		if err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}

	resp, err := svc.ListAdversaries(context.Background(), &pb.DaggerheartListAdversariesRequest{
		CampaignId: "camp-1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Adversaries) != 2 {
		t.Errorf("expected 2 adversaries, got %d", len(resp.Adversaries))
	}
}

func TestListAdversaries_FilterBySession(t *testing.T) {
	svc := newAdversaryTestService()

	_, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1", Name: "Session Goblin",
		SessionId: wrapperspb.String("sess-1"),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	_, err = svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1", Name: "Global Orc",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	resp, err := svc.ListAdversaries(context.Background(), &pb.DaggerheartListAdversariesRequest{
		CampaignId: "camp-1",
		SessionId:  wrapperspb.String("sess-1"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Adversaries) != 1 {
		t.Errorf("expected 1 adversary, got %d", len(resp.Adversaries))
	}
}

// --- UpdateAdversary tests ---

func TestUpdateAdversary_NilRequest(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.UpdateAdversary(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateAdversary_MissingCampaignID(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.UpdateAdversary(context.Background(), &pb.DaggerheartUpdateAdversaryRequest{
		AdversaryId: "adv-1",
		Name:        wrapperspb.String("New Name"),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateAdversary_MissingAdversaryID(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.UpdateAdversary(context.Background(), &pb.DaggerheartUpdateAdversaryRequest{
		CampaignId: "camp-1",
		Name:       wrapperspb.String("New Name"),
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateAdversary_NoFieldsProvided(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.UpdateAdversary(context.Background(), &pb.DaggerheartUpdateAdversaryRequest{
		CampaignId: "camp-1", AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestUpdateAdversary_Success(t *testing.T) {
	svc := newAdversaryTestService()

	createResp, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1", Name: "Goblin",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	updateResp, err := svc.UpdateAdversary(context.Background(), &pb.DaggerheartUpdateAdversaryRequest{
		CampaignId:  "camp-1",
		AdversaryId: createResp.Adversary.Id,
		Name:        wrapperspb.String("Hobgoblin"),
		Notes:       wrapperspb.String("Upgraded"),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if updateResp.Adversary.Name != "Hobgoblin" {
		t.Errorf("name = %q, want Hobgoblin", updateResp.Adversary.Name)
	}
}

func TestUpdateAdversary_UsesDomainEngine(t *testing.T) {
	svc := newAdversaryTestService()
	engine := &dynamicDomainEngine{store: svc.stores.Event}
	svc.stores.Domain = engine

	createResp, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1", Name: "Goblin",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	engine.calls = 0

	_, err = svc.UpdateAdversary(context.Background(), &pb.DaggerheartUpdateAdversaryRequest{
		CampaignId:  "camp-1",
		AdversaryId: createResp.Adversary.Id,
		Name:        wrapperspb.String("Hobgoblin"),
	})
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if engine.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", engine.calls)
	}
	if engine.lastCommand.Type != command.Type("action.adversary.update") {
		t.Fatalf("command type = %s, want %s", engine.lastCommand.Type, "action.adversary.update")
	}

	var payload struct {
		AdversaryID string `json:"adversary_id"`
		Name        string `json:"name"`
	}
	if err := json.Unmarshal(engine.lastCommand.PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload.AdversaryID != createResp.Adversary.Id {
		t.Fatalf("adversary_id = %s, want %s", payload.AdversaryID, createResp.Adversary.Id)
	}
	if payload.Name != "Hobgoblin" {
		t.Fatalf("name = %s, want %s", payload.Name, "Hobgoblin")
	}
}

func TestUpdateAdversary_NotFound(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.UpdateAdversary(context.Background(), &pb.DaggerheartUpdateAdversaryRequest{
		CampaignId:  "camp-1",
		AdversaryId: "nonexistent",
		Name:        wrapperspb.String("New Name"),
	})
	assertStatusCode(t, err, codes.Internal)
}

// --- DeleteAdversary tests ---

func TestDeleteAdversary_NilRequest(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.DeleteAdversary(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteAdversary_MissingCampaignID(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.DeleteAdversary(context.Background(), &pb.DaggerheartDeleteAdversaryRequest{
		AdversaryId: "adv-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteAdversary_MissingAdversaryID(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.DeleteAdversary(context.Background(), &pb.DaggerheartDeleteAdversaryRequest{
		CampaignId: "camp-1",
	})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestDeleteAdversary_NotFound(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.DeleteAdversary(context.Background(), &pb.DaggerheartDeleteAdversaryRequest{
		CampaignId: "camp-1", AdversaryId: "nonexistent",
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestDeleteAdversary_Success(t *testing.T) {
	svc := newAdversaryTestService()

	createResp, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1", Name: "Goblin",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	deleteResp, err := svc.DeleteAdversary(context.Background(), &pb.DaggerheartDeleteAdversaryRequest{
		CampaignId:  "camp-1",
		AdversaryId: createResp.Adversary.Id,
		Reason:      "Test cleanup",
	})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if deleteResp.Adversary.Name != "Goblin" {
		t.Errorf("expected deleted adversary name = Goblin, got %q", deleteResp.Adversary.Name)
	}

	// Verify adversary is gone.
	_, err = svc.GetAdversary(context.Background(), &pb.DaggerheartGetAdversaryRequest{
		CampaignId: "camp-1", AdversaryId: createResp.Adversary.Id,
	})
	assertStatusCode(t, err, codes.Internal)
}

func TestDeleteAdversary_UsesDomainEngine(t *testing.T) {
	svc := newAdversaryTestService()
	engine := &dynamicDomainEngine{store: svc.stores.Event}
	svc.stores.Domain = engine

	createResp, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1", Name: "Goblin",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	engine.calls = 0

	_, err = svc.DeleteAdversary(context.Background(), &pb.DaggerheartDeleteAdversaryRequest{
		CampaignId:  "camp-1",
		AdversaryId: createResp.Adversary.Id,
		Reason:      " cleanup ",
	})
	if err != nil {
		t.Fatalf("delete: %v", err)
	}
	if engine.calls != 1 {
		t.Fatalf("expected domain to be called once, got %d", engine.calls)
	}
	if engine.lastCommand.Type != command.Type("action.adversary.delete") {
		t.Fatalf("command type = %s, want %s", engine.lastCommand.Type, "action.adversary.delete")
	}

	var payload struct {
		AdversaryID string `json:"adversary_id"`
		Reason      string `json:"reason"`
	}
	if err := json.Unmarshal(engine.lastCommand.PayloadJSON, &payload); err != nil {
		t.Fatalf("decode command payload: %v", err)
	}
	if payload.AdversaryID != createResp.Adversary.Id {
		t.Fatalf("adversary_id = %s, want %s", payload.AdversaryID, createResp.Adversary.Id)
	}
	if payload.Reason != "cleanup" {
		t.Fatalf("reason = %s, want %s", payload.Reason, "cleanup")
	}
}

// --- loadAdversaryForSession tests ---

func TestLoadAdversaryForSession_NotFound(t *testing.T) {
	svc := newAdversaryTestService()
	_, err := svc.loadAdversaryForSession(context.Background(), "camp-1", "sess-1", "nonexistent")
	assertStatusCode(t, err, codes.NotFound)
}

func TestLoadAdversaryForSession_WrongSession(t *testing.T) {
	svc := newAdversaryTestService()

	createResp, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1", Name: "Goblin",
		SessionId: wrapperspb.String("sess-1"),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	_, err = svc.loadAdversaryForSession(context.Background(), "camp-1", "other-session", createResp.Adversary.Id)
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestLoadAdversaryForSession_Success(t *testing.T) {
	svc := newAdversaryTestService()

	createResp, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1", Name: "Goblin",
		SessionId: wrapperspb.String("sess-1"),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	a, err := svc.loadAdversaryForSession(context.Background(), "camp-1", "sess-1", createResp.Adversary.Id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if a.Name != "Goblin" {
		t.Errorf("name = %q, want Goblin", a.Name)
	}
}

func TestLoadAdversaryForSession_NoSessionAssigned(t *testing.T) {
	svc := newAdversaryTestService()

	createResp, err := svc.CreateAdversary(context.Background(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1", Name: "Goblin",
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Global adversaries (no session) can be loaded from any session.
	a, err := svc.loadAdversaryForSession(context.Background(), "camp-1", "sess-1", createResp.Adversary.Id)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if a.Name != "Goblin" {
		t.Errorf("name = %q, want Goblin", a.Name)
	}
}
