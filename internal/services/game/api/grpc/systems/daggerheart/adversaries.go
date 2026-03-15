package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func (s *DaggerheartService) CreateAdversary(ctx context.Context, in *pb.DaggerheartCreateAdversaryRequest) (*pb.DaggerheartCreateAdversaryResponse, error) {
	return s.adversaryHandler().CreateAdversary(ctx, in)
}

func (s *DaggerheartService) UpdateAdversary(ctx context.Context, in *pb.DaggerheartUpdateAdversaryRequest) (*pb.DaggerheartUpdateAdversaryResponse, error) {
	return s.adversaryHandler().UpdateAdversary(ctx, in)
}

func (s *DaggerheartService) DeleteAdversary(ctx context.Context, in *pb.DaggerheartDeleteAdversaryRequest) (*pb.DaggerheartDeleteAdversaryResponse, error) {
	return s.adversaryHandler().DeleteAdversary(ctx, in)
}

func (s *DaggerheartService) GetAdversary(ctx context.Context, in *pb.DaggerheartGetAdversaryRequest) (*pb.DaggerheartGetAdversaryResponse, error) {
	return s.adversaryHandler().GetAdversary(ctx, in)
}

func (s *DaggerheartService) ListAdversaries(ctx context.Context, in *pb.DaggerheartListAdversariesRequest) (*pb.DaggerheartListAdversariesResponse, error) {
	return s.adversaryHandler().ListAdversaries(ctx, in)
}
