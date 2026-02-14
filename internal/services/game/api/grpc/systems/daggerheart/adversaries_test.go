package daggerheart

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
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

func newAdversaryTestService() *DaggerheartService {
	campaignStore := newFakeCampaignStore()
	campaignStore.campaigns["camp-1"] = campaign.Campaign{
		ID:     "camp-1",
		Status: campaign.CampaignStatusActive,
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	}
	campaignStore.campaigns["camp-non-dh"] = campaign.Campaign{
		ID:     "camp-non-dh",
		Status: campaign.CampaignStatusActive,
		System: commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED,
	}

	sessStore := newFakeSessionStore()
	sessStore.sessions["camp-1:sess-1"] = session.Session{
		ID:         "sess-1",
		CampaignID: "camp-1",
		Status:     session.SessionStatusActive,
	}

	dhStore := newFakeDaggerheartAdversaryStore()

	return &DaggerheartService{
		stores: Stores{
			Campaign:    campaignStore,
			Daggerheart: dhStore,
			Event:       newFakeActionEventStore(),
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
