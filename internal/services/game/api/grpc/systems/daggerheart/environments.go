package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func (s *DaggerheartService) CreateEnvironmentEntity(ctx context.Context, in *pb.DaggerheartCreateEnvironmentEntityRequest) (*pb.DaggerheartCreateEnvironmentEntityResponse, error) {
	return s.environmentHandler().CreateEnvironmentEntity(ctx, in)
}

func (s *DaggerheartService) UpdateEnvironmentEntity(ctx context.Context, in *pb.DaggerheartUpdateEnvironmentEntityRequest) (*pb.DaggerheartUpdateEnvironmentEntityResponse, error) {
	return s.environmentHandler().UpdateEnvironmentEntity(ctx, in)
}

func (s *DaggerheartService) DeleteEnvironmentEntity(ctx context.Context, in *pb.DaggerheartDeleteEnvironmentEntityRequest) (*pb.DaggerheartDeleteEnvironmentEntityResponse, error) {
	return s.environmentHandler().DeleteEnvironmentEntity(ctx, in)
}

func (s *DaggerheartService) GetEnvironmentEntity(ctx context.Context, in *pb.DaggerheartGetEnvironmentEntityRequest) (*pb.DaggerheartGetEnvironmentEntityResponse, error) {
	return s.environmentHandler().GetEnvironmentEntity(ctx, in)
}

func (s *DaggerheartService) ListEnvironmentEntities(ctx context.Context, in *pb.DaggerheartListEnvironmentEntitiesRequest) (*pb.DaggerheartListEnvironmentEntitiesResponse, error) {
	return s.environmentHandler().ListEnvironmentEntities(ctx, in)
}
