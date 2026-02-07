package service

import (
	"context"
	"errors"
	"testing"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
	"github.com/louisbranch/fracturing.space/internal/campaign/domain"
	"github.com/louisbranch/fracturing.space/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestGetCharacterSheetSuccess(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	characterStore := &fakeCharacterStore{
		getCharacter: domain.Character{
			ID:         "character-123",
			CampaignID: "camp-123",
			Name:       "Alice",
			Kind:       domain.CharacterKindPC,
			CreatedAt:  fixedTime,
			UpdatedAt:  fixedTime,
		},
	}
	profileStore := &fakeCharacterProfileStore{
		getProfile: domain.CharacterProfile{
			CampaignID:      "camp-123",
			CharacterID:     "character-123",
			Traits:          map[string]int{"agility": 1},
			HpMax:           6,
			StressMax:       4,
			Evasion:         3,
			MajorThreshold:  2,
			SevereThreshold: 5,
		},
	}
	stateStore := &fakeCharacterStateStore{
		getState: domain.CharacterState{
			CampaignID:  "camp-123",
			CharacterID: "character-123",
			Hope:        2,
			Stress:      1,
			Hp:          5,
		},
	}
	service := &CampaignService{
		stores: Stores{
			Character:        characterStore,
			CharacterProfile: profileStore,
			CharacterState:   stateStore,
		},
	}

	response, err := service.GetCharacterSheet(context.Background(), &campaignv1.GetCharacterSheetRequest{
		CampaignId:  "camp-123",
		CharacterId: "character-123",
	})
	if err != nil {
		t.Fatalf("get character sheet: %v", err)
	}
	if response == nil || response.Character == nil || response.Profile == nil || response.State == nil {
		t.Fatal("expected character sheet response")
	}
	if response.Character.Name != "Alice" {
		t.Fatalf("expected name Alice, got %q", response.Character.Name)
	}
	if response.Profile.HpMax != 6 {
		t.Fatalf("expected hp max 6, got %d", response.Profile.HpMax)
	}
	if response.State.Hope != 2 {
		t.Fatalf("expected hope 2, got %d", response.State.Hope)
	}
}

func TestGetCharacterSheetNilRequest(t *testing.T) {
	service := &CampaignService{}
	_, err := service.GetCharacterSheet(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", st.Code())
	}
}

func TestGetCharacterSheetMissingStore(t *testing.T) {
	tests := []struct {
		name    string
		service CampaignService
	}{
		{
			name: "missing character store",
			service: CampaignService{stores: Stores{
				CharacterProfile: &fakeCharacterProfileStore{},
				CharacterState:   &fakeCharacterStateStore{},
			}},
		},
		{
			name: "missing profile store",
			service: CampaignService{stores: Stores{
				Character:      &fakeCharacterStore{},
				CharacterState: &fakeCharacterStateStore{},
			}},
		},
		{
			name: "missing state store",
			service: CampaignService{stores: Stores{
				Character:        &fakeCharacterStore{},
				CharacterProfile: &fakeCharacterProfileStore{},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.service.GetCharacterSheet(context.Background(), &campaignv1.GetCharacterSheetRequest{
				CampaignId:  "camp-123",
				CharacterId: "character-123",
			})
			if err == nil {
				t.Fatal("expected error")
			}
			st, ok := status.FromError(err)
			if !ok {
				t.Fatalf("expected grpc status error, got %v", err)
			}
			if st.Code() != codes.Internal {
				t.Fatalf("expected internal error, got %v", st.Code())
			}
		})
	}
}

func TestGetCharacterSheetEmptyIDs(t *testing.T) {
	service := &CampaignService{stores: Stores{
		Character:        &fakeCharacterStore{},
		CharacterProfile: &fakeCharacterProfileStore{},
		CharacterState:   &fakeCharacterStateStore{},
	}}
	_, err := service.GetCharacterSheet(context.Background(), &campaignv1.GetCharacterSheetRequest{
		CampaignId: "  ",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", st.Code())
	}
}

func TestGetCharacterSheetCharacterNotFound(t *testing.T) {
	service := &CampaignService{stores: Stores{
		Character:        &fakeCharacterStore{getErr: storage.ErrNotFound},
		CharacterProfile: &fakeCharacterProfileStore{},
		CharacterState:   &fakeCharacterStateStore{},
	}}
	_, err := service.GetCharacterSheet(context.Background(), &campaignv1.GetCharacterSheetRequest{
		CampaignId:  "camp-123",
		CharacterId: "char-456",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Fatalf("expected not found, got %v", st.Code())
	}
}

func TestGetCharacterSheetProfileNotFound(t *testing.T) {
	service := &CampaignService{stores: Stores{
		Character:        &fakeCharacterStore{getCharacter: domain.Character{ID: "char-456", CampaignID: "camp-123"}},
		CharacterProfile: &fakeCharacterProfileStore{getErr: storage.ErrNotFound},
		CharacterState:   &fakeCharacterStateStore{},
	}}
	_, err := service.GetCharacterSheet(context.Background(), &campaignv1.GetCharacterSheetRequest{
		CampaignId:  "camp-123",
		CharacterId: "char-456",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Fatalf("expected not found, got %v", st.Code())
	}
}

func TestGetCharacterSheetStateNotFound(t *testing.T) {
	service := &CampaignService{stores: Stores{
		Character:        &fakeCharacterStore{getCharacter: domain.Character{ID: "char-456", CampaignID: "camp-123"}},
		CharacterProfile: &fakeCharacterProfileStore{getProfile: domain.CharacterProfile{CampaignID: "camp-123", CharacterID: "char-456"}},
		CharacterState:   &fakeCharacterStateStore{getErr: storage.ErrNotFound},
	}}
	_, err := service.GetCharacterSheet(context.Background(), &campaignv1.GetCharacterSheetRequest{
		CampaignId:  "camp-123",
		CharacterId: "char-456",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Fatalf("expected not found, got %v", st.Code())
	}
}

func TestGetCharacterSheetStoreFailure(t *testing.T) {
	service := &CampaignService{stores: Stores{
		Character:        &fakeCharacterStore{getErr: errors.New("boom")},
		CharacterProfile: &fakeCharacterProfileStore{},
		CharacterState:   &fakeCharacterStateStore{},
	}}
	_, err := service.GetCharacterSheet(context.Background(), &campaignv1.GetCharacterSheetRequest{
		CampaignId:  "camp-123",
		CharacterId: "char-456",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.Internal {
		t.Fatalf("expected internal, got %v", st.Code())
	}
}

func TestPatchCharacterProfileSuccess(t *testing.T) {
	profileStore := &fakeCharacterProfileStore{
		getProfile: domain.CharacterProfile{
			CampaignID:      "camp-123",
			CharacterID:     "char-456",
			Traits:          map[string]int{"agility": 0},
			HpMax:           6,
			StressMax:       4,
			Evasion:         3,
			MajorThreshold:  2,
			SevereThreshold: 5,
		},
	}
	service := &CampaignService{stores: Stores{
		CharacterProfile: profileStore,
	}}
	hpMax := int32(8)
	traits := map[string]int32{"agility": 2}

	response, err := service.PatchCharacterProfile(context.Background(), &campaignv1.PatchCharacterProfileRequest{
		CampaignId:  "camp-123",
		CharacterId: "char-456",
		Traits:      traits,
		HpMax:       &hpMax,
	})
	if err != nil {
		t.Fatalf("patch character profile: %v", err)
	}
	if response == nil || response.Profile == nil {
		t.Fatal("expected profile response")
	}
	if response.Profile.HpMax != 8 {
		t.Fatalf("expected hp max 8, got %d", response.Profile.HpMax)
	}
	if profileStore.putProfile.HpMax != 8 {
		t.Fatalf("expected stored hp max 8, got %d", profileStore.putProfile.HpMax)
	}
}

func TestPatchCharacterProfileNilRequest(t *testing.T) {
	service := &CampaignService{}
	_, err := service.PatchCharacterProfile(context.Background(), nil)
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", st.Code())
	}
}

func TestPatchCharacterProfileMissingProfileStore(t *testing.T) {
	service := &CampaignService{stores: Stores{
		CharacterState: &fakeCharacterStateStore{},
	}}
	_, err := service.PatchCharacterProfile(context.Background(), &campaignv1.PatchCharacterProfileRequest{
		CampaignId:  "camp-123",
		CharacterId: "char-456",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.Internal {
		t.Fatalf("expected internal error, got %v", st.Code())
	}
}

func TestPatchCharacterProfileEmptyIDs(t *testing.T) {
	service := &CampaignService{stores: Stores{
		CharacterProfile: &fakeCharacterProfileStore{},
	}}
	_, err := service.PatchCharacterProfile(context.Background(), &campaignv1.PatchCharacterProfileRequest{
		CampaignId: "  ",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", st.Code())
	}
}

func TestPatchCharacterProfileNotFound(t *testing.T) {
	service := &CampaignService{stores: Stores{
		CharacterProfile: &fakeCharacterProfileStore{getErr: storage.ErrNotFound},
		CharacterState:   &fakeCharacterStateStore{},
	}}
	_, err := service.PatchCharacterProfile(context.Background(), &campaignv1.PatchCharacterProfileRequest{
		CampaignId:  "camp-123",
		CharacterId: "char-456",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.NotFound {
		t.Fatalf("expected not found, got %v", st.Code())
	}
}

func TestPatchCharacterProfileInvalidPatch(t *testing.T) {
	service := &CampaignService{stores: Stores{
		CharacterProfile: &fakeCharacterProfileStore{getProfile: domain.CharacterProfile{CampaignID: "camp-123", CharacterID: "char-456", Traits: map[string]int{"agility": 0}, HpMax: 6, StressMax: 4, Evasion: 3, MajorThreshold: 2, SevereThreshold: 5}},
		CharacterState:   &fakeCharacterStateStore{},
	}}
	invalidHpMax := int32(0)
	_, err := service.PatchCharacterProfile(context.Background(), &campaignv1.PatchCharacterProfileRequest{
		CampaignId:  "camp-123",
		CharacterId: "char-456",
		HpMax:       &invalidHpMax,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("expected invalid argument, got %v", st.Code())
	}
}

func TestPatchCharacterProfilePersistFailure(t *testing.T) {
	profileStore := &fakeCharacterProfileStore{
		getProfile: domain.CharacterProfile{CampaignID: "camp-123", CharacterID: "char-456", Traits: map[string]int{"agility": 0}, HpMax: 6, StressMax: 4, Evasion: 3, MajorThreshold: 2, SevereThreshold: 5},
		putErr:     errors.New("boom"),
	}
	service := &CampaignService{stores: Stores{
		CharacterProfile: profileStore,
	}}
	_, err := service.PatchCharacterProfile(context.Background(), &campaignv1.PatchCharacterProfileRequest{
		CampaignId:  "camp-123",
		CharacterId: "char-456",
	})
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.Internal {
		t.Fatalf("expected internal, got %v", st.Code())
	}
}
