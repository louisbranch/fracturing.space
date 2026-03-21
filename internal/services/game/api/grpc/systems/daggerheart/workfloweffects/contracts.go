package workfloweffects

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
)

type CountdownStore interface {
	GetDaggerheartCountdown(ctx context.Context, campaignID, countdownID string) (projectionstore.DaggerheartCountdown, error)
}

type ConditionChangeReplayCheckInput struct {
	CampaignID  string
	SessionID   string
	RollSeq     uint64
	RequestID   string
	CharacterID string
}

type ConditionChangeCommandInput struct {
	CampaignID    string
	SessionID     string
	RequestID     string
	InvocationID  string
	CorrelationID string
	CharacterID   string
	PayloadJSON   []byte
}

type ApplyStressVulnerableConditionInput struct {
	CampaignID    string
	SessionID     string
	CharacterID   string
	Conditions    []projectionstore.DaggerheartConditionState
	StressBefore  int
	StressAfter   int
	StressMax     int
	RollSeq       *uint64
	RequestID     string
	CorrelationID string
}

type Dependencies struct {
	Daggerheart CountdownStore

	ConditionChangeAlreadyApplied func(ctx context.Context, in ConditionChangeReplayCheckInput) (bool, error)
	ExecuteConditionChange        func(ctx context.Context, in ConditionChangeCommandInput) error
	CreateCountdown               func(ctx context.Context, in *pb.DaggerheartCreateCountdownRequest) error
	UpdateCountdown               func(ctx context.Context, in *pb.DaggerheartUpdateCountdownRequest) error
}
