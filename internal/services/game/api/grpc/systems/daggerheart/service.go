package daggerheart

import (
	"context"
	"fmt"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/gameplaystores"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/mechanicstransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
)

// DaggerheartService implements the Daggerheart gRPC API.
type DaggerheartService struct {
	pb.UnimplementedDaggerheartServiceServer
	stores   gameplaystores.Stores
	applier  projection.Applier
	seedFunc func() (int64, error) // Generates per-request random seeds.
}

// NewDaggerheartService creates a configured gRPC handler with a seed generator.
func NewDaggerheartService(stores gameplaystores.Stores, seedFunc func() (int64, error)) (*DaggerheartService, error) {
	if err := stores.Validate(); err != nil {
		return nil, fmt.Errorf("validate stores: %w", err)
	}
	if seedFunc == nil {
		return nil, fmt.Errorf("seed generator is required")
	}
	applier, err := stores.TryApplier()
	if err != nil {
		return nil, fmt.Errorf("build projection applier: %w", err)
	}
	return &DaggerheartService{stores: stores, applier: applier, seedFunc: seedFunc}, nil
}

func (s *DaggerheartService) resolvedApplier() (projection.Applier, error) {
	if s.applier.Adapters != nil ||
		s.applier.Campaign != nil ||
		s.applier.Character != nil ||
		s.applier.Session != nil ||
		s.applier.SessionGate != nil ||
		s.applier.SessionSpotlight != nil {
		return s.applier, nil
	}
	applier, err := s.stores.TryApplier()
	if err != nil {
		return projection.Applier{}, fmt.Errorf("build projection applier: %w", err)
	}
	s.applier = applier
	return s.applier, nil
}

func (s *DaggerheartService) mechanicsHandler() *mechanicstransport.Handler {
	return mechanicstransport.NewHandler(s.seedFunc)
}

func (s *DaggerheartService) ActionRoll(ctx context.Context, in *pb.ActionRollRequest) (*pb.ActionRollResponse, error) {
	return s.mechanicsHandler().ActionRoll(ctx, in)
}

func (s *DaggerheartService) DualityOutcome(ctx context.Context, in *pb.DualityOutcomeRequest) (*pb.DualityOutcomeResponse, error) {
	return s.mechanicsHandler().DualityOutcome(ctx, in)
}

func (s *DaggerheartService) DualityExplain(ctx context.Context, in *pb.DualityExplainRequest) (*pb.DualityExplainResponse, error) {
	return s.mechanicsHandler().DualityExplain(ctx, in)
}

func (s *DaggerheartService) DualityProbability(ctx context.Context, in *pb.DualityProbabilityRequest) (*pb.DualityProbabilityResponse, error) {
	return s.mechanicsHandler().DualityProbability(ctx, in)
}

func (s *DaggerheartService) RulesVersion(ctx context.Context, in *pb.RulesVersionRequest) (*pb.RulesVersionResponse, error) {
	return s.mechanicsHandler().RulesVersion(ctx, in)
}

func (s *DaggerheartService) RollDice(ctx context.Context, in *pb.RollDiceRequest) (*pb.RollDiceResponse, error) {
	return s.mechanicsHandler().RollDice(ctx, in)
}
