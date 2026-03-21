package creationworkflow

import (
	"context"
	"testing"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestApplyExperiencesInput(t *testing.T) {
	profile := &projectionstore.DaggerheartCharacterProfile{}
	err := applyExperiencesInput(profile, &daggerheartv1.DaggerheartCreationStepExperiencesInput{
		Experiences: []*daggerheartv1.DaggerheartExperience{
			{Name: "Trailblazer"},
			{Name: "Archivist"},
		},
	})
	if err != nil {
		t.Fatalf("applyExperiencesInput() error = %v", err)
	}
	if len(profile.Experiences) != 2 {
		t.Fatalf("len(profile.Experiences) = %d, want 2", len(profile.Experiences))
	}
	if profile.Experiences[0].Modifier != 2 || profile.Experiences[1].Modifier != 2 {
		t.Fatalf("profile.Experiences modifiers = %+v, want both 2", profile.Experiences)
	}
}

func TestApplyDomainCardsInput(t *testing.T) {
	profile := &projectionstore.DaggerheartCharacterProfile{ClassID: "class-1"}
	err := applyDomainCardsInput(
		context.Background(),
		newTestContentStore(),
		profile,
		&daggerheartv1.DaggerheartCreationStepDomainCardsInput{DomainCardIds: []string{"card-1", "card-2"}},
	)
	if err != nil {
		t.Fatalf("applyDomainCardsInput() error = %v", err)
	}
	if len(profile.DomainCardIDs) != 2 || profile.DomainCardIDs[0] != "card-1" || profile.DomainCardIDs[1] != "card-2" {
		t.Fatalf("profile.DomainCardIDs = %v, want [card-1 card-2]", profile.DomainCardIDs)
	}
}

func TestApplyDomainCardsInputRequiresClass(t *testing.T) {
	err := applyDomainCardsInput(
		context.Background(),
		newTestContentStore(),
		&projectionstore.DaggerheartCharacterProfile{},
		&daggerheartv1.DaggerheartCreationStepDomainCardsInput{DomainCardIds: []string{"card-1", "card-2"}},
	)
	if err == nil {
		t.Fatal("applyDomainCardsInput() error = nil, want failure")
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), codes.FailedPrecondition)
	}
}

func TestApplyConnectionsInput(t *testing.T) {
	profile := &projectionstore.DaggerheartCharacterProfile{}
	err := applyConnectionsInput(profile, &daggerheartv1.DaggerheartCreationStepConnectionsInput{Connections: "We survived the siege together."})
	if err != nil {
		t.Fatalf("applyConnectionsInput() error = %v", err)
	}
	if profile.Connections != "We survived the siege together." {
		t.Fatalf("profile.Connections = %q, want %q", profile.Connections, "We survived the siege together.")
	}
}
