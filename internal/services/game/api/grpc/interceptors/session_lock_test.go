package interceptors

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/test/grpcassert"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

// fakeSessionStore is a test double for storage.SessionStore.
type fakeSessionStore struct {
	activeSession map[string]storage.SessionRecord // campaignID -> Session
	activeErr     error
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{
		activeSession: make(map[string]storage.SessionRecord),
	}
}

func (s *fakeSessionStore) PutSession(_ context.Context, sess storage.SessionRecord) error {
	return nil
}

func (s *fakeSessionStore) EndSession(_ context.Context, campaignID, sessionID string, endedAt time.Time) (storage.SessionRecord, bool, error) {
	return storage.SessionRecord{}, false, nil
}

func (s *fakeSessionStore) GetSession(_ context.Context, campaignID, sessionID string) (storage.SessionRecord, error) {
	return storage.SessionRecord{}, nil
}

func (s *fakeSessionStore) GetActiveSession(_ context.Context, campaignID string) (storage.SessionRecord, error) {
	if s.activeErr != nil {
		return storage.SessionRecord{}, s.activeErr
	}
	sess, ok := s.activeSession[campaignID]
	if !ok {
		return storage.SessionRecord{}, storage.ErrNotFound
	}
	return sess, nil
}

func (s *fakeSessionStore) CountSessions(_ context.Context, _ string) (int, error) {
	return 0, nil
}

func (s *fakeSessionStore) ListSessions(_ context.Context, campaignID string, pageSize int, pageToken string) (storage.SessionPage, error) {
	return storage.SessionPage{}, nil
}

// fakeHandler is a test double for grpc.UnaryHandler.
func fakeHandler(ctx context.Context, req any) (any, error) {
	return "success", nil
}

// Test helper to create UnaryServerInfo
func serverInfo(fullMethod string) *grpc.UnaryServerInfo {
	return &grpc.UnaryServerInfo{FullMethod: fullMethod}
}

func TestSessionLockInterceptor_NonBlockedMethod_PassesThrough(t *testing.T) {
	sessionStore := newFakeSessionStore()
	interceptor := SessionLockInterceptor(sessionStore)

	// ListCampaigns is not a blocked method
	info := serverInfo("/game.v1.CampaignService/ListCampaigns")
	req := &statev1.ListCampaignsRequest{}

	resp, err := interceptor(context.Background(), req, info, fakeHandler)
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
	if resp != "success" {
		t.Errorf("response = %v, want %q", resp, "success")
	}
}

func TestSessionLockInterceptor_BlockedMethod_NoActiveSession_PassesThrough(t *testing.T) {
	sessionStore := newFakeSessionStore()
	interceptor := SessionLockInterceptor(sessionStore)

	// CreateParticipant is blocked, but no active session exists
	info := serverInfo(statev1.ParticipantService_CreateParticipant_FullMethodName)
	req := &statev1.CreateParticipantRequest{CampaignId: "c1", Name: "Test"}

	resp, err := interceptor(context.Background(), req, info, fakeHandler)
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
	if resp != "success" {
		t.Errorf("response = %v, want %q", resp, "success")
	}
}

func TestSessionLockInterceptor_CreateParticipant_WithActiveSession_Blocks(t *testing.T) {
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	sessionStore.activeSession["c1"] = storage.SessionRecord{ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now}

	interceptor := SessionLockInterceptor(sessionStore)
	info := serverInfo(statev1.ParticipantService_CreateParticipant_FullMethodName)
	req := &statev1.CreateParticipantRequest{CampaignId: "c1", Name: "Test"}

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	grpcassert.StatusCode(t, err, codes.FailedPrecondition)
}

func TestSessionLockInterceptor_UpdateCampaign_WithActiveSession_Blocks(t *testing.T) {
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	sessionStore.activeSession["c1"] = storage.SessionRecord{ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now}

	interceptor := SessionLockInterceptor(sessionStore)
	info := serverInfo(statev1.CampaignService_UpdateCampaign_FullMethodName)
	req := &statev1.UpdateCampaignRequest{CampaignId: "c1"}

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	grpcassert.StatusCode(t, err, codes.FailedPrecondition)
}

func TestSessionLockInterceptor_CreateCharacter_WithActiveSession_Blocks(t *testing.T) {
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	sessionStore.activeSession["c1"] = storage.SessionRecord{ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now}

	interceptor := SessionLockInterceptor(sessionStore)
	info := serverInfo(statev1.CharacterService_CreateCharacter_FullMethodName)
	req := &statev1.CreateCharacterRequest{CampaignId: "c1", Name: "Hero"}

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	grpcassert.StatusCode(t, err, codes.FailedPrecondition)
}

func TestSessionLockInterceptor_SetDefaultControl_WithActiveSession_Blocks(t *testing.T) {
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	sessionStore.activeSession["c1"] = storage.SessionRecord{ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now}

	interceptor := SessionLockInterceptor(sessionStore)
	info := serverInfo(statev1.CharacterService_SetDefaultControl_FullMethodName)
	req := &statev1.SetDefaultControlRequest{CampaignId: "c1", CharacterId: "ch1"}

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	grpcassert.StatusCode(t, err, codes.FailedPrecondition)
}

func TestSessionLockInterceptor_ClaimCharacterControl_WithActiveSession_Blocks(t *testing.T) {
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	sessionStore.activeSession["c1"] = storage.SessionRecord{ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now}

	interceptor := SessionLockInterceptor(sessionStore)
	info := serverInfo(statev1.CharacterService_ClaimCharacterControl_FullMethodName)
	req := &statev1.ClaimCharacterControlRequest{CampaignId: "c1", CharacterId: "ch1"}

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	grpcassert.StatusCode(t, err, codes.FailedPrecondition)
}

func TestSessionLockInterceptor_ReleaseCharacterControl_WithActiveSession_Blocks(t *testing.T) {
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	sessionStore.activeSession["c1"] = storage.SessionRecord{ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now}

	interceptor := SessionLockInterceptor(sessionStore)
	info := serverInfo(statev1.CharacterService_ReleaseCharacterControl_FullMethodName)
	req := &statev1.ReleaseCharacterControlRequest{CampaignId: "c1", CharacterId: "ch1"}

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	grpcassert.StatusCode(t, err, codes.FailedPrecondition)
}

func TestSessionLockInterceptor_PatchCharacterProfile_WithActiveSession_Blocks(t *testing.T) {
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	sessionStore.activeSession["c1"] = storage.SessionRecord{ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now}

	interceptor := SessionLockInterceptor(sessionStore)
	info := serverInfo(statev1.CharacterService_PatchCharacterProfile_FullMethodName)
	req := &statev1.PatchCharacterProfileRequest{CampaignId: "c1", CharacterId: "ch1"}

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	grpcassert.StatusCode(t, err, codes.FailedPrecondition)
}

func TestSessionLockInterceptor_PatchCharacterState_WithActiveSession_NotBlocked(t *testing.T) {
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	sessionStore.activeSession["c1"] = storage.SessionRecord{ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now}

	interceptor := SessionLockInterceptor(sessionStore)
	info := serverInfo(statev1.SnapshotService_PatchCharacterState_FullMethodName)
	req := &statev1.PatchCharacterStateRequest{CampaignId: "c1", CharacterId: "ch1"}

	resp, err := interceptor(context.Background(), req, info, fakeHandler)
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
	if resp != "success" {
		t.Errorf("response = %v, want %q", resp, "success")
	}
}

func TestSessionLockInterceptor_ForkCampaign_WithActiveSession_BlocksUsingSourceCampaignID(t *testing.T) {
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	sessionStore.activeSession["source-c1"] = storage.SessionRecord{
		ID:         "s1",
		CampaignID: "source-c1",
		Status:     session.StatusActive,
		StartedAt:  now,
	}

	interceptor := SessionLockInterceptor(sessionStore)
	info := serverInfo(statev1.ForkService_ForkCampaign_FullMethodName)
	req := &statev1.ForkCampaignRequest{SourceCampaignId: "source-c1"}

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	grpcassert.StatusCode(t, err, codes.FailedPrecondition)
}

func TestSessionLockInterceptor_BlockedMethod_MissingCampaignId_ReturnsInvalidArgument(t *testing.T) {
	sessionStore := newFakeSessionStore()
	interceptor := SessionLockInterceptor(sessionStore)

	info := serverInfo(statev1.ParticipantService_CreateParticipant_FullMethodName)
	req := &statev1.CreateParticipantRequest{Name: "Test"} // No CampaignId

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
	grpcassert.StatusMessage(t, err, "campaign_id is required")
}

func TestSessionLockInterceptor_ForkCampaign_MissingSourceCampaignID_ReturnsInvalidArgument(t *testing.T) {
	sessionStore := newFakeSessionStore()
	interceptor := SessionLockInterceptor(sessionStore)

	info := serverInfo(statev1.ForkService_ForkCampaign_FullMethodName)
	req := &statev1.ForkCampaignRequest{} // No SourceCampaignId

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	grpcassert.StatusCode(t, err, codes.InvalidArgument)
	grpcassert.StatusMessage(t, err, "source_campaign_id is required")
}

func TestSessionLockInterceptor_BlockedMethod_NilSessionStore_ReturnsInternal(t *testing.T) {
	interceptor := SessionLockInterceptor(nil)

	info := serverInfo(statev1.ParticipantService_CreateParticipant_FullMethodName)
	req := &statev1.CreateParticipantRequest{CampaignId: "c1", Name: "Test"}

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestSessionLockInterceptor_BlockedMethod_SessionStoreError_ReturnsInternal(t *testing.T) {
	sessionStore := newFakeSessionStore()
	sessionStore.activeErr = errors.New("database error")

	interceptor := SessionLockInterceptor(sessionStore)
	info := serverInfo(statev1.ParticipantService_CreateParticipant_FullMethodName)
	req := &statev1.CreateParticipantRequest{CampaignId: "c1", Name: "Test"}

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	grpcassert.StatusCode(t, err, codes.Internal)
}

func TestSessionLockInterceptor_DifferentCampaigns_NotBlocked(t *testing.T) {
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	// Active session for campaign c1
	sessionStore.activeSession["c1"] = storage.SessionRecord{ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now}

	interceptor := SessionLockInterceptor(sessionStore)
	info := serverInfo(statev1.ParticipantService_CreateParticipant_FullMethodName)
	// Request for campaign c2 (no active session)
	req := &statev1.CreateParticipantRequest{CampaignId: "c2", Name: "Test"}

	resp, err := interceptor(context.Background(), req, info, fakeHandler)
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
	if resp != "success" {
		t.Errorf("response = %v, want %q", resp, "success")
	}
}

func TestSessionLockInterceptor_GetCharacterSheet_NotBlocked(t *testing.T) {
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	sessionStore.activeSession["c1"] = storage.SessionRecord{ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now}

	interceptor := SessionLockInterceptor(sessionStore)
	// GetCharacterSheet is a read method, not blocked
	info := serverInfo(statev1.CharacterService_GetCharacterSheet_FullMethodName)
	req := &statev1.GetCharacterSheetRequest{CampaignId: "c1", CharacterId: "ch1"}

	resp, err := interceptor(context.Background(), req, info, fakeHandler)
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
	if resp != "success" {
		t.Errorf("response = %v, want %q", resp, "success")
	}
}

func TestSessionLockInterceptor_ListCharacters_NotBlocked(t *testing.T) {
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	sessionStore.activeSession["c1"] = storage.SessionRecord{ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now}

	interceptor := SessionLockInterceptor(sessionStore)
	// ListCharacters is a read method, not blocked
	info := serverInfo(statev1.CharacterService_ListCharacters_FullMethodName)
	req := &statev1.ListCharactersRequest{CampaignId: "c1"}

	resp, err := interceptor(context.Background(), req, info, fakeHandler)
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
	if resp != "success" {
		t.Errorf("response = %v, want %q", resp, "success")
	}
}

func TestSessionLockInterceptor_GetSnapshot_NotBlocked(t *testing.T) {
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	sessionStore.activeSession["c1"] = storage.SessionRecord{ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now}

	interceptor := SessionLockInterceptor(sessionStore)
	// GetSnapshot is a read method, not blocked
	info := serverInfo(statev1.SnapshotService_GetSnapshot_FullMethodName)
	req := &statev1.GetSnapshotRequest{CampaignId: "c1"}

	resp, err := interceptor(context.Background(), req, info, fakeHandler)
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
	if resp != "success" {
		t.Errorf("response = %v, want %q", resp, "success")
	}
}

func TestSessionLockInterceptor_UpdateSnapshotState_NotBlocked(t *testing.T) {
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	sessionStore.activeSession["c1"] = storage.SessionRecord{ID: "s1", CampaignID: "c1", Status: session.StatusActive, StartedAt: now}

	interceptor := SessionLockInterceptor(sessionStore)
	// UpdateSnapshotState is a gameplay action, allowed during active session
	info := serverInfo(statev1.SnapshotService_UpdateSnapshotState_FullMethodName)
	req := &statev1.UpdateSnapshotStateRequest{CampaignId: "c1"}

	resp, err := interceptor(context.Background(), req, info, fakeHandler)
	if err != nil {
		t.Fatalf("interceptor returned error: %v", err)
	}
	if resp != "success" {
		t.Errorf("response = %v, want %q", resp, "success")
	}
}

func TestSessionLockStreamInterceptor_NonBlockedMethod_PassesThrough(t *testing.T) {
	interceptor := SessionLockStreamInterceptor()
	stream := &fakeServerStream{ctx: context.Background()}
	info := &grpc.StreamServerInfo{FullMethod: statev1.EventService_SubscribeCampaignUpdates_FullMethodName}
	called := false

	err := interceptor(nil, stream, info, func(srv any, stream grpc.ServerStream) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !called {
		t.Fatal("expected stream handler to be called")
	}
}

func TestSessionLockStreamInterceptor_BlockedMethod_FailsClosed(t *testing.T) {
	interceptor := SessionLockStreamInterceptor()
	stream := &fakeServerStream{ctx: context.Background()}
	info := &grpc.StreamServerInfo{FullMethod: statev1.ParticipantService_CreateParticipant_FullMethodName}

	err := interceptor(nil, stream, info, func(srv any, stream grpc.ServerStream) error {
		t.Fatal("blocked streaming mutator should not reach handler")
		return nil
	})
	grpcassert.StatusCode(t, err, codes.FailedPrecondition)
	grpcassert.StatusMessage(t, err, "streaming mutator /game.v1.ParticipantService/CreateParticipant is not supported by session lock enforcement")
}

func TestIsBlockedMethod(t *testing.T) {
	tests := []struct {
		method  string
		blocked bool
	}{
		// Blocked methods
		{statev1.CampaignService_UpdateCampaign_FullMethodName, true},
		{statev1.CampaignService_SetCampaignCover_FullMethodName, true},
		{statev1.CampaignService_SetCampaignAIBinding_FullMethodName, true},
		{statev1.CampaignService_ClearCampaignAIBinding_FullMethodName, true},
		{statev1.ParticipantService_CreateParticipant_FullMethodName, true},
		{statev1.ParticipantService_UpdateParticipant_FullMethodName, true},
		{statev1.ParticipantService_DeleteParticipant_FullMethodName, true},
		// Invite methods are now handled by the invite service, not the game service.
		{invitev1.InviteService_CreateInvite_FullMethodName, false},
		{invitev1.InviteService_ClaimInvite_FullMethodName, false},
		{statev1.CharacterService_CreateCharacter_FullMethodName, true},
		{statev1.CharacterService_UpdateCharacter_FullMethodName, true},
		{statev1.CharacterService_DeleteCharacter_FullMethodName, true},
		{statev1.CharacterService_SetDefaultControl_FullMethodName, true},
		{statev1.CharacterService_ClaimCharacterControl_FullMethodName, true},
		{statev1.CharacterService_ReleaseCharacterControl_FullMethodName, true},
		{statev1.CharacterService_PatchCharacterProfile_FullMethodName, true},
		{statev1.CharacterService_ApplyCharacterCreationStep_FullMethodName, true},
		{statev1.CharacterService_ApplyCharacterCreationWorkflow_FullMethodName, true},
		{statev1.CharacterService_ResetCharacterCreationWorkflow_FullMethodName, true},
		{statev1.ForkService_ForkCampaign_FullMethodName, true},

		// Not blocked methods
		{statev1.ParticipantService_ListParticipants_FullMethodName, false},
		{statev1.ParticipantService_GetParticipant_FullMethodName, false},
		{invitev1.InviteService_RevokeInvite_FullMethodName, false},
		{statev1.CharacterService_ListCharacters_FullMethodName, false},
		{statev1.CharacterService_GetCharacterSheet_FullMethodName, false},
		{statev1.SnapshotService_GetSnapshot_FullMethodName, false},
		{statev1.SnapshotService_PatchCharacterState_FullMethodName, false},
		{statev1.SnapshotService_UpdateSnapshotState_FullMethodName, false},
		{"/game.v1.CampaignService/CreateCampaign", false},
		{"/game.v1.SessionService/StartSession", false},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			got := isBlockedMethod(tt.method)
			if got != tt.blocked {
				t.Errorf("isBlockedMethod(%q) = %v, want %v", tt.method, got, tt.blocked)
			}
		})
	}
}

func TestCampaignIDFromRequest(t *testing.T) {
	tests := []struct {
		name   string
		req    any
		wantID string
	}{
		{
			name:   "CreateParticipantRequest",
			req:    &statev1.CreateParticipantRequest{CampaignId: "c1"},
			wantID: "c1",
		},
		{
			name:   "CreateCharacterRequest",
			req:    &statev1.CreateCharacterRequest{CampaignId: "c2"},
			wantID: "c2",
		},
		{
			name:   "PatchCharacterStateRequest",
			req:    &statev1.PatchCharacterStateRequest{CampaignId: "c3"},
			wantID: "c3",
		},
		{
			name:   "ForkCampaignRequest",
			req:    &statev1.ForkCampaignRequest{SourceCampaignId: "source-c1"},
			wantID: "source-c1",
		},
		{
			name:   "WhitespaceOnly",
			req:    &statev1.CreateParticipantRequest{CampaignId: "   "},
			wantID: "",
		},
		{
			name:   "WithWhitespace",
			req:    &statev1.CreateParticipantRequest{CampaignId: "  c1  "},
			wantID: "c1",
		},
		{
			name:   "NonGetterType",
			req:    "not a request",
			wantID: "",
		},
		{
			name:   "NilRequest",
			req:    nil,
			wantID: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := campaignIDFromRequest(tt.req)
			if got != tt.wantID {
				t.Errorf("campaignIDFromRequest() = %q, want %q", got, tt.wantID)
			}
		})
	}
}

func TestValidateSessionLockPolicyCoverage_PassesWithCurrentConfig(t *testing.T) {
	registries, err := engine.BuildRegistries(daggerheart.NewModule())
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}
	if err := ValidateSessionLockPolicyCoverage(registries.Commands); err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
}

func TestValidateSessionLockPolicyCoverage_DetectsUncoveredNamespace(t *testing.T) {
	registries, err := engine.BuildRegistries(daggerheart.NewModule())
	if err != nil {
		t.Fatalf("build registries: %v", err)
	}
	if err := registries.Commands.Register(command.Definition{
		Type:          command.Type("custom.command"),
		Owner:         command.OwnerCore,
		ActiveSession: command.BlockedDuringActiveSession(),
	}); err != nil {
		t.Fatalf("register custom.command: %v", err)
	}
	err = ValidateSessionLockPolicyCoverage(registries.Commands)
	if err == nil {
		t.Fatal("expected error for uncovered namespace")
	}
	if !strings.Contains(err.Error(), "custom") || !strings.Contains(err.Error(), "no RPC method maps") {
		t.Fatalf("expected namespace coverage error mentioning 'custom', got: %v", err)
	}
}
