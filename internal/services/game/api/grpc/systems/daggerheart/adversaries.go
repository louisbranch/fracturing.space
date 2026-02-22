package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func (s *DaggerheartService) CreateAdversary(ctx context.Context, in *pb.DaggerheartCreateAdversaryRequest) (*pb.DaggerheartCreateAdversaryResponse, error) {
	return newAdversaryApplication(s).runCreateAdversary(ctx, in)
}

func (s *DaggerheartService) UpdateAdversary(ctx context.Context, in *pb.DaggerheartUpdateAdversaryRequest) (*pb.DaggerheartUpdateAdversaryResponse, error) {
	return newAdversaryApplication(s).runUpdateAdversary(ctx, in)
}

func (s *DaggerheartService) DeleteAdversary(ctx context.Context, in *pb.DaggerheartDeleteAdversaryRequest) (*pb.DaggerheartDeleteAdversaryResponse, error) {
	return newAdversaryApplication(s).runDeleteAdversary(ctx, in)
}

func (s *DaggerheartService) GetAdversary(ctx context.Context, in *pb.DaggerheartGetAdversaryRequest) (*pb.DaggerheartGetAdversaryResponse, error) {
	return newAdversaryApplication(s).runGetAdversary(ctx, in)
}

func (s *DaggerheartService) ListAdversaries(ctx context.Context, in *pb.DaggerheartListAdversariesRequest) (*pb.DaggerheartListAdversariesResponse, error) {
	return newAdversaryApplication(s).runListAdversaries(ctx, in)
}
