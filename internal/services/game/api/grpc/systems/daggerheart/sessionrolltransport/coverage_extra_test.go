package sessionrolltransport

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeContentStore struct{}

func (fakeContentStore) GetDaggerheartArmor(context.Context, string) (contentstore.DaggerheartArmor, error) {
	return contentstore.DaggerheartArmor{}, nil
}

func TestSessionRollDependencyGuardsReportMissingExecutors(t *testing.T) {
	t.Parallel()

	base := Dependencies{
		Campaign:    testCampaignStore{},
		Session:     testSessionStore{},
		SessionGate: testSessionGateStore{},
		Daggerheart: testDaggerheartStore{},
		Event:       testEventStore{},
		SeedFunc:    func() (int64, error) { return 7, nil },
		ExecuteActionRollResolve: func(context.Context, RollResolveInput) (uint64, error) {
			return 0, nil
		},
		ExecuteHopeSpend:       func(context.Context, HopeSpendInput) error { return nil },
		AdvanceBreathCountdown: func(context.Context, string, string, string, bool) error { return nil },
		ExecuteDamageRollResolve: func(context.Context, RollResolveInput) (uint64, error) {
			return 0, nil
		},
		ExecuteAdversaryRollResolve: func(context.Context, RollResolveInput) (uint64, error) {
			return 0, nil
		},
		LoadAdversaryForSession: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
			return projectionstore.DaggerheartAdversary{}, nil
		},
	}

	tests := []struct {
		name    string
		run     func(*Handler) error
		mutate  func(*Dependencies)
		wantMsg string
	}{
		{
			name: "missing campaign store",
			run:  func(h *Handler) error { return h.requireActionRollDependencies() },
			mutate: func(deps *Dependencies) {
				deps.Campaign = nil
			},
			wantMsg: "campaign store is not configured",
		},
		{
			name: "missing session store",
			run:  func(h *Handler) error { return h.requireDamageRollDependencies() },
			mutate: func(deps *Dependencies) {
				deps.Session = nil
			},
			wantMsg: "session store is not configured",
		},
		{
			name: "missing session gate store",
			run:  func(h *Handler) error { return h.requireAdversaryRollDependencies() },
			mutate: func(deps *Dependencies) {
				deps.SessionGate = nil
			},
			wantMsg: "session gate store is not configured",
		},
		{
			name: "missing daggerheart store",
			run:  func(h *Handler) error { return h.requireAdversaryActionCheckDependencies() },
			mutate: func(deps *Dependencies) {
				deps.Daggerheart = nil
			},
			wantMsg: "daggerheart store is not configured",
		},
		{
			name: "missing event store",
			run:  func(h *Handler) error { return h.requireActionRollDependencies() },
			mutate: func(deps *Dependencies) {
				deps.Event = nil
			},
			wantMsg: "event store is not configured",
		},
		{
			name: "missing seed generator for damage roll",
			run:  func(h *Handler) error { return h.requireDamageRollDependencies() },
			mutate: func(deps *Dependencies) {
				deps.SeedFunc = nil
			},
			wantMsg: "seed generator is not configured",
		},
		{
			name: "missing action roll executor",
			run:  func(h *Handler) error { return h.requireActionRollDependencies() },
			mutate: func(deps *Dependencies) {
				deps.ExecuteActionRollResolve = nil
			},
			wantMsg: "action roll executor is not configured",
		},
		{
			name: "missing hope spend executor",
			run:  func(h *Handler) error { return h.requireActionRollDependencies() },
			mutate: func(deps *Dependencies) {
				deps.ExecuteHopeSpend = nil
			},
			wantMsg: "hope spend executor is not configured",
		},
		{
			name: "missing breath countdown handler",
			run:  func(h *Handler) error { return h.requireActionRollDependencies() },
			mutate: func(deps *Dependencies) {
				deps.AdvanceBreathCountdown = nil
			},
			wantMsg: "breath countdown handler is not configured",
		},
		{
			name: "missing damage roll executor",
			run:  func(h *Handler) error { return h.requireDamageRollDependencies() },
			mutate: func(deps *Dependencies) {
				deps.ExecuteDamageRollResolve = nil
			},
			wantMsg: "damage roll executor is not configured",
		},
		{
			name: "missing adversary roll executor",
			run:  func(h *Handler) error { return h.requireAdversaryRollDependencies() },
			mutate: func(deps *Dependencies) {
				deps.ExecuteAdversaryRollResolve = nil
			},
			wantMsg: "adversary roll executor is not configured",
		},
		{
			name: "missing adversary loader",
			run:  func(h *Handler) error { return h.requireAdversaryRollDependencies() },
			mutate: func(deps *Dependencies) {
				deps.LoadAdversaryForSession = nil
			},
			wantMsg: "adversary loader is not configured",
		},
		{
			name: "missing seed generator for adversary action check",
			run:  func(h *Handler) error { return h.requireAdversaryActionCheckDependencies() },
			mutate: func(deps *Dependencies) {
				deps.SeedFunc = nil
			},
			wantMsg: "seed generator is not configured",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			deps := base
			tc.mutate(&deps)
			err := tc.run(NewHandler(deps))
			if status.Code(err) != codes.Internal {
				t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
			}
			if got := status.Convert(err).Message(); got != tc.wantMsg {
				t.Fatalf("message = %q, want %q", got, tc.wantMsg)
			}
		})
	}
}

func TestSessionRollHandlersValidateRequestsAndPreconditions(t *testing.T) {
	t.Parallel()

	t.Run("action roll validates request and local preconditions", func(t *testing.T) {
		handler := newTestHandler(Dependencies{
			ExecuteActionRollResolve: func(context.Context, RollResolveInput) (uint64, error) { return 0, nil },
			ExecuteHopeSpend:         func(context.Context, HopeSpendInput) error { return nil },
			AdvanceBreathCountdown:   func(context.Context, string, string, string, bool) error { return nil },
		})
		if _, err := handler.SessionActionRoll(context.Background(), nil); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
		}
		if _, err := handler.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
			CampaignId:  "camp-1",
			SessionId:   "sess-1",
			CharacterId: "char-1",
			Trait:       "agility",
			RollKind:    pb.RollKind_ROLL_KIND_REACTION,
			HopeSpends:  []*pb.ActionRollHopeSpend{{Amount: 1, Source: "experience"}},
		}); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
		}

		handler = newTestHandler(Dependencies{
			ExecuteActionRollResolve: func(context.Context, RollResolveInput) (uint64, error) { return 0, nil },
			ExecuteHopeSpend:         func(context.Context, HopeSpendInput) error { return nil },
			AdvanceBreathCountdown:   func(context.Context, string, string, string, bool) error { return nil },
		})
		if _, err := handler.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
			CampaignId:           "camp-1",
			SessionId:            "sess-1",
			CharacterId:          "char-1",
			Trait:                "agility",
			RollKind:             pb.RollKind_ROLL_KIND_ACTION,
			ReplaceHopeWithArmor: true,
		}); status.Code(err) != codes.Internal {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
		}

		handler = newTestHandler(Dependencies{
			Content:                  fakeContentStore{},
			ExecuteActionRollResolve: func(context.Context, RollResolveInput) (uint64, error) { return 0, nil },
			ExecuteHopeSpend:         func(context.Context, HopeSpendInput) error { return nil },
			AdvanceBreathCountdown:   func(context.Context, string, string, string, bool) error { return nil },
		})
		if _, err := handler.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
			CampaignId:           "camp-1",
			SessionId:            "sess-1",
			CharacterId:          "char-1",
			Trait:                "agility",
			RollKind:             pb.RollKind_ROLL_KIND_ACTION,
			ReplaceHopeWithArmor: true,
		}); status.Code(err) != codes.Internal {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
		}

		handler = newTestHandler(Dependencies{
			Session:                  testSessionStore{record: storage.SessionRecord{ID: "sess-1", CampaignID: "camp-1", Status: session.StatusEnded}},
			ExecuteActionRollResolve: func(context.Context, RollResolveInput) (uint64, error) { return 0, nil },
			ExecuteHopeSpend:         func(context.Context, HopeSpendInput) error { return nil },
			AdvanceBreathCountdown:   func(context.Context, string, string, string, bool) error { return nil },
		})
		if _, err := handler.SessionActionRoll(context.Background(), &pb.SessionActionRollRequest{
			CampaignId:  "camp-1",
			SessionId:   "sess-1",
			CharacterId: "char-1",
			Trait:       "agility",
		}); status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
		}
	})

	t.Run("damage roll validates request and session preconditions", func(t *testing.T) {
		handler := newTestHandler(Dependencies{
			ExecuteDamageRollResolve: func(context.Context, RollResolveInput) (uint64, error) { return 0, nil },
		})
		if _, err := handler.SessionDamageRoll(context.Background(), nil); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
		}
		if _, err := handler.SessionDamageRoll(context.Background(), &pb.SessionDamageRollRequest{
			CampaignId:  "camp-1",
			SessionId:   "sess-1",
			CharacterId: "char-1",
		}); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
		}

		handler = newTestHandler(Dependencies{
			Session:                  testSessionStore{record: storage.SessionRecord{ID: "sess-1", CampaignID: "camp-1", Status: session.StatusEnded}},
			ExecuteDamageRollResolve: func(context.Context, RollResolveInput) (uint64, error) { return 0, nil },
		})
		if _, err := handler.SessionDamageRoll(context.Background(), &pb.SessionDamageRollRequest{
			CampaignId:  "camp-1",
			SessionId:   "sess-1",
			CharacterId: "char-1",
			Dice:        []*pb.DiceSpec{{Sides: 6, Count: 1}},
		}); status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
		}
	})

	t.Run("adversary attack roll validates request and session preconditions", func(t *testing.T) {
		handler := newTestHandler(Dependencies{
			ExecuteAdversaryRollResolve: func(context.Context, RollResolveInput) (uint64, error) { return 0, nil },
			LoadAdversaryForSession: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
				return projectionstore.DaggerheartAdversary{AdversaryID: "adv-1", SessionID: "sess-1"}, nil
			},
		})
		if _, err := handler.SessionAdversaryAttackRoll(context.Background(), nil); status.Code(err) != codes.InvalidArgument {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
		}

		handler = newTestHandler(Dependencies{
			Session:                     testSessionStore{record: storage.SessionRecord{ID: "sess-1", CampaignID: "camp-1", Status: session.StatusEnded}},
			ExecuteAdversaryRollResolve: func(context.Context, RollResolveInput) (uint64, error) { return 0, nil },
			LoadAdversaryForSession: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
				return projectionstore.DaggerheartAdversary{AdversaryID: "adv-1", SessionID: "sess-1"}, nil
			},
		})
		if _, err := handler.SessionAdversaryAttackRoll(context.Background(), &pb.SessionAdversaryAttackRollRequest{
			CampaignId:  "camp-1",
			SessionId:   "sess-1",
			AdversaryId: "adv-1",
		}); status.Code(err) != codes.FailedPrecondition {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.FailedPrecondition)
		}
	})

	t.Run("adversary action check covers dramatic rng and feature-apply failure", func(t *testing.T) {
		var applied AdversaryFeatureApplyInput
		handler := newTestHandler(Dependencies{
			LoadAdversaryForSession: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
				return projectionstore.DaggerheartAdversary{
					AdversaryID: "adv-1",
					SessionID:   "sess-1",
					PendingExperience: &projectionstore.DaggerheartAdversaryPendingExperience{
						Name:     "ambush",
						Modifier: 2,
					},
				}, nil
			},
			ExecuteAdversaryFeatureApply: func(context.Context, AdversaryFeatureApplyInput) error {
				return status.Error(codes.Internal, "feature apply failed")
			},
		})
		if _, err := handler.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
			CampaignId:  "camp-1",
			SessionId:   "sess-1",
			AdversaryId: "adv-1",
			Difficulty:  10,
		}); status.Code(err) != codes.Internal {
			t.Fatalf("status code = %v, want %v", status.Code(err), codes.Internal)
		}

		handler = newTestHandler(Dependencies{
			LoadAdversaryForSession: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
				return projectionstore.DaggerheartAdversary{
					AdversaryID:       "adv-1",
					SessionID:         "sess-1",
					PendingExperience: nil,
				}, nil
			},
			ExecuteAdversaryFeatureApply: func(_ context.Context, in AdversaryFeatureApplyInput) error {
				applied = in
				return nil
			},
			SeedFunc: func() (int64, error) { return 42, nil },
		})
		resp, err := handler.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
			CampaignId:  "camp-1",
			SessionId:   "sess-1",
			AdversaryId: "adv-1",
			Difficulty:  5,
			Dramatic:    true,
			Rng:         &commonv1.RngRequest{},
		})
		if err != nil {
			t.Fatalf("SessionAdversaryActionCheck() error = %v", err)
		}
		if resp.GetAutoSuccess() {
			t.Fatal("expected dramatic action check to roll")
		}
		if resp.GetRng() == nil || resp.GetRoll() == 0 {
			t.Fatalf("response = %+v, want rng and non-zero roll", resp)
		}
		if applied.FeatureID != "" {
			t.Fatalf("unexpected feature apply call: %+v", applied)
		}
	})
}

func TestSessionAdversaryActionCheckValidatesAndClearsPendingExperience(t *testing.T) {
	t.Parallel()

	handler := newTestHandler(Dependencies{
		LoadAdversaryForSession: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
			return projectionstore.DaggerheartAdversary{
				AdversaryID: "adv-1",
				SessionID:   "sess-1",
				PendingExperience: &projectionstore.DaggerheartAdversaryPendingExperience{
					Name:     "ambush",
					Modifier: 2,
				},
			}, nil
		},
	})

	if _, err := handler.SessionAdversaryActionCheck(context.Background(), nil); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}

	if _, err := handler.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		AdversaryId: "adv-1",
		Difficulty:  -1,
	}); status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status code = %v, want %v", status.Code(err), codes.InvalidArgument)
	}

	var applied AdversaryFeatureApplyInput
	handler = newTestHandler(Dependencies{
		LoadAdversaryForSession: func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error) {
			return projectionstore.DaggerheartAdversary{
				AdversaryID: "adv-1",
				SessionID:   "sess-1",
				PendingExperience: &projectionstore.DaggerheartAdversaryPendingExperience{
					Name:     "ambush",
					Modifier: 2,
				},
			}, nil
		},
		ExecuteAdversaryFeatureApply: func(_ context.Context, in AdversaryFeatureApplyInput) error {
			applied = in
			return nil
		},
	})

	resp, err := handler.SessionAdversaryActionCheck(context.Background(), &pb.SessionAdversaryActionCheckRequest{
		CampaignId:  "camp-1",
		SessionId:   "sess-1",
		SceneId:     "scene-1",
		AdversaryId: "adv-1",
		Difficulty:  10,
		Modifiers:   []*pb.ActionRollModifier{{Source: "setup", Value: 1}},
	})
	if err != nil {
		t.Fatalf("SessionAdversaryActionCheck() error = %v", err)
	}
	if !resp.GetAutoSuccess() || !resp.GetSuccess() {
		t.Fatalf("response = %+v", resp)
	}
	if resp.GetTotal() != 3 {
		t.Fatalf("total = %d, want %d", resp.GetTotal(), 3)
	}
	if applied.FeatureID != "experience:ambush" || applied.PendingExperienceBefore == nil || applied.PendingExperienceAfter != nil {
		t.Fatalf("applied = %+v", applied)
	}
}
