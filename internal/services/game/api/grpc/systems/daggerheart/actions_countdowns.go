package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func (s *DaggerheartService) CreateCountdown(ctx context.Context, in *pb.DaggerheartCreateCountdownRequest) (*pb.DaggerheartCreateCountdownResponse, error) {
	return newCountdownApplication(s).runCreateCountdown(ctx, in)
}

func (s *DaggerheartService) UpdateCountdown(ctx context.Context, in *pb.DaggerheartUpdateCountdownRequest) (*pb.DaggerheartUpdateCountdownResponse, error) {
	return newCountdownApplication(s).runUpdateCountdown(ctx, in)
}

func (s *DaggerheartService) DeleteCountdown(ctx context.Context, in *pb.DaggerheartDeleteCountdownRequest) (*pb.DaggerheartDeleteCountdownResponse, error) {
	return newCountdownApplication(s).runDeleteCountdown(ctx, in)
}

func (s *DaggerheartService) ResolveBlazeOfGlory(ctx context.Context, in *pb.DaggerheartResolveBlazeOfGloryRequest) (*pb.DaggerheartResolveBlazeOfGloryResponse, error) {
	return newCountdownApplication(s).runResolveBlazeOfGlory(ctx, in)
}
