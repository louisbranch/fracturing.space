package domain

import (
	"context"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/structpb"
)

// fakeCampaignClient implements statev1.CampaignServiceClient for testing.
type fakeCampaignClient struct {
	statev1.CampaignServiceClient

	createResp  *statev1.CreateCampaignResponse
	createErr   error
	listResp    *statev1.ListCampaignsResponse
	listErr     error
	getResp     *statev1.GetCampaignResponse
	getErr      error
	endResp     *statev1.EndCampaignResponse
	endErr      error
	archiveResp *statev1.ArchiveCampaignResponse
	archiveErr  error
	restoreResp *statev1.RestoreCampaignResponse
	restoreErr  error
}

func (f *fakeCampaignClient) CreateCampaign(_ context.Context, _ *statev1.CreateCampaignRequest, _ ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
	return f.createResp, f.createErr
}

func (f *fakeCampaignClient) ListCampaigns(_ context.Context, _ *statev1.ListCampaignsRequest, _ ...grpc.CallOption) (*statev1.ListCampaignsResponse, error) {
	return f.listResp, f.listErr
}

func (f *fakeCampaignClient) GetCampaign(_ context.Context, _ *statev1.GetCampaignRequest, _ ...grpc.CallOption) (*statev1.GetCampaignResponse, error) {
	return f.getResp, f.getErr
}

func (f *fakeCampaignClient) EndCampaign(_ context.Context, _ *statev1.EndCampaignRequest, _ ...grpc.CallOption) (*statev1.EndCampaignResponse, error) {
	return f.endResp, f.endErr
}

func (f *fakeCampaignClient) ArchiveCampaign(_ context.Context, _ *statev1.ArchiveCampaignRequest, _ ...grpc.CallOption) (*statev1.ArchiveCampaignResponse, error) {
	return f.archiveResp, f.archiveErr
}

func (f *fakeCampaignClient) RestoreCampaign(_ context.Context, _ *statev1.RestoreCampaignRequest, _ ...grpc.CallOption) (*statev1.RestoreCampaignResponse, error) {
	return f.restoreResp, f.restoreErr
}

// fakeSessionClient implements statev1.SessionServiceClient for testing.
type fakeSessionClient struct {
	statev1.SessionServiceClient

	startResp *statev1.StartSessionResponse
	startErr  error
	listResp  *statev1.ListSessionsResponse
	listErr   error
	getResp   *statev1.GetSessionResponse
	getErr    error
	endResp   *statev1.EndSessionResponse
	endErr    error
}

func (f *fakeSessionClient) StartSession(_ context.Context, _ *statev1.StartSessionRequest, _ ...grpc.CallOption) (*statev1.StartSessionResponse, error) {
	return f.startResp, f.startErr
}

func (f *fakeSessionClient) ListSessions(_ context.Context, _ *statev1.ListSessionsRequest, _ ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
	return f.listResp, f.listErr
}

func (f *fakeSessionClient) GetSession(_ context.Context, _ *statev1.GetSessionRequest, _ ...grpc.CallOption) (*statev1.GetSessionResponse, error) {
	return f.getResp, f.getErr
}

func (f *fakeSessionClient) EndSession(_ context.Context, _ *statev1.EndSessionRequest, _ ...grpc.CallOption) (*statev1.EndSessionResponse, error) {
	return f.endResp, f.endErr
}

// fakeParticipantClient implements statev1.ParticipantServiceClient for testing.
type fakeParticipantClient struct {
	statev1.ParticipantServiceClient

	createResp *statev1.CreateParticipantResponse
	createErr  error
	updateResp *statev1.UpdateParticipantResponse
	updateErr  error
	deleteResp *statev1.DeleteParticipantResponse
	deleteErr  error
	listResp   *statev1.ListParticipantsResponse
	listErr    error
	getResp    *statev1.GetParticipantResponse
	getErr     error
}

func (f *fakeParticipantClient) CreateParticipant(_ context.Context, _ *statev1.CreateParticipantRequest, _ ...grpc.CallOption) (*statev1.CreateParticipantResponse, error) {
	return f.createResp, f.createErr
}

func (f *fakeParticipantClient) UpdateParticipant(_ context.Context, _ *statev1.UpdateParticipantRequest, _ ...grpc.CallOption) (*statev1.UpdateParticipantResponse, error) {
	return f.updateResp, f.updateErr
}

func (f *fakeParticipantClient) DeleteParticipant(_ context.Context, _ *statev1.DeleteParticipantRequest, _ ...grpc.CallOption) (*statev1.DeleteParticipantResponse, error) {
	return f.deleteResp, f.deleteErr
}

func (f *fakeParticipantClient) ListParticipants(_ context.Context, _ *statev1.ListParticipantsRequest, _ ...grpc.CallOption) (*statev1.ListParticipantsResponse, error) {
	return f.listResp, f.listErr
}

func (f *fakeParticipantClient) GetParticipant(_ context.Context, _ *statev1.GetParticipantRequest, _ ...grpc.CallOption) (*statev1.GetParticipantResponse, error) {
	return f.getResp, f.getErr
}

// fakeCharacterClient implements statev1.CharacterServiceClient for testing.
type fakeCharacterClient struct {
	statev1.CharacterServiceClient

	createResp  *statev1.CreateCharacterResponse
	createErr   error
	updateResp  *statev1.UpdateCharacterResponse
	updateErr   error
	deleteResp  *statev1.DeleteCharacterResponse
	deleteErr   error
	listResp    *statev1.ListCharactersResponse
	listErr     error
	controlResp *statev1.SetDefaultControlResponse
	controlErr  error
	sheetResp   *statev1.GetCharacterSheetResponse
	sheetErr    error
	profileResp *statev1.PatchCharacterProfileResponse
	profileErr  error
}

func (f *fakeCharacterClient) CreateCharacter(_ context.Context, _ *statev1.CreateCharacterRequest, _ ...grpc.CallOption) (*statev1.CreateCharacterResponse, error) {
	return f.createResp, f.createErr
}

func (f *fakeCharacterClient) UpdateCharacter(_ context.Context, _ *statev1.UpdateCharacterRequest, _ ...grpc.CallOption) (*statev1.UpdateCharacterResponse, error) {
	return f.updateResp, f.updateErr
}

func (f *fakeCharacterClient) DeleteCharacter(_ context.Context, _ *statev1.DeleteCharacterRequest, _ ...grpc.CallOption) (*statev1.DeleteCharacterResponse, error) {
	return f.deleteResp, f.deleteErr
}

func (f *fakeCharacterClient) ListCharacters(_ context.Context, _ *statev1.ListCharactersRequest, _ ...grpc.CallOption) (*statev1.ListCharactersResponse, error) {
	return f.listResp, f.listErr
}

func (f *fakeCharacterClient) SetDefaultControl(_ context.Context, _ *statev1.SetDefaultControlRequest, _ ...grpc.CallOption) (*statev1.SetDefaultControlResponse, error) {
	return f.controlResp, f.controlErr
}

func (f *fakeCharacterClient) GetCharacterSheet(_ context.Context, _ *statev1.GetCharacterSheetRequest, _ ...grpc.CallOption) (*statev1.GetCharacterSheetResponse, error) {
	return f.sheetResp, f.sheetErr
}

func (f *fakeCharacterClient) PatchCharacterProfile(_ context.Context, _ *statev1.PatchCharacterProfileRequest, _ ...grpc.CallOption) (*statev1.PatchCharacterProfileResponse, error) {
	return f.profileResp, f.profileErr
}

// fakeSnapshotClient implements statev1.SnapshotServiceClient for testing.
type fakeSnapshotClient struct {
	statev1.SnapshotServiceClient

	patchStateResp *statev1.PatchCharacterStateResponse
	patchStateErr  error
}

func (f *fakeSnapshotClient) PatchCharacterState(_ context.Context, _ *statev1.PatchCharacterStateRequest, _ ...grpc.CallOption) (*statev1.PatchCharacterStateResponse, error) {
	return f.patchStateResp, f.patchStateErr
}

// fakeEventClient implements statev1.EventServiceClient for testing.
type fakeEventClient struct {
	statev1.EventServiceClient

	listResp *statev1.ListEventsResponse
	listErr  error
}

func (f *fakeEventClient) ListEvents(_ context.Context, _ *statev1.ListEventsRequest, _ ...grpc.CallOption) (*statev1.ListEventsResponse, error) {
	return f.listResp, f.listErr
}

// fakeForkClient implements statev1.ForkServiceClient for testing.
type fakeForkClient struct {
	statev1.ForkServiceClient

	forkResp    *statev1.ForkCampaignResponse
	forkErr     error
	lineageResp *statev1.GetLineageResponse
	lineageErr  error
}

func (f *fakeForkClient) ForkCampaign(_ context.Context, _ *statev1.ForkCampaignRequest, _ ...grpc.CallOption) (*statev1.ForkCampaignResponse, error) {
	return f.forkResp, f.forkErr
}

func (f *fakeForkClient) GetLineage(_ context.Context, _ *statev1.GetLineageRequest, _ ...grpc.CallOption) (*statev1.GetLineageResponse, error) {
	return f.lineageResp, f.lineageErr
}

// fakeDaggerheartClient implements pb.DaggerheartServiceClient for testing.
type fakeDaggerheartClient struct {
	pb.DaggerheartServiceClient

	actionRollResp   *pb.ActionRollResponse
	actionRollErr    error
	outcomeResp      *pb.DualityOutcomeResponse
	outcomeErr       error
	explainResp      *pb.DualityExplainResponse
	explainErr       error
	probabilityResp  *pb.DualityProbabilityResponse
	probabilityErr   error
	rulesVersionResp *pb.RulesVersionResponse
	rulesVersionErr  error
	rollDiceResp     *pb.RollDiceResponse
	rollDiceErr      error
}

func (f *fakeDaggerheartClient) ActionRoll(_ context.Context, _ *pb.ActionRollRequest, _ ...grpc.CallOption) (*pb.ActionRollResponse, error) {
	return f.actionRollResp, f.actionRollErr
}

func (f *fakeDaggerheartClient) DualityOutcome(_ context.Context, _ *pb.DualityOutcomeRequest, _ ...grpc.CallOption) (*pb.DualityOutcomeResponse, error) {
	return f.outcomeResp, f.outcomeErr
}

func (f *fakeDaggerheartClient) DualityExplain(_ context.Context, _ *pb.DualityExplainRequest, _ ...grpc.CallOption) (*pb.DualityExplainResponse, error) {
	return f.explainResp, f.explainErr
}

func (f *fakeDaggerheartClient) DualityProbability(_ context.Context, _ *pb.DualityProbabilityRequest, _ ...grpc.CallOption) (*pb.DualityProbabilityResponse, error) {
	return f.probabilityResp, f.probabilityErr
}

func (f *fakeDaggerheartClient) RulesVersion(_ context.Context, _ *pb.RulesVersionRequest, _ ...grpc.CallOption) (*pb.RulesVersionResponse, error) {
	return f.rulesVersionResp, f.rulesVersionErr
}

func (f *fakeDaggerheartClient) RollDice(_ context.Context, _ *pb.RollDiceRequest, _ ...grpc.CallOption) (*pb.RollDiceResponse, error) {
	return f.rollDiceResp, f.rollDiceErr
}

// newStructValue builds a *structpb.Struct from a map for test data.
func newStructValue(m map[string]any) *structpb.Struct {
	s, _ := structpb.NewStruct(m)
	return s
}

// testCampaign returns a proto campaign for test fixtures.
func testCampaign(id, name string, status statev1.CampaignStatus) *statev1.Campaign {
	return &statev1.Campaign{
		Id:     id,
		Name:   name,
		Status: status,
		GmMode: statev1.GmMode_HUMAN,
		System: commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
	}
}

// testParticipant returns a proto participant for test fixtures.
func testParticipant(id, campaignID, displayName string, role statev1.ParticipantRole) *statev1.Participant {
	return &statev1.Participant{
		Id:          id,
		CampaignId:  campaignID,
		DisplayName: displayName,
		Role:        role,
		Controller:  statev1.Controller_CONTROLLER_HUMAN,
	}
}

// testCharacter returns a proto character for test fixtures.
func testCharacter(id, campaignID, name string, kind statev1.CharacterKind) *statev1.Character {
	return &statev1.Character{
		Id:         id,
		CampaignId: campaignID,
		Name:       name,
		Kind:       kind,
	}
}

// testSession returns a proto session for test fixtures.
func testSession(id, campaignID, name string, status statev1.SessionStatus) *statev1.Session {
	return &statev1.Session{
		Id:         id,
		CampaignId: campaignID,
		Name:       name,
		Status:     status,
	}
}
