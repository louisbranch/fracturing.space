package damagetransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	systembridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type testCampaignStore struct {
	record storage.CampaignRecord
}

func (s testCampaignStore) Get(context.Context, string) (storage.CampaignRecord, error) {
	return s.record, nil
}

type testSessionGateStore struct{}

func (testSessionGateStore) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	return storage.SessionGate{}, storage.ErrNotFound
}

type testOpenGateStore struct{}

func (testOpenGateStore) GetOpenSessionGate(context.Context, string, string) (storage.SessionGate, error) {
	return storage.SessionGate{GateID: "gate-1"}, nil
}

type testDaggerheartStore struct {
	profile   projectionstore.DaggerheartCharacterProfile
	state     projectionstore.DaggerheartCharacterState
	adversary projectionstore.DaggerheartAdversary
}

func (s testDaggerheartStore) GetDaggerheartCharacterProfile(context.Context, string, string) (projectionstore.DaggerheartCharacterProfile, error) {
	return s.profile, nil
}

func (s testDaggerheartStore) GetDaggerheartCharacterState(context.Context, string, string) (projectionstore.DaggerheartCharacterState, error) {
	return s.state, nil
}

func (s testDaggerheartStore) GetDaggerheartAdversary(context.Context, string, string) (projectionstore.DaggerheartAdversary, error) {
	return s.adversary, nil
}

type testEventStore struct {
	event event.Event
}

func (s testEventStore) GetEventBySeq(context.Context, string, uint64) (event.Event, error) {
	return s.event, nil
}

func newTestHandler(deps Dependencies) *Handler {
	if deps.Campaign == nil {
		deps.Campaign = testCampaignStore{record: storage.CampaignRecord{
			ID:     "camp-1",
			System: systembridge.SystemIDDaggerheart,
			Status: campaign.StatusActive,
		}}
	}
	if deps.SessionGate == nil {
		deps.SessionGate = testSessionGateStore{}
	}
	if deps.Daggerheart == nil {
		deps.Daggerheart = testDaggerheartStore{
			profile: projectionstore.DaggerheartCharacterProfile{
				CampaignID:      "camp-1",
				CharacterID:     "char-1",
				MajorThreshold:  5,
				SevereThreshold: 8,
			},
			state: projectionstore.DaggerheartCharacterState{
				CampaignID:  "camp-1",
				CharacterID: "char-1",
				Hp:          10,
				Armor:       1,
			},
			adversary: projectionstore.DaggerheartAdversary{
				CampaignID:  "camp-1",
				AdversaryID: "adv-1",
				SessionID:   "sess-1",
				HP:          10,
				Armor:       1,
				Major:       5,
				Severe:      8,
			},
		}
	}
	if deps.Event == nil {
		deps.Event = testEventStore{}
	}
	return NewHandler(deps)
}

func testContextWithSessionID(sessionID string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.SessionIDHeader, sessionID))
}

func TestHandlerApplyDamageSuccess(t *testing.T) {
	var commandInput SystemCommandInput
	handler := newTestHandler(Dependencies{
		ExecuteSystemCommand: func(_ context.Context, in SystemCommandInput) error {
			commandInput = in
			return nil
		},
	})

	ctx := grpcmeta.WithInvocationID(grpcmeta.WithRequestID(testContextWithSessionID("sess-1"), "req-1"), "inv-1")
	resp, err := handler.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     3,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if err != nil {
		t.Fatalf("ApplyDamage returned error: %v", err)
	}
	if resp.CharacterID != "char-1" {
		t.Fatalf("character_id = %q, want char-1", resp.CharacterID)
	}
	if commandInput.CommandType != commandids.DaggerheartDamageApply {
		t.Fatalf("command type = %q", commandInput.CommandType)
	}
}

func TestHandlerApplyAdversaryDamageSuccess(t *testing.T) {
	var commandInput SystemCommandInput
	handler := newTestHandler(Dependencies{
		ExecuteSystemCommand: func(_ context.Context, in SystemCommandInput) error {
			commandInput = in
			return nil
		},
		LoadAdversaryForSession: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
			return projectionstore.DaggerheartAdversary{
				CampaignID:  "camp-1",
				AdversaryID: "adv-1",
				SessionID:   "sess-1",
				HP:          10,
				Armor:       1,
				Major:       5,
				Severe:      8,
			}, nil
		},
	})

	ctx := grpcmeta.WithInvocationID(grpcmeta.WithRequestID(testContextWithSessionID("sess-1"), "req-adv-1"), "inv-1")
	resp, err := handler.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     3,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if err != nil {
		t.Fatalf("ApplyAdversaryDamage returned error: %v", err)
	}
	if resp.AdversaryID != "adv-1" {
		t.Fatalf("adversary_id = %q, want adv-1", resp.AdversaryID)
	}
	if commandInput.CommandType != commandids.DaggerheartAdversaryDamageApply {
		t.Fatalf("command type = %q", commandInput.CommandType)
	}
}

func TestHandlerApplyDamageRequiresExecutor(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	ctx := testContextWithSessionID("sess-1")
	_, err := handler.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     1,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v (err=%v)", status.Code(err), codes.Internal, err)
	}
}

func TestHandlerApplyDamageRejectsNilRequest(t *testing.T) {
	handler := newTestHandler(Dependencies{})

	_, err := handler.ApplyDamage(context.Background(), nil)
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestHandlerApplyDamageRequiresRollSeqWhenFlagged(t *testing.T) {
	handler := newTestHandler(Dependencies{
		ExecuteSystemCommand: func(context.Context, SystemCommandInput) error { return nil },
	})

	ctx := testContextWithSessionID("sess-1")
	_, err := handler.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:        "camp-1",
		CharacterId:       "char-1",
		RequireDamageRoll: true,
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     2,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v (err=%v)", status.Code(err), codes.InvalidArgument, err)
	}
}

func TestHandlerApplyDamageRejectsNonRollEvent(t *testing.T) {
	handler := newTestHandler(Dependencies{
		Event: testEventStore{event: event.Event{Type: event.Type("other.event")}},
		ExecuteSystemCommand: func(context.Context, SystemCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
	})

	ctx := testContextWithSessionID("sess-1")
	rollSeq := uint64(7)
	_, err := handler.ApplyDamage(ctx, &pb.DaggerheartApplyDamageRequest{
		CampaignId:  "camp-1",
		CharacterId: "char-1",
		RollSeq:     &rollSeq,
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     2,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v (err=%v)", status.Code(err), codes.InvalidArgument, err)
	}
}

func TestHandlerApplyAdversaryDamageRequiresLoader(t *testing.T) {
	handler := newTestHandler(Dependencies{
		ExecuteSystemCommand: func(context.Context, SystemCommandInput) error { return nil },
	})

	ctx := testContextWithSessionID("sess-1")
	_, err := handler.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     1,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if status.Code(err) != codes.Internal {
		t.Fatalf("status code = %v, want %v (err=%v)", status.Code(err), codes.Internal, err)
	}
}

func TestHandlerApplyAdversaryDamageRejectsOpenGate(t *testing.T) {
	handler := newTestHandler(Dependencies{
		SessionGate: testOpenGateStore{},
		ExecuteSystemCommand: func(context.Context, SystemCommandInput) error {
			t.Fatal("unexpected command execution")
			return nil
		},
		LoadAdversaryForSession: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
			t.Fatal("unexpected adversary load")
			return projectionstore.DaggerheartAdversary{}, nil
		},
	})

	ctx := testContextWithSessionID("sess-1")
	_, err := handler.ApplyAdversaryDamage(ctx, &pb.DaggerheartApplyAdversaryDamageRequest{
		CampaignId:  "camp-1",
		AdversaryId: "adv-1",
		Damage: &pb.DaggerheartDamageRequest{
			Amount:     1,
			DamageType: pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL,
		},
	})
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status code = %v, want %v (err=%v)", status.Code(err), codes.FailedPrecondition, err)
	}
}
