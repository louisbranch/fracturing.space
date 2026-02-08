package interceptors

import (
	"context"
	"errors"
	"testing"
	"time"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// fakeSessionStore is a test double for storage.SessionStore.
type fakeSessionStore struct {
	activeSession map[string]session.Session // campaignID -> Session
	activeErr     error
}

func newFakeSessionStore() *fakeSessionStore {
	return &fakeSessionStore{
		activeSession: make(map[string]session.Session),
	}
}

func (s *fakeSessionStore) PutSession(_ context.Context, sess session.Session) error {
	return nil
}

func (s *fakeSessionStore) EndSession(_ context.Context, campaignID, sessionID string, endedAt time.Time) (session.Session, bool, error) {
	return session.Session{}, false, nil
}

func (s *fakeSessionStore) GetSession(_ context.Context, campaignID, sessionID string) (session.Session, error) {
	return session.Session{}, nil
}

func (s *fakeSessionStore) GetActiveSession(_ context.Context, campaignID string) (session.Session, error) {
	if s.activeErr != nil {
		return session.Session{}, s.activeErr
	}
	sess, ok := s.activeSession[campaignID]
	if !ok {
		return session.Session{}, storage.ErrNotFound
	}
	return sess, nil
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
	req := &statev1.CreateParticipantRequest{CampaignId: "c1", DisplayName: "Test"}

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
	sessionStore.activeSession["c1"] = session.Session{ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: now}

	interceptor := SessionLockInterceptor(sessionStore)
	info := serverInfo(statev1.ParticipantService_CreateParticipant_FullMethodName)
	req := &statev1.CreateParticipantRequest{CampaignId: "c1", DisplayName: "Test"}

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestSessionLockInterceptor_CreateCharacter_WithActiveSession_Blocks(t *testing.T) {
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	sessionStore.activeSession["c1"] = session.Session{ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: now}

	interceptor := SessionLockInterceptor(sessionStore)
	info := serverInfo(statev1.CharacterService_CreateCharacter_FullMethodName)
	req := &statev1.CreateCharacterRequest{CampaignId: "c1", Name: "Hero"}

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestSessionLockInterceptor_SetDefaultControl_WithActiveSession_Blocks(t *testing.T) {
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	sessionStore.activeSession["c1"] = session.Session{ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: now}

	interceptor := SessionLockInterceptor(sessionStore)
	info := serverInfo(statev1.CharacterService_SetDefaultControl_FullMethodName)
	req := &statev1.SetDefaultControlRequest{CampaignId: "c1", CharacterId: "ch1"}

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestSessionLockInterceptor_PatchCharacterProfile_WithActiveSession_Blocks(t *testing.T) {
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	sessionStore.activeSession["c1"] = session.Session{ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: now}

	interceptor := SessionLockInterceptor(sessionStore)
	info := serverInfo(statev1.CharacterService_PatchCharacterProfile_FullMethodName)
	req := &statev1.PatchCharacterProfileRequest{CampaignId: "c1", CharacterId: "ch1"}

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestSessionLockInterceptor_PatchCharacterState_WithActiveSession_Blocks(t *testing.T) {
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	sessionStore.activeSession["c1"] = session.Session{ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: now}

	interceptor := SessionLockInterceptor(sessionStore)
	info := serverInfo(statev1.SnapshotService_PatchCharacterState_FullMethodName)
	req := &statev1.PatchCharacterStateRequest{CampaignId: "c1", CharacterId: "ch1"}

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	assertStatusCode(t, err, codes.FailedPrecondition)
}

func TestSessionLockInterceptor_BlockedMethod_MissingCampaignId_ReturnsInvalidArgument(t *testing.T) {
	sessionStore := newFakeSessionStore()
	interceptor := SessionLockInterceptor(sessionStore)

	info := serverInfo(statev1.ParticipantService_CreateParticipant_FullMethodName)
	req := &statev1.CreateParticipantRequest{DisplayName: "Test"} // No CampaignId

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	assertStatusCode(t, err, codes.InvalidArgument)
}

func TestSessionLockInterceptor_BlockedMethod_NilSessionStore_ReturnsInternal(t *testing.T) {
	interceptor := SessionLockInterceptor(nil)

	info := serverInfo(statev1.ParticipantService_CreateParticipant_FullMethodName)
	req := &statev1.CreateParticipantRequest{CampaignId: "c1", DisplayName: "Test"}

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionLockInterceptor_BlockedMethod_SessionStoreError_ReturnsInternal(t *testing.T) {
	sessionStore := newFakeSessionStore()
	sessionStore.activeErr = errors.New("database error")

	interceptor := SessionLockInterceptor(sessionStore)
	info := serverInfo(statev1.ParticipantService_CreateParticipant_FullMethodName)
	req := &statev1.CreateParticipantRequest{CampaignId: "c1", DisplayName: "Test"}

	_, err := interceptor(context.Background(), req, info, fakeHandler)
	assertStatusCode(t, err, codes.Internal)
}

func TestSessionLockInterceptor_DifferentCampaigns_NotBlocked(t *testing.T) {
	sessionStore := newFakeSessionStore()
	now := time.Now().UTC()
	// Active session for campaign c1
	sessionStore.activeSession["c1"] = session.Session{ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: now}

	interceptor := SessionLockInterceptor(sessionStore)
	info := serverInfo(statev1.ParticipantService_CreateParticipant_FullMethodName)
	// Request for campaign c2 (no active session)
	req := &statev1.CreateParticipantRequest{CampaignId: "c2", DisplayName: "Test"}

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
	sessionStore.activeSession["c1"] = session.Session{ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: now}

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
	sessionStore.activeSession["c1"] = session.Session{ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: now}

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
	sessionStore.activeSession["c1"] = session.Session{ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: now}

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
	sessionStore.activeSession["c1"] = session.Session{ID: "s1", CampaignID: "c1", Status: session.SessionStatusActive, StartedAt: now}

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

func TestIsBlockedMethod(t *testing.T) {
	tests := []struct {
		method  string
		blocked bool
	}{
		// Blocked methods
		{statev1.ParticipantService_CreateParticipant_FullMethodName, true},
		{statev1.CharacterService_CreateCharacter_FullMethodName, true},
		{statev1.CharacterService_SetDefaultControl_FullMethodName, true},
		{statev1.CharacterService_PatchCharacterProfile_FullMethodName, true},
		{statev1.SnapshotService_PatchCharacterState_FullMethodName, true},

		// Not blocked methods
		{statev1.ParticipantService_ListParticipants_FullMethodName, false},
		{statev1.ParticipantService_GetParticipant_FullMethodName, false},
		{statev1.CharacterService_ListCharacters_FullMethodName, false},
		{statev1.CharacterService_GetCharacterSheet_FullMethodName, false},
		{statev1.SnapshotService_GetSnapshot_FullMethodName, false},
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

// assertStatusCode verifies the gRPC status code for an error.
func assertStatusCode(t *testing.T, err error, want codes.Code) {
	t.Helper()

	if err == nil {
		t.Fatalf("expected error with code %v", want)
	}
	statusErr, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected gRPC status error, got %T", err)
	}
	if statusErr.Code() != want {
		t.Fatalf("status code = %v, want %v (message: %s)", statusErr.Code(), want, statusErr.Message())
	}
}
