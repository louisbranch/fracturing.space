package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	campaignv1 "github.com/louisbranch/duality-engine/api/gen/go/campaign/v1"
	"github.com/louisbranch/duality-engine/internal/campaign/domain"
	"github.com/louisbranch/duality-engine/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeCharacterStore struct {
	putCharacter           domain.Character
	putErr             error
	getCharacter           domain.Character
	getErr             error
	listPage           storage.CharacterPage
	listErr            error
	listPageSize       int
	listPageToken      string
	listPageCampaignID string
}

func (f *fakeCharacterStore) PutCharacter(ctx context.Context, character domain.Character) error {
	f.putCharacter = character
	return f.putErr
}

func (f *fakeCharacterStore) GetCharacter(ctx context.Context, campaignID, characterID string) (domain.Character, error) {
	return f.getCharacter, f.getErr
}

func (f *fakeCharacterStore) ListCharacters(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.CharacterPage, error) {
	f.listPageCampaignID = campaignID
	f.listPageSize = pageSize
	f.listPageToken = pageToken
	return f.listPage, f.listErr
}

type fakeCharacterProfileStore struct {
	putProfile domain.CharacterProfile
	putErr     error
	getProfile domain.CharacterProfile
	getErr     error
}

func (f *fakeCharacterProfileStore) PutCharacterProfile(ctx context.Context, profile domain.CharacterProfile) error {
	f.putProfile = profile
	return f.putErr
}

func (f *fakeCharacterProfileStore) GetCharacterProfile(ctx context.Context, campaignID, characterID string) (domain.CharacterProfile, error) {
	return f.getProfile, f.getErr
}

type fakeCharacterStateStore struct {
	putState domain.CharacterState
	putErr   error
	getState domain.CharacterState
	getErr   error
}

func (f *fakeCharacterStateStore) PutCharacterState(ctx context.Context, state domain.CharacterState) error {
	f.putState = state
	return f.putErr
}

func (f *fakeCharacterStateStore) GetCharacterState(ctx context.Context, campaignID, characterID string) (domain.CharacterState, error) {
	return f.getState, f.getErr
}

func TestCreateCharacterSuccess(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}
	characterStore := &fakeCharacterStore{}
	service := &CampaignService{
		stores: Stores{
			Campaign:         campaignStore,
			Character:        characterStore,
			CharacterProfile: &fakeCharacterProfileStore{},
			CharacterState:   &fakeCharacterStateStore{},
		},
		clock: func() time.Time {
			return fixedTime
		},
		idGenerator: func() (string, error) {
			return "character-456", nil
		},
	}

	response, err := service.CreateCharacter(context.Background(), &campaignv1.CreateCharacterRequest{
		CampaignId: "camp-123",
		Name:       "  Alice  ",
		Kind:       campaignv1.CharacterKind_PC,
		Notes:      "A brave warrior",
	})
	if err != nil {
		t.Fatalf("create character: %v", err)
	}
	if response == nil || response.Character == nil {
		t.Fatal("expected character response")
	}
	if response.Character.Id != "character-456" {
		t.Fatalf("expected id character-456, got %q", response.Character.Id)
	}
	if response.Character.CampaignId != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", response.Character.CampaignId)
	}
	if response.Character.Name != "Alice" {
		t.Fatalf("expected trimmed name, got %q", response.Character.Name)
	}
	if response.Character.Kind != campaignv1.CharacterKind_PC {
		t.Fatalf("expected kind PC, got %v", response.Character.Kind)
	}
	if response.Character.Notes != "A brave warrior" {
		t.Fatalf("expected notes, got %q", response.Character.Notes)
	}
	if response.Character.CreatedAt.AsTime() != fixedTime {
		t.Fatalf("expected created_at %v, got %v", fixedTime, response.Character.CreatedAt.AsTime())
	}
	if characterStore.putCharacter.ID != "character-456" {
		t.Fatalf("expected stored id character-456, got %q", characterStore.putCharacter.ID)
	}
}

func TestCreateCharacterValidationErrors(t *testing.T) {
	tests := []struct {
		name string
		req  *campaignv1.CreateCharacterRequest
	}{
		{
			name: "empty name",
			req: &campaignv1.CreateCharacterRequest{
				CampaignId: "camp-123",
				Name:       "  ",
				Kind:       campaignv1.CharacterKind_PC,
			},
		},
		{
			name: "missing kind",
			req: &campaignv1.CreateCharacterRequest{
				CampaignId: "camp-123",
				Name:       "Alice",
				Kind:       campaignv1.CharacterKind_CHARACTER_KIND_UNSPECIFIED,
			},
		},
		{
			name: "empty campaign id",
			req: &campaignv1.CreateCharacterRequest{
				CampaignId: "  ",
				Name:       "Alice",
				Kind:       campaignv1.CharacterKind_PC,
			},
		},
	}

	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: id}, nil
	}
	service := &CampaignService{
		stores: Stores{
			Campaign:         campaignStore,
			Character:        &fakeCharacterStore{},
			CharacterProfile: &fakeCharacterProfileStore{},
			CharacterState:   &fakeCharacterStateStore{},
		},
		clock:       time.Now,
		idGenerator: func() (string, error) { return "character-1", nil },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.CreateCharacter(context.Background(), tt.req)
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
		})
	}
}

func TestCreateCharacterCampaignNotFound(t *testing.T) {
	characterStore := &fakeCharacterStore{putErr: storage.ErrNotFound}
	service := &CampaignService{
		stores: Stores{
			Character:        characterStore,
			CharacterProfile: &fakeCharacterProfileStore{},
			CharacterState:   &fakeCharacterStateStore{},
		},
		clock:       time.Now,
		idGenerator: func() (string, error) { return "character-1", nil },
	}

	_, err := service.CreateCharacter(context.Background(), &campaignv1.CreateCharacterRequest{
		CampaignId: "missing",
		Name:       "Alice",
		Kind:       campaignv1.CharacterKind_PC,
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

func TestCreateCharacterNilRequest(t *testing.T) {
	service := NewCampaignService(Stores{
		Campaign:         &fakeCampaignStore{},
		Participant:      &fakeParticipantStore{},
		Character:        &fakeCharacterStore{},
		CharacterProfile: &fakeCharacterProfileStore{},
		CharacterState:   &fakeCharacterStateStore{},
	})

	_, err := service.CreateCharacter(context.Background(), nil)
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

func TestCreateCharacterStoreFailure(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	characterStore := &fakeCharacterStore{putErr: errors.New("boom")}
	service := &CampaignService{
		stores: Stores{
			Campaign:         campaignStore,
			Character:        characterStore,
			CharacterProfile: &fakeCharacterProfileStore{},
			CharacterState:   &fakeCharacterStateStore{},
		},
		clock:       time.Now,
		idGenerator: func() (string, error) { return "character-123", nil },
	}

	_, err := service.CreateCharacter(context.Background(), &campaignv1.CreateCharacterRequest{
		CampaignId: "camp-123",
		Name:       "Alice",
		Kind:       campaignv1.CharacterKind_PC,
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
	if !strings.Contains(st.Message(), "persist character") {
		t.Fatalf("expected error message to mention 'persist character', got %q", st.Message())
	}
}

func TestCreateCharacterNPCKind(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	characterStore := &fakeCharacterStore{}
	service := &CampaignService{
		stores: Stores{
			Campaign:         campaignStore,
			Character:        characterStore,
			CharacterProfile: &fakeCharacterProfileStore{},
			CharacterState:   &fakeCharacterStateStore{},
		},
		clock: func() time.Time {
			return fixedTime
		},
		idGenerator: func() (string, error) {
			return "character-789", nil
		},
	}

	response, err := service.CreateCharacter(context.Background(), &campaignv1.CreateCharacterRequest{
		CampaignId: "camp-123",
		Name:       "Goblin",
		Kind:       campaignv1.CharacterKind_NPC,
		Notes:      "A small creature",
	})
	if err != nil {
		t.Fatalf("create character: %v", err)
	}
	if response.Character.Kind != campaignv1.CharacterKind_NPC {
		t.Fatalf("expected kind NPC, got %v", response.Character.Kind)
	}
	if characterStore.putCharacter.Kind != domain.CharacterKindNPC {
		t.Fatalf("expected stored kind NPC, got %v", characterStore.putCharacter.Kind)
	}
}

func TestCreateCharacterMissingStore(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	service := &CampaignService{
		stores: Stores{
			Campaign:         campaignStore,
			CharacterProfile: &fakeCharacterProfileStore{},
			CharacterState:   &fakeCharacterStateStore{},
		},
	}

	_, err := service.CreateCharacter(context.Background(), &campaignv1.CreateCharacterRequest{
		CampaignId: "camp-123",
		Name:       "Alice",
		Kind:       campaignv1.CharacterKind_PC,
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

func TestListCharactersSuccess(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}
	characterStore := &fakeCharacterStore{
		listPage: storage.CharacterPage{
			Characters: []domain.Character{
				{
					ID:         "character-1",
					CampaignID: "camp-123",
					Name:       "Alice",
					Kind:       domain.CharacterKindPC,
					Notes:      "A brave warrior",
					CreatedAt:  fixedTime,
					UpdatedAt:  fixedTime,
				},
				{
					ID:         "character-2",
					CampaignID: "camp-123",
					Name:       "Goblin",
					Kind:       domain.CharacterKindNPC,
					Notes:      "A small creature",
					CreatedAt:  fixedTime,
					UpdatedAt:  fixedTime,
				},
			},
			NextPageToken: "next-token",
		},
	}
	service := &CampaignService{
		stores: Stores{
			Campaign:         campaignStore,
			Character:        characterStore,
			CharacterProfile: &fakeCharacterProfileStore{},
			CharacterState:   &fakeCharacterStateStore{},
		},
		clock: time.Now,
	}

	response, err := service.ListCharacters(context.Background(), &campaignv1.ListCharactersRequest{
		CampaignId: "camp-123",
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("list characters: %v", err)
	}
	if response == nil {
		t.Fatal("expected response")
	}
	if len(response.Characters) != 2 {
		t.Fatalf("expected 2 characters, got %d", len(response.Characters))
	}
	if response.NextPageToken != "next-token" {
		t.Fatalf("expected next page token next-token, got %q", response.NextPageToken)
	}
	if response.Characters[0].Id != "character-1" {
		t.Fatalf("expected first character id character-1, got %q", response.Characters[0].Id)
	}
	if response.Characters[0].Name != "Alice" {
		t.Fatalf("expected first character name Alice, got %q", response.Characters[0].Name)
	}
	if response.Characters[0].Kind != campaignv1.CharacterKind_PC {
		t.Fatalf("expected first character kind PC, got %v", response.Characters[0].Kind)
	}
	if response.Characters[1].Id != "character-2" {
		t.Fatalf("expected second character id character-2, got %q", response.Characters[1].Id)
	}
	if response.Characters[1].Kind != campaignv1.CharacterKind_NPC {
		t.Fatalf("expected second character kind NPC, got %v", response.Characters[1].Kind)
	}
	if response.Characters[0].CreatedAt.AsTime() != fixedTime {
		t.Fatalf("expected created_at %v, got %v", fixedTime, response.Characters[0].CreatedAt.AsTime())
	}
}

func TestListCharactersDefaults(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	characterStore := &fakeCharacterStore{
		listPage: storage.CharacterPage{
			Characters: []domain.Character{
				{
					ID:         "character-1",
					CampaignID: "camp-123",
					Name:       "Alice",
					Kind:       domain.CharacterKindPC,
					CreatedAt:  fixedTime,
					UpdatedAt:  fixedTime,
				},
			},
			NextPageToken: "next-token",
		},
	}
	service := &CampaignService{
		stores: Stores{
			Campaign:         campaignStore,
			Character:        characterStore,
			CharacterProfile: &fakeCharacterProfileStore{},
			CharacterState:   &fakeCharacterStateStore{},
		},
		clock: time.Now,
	}

	response, err := service.ListCharacters(context.Background(), &campaignv1.ListCharactersRequest{
		CampaignId: "camp-123",
		PageSize:   0,
	})
	if err != nil {
		t.Fatalf("list characters: %v", err)
	}
	if response == nil {
		t.Fatal("expected response")
	}
	if characterStore.listPageSize != defaultListCharactersPageSize {
		t.Fatalf("expected default page size %d, got %d", defaultListCharactersPageSize, characterStore.listPageSize)
	}
	if response.NextPageToken != "next-token" {
		t.Fatalf("expected next page token, got %q", response.NextPageToken)
	}
	if len(response.Characters) != 1 {
		t.Fatalf("expected 1 character, got %d", len(response.Characters))
	}
}

func TestListCharactersEmpty(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	characterStore := &fakeCharacterStore{
		listPage: storage.CharacterPage{},
	}
	service := &CampaignService{
		stores: Stores{
			Campaign:         campaignStore,
			Character:        characterStore,
			CharacterProfile: &fakeCharacterProfileStore{},
			CharacterState:   &fakeCharacterStateStore{},
		},
		clock: time.Now,
	}

	response, err := service.ListCharacters(context.Background(), &campaignv1.ListCharactersRequest{
		CampaignId: "camp-123",
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("list characters: %v", err)
	}
	if response == nil {
		t.Fatal("expected response")
	}
	if len(response.Characters) != 0 {
		t.Fatalf("expected 0 characters, got %d", len(response.Characters))
	}
}

func TestListCharactersClampPageSize(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	characterStore := &fakeCharacterStore{listPage: storage.CharacterPage{}}
	service := &CampaignService{
		stores: Stores{
			Campaign:         campaignStore,
			Character:        characterStore,
			CharacterProfile: &fakeCharacterProfileStore{},
			CharacterState:   &fakeCharacterStateStore{},
		},
		clock: time.Now,
	}

	_, err := service.ListCharacters(context.Background(), &campaignv1.ListCharactersRequest{
		CampaignId: "camp-123",
		PageSize:   25,
	})
	if err != nil {
		t.Fatalf("list characters: %v", err)
	}
	if characterStore.listPageSize != maxListCharactersPageSize {
		t.Fatalf("expected max page size %d, got %d", maxListCharactersPageSize, characterStore.listPageSize)
	}
}

func TestListCharactersPassesToken(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	characterStore := &fakeCharacterStore{listPage: storage.CharacterPage{}}
	service := &CampaignService{
		stores: Stores{
			Campaign:         campaignStore,
			Character:        characterStore,
			CharacterProfile: &fakeCharacterProfileStore{},
			CharacterState:   &fakeCharacterStateStore{},
		},
		clock: time.Now,
	}

	_, err := service.ListCharacters(context.Background(), &campaignv1.ListCharactersRequest{
		CampaignId: "camp-123",
		PageSize:   1,
		PageToken:  "next",
	})
	if err != nil {
		t.Fatalf("list characters: %v", err)
	}
	if characterStore.listPageToken != "next" {
		t.Fatalf("expected page token next, got %q", characterStore.listPageToken)
	}
	if characterStore.listPageCampaignID != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", characterStore.listPageCampaignID)
	}
}

func TestListCharactersNilRequest(t *testing.T) {
	service := NewCampaignService(Stores{
		Campaign:         &fakeCampaignStore{},
		Participant:      &fakeParticipantStore{},
		Character:        &fakeCharacterStore{},
		CharacterProfile: &fakeCharacterProfileStore{},
		CharacterState:   &fakeCharacterStateStore{},
	})

	_, err := service.ListCharacters(context.Background(), nil)
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

func TestListCharactersCampaignNotFound(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{}, storage.ErrNotFound
	}
	service := &CampaignService{
		stores: Stores{
			Campaign:         campaignStore,
			Character:        &fakeCharacterStore{},
			CharacterProfile: &fakeCharacterProfileStore{},
			CharacterState:   &fakeCharacterStateStore{},
		},
		clock: time.Now,
	}

	_, err := service.ListCharacters(context.Background(), &campaignv1.ListCharactersRequest{
		CampaignId: "missing",
		PageSize:   10,
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

func TestListCharactersStoreFailure(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	characterStore := &fakeCharacterStore{listErr: errors.New("boom")}
	service := &CampaignService{
		stores: Stores{
			Campaign:         campaignStore,
			Character:        characterStore,
			CharacterProfile: &fakeCharacterProfileStore{},
			CharacterState:   &fakeCharacterStateStore{},
		},
		clock: time.Now,
	}

	_, err := service.ListCharacters(context.Background(), &campaignv1.ListCharactersRequest{
		CampaignId: "camp-123",
		PageSize:   10,
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

func TestListCharactersMissingStore(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	service := &CampaignService{
		stores: Stores{
			Campaign:         campaignStore,
			CharacterProfile: &fakeCharacterProfileStore{},
			CharacterState:   &fakeCharacterStateStore{},
		},
	}

	_, err := service.ListCharacters(context.Background(), &campaignv1.ListCharactersRequest{
		CampaignId: "camp-123",
		PageSize:   10,
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

func TestListCharactersEmptyCampaignID(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	service := &CampaignService{
		stores: Stores{
			Campaign:         campaignStore,
			Character:        &fakeCharacterStore{},
			CharacterProfile: &fakeCharacterProfileStore{},
			CharacterState:   &fakeCharacterStateStore{},
		},
		clock: time.Now,
	}

	_, err := service.ListCharacters(context.Background(), &campaignv1.ListCharactersRequest{
		CampaignId: "  ",
		PageSize:   10,
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
