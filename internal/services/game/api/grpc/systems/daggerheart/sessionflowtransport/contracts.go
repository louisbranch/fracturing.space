package sessionflowtransport

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
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
}
