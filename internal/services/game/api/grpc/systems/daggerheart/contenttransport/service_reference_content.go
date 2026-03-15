package contenttransport

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func (h *Handler) GetClass(ctx context.Context, in *pb.GetDaggerheartClassRequest) (*pb.GetDaggerheartClassResponse, error) {
	return newContentApplication(h).runGetClass(ctx, in)
}

func (h *Handler) ListClasses(ctx context.Context, in *pb.ListDaggerheartClassesRequest) (*pb.ListDaggerheartClassesResponse, error) {
	return newContentApplication(h).runListClasses(ctx, in)
}

func (h *Handler) GetSubclass(ctx context.Context, in *pb.GetDaggerheartSubclassRequest) (*pb.GetDaggerheartSubclassResponse, error) {
	return newContentApplication(h).runGetSubclass(ctx, in)
}

func (h *Handler) ListSubclasses(ctx context.Context, in *pb.ListDaggerheartSubclassesRequest) (*pb.ListDaggerheartSubclassesResponse, error) {
	return newContentApplication(h).runListSubclasses(ctx, in)
}

func (h *Handler) GetHeritage(ctx context.Context, in *pb.GetDaggerheartHeritageRequest) (*pb.GetDaggerheartHeritageResponse, error) {
	return newContentApplication(h).runGetHeritage(ctx, in)
}

func (h *Handler) ListHeritages(ctx context.Context, in *pb.ListDaggerheartHeritagesRequest) (*pb.ListDaggerheartHeritagesResponse, error) {
	return newContentApplication(h).runListHeritages(ctx, in)
}

func (h *Handler) GetExperience(ctx context.Context, in *pb.GetDaggerheartExperienceRequest) (*pb.GetDaggerheartExperienceResponse, error) {
	return newContentApplication(h).runGetExperience(ctx, in)
}

func (h *Handler) ListExperiences(ctx context.Context, in *pb.ListDaggerheartExperiencesRequest) (*pb.ListDaggerheartExperiencesResponse, error) {
	return newContentApplication(h).runListExperiences(ctx, in)
}
