package daggerhearttools

import (
	"context"
	"encoding/json"
	"errors"
	"net"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"
	"google.golang.org/protobuf/types/known/timestamppb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

type daggerheartReadTestRuntime struct {
	characters  statev1.CharacterServiceClient
	sessions    statev1.SessionServiceClient
	snapshots   statev1.SnapshotServiceClient
	daggerheart pb.DaggerheartServiceClient

	resolveCampaignID func(string) string
	resolveSessionID  func(string) string
	resolveSceneID    func(context.Context, string, string) (string, error)
}

func (r daggerheartReadTestRuntime) CharacterClient() statev1.CharacterServiceClient {
	return r.characters
}

func (r daggerheartReadTestRuntime) SessionClient() statev1.SessionServiceClient { return r.sessions }

func (r daggerheartReadTestRuntime) SnapshotClient() statev1.SnapshotServiceClient {
	return r.snapshots
}

func (r daggerheartReadTestRuntime) DaggerheartClient() pb.DaggerheartServiceClient {
	return r.daggerheart
}

func (daggerheartReadTestRuntime) CallContext(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithCancel(ctx)
}

func (r daggerheartReadTestRuntime) ResolveCampaignID(explicit string) string {
	if r.resolveCampaignID != nil {
		return r.resolveCampaignID(explicit)
	}
	return explicit
}

func (r daggerheartReadTestRuntime) ResolveSessionID(explicit string) string {
	if r.resolveSessionID != nil {
		return r.resolveSessionID(explicit)
	}
	return explicit
}

func (r daggerheartReadTestRuntime) ResolveSceneID(ctx context.Context, campaignID, explicit string) (string, error) {
	if r.resolveSceneID != nil {
		return r.resolveSceneID(ctx, campaignID, explicit)
	}
	return explicit, nil
}

type daggerheartReadTestServers struct {
	characters  *characterServiceServerStub
	sessions    *sessionServiceServerStub
	snapshots   *snapshotServiceServerStub
	daggerheart *daggerheartServiceServerStub
}

func newDaggerheartReadTestRuntime(t *testing.T, servers daggerheartReadTestServers) daggerheartReadTestRuntime {
	t.Helper()

	listener := bufconn.Listen(1024 * 1024)
	server := grpc.NewServer()
	if servers.characters != nil {
		statev1.RegisterCharacterServiceServer(server, servers.characters)
	}
	if servers.sessions != nil {
		statev1.RegisterSessionServiceServer(server, servers.sessions)
	}
	if servers.snapshots != nil {
		statev1.RegisterSnapshotServiceServer(server, servers.snapshots)
	}
	if servers.daggerheart != nil {
		pb.RegisterDaggerheartServiceServer(server, servers.daggerheart)
	}
	go func() {
		_ = server.Serve(listener)
	}()
	t.Cleanup(func() {
		server.Stop()
		_ = listener.Close()
	})

	conn, err := grpc.DialContext(
		context.Background(),
		"bufnet",
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) {
			return listener.Dial()
		}),
	)
	if err != nil {
		t.Fatalf("grpc.DialContext() error = %v", err)
	}
	t.Cleanup(func() {
		_ = conn.Close()
	})

	return daggerheartReadTestRuntime{
		characters:        statev1.NewCharacterServiceClient(conn),
		sessions:          statev1.NewSessionServiceClient(conn),
		snapshots:         statev1.NewSnapshotServiceClient(conn),
		daggerheart:       pb.NewDaggerheartServiceClient(conn),
		resolveCampaignID: func(explicit string) string { return explicit },
		resolveSessionID:  func(explicit string) string { return explicit },
		resolveSceneID:    func(context.Context, string, string) (string, error) { return "", nil },
	}
}

type characterServiceServerStub struct {
	statev1.UnimplementedCharacterServiceServer
	getCharacterSheetFunc func(context.Context, *statev1.GetCharacterSheetRequest) (*statev1.GetCharacterSheetResponse, error)
}

func (s *characterServiceServerStub) GetCharacterSheet(ctx context.Context, req *statev1.GetCharacterSheetRequest) (*statev1.GetCharacterSheetResponse, error) {
	if s.getCharacterSheetFunc == nil {
		return &statev1.GetCharacterSheetResponse{}, nil
	}
	return s.getCharacterSheetFunc(ctx, req)
}

type snapshotServiceServerStub struct {
	statev1.UnimplementedSnapshotServiceServer
	getSnapshotFunc func(context.Context, *statev1.GetSnapshotRequest) (*statev1.GetSnapshotResponse, error)
}

func (s *snapshotServiceServerStub) GetSnapshot(ctx context.Context, req *statev1.GetSnapshotRequest) (*statev1.GetSnapshotResponse, error) {
	if s.getSnapshotFunc == nil {
		return &statev1.GetSnapshotResponse{}, nil
	}
	return s.getSnapshotFunc(ctx, req)
}

type sessionServiceServerStub struct {
	statev1.UnimplementedSessionServiceServer
	getSessionSpotlightFunc func(context.Context, *statev1.GetSessionSpotlightRequest) (*statev1.GetSessionSpotlightResponse, error)
}

func (s *sessionServiceServerStub) GetSessionSpotlight(ctx context.Context, req *statev1.GetSessionSpotlightRequest) (*statev1.GetSessionSpotlightResponse, error) {
	if s.getSessionSpotlightFunc == nil {
		return &statev1.GetSessionSpotlightResponse{}, nil
	}
	return s.getSessionSpotlightFunc(ctx, req)
}

type daggerheartServiceServerStub struct {
	pb.UnimplementedDaggerheartServiceServer
	actionRollFunc             func(context.Context, *pb.ActionRollRequest) (*pb.ActionRollResponse, error)
	dualityOutcomeFunc         func(context.Context, *pb.DualityOutcomeRequest) (*pb.DualityOutcomeResponse, error)
	dualityExplainFunc         func(context.Context, *pb.DualityExplainRequest) (*pb.DualityExplainResponse, error)
	dualityProbabilityFunc     func(context.Context, *pb.DualityProbabilityRequest) (*pb.DualityProbabilityResponse, error)
	rollDiceFunc               func(context.Context, *pb.RollDiceRequest) (*pb.RollDiceResponse, error)
	rulesVersionFunc           func(context.Context, *pb.RulesVersionRequest) (*pb.RulesVersionResponse, error)
	listCampaignCountdownsFunc func(context.Context, *pb.DaggerheartListCampaignCountdownsRequest) (*pb.DaggerheartListCampaignCountdownsResponse, error)
	listAdversariesFunc        func(context.Context, *pb.DaggerheartListAdversariesRequest) (*pb.DaggerheartListAdversariesResponse, error)
	listSceneCountdownsFunc    func(context.Context, *pb.DaggerheartListSceneCountdownsRequest) (*pb.DaggerheartListSceneCountdownsResponse, error)
	sessionAttackFlowFunc      func(context.Context, *pb.SessionAttackFlowRequest) (*pb.SessionAttackFlowResponse, error)
}

func (s *daggerheartServiceServerStub) ActionRoll(ctx context.Context, req *pb.ActionRollRequest) (*pb.ActionRollResponse, error) {
	if s.actionRollFunc == nil {
		return &pb.ActionRollResponse{}, nil
	}
	return s.actionRollFunc(ctx, req)
}

func (s *daggerheartServiceServerStub) DualityOutcome(ctx context.Context, req *pb.DualityOutcomeRequest) (*pb.DualityOutcomeResponse, error) {
	if s.dualityOutcomeFunc == nil {
		return &pb.DualityOutcomeResponse{}, nil
	}
	return s.dualityOutcomeFunc(ctx, req)
}

func (s *daggerheartServiceServerStub) DualityExplain(ctx context.Context, req *pb.DualityExplainRequest) (*pb.DualityExplainResponse, error) {
	if s.dualityExplainFunc == nil {
		return &pb.DualityExplainResponse{}, nil
	}
	return s.dualityExplainFunc(ctx, req)
}

func (s *daggerheartServiceServerStub) DualityProbability(ctx context.Context, req *pb.DualityProbabilityRequest) (*pb.DualityProbabilityResponse, error) {
	if s.dualityProbabilityFunc == nil {
		return &pb.DualityProbabilityResponse{}, nil
	}
	return s.dualityProbabilityFunc(ctx, req)
}

func (s *daggerheartServiceServerStub) RollDice(ctx context.Context, req *pb.RollDiceRequest) (*pb.RollDiceResponse, error) {
	if s.rollDiceFunc == nil {
		return &pb.RollDiceResponse{}, nil
	}
	return s.rollDiceFunc(ctx, req)
}

func (s *daggerheartServiceServerStub) RulesVersion(ctx context.Context, req *pb.RulesVersionRequest) (*pb.RulesVersionResponse, error) {
	if s.rulesVersionFunc == nil {
		return &pb.RulesVersionResponse{}, nil
	}
	return s.rulesVersionFunc(ctx, req)
}

func (s *daggerheartServiceServerStub) ListCampaignCountdowns(ctx context.Context, req *pb.DaggerheartListCampaignCountdownsRequest) (*pb.DaggerheartListCampaignCountdownsResponse, error) {
	if s.listCampaignCountdownsFunc == nil {
		return &pb.DaggerheartListCampaignCountdownsResponse{}, nil
	}
	return s.listCampaignCountdownsFunc(ctx, req)
}

func (s *daggerheartServiceServerStub) ListAdversaries(ctx context.Context, req *pb.DaggerheartListAdversariesRequest) (*pb.DaggerheartListAdversariesResponse, error) {
	if s.listAdversariesFunc == nil {
		return &pb.DaggerheartListAdversariesResponse{}, nil
	}
	return s.listAdversariesFunc(ctx, req)
}

func (s *daggerheartServiceServerStub) ListSceneCountdowns(ctx context.Context, req *pb.DaggerheartListSceneCountdownsRequest) (*pb.DaggerheartListSceneCountdownsResponse, error) {
	if s.listSceneCountdownsFunc == nil {
		return &pb.DaggerheartListSceneCountdownsResponse{}, nil
	}
	return s.listSceneCountdownsFunc(ctx, req)
}

func (s *daggerheartServiceServerStub) SessionAttackFlow(ctx context.Context, req *pb.SessionAttackFlowRequest) (*pb.SessionAttackFlowResponse, error) {
	if s.sessionAttackFlowFunc == nil {
		return &pb.SessionAttackFlowResponse{}, nil
	}
	return s.sessionAttackFlowFunc(ctx, req)
}

func TestReadRulesVersionResource(t *testing.T) {
	t.Parallel()

	runtime := newDaggerheartReadTestRuntime(t, daggerheartReadTestServers{
		daggerheart: &daggerheartServiceServerStub{
			rulesVersionFunc: func(_ context.Context, req *pb.RulesVersionRequest) (*pb.RulesVersionResponse, error) {
				if req == nil {
					t.Fatal("RulesVersion request = nil")
				}
				return &pb.RulesVersionResponse{
					System:         "daggerheart",
					Module:         "core",
					RulesVersion:   "1.0.2",
					DiceModel:      "duality",
					TotalFormula:   "trait+modifier",
					CritRule:       "double_damage",
					DifficultyRule: "meet_or_beat",
					Outcomes: []pb.Outcome{
						pb.Outcome_SUCCESS_WITH_HOPE,
						pb.Outcome_FAILURE_WITH_FEAR,
					},
				}, nil
			},
		},
	})

	value, err := ReadRulesVersionResource(runtime, context.Background())
	if err != nil {
		t.Fatalf("ReadRulesVersionResource() error = %v", err)
	}

	var payload rulesVersionResult
	unmarshalJSON(t, value, &payload)
	if payload.Module != "core" || len(payload.Outcomes) != 2 || payload.Outcomes[0] != pb.Outcome_SUCCESS_WITH_HOPE.String() {
		t.Fatalf("rules version payload = %#v", payload)
	}
}

func TestReadSnapshotResource(t *testing.T) {
	t.Parallel()

	runtime := newDaggerheartReadTestRuntime(t, daggerheartReadTestServers{
		snapshots: &snapshotServiceServerStub{
			getSnapshotFunc: func(_ context.Context, req *statev1.GetSnapshotRequest) (*statev1.GetSnapshotResponse, error) {
				if req.GetCampaignId() != "camp-1" {
					t.Fatalf("campaign_id = %q, want camp-1", req.GetCampaignId())
				}
				return &statev1.GetSnapshotResponse{
					Snapshot: &statev1.Snapshot{
						SystemSnapshot: &statev1.Snapshot_Daggerheart{
							Daggerheart: &pb.DaggerheartSnapshot{
								GmFear:                4,
								ConsecutiveShortRests: 2,
							},
						},
						CharacterStates: []*statev1.CharacterState{
							{
								CharacterId: "char-1",
								SystemState: &statev1.CharacterState_Daggerheart{
									Daggerheart: &pb.DaggerheartCharacterState{
										Hp:        7,
										Hope:      3,
										HopeMax:   6,
										Stress:    1,
										Armor:     2,
										LifeState: pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE,
										ConditionStates: []*pb.DaggerheartConditionState{{
											Label: "Vulnerable",
											ClearTriggers: []pb.DaggerheartConditionClearTrigger{
												pb.DaggerheartConditionClearTrigger_DAGGERHEART_CONDITION_CLEAR_TRIGGER_LONG_REST,
											},
										}},
										TemporaryArmorBuckets: []*pb.DaggerheartTemporaryArmorBucket{{
											Source: "ward",
											Amount: 2,
										}},
										StatModifiers: []*pb.DaggerheartStatModifier{{
											Target: "agility",
											Delta:  1,
											Label:  "blessing",
										}},
									},
								},
							},
							{CharacterId: "char-no-system"},
						},
					},
				}, nil
			},
		},
	})

	value, err := ReadSnapshotResource(runtime, context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("ReadSnapshotResource() error = %v", err)
	}

	var payload snapshotPayload
	unmarshalJSON(t, value, &payload)
	if payload.GmFear != 4 || payload.ConsecutiveShortRests != 2 {
		t.Fatalf("snapshot payload header = %#v", payload)
	}
	if len(payload.Characters) != 1 {
		t.Fatalf("len(characters) = %d, want 1", len(payload.Characters))
	}
	if got := payload.Characters[0]; got.CharacterID != "char-1" || got.LifeState != "ALIVE" || len(got.Conditions) != 1 || len(got.TemporaryArmor) != 1 || len(got.StatModifiers) != 1 {
		t.Fatalf("character entry = %#v", got)
	}
}

func TestReadCampaignCountdownsResource(t *testing.T) {
	t.Parallel()

	runtime := newDaggerheartReadTestRuntime(t, daggerheartReadTestServers{
		daggerheart: &daggerheartServiceServerStub{
			listCampaignCountdownsFunc: func(_ context.Context, req *pb.DaggerheartListCampaignCountdownsRequest) (*pb.DaggerheartListCampaignCountdownsResponse, error) {
				if req.GetCampaignId() != "camp-1" {
					t.Fatalf("campaign_id = %q, want camp-1", req.GetCampaignId())
				}
				return &pb.DaggerheartListCampaignCountdownsResponse{
					Countdowns: []*pb.DaggerheartCampaignCountdown{{
						CountdownId:       "cd-1",
						Name:              "Storm Front",
						Tone:              pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_PROGRESS,
						AdvancementPolicy: pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL,
						StartingValue:     6,
						RemainingValue:    3,
						LoopBehavior:      pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE,
						Status:            pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE,
					}},
				}, nil
			},
		},
	})

	value, err := ReadCampaignCountdownsResource(runtime, context.Background(), "camp-1")
	if err != nil {
		t.Fatalf("ReadCampaignCountdownsResource() error = %v", err)
	}

	var payload daggerheartCampaignCountdownsPayload
	unmarshalJSON(t, value, &payload)
	if len(payload.Countdowns) != 1 || payload.Countdowns[0].ID != "cd-1" || payload.Countdowns[0].Tone != "PROGRESS" {
		t.Fatalf("campaign countdown payload = %#v", payload)
	}
}

func TestReadCharacterSheetResource(t *testing.T) {
	t.Parallel()

	runtime := newDaggerheartReadTestRuntime(t, daggerheartReadTestServers{
		characters: &characterServiceServerStub{
			getCharacterSheetFunc: func(_ context.Context, req *statev1.GetCharacterSheetRequest) (*statev1.GetCharacterSheetResponse, error) {
				if req.GetCampaignId() != "camp-1" || req.GetCharacterId() != "char-1" {
					t.Fatalf("request = %#v", req)
				}
				return &statev1.GetCharacterSheetResponse{
					Character: &statev1.Character{
						Id:                 "char-1",
						CampaignId:         "camp-1",
						Name:               "Aria",
						Kind:               statev1.CharacterKind_PC,
						OwnerParticipantId: wrapperspb.String("part-1"),
						Aliases:            []string{"Lantern"},
					},
					Profile: &statev1.CharacterProfile{
						SystemProfile: &statev1.CharacterProfile_Daggerheart{
							Daggerheart: &pb.DaggerheartProfile{
								Level:         2,
								ClassId:       "class.guardian",
								SubclassId:    "subclass.stalwart",
								HpMax:         12,
								StressMax:     wrapperspb.Int32(6),
								ArmorMax:      wrapperspb.Int32(3),
								PrimaryWeapon: &pb.DaggerheartSheetWeaponSummary{Id: "weapon.longsword", Name: "Longsword", Trait: "Strength", DamageDice: "1d10"},
								ActiveArmor:   &pb.DaggerheartSheetArmorSummary{Id: "armor.gambeson", Name: "Gambeson", BaseScore: 2},
							},
						},
					},
					State: &statev1.CharacterState{
						SystemState: &statev1.CharacterState_Daggerheart{
							Daggerheart: &pb.DaggerheartCharacterState{
								Hp:        9,
								Hope:      4,
								HopeMax:   6,
								Stress:    1,
								Armor:     2,
								LifeState: pb.DaggerheartLifeState_DAGGERHEART_LIFE_STATE_ALIVE,
							},
						},
					},
				}, nil
			},
		},
	})

	value, err := ReadCharacterSheetResource(runtime, context.Background(), "camp-1", " char-1 ")
	if err != nil {
		t.Fatalf("ReadCharacterSheetResource() error = %v", err)
	}

	var payload characterSheetPayload
	unmarshalJSON(t, value, &payload)
	if payload.Character.Name != "Aria" {
		t.Fatalf("character = %#v", payload.Character)
	}
	if payload.Daggerheart == nil || payload.Daggerheart.Class == nil || payload.Daggerheart.Class.Name != "Guardian" {
		t.Fatalf("daggerheart payload = %#v", payload.Daggerheart)
	}
	if payload.Daggerheart.Equipment == nil || payload.Daggerheart.Equipment.PrimaryWeapon == nil || payload.Daggerheart.Equipment.PrimaryWeapon.Name != "Longsword" {
		t.Fatalf("equipment = %#v", payload.Daggerheart.Equipment)
	}
}

func TestReadCombatBoardResource(t *testing.T) {
	t.Parallel()

	t.Run("ready board includes spotlight countdowns and scene adversaries", func(t *testing.T) {
		t.Parallel()

		updatedAt := time.Date(2026, time.March, 27, 15, 4, 5, 0, time.UTC)
		runtime := newDaggerheartReadTestRuntime(t, daggerheartReadTestServers{
			snapshots: &snapshotServiceServerStub{
				getSnapshotFunc: func(_ context.Context, req *statev1.GetSnapshotRequest) (*statev1.GetSnapshotResponse, error) {
					if req.GetCampaignId() != "camp-1" {
						t.Fatalf("campaign_id = %q, want camp-1", req.GetCampaignId())
					}
					return &statev1.GetSnapshotResponse{
						Snapshot: &statev1.Snapshot{
							SystemSnapshot: &statev1.Snapshot_Daggerheart{
								Daggerheart: &pb.DaggerheartSnapshot{GmFear: 5},
							},
						},
					}, nil
				},
			},
			sessions: &sessionServiceServerStub{
				getSessionSpotlightFunc: func(_ context.Context, req *statev1.GetSessionSpotlightRequest) (*statev1.GetSessionSpotlightResponse, error) {
					if req.GetSessionId() != "sess-1" {
						t.Fatalf("session_id = %q, want sess-1", req.GetSessionId())
					}
					return &statev1.GetSessionSpotlightResponse{
						Spotlight: &statev1.SessionSpotlight{
							Type:        statev1.SessionSpotlightType_SESSION_SPOTLIGHT_TYPE_CHARACTER,
							CharacterId: "char-1",
							UpdatedAt:   timestamppb.New(updatedAt),
						},
					}, nil
				},
			},
			daggerheart: &daggerheartServiceServerStub{
				listAdversariesFunc: func(_ context.Context, req *pb.DaggerheartListAdversariesRequest) (*pb.DaggerheartListAdversariesResponse, error) {
					if req.GetCampaignId() != "camp-1" || req.GetSessionId().GetValue() != "sess-1" {
						t.Fatalf("ListAdversaries request = %#v", req)
					}
					return &pb.DaggerheartListAdversariesResponse{
						Adversaries: []*pb.DaggerheartAdversary{
							{Id: "adv-1", Name: "Bandit", SceneId: "scene-1", Hp: 5},
							{Id: "adv-2", Name: "Offscene", SceneId: "scene-2", Hp: 4},
						},
					}, nil
				},
				listSceneCountdownsFunc: func(_ context.Context, req *pb.DaggerheartListSceneCountdownsRequest) (*pb.DaggerheartListSceneCountdownsResponse, error) {
					if req.GetSceneId() != "scene-1" {
						t.Fatalf("scene_id = %q, want scene-1", req.GetSceneId())
					}
					return &pb.DaggerheartListSceneCountdownsResponse{
						Countdowns: []*pb.DaggerheartSceneCountdown{{
							CountdownId:       "cd-1",
							SessionId:         "sess-1",
							SceneId:           "scene-1",
							Name:              "Collapse",
							Tone:              pb.DaggerheartCountdownTone_DAGGERHEART_COUNTDOWN_TONE_CONSEQUENCE,
							AdvancementPolicy: pb.DaggerheartCountdownAdvancementPolicy_DAGGERHEART_COUNTDOWN_ADVANCEMENT_POLICY_MANUAL,
							StartingValue:     4,
							RemainingValue:    2,
							LoopBehavior:      pb.DaggerheartCountdownLoopBehavior_DAGGERHEART_COUNTDOWN_LOOP_BEHAVIOR_NONE,
							Status:            pb.DaggerheartCountdownStatus_DAGGERHEART_COUNTDOWN_STATUS_ACTIVE,
						}},
					}, nil
				},
			},
		})
		runtime.resolveSceneID = func(context.Context, string, string) (string, error) { return "scene-1", nil }

		value, err := ReadCombatBoardResource(runtime, context.Background(), "camp-1", "sess-1")
		if err != nil {
			t.Fatalf("ReadCombatBoardResource() error = %v", err)
		}

		var payload daggerheartCombatBoardPayload
		unmarshalJSON(t, value, &payload)
		if payload.Status != "READY" || payload.GmFear != 5 {
			t.Fatalf("combat board status/header = %#v", payload)
		}
		if payload.Spotlight == nil || payload.Spotlight.CharacterID != "char-1" || payload.Spotlight.UpdatedAt != updatedAt.Format(time.RFC3339) {
			t.Fatalf("spotlight = %#v", payload.Spotlight)
		}
		if len(payload.Countdowns) != 1 || payload.Countdowns[0].ID != "cd-1" {
			t.Fatalf("countdowns = %#v", payload.Countdowns)
		}
		if len(payload.Adversaries) != 1 || payload.Adversaries[0].ID != "adv-1" {
			t.Fatalf("adversaries = %#v", payload.Adversaries)
		}
	})

	t.Run("missing spotlight is tolerated and no active scene is diagnostic", func(t *testing.T) {
		t.Parallel()

		runtime := newDaggerheartReadTestRuntime(t, daggerheartReadTestServers{
			snapshots: &snapshotServiceServerStub{
				getSnapshotFunc: func(context.Context, *statev1.GetSnapshotRequest) (*statev1.GetSnapshotResponse, error) {
					return &statev1.GetSnapshotResponse{
						Snapshot: &statev1.Snapshot{
							SystemSnapshot: &statev1.Snapshot_Daggerheart{
								Daggerheart: &pb.DaggerheartSnapshot{GmFear: 1},
							},
						},
					}, nil
				},
			},
			sessions: &sessionServiceServerStub{
				getSessionSpotlightFunc: func(context.Context, *statev1.GetSessionSpotlightRequest) (*statev1.GetSessionSpotlightResponse, error) {
					return nil, status.Error(codes.NotFound, "missing")
				},
			},
			daggerheart: &daggerheartServiceServerStub{},
		})
		runtime.resolveSceneID = func(context.Context, string, string) (string, error) {
			return "", errors.New("no active scene")
		}

		value, err := ReadCombatBoardResource(runtime, context.Background(), "camp-1", "sess-1")
		if err != nil {
			t.Fatalf("ReadCombatBoardResource() error = %v", err)
		}

		var payload daggerheartCombatBoardPayload
		unmarshalJSON(t, value, &payload)
		if payload.Status != "NO_ACTIVE_SCENE" || len(payload.Issues) != 1 || payload.Issues[0].Code != "no_active_scene" {
			t.Fatalf("combat board diagnostics = %#v", payload)
		}
		if payload.Spotlight != nil {
			t.Fatalf("spotlight = %#v, want nil", payload.Spotlight)
		}
	})
}

func TestReadResourceDispatch(t *testing.T) {
	t.Parallel()

	runtime := newDaggerheartReadTestRuntime(t, daggerheartReadTestServers{
		characters: &characterServiceServerStub{
			getCharacterSheetFunc: func(context.Context, *statev1.GetCharacterSheetRequest) (*statev1.GetCharacterSheetResponse, error) {
				return &statev1.GetCharacterSheetResponse{
					Character: &statev1.Character{Id: "char-1", CampaignId: "camp-1", Name: "Aria"},
				}, nil
			},
		},
		snapshots: &snapshotServiceServerStub{
			getSnapshotFunc: func(context.Context, *statev1.GetSnapshotRequest) (*statev1.GetSnapshotResponse, error) {
				return &statev1.GetSnapshotResponse{
					Snapshot: &statev1.Snapshot{
						SystemSnapshot: &statev1.Snapshot_Daggerheart{
							Daggerheart: &pb.DaggerheartSnapshot{GmFear: 2},
						},
					},
				}, nil
			},
		},
		sessions: &sessionServiceServerStub{
			getSessionSpotlightFunc: func(context.Context, *statev1.GetSessionSpotlightRequest) (*statev1.GetSessionSpotlightResponse, error) {
				return &statev1.GetSessionSpotlightResponse{}, nil
			},
		},
		daggerheart: &daggerheartServiceServerStub{
			rulesVersionFunc: func(context.Context, *pb.RulesVersionRequest) (*pb.RulesVersionResponse, error) {
				return &pb.RulesVersionResponse{System: "daggerheart"}, nil
			},
			listCampaignCountdownsFunc: func(context.Context, *pb.DaggerheartListCampaignCountdownsRequest) (*pb.DaggerheartListCampaignCountdownsResponse, error) {
				return &pb.DaggerheartListCampaignCountdownsResponse{}, nil
			},
			listAdversariesFunc: func(context.Context, *pb.DaggerheartListAdversariesRequest) (*pb.DaggerheartListAdversariesResponse, error) {
				return &pb.DaggerheartListAdversariesResponse{}, nil
			},
			listSceneCountdownsFunc: func(context.Context, *pb.DaggerheartListSceneCountdownsRequest) (*pb.DaggerheartListSceneCountdownsResponse, error) {
				return &pb.DaggerheartListSceneCountdownsResponse{}, nil
			},
		},
	})
	runtime.resolveSceneID = func(context.Context, string, string) (string, error) { return "scene-1", nil }

	testCases := []struct {
		name      string
		uri       string
		wantKnown bool
		wantError bool
	}{
		{name: "rules version", uri: "daggerheart://rules/version", wantKnown: true},
		{name: "snapshot", uri: "daggerheart://campaign/camp-1/snapshot", wantKnown: true},
		{name: "combat board", uri: "daggerheart://campaign/camp-1/sessions/sess-1/combat_board", wantKnown: true},
		{name: "campaign countdowns", uri: "daggerheart://campaign/camp-1/campaign_countdowns", wantKnown: true},
		{name: "character sheet", uri: "campaign://camp-1/characters/char-1/sheet", wantKnown: true},
		{name: "invalid handled uri", uri: "daggerheart://campaign//snapshot", wantKnown: true, wantError: true},
		{name: "not handled", uri: "daggerheart://campaign/camp-1/other", wantKnown: false},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			value, known, err := ReadResource(runtime, context.Background(), tc.uri)
			if known != tc.wantKnown {
				t.Fatalf("known = %v, want %v", known, tc.wantKnown)
			}
			if (err != nil) != tc.wantError {
				t.Fatalf("error = %v, wantError=%v", err, tc.wantError)
			}
			if tc.wantKnown && !tc.wantError && value == "" {
				t.Fatal("value = empty, want non-empty")
			}
		})
	}
}

func unmarshalJSON(t *testing.T, input string, target any) {
	t.Helper()

	if err := json.Unmarshal([]byte(input), target); err != nil {
		t.Fatalf("json.Unmarshal(%q) error = %v", input, err)
	}
}
