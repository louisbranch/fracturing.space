package daggerheart

import (
	"context"
	"testing"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
)

func TestGetContentAdversary_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetAdversary(context.Background(), &pb.GetDaggerheartAdversaryRequest{Id: "adv-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetAdversary().GetName() != "Goblin" {
		t.Errorf("name = %q, want Goblin", resp.GetAdversary().GetName())
	}
}

func TestListContentAdversaries_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListAdversaries(context.Background(), &pb.ListDaggerheartAdversariesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetAdversaries()) != 1 {
		t.Errorf("adversaries = %d, want 1", len(resp.GetAdversaries()))
	}
}

func TestGetBeastform_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetBeastform(context.Background(), &pb.GetDaggerheartBeastformRequest{Id: "beast-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetBeastform().GetName() != "Wolf" {
		t.Errorf("name = %q, want Wolf", resp.GetBeastform().GetName())
	}
}

func TestListBeastforms_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListBeastforms(context.Background(), &pb.ListDaggerheartBeastformsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetBeastforms()) != 1 {
		t.Errorf("beastforms = %d, want 1", len(resp.GetBeastforms()))
	}
}

func TestGetCompanionExperience_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetCompanionExperience(context.Background(), &pb.GetDaggerheartCompanionExperienceRequest{Id: "cexp-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetExperience().GetName() != "Guard" {
		t.Errorf("name = %q, want Guard", resp.GetExperience().GetName())
	}
}

func TestListCompanionExperiences_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListCompanionExperiences(context.Background(), &pb.ListDaggerheartCompanionExperiencesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetExperiences()) != 1 {
		t.Errorf("companion experiences = %d, want 1", len(resp.GetExperiences()))
	}
}
