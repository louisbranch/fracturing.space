package daggerheart

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func (s *DaggerheartContentService) GetClass(ctx context.Context, in *pb.GetDaggerheartClassRequest) (*pb.GetDaggerheartClassResponse, error) {
	return s.handler().GetClass(ctx, in)
}

func (s *DaggerheartContentService) ListClasses(ctx context.Context, in *pb.ListDaggerheartClassesRequest) (*pb.ListDaggerheartClassesResponse, error) {
	return s.handler().ListClasses(ctx, in)
}

func (s *DaggerheartContentService) GetSubclass(ctx context.Context, in *pb.GetDaggerheartSubclassRequest) (*pb.GetDaggerheartSubclassResponse, error) {
	return s.handler().GetSubclass(ctx, in)
}

func (s *DaggerheartContentService) ListSubclasses(ctx context.Context, in *pb.ListDaggerheartSubclassesRequest) (*pb.ListDaggerheartSubclassesResponse, error) {
	return s.handler().ListSubclasses(ctx, in)
}

func (s *DaggerheartContentService) GetHeritage(ctx context.Context, in *pb.GetDaggerheartHeritageRequest) (*pb.GetDaggerheartHeritageResponse, error) {
	return s.handler().GetHeritage(ctx, in)
}

func (s *DaggerheartContentService) ListHeritages(ctx context.Context, in *pb.ListDaggerheartHeritagesRequest) (*pb.ListDaggerheartHeritagesResponse, error) {
	return s.handler().ListHeritages(ctx, in)
}

func (s *DaggerheartContentService) GetExperience(ctx context.Context, in *pb.GetDaggerheartExperienceRequest) (*pb.GetDaggerheartExperienceResponse, error) {
	return s.handler().GetExperience(ctx, in)
}

func (s *DaggerheartContentService) ListExperiences(ctx context.Context, in *pb.ListDaggerheartExperiencesRequest) (*pb.ListDaggerheartExperiencesResponse, error) {
	return s.handler().ListExperiences(ctx, in)
}
