package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc/codes"
)

func TestListClasses_DescOrder(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListClasses(context.Background(), &pb.ListDaggerheartClassesRequest{
		OrderBy: "name desc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	classes := resp.GetClasses()
	if len(classes) != 2 {
		t.Fatalf("classes = %d, want 2", len(classes))
	}
	if classes[0].GetName() != "Sorcerer" {
		t.Errorf("first class = %q, want Sorcerer", classes[0].GetName())
	}
}

func TestListClasses_WithFilter(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListClasses(context.Background(), &pb.ListDaggerheartClassesRequest{
		Filter: `name = "Guardian"`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetClasses()) != 1 {
		t.Fatalf("classes = %d, want 1", len(resp.GetClasses()))
	}
	if resp.GetClasses()[0].GetName() != "Guardian" {
		t.Errorf("class = %q, want Guardian", resp.GetClasses()[0].GetName())
	}
}

func TestListClasses_InvalidFilter(t *testing.T) {
	svc := newContentTestService()
	_, err := svc.ListClasses(context.Background(), &pb.ListDaggerheartClassesRequest{
		Filter: "invalid @@@ filter",
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestListClasses_InvalidOrderBy(t *testing.T) {
	svc := newContentTestService()
	_, err := svc.ListClasses(context.Background(), &pb.ListDaggerheartClassesRequest{
		OrderBy: "unknown_column",
	})
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
}

func TestListClasses_PaginationSecondPage(t *testing.T) {
	svc := newContentTestService()
	firstPage, err := svc.ListClasses(context.Background(), &pb.ListDaggerheartClassesRequest{
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("first page: %v", err)
	}
	if firstPage.GetNextPageToken() == "" {
		t.Fatal("expected next_page_token")
	}

	secondPage, err := svc.ListClasses(context.Background(), &pb.ListDaggerheartClassesRequest{
		PageSize:  1,
		PageToken: firstPage.GetNextPageToken(),
	})
	if err != nil {
		t.Fatalf("second page: %v", err)
	}
	if len(secondPage.GetClasses()) != 1 {
		t.Errorf("classes on second page = %d, want 1", len(secondPage.GetClasses()))
	}
	if secondPage.GetClasses()[0].GetId() == firstPage.GetClasses()[0].GetId() {
		t.Error("second page returned same class as first page")
	}
}

func TestListSubclasses_WithFilter(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListSubclasses(context.Background(), &pb.ListDaggerheartSubclassesRequest{
		Filter: `name = "Bladeweaver"`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetSubclasses()) != 1 {
		t.Fatalf("subclasses = %d, want 1", len(resp.GetSubclasses()))
	}
}

func TestListHeritages_DescOrder(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListHeritages(context.Background(), &pb.ListDaggerheartHeritagesRequest{
		OrderBy: "name desc",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetHeritages()) != 1 {
		t.Fatalf("heritages = %d, want 1", len(resp.GetHeritages()))
	}
}

func TestListExperiences_WithFilter(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListExperiences(context.Background(), &pb.ListDaggerheartExperiencesRequest{
		Filter: `name = "Wanderer"`,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetExperiences()) != 1 {
		t.Fatalf("experiences = %d, want 1", len(resp.GetExperiences()))
	}
}

func TestListReferenceContentEndpoints_NilRequests(t *testing.T) {
	svc := newContentTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"ListClasses", func() error { _, err := svc.ListClasses(ctx, nil); return err }},
		{"ListSubclasses", func() error { _, err := svc.ListSubclasses(ctx, nil); return err }},
		{"ListHeritages", func() error { _, err := svc.ListHeritages(ctx, nil); return err }},
		{"ListExperiences", func() error { _, err := svc.ListExperiences(ctx, nil); return err }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			grpcassert.StatusCode(t, tc.fn(), codes.InvalidArgument)
		})
	}
}

func TestListReferenceContentEndpoints_NoStore(t *testing.T) {
	svc := &DaggerheartContentService{}
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"ListClasses", func() error { _, err := svc.ListClasses(ctx, &pb.ListDaggerheartClassesRequest{}); return err }},
		{"ListSubclasses", func() error { _, err := svc.ListSubclasses(ctx, &pb.ListDaggerheartSubclassesRequest{}); return err }},
		{"ListHeritages", func() error { _, err := svc.ListHeritages(ctx, &pb.ListDaggerheartHeritagesRequest{}); return err }},
		{"ListExperiences", func() error { _, err := svc.ListExperiences(ctx, &pb.ListDaggerheartExperiencesRequest{}); return err }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			grpcassert.StatusCode(t, tc.fn(), codes.Internal)
		})
	}
}

func TestGetReferenceContentEndpoints_NilRequests(t *testing.T) {
	svc := newContentTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetClass", func() error { _, err := svc.GetClass(ctx, nil); return err }},
		{"GetSubclass", func() error { _, err := svc.GetSubclass(ctx, nil); return err }},
		{"GetHeritage", func() error { _, err := svc.GetHeritage(ctx, nil); return err }},
		{"GetExperience", func() error { _, err := svc.GetExperience(ctx, nil); return err }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			grpcassert.StatusCode(t, tc.fn(), codes.InvalidArgument)
		})
	}
}

func TestGetReferenceContentEndpoints_NoStore(t *testing.T) {
	svc := &DaggerheartContentService{}
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetClass", func() error { _, err := svc.GetClass(ctx, &pb.GetDaggerheartClassRequest{Id: "x"}); return err }},
		{"GetSubclass", func() error { _, err := svc.GetSubclass(ctx, &pb.GetDaggerheartSubclassRequest{Id: "x"}); return err }},
		{"GetHeritage", func() error { _, err := svc.GetHeritage(ctx, &pb.GetDaggerheartHeritageRequest{Id: "x"}); return err }},
		{"GetExperience", func() error {
			_, err := svc.GetExperience(ctx, &pb.GetDaggerheartExperienceRequest{Id: "x"})
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			grpcassert.StatusCode(t, tc.fn(), codes.Internal)
		})
	}
}

func TestGetReferenceContentEndpoints_EmptyID(t *testing.T) {
	svc := newContentTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetSubclass", func() error { _, err := svc.GetSubclass(ctx, &pb.GetDaggerheartSubclassRequest{Id: ""}); return err }},
		{"GetHeritage", func() error { _, err := svc.GetHeritage(ctx, &pb.GetDaggerheartHeritageRequest{Id: ""}); return err }},
		{"GetExperience", func() error {
			_, err := svc.GetExperience(ctx, &pb.GetDaggerheartExperienceRequest{Id: ""})
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			grpcassert.StatusCode(t, tc.fn(), codes.InvalidArgument)
		})
	}
}

func TestGetReferenceContentEndpoints_NotFound(t *testing.T) {
	svc := newContentTestService()
	ctx := context.Background()

	tests := []struct {
		name string
		fn   func() error
	}{
		{"GetSubclass", func() error {
			_, err := svc.GetSubclass(ctx, &pb.GetDaggerheartSubclassRequest{Id: "missing"})
			return err
		}},
		{"GetHeritage", func() error {
			_, err := svc.GetHeritage(ctx, &pb.GetDaggerheartHeritageRequest{Id: "missing"})
			return err
		}},
		{"GetExperience", func() error {
			_, err := svc.GetExperience(ctx, &pb.GetDaggerheartExperienceRequest{Id: "missing"})
			return err
		}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			grpcassert.StatusCode(t, tc.fn(), codes.NotFound)
		})
	}
}
