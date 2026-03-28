package daggerhearttools

import (
	"context"
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

type daggerheartEntryPointClientStub struct {
	pb.DaggerheartServiceClient
	dualityExplainFunc func(context.Context, *pb.DualityExplainRequest) (*pb.DualityExplainResponse, error)
	rollDiceFunc       func(context.Context, *pb.RollDiceRequest) (*pb.RollDiceResponse, error)
}

func (s *daggerheartEntryPointClientStub) DualityExplain(ctx context.Context, req *pb.DualityExplainRequest, _ ...grpc.CallOption) (*pb.DualityExplainResponse, error) {
	return s.dualityExplainFunc(ctx, req)
}

func (s *daggerheartEntryPointClientStub) RollDice(ctx context.Context, req *pb.RollDiceRequest, _ ...grpc.CallOption) (*pb.RollDiceResponse, error) {
	return s.rollDiceFunc(ctx, req)
}

func TestDiceEntryPoints(t *testing.T) {
	t.Parallel()

	actionDifficulty := int32(15)
	outcomeDifficulty := int32(10)
	explainDifficulty := int32(12)

	runtime := newDaggerheartReadTestRuntime(t, daggerheartReadTestServers{
		daggerheart: &daggerheartServiceServerStub{
			actionRollFunc: func(_ context.Context, req *pb.ActionRollRequest) (*pb.ActionRollResponse, error) {
				if req.GetModifier() != 2 || req.GetDifficulty() != 15 || req.GetRng().GetSeed() != 99 {
					t.Fatalf("action roll request = %#v", req)
				}
				return &pb.ActionRollResponse{
					Hope:            8,
					Fear:            3,
					Modifier:        2,
					Difficulty:      &actionDifficulty,
					Total:           13,
					IsCrit:          false,
					MeetsDifficulty: false,
					Outcome:         pb.Outcome_FAILURE_WITH_HOPE,
					Rng:             &commonv1.RngResponse{SeedUsed: 99},
				}, nil
			},
			dualityOutcomeFunc: func(_ context.Context, req *pb.DualityOutcomeRequest) (*pb.DualityOutcomeResponse, error) {
				if req.GetHope() != 6 || req.GetFear() != 2 || req.GetDifficulty() != 10 {
					t.Fatalf("duality outcome request = %#v", req)
				}
				return &pb.DualityOutcomeResponse{
					Hope:            6,
					Fear:            2,
					Modifier:        1,
					Difficulty:      &outcomeDifficulty,
					Total:           9,
					IsCrit:          false,
					MeetsDifficulty: false,
					Outcome:         pb.Outcome_FAILURE_WITH_HOPE,
				}, nil
			},
			dualityExplainFunc: func(_ context.Context, req *pb.DualityExplainRequest) (*pb.DualityExplainResponse, error) {
				if req.GetRequestId() != "req-1" || req.GetDifficulty() != 12 {
					t.Fatalf("duality explain request = %#v", req)
				}
				data, err := structpb.NewStruct(map[string]any{"delta": 2})
				if err != nil {
					t.Fatalf("structpb.NewStruct() error = %v", err)
				}
				return &pb.DualityExplainResponse{
					Hope:            7,
					Fear:            4,
					Modifier:        1,
					Difficulty:      &explainDifficulty,
					Total:           12,
					IsCrit:          false,
					MeetsDifficulty: true,
					Outcome:         pb.Outcome_SUCCESS_WITH_HOPE,
					RulesVersion:    "1.0.3",
					Intermediates: &pb.Intermediates{
						BaseTotal:       11,
						Total:           12,
						IsCrit:          false,
						MeetsDifficulty: true,
						HopeGtFear:      true,
					},
					Steps: []*pb.ExplainStep{{
						Code:    "modifier",
						Message: "Added modifier",
						Data:    data,
					}},
				}, nil
			},
			dualityProbabilityFunc: func(_ context.Context, req *pb.DualityProbabilityRequest) (*pb.DualityProbabilityResponse, error) {
				if req.GetModifier() != 2 || req.GetDifficulty() != 14 {
					t.Fatalf("duality probability request = %#v", req)
				}
				return &pb.DualityProbabilityResponse{
					TotalOutcomes: 36,
					CritCount:     6,
					SuccessCount:  20,
					FailureCount:  16,
					OutcomeCounts: []*pb.OutcomeCount{{
						Outcome: pb.Outcome_SUCCESS_WITH_HOPE,
						Count:   10,
					}},
				}, nil
			},
			rollDiceFunc: func(_ context.Context, req *pb.RollDiceRequest) (*pb.RollDiceResponse, error) {
				if len(req.GetDice()) != 2 || req.GetDice()[0].GetSides() != 6 || req.GetRng().GetSeed() != 123 {
					t.Fatalf("roll dice request = %#v", req)
				}
				return &pb.RollDiceResponse{
					Rolls: []*pb.DiceRoll{
						{Sides: 6, Results: []int32{2, 5}, Total: 7},
						{Sides: 8, Results: []int32{4}, Total: 4},
					},
					Total: 11,
					Rng:   &commonv1.RngResponse{SeedUsed: 123},
				}, nil
			},
			rulesVersionFunc: func(context.Context, *pb.RulesVersionRequest) (*pb.RulesVersionResponse, error) {
				return &pb.RulesVersionResponse{System: "daggerheart", Module: "core"}, nil
			},
		},
	})

	actionResult, err := DualityActionRoll(runtime, context.Background(), []byte(`{"modifier":2,"difficulty":15,"rng":{"seed":99}}`))
	if err != nil {
		t.Fatalf("DualityActionRoll() error = %v", err)
	}
	var action actionRollResult
	unmarshalToolResult(t, actionResult, &action)
	if action.Total != 13 || action.Outcome != pb.Outcome_FAILURE_WITH_HOPE.String() || action.Rng == nil || action.Rng.SeedUsed != 99 {
		t.Fatalf("action result = %#v", action)
	}

	outcomeResult, err := DualityOutcome(runtime, context.Background(), []byte(`{"hope":6,"fear":2,"modifier":1,"difficulty":10}`))
	if err != nil {
		t.Fatalf("DualityOutcome() error = %v", err)
	}
	var outcome actionRollResult
	unmarshalToolResult(t, outcomeResult, &outcome)
	if outcome.Total != 9 || outcome.Outcome != pb.Outcome_FAILURE_WITH_HOPE.String() {
		t.Fatalf("outcome result = %#v", outcome)
	}

	explainResult, err := DualityExplain(runtime, context.Background(), []byte(`{"hope":7,"fear":4,"modifier":1,"difficulty":12,"request_id":"req-1"}`))
	if err != nil {
		t.Fatalf("DualityExplain() error = %v", err)
	}
	var explain dualityExplainResult
	unmarshalToolResult(t, explainResult, &explain)
	if explain.RulesVersion != "1.0.3" || len(explain.Steps) != 1 || explain.Steps[0].Data["delta"] != float64(2) {
		t.Fatalf("explain result = %#v", explain)
	}

	probabilityResult, err := DualityProbability(runtime, context.Background(), []byte(`{"modifier":2,"difficulty":14}`))
	if err != nil {
		t.Fatalf("DualityProbability() error = %v", err)
	}
	var probability dualityProbabilityResult
	unmarshalToolResult(t, probabilityResult, &probability)
	if probability.TotalOutcomes != 36 || len(probability.OutcomeCounts) != 1 || probability.OutcomeCounts[0].Outcome != pb.Outcome_SUCCESS_WITH_HOPE.String() {
		t.Fatalf("probability result = %#v", probability)
	}

	rollResult, err := RollDice(runtime, context.Background(), []byte(`{"dice":[{"sides":6,"count":2},{"sides":8,"count":1}],"rng":{"seed":123}}`))
	if err != nil {
		t.Fatalf("RollDice() error = %v", err)
	}
	var rolled rollDiceResult
	unmarshalToolResult(t, rollResult, &rolled)
	if rolled.Total != 11 || len(rolled.Rolls) != 2 || rolled.Rng == nil || rolled.Rng.SeedUsed != 123 {
		t.Fatalf("roll result = %#v", rolled)
	}

	rulesResult, err := DualityRulesVersion(runtime, context.Background(), nil)
	if err != nil {
		t.Fatalf("DualityRulesVersion() error = %v", err)
	}
	var rules rulesVersionResult
	unmarshalToolResult(t, rulesResult, &rules)
	if rules.System != "daggerheart" || rules.Module != "core" {
		t.Fatalf("rules version = %#v", rules)
	}
}

func TestDiceEntryPointsValidateMissingResponses(t *testing.T) {
	t.Parallel()

	runtime := daggerheartRuntimeTest{
		daggerheart: &daggerheartEntryPointClientStub{
			dualityExplainFunc: func(context.Context, *pb.DualityExplainRequest) (*pb.DualityExplainResponse, error) {
				return &pb.DualityExplainResponse{}, nil
			},
			rollDiceFunc: func(context.Context, *pb.RollDiceRequest) (*pb.RollDiceResponse, error) {
				return nil, nil
			},
		},
	}

	if _, err := DualityExplain(runtime, context.Background(), []byte(`{"hope":4,"fear":3}`)); err == nil || err.Error() != "duality explain intermediates are missing" {
		t.Fatalf("DualityExplain() error = %v", err)
	}
	if _, err := RollDice(runtime, context.Background(), []byte(`{"dice":[{"sides":6,"count":1}]}`)); err == nil || err.Error() != "dice roll response is missing" {
		t.Fatalf("RollDice() error = %v", err)
	}
}

func TestCharacterSheetAndCombatBoardReadHandlers(t *testing.T) {
	t.Parallel()

	runtime := newDaggerheartReadTestRuntime(t, daggerheartReadTestServers{
		characters: &characterServiceServerStub{
			getCharacterSheetFunc: func(_ context.Context, req *statev1.GetCharacterSheetRequest) (*statev1.GetCharacterSheetResponse, error) {
				if req.GetCharacterId() != "char-1" {
					t.Fatalf("character sheet request = %#v", req)
				}
				return &statev1.GetCharacterSheetResponse{
					Character: &statev1.Character{Id: "char-1", CampaignId: "camp-1", Name: "Aria"},
					Profile: &statev1.CharacterProfile{
						SystemProfile: &statev1.CharacterProfile_Daggerheart{
							Daggerheart: &pb.DaggerheartProfile{
								ClassId:       "class.guardian",
								PrimaryWeapon: &pb.DaggerheartSheetWeaponSummary{Name: "Longsword", Trait: "Strength", Range: pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_MELEE.String(), DamageDice: "1d10"},
							},
						},
					},
				}, nil
			},
		},
		snapshots: &snapshotServiceServerStub{
			getSnapshotFunc: func(context.Context, *statev1.GetSnapshotRequest) (*statev1.GetSnapshotResponse, error) {
				return &statev1.GetSnapshotResponse{
					Snapshot: &statev1.Snapshot{
						SystemSnapshot: &statev1.Snapshot_Daggerheart{Daggerheart: &pb.DaggerheartSnapshot{GmFear: 3}},
					},
				}, nil
			},
		},
		sessions: &sessionServiceServerStub{
			getSessionSpotlightFunc: func(context.Context, *statev1.GetSessionSpotlightRequest) (*statev1.GetSessionSpotlightResponse, error) {
				return nil, status.Error(codes.NotFound, "missing")
			},
		},
		daggerheart: &daggerheartServiceServerStub{
			listAdversariesFunc: func(context.Context, *pb.DaggerheartListAdversariesRequest) (*pb.DaggerheartListAdversariesResponse, error) {
				return &pb.DaggerheartListAdversariesResponse{
					Adversaries: []*pb.DaggerheartAdversary{{Id: "adv-1", SceneId: "scene-1", Name: "Raider"}},
				}, nil
			},
			listSceneCountdownsFunc: func(context.Context, *pb.DaggerheartListSceneCountdownsRequest) (*pb.DaggerheartListSceneCountdownsResponse, error) {
				return &pb.DaggerheartListSceneCountdownsResponse{
					Countdowns: []*pb.DaggerheartSceneCountdown{{CountdownId: "cd-1", SessionId: "sess-1", SceneId: "scene-1", Name: "Collapse"}},
				}, nil
			},
		},
	})
	runtime.resolveCampaignID = func(string) string { return "camp-1" }
	runtime.resolveSessionID = func(string) string { return "sess-1" }
	runtime.resolveSceneID = func(context.Context, string, string) (string, error) { return "scene-1", nil }

	sheetResult, err := CharacterSheetRead(runtime, context.Background(), []byte(`{"character_id":"char-1"}`))
	if err != nil {
		t.Fatalf("CharacterSheetRead() error = %v", err)
	}
	var sheet characterSheetPayload
	unmarshalToolResult(t, sheetResult, &sheet)
	if sheet.Character.ID != "char-1" {
		t.Fatalf("character sheet = %#v", sheet)
	}

	boardResult, err := CombatBoardRead(runtime, context.Background(), nil)
	if err != nil {
		t.Fatalf("CombatBoardRead() error = %v", err)
	}
	var board daggerheartCombatBoardPayload
	unmarshalToolResult(t, boardResult, &board)
	if board.GmFear != 3 || len(board.Adversaries) != 1 || len(board.Countdowns) != 1 {
		t.Fatalf("combat board = %#v", board)
	}
}

func TestAttackFlowResolveUsesExplicitAndInferredProfiles(t *testing.T) {
	t.Parallel()

	t.Run("explicit target and standard attack", func(t *testing.T) {
		t.Parallel()

		runtime := newDaggerheartReadTestRuntime(t, daggerheartReadTestServers{
			snapshots: &snapshotServiceServerStub{
				getSnapshotFunc: func(context.Context, *statev1.GetSnapshotRequest) (*statev1.GetSnapshotResponse, error) {
					return &statev1.GetSnapshotResponse{Snapshot: &statev1.Snapshot{}}, nil
				},
			},
			sessions: &sessionServiceServerStub{
				getSessionSpotlightFunc: func(context.Context, *statev1.GetSessionSpotlightRequest) (*statev1.GetSessionSpotlightResponse, error) {
					return &statev1.GetSessionSpotlightResponse{}, nil
				},
			},
			daggerheart: &daggerheartServiceServerStub{
				listAdversariesFunc: func(context.Context, *pb.DaggerheartListAdversariesRequest) (*pb.DaggerheartListAdversariesResponse, error) {
					return &pb.DaggerheartListAdversariesResponse{}, nil
				},
				listSceneCountdownsFunc: func(context.Context, *pb.DaggerheartListSceneCountdownsRequest) (*pb.DaggerheartListSceneCountdownsResponse, error) {
					return &pb.DaggerheartListSceneCountdownsResponse{}, nil
				},
				sessionAttackFlowFunc: func(_ context.Context, req *pb.SessionAttackFlowRequest) (*pb.SessionAttackFlowResponse, error) {
					if req.GetTargetId() != "adv-explicit" || req.GetSceneId() != "scene-1" || req.GetCharacterId() != "char-1" {
						t.Fatalf("attack flow request = %#v", req)
					}
					attack := req.GetStandardAttack()
					if attack == nil || attack.GetTrait() != "Strength" || len(attack.GetDamageDice()) != 1 || attack.GetAttackRange() != pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_MELEE {
						t.Fatalf("standard attack = %#v", attack)
					}
					if req.GetDamage().GetDamageType() != pb.DaggerheartDamageType_DAGGERHEART_DAMAGE_TYPE_PHYSICAL || req.GetDamage().GetSource() != "Longsword" {
						t.Fatalf("damage spec = %#v", req.GetDamage())
					}
					return &pb.SessionAttackFlowResponse{
						ActionRoll:  &pb.SessionActionRollResponse{RollSeq: 7, HopeDie: 6, FearDie: 4, Total: 12, Difficulty: 11, Success: true, Flavor: "hope"},
						RollOutcome: &pb.ApplyRollOutcomeResponse{RollSeq: 7},
						AttackOutcome: &pb.DaggerheartApplyAttackOutcomeResponse{
							RollSeq:     7,
							CharacterId: "char-1",
							Targets:     []string{"adv-explicit"},
							Result:      &pb.DaggerheartAttackOutcomeResult{Outcome: pb.Outcome_SUCCESS_WITH_HOPE, Success: true, Flavor: "hope"},
						},
					}, nil
				},
			},
		})
		runtime.resolveCampaignID = func(string) string { return "camp-1" }
		runtime.resolveSessionID = func(string) string { return "sess-1" }
		runtime.resolveSceneID = func(context.Context, string, string) (string, error) { return "scene-1", nil }

		result, err := AttackFlowResolve(runtime, context.Background(), []byte(`{
			"character_id":"char-1",
			"difficulty":11,
			"target_id":"adv-explicit",
			"damage":{"damage_type":"physical","source":"Longsword"},
			"standard_attack":{"trait":"Strength","damage_dice":[{"count":1,"sides":10}],"attack_range":"melee"}
		}`))
		if err != nil {
			t.Fatalf("AttackFlowResolve() error = %v", err)
		}

		var decoded attackFlowResolveResult
		unmarshalToolResult(t, result, &decoded)
		if decoded.ActionRoll == nil || decoded.ActionRoll.RollSeq != 7 || decoded.AttackOutcome == nil || decoded.AttackOutcome.Targets[0] != "adv-explicit" {
			t.Fatalf("decoded = %#v", decoded)
		}
	})

	t.Run("infers target and attack profile from board and sheet", func(t *testing.T) {
		t.Parallel()

		runtime := newDaggerheartReadTestRuntime(t, daggerheartReadTestServers{
			characters: &characterServiceServerStub{
				getCharacterSheetFunc: func(context.Context, *statev1.GetCharacterSheetRequest) (*statev1.GetCharacterSheetResponse, error) {
					return &statev1.GetCharacterSheetResponse{
						Character: &statev1.Character{Id: "char-1", CampaignId: "camp-1", Name: "Aria"},
						Profile: &statev1.CharacterProfile{
							SystemProfile: &statev1.CharacterProfile_Daggerheart{
								Daggerheart: &pb.DaggerheartProfile{
									PrimaryWeapon: &pb.DaggerheartSheetWeaponSummary{
										Name:       "Warhammer",
										Trait:      "Strength",
										Range:      pb.DaggerheartAttackRange_DAGGERHEART_ATTACK_RANGE_MELEE.String(),
										DamageDice: "2d8",
										DamageType: "physical",
									},
								},
							},
						},
					}, nil
				},
			},
			snapshots: &snapshotServiceServerStub{
				getSnapshotFunc: func(context.Context, *statev1.GetSnapshotRequest) (*statev1.GetSnapshotResponse, error) {
					return &statev1.GetSnapshotResponse{Snapshot: &statev1.Snapshot{}}, nil
				},
			},
			sessions: &sessionServiceServerStub{
				getSessionSpotlightFunc: func(context.Context, *statev1.GetSessionSpotlightRequest) (*statev1.GetSessionSpotlightResponse, error) {
					return &statev1.GetSessionSpotlightResponse{}, nil
				},
			},
			daggerheart: &daggerheartServiceServerStub{
				listAdversariesFunc: func(context.Context, *pb.DaggerheartListAdversariesRequest) (*pb.DaggerheartListAdversariesResponse, error) {
					return &pb.DaggerheartListAdversariesResponse{
						Adversaries: []*pb.DaggerheartAdversary{{Id: "adv-1", SceneId: "scene-1", Name: "Ogre"}},
					}, nil
				},
				listSceneCountdownsFunc: func(context.Context, *pb.DaggerheartListSceneCountdownsRequest) (*pb.DaggerheartListSceneCountdownsResponse, error) {
					return &pb.DaggerheartListSceneCountdownsResponse{}, nil
				},
				sessionAttackFlowFunc: func(_ context.Context, req *pb.SessionAttackFlowRequest) (*pb.SessionAttackFlowResponse, error) {
					if req.GetTargetId() != "adv-1" || req.GetDamage().GetSource() != "Warhammer" || len(req.GetStandardAttack().GetDamageDice()) != 1 || req.GetStandardAttack().GetDamageDice()[0].GetCount() != 2 {
						t.Fatalf("inferred attack request = %#v", req)
					}
					return &pb.SessionAttackFlowResponse{ActionRoll: &pb.SessionActionRollResponse{RollSeq: 9, Total: 15, Success: true}}, nil
				},
			},
		})
		runtime.resolveCampaignID = func(string) string { return "camp-1" }
		runtime.resolveSessionID = func(string) string { return "sess-1" }
		runtime.resolveSceneID = func(context.Context, string, string) (string, error) { return "scene-1", nil }

		result, err := AttackFlowResolve(runtime, context.Background(), []byte(`{"character_id":"char-1","difficulty":13}`))
		if err != nil {
			t.Fatalf("AttackFlowResolve() error = %v", err)
		}

		var decoded attackFlowResolveResult
		unmarshalToolResult(t, result, &decoded)
		if decoded.ActionRoll == nil || decoded.ActionRoll.RollSeq != 9 {
			t.Fatalf("decoded = %#v", decoded)
		}
	})
}

func TestAttackFlowResolveRejectsEmptyBoardAndMissingInferredProfile(t *testing.T) {
	t.Parallel()

	t.Run("empty board requires explicit target", func(t *testing.T) {
		t.Parallel()

		runtime := newDaggerheartReadTestRuntime(t, daggerheartReadTestServers{
			snapshots: &snapshotServiceServerStub{
				getSnapshotFunc: func(context.Context, *statev1.GetSnapshotRequest) (*statev1.GetSnapshotResponse, error) {
					return &statev1.GetSnapshotResponse{Snapshot: &statev1.Snapshot{}}, nil
				},
			},
			sessions: &sessionServiceServerStub{
				getSessionSpotlightFunc: func(context.Context, *statev1.GetSessionSpotlightRequest) (*statev1.GetSessionSpotlightResponse, error) {
					return &statev1.GetSessionSpotlightResponse{}, nil
				},
			},
			daggerheart: &daggerheartServiceServerStub{
				listAdversariesFunc: func(context.Context, *pb.DaggerheartListAdversariesRequest) (*pb.DaggerheartListAdversariesResponse, error) {
					return &pb.DaggerheartListAdversariesResponse{}, nil
				},
				listSceneCountdownsFunc: func(context.Context, *pb.DaggerheartListSceneCountdownsRequest) (*pb.DaggerheartListSceneCountdownsResponse, error) {
					return &pb.DaggerheartListSceneCountdownsResponse{}, nil
				},
			},
		})
		runtime.resolveCampaignID = func(string) string { return "camp-1" }
		runtime.resolveSessionID = func(string) string { return "sess-1" }
		runtime.resolveSceneID = func(context.Context, string, string) (string, error) { return "scene-1", nil }

		_, err := AttackFlowResolve(runtime, context.Background(), []byte(`{"character_id":"char-1","difficulty":13}`))
		if err == nil || !strings.Contains(err.Error(), "combat board is empty") {
			t.Fatalf("AttackFlowResolve() error = %v", err)
		}
	})

	t.Run("missing inferred attack profile requires explicit attack", func(t *testing.T) {
		t.Parallel()

		runtime := newDaggerheartReadTestRuntime(t, daggerheartReadTestServers{
			characters: &characterServiceServerStub{
				getCharacterSheetFunc: func(context.Context, *statev1.GetCharacterSheetRequest) (*statev1.GetCharacterSheetResponse, error) {
					return &statev1.GetCharacterSheetResponse{
						Character: &statev1.Character{Id: "char-1", CampaignId: "camp-1", Name: "Aria"},
						Profile:   &statev1.CharacterProfile{},
					}, nil
				},
			},
			snapshots: &snapshotServiceServerStub{
				getSnapshotFunc: func(context.Context, *statev1.GetSnapshotRequest) (*statev1.GetSnapshotResponse, error) {
					return &statev1.GetSnapshotResponse{Snapshot: &statev1.Snapshot{}}, nil
				},
			},
			sessions: &sessionServiceServerStub{
				getSessionSpotlightFunc: func(context.Context, *statev1.GetSessionSpotlightRequest) (*statev1.GetSessionSpotlightResponse, error) {
					return &statev1.GetSessionSpotlightResponse{}, nil
				},
			},
			daggerheart: &daggerheartServiceServerStub{
				listAdversariesFunc: func(context.Context, *pb.DaggerheartListAdversariesRequest) (*pb.DaggerheartListAdversariesResponse, error) {
					return &pb.DaggerheartListAdversariesResponse{
						Adversaries: []*pb.DaggerheartAdversary{{Id: "adv-1", SceneId: "scene-1"}},
					}, nil
				},
				listSceneCountdownsFunc: func(context.Context, *pb.DaggerheartListSceneCountdownsRequest) (*pb.DaggerheartListSceneCountdownsResponse, error) {
					return &pb.DaggerheartListSceneCountdownsResponse{}, nil
				},
			},
		})
		runtime.resolveCampaignID = func(string) string { return "camp-1" }
		runtime.resolveSessionID = func(string) string { return "sess-1" }
		runtime.resolveSceneID = func(context.Context, string, string) (string, error) { return "scene-1", nil }

		_, err := AttackFlowResolve(runtime, context.Background(), []byte(`{"character_id":"char-1","difficulty":13}`))
		if err == nil || !strings.Contains(err.Error(), "cannot infer an attack profile") {
			t.Fatalf("AttackFlowResolve() error = %v", err)
		}
	})
}
