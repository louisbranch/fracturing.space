package daggerheart

import (
	"context"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/i18n"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/contentstore"
	"google.golang.org/grpc/codes"
)

func TestGetClass_NilRequest(t *testing.T) {
	svc := newContentTestService()
	_, err := svc.GetClass(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetClass_EmptyID(t *testing.T) {
	svc := newContentTestService()
	_, err := svc.GetClass(context.Background(), &pb.GetDaggerheartClassRequest{Id: ""})
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetClass_NotFound(t *testing.T) {
	svc := newContentTestService()
	_, err := svc.GetClass(context.Background(), &pb.GetDaggerheartClassRequest{Id: "nonexistent"})
	assertStatusCode(t, err, codes.NotFound)
}

func TestGetClass_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetClass(context.Background(), &pb.GetDaggerheartClassRequest{Id: "class-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetClass().GetName() != "Guardian" {
		t.Errorf("name = %q, want Guardian", resp.GetClass().GetName())
	}
}

func TestGetClass_LocaleOverride(t *testing.T) {
	svc := newContentTestService()
	store, ok := svc.store.(*fakeContentStore)
	if !ok {
		t.Fatalf("expected fake content store, got %T", svc.store)
	}
	locale := i18n.LocaleString(commonv1.Locale_LOCALE_PT_BR)
	if err := store.PutDaggerheartContentString(context.Background(), contentstore.DaggerheartContentString{
		ContentID:   "class-1",
		ContentType: "class",
		Field:       "name",
		Locale:      locale,
		Text:        "Guardiao",
	}); err != nil {
		t.Fatalf("put content string: %v", err)
	}

	resp, err := svc.GetClass(context.Background(), &pb.GetDaggerheartClassRequest{
		Id:     "class-1",
		Locale: commonv1.Locale_LOCALE_PT_BR,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetClass().GetName() != "Guardiao" {
		t.Errorf("name = %q, want Guardiao", resp.GetClass().GetName())
	}
}

func TestListClasses_NilRequest(t *testing.T) {
	svc := newContentTestService()
	_, err := svc.ListClasses(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestListClasses_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListClasses(context.Background(), &pb.ListDaggerheartClassesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetClasses()) != 2 {
		t.Errorf("classes = %d, want 2", len(resp.GetClasses()))
	}
	if resp.GetTotalSize() != 2 {
		t.Errorf("total_size = %d, want 2", resp.GetTotalSize())
	}
}

func TestListClasses_WithPagination(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListClasses(context.Background(), &pb.ListDaggerheartClassesRequest{
		PageSize: 1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetClasses()) != 1 {
		t.Errorf("classes = %d, want 1", len(resp.GetClasses()))
	}
	if resp.GetNextPageToken() == "" {
		t.Error("expected next_page_token")
	}
}

func TestGetSubclass_NilRequest(t *testing.T) {
	svc := newContentTestService()
	_, err := svc.GetSubclass(context.Background(), nil)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestGetSubclass_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetSubclass(context.Background(), &pb.GetDaggerheartSubclassRequest{Id: "sub-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetSubclass().GetName() != "Bladeweaver" {
		t.Errorf("name = %q, want Bladeweaver", resp.GetSubclass().GetName())
	}
}

func TestListSubclasses_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListSubclasses(context.Background(), &pb.ListDaggerheartSubclassesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetSubclasses()) != 1 {
		t.Errorf("subclasses = %d, want 1", len(resp.GetSubclasses()))
	}
}

func TestGetHeritage_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetHeritage(context.Background(), &pb.GetDaggerheartHeritageRequest{Id: "her-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetHeritage().GetName() != "Elf" {
		t.Errorf("name = %q, want Elf", resp.GetHeritage().GetName())
	}
}

func TestListHeritages_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListHeritages(context.Background(), &pb.ListDaggerheartHeritagesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetHeritages()) != 1 {
		t.Errorf("heritages = %d, want 1", len(resp.GetHeritages()))
	}
}

func TestGetExperience_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.GetExperience(context.Background(), &pb.GetDaggerheartExperienceRequest{Id: "exp-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.GetExperience().GetName() != "Wanderer" {
		t.Errorf("name = %q, want Wanderer", resp.GetExperience().GetName())
	}
}

func TestListExperiences_Success(t *testing.T) {
	svc := newContentTestService()
	resp, err := svc.ListExperiences(context.Background(), &pb.ListDaggerheartExperiencesRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.GetExperiences()) != 1 {
		t.Errorf("experiences = %d, want 1", len(resp.GetExperiences()))
	}
}
