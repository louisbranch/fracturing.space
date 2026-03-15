package contenttransport

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/validate"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// GetAdversary returns a single Daggerheart adversary catalog entry.
func (a contentApplication) runGetAdversary(ctx context.Context, in *pb.GetDaggerheartAdversaryRequest) (*pb.GetDaggerheartAdversaryResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "adversary request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	if _, err := validate.RequiredID(in.GetId(), "adversary id"); err != nil {
		return nil, err
	}

	adversary, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), adversaryDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartAdversaryResponse{Adversary: adversary}, nil
}

// ListAdversaries returns Daggerheart adversary catalog entries.
func (a contentApplication) runListAdversaries(ctx context.Context, in *pb.ListDaggerheartAdversariesRequest) (*pb.ListDaggerheartAdversariesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list adversaries request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}

	adversaries, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), adversaryDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartAdversariesResponse{
		Adversaries:       adversaries,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetBeastform returns a single Daggerheart beastform catalog entry.
func (a contentApplication) runGetBeastform(ctx context.Context, in *pb.GetDaggerheartBeastformRequest) (*pb.GetDaggerheartBeastformResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "beastform request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	if _, err := validate.RequiredID(in.GetId(), "beastform id"); err != nil {
		return nil, err
	}

	beastform, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), beastformDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartBeastformResponse{Beastform: beastform}, nil
}

// ListBeastforms returns Daggerheart beastform catalog entries.
func (a contentApplication) runListBeastforms(ctx context.Context, in *pb.ListDaggerheartBeastformsRequest) (*pb.ListDaggerheartBeastformsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list beastforms request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}

	beastforms, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), beastformDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartBeastformsResponse{
		Beastforms:        beastforms,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}

// GetCompanionExperience returns a single Daggerheart companion experience catalog entry.
func (a contentApplication) runGetCompanionExperience(ctx context.Context, in *pb.GetDaggerheartCompanionExperienceRequest) (*pb.GetDaggerheartCompanionExperienceResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "companion experience request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}
	if _, err := validate.RequiredID(in.GetId(), "companion experience id"); err != nil {
		return nil, err
	}

	experience, err := getContentEntry(ctx, store, in.GetId(), in.GetLocale(), companionExperienceDescriptor)
	if err != nil {
		return nil, err
	}
	return &pb.GetDaggerheartCompanionExperienceResponse{Experience: experience}, nil
}

// ListCompanionExperiences returns Daggerheart companion experience catalog entries.
func (a contentApplication) runListCompanionExperiences(ctx context.Context, in *pb.ListDaggerheartCompanionExperiencesRequest) (*pb.ListDaggerheartCompanionExperiencesResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list companion experiences request is required")
	}
	store, err := a.handler.contentStore()
	if err != nil {
		return nil, err
	}

	experiences, page, err := listContentEntries(ctx, store, newContentListRequest(in), in.GetLocale(), companionExperienceDescriptor)
	if err != nil {
		return nil, err
	}

	return &pb.ListDaggerheartCompanionExperiencesResponse{
		Experiences:       experiences,
		NextPageToken:     page.NextPageToken,
		PreviousPageToken: page.PreviousPageToken,
		TotalSize:         int32(page.TotalSize),
	}, nil
}
