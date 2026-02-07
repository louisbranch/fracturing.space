package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
	sessiondomain "github.com/louisbranch/fracturing.space/internal/session/domain"
	"github.com/louisbranch/fracturing.space/internal/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeSessionStore struct {
	activeSession sessiondomain.Session
	activeErr     error
}

func (f *fakeSessionStore) PutSession(ctx context.Context, session sessiondomain.Session) error {
	return errors.New("not implemented")
}

func (f *fakeSessionStore) EndSession(ctx context.Context, campaignID, sessionID string, endedAt time.Time) (sessiondomain.Session, bool, error) {
	return sessiondomain.Session{}, false, errors.New("not implemented")
}

func (f *fakeSessionStore) GetSession(ctx context.Context, campaignID, sessionID string) (sessiondomain.Session, error) {
	return sessiondomain.Session{}, errors.New("not implemented")
}

func (f *fakeSessionStore) GetActiveSession(ctx context.Context, campaignID string) (sessiondomain.Session, error) {
	return f.activeSession, f.activeErr
}

func (f *fakeSessionStore) ListSessions(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.SessionPage, error) {
	return storage.SessionPage{}, errors.New("not implemented")
}

func TestSessionLockInterceptorBlocksMutators(t *testing.T) {
	store := &fakeSessionStore{
		activeSession: sessiondomain.Session{ID: "sess-123"},
	}
	interceptor := SessionLockInterceptor(store)

	handlerCalled := false
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: campaignv1.CampaignService_CreateParticipant_FullMethodName}
	_, err := interceptor(context.Background(), &campaignv1.CreateParticipantRequest{CampaignId: "camp-1"}, info, handler)
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.FailedPrecondition {
		t.Fatalf("expected failed precondition, got %v", st.Code())
	}
	if !strings.Contains(st.Message(), "campaign has an active session") {
		t.Fatalf("expected active session message, got %q", st.Message())
	}
	if !strings.Contains(st.Message(), "active_session_id=sess-123") {
		t.Fatalf("expected active session id in message, got %q", st.Message())
	}
	if handlerCalled {
		t.Fatal("expected handler not to be called")
	}
}

func TestSessionLockInterceptorAllowsReadMethods(t *testing.T) {
	store := &fakeSessionStore{
		activeSession: sessiondomain.Session{ID: "sess-123"},
	}
	interceptor := SessionLockInterceptor(store)

	handlerCalled := false
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: campaignv1.CampaignService_ListParticipants_FullMethodName}
	_, err := interceptor(context.Background(), &campaignv1.ListParticipantsRequest{CampaignId: "camp-1"}, info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Fatal("expected handler to be called")
	}
}

func TestSessionLockInterceptorAllowsMutatorsWithoutActiveSession(t *testing.T) {
	store := &fakeSessionStore{
		activeErr: storage.ErrNotFound,
	}
	interceptor := SessionLockInterceptor(store)

	handlerCalled := false
	handler := func(ctx context.Context, req any) (any, error) {
		handlerCalled = true
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: campaignv1.CampaignService_CreateCharacter_FullMethodName}
	_, err := interceptor(context.Background(), &campaignv1.CreateCharacterRequest{CampaignId: "camp-1"}, info, handler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handlerCalled {
		t.Fatal("expected handler to be called")
	}
}

func TestSessionLockInterceptorRequiresCampaignID(t *testing.T) {
	store := &fakeSessionStore{
		activeSession: sessiondomain.Session{ID: "sess-123"},
	}
	interceptor := SessionLockInterceptor(store)

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: campaignv1.CampaignService_PatchCharacterState_FullMethodName}
	_, err := interceptor(context.Background(), &campaignv1.PatchCharacterStateRequest{}, info, handler)
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

func TestSessionLockInterceptorRequiresSessionStore(t *testing.T) {
	interceptor := SessionLockInterceptor(nil)

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: campaignv1.CampaignService_CreateParticipant_FullMethodName}
	_, err := interceptor(context.Background(), &campaignv1.CreateParticipantRequest{CampaignId: "camp-1"}, info, handler)
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
	if st.Message() != "session store is not configured" {
		t.Fatalf("unexpected message: %q", st.Message())
	}
}

func TestSessionLockInterceptorPropagatesSessionStoreError(t *testing.T) {
	store := &fakeSessionStore{
		activeErr: errors.New("db unavailable"),
	}
	interceptor := SessionLockInterceptor(store)

	handler := func(ctx context.Context, req any) (any, error) {
		return "ok", nil
	}

	info := &grpc.UnaryServerInfo{FullMethod: campaignv1.CampaignService_CreateParticipant_FullMethodName}
	_, err := interceptor(context.Background(), &campaignv1.CreateParticipantRequest{CampaignId: "camp-1"}, info, handler)
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
	if !strings.Contains(st.Message(), "check active session") {
		t.Fatalf("expected active session check message, got %q", st.Message())
	}
}
