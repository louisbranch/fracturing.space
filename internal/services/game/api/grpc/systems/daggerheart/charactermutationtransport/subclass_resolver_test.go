package charactermutationtransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

// subclassTestStore extends testDaggerheartStore with configurable state and
// adversary responses so the resolver can load targets during feature resolution.
type subclassTestStore struct {
	testDaggerheartStore
	states      map[string]projectionstore.DaggerheartCharacterState
	adversaries map[string]projectionstore.DaggerheartAdversary
}

func (s *subclassTestStore) GetDaggerheartCharacterState(_ context.Context, _, characterID string) (projectionstore.DaggerheartCharacterState, error) {
	if st, ok := s.states[characterID]; ok {
		return st, nil
	}
	return projectionstore.DaggerheartCharacterState{}, nil
}

func (s *subclassTestStore) GetDaggerheartCharacterProfile(_ context.Context, _, characterID string) (projectionstore.DaggerheartCharacterProfile, error) {
	if p, ok := s.profiles[characterID]; ok {
		return p, nil
	}
	return projectionstore.DaggerheartCharacterProfile{HpMax: 20}, nil
}

func (s *subclassTestStore) GetDaggerheartAdversary(_ context.Context, _, adversaryID string) (projectionstore.DaggerheartAdversary, error) {
	if a, ok := s.adversaries[adversaryID]; ok {
		return a, nil
	}
	return projectionstore.DaggerheartAdversary{}, nil
}

// profileWithSubclass returns a profile with the given subclass track at the
// specified rank.
func profileWithSubclass(subclassID, rank string) projectionstore.DaggerheartCharacterProfile {
	return projectionstore.DaggerheartCharacterProfile{
		CampaignID:  "camp-1",
		CharacterID: "char-1",
		Level:       5,
		HpMax:       20,
		Instinct:    3,
		SubclassTracks: []projectionstore.DaggerheartSubclassTrack{
			{SubclassID: subclassID, Rank: projectionstore.DaggerheartSubclassTrackRank(rank)},
		},
	}
}

func baseState() projectionstore.DaggerheartCharacterState {
	return projectionstore.DaggerheartCharacterState{
		Hp:      10,
		Hope:    5,
		HopeMax: 6,
		Stress:  3,
	}
}

// callResolver is the test entry point that calls resolveSubclassFeaturePayload
// with sensible defaults for classState (unused) and the provided overrides.
func callResolver(
	t *testing.T,
	store *subclassTestStore,
	profile projectionstore.DaggerheartCharacterProfile,
	state projectionstore.DaggerheartCharacterState,
	subclassState daggerheartstate.CharacterSubclassState,
	req *pb.DaggerheartApplySubclassFeatureRequest,
) (string, codes.Code) {
	t.Helper()
	h := NewHandler(Dependencies{
		Campaign:    testCampaignStore{},
		Daggerheart: store,
	})
	payload, err := h.resolveSubclassFeaturePayload(
		context.Background(),
		"camp-1",
		profile,
		state,
		daggerheartstate.CharacterClassState{},
		subclassState,
		req,
	)
	if err != nil {
		return "", status.Code(err)
	}
	return payload.Feature, codes.OK
}

func TestResolveSubclassFeature_NoFeatureSet(t *testing.T) {
	store := &subclassTestStore{}
	req := &pb.DaggerheartApplySubclassFeatureRequest{CharacterId: "char-1"}
	_, code := callResolver(t, store, profileWithSubclass("", ""), baseState(), daggerheartstate.CharacterSubclassState{}, req)
	if code != codes.InvalidArgument {
		t.Fatalf("want InvalidArgument, got %v", code)
	}
}

// ---------- BattleRitual ----------

func TestResolveSubclassFeature_BattleRitual(t *testing.T) {
	store := &subclassTestStore{}

	t.Run("happy_path", func(t *testing.T) {
		profile := profileWithSubclass("subclass.call_of_the_brave", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature:     &pb.DaggerheartApplySubclassFeatureRequest_BattleRitual{BattleRitual: &emptypb.Empty{}},
		}
		feat, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.OK {
			t.Fatalf("want OK, got %v", code)
		}
		if feat != "battle_ritual" {
			t.Fatalf("want feature battle_ritual, got %s", feat)
		}
	})

	t.Run("missing_prerequisite", func(t *testing.T) {
		profile := profileWithSubclass("subclass.other", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature:     &pb.DaggerheartApplySubclassFeatureRequest_BattleRitual{BattleRitual: &emptypb.Empty{}},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("already_used", func(t *testing.T) {
		profile := profileWithSubclass("subclass.call_of_the_brave", "foundation")
		ss := daggerheartstate.CharacterSubclassState{BattleRitualUsedThisLongRest: true}
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature:     &pb.DaggerheartApplySubclassFeatureRequest_BattleRitual{BattleRitual: &emptypb.Empty{}},
		}
		_, code := callResolver(t, store, profile, baseState(), ss, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})
}

// ---------- ContactsEverywhere ----------

func TestResolveSubclassFeature_ContactsEverywhere(t *testing.T) {
	store := &subclassTestStore{}

	t.Run("happy_path_action_bonus", func(t *testing.T) {
		profile := profileWithSubclass("subclass.syndicate", "specialization")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_ContactsEverywhere{
				ContactsEverywhere: &pb.DaggerheartContactsEverywhereRequest{Option: "next_action_bonus"},
			},
		}
		feat, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.OK {
			t.Fatalf("want OK, got %v", code)
		}
		if feat != "contacts_everywhere" {
			t.Fatalf("want feature contacts_everywhere, got %s", feat)
		}
	})

	t.Run("missing_prerequisite", func(t *testing.T) {
		profile := profileWithSubclass("subclass.syndicate", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_ContactsEverywhere{
				ContactsEverywhere: &pb.DaggerheartContactsEverywhereRequest{Option: "next_action_bonus"},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("max_uses_exhausted", func(t *testing.T) {
		profile := profileWithSubclass("subclass.syndicate", "specialization")
		ss := daggerheartstate.CharacterSubclassState{ContactsEverywhereUsesThisSession: 1}
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_ContactsEverywhere{
				ContactsEverywhere: &pb.DaggerheartContactsEverywhereRequest{Option: "next_damage_bonus"},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), ss, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("invalid_option", func(t *testing.T) {
		profile := profileWithSubclass("subclass.syndicate", "specialization")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_ContactsEverywhere{
				ContactsEverywhere: &pb.DaggerheartContactsEverywhereRequest{Option: "invalid"},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.InvalidArgument {
			t.Fatalf("want InvalidArgument, got %v", code)
		}
	})
}

// ---------- SparingTouch ----------

func TestResolveSubclassFeature_SparingTouch(t *testing.T) {
	store := &subclassTestStore{
		testDaggerheartStore: testDaggerheartStore{
			profiles: map[string]projectionstore.DaggerheartCharacterProfile{
				"target-1": {CharacterID: "target-1", HpMax: 20},
			},
		},
		states: map[string]projectionstore.DaggerheartCharacterState{
			"target-1": {Hp: 8, Stress: 4},
		},
	}

	t.Run("happy_path_hp", func(t *testing.T) {
		profile := profileWithSubclass("subclass.divine_wielder", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_SparingTouch{
				SparingTouch: &pb.DaggerheartSparingTouchRequest{TargetCharacterId: "target-1", Clear: "hp"},
			},
		}
		feat, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.OK {
			t.Fatalf("want OK, got %v", code)
		}
		if feat != "sparing_touch" {
			t.Fatalf("want feature sparing_touch, got %s", feat)
		}
	})

	t.Run("happy_path_stress", func(t *testing.T) {
		profile := profileWithSubclass("subclass.divine_wielder", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_SparingTouch{
				SparingTouch: &pb.DaggerheartSparingTouchRequest{TargetCharacterId: "target-1", Clear: "stress"},
			},
		}
		feat, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.OK {
			t.Fatalf("want OK, got %v", code)
		}
		if feat != "sparing_touch" {
			t.Fatalf("want feature sparing_touch, got %s", feat)
		}
	})

	t.Run("missing_prerequisite", func(t *testing.T) {
		profile := profileWithSubclass("subclass.other", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_SparingTouch{
				SparingTouch: &pb.DaggerheartSparingTouchRequest{TargetCharacterId: "target-1", Clear: "hp"},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("max_uses_exhausted", func(t *testing.T) {
		profile := profileWithSubclass("subclass.divine_wielder", "foundation")
		ss := daggerheartstate.CharacterSubclassState{SparingTouchUsesThisLongRest: 1}
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_SparingTouch{
				SparingTouch: &pb.DaggerheartSparingTouchRequest{TargetCharacterId: "target-1", Clear: "hp"},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), ss, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("missing_target", func(t *testing.T) {
		profile := profileWithSubclass("subclass.divine_wielder", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_SparingTouch{
				SparingTouch: &pb.DaggerheartSparingTouchRequest{TargetCharacterId: "", Clear: "hp"},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.InvalidArgument {
			t.Fatalf("want InvalidArgument, got %v", code)
		}
	})

	t.Run("invalid_clear", func(t *testing.T) {
		profile := profileWithSubclass("subclass.divine_wielder", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_SparingTouch{
				SparingTouch: &pb.DaggerheartSparingTouchRequest{TargetCharacterId: "target-1", Clear: "invalid"},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.InvalidArgument {
			t.Fatalf("want InvalidArgument, got %v", code)
		}
	})
}

// ---------- Elementalist ----------

func TestResolveSubclassFeature_Elementalist(t *testing.T) {
	store := &subclassTestStore{}

	t.Run("happy_path_action", func(t *testing.T) {
		profile := profileWithSubclass("subclass.elemental_origin", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Elementalist{
				Elementalist: &pb.DaggerheartElementalistRequest{Bonus: "action"},
			},
		}
		feat, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.OK {
			t.Fatalf("want OK, got %v", code)
		}
		if feat != "elementalist" {
			t.Fatalf("want feature elementalist, got %s", feat)
		}
	})

	t.Run("missing_prerequisite", func(t *testing.T) {
		profile := profileWithSubclass("subclass.other", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Elementalist{
				Elementalist: &pb.DaggerheartElementalistRequest{Bonus: "action"},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("insufficient_hope", func(t *testing.T) {
		profile := profileWithSubclass("subclass.elemental_origin", "foundation")
		st := baseState()
		st.Hope = 0
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Elementalist{
				Elementalist: &pb.DaggerheartElementalistRequest{Bonus: "damage"},
			},
		}
		_, code := callResolver(t, store, profile, st, daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("invalid_bonus", func(t *testing.T) {
		profile := profileWithSubclass("subclass.elemental_origin", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Elementalist{
				Elementalist: &pb.DaggerheartElementalistRequest{Bonus: "invalid"},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.InvalidArgument {
			t.Fatalf("want InvalidArgument, got %v", code)
		}
	})
}

// ---------- Transcendence ----------

func TestResolveSubclassFeature_Transcendence(t *testing.T) {
	store := &subclassTestStore{}

	t.Run("happy_path", func(t *testing.T) {
		profile := profileWithSubclass("subclass.elemental_origin", "specialization")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Transcendence{
				Transcendence: &pb.DaggerheartTranscendenceRequest{
					Bonuses: []string{"evasion", "proficiency"},
				},
			},
		}
		feat, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.OK {
			t.Fatalf("want OK, got %v", code)
		}
		if feat != "transcendence" {
			t.Fatalf("want feature transcendence, got %s", feat)
		}
	})

	t.Run("missing_prerequisite", func(t *testing.T) {
		profile := profileWithSubclass("subclass.elemental_origin", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Transcendence{
				Transcendence: &pb.DaggerheartTranscendenceRequest{
					Bonuses: []string{"evasion", "proficiency"},
				},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("already_active", func(t *testing.T) {
		profile := profileWithSubclass("subclass.elemental_origin", "specialization")
		ss := daggerheartstate.CharacterSubclassState{TranscendenceActive: true}
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Transcendence{
				Transcendence: &pb.DaggerheartTranscendenceRequest{
					Bonuses: []string{"evasion", "proficiency"},
				},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), ss, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("wrong_bonus_count", func(t *testing.T) {
		profile := profileWithSubclass("subclass.elemental_origin", "specialization")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Transcendence{
				Transcendence: &pb.DaggerheartTranscendenceRequest{
					Bonuses: []string{"evasion"},
				},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.InvalidArgument {
			t.Fatalf("want InvalidArgument, got %v", code)
		}
	})

	t.Run("trait_bonus_requires_trait", func(t *testing.T) {
		profile := profileWithSubclass("subclass.elemental_origin", "specialization")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Transcendence{
				Transcendence: &pb.DaggerheartTranscendenceRequest{
					Bonuses: []string{"trait", "evasion"},
					Trait:   "",
				},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.InvalidArgument {
			t.Fatalf("want InvalidArgument, got %v", code)
		}
	})
}

// ---------- VanishingAct ----------

func TestResolveSubclassFeature_VanishingAct(t *testing.T) {
	store := &subclassTestStore{}

	t.Run("happy_path", func(t *testing.T) {
		profile := profileWithSubclass("subclass.nightwalker", "specialization")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature:     &pb.DaggerheartApplySubclassFeatureRequest_VanishingAct{VanishingAct: &emptypb.Empty{}},
		}
		feat, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.OK {
			t.Fatalf("want OK, got %v", code)
		}
		if feat != "vanishing_act" {
			t.Fatalf("want feature vanishing_act, got %s", feat)
		}
	})

	t.Run("missing_prerequisite", func(t *testing.T) {
		profile := profileWithSubclass("subclass.nightwalker", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature:     &pb.DaggerheartApplySubclassFeatureRequest_VanishingAct{VanishingAct: &emptypb.Empty{}},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})
}

// ---------- ClarityOfNature ----------

func TestResolveSubclassFeature_ClarityOfNature(t *testing.T) {
	store := &subclassTestStore{
		states: map[string]projectionstore.DaggerheartCharacterState{
			"target-1": {Stress: 4},
		},
	}

	t.Run("happy_path", func(t *testing.T) {
		profile := profileWithSubclass("subclass.warden_of_renewal", "foundation")
		profile.Instinct = 3
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_ClarityOfNature{
				ClarityOfNature: &pb.DaggerheartClarityOfNatureRequest{
					Targets: []*pb.DaggerheartStressClearTarget{
						{CharacterId: "target-1", StressClear: 2},
					},
				},
			},
		}
		feat, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.OK {
			t.Fatalf("want OK, got %v", code)
		}
		if feat != "clarity_of_nature" {
			t.Fatalf("want feature clarity_of_nature, got %s", feat)
		}
	})

	t.Run("missing_prerequisite", func(t *testing.T) {
		profile := profileWithSubclass("subclass.other", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_ClarityOfNature{
				ClarityOfNature: &pb.DaggerheartClarityOfNatureRequest{
					Targets: []*pb.DaggerheartStressClearTarget{
						{CharacterId: "target-1", StressClear: 1},
					},
				},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("already_used", func(t *testing.T) {
		profile := profileWithSubclass("subclass.warden_of_renewal", "foundation")
		profile.Instinct = 3
		ss := daggerheartstate.CharacterSubclassState{ClarityOfNatureUsedThisLongRest: true}
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_ClarityOfNature{
				ClarityOfNature: &pb.DaggerheartClarityOfNatureRequest{
					Targets: []*pb.DaggerheartStressClearTarget{
						{CharacterId: "target-1", StressClear: 1},
					},
				},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), ss, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("stress_clear_exceeds_instinct", func(t *testing.T) {
		profile := profileWithSubclass("subclass.warden_of_renewal", "foundation")
		profile.Instinct = 2
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_ClarityOfNature{
				ClarityOfNature: &pb.DaggerheartClarityOfNatureRequest{
					Targets: []*pb.DaggerheartStressClearTarget{
						{CharacterId: "target-1", StressClear: 3},
					},
				},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.InvalidArgument {
			t.Fatalf("want InvalidArgument, got %v", code)
		}
	})
}

// ---------- Regeneration ----------

func TestResolveSubclassFeature_Regeneration(t *testing.T) {
	store := &subclassTestStore{
		testDaggerheartStore: testDaggerheartStore{
			profiles: map[string]projectionstore.DaggerheartCharacterProfile{
				"target-1": {CharacterID: "target-1", HpMax: 20},
			},
		},
		states: map[string]projectionstore.DaggerheartCharacterState{
			"target-1": {Hp: 5},
		},
	}

	t.Run("happy_path", func(t *testing.T) {
		profile := profileWithSubclass("subclass.warden_of_renewal", "specialization")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Regeneration{
				Regeneration: &pb.DaggerheartRegenerationRequest{TargetCharacterId: "target-1", ClearHp: 3},
			},
		}
		feat, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.OK {
			t.Fatalf("want OK, got %v", code)
		}
		if feat != "regeneration" {
			t.Fatalf("want feature regeneration, got %s", feat)
		}
	})

	t.Run("missing_prerequisite", func(t *testing.T) {
		profile := profileWithSubclass("subclass.warden_of_renewal", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Regeneration{
				Regeneration: &pb.DaggerheartRegenerationRequest{TargetCharacterId: "target-1", ClearHp: 2},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("insufficient_hope", func(t *testing.T) {
		profile := profileWithSubclass("subclass.warden_of_renewal", "specialization")
		st := baseState()
		st.Hope = 2
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Regeneration{
				Regeneration: &pb.DaggerheartRegenerationRequest{TargetCharacterId: "target-1", ClearHp: 2},
			},
		}
		_, code := callResolver(t, store, profile, st, daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("clear_hp_out_of_range", func(t *testing.T) {
		profile := profileWithSubclass("subclass.warden_of_renewal", "specialization")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Regeneration{
				Regeneration: &pb.DaggerheartRegenerationRequest{TargetCharacterId: "target-1", ClearHp: 5},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.InvalidArgument {
			t.Fatalf("want InvalidArgument, got %v", code)
		}
	})

	t.Run("missing_target", func(t *testing.T) {
		profile := profileWithSubclass("subclass.warden_of_renewal", "specialization")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Regeneration{
				Regeneration: &pb.DaggerheartRegenerationRequest{TargetCharacterId: "", ClearHp: 2},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.InvalidArgument {
			t.Fatalf("want InvalidArgument, got %v", code)
		}
	})
}

// ---------- WardensProtection ----------

func TestResolveSubclassFeature_WardensProtection(t *testing.T) {
	store := &subclassTestStore{
		testDaggerheartStore: testDaggerheartStore{
			profiles: map[string]projectionstore.DaggerheartCharacterProfile{
				"target-1": {CharacterID: "target-1", HpMax: 20},
			},
		},
		states: map[string]projectionstore.DaggerheartCharacterState{
			"target-1": {Hp: 8},
		},
	}

	t.Run("happy_path", func(t *testing.T) {
		profile := profileWithSubclass("subclass.warden_of_renewal", "mastery")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_WardensProtection{
				WardensProtection: &pb.DaggerheartWardensProtectionRequest{TargetCharacterIds: []string{"target-1"}},
			},
		}
		feat, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.OK {
			t.Fatalf("want OK, got %v", code)
		}
		if feat != "wardens_protection" {
			t.Fatalf("want feature wardens_protection, got %s", feat)
		}
	})

	t.Run("missing_prerequisite", func(t *testing.T) {
		profile := profileWithSubclass("subclass.warden_of_renewal", "specialization")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_WardensProtection{
				WardensProtection: &pb.DaggerheartWardensProtectionRequest{TargetCharacterIds: []string{"target-1"}},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("already_used", func(t *testing.T) {
		profile := profileWithSubclass("subclass.warden_of_renewal", "mastery")
		ss := daggerheartstate.CharacterSubclassState{WardensProtectionUsedThisLongRest: true}
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_WardensProtection{
				WardensProtection: &pb.DaggerheartWardensProtectionRequest{TargetCharacterIds: []string{"target-1"}},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), ss, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("insufficient_hope", func(t *testing.T) {
		profile := profileWithSubclass("subclass.warden_of_renewal", "mastery")
		st := baseState()
		st.Hope = 1
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_WardensProtection{
				WardensProtection: &pb.DaggerheartWardensProtectionRequest{TargetCharacterIds: []string{"target-1"}},
			},
		}
		_, code := callResolver(t, store, profile, st, daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("too_many_targets", func(t *testing.T) {
		profile := profileWithSubclass("subclass.warden_of_renewal", "mastery")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_WardensProtection{
				WardensProtection: &pb.DaggerheartWardensProtectionRequest{
					TargetCharacterIds: []string{"a", "b", "c", "d", "e"},
				},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.InvalidArgument {
			t.Fatalf("want InvalidArgument, got %v", code)
		}
	})
}

// ---------- ElementalIncarnation ----------

func TestResolveSubclassFeature_ElementalIncarnation(t *testing.T) {
	store := &subclassTestStore{}

	t.Run("happy_path", func(t *testing.T) {
		profile := profileWithSubclass("subclass.warden_of_the_elements", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_ElementalIncarnation{
				ElementalIncarnation: &pb.DaggerheartElementalIncarnationRequest{Channel: daggerheartstate.ElementalChannelFire},
			},
		}
		feat, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.OK {
			t.Fatalf("want OK, got %v", code)
		}
		if feat != "elemental_incarnation" {
			t.Fatalf("want feature elemental_incarnation, got %s", feat)
		}
	})

	t.Run("missing_prerequisite", func(t *testing.T) {
		profile := profileWithSubclass("subclass.other", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_ElementalIncarnation{
				ElementalIncarnation: &pb.DaggerheartElementalIncarnationRequest{Channel: daggerheartstate.ElementalChannelAir},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("invalid_channel", func(t *testing.T) {
		profile := profileWithSubclass("subclass.warden_of_the_elements", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_ElementalIncarnation{
				ElementalIncarnation: &pb.DaggerheartElementalIncarnationRequest{Channel: "lightning"},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.InvalidArgument {
			t.Fatalf("want InvalidArgument, got %v", code)
		}
	})
}

// ---------- RousingSpeech ----------

func TestResolveSubclassFeature_RousingSpeech(t *testing.T) {
	store := &subclassTestStore{
		states: map[string]projectionstore.DaggerheartCharacterState{
			"target-1": {Stress: 5},
		},
	}

	t.Run("happy_path", func(t *testing.T) {
		profile := profileWithSubclass("subclass.wordsmith", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_RousingSpeech{
				RousingSpeech: &pb.DaggerheartRousingSpeechRequest{TargetCharacterIds: []string{"target-1"}},
			},
		}
		feat, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.OK {
			t.Fatalf("want OK, got %v", code)
		}
		if feat != "rousing_speech" {
			t.Fatalf("want feature rousing_speech, got %s", feat)
		}
	})

	t.Run("missing_prerequisite", func(t *testing.T) {
		profile := profileWithSubclass("subclass.other", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_RousingSpeech{
				RousingSpeech: &pb.DaggerheartRousingSpeechRequest{TargetCharacterIds: []string{"target-1"}},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("already_used", func(t *testing.T) {
		profile := profileWithSubclass("subclass.wordsmith", "foundation")
		ss := daggerheartstate.CharacterSubclassState{RousingSpeechUsedThisLongRest: true}
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_RousingSpeech{
				RousingSpeech: &pb.DaggerheartRousingSpeechRequest{TargetCharacterIds: []string{"target-1"}},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), ss, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("no_targets", func(t *testing.T) {
		profile := profileWithSubclass("subclass.wordsmith", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_RousingSpeech{
				RousingSpeech: &pb.DaggerheartRousingSpeechRequest{},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.InvalidArgument {
			t.Fatalf("want InvalidArgument, got %v", code)
		}
	})
}

// ---------- Nemesis ----------

func TestResolveSubclassFeature_Nemesis(t *testing.T) {
	store := &subclassTestStore{}

	t.Run("happy_path", func(t *testing.T) {
		profile := profileWithSubclass("subclass.vengeance", "mastery")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Nemesis{
				Nemesis: &pb.DaggerheartNemesisRequest{AdversaryId: "adv-1"},
			},
		}
		feat, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.OK {
			t.Fatalf("want OK, got %v", code)
		}
		if feat != "nemesis" {
			t.Fatalf("want feature nemesis, got %s", feat)
		}
	})

	t.Run("missing_prerequisite", func(t *testing.T) {
		profile := profileWithSubclass("subclass.vengeance", "specialization")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Nemesis{
				Nemesis: &pb.DaggerheartNemesisRequest{AdversaryId: "adv-1"},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("insufficient_hope", func(t *testing.T) {
		profile := profileWithSubclass("subclass.vengeance", "mastery")
		st := baseState()
		st.Hope = 1
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Nemesis{
				Nemesis: &pb.DaggerheartNemesisRequest{AdversaryId: "adv-1"},
			},
		}
		_, code := callResolver(t, store, profile, st, daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("missing_adversary_id", func(t *testing.T) {
		profile := profileWithSubclass("subclass.vengeance", "mastery")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_Nemesis{
				Nemesis: &pb.DaggerheartNemesisRequest{AdversaryId: ""},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.InvalidArgument {
			t.Fatalf("want InvalidArgument, got %v", code)
		}
	})
}

// ---------- GiftedPerformer ----------

func TestResolveSubclassFeature_GiftedPerformer(t *testing.T) {
	store := &subclassTestStore{
		testDaggerheartStore: testDaggerheartStore{
			profiles: map[string]projectionstore.DaggerheartCharacterProfile{
				"char-1":   {CharacterID: "char-1", HpMax: 20},
				"target-1": {CharacterID: "target-1", HpMax: 20},
			},
		},
		states: map[string]projectionstore.DaggerheartCharacterState{
			"char-1":   {Hp: 10, Hope: 3, HopeMax: 6, Stress: 2},
			"target-1": {Hp: 8, Hope: 3, HopeMax: 6},
		},
	}

	t.Run("relaxing_song_happy_path", func(t *testing.T) {
		profile := profileWithSubclass("subclass.troubadour", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_GiftedPerformer{
				GiftedPerformer: &pb.DaggerheartGiftedPerformerRequest{
					Song:               "relaxing_song",
					TargetCharacterIds: []string{"target-1"},
				},
			},
		}
		feat, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.OK {
			t.Fatalf("want OK, got %v", code)
		}
		if feat != "gifted_performer_relaxing_song" {
			t.Fatalf("want feature gifted_performer_relaxing_song, got %s", feat)
		}
	})

	t.Run("heartbreaking_song_happy_path", func(t *testing.T) {
		profile := profileWithSubclass("subclass.troubadour", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_GiftedPerformer{
				GiftedPerformer: &pb.DaggerheartGiftedPerformerRequest{
					Song:               "heartbreaking_song",
					TargetCharacterIds: []string{"target-1"},
				},
			},
		}
		feat, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.OK {
			t.Fatalf("want OK, got %v", code)
		}
		if feat != "gifted_performer_heartbreaking_song" {
			t.Fatalf("want feature gifted_performer_heartbreaking_song, got %s", feat)
		}
	})

	t.Run("epic_song_happy_path_character_target", func(t *testing.T) {
		profile := profileWithSubclass("subclass.troubadour", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_GiftedPerformer{
				GiftedPerformer: &pb.DaggerheartGiftedPerformerRequest{
					Song:     "epic_song",
					TargetId: "target-1",
				},
			},
		}
		feat, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.OK {
			t.Fatalf("want OK, got %v", code)
		}
		if feat != "gifted_performer_epic_song" {
			t.Fatalf("want feature gifted_performer_epic_song, got %s", feat)
		}
	})

	t.Run("missing_prerequisite", func(t *testing.T) {
		profile := profileWithSubclass("subclass.other", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_GiftedPerformer{
				GiftedPerformer: &pb.DaggerheartGiftedPerformerRequest{Song: "relaxing_song"},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("relaxing_song_max_uses", func(t *testing.T) {
		profile := profileWithSubclass("subclass.troubadour", "foundation")
		ss := daggerheartstate.CharacterSubclassState{GiftedPerformerRelaxingSongUses: 1}
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_GiftedPerformer{
				GiftedPerformer: &pb.DaggerheartGiftedPerformerRequest{
					Song:               "relaxing_song",
					TargetCharacterIds: []string{"target-1"},
				},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), ss, req)
		if code != codes.FailedPrecondition {
			t.Fatalf("want FailedPrecondition, got %v", code)
		}
	})

	t.Run("epic_song_missing_target", func(t *testing.T) {
		profile := profileWithSubclass("subclass.troubadour", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_GiftedPerformer{
				GiftedPerformer: &pb.DaggerheartGiftedPerformerRequest{
					Song:     "epic_song",
					TargetId: "",
				},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.InvalidArgument {
			t.Fatalf("want InvalidArgument, got %v", code)
		}
	})

	t.Run("unknown_song", func(t *testing.T) {
		profile := profileWithSubclass("subclass.troubadour", "foundation")
		req := &pb.DaggerheartApplySubclassFeatureRequest{
			CharacterId: "char-1",
			Feature: &pb.DaggerheartApplySubclassFeatureRequest_GiftedPerformer{
				GiftedPerformer: &pb.DaggerheartGiftedPerformerRequest{Song: ""},
			},
		}
		_, code := callResolver(t, store, profile, baseState(), daggerheartstate.CharacterSubclassState{}, req)
		if code != codes.InvalidArgument {
			t.Fatalf("want InvalidArgument, got %v", code)
		}
	})
}
