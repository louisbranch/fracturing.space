package gateway

import (
	"context"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc"
)

// ParticipantReadClient exposes participant queries for campaign workspace pages.
type ParticipantReadClient interface {
	ListParticipants(context.Context, *statev1.ListParticipantsRequest, ...grpc.CallOption) (*statev1.ListParticipantsResponse, error)
	GetParticipant(context.Context, *statev1.GetParticipantRequest, ...grpc.CallOption) (*statev1.GetParticipantResponse, error)
}

// ParticipantMutationClient exposes participant mutations for campaign workspace pages.
type ParticipantMutationClient interface {
	CreateParticipant(context.Context, *statev1.CreateParticipantRequest, ...grpc.CallOption) (*statev1.CreateParticipantResponse, error)
	UpdateParticipant(context.Context, *statev1.UpdateParticipantRequest, ...grpc.CallOption) (*statev1.UpdateParticipantResponse, error)
}

// CharacterReadClient exposes character query operations for campaign workspace pages.
type CharacterReadClient interface {
	ListCharacters(context.Context, *statev1.ListCharactersRequest, ...grpc.CallOption) (*statev1.ListCharactersResponse, error)
	ListCharacterProfiles(context.Context, *statev1.ListCharacterProfilesRequest, ...grpc.CallOption) (*statev1.ListCharacterProfilesResponse, error)
	GetCharacterSheet(context.Context, *statev1.GetCharacterSheetRequest, ...grpc.CallOption) (*statev1.GetCharacterSheetResponse, error)
	GetCharacterCreationProgress(context.Context, *statev1.GetCharacterCreationProgressRequest, ...grpc.CallOption) (*statev1.GetCharacterCreationProgressResponse, error)
}

// CharacterMutationClient exposes character mutations for campaign workspace pages.
type CharacterMutationClient interface {
	CreateCharacter(context.Context, *statev1.CreateCharacterRequest, ...grpc.CallOption) (*statev1.CreateCharacterResponse, error)
	UpdateCharacter(context.Context, *statev1.UpdateCharacterRequest, ...grpc.CallOption) (*statev1.UpdateCharacterResponse, error)
	DeleteCharacter(context.Context, *statev1.DeleteCharacterRequest, ...grpc.CallOption) (*statev1.DeleteCharacterResponse, error)
	SetDefaultControl(context.Context, *statev1.SetDefaultControlRequest, ...grpc.CallOption) (*statev1.SetDefaultControlResponse, error)
	ClaimCharacterControl(context.Context, *statev1.ClaimCharacterControlRequest, ...grpc.CallOption) (*statev1.ClaimCharacterControlResponse, error)
	ReleaseCharacterControl(context.Context, *statev1.ReleaseCharacterControlRequest, ...grpc.CallOption) (*statev1.ReleaseCharacterControlResponse, error)
	ApplyCharacterCreationStep(context.Context, *statev1.ApplyCharacterCreationStepRequest, ...grpc.CallOption) (*statev1.ApplyCharacterCreationStepResponse, error)
	ResetCharacterCreationWorkflow(context.Context, *statev1.ResetCharacterCreationWorkflowRequest, ...grpc.CallOption) (*statev1.ResetCharacterCreationWorkflowResponse, error)
}

// DaggerheartContentClient exposes Daggerheart content catalog operations.
type DaggerheartContentClient interface {
	GetContentCatalog(context.Context, *daggerheartv1.GetDaggerheartContentCatalogRequest, ...grpc.CallOption) (*daggerheartv1.GetDaggerheartContentCatalogResponse, error)
}

// DaggerheartAssetClient exposes Daggerheart content-asset map operations.
type DaggerheartAssetClient interface {
	GetAssetMap(context.Context, *daggerheartv1.GetDaggerheartAssetMapRequest, ...grpc.CallOption) (*daggerheartv1.GetDaggerheartAssetMapResponse, error)
}
