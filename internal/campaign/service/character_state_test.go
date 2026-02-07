package service

import (
	"context"
	"errors"
	"testing"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
	"github.com/louisbranch/fracturing.space/internal/campaign/domain"
	"github.com/louisbranch/fracturing.space/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestPatchCharacterStateSuccess(t *testing.T) {
	stateStore := &fakeCharacterStateStore{getState: domain.CharacterState{CampaignID: "camp-123", CharacterID: "char-456", Hope: 1, Stress: 2, Hp: 3}}
	profileStore := &fakeCharacterProfileStore{getProfile: domain.CharacterProfile{CampaignID: "camp-123", CharacterID: "char-456", HpMax: 6, StressMax: 4}}
	service := &CampaignService{stores: Stores{
		CharacterState:   stateStore,
		CharacterProfile: profileStore,
	}}
	newHope := int32(4)

	response, err := service.PatchCharacterState(context.Background(), &campaignv1.PatchCharacterStateRequest{
		CampaignId:  "camp-123",
		CharacterId: "char-456",
		Hope:        &newHope,
	})
	if err != nil {
		t.Fatalf("patch character state: %v", err)
	}
	if response == nil || response.State == nil {
		t.Fatal("expected state response")
	}
	if response.State.Hope != 4 {
		t.Fatalf("expected hope 4, got %d", response.State.Hope)
	}
	if stateStore.putState.Hope != 4 {
		t.Fatalf("expected stored hope 4, got %d", stateStore.putState.Hope)
	}
}

func TestPatchCharacterStateNilRequest(t *testing.T) {
	service := &CampaignService{}
	_, err := service.PatchCharacterState(context.Background(), nil)
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

func TestPatchCharacterStateMissingStore(t *testing.T) {
	tests := []struct {
		name    string
		service CampaignService
	}{
		{
			name: "missing state store",
			service: CampaignService{stores: Stores{
				CharacterProfile: &fakeCharacterProfileStore{},
			}},
		},
		{
			name: "missing profile store",
			service: CampaignService{stores: Stores{
				CharacterState: &fakeCharacterStateStore{},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.service.PatchCharacterState(context.Background(), &campaignv1.PatchCharacterStateRequest{
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
		})
	}
}

func TestPatchCharacterStateEmptyIDs(t *testing.T) {
	service := &CampaignService{stores: Stores{
		CharacterState:   &fakeCharacterStateStore{},
		CharacterProfile: &fakeCharacterProfileStore{},
	}}
	_, err := service.PatchCharacterState(context.Background(), &campaignv1.PatchCharacterStateRequest{
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

func TestPatchCharacterStateNotFound(t *testing.T) {
	service := &CampaignService{stores: Stores{
		CharacterState:   &fakeCharacterStateStore{getErr: storage.ErrNotFound},
		CharacterProfile: &fakeCharacterProfileStore{},
	}}
	_, err := service.PatchCharacterState(context.Background(), &campaignv1.PatchCharacterStateRequest{
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

func TestPatchCharacterStateProfileNotFound(t *testing.T) {
	service := &CampaignService{stores: Stores{
		CharacterState:   &fakeCharacterStateStore{getState: domain.CharacterState{CampaignID: "camp-123", CharacterID: "char-456"}},
		CharacterProfile: &fakeCharacterProfileStore{getErr: storage.ErrNotFound},
	}}
	_, err := service.PatchCharacterState(context.Background(), &campaignv1.PatchCharacterStateRequest{
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

func TestPatchCharacterStateInvalidPatch(t *testing.T) {
	service := &CampaignService{stores: Stores{
		CharacterState:   &fakeCharacterStateStore{getState: domain.CharacterState{CampaignID: "camp-123", CharacterID: "char-456", Hope: 1, Stress: 2, Hp: 3}},
		CharacterProfile: &fakeCharacterProfileStore{getProfile: domain.CharacterProfile{CampaignID: "camp-123", CharacterID: "char-456", HpMax: 6, StressMax: 4}},
	}}
	invalidHope := int32(9)
	_, err := service.PatchCharacterState(context.Background(), &campaignv1.PatchCharacterStateRequest{
		CampaignId:  "camp-123",
		CharacterId: "char-456",
		Hope:        &invalidHope,
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

func TestPatchCharacterStatePersistFailure(t *testing.T) {
	stateStore := &fakeCharacterStateStore{
		getState: domain.CharacterState{CampaignID: "camp-123", CharacterID: "char-456", Hope: 1, Stress: 2, Hp: 3},
		putErr:   errors.New("boom"),
	}
	profileStore := &fakeCharacterProfileStore{getProfile: domain.CharacterProfile{CampaignID: "camp-123", CharacterID: "char-456", HpMax: 6, StressMax: 4}}
	service := &CampaignService{stores: Stores{
		CharacterState:   stateStore,
		CharacterProfile: profileStore,
	}}
	_, err := service.PatchCharacterState(context.Background(), &campaignv1.PatchCharacterStateRequest{
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
