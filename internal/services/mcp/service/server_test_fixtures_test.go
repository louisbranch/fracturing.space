package service

import (
	"context"
	"errors"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/test/mock/mcpfakes"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/grpc"
)

type fakeDaggerheartClient = mcpfakes.DaggerheartClient
type fakeCampaignServiceServer = mcpfakes.CampaignServiceServer

// requireToolMetadata asserts tool result metadata includes correlation IDs.
func requireToolMetadata(t *testing.T, result *mcp.CallToolResult) (string, string) {
	t.Helper()
	if result == nil {
		t.Fatal("expected result metadata")
	}
	if result.Meta == nil {
		t.Fatal("expected result metadata map")
	}
	requestID, _ := result.Meta[grpcmeta.RequestIDHeader].(string)
	if requestID == "" {
		t.Fatal("expected request id metadata")
	}
	invocationID, _ := result.Meta[grpcmeta.InvocationIDHeader].(string)
	if invocationID == "" {
		t.Fatal("expected invocation id metadata")
	}
	return requestID, invocationID
}

// fakeCampaignClient implements statev1.CampaignServiceClient for tests.
type fakeCampaignClient struct {
	response                    *statev1.CreateCampaignResponse
	listResponse                *statev1.ListCampaignsResponse
	getCampaignResponse         *statev1.GetCampaignResponse
	endCampaignResponse         *statev1.EndCampaignResponse
	archiveCampaignResponse     *statev1.ArchiveCampaignResponse
	restoreCampaignResponse     *statev1.RestoreCampaignResponse
	setCampaignCoverResponse    *statev1.SetCampaignCoverResponse
	err                         error
	listErr                     error
	getCampaignErr              error
	endCampaignErr              error
	archiveCampaignErr          error
	restoreCampaignErr          error
	setCampaignCoverErr         error
	lastRequest                 *statev1.CreateCampaignRequest
	lastListRequest             *statev1.ListCampaignsRequest
	lastGetCampaignRequest      *statev1.GetCampaignRequest
	lastEndCampaignRequest      *statev1.EndCampaignRequest
	lastArchiveCampaignRequest  *statev1.ArchiveCampaignRequest
	lastRestoreCampaignRequest  *statev1.RestoreCampaignRequest
	lastSetCampaignCoverRequest *statev1.SetCampaignCoverRequest
	listCalls                   int
}

// fakeParticipantClient implements statev1.ParticipantServiceClient for tests.
type fakeParticipantClient struct {
	createParticipantResponse    *statev1.CreateParticipantResponse
	updateParticipantResponse    *statev1.UpdateParticipantResponse
	deleteParticipantResponse    *statev1.DeleteParticipantResponse
	listParticipantsResponse     *statev1.ListParticipantsResponse
	getParticipantResponse       *statev1.GetParticipantResponse
	createParticipantErr         error
	updateParticipantErr         error
	deleteParticipantErr         error
	listParticipantsErr          error
	getParticipantErr            error
	lastCreateParticipantRequest *statev1.CreateParticipantRequest
	lastUpdateParticipantRequest *statev1.UpdateParticipantRequest
	lastDeleteParticipantRequest *statev1.DeleteParticipantRequest
	lastListParticipantsRequest  *statev1.ListParticipantsRequest
	lastGetParticipantRequest    *statev1.GetParticipantRequest
}

// fakeCharacterClient implements statev1.CharacterServiceClient for tests.
type fakeCharacterClient struct {
	createCharacterResponse          *statev1.CreateCharacterResponse
	updateCharacterResponse          *statev1.UpdateCharacterResponse
	deleteCharacterResponse          *statev1.DeleteCharacterResponse
	listCharactersResponse           *statev1.ListCharactersResponse
	setDefaultControlResponse        *statev1.SetDefaultControlResponse
	claimCharacterControlResponse    *statev1.ClaimCharacterControlResponse
	releaseCharacterControlResponse  *statev1.ReleaseCharacterControlResponse
	getCharacterSheetResponse        *statev1.GetCharacterSheetResponse
	patchCharacterProfileResponse    *statev1.PatchCharacterProfileResponse
	createCharacterErr               error
	updateCharacterErr               error
	deleteCharacterErr               error
	listCharactersErr                error
	setDefaultControlErr             error
	claimCharacterControlErr         error
	releaseCharacterControlErr       error
	getCharacterSheetErr             error
	patchCharacterProfileErr         error
	lastCreateCharacterRequest       *statev1.CreateCharacterRequest
	lastUpdateCharacterRequest       *statev1.UpdateCharacterRequest
	lastDeleteCharacterRequest       *statev1.DeleteCharacterRequest
	lastListCharactersRequest        *statev1.ListCharactersRequest
	lastSetDefaultControlRequest     *statev1.SetDefaultControlRequest
	lastClaimCharacterControlRequest *statev1.ClaimCharacterControlRequest
	lastReleaseControlRequest        *statev1.ReleaseCharacterControlRequest
	lastGetCharacterSheetRequest     *statev1.GetCharacterSheetRequest
	lastPatchCharacterProfileRequest *statev1.PatchCharacterProfileRequest
}

// fakeSnapshotClient implements statev1.SnapshotServiceClient for tests.
type fakeSnapshotClient struct {
	getSnapshotResponse            *statev1.GetSnapshotResponse
	patchCharacterStateResponse    *statev1.PatchCharacterStateResponse
	updateSnapshotStateResponse    *statev1.UpdateSnapshotStateResponse
	getSnapshotErr                 error
	patchCharacterStateErr         error
	updateSnapshotStateErr         error
	lastGetSnapshotRequest         *statev1.GetSnapshotRequest
	lastPatchCharacterStateRequest *statev1.PatchCharacterStateRequest
	lastUpdateSnapshotStateRequest *statev1.UpdateSnapshotStateRequest
}

// fakeSessionClient implements statev1.SessionServiceClient for tests.
type fakeSessionClient struct {
	statev1.SessionServiceClient // embed for forward-compatibility

	startSessionResponse    *statev1.StartSessionResponse
	endSessionResponse      *statev1.EndSessionResponse
	listSessionsResponse    *statev1.ListSessionsResponse
	getSessionResponse      *statev1.GetSessionResponse
	err                     error
	endSessionErr           error
	listSessionsErr         error
	getSessionErr           error
	lastRequest             *statev1.StartSessionRequest
	lastEndSessionRequest   *statev1.EndSessionRequest
	lastListSessionsRequest *statev1.ListSessionsRequest
	lastGetSessionRequest   *statev1.GetSessionRequest
}

// fakeInteractionClient implements statev1.InteractionServiceClient for tests.
type fakeInteractionClient struct {
	statev1.InteractionServiceClient

	getInteractionStateResponse            *statev1.GetInteractionStateResponse
	setActiveSceneResponse                 *statev1.SetActiveSceneResponse
	startScenePlayerPhaseResponse          *statev1.StartScenePlayerPhaseResponse
	submitScenePlayerPostResponse          *statev1.SubmitScenePlayerPostResponse
	yieldScenePlayerPhaseResponse          *statev1.YieldScenePlayerPhaseResponse
	unyieldScenePlayerPhaseResponse        *statev1.UnyieldScenePlayerPhaseResponse
	acceptScenePlayerPhaseResponse         *statev1.AcceptScenePlayerPhaseResponse
	requestScenePlayerRevisionsResponse    *statev1.RequestScenePlayerRevisionsResponse
	endScenePlayerPhaseResponse            *statev1.EndScenePlayerPhaseResponse
	pauseSessionForOOCResponse             *statev1.PauseSessionForOOCResponse
	postSessionOOCResponse                 *statev1.PostSessionOOCResponse
	markOOCReadyResponse                   *statev1.MarkOOCReadyToResumeResponse
	clearOOCReadyResponse                  *statev1.ClearOOCReadyToResumeResponse
	resumeFromOOCResponse                  *statev1.ResumeFromOOCResponse
	getInteractionStateErr                 error
	setActiveSceneErr                      error
	startScenePlayerPhaseErr               error
	submitScenePlayerPostErr               error
	yieldScenePlayerPhaseErr               error
	unyieldScenePlayerPhaseErr             error
	acceptScenePlayerPhaseErr              error
	requestScenePlayerRevisionsErr         error
	endScenePlayerPhaseErr                 error
	pauseSessionForOOCErr                  error
	postSessionOOCErr                      error
	markOOCReadyErr                        error
	clearOOCReadyErr                       error
	resumeFromOOCErr                       error
	lastGetInteractionStateRequest         *statev1.GetInteractionStateRequest
	lastSetActiveSceneRequest              *statev1.SetActiveSceneRequest
	lastStartScenePlayerPhaseRequest       *statev1.StartScenePlayerPhaseRequest
	lastSubmitScenePlayerPostRequest       *statev1.SubmitScenePlayerPostRequest
	lastYieldScenePlayerPhaseRequest       *statev1.YieldScenePlayerPhaseRequest
	lastUnyieldScenePlayerPhaseRequest     *statev1.UnyieldScenePlayerPhaseRequest
	lastAcceptScenePlayerPhaseRequest      *statev1.AcceptScenePlayerPhaseRequest
	lastRequestScenePlayerRevisionsRequest *statev1.RequestScenePlayerRevisionsRequest
	lastEndScenePlayerPhaseRequest         *statev1.EndScenePlayerPhaseRequest
	lastPauseSessionForOOCRequest          *statev1.PauseSessionForOOCRequest
	lastPostSessionOOCRequest              *statev1.PostSessionOOCRequest
	lastMarkOOCReadyRequest                *statev1.MarkOOCReadyToResumeRequest
	lastClearOOCReadyRequest               *statev1.ClearOOCReadyToResumeRequest
	lastResumeFromOOCRequest               *statev1.ResumeFromOOCRequest
}

// failingTransport returns a connection error for tests.
type failingTransport struct{}

// Connect returns the configured error for tests.
func (f failingTransport) Connect(context.Context) (mcp.Connection, error) {
	return nil, errors.New("transport failure")
}

// CreateCampaign records the request and returns the configured response.
func (f *fakeCampaignClient) CreateCampaign(ctx context.Context, req *statev1.CreateCampaignRequest, opts ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
	f.lastRequest = req
	return f.response, f.err
}

// ListCampaigns records the request and returns the configured response.
func (f *fakeCampaignClient) ListCampaigns(ctx context.Context, req *statev1.ListCampaignsRequest, opts ...grpc.CallOption) (*statev1.ListCampaignsResponse, error) {
	f.lastListRequest = req
	f.listCalls++
	return f.listResponse, f.listErr
}

// GetCampaign records the request and returns the configured response.
func (f *fakeCampaignClient) GetCampaign(ctx context.Context, req *statev1.GetCampaignRequest, opts ...grpc.CallOption) (*statev1.GetCampaignResponse, error) {
	f.lastGetCampaignRequest = req
	return f.getCampaignResponse, f.getCampaignErr
}

// UpdateCampaign records the request and returns a default successful response.
func (f *fakeCampaignClient) UpdateCampaign(ctx context.Context, req *statev1.UpdateCampaignRequest, opts ...grpc.CallOption) (*statev1.UpdateCampaignResponse, error) {
	return &statev1.UpdateCampaignResponse{}, nil
}

// EndCampaign records the request and returns the configured response.
func (f *fakeCampaignClient) EndCampaign(ctx context.Context, req *statev1.EndCampaignRequest, opts ...grpc.CallOption) (*statev1.EndCampaignResponse, error) {
	f.lastEndCampaignRequest = req
	return f.endCampaignResponse, f.endCampaignErr
}

// ArchiveCampaign records the request and returns the configured response.
func (f *fakeCampaignClient) ArchiveCampaign(ctx context.Context, req *statev1.ArchiveCampaignRequest, opts ...grpc.CallOption) (*statev1.ArchiveCampaignResponse, error) {
	f.lastArchiveCampaignRequest = req
	return f.archiveCampaignResponse, f.archiveCampaignErr
}

// RestoreCampaign records the request and returns the configured response.
func (f *fakeCampaignClient) RestoreCampaign(ctx context.Context, req *statev1.RestoreCampaignRequest, opts ...grpc.CallOption) (*statev1.RestoreCampaignResponse, error) {
	f.lastRestoreCampaignRequest = req
	return f.restoreCampaignResponse, f.restoreCampaignErr
}

// SetCampaignCover records the request and returns the configured response.
func (f *fakeCampaignClient) SetCampaignCover(ctx context.Context, req *statev1.SetCampaignCoverRequest, opts ...grpc.CallOption) (*statev1.SetCampaignCoverResponse, error) {
	f.lastSetCampaignCoverRequest = req
	return f.setCampaignCoverResponse, f.setCampaignCoverErr
}

// SetCampaignAIBinding is a stub so fakeCampaignClient satisfies CampaignServiceClient.
func (f *fakeCampaignClient) SetCampaignAIBinding(context.Context, *statev1.SetCampaignAIBindingRequest, ...grpc.CallOption) (*statev1.SetCampaignAIBindingResponse, error) {
	return &statev1.SetCampaignAIBindingResponse{}, nil
}

// ClearCampaignAIBinding is a stub so fakeCampaignClient satisfies CampaignServiceClient.
func (f *fakeCampaignClient) ClearCampaignAIBinding(context.Context, *statev1.ClearCampaignAIBindingRequest, ...grpc.CallOption) (*statev1.ClearCampaignAIBindingResponse, error) {
	return &statev1.ClearCampaignAIBindingResponse{}, nil
}

// GetCampaignAIBindingUsage is a stub so fakeCampaignClient satisfies CampaignServiceClient.
func (f *fakeCampaignClient) GetCampaignAIBindingUsage(context.Context, *statev1.GetCampaignAIBindingUsageRequest, ...grpc.CallOption) (*statev1.GetCampaignAIBindingUsageResponse, error) {
	return &statev1.GetCampaignAIBindingUsageResponse{}, nil
}

// GetCampaignSessionReadiness is a stub so fakeCampaignClient satisfies CampaignServiceClient.
func (f *fakeCampaignClient) GetCampaignSessionReadiness(context.Context, *statev1.GetCampaignSessionReadinessRequest, ...grpc.CallOption) (*statev1.GetCampaignSessionReadinessResponse, error) {
	return &statev1.GetCampaignSessionReadinessResponse{}, nil
}

// CreateParticipant records the request and returns the configured response.
func (f *fakeParticipantClient) CreateParticipant(ctx context.Context, req *statev1.CreateParticipantRequest, opts ...grpc.CallOption) (*statev1.CreateParticipantResponse, error) {
	f.lastCreateParticipantRequest = req
	return f.createParticipantResponse, f.createParticipantErr
}

// UpdateParticipant records the request and returns the configured response.
func (f *fakeParticipantClient) UpdateParticipant(ctx context.Context, req *statev1.UpdateParticipantRequest, opts ...grpc.CallOption) (*statev1.UpdateParticipantResponse, error) {
	f.lastUpdateParticipantRequest = req
	return f.updateParticipantResponse, f.updateParticipantErr
}

// DeleteParticipant records the request and returns the configured response.
func (f *fakeParticipantClient) DeleteParticipant(ctx context.Context, req *statev1.DeleteParticipantRequest, opts ...grpc.CallOption) (*statev1.DeleteParticipantResponse, error) {
	f.lastDeleteParticipantRequest = req
	return f.deleteParticipantResponse, f.deleteParticipantErr
}

// ListParticipants records the request and returns the configured response.
func (f *fakeParticipantClient) ListParticipants(ctx context.Context, req *statev1.ListParticipantsRequest, opts ...grpc.CallOption) (*statev1.ListParticipantsResponse, error) {
	f.lastListParticipantsRequest = req
	return f.listParticipantsResponse, f.listParticipantsErr
}

// GetParticipant records the request and returns the configured response.
func (f *fakeParticipantClient) GetParticipant(ctx context.Context, req *statev1.GetParticipantRequest, opts ...grpc.CallOption) (*statev1.GetParticipantResponse, error) {
	f.lastGetParticipantRequest = req
	return f.getParticipantResponse, f.getParticipantErr
}

// CreateCharacter records the request and returns the configured response.
func (f *fakeCharacterClient) CreateCharacter(ctx context.Context, req *statev1.CreateCharacterRequest, opts ...grpc.CallOption) (*statev1.CreateCharacterResponse, error) {
	f.lastCreateCharacterRequest = req
	return f.createCharacterResponse, f.createCharacterErr
}

// UpdateCharacter records the request and returns the configured response.
func (f *fakeCharacterClient) UpdateCharacter(ctx context.Context, req *statev1.UpdateCharacterRequest, opts ...grpc.CallOption) (*statev1.UpdateCharacterResponse, error) {
	f.lastUpdateCharacterRequest = req
	return f.updateCharacterResponse, f.updateCharacterErr
}

// DeleteCharacter records the request and returns the configured response.
func (f *fakeCharacterClient) DeleteCharacter(ctx context.Context, req *statev1.DeleteCharacterRequest, opts ...grpc.CallOption) (*statev1.DeleteCharacterResponse, error) {
	f.lastDeleteCharacterRequest = req
	return f.deleteCharacterResponse, f.deleteCharacterErr
}

// ListCharacters records the request and returns the configured response.
func (f *fakeCharacterClient) ListCharacters(ctx context.Context, req *statev1.ListCharactersRequest, opts ...grpc.CallOption) (*statev1.ListCharactersResponse, error) {
	f.lastListCharactersRequest = req
	return f.listCharactersResponse, f.listCharactersErr
}

func (f *fakeCharacterClient) ListCharacterProfiles(_ context.Context, _ *statev1.ListCharacterProfilesRequest, _ ...grpc.CallOption) (*statev1.ListCharacterProfilesResponse, error) {
	return &statev1.ListCharacterProfilesResponse{}, nil
}

// SetDefaultControl records the request and returns the configured response.
func (f *fakeCharacterClient) SetDefaultControl(ctx context.Context, req *statev1.SetDefaultControlRequest, opts ...grpc.CallOption) (*statev1.SetDefaultControlResponse, error) {
	f.lastSetDefaultControlRequest = req
	return f.setDefaultControlResponse, f.setDefaultControlErr
}

// ClaimCharacterControl records the request and returns the configured response.
func (f *fakeCharacterClient) ClaimCharacterControl(ctx context.Context, req *statev1.ClaimCharacterControlRequest, opts ...grpc.CallOption) (*statev1.ClaimCharacterControlResponse, error) {
	f.lastClaimCharacterControlRequest = req
	return f.claimCharacterControlResponse, f.claimCharacterControlErr
}

// ReleaseCharacterControl records the request and returns the configured response.
func (f *fakeCharacterClient) ReleaseCharacterControl(ctx context.Context, req *statev1.ReleaseCharacterControlRequest, opts ...grpc.CallOption) (*statev1.ReleaseCharacterControlResponse, error) {
	f.lastReleaseControlRequest = req
	return f.releaseCharacterControlResponse, f.releaseCharacterControlErr
}

// GetCharacterSheet records the request and returns the configured response.
func (f *fakeCharacterClient) GetCharacterSheet(ctx context.Context, req *statev1.GetCharacterSheetRequest, opts ...grpc.CallOption) (*statev1.GetCharacterSheetResponse, error) {
	f.lastGetCharacterSheetRequest = req
	return f.getCharacterSheetResponse, f.getCharacterSheetErr
}

// PatchCharacterProfile records the request and returns the configured response.
func (f *fakeCharacterClient) PatchCharacterProfile(ctx context.Context, req *statev1.PatchCharacterProfileRequest, opts ...grpc.CallOption) (*statev1.PatchCharacterProfileResponse, error) {
	f.lastPatchCharacterProfileRequest = req
	return f.patchCharacterProfileResponse, f.patchCharacterProfileErr
}

// GetCharacterCreationProgress records the request and returns unimplemented for tests that don't use it.
func (f *fakeCharacterClient) GetCharacterCreationProgress(ctx context.Context, req *statev1.GetCharacterCreationProgressRequest, opts ...grpc.CallOption) (*statev1.GetCharacterCreationProgressResponse, error) {
	return nil, errors.New("GetCharacterCreationProgress not implemented")
}

// ApplyCharacterCreationStep records the request and returns unimplemented for tests that don't use it.
func (f *fakeCharacterClient) ApplyCharacterCreationStep(ctx context.Context, req *statev1.ApplyCharacterCreationStepRequest, opts ...grpc.CallOption) (*statev1.ApplyCharacterCreationStepResponse, error) {
	return nil, errors.New("ApplyCharacterCreationStep not implemented")
}

// ApplyCharacterCreationWorkflow records the request and returns unimplemented for tests that don't use it.
func (f *fakeCharacterClient) ApplyCharacterCreationWorkflow(ctx context.Context, req *statev1.ApplyCharacterCreationWorkflowRequest, opts ...grpc.CallOption) (*statev1.ApplyCharacterCreationWorkflowResponse, error) {
	return nil, errors.New("ApplyCharacterCreationWorkflow not implemented")
}

// ResetCharacterCreationWorkflow records the request and returns unimplemented for tests that don't use it.
func (f *fakeCharacterClient) ResetCharacterCreationWorkflow(ctx context.Context, req *statev1.ResetCharacterCreationWorkflowRequest, opts ...grpc.CallOption) (*statev1.ResetCharacterCreationWorkflowResponse, error) {
	return nil, errors.New("ResetCharacterCreationWorkflow not implemented")
}

// GetSnapshot records the request and returns the configured response.
func (f *fakeSnapshotClient) GetSnapshot(ctx context.Context, req *statev1.GetSnapshotRequest, opts ...grpc.CallOption) (*statev1.GetSnapshotResponse, error) {
	f.lastGetSnapshotRequest = req
	return f.getSnapshotResponse, f.getSnapshotErr
}

// PatchCharacterState records the request and returns the configured response.
func (f *fakeSnapshotClient) PatchCharacterState(ctx context.Context, req *statev1.PatchCharacterStateRequest, opts ...grpc.CallOption) (*statev1.PatchCharacterStateResponse, error) {
	f.lastPatchCharacterStateRequest = req
	return f.patchCharacterStateResponse, f.patchCharacterStateErr
}

// UpdateSnapshotState records the request and returns the configured response.
func (f *fakeSnapshotClient) UpdateSnapshotState(ctx context.Context, req *statev1.UpdateSnapshotStateRequest, opts ...grpc.CallOption) (*statev1.UpdateSnapshotStateResponse, error) {
	f.lastUpdateSnapshotStateRequest = req
	return f.updateSnapshotStateResponse, f.updateSnapshotStateErr
}

// StartSession records the request and returns the configured response.
func (f *fakeSessionClient) StartSession(ctx context.Context, req *statev1.StartSessionRequest, opts ...grpc.CallOption) (*statev1.StartSessionResponse, error) {
	f.lastRequest = req
	return f.startSessionResponse, f.err
}

// EndSession records the request and returns the configured response.
func (f *fakeSessionClient) EndSession(ctx context.Context, req *statev1.EndSessionRequest, opts ...grpc.CallOption) (*statev1.EndSessionResponse, error) {
	f.lastEndSessionRequest = req
	return f.endSessionResponse, f.endSessionErr
}

// ListSessions records the request and returns the configured response.
func (f *fakeSessionClient) ListSessions(ctx context.Context, req *statev1.ListSessionsRequest, opts ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
	f.lastListSessionsRequest = req
	return f.listSessionsResponse, f.listSessionsErr
}

// GetSession records the request and returns the configured response.
func (f *fakeSessionClient) GetSession(ctx context.Context, req *statev1.GetSessionRequest, opts ...grpc.CallOption) (*statev1.GetSessionResponse, error) {
	f.lastGetSessionRequest = req
	return f.getSessionResponse, f.getSessionErr
}

func (f *fakeInteractionClient) GetInteractionState(ctx context.Context, req *statev1.GetInteractionStateRequest, opts ...grpc.CallOption) (*statev1.GetInteractionStateResponse, error) {
	f.lastGetInteractionStateRequest = req
	return f.getInteractionStateResponse, f.getInteractionStateErr
}

func (f *fakeInteractionClient) SetActiveScene(ctx context.Context, req *statev1.SetActiveSceneRequest, opts ...grpc.CallOption) (*statev1.SetActiveSceneResponse, error) {
	f.lastSetActiveSceneRequest = req
	return f.setActiveSceneResponse, f.setActiveSceneErr
}

func (f *fakeInteractionClient) StartScenePlayerPhase(ctx context.Context, req *statev1.StartScenePlayerPhaseRequest, opts ...grpc.CallOption) (*statev1.StartScenePlayerPhaseResponse, error) {
	f.lastStartScenePlayerPhaseRequest = req
	return f.startScenePlayerPhaseResponse, f.startScenePlayerPhaseErr
}

func (f *fakeInteractionClient) SubmitScenePlayerPost(ctx context.Context, req *statev1.SubmitScenePlayerPostRequest, opts ...grpc.CallOption) (*statev1.SubmitScenePlayerPostResponse, error) {
	f.lastSubmitScenePlayerPostRequest = req
	return f.submitScenePlayerPostResponse, f.submitScenePlayerPostErr
}

func (f *fakeInteractionClient) YieldScenePlayerPhase(ctx context.Context, req *statev1.YieldScenePlayerPhaseRequest, opts ...grpc.CallOption) (*statev1.YieldScenePlayerPhaseResponse, error) {
	f.lastYieldScenePlayerPhaseRequest = req
	return f.yieldScenePlayerPhaseResponse, f.yieldScenePlayerPhaseErr
}

func (f *fakeInteractionClient) UnyieldScenePlayerPhase(ctx context.Context, req *statev1.UnyieldScenePlayerPhaseRequest, opts ...grpc.CallOption) (*statev1.UnyieldScenePlayerPhaseResponse, error) {
	f.lastUnyieldScenePlayerPhaseRequest = req
	return f.unyieldScenePlayerPhaseResponse, f.unyieldScenePlayerPhaseErr
}

func (f *fakeInteractionClient) AcceptScenePlayerPhase(ctx context.Context, req *statev1.AcceptScenePlayerPhaseRequest, opts ...grpc.CallOption) (*statev1.AcceptScenePlayerPhaseResponse, error) {
	f.lastAcceptScenePlayerPhaseRequest = req
	return f.acceptScenePlayerPhaseResponse, f.acceptScenePlayerPhaseErr
}

func (f *fakeInteractionClient) RequestScenePlayerRevisions(ctx context.Context, req *statev1.RequestScenePlayerRevisionsRequest, opts ...grpc.CallOption) (*statev1.RequestScenePlayerRevisionsResponse, error) {
	f.lastRequestScenePlayerRevisionsRequest = req
	return f.requestScenePlayerRevisionsResponse, f.requestScenePlayerRevisionsErr
}

func (f *fakeInteractionClient) EndScenePlayerPhase(ctx context.Context, req *statev1.EndScenePlayerPhaseRequest, opts ...grpc.CallOption) (*statev1.EndScenePlayerPhaseResponse, error) {
	f.lastEndScenePlayerPhaseRequest = req
	return f.endScenePlayerPhaseResponse, f.endScenePlayerPhaseErr
}

func (f *fakeInteractionClient) PauseSessionForOOC(ctx context.Context, req *statev1.PauseSessionForOOCRequest, opts ...grpc.CallOption) (*statev1.PauseSessionForOOCResponse, error) {
	f.lastPauseSessionForOOCRequest = req
	return f.pauseSessionForOOCResponse, f.pauseSessionForOOCErr
}

func (f *fakeInteractionClient) PostSessionOOC(ctx context.Context, req *statev1.PostSessionOOCRequest, opts ...grpc.CallOption) (*statev1.PostSessionOOCResponse, error) {
	f.lastPostSessionOOCRequest = req
	return f.postSessionOOCResponse, f.postSessionOOCErr
}

func (f *fakeInteractionClient) MarkOOCReadyToResume(ctx context.Context, req *statev1.MarkOOCReadyToResumeRequest, opts ...grpc.CallOption) (*statev1.MarkOOCReadyToResumeResponse, error) {
	f.lastMarkOOCReadyRequest = req
	return f.markOOCReadyResponse, f.markOOCReadyErr
}

func (f *fakeInteractionClient) ClearOOCReadyToResume(ctx context.Context, req *statev1.ClearOOCReadyToResumeRequest, opts ...grpc.CallOption) (*statev1.ClearOOCReadyToResumeResponse, error) {
	f.lastClearOOCReadyRequest = req
	return f.clearOOCReadyResponse, f.clearOOCReadyErr
}

func (f *fakeInteractionClient) ResumeFromOOC(ctx context.Context, req *statev1.ResumeFromOOCRequest, opts ...grpc.CallOption) (*statev1.ResumeFromOOCResponse, error) {
	f.lastResumeFromOOCRequest = req
	return f.resumeFromOOCResponse, f.resumeFromOOCErr
}

// intPointer returns an int pointer for test inputs.
func intPointer(value int) *int {
	return &value
}
