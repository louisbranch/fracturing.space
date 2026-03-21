package creationworkflow

import (
	"context"
	"testing"

	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	daggerheartprofile "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/profile"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/projectionstore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestApplyClassSubclassInput(t *testing.T) {
	profile := &projectionstore.DaggerheartCharacterProfile{}
	err := applyClassSubclassInput(
		context.Background(),
		newTestContentStore(),
		profile,
		&daggerheartv1.DaggerheartCreationStepClassSubclassInput{ClassId: "class-1", SubclassId: "subclass-1"},
	)
	if err != nil {
		t.Fatalf("applyClassSubclassInput() error = %v", err)
	}
	if profile.ClassID != "class-1" || profile.SubclassID != "subclass-1" {
		t.Fatalf("profile class/subclass = (%q, %q), want (%q, %q)", profile.ClassID, profile.SubclassID, "class-1", "subclass-1")
	}
}

func TestApplyHeritageInputRejectsWrongKind(t *testing.T) {
	err := applyHeritageInput(
		context.Background(),
		newTestContentStore(),
		&projectionstore.DaggerheartCharacterProfile{},
		&daggerheartv1.DaggerheartCreationStepHeritageInput{
			Heritage: &daggerheartv1.DaggerheartCreationStepHeritageSelectionInput{
				FirstFeatureAncestryId:  "community-1",
				SecondFeatureAncestryId: "community-1",
				CommunityId:             "community-1",
			},
		},
	)
	if err == nil {
		t.Fatal("applyHeritageInput() error = nil, want failure")
	}
	if status.Code(err) != codes.InvalidArgument {
		t.Fatalf("status.Code(err) = %v, want %v", status.Code(err), codes.InvalidArgument)
	}
}

func TestApplyDetailsInputPopulatesClassDerivedStats(t *testing.T) {
	profile := &projectionstore.DaggerheartCharacterProfile{ClassID: "class-1"}
	err := applyDetailsInput(
		context.Background(),
		newTestContentStore(),
		profile,
		&daggerheartv1.DaggerheartCreationStepDetailsInput{Description: "Steady shieldbearer"},
	)
	if err != nil {
		t.Fatalf("applyDetailsInput() error = %v", err)
	}
	if profile.Level != daggerheartprofile.PCLevelDefault {
		t.Fatalf("profile.Level = %d, want %d", profile.Level, daggerheartprofile.PCLevelDefault)
	}
	if profile.HpMax != 7 || profile.Evasion != 11 {
		t.Fatalf("profile combat stats = (hp=%d, evasion=%d), want (7, 11)", profile.HpMax, profile.Evasion)
	}
	if profile.MajorThreshold != 1 || profile.SevereThreshold != 2 {
		t.Fatalf("profile thresholds = (%d, %d), want (1, 2)", profile.MajorThreshold, profile.SevereThreshold)
	}
	if !profile.DetailsRecorded || profile.Description != "Steady shieldbearer" {
		t.Fatalf("profile details = recorded:%v description:%q", profile.DetailsRecorded, profile.Description)
	}
}
