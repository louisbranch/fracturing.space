package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
)

func TestListAdversaries_DescOrder(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListAdversaries(context.Background(), &pb.ListDaggerheartAdversariesRequest{
		OrderBy: "name desc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetAdversaries()) != 1 {
		t.Fatalf("adversaries = %d, want 1", len(resp.GetAdversaries()))
	}
}

func TestListBeastforms_WithFilter(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListBeastforms(context.Background(), &pb.ListDaggerheartBeastformsRequest{
		Filter: `name = "Wolf"`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetBeastforms()) != 1 {
		t.Fatalf("beastforms = %d, want 1", len(resp.GetBeastforms()))
	}
}

func TestListCompanionExperiences_DescOrder(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListCompanionExperiences(context.Background(), &pb.ListDaggerheartCompanionExperiencesRequest{
		OrderBy: "name desc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetExperiences()) != 1 {
		t.Fatalf("companion experiences = %d, want 1", len(resp.GetExperiences()))
	}
}

func TestListCompendiumCreatureEndpoints_NilRequests(t *testing.T) {
	svc := newContentTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"ListAdversaries", func() error { _, err := svc.ListAdversaries(ctx, nil); return err }},
		{"ListBeastforms", func() error { _, err := svc.ListBeastforms(ctx, nil); return err }},
		{"ListCompanionExperiences", func() error { _, err := svc.ListCompanionExperiences(ctx, nil); return err }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			grpcassert.StatusCode(t, tc.fn(), codes.InvalidArgument)
		})
	}
}

func TestListCompendiumCreatureEndpoints_NoStore(t *testing.T) {
	svc := &DaggerheartContentService{}
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"ListAdversaries", func() error { _, err := svc.ListAdversaries(ctx, &pb.ListDaggerheartAdversariesRequest{}); return err }},
		{"ListBeastforms", func() error { _, err := svc.ListBeastforms(ctx, &pb.ListDaggerheartBeastformsRequest{}); return err }},
		{"ListCompanionExperiences", func() error {
			_, err := svc.ListCompanionExperiences(ctx, &pb.ListDaggerheartCompanionExperiencesRequest{})
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			grpcassert.StatusCode(t, tc.fn(), codes.Internal)
		})
	}
}

func TestGetCompendiumCreatureEndpoints_NilRequests(t *testing.T) {
	svc := newContentTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetAdversary", func() error { _, err := svc.GetAdversary(ctx, nil); return err }},
		{"GetBeastform", func() error { _, err := svc.GetBeastform(ctx, nil); return err }},
		{"GetCompanionExperience", func() error { _, err := svc.GetCompanionExperience(ctx, nil); return err }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			grpcassert.StatusCode(t, tc.fn(), codes.InvalidArgument)
		})
	}
}

func TestGetCompendiumCreatureEndpoints_NoStore(t *testing.T) {
	svc := &DaggerheartContentService{}
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetAdversary", func() error { _, err := svc.GetAdversary(ctx, &pb.GetDaggerheartAdversaryRequest{Id: "x"}); return err }},
		{"GetBeastform", func() error { _, err := svc.GetBeastform(ctx, &pb.GetDaggerheartBeastformRequest{Id: "x"}); return err }},
		{"GetCompanionExperience", func() error {
			_, err := svc.GetCompanionExperience(ctx, &pb.GetDaggerheartCompanionExperienceRequest{Id: "x"})
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			grpcassert.StatusCode(t, tc.fn(), codes.Internal)
		})
	}
}

func TestGetCompendiumCreatureEndpoints_EmptyID(t *testing.T) {
	svc := newContentTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetAdversary", func() error { _, err := svc.GetAdversary(ctx, &pb.GetDaggerheartAdversaryRequest{Id: ""}); return err }},
		{"GetBeastform", func() error { _, err := svc.GetBeastform(ctx, &pb.GetDaggerheartBeastformRequest{Id: ""}); return err }},
		{"GetCompanionExperience", func() error {
			_, err := svc.GetCompanionExperience(ctx, &pb.GetDaggerheartCompanionExperienceRequest{Id: ""})
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			grpcassert.StatusCode(t, tc.fn(), codes.InvalidArgument)
		})
	}
}

func TestGetCompendiumCreatureEndpoints_NotFound(t *testing.T) {
	svc := newContentTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetAdversary", func() error {
			_, err := svc.GetAdversary(ctx, &pb.GetDaggerheartAdversaryRequest{Id: "missing"})
			return err
		}},
		{"GetBeastform", func() error {
			_, err := svc.GetBeastform(ctx, &pb.GetDaggerheartBeastformRequest{Id: "missing"})
			return err
		}},
		{"GetCompanionExperience", func() error {
			_, err := svc.GetCompanionExperience(ctx, &pb.GetDaggerheartCompanionExperienceRequest{Id: "missing"})
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			grpcassert.StatusCode(t, tc.fn(), codes.NotFound)
		})
	}
}
