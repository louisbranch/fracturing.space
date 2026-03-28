package daggerhearttools

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type daggerheartRuntimeTest struct {
	daggerheart       pb.DaggerheartServiceClient
	resolveCampaignID func(string) string
	resolveSessionID  func(string) string
	resolveSceneID    func(context.Context, string, string) (string, error)
}

func (daggerheartRuntimeTest) CharacterClient() statev1.CharacterServiceClient  { return nil }
func (daggerheartRuntimeTest) SessionClient() statev1.SessionServiceClient      { return nil }
func (daggerheartRuntimeTest) SnapshotClient() statev1.SnapshotServiceClient    { return nil }
func (r daggerheartRuntimeTest) DaggerheartClient() pb.DaggerheartServiceClient { return r.daggerheart }
func (daggerheartRuntimeTest) CallContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}
func (r daggerheartRuntimeTest) ResolveCampaignID(explicit string) string {
	if r.resolveCampaignID != nil {
		return r.resolveCampaignID(explicit)
	}
	return explicit
}
func (r daggerheartRuntimeTest) ResolveSessionID(explicit string) string {
	if r.resolveSessionID != nil {
		return r.resolveSessionID(explicit)
	}
	return explicit
}
func (r daggerheartRuntimeTest) ResolveSceneID(ctx context.Context, campaignID, explicit string) (string, error) {
	if r.resolveSceneID != nil {
		return r.resolveSceneID(ctx, campaignID, explicit)
	}
	return explicit, nil
}

type daggerheartRuntimeClientStub struct {
	pb.DaggerheartServiceClient
	sessionActionRollFunc            func(context.Context, *pb.SessionActionRollRequest, ...grpc.CallOption) (*pb.SessionActionRollResponse, error)
	applyRollOutcomeFunc             func(context.Context, *pb.ApplyRollOutcomeRequest, ...grpc.CallOption) (*pb.ApplyRollOutcomeResponse, error)
	applyGmMoveFunc                  func(context.Context, *pb.DaggerheartApplyGmMoveRequest, ...grpc.CallOption) (*pb.DaggerheartApplyGmMoveResponse, error)
	createAdversaryFunc              func(context.Context, *pb.DaggerheartCreateAdversaryRequest, ...grpc.CallOption) (*pb.DaggerheartCreateAdversaryResponse, error)
	createSceneCountdownFunc         func(context.Context, *pb.DaggerheartCreateSceneCountdownRequest, ...grpc.CallOption) (*pb.DaggerheartCreateSceneCountdownResponse, error)
	advanceSceneCountdownFunc        func(context.Context, *pb.DaggerheartAdvanceSceneCountdownRequest, ...grpc.CallOption) (*pb.DaggerheartAdvanceSceneCountdownResponse, error)
	resolveSceneCountdownTriggerFunc func(context.Context, *pb.DaggerheartResolveSceneCountdownTriggerRequest, ...grpc.CallOption) (*pb.DaggerheartResolveSceneCountdownTriggerResponse, error)
	updateAdversaryFunc              func(context.Context, *pb.DaggerheartUpdateAdversaryRequest, ...grpc.CallOption) (*pb.DaggerheartUpdateAdversaryResponse, error)
	sessionAdversaryAttackFlowFunc   func(context.Context, *pb.SessionAdversaryAttackFlowRequest, ...grpc.CallOption) (*pb.SessionAdversaryAttackFlowResponse, error)
	sessionGroupActionFlowFunc       func(context.Context, *pb.SessionGroupActionFlowRequest, ...grpc.CallOption) (*pb.SessionGroupActionFlowResponse, error)
	sessionTagTeamFlowFunc           func(context.Context, *pb.SessionTagTeamFlowRequest, ...grpc.CallOption) (*pb.SessionTagTeamFlowResponse, error)
	sessionReactionFlowFunc          func(context.Context, *pb.SessionReactionFlowRequest, ...grpc.CallOption) (*pb.SessionReactionFlowResponse, error)
}

func (s *daggerheartRuntimeClientStub) SessionActionRoll(ctx context.Context, req *pb.SessionActionRollRequest, opts ...grpc.CallOption) (*pb.SessionActionRollResponse, error) {
	return s.sessionActionRollFunc(ctx, req, opts...)
}
func (s *daggerheartRuntimeClientStub) ApplyRollOutcome(ctx context.Context, req *pb.ApplyRollOutcomeRequest, opts ...grpc.CallOption) (*pb.ApplyRollOutcomeResponse, error) {
	return s.applyRollOutcomeFunc(ctx, req, opts...)
}
func (s *daggerheartRuntimeClientStub) ApplyGmMove(ctx context.Context, req *pb.DaggerheartApplyGmMoveRequest, opts ...grpc.CallOption) (*pb.DaggerheartApplyGmMoveResponse, error) {
	return s.applyGmMoveFunc(ctx, req, opts...)
}
func (s *daggerheartRuntimeClientStub) CreateAdversary(ctx context.Context, req *pb.DaggerheartCreateAdversaryRequest, opts ...grpc.CallOption) (*pb.DaggerheartCreateAdversaryResponse, error) {
	return s.createAdversaryFunc(ctx, req, opts...)
}
func (s *daggerheartRuntimeClientStub) CreateSceneCountdown(ctx context.Context, req *pb.DaggerheartCreateSceneCountdownRequest, opts ...grpc.CallOption) (*pb.DaggerheartCreateSceneCountdownResponse, error) {
	return s.createSceneCountdownFunc(ctx, req, opts...)
}
func (s *daggerheartRuntimeClientStub) AdvanceSceneCountdown(ctx context.Context, req *pb.DaggerheartAdvanceSceneCountdownRequest, opts ...grpc.CallOption) (*pb.DaggerheartAdvanceSceneCountdownResponse, error) {
	return s.advanceSceneCountdownFunc(ctx, req, opts...)
}
func (s *daggerheartRuntimeClientStub) ResolveSceneCountdownTrigger(ctx context.Context, req *pb.DaggerheartResolveSceneCountdownTriggerRequest, opts ...grpc.CallOption) (*pb.DaggerheartResolveSceneCountdownTriggerResponse, error) {
	return s.resolveSceneCountdownTriggerFunc(ctx, req, opts...)
}
func (s *daggerheartRuntimeClientStub) UpdateAdversary(ctx context.Context, req *pb.DaggerheartUpdateAdversaryRequest, opts ...grpc.CallOption) (*pb.DaggerheartUpdateAdversaryResponse, error) {
	return s.updateAdversaryFunc(ctx, req, opts...)
}
func (s *daggerheartRuntimeClientStub) SessionAdversaryAttackFlow(ctx context.Context, req *pb.SessionAdversaryAttackFlowRequest, opts ...grpc.CallOption) (*pb.SessionAdversaryAttackFlowResponse, error) {
	return s.sessionAdversaryAttackFlowFunc(ctx, req, opts...)
}
func (s *daggerheartRuntimeClientStub) SessionGroupActionFlow(ctx context.Context, req *pb.SessionGroupActionFlowRequest, opts ...grpc.CallOption) (*pb.SessionGroupActionFlowResponse, error) {
	return s.sessionGroupActionFlowFunc(ctx, req, opts...)
}
func (s *daggerheartRuntimeClientStub) SessionTagTeamFlow(ctx context.Context, req *pb.SessionTagTeamFlowRequest, opts ...grpc.CallOption) (*pb.SessionTagTeamFlowResponse, error) {
	return s.sessionTagTeamFlowFunc(ctx, req, opts...)
}
func (s *daggerheartRuntimeClientStub) SessionReactionFlow(ctx context.Context, req *pb.SessionReactionFlowRequest, opts ...grpc.CallOption) (*pb.SessionReactionFlowResponse, error) {
	return s.sessionReactionFlowFunc(ctx, req, opts...)
}

func unmarshalToolResult(t *testing.T, result orchestration.ToolResult, target any) {
	t.Helper()
	if err := json.Unmarshal([]byte(result.Output), target); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
}

func testRuntime(stub *daggerheartRuntimeClientStub) daggerheartRuntimeTest {
	return daggerheartRuntimeTest{
		daggerheart:       stub,
		resolveCampaignID: func(string) string { return "camp-1" },
		resolveSessionID:  func(string) string { return "sess-1" },
		resolveSceneID:    func(context.Context, string, string) (string, error) { return "scene-1", nil },
	}
}

func sampleSceneCountdown() *pb.DaggerheartSceneCountdown {
	return &pb.DaggerheartSceneCountdown{
		CountdownId:       "cd-1",
		CampaignId:        "camp-1",
		SessionId:         "sess-1",
		SceneId:           "scene-1",
		Name:              "Breach",
		Tone:              pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_CONSEQUENCE,
		AdvancementPolicy: pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_ACTION_STANDARD,
		StartingValue:     4,
		RemainingValue:    2,
		LoopBehavior:      pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET,
		Status:            pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE,
		LinkedCountdownId: "cd-2",
	}
}

func sampleActionRoll() *pb.SessionActionRollResponse {
	return &pb.SessionActionRollResponse{
		RollSeq:    11,
		HopeDie:    6,
		FearDie:    4,
		Total:      14,
		Difficulty: 12,
		Success:    true,
		Flavor:     "with hope",
	}
}

func sampleRollOutcome() *pb.ApplyRollOutcomeResponse {
	return &pb.ApplyRollOutcomeResponse{
		RollSeq:              11,
		RequiresComplication: true,
	}
}

func TestActionRollResolve(t *testing.T) {
	t.Parallel()

	stub := &daggerheartRuntimeClientStub{
		sessionActionRollFunc: func(_ context.Context, req *pb.SessionActionRollRequest, _ ...grpc.CallOption) (*pb.SessionActionRollResponse, error) {
			if req.GetCampaignId() != "camp-1" || req.GetSessionId() != "sess-1" || req.GetSceneId() != "scene-1" {
				t.Fatalf("request scope = %#v", req)
			}
			if req.GetCharacterId() != "char-1" || req.GetTrait() != "Agility" || len(req.GetModifiers()) != 1 {
				t.Fatalf("request = %#v", req)
			}
			return sampleActionRoll(), nil
		},
		applyRollOutcomeFunc: func(_ context.Context, req *pb.ApplyRollOutcomeRequest, _ ...grpc.CallOption) (*pb.ApplyRollOutcomeResponse, error) {
			if req.GetRollSeq() != 11 || len(req.GetTargets()) != 1 || req.GetTargets()[0] != "target-1" || !req.GetSwapHopeFear() {
				t.Fatalf("apply outcome request = %#v", req)
			}
			return sampleRollOutcome(), nil
		},
	}

	result, err := ActionRollResolve(testRuntime(stub), context.Background(), []byte(`{
		"character_id":"char-1",
		"trait":"Agility",
		"difficulty":12,
		"modifiers":[{"source":"advantage","value":2}],
		"targets":["target-1"],
		"swap_hope_fear":true
	}`))
	if err != nil {
		t.Fatalf("ActionRollResolve() error = %v", err)
	}

	var decoded actionRollResolveResult
	unmarshalToolResult(t, result, &decoded)
	if decoded.ActionRoll.RollSeq != 11 || decoded.RollOutcome.RollSeq != 11 {
		t.Fatalf("decoded result = %#v", decoded)
	}
}

func TestMechanicsRuntimeHandlers(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 3, 27, 16, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		run  func(*testing.T, daggerheartRuntimeTest)
	}{
		{
			name: "gm move apply",
			run: func(t *testing.T, runtime daggerheartRuntimeTest) {
				stub := runtime.daggerheart.(*daggerheartRuntimeClientStub)
				stub.applyGmMoveFunc = func(_ context.Context, req *pb.DaggerheartApplyGmMoveRequest, _ ...grpc.CallOption) (*pb.DaggerheartApplyGmMoveResponse, error) {
					if req.GetSceneId() != "scene-1" || req.GetFearSpent() != 2 || req.GetDirectMove() == nil {
						t.Fatalf("request = %#v", req)
					}
					return &pb.DaggerheartApplyGmMoveResponse{GmFearBefore: 4, GmFearAfter: 2}, nil
				}
				result, err := GmMoveApply(runtime, context.Background(), []byte(`{"fear_spent":2,"direct_move":{"kind":"ADDITIONAL_MOVE","shape":"REVEAL_DANGER","description":"Show the danger"}}`))
				if err != nil {
					t.Fatalf("GmMoveApply() error = %v", err)
				}
				var decoded gmMoveApplyResult
				unmarshalToolResult(t, result, &decoded)
				if decoded.GMFearBefore != 4 || decoded.GMFearAfter != 2 {
					t.Fatalf("decoded = %#v", decoded)
				}
			},
		},
		{
			name: "adversary create",
			run: func(t *testing.T, runtime daggerheartRuntimeTest) {
				stub := runtime.daggerheart.(*daggerheartRuntimeClientStub)
				stub.createAdversaryFunc = func(_ context.Context, req *pb.DaggerheartCreateAdversaryRequest, _ ...grpc.CallOption) (*pb.DaggerheartCreateAdversaryResponse, error) {
					if req.GetAdversaryEntryId() != "entry.goblin" || req.GetNotes() != "On the parapet" {
						t.Fatalf("request = %#v", req)
					}
					return &pb.DaggerheartCreateAdversaryResponse{Adversary: &pb.DaggerheartAdversary{
						Id:               "adv-1",
						CampaignId:       "camp-1",
						AdversaryEntryId: "entry.goblin",
						Name:             "Goblin Archer",
						Kind:             "minion",
						SceneId:          "scene-1",
						SessionId:        "sess-1",
						CreatedAt:        timestamppb.New(now),
						UpdatedAt:        timestamppb.New(now),
					}}, nil
				}
				result, err := AdversaryCreate(runtime, context.Background(), []byte(`{"adversary_entry_id":"entry.goblin","notes":"On the parapet"}`))
				if err != nil {
					t.Fatalf("AdversaryCreate() error = %v", err)
				}
				var decoded adversarySummary
				unmarshalToolResult(t, result, &decoded)
				if decoded.ID != "adv-1" || decoded.Name != "Goblin Archer" {
					t.Fatalf("decoded = %#v", decoded)
				}
			},
		},
		{
			name: "countdown create",
			run: func(t *testing.T, runtime daggerheartRuntimeTest) {
				stub := runtime.daggerheart.(*daggerheartRuntimeClientStub)
				stub.createSceneCountdownFunc = func(_ context.Context, req *pb.DaggerheartCreateSceneCountdownRequest, _ ...grpc.CallOption) (*pb.DaggerheartCreateSceneCountdownResponse, error) {
					if req.GetName() != "Breach" || req.GetLoopBehavior() != pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_RESET {
						t.Fatalf("request = %#v", req)
					}
					if req.GetFixedStartingValue() != 4 {
						t.Fatalf("fixed starting value = %d, want 4", req.GetFixedStartingValue())
					}
					return &pb.DaggerheartCreateSceneCountdownResponse{Countdown: sampleSceneCountdown()}, nil
				}
				result, err := CountdownCreate(runtime, context.Background(), []byte(`{"name":"Breach","tone":"CONSEQUENCE","advancement_policy":"ACTION_STANDARD","fixed_starting_value":4,"loop_behavior":"RESET","linked_countdown_id":"cd-2"}`))
				if err != nil {
					t.Fatalf("CountdownCreate() error = %v", err)
				}
				var decoded countdownSummary
				unmarshalToolResult(t, result, &decoded)
				if decoded.ID != "cd-1" || decoded.Name != "Breach" || decoded.LoopBehavior != "RESET" {
					t.Fatalf("decoded = %#v", decoded)
				}
			},
		},
		{
			name: "countdown advance",
			run: func(t *testing.T, runtime daggerheartRuntimeTest) {
				stub := runtime.daggerheart.(*daggerheartRuntimeClientStub)
				stub.advanceSceneCountdownFunc = func(_ context.Context, req *pb.DaggerheartAdvanceSceneCountdownRequest, _ ...grpc.CallOption) (*pb.DaggerheartAdvanceSceneCountdownResponse, error) {
					if req.GetCountdownId() != "cd-1" || req.GetAmount() != 2 || req.GetReason() != "storm pressure" {
						t.Fatalf("request = %#v", req)
					}
					return &pb.DaggerheartAdvanceSceneCountdownResponse{
						Countdown: sampleSceneCountdown(),
						Advance: &pb.DaggerheartCountdownAdvance{
							RemainingBefore: 4,
							RemainingAfter:  2,
							AdvancedBy:      2,
							StatusBefore:    pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE,
							StatusAfter:     pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_TRIGGER_PENDING,
							Triggered:       true,
							Reason:          "storm pressure",
						},
					}, nil
				}
				result, err := CountdownAdvance(runtime, context.Background(), []byte(`{"countdown_id":"cd-1","amount":2,"reason":"storm pressure"}`))
				if err != nil {
					t.Fatalf("CountdownAdvance() error = %v", err)
				}
				var decoded countdownAdvanceResult
				unmarshalToolResult(t, result, &decoded)
				if decoded.Advance.AfterRemaining != 2 || !decoded.Advance.Triggered {
					t.Fatalf("decoded = %#v", decoded)
				}
			},
		},
		{
			name: "countdown resolve trigger",
			run: func(t *testing.T, runtime daggerheartRuntimeTest) {
				stub := runtime.daggerheart.(*daggerheartRuntimeClientStub)
				stub.resolveSceneCountdownTriggerFunc = func(_ context.Context, req *pb.DaggerheartResolveSceneCountdownTriggerRequest, _ ...grpc.CallOption) (*pb.DaggerheartResolveSceneCountdownTriggerResponse, error) {
					if req.GetCountdownId() != "cd-1" || req.GetReason() != "gate collapses" {
						t.Fatalf("request = %#v", req)
					}
					return &pb.DaggerheartResolveSceneCountdownTriggerResponse{Countdown: sampleSceneCountdown()}, nil
				}
				result, err := CountdownResolveTrigger(runtime, context.Background(), []byte(`{"countdown_id":"cd-1","reason":"gate collapses"}`))
				if err != nil {
					t.Fatalf("CountdownResolveTrigger() error = %v", err)
				}
				var decoded countdownSummary
				unmarshalToolResult(t, result, &decoded)
				if decoded.ID != "cd-1" {
					t.Fatalf("decoded = %#v", decoded)
				}
			},
		},
		{
			name: "adversary update",
			run: func(t *testing.T, runtime daggerheartRuntimeTest) {
				stub := runtime.daggerheart.(*daggerheartRuntimeClientStub)
				stub.updateAdversaryFunc = func(_ context.Context, req *pb.DaggerheartUpdateAdversaryRequest, _ ...grpc.CallOption) (*pb.DaggerheartUpdateAdversaryResponse, error) {
					if req.GetAdversaryId() != "adv-1" || req.GetNotes().GetValue() != "Pressed back" {
						t.Fatalf("request = %#v", req)
					}
					return &pb.DaggerheartUpdateAdversaryResponse{Adversary: &pb.DaggerheartAdversary{
						Id:               "adv-1",
						CampaignId:       "camp-1",
						AdversaryEntryId: "entry.goblin",
						Name:             "Goblin Archer",
						Notes:            "Pressed back",
						SceneId:          "scene-1",
						SessionId:        "sess-1",
						CreatedAt:        timestamppb.New(now),
						UpdatedAt:        timestamppb.New(now),
					}}, nil
				}
				result, err := AdversaryUpdate(runtime, context.Background(), []byte(`{"adversary_id":"adv-1","notes":"Pressed back"}`))
				if err != nil {
					t.Fatalf("AdversaryUpdate() error = %v", err)
				}
				var decoded adversarySummary
				unmarshalToolResult(t, result, &decoded)
				if decoded.Notes != "Pressed back" {
					t.Fatalf("decoded = %#v", decoded)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tt.run(t, testRuntime(&daggerheartRuntimeClientStub{}))
		})
	}
}

func TestCombatRuntimeHandlers(t *testing.T) {
	t.Parallel()

	stub := &daggerheartRuntimeClientStub{
		sessionAdversaryAttackFlowFunc: func(_ context.Context, req *pb.SessionAdversaryAttackFlowRequest, _ ...grpc.CallOption) (*pb.SessionAdversaryAttackFlowResponse, error) {
			if req.GetAdversaryId() != "adv-1" || req.GetTargetId() != "char-1" || len(req.GetTargetIds()) != 1 || req.GetTargetIds()[0] != "char-2" {
				t.Fatalf("request = %#v", req)
			}
			if req.GetTargetArmorReaction().GetTimeslowing() == nil {
				t.Fatalf("expected timeslowing armor reaction, got %#v", req.GetTargetArmorReaction())
			}
			return &pb.SessionAdversaryAttackFlowResponse{
				AttackRoll: &pb.SessionAdversaryAttackRollResponse{
					RollSeq: 21,
					Roll:    9,
					Total:   13,
					Rolls:   []int32{5, 4},
				},
				AttackOutcome: &pb.DaggerheartApplyAdversaryAttackOutcomeResponse{
					RollSeq:     21,
					AdversaryId: "adv-1",
					Targets:     []string{"char-1", "char-2"},
					Result: &pb.DaggerheartAdversaryAttackOutcomeResult{
						Success:    true,
						Crit:       false,
						Roll:       9,
						Total:      13,
						Difficulty: 12,
					},
				},
				DamageRoll: &pb.SessionDamageRollResponse{RollSeq: 22, Total: 7},
				DamageApplied: &pb.DaggerheartApplyDamageResponse{
					CharacterId: "char-1",
					State:       &pb.DaggerheartCharacterState{Hp: 4, Stress: 1, Armor: 2},
				},
				DamageApplications: []*pb.DaggerheartApplyDamageResponse{{
					CharacterId: "char-2",
					State:       &pb.DaggerheartCharacterState{Hp: 3, Stress: 2},
				}},
			}, nil
		},
		sessionGroupActionFlowFunc: func(_ context.Context, req *pb.SessionGroupActionFlowRequest, _ ...grpc.CallOption) (*pb.SessionGroupActionFlowResponse, error) {
			if req.GetLeaderCharacterId() != "char-lead" || len(req.GetSupporters()) != 1 || req.GetSupporters()[0].GetCharacterId() != "char-ally" {
				t.Fatalf("request = %#v", req)
			}
			return &pb.SessionGroupActionFlowResponse{
				LeaderRoll:       sampleActionRoll(),
				LeaderOutcome:    sampleRollOutcome(),
				SupporterRolls:   []*pb.GroupActionSupporterRoll{{CharacterId: "char-ally", ActionRoll: sampleActionRoll(), Success: true}},
				SupportModifier:  2,
				SupportSuccesses: 1,
				SupportFailures:  0,
			}, nil
		},
		sessionTagTeamFlowFunc: func(_ context.Context, req *pb.SessionTagTeamFlowRequest, _ ...grpc.CallOption) (*pb.SessionTagTeamFlowResponse, error) {
			if req.GetFirst().GetCharacterId() != "char-a" || req.GetSecond().GetCharacterId() != "char-b" || req.GetSelectedCharacterId() != "char-b" {
				t.Fatalf("request = %#v", req)
			}
			return &pb.SessionTagTeamFlowResponse{
				FirstRoll:           sampleActionRoll(),
				SecondRoll:          &pb.SessionActionRollResponse{RollSeq: 12, HopeDie: 5, FearDie: 2, Total: 11, Difficulty: 10, Success: true},
				SelectedOutcome:     sampleRollOutcome(),
				SelectedCharacterId: "char-b",
				SelectedRollSeq:     12,
			}, nil
		},
		sessionReactionFlowFunc: func(_ context.Context, req *pb.SessionReactionFlowRequest, _ ...grpc.CallOption) (*pb.SessionReactionFlowResponse, error) {
			if req.GetCharacterId() != "char-1" || req.GetTrait() != "Instinct" || !req.GetReplaceHopeWithArmor() {
				t.Fatalf("request = %#v", req)
			}
			return &pb.SessionReactionFlowResponse{
				ActionRoll:  sampleActionRoll(),
				RollOutcome: sampleRollOutcome(),
				ReactionOutcome: &pb.DaggerheartApplyReactionOutcomeResponse{
					RollSeq:     11,
					CharacterId: "char-1",
					Result:      &pb.DaggerheartReactionOutcomeResult{Outcome: pb.Outcome_SUCCESS_WITH_HOPE, Success: true},
				},
			}, nil
		},
	}

	runtime := testRuntime(stub)

	t.Run("adversary attack flow", func(t *testing.T) {
		t.Parallel()
		result, err := AdversaryAttackFlowResolve(runtime, context.Background(), []byte(`{
			"adversary_id":"adv-1",
			"target_id":"char-1",
			"target_ids":["char-2"],
			"difficulty":12,
			"damage":{"damage_type":"PHYSICAL","source":"Spear"},
			"target_armor_reaction":{"timeslowing":{}}
		}`))
		if err != nil {
			t.Fatalf("AdversaryAttackFlowResolve() error = %v", err)
		}
		var decoded adversaryAttackFlowResolveResult
		unmarshalToolResult(t, result, &decoded)
		if decoded.AttackRoll.RollSeq != 21 || len(decoded.DamageApplications) != 1 {
			t.Fatalf("decoded = %#v", decoded)
		}
	})

	t.Run("group action flow", func(t *testing.T) {
		t.Parallel()
		result, err := GroupActionFlowResolve(runtime, context.Background(), []byte(`{
			"leader_character_id":"char-lead",
			"leader_trait":"Presence",
			"difficulty":11,
			"supporters":[{"character_id":"char-ally","trait":"Agility"}]
		}`))
		if err != nil {
			t.Fatalf("GroupActionFlowResolve() error = %v", err)
		}
		var decoded groupActionFlowResolveResult
		unmarshalToolResult(t, result, &decoded)
		if decoded.SupportModifier != 2 || len(decoded.SupporterRolls) != 1 {
			t.Fatalf("decoded = %#v", decoded)
		}
	})

	t.Run("tag team flow", func(t *testing.T) {
		t.Parallel()
		result, err := TagTeamFlowResolve(runtime, context.Background(), []byte(`{
			"first":{"character_id":"char-a","trait":"Agility"},
			"second":{"character_id":"char-b","trait":"Strength"},
			"difficulty":10,
			"selected_character_id":"char-b"
		}`))
		if err != nil {
			t.Fatalf("TagTeamFlowResolve() error = %v", err)
		}
		var decoded tagTeamFlowResolveResult
		unmarshalToolResult(t, result, &decoded)
		if decoded.SelectedCharacterID != "char-b" || decoded.SelectedRollSeq != 12 {
			t.Fatalf("decoded = %#v", decoded)
		}
	})

	t.Run("reaction flow", func(t *testing.T) {
		t.Parallel()
		result, err := ReactionFlowResolve(runtime, context.Background(), []byte(`{
			"character_id":"char-1",
			"trait":"Instinct",
			"difficulty":10,
			"replace_hope_with_armor":true
		}`))
		if err != nil {
			t.Fatalf("ReactionFlowResolve() error = %v", err)
		}
		var decoded reactionFlowResolveResult
		unmarshalToolResult(t, result, &decoded)
		if decoded.ReactionOutcome == nil || decoded.ReactionOutcome.CharacterID != "char-1" {
			t.Fatalf("decoded = %#v", decoded)
		}
	})
}
