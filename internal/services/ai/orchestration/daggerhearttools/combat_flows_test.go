package daggerhearttools

import (
	"context"
	"encoding/json"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc"
)

type combatFlowTestRuntime struct {
	campaignID        string
	sessionID         string
	sceneID           string
	snapshotClient    statev1.SnapshotServiceClient
	sessionClient     statev1.SessionServiceClient
	daggerheartClient pb.DaggerheartServiceClient
}

func (combatFlowTestRuntime) CharacterClient() statev1.CharacterServiceClient { return nil }
func (r combatFlowTestRuntime) SnapshotClient() statev1.SnapshotServiceClient {
	return r.snapshotClient
}
func (r combatFlowTestRuntime) SessionClient() statev1.SessionServiceClient { return r.sessionClient }
func (r combatFlowTestRuntime) DaggerheartClient() pb.DaggerheartServiceClient {
	return r.daggerheartClient
}
func (combatFlowTestRuntime) CallContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}
func (r combatFlowTestRuntime) ResolveCampaignID(explicit string) string {
	if explicit != "" {
		return explicit
	}
	return r.campaignID
}
func (r combatFlowTestRuntime) ResolveSessionID(explicit string) string {
	if explicit != "" {
		return explicit
	}
	return r.sessionID
}
func (r combatFlowTestRuntime) ResolveSceneID(_ context.Context, _, explicit string) (string, error) {
	if explicit != "" {
		return explicit, nil
	}
	return r.sceneID, nil
}

type fakeSnapshotClient struct {
	statev1.SnapshotServiceClient
	getSnapshot func(context.Context, *statev1.GetSnapshotRequest, ...grpc.CallOption) (*statev1.GetSnapshotResponse, error)
}

func (c fakeSnapshotClient) GetSnapshot(ctx context.Context, in *statev1.GetSnapshotRequest, opts ...grpc.CallOption) (*statev1.GetSnapshotResponse, error) {
	return c.getSnapshot(ctx, in, opts...)
}

type fakeSessionClient struct {
	statev1.SessionServiceClient
	getSessionSpotlight func(context.Context, *statev1.GetSessionSpotlightRequest, ...grpc.CallOption) (*statev1.GetSessionSpotlightResponse, error)
}

func (c fakeSessionClient) GetSessionSpotlight(ctx context.Context, in *statev1.GetSessionSpotlightRequest, opts ...grpc.CallOption) (*statev1.GetSessionSpotlightResponse, error) {
	return c.getSessionSpotlight(ctx, in, opts...)
}

type fakeDaggerheartToolClient struct {
	pb.DaggerheartServiceClient
	applyDamage                func(context.Context, *pb.DaggerheartApplyDamageRequest, ...grpc.CallOption) (*pb.DaggerheartApplyDamageResponse, error)
	sessionAttackFlow          func(context.Context, *pb.SessionAttackFlowRequest, ...grpc.CallOption) (*pb.SessionAttackFlowResponse, error)
	sessionAdversaryAttackFlow func(context.Context, *pb.SessionAdversaryAttackFlowRequest, ...grpc.CallOption) (*pb.SessionAdversaryAttackFlowResponse, error)
}

func (c fakeDaggerheartToolClient) ApplyDamage(ctx context.Context, in *pb.DaggerheartApplyDamageRequest, opts ...grpc.CallOption) (*pb.DaggerheartApplyDamageResponse, error) {
	return c.applyDamage(ctx, in, opts...)
}

func (c fakeDaggerheartToolClient) SessionAttackFlow(ctx context.Context, in *pb.SessionAttackFlowRequest, opts ...grpc.CallOption) (*pb.SessionAttackFlowResponse, error) {
	return c.sessionAttackFlow(ctx, in, opts...)
}

func (c fakeDaggerheartToolClient) SessionAdversaryAttackFlow(ctx context.Context, in *pb.SessionAdversaryAttackFlowRequest, opts ...grpc.CallOption) (*pb.SessionAdversaryAttackFlowResponse, error) {
	return c.sessionAdversaryAttackFlow(ctx, in, opts...)
}

func TestAttackFlowResolveCheckpointUsesApplyDamage(t *testing.T) {
	var applyDamageReq *pb.DaggerheartApplyDamageRequest
	runtime := combatFlowTestRuntime{
		campaignID: "camp-1",
		sessionID:  "sess-1",
		snapshotClient: fakeSnapshotClient{
			getSnapshot: func(context.Context, *statev1.GetSnapshotRequest, ...grpc.CallOption) (*statev1.GetSnapshotResponse, error) {
				return &statev1.GetSnapshotResponse{}, nil
			},
		},
		sessionClient: fakeSessionClient{
			getSessionSpotlight: func(context.Context, *statev1.GetSessionSpotlightRequest, ...grpc.CallOption) (*statev1.GetSessionSpotlightResponse, error) {
				return &statev1.GetSessionSpotlightResponse{}, nil
			},
		},
		daggerheartClient: fakeDaggerheartToolClient{
			applyDamage: func(_ context.Context, in *pb.DaggerheartApplyDamageRequest, _ ...grpc.CallOption) (*pb.DaggerheartApplyDamageResponse, error) {
				applyDamageReq = in
				return &pb.DaggerheartApplyDamageResponse{
					CharacterId: in.GetCharacterId(),
					State:       &pb.DaggerheartCharacterState{Hp: 4},
				}, nil
			},
			sessionAttackFlow: func(context.Context, *pb.SessionAttackFlowRequest, ...grpc.CallOption) (*pb.SessionAttackFlowResponse, error) {
				t.Fatal("unexpected SessionAttackFlow call during checkpoint resume")
				return nil, nil
			},
		},
	}

	args, err := json.Marshal(map[string]any{
		"character_id":  "char-attacker",
		"target_id":     "char-target",
		"checkpoint_id": "damage-roll:42:7",
		"difficulty":    999,
		"damage": map[string]any{
			"damage_type": "PHYSICAL",
			"source":      "attack",
		},
		"target_mitigation_decision": map[string]any{
			"base_armor": "DECLINE",
		},
	})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	result, err := AttackFlowResolve(runtime, context.Background(), args)
	if err != nil {
		t.Fatalf("AttackFlowResolve returned error: %v", err)
	}
	if applyDamageReq == nil {
		t.Fatal("expected ApplyDamage request")
	}
	if applyDamageReq.GetRollSeq() != 42 {
		t.Fatalf("roll_seq = %d, want 42", applyDamageReq.GetRollSeq())
	}
	if applyDamageReq.GetDamage().GetAmount() != 7 {
		t.Fatalf("damage amount = %d, want 7", applyDamageReq.GetDamage().GetAmount())
	}
	if got := applyDamageReq.GetDamage().GetSourceCharacterIds(); len(got) != 1 || got[0] != "char-attacker" {
		t.Fatalf("source_character_ids = %v, want [char-attacker]", got)
	}
	var decoded struct {
		CharacterDamageApplied struct {
			State struct {
				HP int `json:"hp"`
			} `json:"state"`
		} `json:"character_damage_applied"`
	}
	if err := json.Unmarshal([]byte(result.Output), &decoded); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if decoded.CharacterDamageApplied.State.HP != 4 {
		t.Fatalf("hp = %d, want 4", decoded.CharacterDamageApplied.State.HP)
	}
}

func TestAdversaryAttackFlowResolveCheckpointUsesApplyDamage(t *testing.T) {
	var applyDamageReq *pb.DaggerheartApplyDamageRequest
	runtime := combatFlowTestRuntime{
		campaignID: "camp-1",
		sessionID:  "sess-1",
		daggerheartClient: fakeDaggerheartToolClient{
			applyDamage: func(_ context.Context, in *pb.DaggerheartApplyDamageRequest, _ ...grpc.CallOption) (*pb.DaggerheartApplyDamageResponse, error) {
				applyDamageReq = in
				return &pb.DaggerheartApplyDamageResponse{
					CharacterId: in.GetCharacterId(),
					State:       &pb.DaggerheartCharacterState{Hp: 3},
				}, nil
			},
			sessionAdversaryAttackFlow: func(context.Context, *pb.SessionAdversaryAttackFlowRequest, ...grpc.CallOption) (*pb.SessionAdversaryAttackFlowResponse, error) {
				t.Fatal("unexpected SessionAdversaryAttackFlow call during checkpoint resume")
				return nil, nil
			},
		},
	}

	args, err := json.Marshal(map[string]any{
		"adversary_id":  "adv-1",
		"target_id":     "char-target",
		"checkpoint_id": "damage-roll:84:5",
		"difficulty":    999,
		"damage": map[string]any{
			"damage_type": "PHYSICAL",
			"source":      "adversary attack",
		},
		"target_mitigation_decision": map[string]any{
			"base_armor": "DECLINE",
		},
	})
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}

	result, err := AdversaryAttackFlowResolve(runtime, context.Background(), args)
	if err != nil {
		t.Fatalf("AdversaryAttackFlowResolve returned error: %v", err)
	}
	if applyDamageReq == nil {
		t.Fatal("expected ApplyDamage request")
	}
	if applyDamageReq.GetRollSeq() != 84 {
		t.Fatalf("roll_seq = %d, want 84", applyDamageReq.GetRollSeq())
	}
	if applyDamageReq.GetDamage().GetAmount() != 5 {
		t.Fatalf("damage amount = %d, want 5", applyDamageReq.GetDamage().GetAmount())
	}
	if got := applyDamageReq.GetDamage().GetSourceCharacterIds(); len(got) != 1 || got[0] != "adv-1" {
		t.Fatalf("source_character_ids = %v, want [adv-1]", got)
	}
	var decoded struct {
		DamageApplied struct {
			State struct {
				HP int `json:"hp"`
			} `json:"state"`
		} `json:"damage_applied"`
	}
	if err := json.Unmarshal([]byte(result.Output), &decoded); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if decoded.DamageApplied.State.HP != 3 {
		t.Fatalf("hp = %d, want 3", decoded.DamageApplied.State.HP)
	}
}
