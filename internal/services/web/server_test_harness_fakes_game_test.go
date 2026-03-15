package web

import (
	"context"
	"strings"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc"
)

type fakeCampaignClient struct {
	response      *statev1.ListCampaignsResponse
	err           error
	getResp       *statev1.GetCampaignResponse
	getErr        error
	readinessResp *statev1.GetCampaignSessionReadinessResponse
	readinessErr  error
	createResp    *statev1.CreateCampaignResponse
	createErr     error
}

type fakeWebInteractionClient struct {
	response *statev1.GetInteractionStateResponse
	err      error
}

func (f fakeWebInteractionClient) GetInteractionState(context.Context, *statev1.GetInteractionStateRequest, ...grpc.CallOption) (*statev1.GetInteractionStateResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.GetInteractionStateResponse{
		State: &statev1.InteractionState{
			Viewer: &statev1.InteractionViewer{
				ParticipantId: "p1",
				Name:          "Owner",
				Role:          statev1.ParticipantRole_PLAYER,
			},
			ActiveSession: &statev1.InteractionSession{
				SessionId: "sess-1",
				Name:      "Session One",
			},
			ActiveScene: &statev1.InteractionScene{
				SceneId:     "scene-1",
				SessionId:   "sess-1",
				Name:        "Bridge",
				Description: "The bridge shudders under the party's weight.",
				Characters: []*statev1.InteractionCharacter{
					{
						CharacterId:        "char-1",
						Name:               "Aria",
						OwnerParticipantId: "p1",
					},
				},
			},
			PlayerPhase: &statev1.ScenePlayerPhase{
				PhaseId:              "phase-1",
				Status:               statev1.ScenePhaseStatus_SCENE_PHASE_STATUS_PLAYERS,
				FrameText:            "The bridge starts to crack. What do you do?",
				ActingCharacterIds:   []string{"char-1"},
				ActingParticipantIds: []string{"p1"},
				Slots: []*statev1.ScenePlayerSlot{
					{
						ParticipantId: "p1",
						SummaryText:   "Aria braces against the railing and looks for a safe crossing.",
						CharacterIds:  []string{"char-1"},
						Yielded:       true,
					},
				},
			},
			Ooc: &statev1.OOCState{
				Posts: []*statev1.OOCPost{
					{
						PostId:        "ooc-1",
						ParticipantId: "p1",
						Body:          "Quick ruling check before we commit.",
					},
				},
			},
		},
	}, nil
}

func (f fakeCampaignClient) ListCampaigns(context.Context, *statev1.ListCampaignsRequest, ...grpc.CallOption) (*statev1.ListCampaignsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.response, nil
}

func (f fakeCampaignClient) GetCampaign(context.Context, *statev1.GetCampaignRequest, ...grpc.CallOption) (*statev1.GetCampaignResponse, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	if f.getResp != nil {
		return f.getResp, nil
	}
	return &statev1.GetCampaignResponse{Campaign: &statev1.Campaign{Id: "c1", Name: "Campaign"}}, nil
}

func (f fakeCampaignClient) GetCampaignSessionReadiness(context.Context, *statev1.GetCampaignSessionReadinessRequest, ...grpc.CallOption) (*statev1.GetCampaignSessionReadinessResponse, error) {
	if f.readinessErr != nil {
		return nil, f.readinessErr
	}
	if f.readinessResp != nil {
		return f.readinessResp, nil
	}
	return &statev1.GetCampaignSessionReadinessResponse{
		Readiness: &statev1.CampaignSessionReadiness{Ready: true},
	}, nil
}

func (f fakeCampaignClient) CreateCampaign(context.Context, *statev1.CreateCampaignRequest, ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	if f.createResp != nil {
		return f.createResp, nil
	}
	return &statev1.CreateCampaignResponse{Campaign: &statev1.Campaign{Id: "created"}}, nil
}

func (f fakeCampaignClient) UpdateCampaign(context.Context, *statev1.UpdateCampaignRequest, ...grpc.CallOption) (*statev1.UpdateCampaignResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.UpdateCampaignResponse{}, nil
}

func (f fakeCampaignClient) SetCampaignAIBinding(context.Context, *statev1.SetCampaignAIBindingRequest, ...grpc.CallOption) (*statev1.SetCampaignAIBindingResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.SetCampaignAIBindingResponse{}, nil
}

func (f fakeCampaignClient) ClearCampaignAIBinding(context.Context, *statev1.ClearCampaignAIBindingRequest, ...grpc.CallOption) (*statev1.ClearCampaignAIBindingResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.ClearCampaignAIBindingResponse{}, nil
}

type fakeWebParticipantClient struct {
	response *statev1.ListParticipantsResponse
	err      error
}

func (f fakeWebParticipantClient) ListParticipants(context.Context, *statev1.ListParticipantsRequest, ...grpc.CallOption) (*statev1.ListParticipantsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.ListParticipantsResponse{}, nil
}

func (f fakeWebParticipantClient) GetParticipant(context.Context, *statev1.GetParticipantRequest, ...grpc.CallOption) (*statev1.GetParticipantResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		for _, participant := range f.response.GetParticipants() {
			if participant != nil {
				return &statev1.GetParticipantResponse{Participant: participant}, nil
			}
		}
	}
	return &statev1.GetParticipantResponse{}, nil
}

func (f fakeWebParticipantClient) CreateParticipant(context.Context, *statev1.CreateParticipantRequest, ...grpc.CallOption) (*statev1.CreateParticipantResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.CreateParticipantResponse{Participant: &statev1.Participant{Id: "participant-created"}}, nil
}

func (f fakeWebParticipantClient) UpdateParticipant(context.Context, *statev1.UpdateParticipantRequest, ...grpc.CallOption) (*statev1.UpdateParticipantResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.UpdateParticipantResponse{}, nil
}

type fakeWebCharacterClient struct {
	response *statev1.ListCharactersResponse
	err      error
}

func (f fakeWebCharacterClient) ListCharacters(context.Context, *statev1.ListCharactersRequest, ...grpc.CallOption) (*statev1.ListCharactersResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.ListCharactersResponse{}, nil
}

func (f fakeWebCharacterClient) ListCharacterProfiles(context.Context, *statev1.ListCharacterProfilesRequest, ...grpc.CallOption) (*statev1.ListCharacterProfilesResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.ListCharacterProfilesResponse{}, nil
}

func (f fakeWebCharacterClient) CreateCharacter(context.Context, *statev1.CreateCharacterRequest, ...grpc.CallOption) (*statev1.CreateCharacterResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.CreateCharacterResponse{Character: &statev1.Character{Id: "char-created"}}, nil
}

func (f fakeWebCharacterClient) UpdateCharacter(context.Context, *statev1.UpdateCharacterRequest, ...grpc.CallOption) (*statev1.UpdateCharacterResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.UpdateCharacterResponse{}, nil
}

func (f fakeWebCharacterClient) DeleteCharacter(context.Context, *statev1.DeleteCharacterRequest, ...grpc.CallOption) (*statev1.DeleteCharacterResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.DeleteCharacterResponse{}, nil
}

func (f fakeWebCharacterClient) SetDefaultControl(context.Context, *statev1.SetDefaultControlRequest, ...grpc.CallOption) (*statev1.SetDefaultControlResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.SetDefaultControlResponse{}, nil
}

func (f fakeWebCharacterClient) ClaimCharacterControl(context.Context, *statev1.ClaimCharacterControlRequest, ...grpc.CallOption) (*statev1.ClaimCharacterControlResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.ClaimCharacterControlResponse{}, nil
}

func (f fakeWebCharacterClient) ReleaseCharacterControl(context.Context, *statev1.ReleaseCharacterControlRequest, ...grpc.CallOption) (*statev1.ReleaseCharacterControlResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.ReleaseCharacterControlResponse{}, nil
}

func (f fakeWebCharacterClient) GetCharacterSheet(_ context.Context, req *statev1.GetCharacterSheetRequest, _ ...grpc.CallOption) (*statev1.GetCharacterSheetResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		characterID := strings.TrimSpace(req.GetCharacterId())
		for _, character := range f.response.GetCharacters() {
			if character == nil {
				continue
			}
			if strings.TrimSpace(character.GetId()) == characterID {
				return &statev1.GetCharacterSheetResponse{Character: character}, nil
			}
		}
	}
	return &statev1.GetCharacterSheetResponse{}, nil
}

func (f fakeWebCharacterClient) GetCharacterCreationProgress(context.Context, *statev1.GetCharacterCreationProgressRequest, ...grpc.CallOption) (*statev1.GetCharacterCreationProgressResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.GetCharacterCreationProgressResponse{}, nil
}

func (f fakeWebCharacterClient) ApplyCharacterCreationStep(context.Context, *statev1.ApplyCharacterCreationStepRequest, ...grpc.CallOption) (*statev1.ApplyCharacterCreationStepResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.ApplyCharacterCreationStepResponse{}, nil
}

func (f fakeWebCharacterClient) ResetCharacterCreationWorkflow(context.Context, *statev1.ResetCharacterCreationWorkflowRequest, ...grpc.CallOption) (*statev1.ResetCharacterCreationWorkflowResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.ResetCharacterCreationWorkflowResponse{}, nil
}

type fakeWebSessionClient struct {
	response *statev1.ListSessionsResponse
	err      error
}

func (f fakeWebSessionClient) ListSessions(context.Context, *statev1.ListSessionsRequest, ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.ListSessionsResponse{}, nil
}

func (f fakeWebSessionClient) StartSession(_ context.Context, req *statev1.StartSessionRequest, _ ...grpc.CallOption) (*statev1.StartSessionResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.StartSessionResponse{Session: &statev1.Session{
		Id:         "sess-started",
		CampaignId: strings.TrimSpace(req.GetCampaignId()),
		Name:       strings.TrimSpace(req.GetName()),
		Status:     statev1.SessionStatus_SESSION_ACTIVE,
	}}, nil
}

func (f fakeWebSessionClient) EndSession(_ context.Context, req *statev1.EndSessionRequest, _ ...grpc.CallOption) (*statev1.EndSessionResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.EndSessionResponse{Session: &statev1.Session{
		Id:         strings.TrimSpace(req.GetSessionId()),
		CampaignId: strings.TrimSpace(req.GetCampaignId()),
		Status:     statev1.SessionStatus_SESSION_ENDED,
	}}, nil
}

type fakeWebInviteClient struct {
	response *statev1.ListInvitesResponse
	err      error
}

func (f fakeWebInviteClient) ListInvites(context.Context, *statev1.ListInvitesRequest, ...grpc.CallOption) (*statev1.ListInvitesResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.ListInvitesResponse{}, nil
}

func (f fakeWebInviteClient) GetPublicInvite(context.Context, *statev1.GetPublicInviteRequest, ...grpc.CallOption) (*statev1.GetPublicInviteResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.GetPublicInviteResponse{Invite: &statev1.Invite{}}, nil
}

func (f fakeWebInviteClient) CreateInvite(_ context.Context, req *statev1.CreateInviteRequest, _ ...grpc.CallOption) (*statev1.CreateInviteResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.CreateInviteResponse{Invite: &statev1.Invite{
		Id:              "inv-created",
		CampaignId:      strings.TrimSpace(req.GetCampaignId()),
		ParticipantId:   strings.TrimSpace(req.GetParticipantId()),
		RecipientUserId: strings.TrimSpace(req.GetRecipientUserId()),
		Status:          statev1.InviteStatus_PENDING,
	}}, nil
}

func (f fakeWebInviteClient) ClaimInvite(context.Context, *statev1.ClaimInviteRequest, ...grpc.CallOption) (*statev1.ClaimInviteResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.ClaimInviteResponse{}, nil
}

func (f fakeWebInviteClient) DeclineInvite(context.Context, *statev1.DeclineInviteRequest, ...grpc.CallOption) (*statev1.DeclineInviteResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.DeclineInviteResponse{}, nil
}

func (f fakeWebInviteClient) RevokeInvite(_ context.Context, req *statev1.RevokeInviteRequest, _ ...grpc.CallOption) (*statev1.RevokeInviteResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return &statev1.RevokeInviteResponse{Invite: &statev1.Invite{
		Id:     strings.TrimSpace(req.GetInviteId()),
		Status: statev1.InviteStatus_REVOKED,
	}}, nil
}

type fakeWebDaggerheartContentClient struct {
	response *daggerheartv1.GetDaggerheartContentCatalogResponse
	err      error
}

func (f fakeWebDaggerheartContentClient) GetContentCatalog(context.Context, *daggerheartv1.GetDaggerheartContentCatalogRequest, ...grpc.CallOption) (*daggerheartv1.GetDaggerheartContentCatalogResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &daggerheartv1.GetDaggerheartContentCatalogResponse{Catalog: &daggerheartv1.DaggerheartContentCatalog{}}, nil
}

type fakeWebDaggerheartAssetClient struct {
	response *daggerheartv1.GetDaggerheartAssetMapResponse
	err      error
}

func (f fakeWebDaggerheartAssetClient) GetAssetMap(context.Context, *daggerheartv1.GetDaggerheartAssetMapRequest, ...grpc.CallOption) (*daggerheartv1.GetDaggerheartAssetMapResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	if f.response != nil {
		return f.response, nil
	}
	return &daggerheartv1.GetDaggerheartAssetMapResponse{}, nil
}

type fakeWebAuthorizationClient struct{}

func (fakeWebAuthorizationClient) Can(context.Context, *statev1.CanRequest, ...grpc.CallOption) (*statev1.CanResponse, error) {
	return &statev1.CanResponse{Allowed: true, ReasonCode: "AUTHZ_ALLOW_ACCESS_LEVEL"}, nil
}

func (fakeWebAuthorizationClient) BatchCan(context.Context, *statev1.BatchCanRequest, ...grpc.CallOption) (*statev1.BatchCanResponse, error) {
	return &statev1.BatchCanResponse{}, nil
}
