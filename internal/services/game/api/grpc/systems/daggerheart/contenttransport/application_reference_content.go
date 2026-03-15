package contenttransport

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetClass returns a single Daggerheart class.
func (a contentApplication) runGetClass(ctx context.Context, in *pb.GetDaggerheartClassRequest) (*pb.GetDaggerheartClassResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "class request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	if _, err := validate.RequiredID(in.GetId(), "class id"); err != nil {
		return nil, err
	}

	class, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), classDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartClassResponse{Class: class}, nil
}

// ListClasses returns Daggerheart classes.
func (a contentApplication) runListClasses(ctx context.Context, in *pb.ListDaggerheartClassesRequest) (*pb.ListDaggerheartClassesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list classes request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	classes, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), classDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartClassesResponse{
		Classes:           classes,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetSubclass returns a single Daggerheart subclass.
func (a contentApplication) runGetSubclass(ctx context.Context, in *pb.GetDaggerheartSubclassRequest) (*pb.GetDaggerheartSubclassResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "subclass request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	if _, err := validate.RequiredID(in.GetId(), "subclass id"); err != nil {
		return nil, err
	}

	subclass, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), subclassDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartSubclassResponse{Subclass: subclass}, nil
}

// ListSubclasses returns Daggerheart subclasses.
func (a contentApplication) runListSubclasses(ctx context.Context, in *pb.ListDaggerheartSubclassesRequest) (*pb.ListDaggerheartSubclassesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list subclasses request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	subclasses, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), subclassDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartSubclassesResponse{
		Subclasses:        subclasses,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetHeritage returns a single Daggerheart heritage.
func (a contentApplication) runGetHeritage(ctx context.Context, in *pb.GetDaggerheartHeritageRequest) (*pb.GetDaggerheartHeritageResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "heritage request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	if _, err := validate.RequiredID(in.GetId(), "heritage id"); err != nil {
		return nil, err
	}

	heritage, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), heritageDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartHeritageResponse{Heritage: heritage}, nil
}

// ListHeritages returns Daggerheart heritages.
func (a contentApplication) runListHeritages(ctx context.Context, in *pb.ListDaggerheartHeritagesRequest) (*pb.ListDaggerheartHeritagesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list heritages request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	heritages, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), heritageDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartHeritagesResponse{
		Heritages:         heritages,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetExperience returns a single Daggerheart experience.
func (a contentApplication) runGetExperience(ctx context.Context, in *pb.GetDaggerheartExperienceRequest) (*pb.GetDaggerheartExperienceResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "experience request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	if _, err := validate.RequiredID(in.GetId(), "experience id"); err != nil {
		return nil, err
	}

	experience, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), experienceDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartExperienceResponse{Experience: experience}, nil
}

// ListExperiences returns Daggerheart experiences.
func (a contentApplication) runListExperiences(ctx context.Context, in *pb.ListDaggerheartExperiencesRequest) (*pb.ListDaggerheartExperiencesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list experiences request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	experiences, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), experienceDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartExperiencesResponse{
		Experiences:       experiences,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}
