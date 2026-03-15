package contenttransport

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func TestHandlerReferenceContentEndpoints(t *testing.T) {
	ctx := context.Background()
	handler := NewHandler(newFakeContentStore())

	tests := []struct {
		name string
		run  func(*testing.T)
	}{
		{
			name: "GetClass",
			run: func(t *testing.T) {
				resp, err := handler.GetClass(ctx, &pb.GetDaggerheartClassRequest{Id: "class-1"})
				if err != nil || resp.GetClass().GetId() != "class-1" {
					t.Fatalf("GetClass: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "ListClasses",
			run: func(t *testing.T) {
				resp, err := handler.ListClasses(ctx, &pb.ListDaggerheartClassesRequest{})
				if err != nil || len(resp.GetClasses()) != 1 {
					t.Fatalf("ListClasses: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "GetSubclass",
			run: func(t *testing.T) {
				resp, err := handler.GetSubclass(ctx, &pb.GetDaggerheartSubclassRequest{Id: "sub-1"})
				if err != nil || resp.GetSubclass().GetId() != "sub-1" {
					t.Fatalf("GetSubclass: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "ListSubclasses",
			run: func(t *testing.T) {
				resp, err := handler.ListSubclasses(ctx, &pb.ListDaggerheartSubclassesRequest{})
				if err != nil || len(resp.GetSubclasses()) != 1 {
					t.Fatalf("ListSubclasses: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "GetHeritage",
			run: func(t *testing.T) {
				resp, err := handler.GetHeritage(ctx, &pb.GetDaggerheartHeritageRequest{Id: "her-1"})
				if err != nil || resp.GetHeritage().GetId() != "her-1" {
					t.Fatalf("GetHeritage: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "ListHeritages",
			run: func(t *testing.T) {
				resp, err := handler.ListHeritages(ctx, &pb.ListDaggerheartHeritagesRequest{})
				if err != nil || len(resp.GetHeritages()) != 1 {
					t.Fatalf("ListHeritages: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "GetExperience",
			run: func(t *testing.T) {
				resp, err := handler.GetExperience(ctx, &pb.GetDaggerheartExperienceRequest{Id: "exp-1"})
				if err != nil || resp.GetExperience().GetId() != "exp-1" {
					t.Fatalf("GetExperience: resp=%v err=%v", resp, err)
				}
			},
		},
		{
			name: "ListExperiences",
			run: func(t *testing.T) {
				resp, err := handler.ListExperiences(ctx, &pb.ListDaggerheartExperiencesRequest{})
				if err != nil || len(resp.GetExperiences()) != 1 {
					t.Fatalf("ListExperiences: resp=%v err=%v", resp, err)
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, tc.run)
	}
}
