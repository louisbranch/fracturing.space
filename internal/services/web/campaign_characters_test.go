package web

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

func TestAppCampaignCharactersPageRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/characters", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

func TestAppCampaignCharacterDetailRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/characters/char-1", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

func TestAppCampaignCharacterCreateRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/characters/create", strings.NewReader("name=Kara&kind=pc"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

func TestAppCampaignCharacterUpdateRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/characters/update", strings.NewReader("character_id=char-1&name=Kara"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

func TestAppCampaignCharacterControlRedirectsUnauthenticatedToLogin(t *testing.T) {
	handler := NewHandler(Config{AuthBaseURL: "http://auth.local"}, nil)
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/characters/control", strings.NewReader("character_id=char-1&participant_id=part-1"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/auth/login" {
		t.Fatalf("location = %q, want %q", location, "/auth/login")
	}
}

type fakeWebCharacterClient struct {
	response   *statev1.ListCharactersResponse
	lastReq    *statev1.ListCharactersRequest
	listMD     metadata.MD
	listCalls  int
	sheetReq   *statev1.GetCharacterSheetRequest
	sheetMD    metadata.MD
	sheetResp  *statev1.GetCharacterSheetResponse
	createReq  *statev1.CreateCharacterRequest
	createMD   metadata.MD
	createResp *statev1.CreateCharacterResponse
	updateReq  *statev1.UpdateCharacterRequest
	updateMD   metadata.MD
	updateResp *statev1.UpdateCharacterResponse
	controlReq *statev1.SetDefaultControlRequest
	controlMD  metadata.MD
	controlRsp *statev1.SetDefaultControlResponse
}

func (f *fakeWebCharacterClient) CreateCharacter(ctx context.Context, req *statev1.CreateCharacterRequest, _ ...grpc.CallOption) (*statev1.CreateCharacterResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	f.createMD = md
	f.createReq = req
	if f.createResp != nil {
		return f.createResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeWebCharacterClient) UpdateCharacter(ctx context.Context, req *statev1.UpdateCharacterRequest, _ ...grpc.CallOption) (*statev1.UpdateCharacterResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	f.updateMD = md
	f.updateReq = req
	if f.updateResp != nil {
		return f.updateResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeWebCharacterClient) DeleteCharacter(context.Context, *statev1.DeleteCharacterRequest, ...grpc.CallOption) (*statev1.DeleteCharacterResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeWebCharacterClient) ListCharacters(ctx context.Context, req *statev1.ListCharactersRequest, _ ...grpc.CallOption) (*statev1.ListCharactersResponse, error) {
	f.listCalls++
	md, _ := metadata.FromOutgoingContext(ctx)
	f.listMD = md
	f.lastReq = req
	if f.response != nil {
		return f.response, nil
	}
	return &statev1.ListCharactersResponse{}, nil
}

func TestAppCampaignCharactersPageCachesCharactersAndControlParticipants(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{Active: true, UserID: "Alice"})
	}))
	t.Cleanup(authServer.Close)

	cacheStore := newFakeWebCacheStore()
	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-manager",
						CampaignId:     "camp-123",
						UserId:         "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
					},
					{
						Id:             "part-player",
						CampaignId:     "camp-123",
						UserId:         "Bob",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER,
					},
				},
			},
		},
	}
	characterClient := &fakeWebCharacterClient{
		response: &statev1.ListCharactersResponse{
			Characters: []*statev1.Character{
				{Id: "char-1", CampaignId: "camp-123", Name: "Mira"},
			},
		},
	}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		participantClient: participantClient,
		characterClient:   characterClient,
		cacheStore:        cacheStore,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))

	req1 := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/characters", nil)
	req1.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w1 := httptest.NewRecorder()
	h.handleAppCampaignDetail(w1, req1)
	if w1.Code != http.StatusOK {
		t.Fatalf("first status = %d, want %d", w1.Code, http.StatusOK)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/characters", nil)
	req2.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w2 := httptest.NewRecorder()
	h.handleAppCampaignDetail(w2, req2)
	if w2.Code != http.StatusOK {
		t.Fatalf("second status = %d, want %d", w2.Code, http.StatusOK)
	}

	if characterClient.listCalls != 1 {
		t.Fatalf("list characters calls = %d, want %d", characterClient.listCalls, 1)
	}
	if len(participantClient.calls) != 3 {
		t.Fatalf("list participants calls = %d, want %d", len(participantClient.calls), 3)
	}
	if cacheStore.putCalls == 0 {
		t.Fatalf("expected cache store put calls")
	}
}

func (f *fakeWebCharacterClient) SetDefaultControl(ctx context.Context, req *statev1.SetDefaultControlRequest, _ ...grpc.CallOption) (*statev1.SetDefaultControlResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	f.controlMD = md
	f.controlReq = req
	if f.controlRsp != nil {
		return f.controlRsp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeWebCharacterClient) GetCharacterSheet(ctx context.Context, req *statev1.GetCharacterSheetRequest, _ ...grpc.CallOption) (*statev1.GetCharacterSheetResponse, error) {
	md, _ := metadata.FromOutgoingContext(ctx)
	f.sheetMD = md
	f.sheetReq = req
	if f.sheetResp != nil {
		return f.sheetResp, nil
	}
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func (f *fakeWebCharacterClient) PatchCharacterProfile(context.Context, *statev1.PatchCharacterProfileRequest, ...grpc.CallOption) (*statev1.PatchCharacterProfileResponse, error) {
	return nil, status.Error(codes.Unimplemented, "not implemented")
}

func TestAppCampaignCharactersPageParticipantRendersCharacters(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{Active: true, UserID: "Alice"})
	}))
	t.Cleanup(authServer.Close)
	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-manager",
						CampaignId:     "camp-123",
						UserId:         "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
					},
				},
			},
		},
	}
	characterClient := &fakeWebCharacterClient{
		response: &statev1.ListCharactersResponse{
			Characters: []*statev1.Character{
				{Id: "char-1", CampaignId: "camp-123", Name: "Mira"},
				{Id: "char-2", CampaignId: "camp-123", Name: "Orin"},
			},
		},
	}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		participantClient: participantClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
		characterClient: characterClient,
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/characters", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if characterClient.lastReq == nil {
		t.Fatalf("expected ListCharacters request to be captured")
	}
	if characterClient.lastReq.GetCampaignId() != "camp-123" {
		t.Fatalf("campaign_id = %q, want %q", characterClient.lastReq.GetCampaignId(), "camp-123")
	}
	body := w.Body.String()
	if !strings.Contains(body, "Mira") {
		t.Fatalf("expected Mira in response body")
	}
	if !strings.Contains(body, "Orin") {
		t.Fatalf("expected Orin in response body")
	}
	if !strings.Contains(body, "/campaigns/camp-123/characters/char-1") {
		t.Fatalf("expected character detail link for char-1")
	}
}

func TestAppCampaignCharactersPagePropagatesUserMetadataToCharacterRead(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{Active: true, UserID: "Alice"})
	}))
	t.Cleanup(authServer.Close)
	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-manager",
						CampaignId:     "camp-123",
						UserId:         "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
					},
				},
			},
		},
	}
	characterClient := &fakeWebCharacterClient{
		response: &statev1.ListCharactersResponse{
			Characters: []*statev1.Character{
				{Id: "char-1", CampaignId: "camp-123", Name: "Mira"},
			},
		},
	}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		participantClient: participantClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
		characterClient: characterClient,
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/characters", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	userIDs := characterClient.listMD.Get(grpcmeta.UserIDHeader)
	if len(userIDs) != 1 || userIDs[0] != "Alice" {
		t.Fatalf("metadata %s = %v, want [Alice]", grpcmeta.UserIDHeader, userIDs)
	}
}

func TestAppCampaignCharacterDetailParticipantRendersCharacter(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{Active: true, UserID: "Alice"})
	}))
	t.Cleanup(authServer.Close)
	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-manager",
						CampaignId:     "camp-123",
						UserId:         "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
					},
				},
			},
		},
	}
	characterClient := &fakeWebCharacterClient{
		sheetResp: &statev1.GetCharacterSheetResponse{
			Character: &statev1.Character{
				Id:         "char-1",
				CampaignId: "camp-123",
				Name:       "Mira",
				Kind:       statev1.CharacterKind_PC,
			},
		},
	}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		participantClient: participantClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
		characterClient: characterClient,
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/characters/char-1", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if characterClient.sheetReq == nil {
		t.Fatalf("expected GetCharacterSheet request to be captured")
	}
	if characterClient.sheetReq.GetCampaignId() != "camp-123" {
		t.Fatalf("campaign_id = %q, want %q", characterClient.sheetReq.GetCampaignId(), "camp-123")
	}
	if characterClient.sheetReq.GetCharacterId() != "char-1" {
		t.Fatalf("character_id = %q, want %q", characterClient.sheetReq.GetCharacterId(), "char-1")
	}
	body := w.Body.String()
	if !strings.Contains(body, "Mira") {
		t.Fatalf("expected character name in response body")
	}
}

func TestAppCampaignCharacterDetailPropagatesUserMetadataToCharacterRead(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{Active: true, UserID: "Alice"})
	}))
	t.Cleanup(authServer.Close)
	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-manager",
						CampaignId:     "camp-123",
						UserId:         "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
					},
				},
			},
		},
	}
	characterClient := &fakeWebCharacterClient{
		sheetResp: &statev1.GetCharacterSheetResponse{
			Character: &statev1.Character{
				Id:         "char-1",
				CampaignId: "camp-123",
				Name:       "Mira",
				Kind:       statev1.CharacterKind_PC,
			},
		},
	}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		participantClient: participantClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
		characterClient: characterClient,
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/characters/char-1", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	userIDs := characterClient.sheetMD.Get(grpcmeta.UserIDHeader)
	if len(userIDs) != 1 || userIDs[0] != "Alice" {
		t.Fatalf("metadata %s = %v, want [Alice]", grpcmeta.UserIDHeader, userIDs)
	}
}

func TestAppCampaignCharacterCreateManagerCallsCreateCharacter(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{
			Active: true,
			UserID: "user-123",
		})
	}))
	t.Cleanup(authServer.Close)

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-manager",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						Name:           "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
					},
				},
			},
		},
	}
	characterClient := &fakeWebCharacterClient{
		createResp: &statev1.CreateCharacterResponse{
			Character: &statev1.Character{
				Id:         "char-3",
				CampaignId: "camp-123",
				Name:       "Kara",
				Kind:       statev1.CharacterKind_PC,
			},
		},
	}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		characterClient:   characterClient,
		participantClient: participantClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{
		"name": {"Kara"},
		"kind": {"pc"},
	}
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/characters/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/campaigns/camp-123/characters" {
		t.Fatalf("location = %q, want %q", location, "/campaigns/camp-123/characters")
	}
	if characterClient.createReq == nil {
		t.Fatalf("expected CreateCharacter request to be captured")
	}
	if characterClient.createReq.GetCampaignId() != "camp-123" {
		t.Fatalf("campaign_id = %q, want %q", characterClient.createReq.GetCampaignId(), "camp-123")
	}
	if characterClient.createReq.GetName() != "Kara" {
		t.Fatalf("name = %q, want %q", characterClient.createReq.GetName(), "Kara")
	}
	if characterClient.createReq.GetKind() != statev1.CharacterKind_PC {
		t.Fatalf("kind = %v, want %v", characterClient.createReq.GetKind(), statev1.CharacterKind_PC)
	}
	participantIDs := characterClient.createMD.Get(grpcmeta.ParticipantIDHeader)
	if len(participantIDs) != 1 || participantIDs[0] != "part-manager" {
		t.Fatalf("metadata %s = %v, want [part-manager]", grpcmeta.ParticipantIDHeader, participantIDs)
	}
}

func TestAppCampaignCharacterUpdateManagerCallsUpdateCharacter(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{
			Active: true,
			UserID: "user-123",
		})
	}))
	t.Cleanup(authServer.Close)

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-manager",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						Name:           "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
					},
				},
			},
		},
	}
	characterClient := &fakeWebCharacterClient{
		updateResp: &statev1.UpdateCharacterResponse{
			Character: &statev1.Character{
				Id:         "char-3",
				CampaignId: "camp-123",
				Name:       "Kara Prime",
				Kind:       statev1.CharacterKind_NPC,
			},
		},
	}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		characterClient:   characterClient,
		participantClient: participantClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{
		"character_id": {"char-3"},
		"name":         {"Kara Prime"},
		"kind":         {"npc"},
	}
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/characters/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/campaigns/camp-123/characters" {
		t.Fatalf("location = %q, want %q", location, "/campaigns/camp-123/characters")
	}
	if characterClient.updateReq == nil {
		t.Fatalf("expected UpdateCharacter request to be captured")
	}
	if characterClient.updateReq.GetCampaignId() != "camp-123" {
		t.Fatalf("campaign_id = %q, want %q", characterClient.updateReq.GetCampaignId(), "camp-123")
	}
	if characterClient.updateReq.GetCharacterId() != "char-3" {
		t.Fatalf("character_id = %q, want %q", characterClient.updateReq.GetCharacterId(), "char-3")
	}
	if characterClient.updateReq.GetName() == nil || characterClient.updateReq.GetName().GetValue() != "Kara Prime" {
		t.Fatalf("name = %v, want %q", characterClient.updateReq.GetName(), "Kara Prime")
	}
	if characterClient.updateReq.GetKind() != statev1.CharacterKind_NPC {
		t.Fatalf("kind = %v, want %v", characterClient.updateReq.GetKind(), statev1.CharacterKind_NPC)
	}
	participantIDs := characterClient.updateMD.Get(grpcmeta.ParticipantIDHeader)
	if len(participantIDs) != 1 || participantIDs[0] != "part-manager" {
		t.Fatalf("metadata %s = %v, want [part-manager]", grpcmeta.ParticipantIDHeader, participantIDs)
	}
}

func TestAppCampaignCharacterControlManagerCallsSetDefaultControl(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{
			Active: true,
			UserID: "user-123",
		})
	}))
	t.Cleanup(authServer.Close)

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-manager",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						Name:           "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
					},
				},
			},
		},
	}
	characterClient := &fakeWebCharacterClient{
		controlRsp: &statev1.SetDefaultControlResponse{
			CampaignId:  "camp-123",
			CharacterId: "char-3",
		},
	}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		characterClient:   characterClient,
		participantClient: participantClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{
		"character_id":   {"char-3"},
		"participant_id": {"part-member"},
	}
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/characters/control", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusFound)
	}
	if location := w.Header().Get("Location"); location != "/campaigns/camp-123/characters" {
		t.Fatalf("location = %q, want %q", location, "/campaigns/camp-123/characters")
	}
	if characterClient.controlReq == nil {
		t.Fatalf("expected SetDefaultControl request to be captured")
	}
	if characterClient.controlReq.GetCampaignId() != "camp-123" {
		t.Fatalf("campaign_id = %q, want %q", characterClient.controlReq.GetCampaignId(), "camp-123")
	}
	if characterClient.controlReq.GetCharacterId() != "char-3" {
		t.Fatalf("character_id = %q, want %q", characterClient.controlReq.GetCharacterId(), "char-3")
	}
	if characterClient.controlReq.GetParticipantId() == nil || characterClient.controlReq.GetParticipantId().GetValue() != "part-member" {
		t.Fatalf("participant_id = %v, want %q", characterClient.controlReq.GetParticipantId(), "part-member")
	}
	participantIDs := characterClient.controlMD.Get(grpcmeta.ParticipantIDHeader)
	if len(participantIDs) != 1 || participantIDs[0] != "part-manager" {
		t.Fatalf("metadata %s = %v, want [part-manager]", grpcmeta.ParticipantIDHeader, participantIDs)
	}
}

func TestAppCampaignCharacterCreateMemberForbidden(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{
			Active: true,
			UserID: "user-123",
		})
	}))
	t.Cleanup(authServer.Close)

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-member",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						Name:           "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER,
					},
				},
			},
		},
	}
	characterClient := &fakeWebCharacterClient{}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		characterClient:   characterClient,
		participantClient: participantClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{
		"name": {"Kara"},
		"kind": {"pc"},
	}
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/characters/create", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if characterClient.createReq != nil {
		t.Fatalf("expected CreateCharacter not to be called for member access")
	}
}

func TestAppCampaignCharacterUpdateMemberForbidden(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{
			Active: true,
			UserID: "user-123",
		})
	}))
	t.Cleanup(authServer.Close)

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-member",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						Name:           "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER,
					},
				},
			},
		},
	}
	characterClient := &fakeWebCharacterClient{}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		characterClient:   characterClient,
		participantClient: participantClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{
		"character_id": {"char-3"},
		"name":         {"Kara Prime"},
		"kind":         {"npc"},
	}
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/characters/update", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if characterClient.updateReq != nil {
		t.Fatalf("expected UpdateCharacter not to be called for member access")
	}
}

func TestAppCampaignCharacterControlMemberForbidden(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{
			Active: true,
			UserID: "user-123",
		})
	}))
	t.Cleanup(authServer.Close)

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-member",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						Name:           "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER,
					},
				},
			},
		},
	}
	characterClient := &fakeWebCharacterClient{}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		characterClient:   characterClient,
		participantClient: participantClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	form := url.Values{
		"character_id":   {"char-3"},
		"participant_id": {"part-member"},
	}
	req := httptest.NewRequest(http.MethodPost, "/campaigns/camp-123/characters/control", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
	if characterClient.controlReq != nil {
		t.Fatalf("expected SetDefaultControl not to be called for member access")
	}
}

func TestAppCampaignCharactersPageManagerShowsCreateControls(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{
			Active: true,
			UserID: "user-123",
		})
	}))
	t.Cleanup(authServer.Close)

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-manager",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						Name:           "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MANAGER,
					},
				},
			},
		},
	}
	characterClient := &fakeWebCharacterClient{
		response: &statev1.ListCharactersResponse{
			Characters: []*statev1.Character{
				{Id: "char-1", CampaignId: "camp-123", Name: "Mira"},
			},
		},
	}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		characterClient:   characterClient,
		participantClient: participantClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/characters", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if !strings.Contains(body, "Create Character") {
		t.Fatalf("expected create character control in response body")
	}
	if !strings.Contains(body, "Update Character") {
		t.Fatalf("expected update character control in response body")
	}
	if !strings.Contains(body, "Set Controller") {
		t.Fatalf("expected character controller control in response body")
	}
}

func TestAppCampaignCharactersPageMemberHidesCreateControls(t *testing.T) {
	authServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/introspect" {
			t.Fatalf("path = %q, want %q", r.URL.Path, "/introspect")
		}
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(introspectResponse{
			Active: true,
			UserID: "user-123",
		})
	}))
	t.Cleanup(authServer.Close)

	participantClient := &fakeWebParticipantClient{
		pages: map[string]*statev1.ListParticipantsResponse{
			"": {
				Participants: []*statev1.Participant{
					{
						Id:             "part-member",
						CampaignId:     "camp-123",
						UserId:         "user-123",
						Name:           "Alice",
						CampaignAccess: statev1.CampaignAccess_CAMPAIGN_ACCESS_MEMBER,
					},
				},
			},
		},
	}
	characterClient := &fakeWebCharacterClient{
		response: &statev1.ListCharactersResponse{
			Characters: []*statev1.Character{
				{Id: "char-1", CampaignId: "camp-123", Name: "Mira"},
			},
		},
	}
	h := &handler{
		config: Config{
			AuthBaseURL:         authServer.URL,
			OAuthResourceSecret: "secret-1",
		},
		sessions:          newSessionStore(),
		pendingFlows:      newPendingFlowStore(),
		characterClient:   characterClient,
		participantClient: participantClient,
		campaignAccess: &campaignAccessService{
			authBaseURL:         authServer.URL,
			oauthResourceSecret: "secret-1",
			httpClient:          authServer.Client(),
			participantClient:   participantClient,
		},
	}
	sessionID := h.sessions.create("token-1", "Alice", time.Now().Add(time.Hour))
	req := httptest.NewRequest(http.MethodGet, "/campaigns/camp-123/characters", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: sessionID})
	w := httptest.NewRecorder()

	h.handleAppCampaignDetail(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := w.Body.String()
	if strings.Contains(body, "Create Character") {
		t.Fatalf("did not expect create character control in response body")
	}
	if strings.Contains(body, "Update Character") {
		t.Fatalf("did not expect update character control in response body")
	}
	if strings.Contains(body, "Set Controller") {
		t.Fatalf("did not expect character controller control in response body")
	}
}
