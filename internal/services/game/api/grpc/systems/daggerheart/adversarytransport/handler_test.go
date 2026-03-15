package adversarytransport

import (
	"context"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	gmetadata "google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type testCampaignStore struct {
	record storage.CampaignRecord
	err    error
}

func (s testCampaignStore) Get(context.Context, string) (storage.CampaignRecord, error) {
	if s.err != nil {
		return storage.CampaignRecord{}, s.err
	}
	return s.record, nil
}

type testSessionStore struct {
	err error
}

func (s testSessionStore) GetSession(context.Context, string, string) (storage.SessionRecord, error) {
	if s.err != nil {
		return storage.SessionRecord{}, s.err
	}
	return storage.SessionRecord{}, nil
}

type testGateStore struct {
	gate storage.SessionGate
	err  error
}

func (s testGateStore) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	if s.err != nil {
		return storage.SessionGate{}, s.err
	}
	if s.gate.GateID != "" {
		return s.gate, nil
	}
	return storage.SessionGate{}, storage.ErrNotFound
}

type testDaggerheartStore struct {
	adversaries map[string]projectionstore.DaggerheartAdversary
	err         error
}

func (s *testDaggerheartStore) GetDaggerheartAdversary(_ context.Context, _, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	if s.err != nil {
		return projectionstore.DaggerheartAdversary{}, s.err
	}
	adversary, ok := s.adversaries[adversaryID]
	if !ok {
		return projectionstore.DaggerheartAdversary{}, storage.ErrNotFound
	}
	return adversary, nil
}

func (s *testDaggerheartStore) ListDaggerheartAdversaries(_ context.Context, _, sessionID string) ([]projectionstore.DaggerheartAdversary, error) {
	if s.err != nil {
		return nil, s.err
	}
	out := make([]projectionstore.DaggerheartAdversary, 0, len(s.adversaries))
	for _, adversary := range s.adversaries {
		if sessionID == "" || adversary.SessionID == sessionID {
			out = append(out, adversary)
		}
	}
	return out, nil
}

func testContext() context.Context {
	return gmetadata.NewIncomingContext(context.Background(), gmetadata.Pairs("x-session-id", "sess-1"))
}

func newTestHandler(deps Dependencies) *Handler {
	if deps.Campaign == nil {
		deps.Campaign = testCampaignStore{record: storage.CampaignRecord{
			ID:     "camp-1",
			System: systembridge.SystemIDDaggerheart,
			Status: campaign.StatusActive,
		}}
	}
	if deps.Gate == nil {
		deps.Gate = testGateStore{err: storage.ErrNotFound}
	}
	if deps.Daggerheart == nil {
		deps.Daggerheart = &testDaggerheartStore{adversaries: map[string]projectionstore.DaggerheartAdversary{}}
	}
	return NewHandler(deps)
}

func TestHandlerCreateAdversarySuccess(t *testing.T) {
	store := &testDaggerheartStore{adversaries: map[string]projectionstore.DaggerheartAdversary{}}
	var command DomainCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		GenerateID:  func() (string, error) { return "adv-1", nil },
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			command = in
			store.adversaries[in.EntityID] = projectionstore.DaggerheartAdversary{
				AdversaryID: in.EntityID,
				CampaignID:  in.CampaignID,
				Name:        "Rival",
				HP:          4,
				HPMax:       6,
				CreatedAt:   time.Now(),
				UpdatedAt:   time.Now(),
			}
			return nil
		},
	})

	resp, err := handler.CreateAdversary(testContext(), &pb.DaggerheartCreateAdversaryRequest{
		CampaignId: "camp-1",
		Name:       "Rival",
		HpMax:      wrapperspb.Int32(6),
	})
	if err != nil {
		t.Fatalf("CreateAdversary returned error: %v", err)
	}
	if resp.GetAdversary().GetId() != "adv-1" {
		t.Fatalf("adversary id = %q, want adv-1", resp.GetAdversary().GetId())
	}
	if command.CommandType == "" {
		t.Fatal("expected command callback to be invoked")
	}
}

func TestHandlerUpdateAdversarySuccess(t *testing.T) {
	store := &testDaggerheartStore{adversaries: map[string]projectionstore.DaggerheartAdversary{
		"adv-1": {AdversaryID: "adv-1", CampaignID: "camp-1", Name: "Old", HP: 4, HPMax: 6, Stress: 1, StressMax: 2, CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}}
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			current := store.adversaries[in.EntityID]
			current.Name = "New"
			current.HP = 5
			current.UpdatedAt = time.Now()
			store.adversaries[in.EntityID] = current
			return nil
		},
	})

	resp, err := handler.UpdateAdversary(testContext(), &pb.DaggerheartUpdateAdversaryRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Name:        wrapperspb.String("New"),
		Hp:          wrapperspb.Int32(5),
	})
	if err != nil {
		t.Fatalf("UpdateAdversary returned error: %v", err)
	}
	if resp.GetAdversary().GetName() != "New" {
		t.Fatalf("name = %q, want New", resp.GetAdversary().GetName())
	}
}

func TestHandlerDeleteAdversarySuccess(t *testing.T) {
	store := &testDaggerheartStore{adversaries: map[string]projectionstore.DaggerheartAdversary{
		"adv-1": {AdversaryID: "adv-1", CampaignID: "camp-1", Name: "Old", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}}
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			delete(store.adversaries, in.EntityID)
			return nil
		},
	})

	resp, err := handler.DeleteAdversary(testContext(), &pb.DaggerheartDeleteAdversaryRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
	})
	if err != nil {
		t.Fatalf("DeleteAdversary returned error: %v", err)
	}
	if resp.GetAdversary().GetId() != "adv-1" {
		t.Fatalf("adversary id = %q, want adv-1", resp.GetAdversary().GetId())
	}
}

func TestHandlerReadOperations(t *testing.T) {
	store := &testDaggerheartStore{adversaries: map[string]projectionstore.DaggerheartAdversary{
		"adv-1": {AdversaryID: "adv-1", CampaignID: "camp-1", Name: "One", SessionID: "sess-1", CreatedAt: time.Now(), UpdatedAt: time.Now()},
		"adv-2": {AdversaryID: "adv-2", CampaignID: "camp-1", Name: "Two", CreatedAt: time.Now(), UpdatedAt: time.Now()},
	}}
	handler := newTestHandler(Dependencies{Daggerheart: store})

	getResp, err := handler.GetAdversary(testContext(), &pb.DaggerheartGetAdversaryRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
	})
	if err != nil {
		t.Fatalf("GetAdversary returned error: %v", err)
	}
	if getResp.GetAdversary().GetId() != "adv-1" {
		t.Fatalf("adversary id = %q, want adv-1", getResp.GetAdversary().GetId())
	}

	listResp, err := handler.ListAdversaries(testContext(), &pb.DaggerheartListAdversariesRequest{
		CampaignId: "camp-1",
	})
	if err != nil {
		t.Fatalf("ListAdversaries returned error: %v", err)
	}
	if len(listResp.GetAdversaries()) != 2 {
		t.Fatalf("adversaries = %d, want 2", len(listResp.GetAdversaries()))
	}
}

func TestHandlerRequireDependencies(t *testing.T) {
	handler := NewHandler(Dependencies{})
	if _, err := handler.GetAdversary(testContext(), &pb.DaggerheartGetAdversaryRequest{CampaignId: "camp-1", AdversaryId: "adv-1"}); status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
	}
}
