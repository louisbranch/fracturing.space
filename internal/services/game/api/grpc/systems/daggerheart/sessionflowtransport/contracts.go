package sessionflowtransport

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/projectionstore"
)

// Dependencies groups the lower-level transport handlers the session flow layer
// composes.
type Dependencies struct {
	SessionActionRoll           func(context.Context, *pb.SessionActionRollRequest) (*pb.SessionActionRollResponse, error)
	SessionDamageRoll           func(context.Context, *pb.SessionDamageRollRequest) (*pb.SessionDamageRollResponse, error)
	SessionAdversaryAttackRoll  func(context.Context, *pb.SessionAdversaryAttackRollRequest) (*pb.SessionAdversaryAttackRollResponse, error)
	ApplyRollOutcome            func(context.Context, *pb.ApplyRollOutcomeRequest) (*pb.ApplyRollOutcomeResponse, error)
	ApplyAttackOutcome          func(context.Context, *pb.DaggerheartApplyAttackOutcomeRequest) (*pb.DaggerheartApplyAttackOutcomeResponse, error)
	ApplyReactionOutcome        func(context.Context, *pb.DaggerheartApplyReactionOutcomeRequest) (*pb.DaggerheartApplyReactionOutcomeResponse, error)
	ApplyAdversaryAttackOutcome func(context.Context, *pb.DaggerheartApplyAdversaryAttackOutcomeRequest) (*pb.DaggerheartApplyAdversaryAttackOutcomeResponse, error)
	ApplyDamage                 func(context.Context, *pb.DaggerheartApplyDamageRequest) (*pb.DaggerheartApplyDamageResponse, error)
	ApplyAdversaryDamage        func(context.Context, *pb.DaggerheartApplyAdversaryDamageRequest) (*pb.DaggerheartApplyAdversaryDamageResponse, error)
	LoadCharacterProfile        func(context.Context, string, string) (projectionstore.DaggerheartCharacterProfile, error)
	LoadCharacterState          func(context.Context, string, string) (projectionstore.DaggerheartCharacterState, error)
	LoadAdversary               func(context.Context, string, string, string) (projectionstore.DaggerheartAdversary, error)
	LoadAdversaryEntry          func(context.Context, string) (contentstore.DaggerheartAdversaryEntry, error)
	LoadSubclass                func(context.Context, string) (contentstore.DaggerheartSubclass, error)
	LoadArmor                   func(context.Context, string) (contentstore.DaggerheartArmor, error)
	ExecuteCharacterStatePatch  func(context.Context, CharacterStatePatchInput) error
	ExecuteAdversaryUpdate      func(context.Context, AdversaryUpdateInput) error
	AdjustGMFear                func(context.Context, GMFearAdjustInput) error
	SeedFunc                    func() (int64, error)
}

type CharacterStatePatchInput struct {
	CampaignID                          string
	SessionID                           string
	SceneID                             string
	RequestID                           string
	InvocationID                        string
	CharacterID                         string
	Source                              string
	HopeBefore                          *int
	HopeAfter                           *int
	ArmorBefore                         *int
	ArmorAfter                          *int
	ClassStateBefore                    *daggerheart.CharacterClassState
	ClassStateAfter                     *daggerheart.CharacterClassState
	SubclassStateBefore                 *daggerheart.CharacterSubclassState
	SubclassStateAfter                  *daggerheart.CharacterSubclassState
	ImpenetrableUsedThisShortRestBefore *bool
	ImpenetrableUsedThisShortRestAfter  *bool
}

type AdversaryUpdateInput struct {
	CampaignID               string
	SessionID                string
	SceneID                  string
	RequestID                string
	InvocationID             string
	Adversary                projectionstore.DaggerheartAdversary
	UpdatedStress            int
	UpdatedFeatureStates     []projectionstore.DaggerheartAdversaryFeatureState
	UpdatedPendingExperience *projectionstore.DaggerheartAdversaryPendingExperience
	ClearPendingExperience   bool
	Source                   string
}

type GMFearAdjustInput struct {
	CampaignID   string
	SessionID    string
	SceneID      string
	RequestID    string
	InvocationID string
	Delta        int
	Reason       string
}

type ArmorFeatureRollInput struct {
	Rng   *commonv1.RngRequest
	Sides int
}
