package environmenttransport

import (
	"context"
	"testing"
	"time"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
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
	record storage.SessionRecord
	err    error
}

func (s testSessionStore) GetSession(context.Context, string, string) (storage.SessionRecord, error) {
	if s.err != nil {
		return storage.SessionRecord{}, s.err
	}
	return s.record, nil
}

type testGateStore struct {
	gate storage.SessionGate
	err  error
}

func (s testGateStore) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	if s.err != nil {
		return storage.SessionGate{}, s.err
	}
	return s.gate, nil
}

type testContentStore struct {
	environment contentstore.DaggerheartEnvironment
	err         error
}

func (s testContentStore) GetDaggerheartEnvironment(context.Context, string) (contentstore.DaggerheartEnvironment, error) {
	if s.err != nil {
		return contentstore.DaggerheartEnvironment{}, s.err
	}
	return s.environment, nil
}

type testDaggerheartStore struct {
	entity   projectionstore.DaggerheartEnvironmentEntity
	entities []projectionstore.DaggerheartEnvironmentEntity
	getErr   error
	listErr  error
	lastList struct{ campaignID, sessionID, sceneID string }
}

func (s *testDaggerheartStore) GetDaggerheartEnvironmentEntity(context.Context, string, string) (projectionstore.DaggerheartEnvironmentEntity, error) {
	if s.getErr != nil {
		return projectionstore.DaggerheartEnvironmentEntity{}, s.getErr
	}
	return s.entity, nil
}

func (s *testDaggerheartStore) ListDaggerheartEnvironmentEntities(_ context.Context, campaignID, sessionID, sceneID string) ([]projectionstore.DaggerheartEnvironmentEntity, error) {
	if s.listErr != nil {
		return nil, s.listErr
	}
	s.lastList = struct{ campaignID, sessionID, sceneID string }{campaignID: campaignID, sessionID: sessionID, sceneID: sceneID}
	return s.entities, nil
}

func newTestHandler(deps Dependencies) *Handler {
	if deps.Campaign == nil {
		deps.Campaign = testCampaignStore{record: storage.CampaignRecord{
			ID:     "camp-1",
			System: systembridge.SystemIDDaggerheart,
			Status: campaign.StatusActive,
		}}
	}
	if deps.Session == nil {
		deps.Session = testSessionStore{record: storage.SessionRecord{
			ID:         "sess-1",
			CampaignID: "camp-1",
			Status:     session.StatusActive,
		}}
	}
	if deps.Gate == nil {
		deps.Gate = testGateStore{err: storage.ErrNotFound}
	}
	if deps.Daggerheart == nil {
		deps.Daggerheart = &testDaggerheartStore{}
	}
	if deps.Content == nil {
		deps.Content = testContentStore{environment: contentstore.DaggerheartEnvironment{
			ID:         "env-1",
			Name:       "Storm Front",
			Type:       "hazard",
			Tier:       2,
			Difficulty: 4,
		}}
	}
	if deps.GenerateID == nil {
		deps.GenerateID = func() (string, error) { return "env-entity-1", nil }
	}
	return NewHandler(deps)
}

func testContext() context.Context {
	ctx := grpcmeta.WithRequestID(context.Background(), "req-1")
	return grpcmeta.WithInvocationID(ctx, "inv-1")
}

func TestCreateEnvironmentEntityRejectsNilRequest(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.CreateEnvironmentEntity(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestCreateEnvironmentEntitySuccess(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	store := &testDaggerheartStore{}
	var commandInput DomainCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			commandInput = in
			store.entity = projectionstore.DaggerheartEnvironmentEntity{
				CampaignID:          "camp-1",
				EnvironmentEntityID: "env-entity-1",
				EnvironmentID:       "env-1",
				Name:                "Storm Front",
				Type:                "hazard",
				Tier:                2,
				Difficulty:          6,
				SessionID:           "sess-1",
				SceneID:             "scene-1",
				Notes:               "dense fog",
				CreatedAt:           now,
				UpdatedAt:           now,
			}
			return nil
		},
	})

	resp, err := handler.CreateEnvironmentEntity(testContext(), &pb.DaggerheartCreateEnvironmentEntityRequest{
		CampaignId:    "camp-1",
		SessionId:     "sess-1",
		SceneId:       "scene-1",
		EnvironmentId: "env-1",
		Difficulty:    wrapperspb.Int32(6),
		Notes:         "  dense fog  ",
	})
	if err != nil {
		t.Fatalf("CreateEnvironmentEntity returned error: %v", err)
	}
	if got := resp.GetEnvironmentEntity().GetId(); got != "env-entity-1" {
		t.Fatalf("environment_entity.id = %q, want env-entity-1", got)
	}
	if got := resp.GetEnvironmentEntity().GetNotes(); got != "dense fog" {
		t.Fatalf("notes = %q, want dense fog", got)
	}
	if commandInput.CommandType != commandids.DaggerheartEnvironmentEntityCreate {
		t.Fatalf("command type = %q, want %q", commandInput.CommandType, commandids.DaggerheartEnvironmentEntityCreate)
	}
	if commandInput.EntityID != "env-entity-1" {
		t.Fatalf("entity id = %q, want env-entity-1", commandInput.EntityID)
	}
}

func TestCreateEnvironmentEntityRejectsInvalidDifficulty(t *testing.T) {
	handler := newTestHandler(Dependencies{
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.CreateEnvironmentEntity(testContext(), &pb.DaggerheartCreateEnvironmentEntityRequest{
		CampaignId:    "camp-1",
		SessionId:     "sess-1",
		SceneId:       "scene-1",
		EnvironmentId: "env-1",
		Difficulty:    wrapperspb.Int32(0),
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestUpdateEnvironmentEntitySuccess(t *testing.T) {
	now := time.Unix(1700000100, 0).UTC()
	store := &testDaggerheartStore{entity: projectionstore.DaggerheartEnvironmentEntity{
		CampaignID:          "camp-1",
		EnvironmentEntityID: "env-entity-1",
		EnvironmentID:       "env-1",
		Name:                "Storm Front",
		Type:                "hazard",
		Tier:                2,
		Difficulty:          4,
		SessionID:           "sess-1",
		SceneID:             "scene-1",
		Notes:               "old",
		CreatedAt:           now,
		UpdatedAt:           now,
	}}
	var commandInput DomainCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			commandInput = in
			store.entity.SceneID = "scene-2"
			store.entity.Notes = "fresh air"
			store.entity.Difficulty = 7
			return nil
		},
	})

	resp, err := handler.UpdateEnvironmentEntity(testContext(), &pb.DaggerheartUpdateEnvironmentEntityRequest{
		CampaignId:          "camp-1",
		EnvironmentEntityId: "env-entity-1",
		SceneId:             "scene-2",
		Notes:               wrapperspb.String("  fresh air "),
		Difficulty:          wrapperspb.Int32(7),
	})
	if err != nil {
		t.Fatalf("UpdateEnvironmentEntity returned error: %v", err)
	}
	if got := resp.GetEnvironmentEntity().GetSceneId(); got != "scene-2" {
		t.Fatalf("scene_id = %q, want scene-2", got)
	}
	if got := resp.GetEnvironmentEntity().GetDifficulty(); got != 7 {
		t.Fatalf("difficulty = %d, want 7", got)
	}
	if commandInput.CommandType != commandids.DaggerheartEnvironmentEntityUpdate {
		t.Fatalf("command type = %q, want %q", commandInput.CommandType, commandids.DaggerheartEnvironmentEntityUpdate)
	}
}

func TestUpdateEnvironmentEntityRequiresAtLeastOneField(t *testing.T) {
	store := &testDaggerheartStore{entity: projectionstore.DaggerheartEnvironmentEntity{
		CampaignID:          "camp-1",
		EnvironmentEntityID: "env-entity-1",
		SessionID:           "sess-1",
		SceneID:             "scene-1",
	}}
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteDomainCommand: func(context.Context, DomainCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	_, err := handler.UpdateEnvironmentEntity(testContext(), &pb.DaggerheartUpdateEnvironmentEntityRequest{
		CampaignId:          "camp-1",
		EnvironmentEntityId: "env-entity-1",
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestDeleteEnvironmentEntitySuccess(t *testing.T) {
	now := time.Unix(1700000200, 0).UTC()
	store := &testDaggerheartStore{entity: projectionstore.DaggerheartEnvironmentEntity{
		CampaignID:          "camp-1",
		EnvironmentEntityID: "env-entity-1",
		EnvironmentID:       "env-1",
		Name:                "Storm Front",
		Type:                "hazard",
		Tier:                2,
		Difficulty:          4,
		SessionID:           "sess-1",
		SceneID:             "scene-1",
		Notes:               "volatile",
		CreatedAt:           now,
		UpdatedAt:           now,
	}}
	var commandInput DomainCommandInput
	handler := newTestHandler(Dependencies{
		Daggerheart: store,
		ExecuteDomainCommand: func(_ context.Context, in DomainCommandInput) error {
			commandInput = in
			return nil
		},
	})

	resp, err := handler.DeleteEnvironmentEntity(testContext(), &pb.DaggerheartDeleteEnvironmentEntityRequest{
		CampaignId:          "camp-1",
		EnvironmentEntityId: "env-entity-1",
		Reason:              "resolved",
	})
	if err != nil {
		t.Fatalf("DeleteEnvironmentEntity returned error: %v", err)
	}
	if got := resp.GetEnvironmentEntity().GetId(); got != "env-entity-1" {
		t.Fatalf("environment_entity.id = %q, want env-entity-1", got)
	}
	if commandInput.CommandType != commandids.DaggerheartEnvironmentEntityDelete {
		t.Fatalf("command type = %q, want %q", commandInput.CommandType, commandids.DaggerheartEnvironmentEntityDelete)
	}
}

func TestGetEnvironmentEntitySuccess(t *testing.T) {
	store := &testDaggerheartStore{entity: projectionstore.DaggerheartEnvironmentEntity{
		CampaignID:          "camp-1",
		EnvironmentEntityID: "env-entity-1",
		EnvironmentID:       "env-1",
		Name:                "Storm Front",
		Type:                "hazard",
		Tier:                2,
		Difficulty:          4,
		SessionID:           "sess-1",
		SceneID:             "scene-1",
	}}
	handler := newTestHandler(Dependencies{Daggerheart: store})

	resp, err := handler.GetEnvironmentEntity(testContext(), &pb.DaggerheartGetEnvironmentEntityRequest{
		CampaignId:          "camp-1",
		EnvironmentEntityId: "env-entity-1",
	})
	if err != nil {
		t.Fatalf("GetEnvironmentEntity returned error: %v", err)
	}
	if got := resp.GetEnvironmentEntity().GetName(); got != "Storm Front" {
		t.Fatalf("name = %q, want Storm Front", got)
	}
}

func TestListEnvironmentEntitiesPassesSceneFilter(t *testing.T) {
	store := &testDaggerheartStore{entities: []projectionstore.DaggerheartEnvironmentEntity{
		{
			CampaignID:          "camp-1",
			EnvironmentEntityID: "env-entity-1",
			EnvironmentID:       "env-1",
			Name:                "Storm Front",
			Type:                "hazard",
			Tier:                2,
			Difficulty:          4,
			SessionID:           "sess-1",
			SceneID:             "scene-2",
		},
	}}
	handler := newTestHandler(Dependencies{Daggerheart: store})

	resp, err := handler.ListEnvironmentEntities(testContext(), &pb.DaggerheartListEnvironmentEntitiesRequest{
		CampaignId: "camp-1",
		SessionId:  "sess-1",
		SceneId:    wrapperspb.String(" scene-2 "),
	})
	if err != nil {
		t.Fatalf("ListEnvironmentEntities returned error: %v", err)
	}
	if len(resp.GetEnvironmentEntities()) != 1 {
		t.Fatalf("environment entity count = %d, want 1", len(resp.GetEnvironmentEntities()))
	}
	if store.lastList.sceneID != "scene-2" {
		t.Fatalf("scene filter = %q, want scene-2", store.lastList.sceneID)
	}
}
