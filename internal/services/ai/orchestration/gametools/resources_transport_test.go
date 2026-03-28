package gametools

import (
	"context"
	"strings"
	"testing"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/platform/grpcmeta"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type campaignClientStub struct {
	getResp      *statev1.GetCampaignResponse
	getErr       error
	lastCampaign string
	lastMetadata metadata.MD
}

func (stub *campaignClientStub) CreateCampaign(context.Context, *statev1.CreateCampaignRequest, ...grpc.CallOption) (*statev1.CreateCampaignResponse, error) {
	return nil, nil
}

func (stub *campaignClientStub) ListCampaigns(context.Context, *statev1.ListCampaignsRequest, ...grpc.CallOption) (*statev1.ListCampaignsResponse, error) {
	return nil, nil
}

func (stub *campaignClientStub) GetCampaign(ctx context.Context, req *statev1.GetCampaignRequest, _ ...grpc.CallOption) (*statev1.GetCampaignResponse, error) {
	stub.lastCampaign = req.GetCampaignId()
	stub.lastMetadata, _ = metadata.FromOutgoingContext(ctx)
	return stub.getResp, stub.getErr
}

func (stub *campaignClientStub) GetCampaignSessionReadiness(context.Context, *statev1.GetCampaignSessionReadinessRequest, ...grpc.CallOption) (*statev1.GetCampaignSessionReadinessResponse, error) {
	return nil, nil
}

func (stub *campaignClientStub) UpdateCampaign(context.Context, *statev1.UpdateCampaignRequest, ...grpc.CallOption) (*statev1.UpdateCampaignResponse, error) {
	return nil, nil
}

func (stub *campaignClientStub) EndCampaign(context.Context, *statev1.EndCampaignRequest, ...grpc.CallOption) (*statev1.EndCampaignResponse, error) {
	return nil, nil
}

func (stub *campaignClientStub) ArchiveCampaign(context.Context, *statev1.ArchiveCampaignRequest, ...grpc.CallOption) (*statev1.ArchiveCampaignResponse, error) {
	return nil, nil
}

func (stub *campaignClientStub) RestoreCampaign(context.Context, *statev1.RestoreCampaignRequest, ...grpc.CallOption) (*statev1.RestoreCampaignResponse, error) {
	return nil, nil
}

func (stub *campaignClientStub) SetCampaignCover(context.Context, *statev1.SetCampaignCoverRequest, ...grpc.CallOption) (*statev1.SetCampaignCoverResponse, error) {
	return nil, nil
}

func (stub *campaignClientStub) SetCampaignAIBinding(context.Context, *statev1.SetCampaignAIBindingRequest, ...grpc.CallOption) (*statev1.SetCampaignAIBindingResponse, error) {
	return nil, nil
}

func (stub *campaignClientStub) ClearCampaignAIBinding(context.Context, *statev1.ClearCampaignAIBindingRequest, ...grpc.CallOption) (*statev1.ClearCampaignAIBindingResponse, error) {
	return nil, nil
}

type participantClientStub struct {
	listResp     *statev1.ListParticipantsResponse
	listErr      error
	lastRequest  *statev1.ListParticipantsRequest
	lastMetadata metadata.MD
}

func (stub *participantClientStub) CreateParticipant(context.Context, *statev1.CreateParticipantRequest, ...grpc.CallOption) (*statev1.CreateParticipantResponse, error) {
	return nil, nil
}

func (stub *participantClientStub) UpdateParticipant(context.Context, *statev1.UpdateParticipantRequest, ...grpc.CallOption) (*statev1.UpdateParticipantResponse, error) {
	return nil, nil
}

func (stub *participantClientStub) DeleteParticipant(context.Context, *statev1.DeleteParticipantRequest, ...grpc.CallOption) (*statev1.DeleteParticipantResponse, error) {
	return nil, nil
}

func (stub *participantClientStub) ListParticipants(ctx context.Context, req *statev1.ListParticipantsRequest, _ ...grpc.CallOption) (*statev1.ListParticipantsResponse, error) {
	stub.lastRequest = req
	stub.lastMetadata, _ = metadata.FromOutgoingContext(ctx)
	return stub.listResp, stub.listErr
}

func (stub *participantClientStub) GetParticipant(context.Context, *statev1.GetParticipantRequest, ...grpc.CallOption) (*statev1.GetParticipantResponse, error) {
	return nil, nil
}

func (stub *participantClientStub) BindParticipant(context.Context, *statev1.BindParticipantRequest, ...grpc.CallOption) (*statev1.BindParticipantResponse, error) {
	return nil, nil
}

type characterClientStub struct {
	listResp    *statev1.ListCharactersResponse
	listErr     error
	lastRequest *statev1.ListCharactersRequest
}

func (stub *characterClientStub) CreateCharacter(context.Context, *statev1.CreateCharacterRequest, ...grpc.CallOption) (*statev1.CreateCharacterResponse, error) {
	return nil, nil
}

func (stub *characterClientStub) UpdateCharacter(context.Context, *statev1.UpdateCharacterRequest, ...grpc.CallOption) (*statev1.UpdateCharacterResponse, error) {
	return nil, nil
}

func (stub *characterClientStub) DeleteCharacter(context.Context, *statev1.DeleteCharacterRequest, ...grpc.CallOption) (*statev1.DeleteCharacterResponse, error) {
	return nil, nil
}

func (stub *characterClientStub) ListCharacters(context.Context, *statev1.ListCharactersRequest, ...grpc.CallOption) (*statev1.ListCharactersResponse, error) {
	return stub.listResp, stub.listErr
}

func (stub *characterClientStub) ListCharacterProfiles(context.Context, *statev1.ListCharacterProfilesRequest, ...grpc.CallOption) (*statev1.ListCharacterProfilesResponse, error) {
	return nil, nil
}

func (stub *characterClientStub) GetCharacterSheet(context.Context, *statev1.GetCharacterSheetRequest, ...grpc.CallOption) (*statev1.GetCharacterSheetResponse, error) {
	return nil, nil
}

func (stub *characterClientStub) PatchCharacterProfile(context.Context, *statev1.PatchCharacterProfileRequest, ...grpc.CallOption) (*statev1.PatchCharacterProfileResponse, error) {
	return nil, nil
}

func (stub *characterClientStub) GetCharacterCreationProgress(context.Context, *statev1.GetCharacterCreationProgressRequest, ...grpc.CallOption) (*statev1.GetCharacterCreationProgressResponse, error) {
	return nil, nil
}

func (stub *characterClientStub) ApplyCharacterCreationStep(context.Context, *statev1.ApplyCharacterCreationStepRequest, ...grpc.CallOption) (*statev1.ApplyCharacterCreationStepResponse, error) {
	return nil, nil
}

func (stub *characterClientStub) ApplyCharacterCreationWorkflow(context.Context, *statev1.ApplyCharacterCreationWorkflowRequest, ...grpc.CallOption) (*statev1.ApplyCharacterCreationWorkflowResponse, error) {
	return nil, nil
}

func (stub *characterClientStub) ResetCharacterCreationWorkflow(context.Context, *statev1.ResetCharacterCreationWorkflowRequest, ...grpc.CallOption) (*statev1.ResetCharacterCreationWorkflowResponse, error) {
	return nil, nil
}

type sessionClientStub struct {
	listResp    *statev1.ListSessionsResponse
	listErr     error
	recapResp   *statev1.GetSessionRecapResponse
	recapErr    error
	lastRequest *statev1.ListSessionsRequest
}

func (stub *sessionClientStub) StartSession(context.Context, *statev1.StartSessionRequest, ...grpc.CallOption) (*statev1.StartSessionResponse, error) {
	return nil, nil
}

func (stub *sessionClientStub) ListSessions(context.Context, *statev1.ListSessionsRequest, ...grpc.CallOption) (*statev1.ListSessionsResponse, error) {
	return stub.listResp, stub.listErr
}

func (stub *sessionClientStub) ListActiveSessionsForUser(context.Context, *statev1.ListActiveSessionsForUserRequest, ...grpc.CallOption) (*statev1.ListActiveSessionsForUserResponse, error) {
	return nil, nil
}

func (stub *sessionClientStub) GetSession(context.Context, *statev1.GetSessionRequest, ...grpc.CallOption) (*statev1.GetSessionResponse, error) {
	return nil, nil
}

func (stub *sessionClientStub) GetSessionRecap(context.Context, *statev1.GetSessionRecapRequest, ...grpc.CallOption) (*statev1.GetSessionRecapResponse, error) {
	return stub.recapResp, stub.recapErr
}

func (stub *sessionClientStub) EndSession(context.Context, *statev1.EndSessionRequest, ...grpc.CallOption) (*statev1.EndSessionResponse, error) {
	return nil, nil
}

func (stub *sessionClientStub) OpenSessionGate(context.Context, *statev1.OpenSessionGateRequest, ...grpc.CallOption) (*statev1.OpenSessionGateResponse, error) {
	return nil, nil
}

func (stub *sessionClientStub) ResolveSessionGate(context.Context, *statev1.ResolveSessionGateRequest, ...grpc.CallOption) (*statev1.ResolveSessionGateResponse, error) {
	return nil, nil
}

func (stub *sessionClientStub) AbandonSessionGate(context.Context, *statev1.AbandonSessionGateRequest, ...grpc.CallOption) (*statev1.AbandonSessionGateResponse, error) {
	return nil, nil
}

func (stub *sessionClientStub) GetSessionSpotlight(context.Context, *statev1.GetSessionSpotlightRequest, ...grpc.CallOption) (*statev1.GetSessionSpotlightResponse, error) {
	return nil, nil
}

func (stub *sessionClientStub) SetSessionSpotlight(context.Context, *statev1.SetSessionSpotlightRequest, ...grpc.CallOption) (*statev1.SetSessionSpotlightResponse, error) {
	return nil, nil
}

func (stub *sessionClientStub) ClearSessionSpotlight(context.Context, *statev1.ClearSessionSpotlightRequest, ...grpc.CallOption) (*statev1.ClearSessionSpotlightResponse, error) {
	return nil, nil
}

type sceneClientStub struct {
	createResp        *statev1.CreateSceneResponse
	createErr         error
	updateResp        *statev1.UpdateSceneResponse
	updateErr         error
	endErr            error
	transitionResp    *statev1.TransitionSceneResponse
	transitionErr     error
	addErr            error
	removeErr         error
	listResp          *statev1.ListScenesResponse
	listErr           error
	lastCreateRequest *statev1.CreateSceneRequest
	lastUpdateRequest *statev1.UpdateSceneRequest
	lastEndRequest    *statev1.EndSceneRequest
	lastAddRequest    *statev1.AddCharacterToSceneRequest
	lastRemoveRequest *statev1.RemoveCharacterFromSceneRequest
	lastListRequest   *statev1.ListScenesRequest
	lastTransitionReq *statev1.TransitionSceneRequest
	lastMetadata      metadata.MD
}

func (stub *sceneClientStub) CreateScene(ctx context.Context, req *statev1.CreateSceneRequest, _ ...grpc.CallOption) (*statev1.CreateSceneResponse, error) {
	stub.lastCreateRequest = req
	stub.lastMetadata, _ = metadata.FromOutgoingContext(ctx)
	return stub.createResp, stub.createErr
}

func (stub *sceneClientStub) UpdateScene(ctx context.Context, req *statev1.UpdateSceneRequest, _ ...grpc.CallOption) (*statev1.UpdateSceneResponse, error) {
	stub.lastUpdateRequest = req
	stub.lastMetadata, _ = metadata.FromOutgoingContext(ctx)
	if stub.updateResp != nil || stub.updateErr != nil {
		return stub.updateResp, stub.updateErr
	}
	return &statev1.UpdateSceneResponse{}, nil
}

func (stub *sceneClientStub) EndScene(ctx context.Context, req *statev1.EndSceneRequest, _ ...grpc.CallOption) (*statev1.EndSceneResponse, error) {
	stub.lastEndRequest = req
	stub.lastMetadata, _ = metadata.FromOutgoingContext(ctx)
	return &statev1.EndSceneResponse{}, stub.endErr
}

func (stub *sceneClientStub) AddCharacterToScene(ctx context.Context, req *statev1.AddCharacterToSceneRequest, _ ...grpc.CallOption) (*statev1.AddCharacterToSceneResponse, error) {
	stub.lastAddRequest = req
	stub.lastMetadata, _ = metadata.FromOutgoingContext(ctx)
	return &statev1.AddCharacterToSceneResponse{}, stub.addErr
}

func (stub *sceneClientStub) RemoveCharacterFromScene(ctx context.Context, req *statev1.RemoveCharacterFromSceneRequest, _ ...grpc.CallOption) (*statev1.RemoveCharacterFromSceneResponse, error) {
	stub.lastRemoveRequest = req
	stub.lastMetadata, _ = metadata.FromOutgoingContext(ctx)
	return &statev1.RemoveCharacterFromSceneResponse{}, stub.removeErr
}

func (stub *sceneClientStub) TransferCharacter(context.Context, *statev1.TransferCharacterRequest, ...grpc.CallOption) (*statev1.TransferCharacterResponse, error) {
	return nil, nil
}

func (stub *sceneClientStub) TransitionScene(ctx context.Context, req *statev1.TransitionSceneRequest, _ ...grpc.CallOption) (*statev1.TransitionSceneResponse, error) {
	stub.lastTransitionReq = req
	stub.lastMetadata, _ = metadata.FromOutgoingContext(ctx)
	return stub.transitionResp, stub.transitionErr
}

func (stub *sceneClientStub) OpenSceneGate(context.Context, *statev1.OpenSceneGateRequest, ...grpc.CallOption) (*statev1.OpenSceneGateResponse, error) {
	return nil, nil
}

func (stub *sceneClientStub) ResolveSceneGate(context.Context, *statev1.ResolveSceneGateRequest, ...grpc.CallOption) (*statev1.ResolveSceneGateResponse, error) {
	return nil, nil
}

func (stub *sceneClientStub) AbandonSceneGate(context.Context, *statev1.AbandonSceneGateRequest, ...grpc.CallOption) (*statev1.AbandonSceneGateResponse, error) {
	return nil, nil
}

func (stub *sceneClientStub) SetSceneSpotlight(context.Context, *statev1.SetSceneSpotlightRequest, ...grpc.CallOption) (*statev1.SetSceneSpotlightResponse, error) {
	return nil, nil
}

func (stub *sceneClientStub) ClearSceneSpotlight(context.Context, *statev1.ClearSceneSpotlightRequest, ...grpc.CallOption) (*statev1.ClearSceneSpotlightResponse, error) {
	return nil, nil
}

func (stub *sceneClientStub) GetScene(context.Context, *statev1.GetSceneRequest, ...grpc.CallOption) (*statev1.GetSceneResponse, error) {
	return nil, nil
}

func (stub *sceneClientStub) ListScenes(ctx context.Context, req *statev1.ListScenesRequest, _ ...grpc.CallOption) (*statev1.ListScenesResponse, error) {
	stub.lastListRequest = req
	stub.lastMetadata, _ = metadata.FromOutgoingContext(ctx)
	return stub.listResp, stub.listErr
}

func TestReadCampaignTransportPaths(t *testing.T) {
	t.Run("not found maps to stable error", func(t *testing.T) {
		session := NewDirectSession(Clients{
			Campaign: &campaignClientStub{getErr: status.Error(codes.NotFound, "missing")},
		}, SessionContext{})

		_, err := session.readCampaign(context.Background(), "campaign://camp-1")
		if err == nil || err.Error() != "campaign not found" {
			t.Fatalf("readCampaign() error = %v", err)
		}
	})

	t.Run("success shapes payload and metadata", func(t *testing.T) {
		client := &campaignClientStub{getResp: &statev1.GetCampaignResponse{
			Campaign: &statev1.Campaign{
				Id:        "camp-1",
				Name:      "Broken Spire",
				Status:    statev1.CampaignStatus_ACTIVE,
				CreatedAt: testTS,
				UpdatedAt: testTS,
			},
		}}
		session := NewDirectSession(Clients{Campaign: client}, SessionContext{
			CampaignID:    "ctx-campaign",
			SessionID:     "sess-1",
			ParticipantID: "part-1",
		})

		result, err := session.readCampaign(context.Background(), "campaign://camp-1")
		if err != nil {
			t.Fatalf("readCampaign() error = %v", err)
		}

		payload := decodeToolOutput[campaignPayload](t, result)
		if payload.Campaign.Name != "Broken Spire" {
			t.Fatalf("campaign payload = %#v", payload.Campaign)
		}
		if client.lastCampaign != "camp-1" {
			t.Fatalf("campaign ID sent = %q, want camp-1", client.lastCampaign)
		}
		if got := client.lastMetadata.Get(grpcmeta.ParticipantIDHeader); len(got) != 1 || got[0] != "part-1" {
			t.Fatalf("participant metadata = %#v", got)
		}
	})
}

func TestReadListResourcesAndInteraction(t *testing.T) {
	t.Run("participants", func(t *testing.T) {
		client := &participantClientStub{listResp: &statev1.ListParticipantsResponse{
			Participants: []*statev1.Participant{{
				Id:         "p-1",
				CampaignId: "camp-1",
				Name:       "Morgan",
				Role:       statev1.ParticipantRole_GM,
				CreatedAt:  testTS,
			}},
		}}
		session := NewDirectSession(Clients{Participant: client}, SessionContext{})

		result, err := session.readParticipantList(context.Background(), "campaign://camp-1/participants")
		if err != nil {
			t.Fatalf("readParticipantList() error = %v", err)
		}

		payload := decodeToolOutput[participantListPayload](t, result)
		if len(payload.Participants) != 1 || payload.Participants[0].Role != "GM" {
			t.Fatalf("participant payload = %#v", payload)
		}
		if client.lastRequest.GetPageSize() != 10 {
			t.Fatalf("page_size = %d, want 10", client.lastRequest.GetPageSize())
		}
	})

	t.Run("characters nil response", func(t *testing.T) {
		session := NewDirectSession(Clients{Character: &characterClientStub{}}, SessionContext{})

		_, err := session.readCharacterList(context.Background(), "campaign://camp-1/characters")
		if err == nil || err.Error() != "character list response is missing" {
			t.Fatalf("readCharacterList() error = %v", err)
		}
	})

	t.Run("sessions success", func(t *testing.T) {
		client := &sessionClientStub{listResp: &statev1.ListSessionsResponse{
			Sessions: []*statev1.Session{{
				Id:         "sess-1",
				CampaignId: "camp-1",
				Name:       "Session Zero",
				Status:     statev1.SessionStatus_SESSION_ACTIVE,
				StartedAt:  testTS,
				UpdatedAt:  testTS,
				EndedAt:    testTS,
			}},
		}}
		session := NewDirectSession(Clients{Session: client}, SessionContext{})

		result, err := session.readSessionList(context.Background(), "campaign://camp-1/sessions")
		if err != nil {
			t.Fatalf("readSessionList() error = %v", err)
		}

		payload := decodeToolOutput[sessionListPayload](t, result)
		if len(payload.Sessions) != 1 || payload.Sessions[0].EndedAt == "" {
			t.Fatalf("session payload = %#v", payload)
		}
	})

	t.Run("scenes success", func(t *testing.T) {
		client := &sceneClientStub{listResp: &statev1.ListScenesResponse{
			Scenes: []*statev1.Scene{{
				SceneId:      "scene-1",
				SessionId:    "sess-1",
				Name:         "Docks",
				Description:  "Night fog.",
				Open:         true,
				CharacterIds: []string{"char-1"},
				CreatedAt:    testTS,
				UpdatedAt:    testTS,
			}},
		}}
		session := NewDirectSession(Clients{Scene: client}, SessionContext{ParticipantID: "part-1"})

		result, err := session.readSceneList(context.Background(), "campaign://camp-1/sessions/sess-1/scenes")
		if err != nil {
			t.Fatalf("readSceneList() error = %v", err)
		}

		payload := decodeToolOutput[sceneListPayload](t, result)
		if len(payload.Scenes) != 1 || payload.Scenes[0].SceneID != "scene-1" {
			t.Fatalf("scene payload = %#v", payload)
		}
		if client.lastListRequest.GetPageSize() != 20 {
			t.Fatalf("page_size = %d, want 20", client.lastListRequest.GetPageSize())
		}
		if got := client.lastMetadata.Get(grpcmeta.ParticipantIDHeader); len(got) != 1 || got[0] != "part-1" {
			t.Fatalf("participant metadata = %#v", got)
		}
	})

	t.Run("interaction success", func(t *testing.T) {
		session := NewDirectSession(Clients{
			Interaction: interactionClientStub{
				response: &statev1.GetInteractionStateResponse{
					State: &statev1.InteractionState{
						CampaignId:   "camp-1",
						CampaignName: "Broken Spire",
						ActiveScene:  &statev1.InteractionScene{SceneId: "scene-1"},
						Ooc:          &statev1.OOCState{},
					},
				},
			},
		}, SessionContext{})

		result, err := session.readInteraction(context.Background(), "campaign://camp-1/interaction")
		if err != nil {
			t.Fatalf("readInteraction() error = %v", err)
		}

		payload := decodeToolOutput[interactionStateResult](t, result)
		if payload.CampaignID != "camp-1" || payload.ActiveScene.SceneID != "scene-1" {
			t.Fatalf("interaction payload = %#v", payload)
		}
	})
}

func TestSceneMutationSuccessAndFailurePaths(t *testing.T) {
	t.Run("create success uses fallback context", func(t *testing.T) {
		client := &sceneClientStub{createResp: &statev1.CreateSceneResponse{SceneId: "scene-1"}}
		session := NewDirectSession(Clients{Scene: client}, SessionContext{
			CampaignID:    "camp-1",
			SessionID:     "sess-1",
			ParticipantID: "part-1",
		})

		result, err := session.sceneCreate(context.Background(), []byte(`{"name":"Docks","description":"Night fog.","character_ids":["char-1"]}`))
		if err != nil {
			t.Fatalf("sceneCreate() error = %v", err)
		}

		payload := decodeToolOutput[sceneCreateResult](t, result.Output)
		if payload.SceneID != "scene-1" || payload.SessionID != "sess-1" {
			t.Fatalf("scene create payload = %#v", payload)
		}
		if client.lastCreateRequest.GetCampaignId() != "camp-1" || client.lastCreateRequest.GetSessionId() != "sess-1" {
			t.Fatalf("create request = %#v", client.lastCreateRequest)
		}
		if got := client.lastMetadata.Get(grpcmeta.ParticipantIDHeader); len(got) != 1 || got[0] != "part-1" {
			t.Fatalf("participant metadata = %#v", got)
		}
	})

	t.Run("create missing response is rejected", func(t *testing.T) {
		session := NewDirectSession(Clients{Scene: &sceneClientStub{createResp: &statev1.CreateSceneResponse{}}}, SessionContext{
			CampaignID: "camp-1",
			SessionID:  "sess-1",
		})

		_, err := session.sceneCreate(context.Background(), []byte(`{"name":"Docks"}`))
		if err == nil || err.Error() != "create scene response is missing" {
			t.Fatalf("sceneCreate() error = %v", err)
		}
	})

	t.Run("transition resolves active scene and shapes response", func(t *testing.T) {
		sceneClient := &sceneClientStub{transitionResp: &statev1.TransitionSceneResponse{NewSceneId: "scene-2"}}
		session := NewDirectSession(Clients{
			Scene: sceneClient,
			Interaction: interactionClientStub{
				response: &statev1.GetInteractionStateResponse{
					State: &statev1.InteractionState{
						ActiveScene: &statev1.InteractionScene{SceneId: "scene-1"},
					},
				},
			},
		}, SessionContext{CampaignID: "camp-1"})

		result, err := session.sceneTransition(context.Background(), []byte(`{"name":"The Keep","description":"Stone halls."}`))
		if err != nil {
			t.Fatalf("sceneTransition() error = %v", err)
		}

		payload := decodeToolOutput[sceneTransitionResult](t, result.Output)
		if payload.NewSceneID != "scene-2" || payload.SourceSceneID != "scene-1" {
			t.Fatalf("scene transition payload = %#v", payload)
		}
		if sceneClient.lastTransitionReq.GetSourceSceneId() != "scene-1" {
			t.Fatalf("source scene id = %q, want scene-1", sceneClient.lastTransitionReq.GetSourceSceneId())
		}
	})

	t.Run("transition missing response id is rejected", func(t *testing.T) {
		session := NewDirectSession(Clients{
			Scene: &sceneClientStub{transitionResp: &statev1.TransitionSceneResponse{}},
			Interaction: interactionClientStub{
				response: &statev1.GetInteractionStateResponse{
					State: &statev1.InteractionState{
						ActiveScene: &statev1.InteractionScene{SceneId: "scene-9"},
					},
				},
			},
		}, SessionContext{CampaignID: "camp-1"})

		_, err := session.sceneTransition(context.Background(), []byte(`{"source_scene_id":"scene-1","name":"The Keep"}`))
		if err == nil || err.Error() != "transition scene response is missing new_scene_id" {
			t.Fatalf("sceneTransition() error = %v", err)
		}
	})
}

func TestReadResourceDispatchesCampaignAndSceneReaders(t *testing.T) {
	session := NewDirectSession(Clients{
		Campaign: &campaignClientStub{getResp: &statev1.GetCampaignResponse{
			Campaign: &statev1.Campaign{Id: "camp-1", Name: "Broken Spire", CreatedAt: testTS, UpdatedAt: testTS},
		}},
		Scene: &sceneClientStub{listResp: &statev1.ListScenesResponse{
			Scenes: []*statev1.Scene{{SceneId: "scene-1", SessionId: "sess-1", Name: "Docks", CreatedAt: testTS, UpdatedAt: testTS}},
		}},
	}, SessionContext{})

	campaignValue, err := session.readResource(context.Background(), "campaign://camp-1")
	if err != nil || !strings.Contains(campaignValue, "\"campaign\"") {
		t.Fatalf("readResource(campaign) = (%q, %v)", campaignValue, err)
	}

	sceneValue, err := session.readResource(context.Background(), "campaign://camp-1/sessions/sess-1/scenes")
	if err != nil || !strings.Contains(sceneValue, "\"scene_id\": \"scene-1\"") {
		t.Fatalf("readResource(scene list) = (%q, %v)", sceneValue, err)
	}
}
