package service

import (
	"context"
	"errors"
	"testing"
	"time"

	sessionv1 "github.com/louisbranch/duality-engine/api/gen/go/session/v1"
	"github.com/louisbranch/duality-engine/internal/campaign/domain"
	sessiondomain "github.com/louisbranch/duality-engine/internal/session/domain"
	"github.com/louisbranch/duality-engine/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type fakeCampaignStore struct {
	getCampaign domain.Campaign
	getErr      error
	getFunc     func(ctx context.Context, id string) (domain.Campaign, error)
}

func (f *fakeCampaignStore) Put(ctx context.Context, campaign domain.Campaign) error {
	return nil
}

func (f *fakeCampaignStore) Get(ctx context.Context, id string) (domain.Campaign, error) {
	if f.getFunc != nil {
		return f.getFunc(ctx, id)
	}
	return f.getCampaign, f.getErr
}

func (f *fakeCampaignStore) List(ctx context.Context, pageSize int, pageToken string) (storage.CampaignPage, error) {
	return storage.CampaignPage{}, nil
}

type fakeSessionStore struct {
	putSession              sessiondomain.Session
	putErr                  error
	putWithActiveSession    sessiondomain.Session
	putWithActiveErr        error
	getSession              sessiondomain.Session
	getSessionErr           error
	getActiveSession        sessiondomain.Session
	getActiveSessionErr     error
	listPage                storage.SessionPage
	listErr                 error
	listPageSize            int
	listPageToken           string
	listPageCampaignID      string
}

func (f *fakeSessionStore) PutSession(ctx context.Context, session sessiondomain.Session) error {
	f.putSession = session
	return f.putErr
}

func (f *fakeSessionStore) GetSession(ctx context.Context, campaignID, sessionID string) (sessiondomain.Session, error) {
	return f.getSession, f.getSessionErr
}

func (f *fakeSessionStore) GetActiveSession(ctx context.Context, campaignID string) (sessiondomain.Session, error) {
	return f.getActiveSession, f.getActiveSessionErr
}

func (f *fakeSessionStore) PutSessionWithActivePointer(ctx context.Context, session sessiondomain.Session) error {
	f.putWithActiveSession = session
	return f.putWithActiveErr
}

func (f *fakeSessionStore) ListSessions(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.SessionPage, error) {
	f.listPageCampaignID = campaignID
	f.listPageSize = pageSize
	f.listPageToken = pageToken
	return f.listPage, f.listErr
}

func TestStartSessionSuccess(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaignStore := &fakeCampaignStore{
		getCampaign: domain.Campaign{
			ID:   "camp-123",
			Name: "Test Campaign",
		},
	}
	sessionStore := &fakeSessionStore{
		getActiveSessionErr: storage.ErrNotFound,
	}
	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
		},
		clock: func() time.Time {
			return fixedTime
		},
		idGenerator: func() (string, error) {
			return "sess-456", nil
		},
	}

	response, err := service.StartSession(context.Background(), &sessionv1.StartSessionRequest{
		CampaignId: "camp-123",
		Name:        "  First Session ",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if response == nil || response.Session == nil {
		t.Fatal("expected session response")
	}
	if response.Session.Id != "sess-456" {
		t.Fatalf("expected id sess-456, got %q", response.Session.Id)
	}
	if response.Session.CampaignId != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", response.Session.CampaignId)
	}
	if response.Session.Name != "First Session" {
		t.Fatalf("expected trimmed name, got %q", response.Session.Name)
	}
	if response.Session.Status != sessionv1.SessionStatus_ACTIVE {
		t.Fatalf("expected ACTIVE status, got %v", response.Session.Status)
	}
	if response.Session.StartedAt.AsTime() != fixedTime {
		t.Fatalf("expected started_at %v, got %v", fixedTime, response.Session.StartedAt.AsTime())
	}
	if response.Session.UpdatedAt.AsTime() != fixedTime {
		t.Fatalf("expected updated_at %v, got %v", fixedTime, response.Session.UpdatedAt.AsTime())
	}
	if sessionStore.putWithActiveSession.ID != "sess-456" {
		t.Fatalf("expected stored id sess-456, got %q", sessionStore.putWithActiveSession.ID)
	}
}

func TestStartSessionWithEmptyName(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaignStore := &fakeCampaignStore{
		getCampaign: domain.Campaign{
			ID:   "camp-123",
			Name: "Test Campaign",
		},
	}
	sessionStore := &fakeSessionStore{
		getActiveSessionErr: storage.ErrNotFound,
	}
	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
		},
		clock: func() time.Time {
			return fixedTime
		},
		idGenerator: func() (string, error) {
			return "sess-456", nil
		},
	}

	response, err := service.StartSession(context.Background(), &sessionv1.StartSessionRequest{
		CampaignId: "camp-123",
		Name:        "",
	})
	if err != nil {
		t.Fatalf("start session: %v", err)
	}
	if response.Session.Name != "" {
		t.Fatalf("expected empty name, got %q", response.Session.Name)
	}
}

func TestStartSessionCampaignNotFound(t *testing.T) {
	campaignStore := &fakeCampaignStore{
		getErr: storage.ErrNotFound,
	}
	sessionStore := &fakeSessionStore{}
	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
		},
		clock:       time.Now,
		idGenerator: func() (string, error) { return "sess-1", nil },
	}

	_, err := service.StartSession(context.Background(), &sessionv1.StartSessionRequest{
		CampaignId: "camp-123",
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
	if st.Message() != "campaign not found" {
		t.Fatalf("expected 'campaign not found' message, got %q", st.Message())
	}
}

func TestStartSessionActiveSessionExists(t *testing.T) {
	campaignStore := &fakeCampaignStore{
		getCampaign: domain.Campaign{
			ID:   "camp-123",
			Name: "Test Campaign",
		},
	}
	sessionStore := &fakeSessionStore{
		getActiveSession: sessiondomain.Session{
			ID:         "sess-existing",
			CampaignID: "camp-123",
			Status:     sessiondomain.SessionStatusActive,
		},
	}
	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
		},
		clock:       time.Now,
		idGenerator: func() (string, error) { return "sess-1", nil },
	}

	_, err := service.StartSession(context.Background(), &sessionv1.StartSessionRequest{
		CampaignId: "camp-123",
	})
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
	if st.Message() != "active session exists" {
		t.Fatalf("expected 'active session exists' message, got %q", st.Message())
	}
}

func TestStartSessionEmptyCampaignID(t *testing.T) {
	service := &SessionService{
		stores: Stores{
			Campaign: &fakeCampaignStore{},
			Session:  &fakeSessionStore{},
		},
		clock:       time.Now,
		idGenerator: func() (string, error) { return "sess-1", nil },
	}

	_, err := service.StartSession(context.Background(), &sessionv1.StartSessionRequest{
		CampaignId: "  ",
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
	if st.Message() != "campaign id is required" {
		t.Fatalf("expected 'campaign id is required' message, got %q", st.Message())
	}
}

func TestStartSessionNilRequest(t *testing.T) {
	service := NewSessionService(Stores{
		Campaign: &fakeCampaignStore{},
		Session:  &fakeSessionStore{},
	})

	_, err := service.StartSession(context.Background(), nil)
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

func TestStartSessionIDGenerationFailure(t *testing.T) {
	campaignStore := &fakeCampaignStore{
		getCampaign: domain.Campaign{
			ID:   "camp-123",
			Name: "Test Campaign",
		},
	}
	sessionStore := &fakeSessionStore{
		getActiveSessionErr: storage.ErrNotFound,
	}
	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
		},
		clock: time.Now,
		idGenerator: func() (string, error) {
			return "", errors.New("boom")
		},
	}

	_, err := service.StartSession(context.Background(), &sessionv1.StartSessionRequest{
		CampaignId: "camp-123",
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

func TestStartSessionStoreFailure(t *testing.T) {
	campaignStore := &fakeCampaignStore{
		getCampaign: domain.Campaign{
			ID:   "camp-123",
			Name: "Test Campaign",
		},
	}
	sessionStore := &fakeSessionStore{
		getActiveSessionErr: storage.ErrNotFound,
		putWithActiveErr:    errors.New("boom"),
	}
	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
		},
		clock: time.Now,
		idGenerator: func() (string, error) {
			return "sess-123", nil
		},
	}

	_, err := service.StartSession(context.Background(), &sessionv1.StartSessionRequest{
		CampaignId: "camp-123",
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

func TestStartSessionActiveSessionConflict(t *testing.T) {
	campaignStore := &fakeCampaignStore{
		getCampaign: domain.Campaign{
			ID:   "camp-123",
			Name: "Test Campaign",
		},
	}
	sessionStore := &fakeSessionStore{
		getActiveSessionErr: storage.ErrNotFound,
		putWithActiveErr:    storage.ErrActiveSessionExists,
	}
	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
		},
		clock: time.Now,
		idGenerator: func() (string, error) {
			return "sess-123", nil
		},
	}

	_, err := service.StartSession(context.Background(), &sessionv1.StartSessionRequest{
		CampaignId: "camp-123",
	})
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
	if st.Message() != "active session exists" {
		t.Fatalf("expected 'active session exists' message, got %q", st.Message())
	}
}

func TestStartSessionMissingStore(t *testing.T) {
	service := &SessionService{
		stores: Stores{},
		clock:  time.Now,
		idGenerator: func() (string, error) {
			return "sess-123", nil
		},
	}

	_, err := service.StartSession(context.Background(), &sessionv1.StartSessionRequest{
		CampaignId: "camp-123",
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

func TestListSessionsSuccess(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	endedTime := time.Date(2026, 1, 24, 12, 0, 0, 0, time.UTC)
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}
	sessionStore := &fakeSessionStore{
		listPage: storage.SessionPage{
			Sessions: []sessiondomain.Session{
				{
					ID:         "session-1",
					CampaignID: "camp-123",
					Name:       "Session One",
					Status:     sessiondomain.SessionStatusActive,
					StartedAt:  fixedTime,
					UpdatedAt:  fixedTime,
					EndedAt:    nil,
				},
				{
					ID:         "session-2",
					CampaignID: "camp-123",
					Name:       "Session Two",
					Status:     sessiondomain.SessionStatusEnded,
					StartedAt:  fixedTime,
					UpdatedAt:  fixedTime,
					EndedAt:    &endedTime,
				},
			},
			NextPageToken: "next-token",
		},
	}
	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
		},
		clock: time.Now,
	}

	response, err := service.ListSessions(context.Background(), &sessionv1.ListSessionsRequest{
		CampaignId: "camp-123",
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if response == nil {
		t.Fatal("expected response")
	}
	if len(response.Sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(response.Sessions))
	}
	if response.NextPageToken != "next-token" {
		t.Fatalf("expected next page token next-token, got %q", response.NextPageToken)
	}
	if response.Sessions[0].Id != "session-1" {
		t.Fatalf("expected first session id session-1, got %q", response.Sessions[0].Id)
	}
	if response.Sessions[0].Status != sessionv1.SessionStatus_ACTIVE {
		t.Fatalf("expected first session status ACTIVE, got %v", response.Sessions[0].Status)
	}
	if response.Sessions[1].Status != sessionv1.SessionStatus_ENDED {
		t.Fatalf("expected second session status ENDED, got %v", response.Sessions[1].Status)
	}
	if sessionStore.listPageCampaignID != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", sessionStore.listPageCampaignID)
	}
}

func TestListSessionsDefaults(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	sessionStore := &fakeSessionStore{
		listPage: storage.SessionPage{
			Sessions: []sessiondomain.Session{
				{
					ID:         "session-1",
					CampaignID: "camp-123",
					Name:       "Session One",
					Status:     sessiondomain.SessionStatusPaused,
					StartedAt:  fixedTime,
					UpdatedAt:  fixedTime,
					EndedAt:    nil,
				},
			},
			NextPageToken: "next-token",
		},
	}
	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
		},
		clock: time.Now,
	}

	response, err := service.ListSessions(context.Background(), &sessionv1.ListSessionsRequest{
		CampaignId: "camp-123",
		PageSize:   0,
	})
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if response == nil {
		t.Fatal("expected response")
	}
	if sessionStore.listPageSize != defaultListSessionsPageSize {
		t.Fatalf("expected default page size %d, got %d", defaultListSessionsPageSize, sessionStore.listPageSize)
	}
	if response.NextPageToken != "next-token" {
		t.Fatalf("expected next page token, got %q", response.NextPageToken)
	}
	if len(response.Sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(response.Sessions))
	}
}

func TestListSessionsEmpty(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	sessionStore := &fakeSessionStore{
		listPage: storage.SessionPage{},
	}
	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
		},
		clock: time.Now,
	}

	response, err := service.ListSessions(context.Background(), &sessionv1.ListSessionsRequest{
		CampaignId: "camp-123",
		PageSize:   10,
	})
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if response == nil {
		t.Fatal("expected response")
	}
	if len(response.Sessions) != 0 {
		t.Fatalf("expected 0 sessions, got %d", len(response.Sessions))
	}
}

func TestListSessionsClampPageSize(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	sessionStore := &fakeSessionStore{listPage: storage.SessionPage{}}
	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
		},
		clock: time.Now,
	}

	_, err := service.ListSessions(context.Background(), &sessionv1.ListSessionsRequest{
		CampaignId: "camp-123",
		PageSize:   25,
	})
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if sessionStore.listPageSize != maxListSessionsPageSize {
		t.Fatalf("expected max page size %d, got %d", maxListSessionsPageSize, sessionStore.listPageSize)
	}
}

func TestListSessionsPassesToken(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	sessionStore := &fakeSessionStore{listPage: storage.SessionPage{}}
	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
		},
		clock: time.Now,
	}

	_, err := service.ListSessions(context.Background(), &sessionv1.ListSessionsRequest{
		CampaignId: "camp-123",
		PageSize:   1,
		PageToken:  "next",
	})
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if sessionStore.listPageToken != "next" {
		t.Fatalf("expected page token next, got %q", sessionStore.listPageToken)
	}
	if sessionStore.listPageCampaignID != "camp-123" {
		t.Fatalf("expected campaign id camp-123, got %q", sessionStore.listPageCampaignID)
	}
}

func TestListSessionsNilRequest(t *testing.T) {
	service := &SessionService{
		stores: Stores{
			Campaign: &fakeCampaignStore{},
			Session:  &fakeSessionStore{},
		},
		clock: time.Now,
	}

	_, err := service.ListSessions(context.Background(), nil)
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

func TestListSessionsCampaignNotFound(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{}, storage.ErrNotFound
	}
	sessionStore := &fakeSessionStore{}
	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
		},
		clock: time.Now,
	}

	_, err := service.ListSessions(context.Background(), &sessionv1.ListSessionsRequest{
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

func TestListSessionsStoreFailure(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	sessionStore := &fakeSessionStore{listErr: errors.New("boom")}
	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
		},
		clock: time.Now,
	}

	_, err := service.ListSessions(context.Background(), &sessionv1.ListSessionsRequest{
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

func TestListSessionsMissingCampaignStore(t *testing.T) {
	service := &SessionService{
		stores: Stores{
			Session: &fakeSessionStore{},
		},
		clock: time.Now,
	}

	_, err := service.ListSessions(context.Background(), &sessionv1.ListSessionsRequest{
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

func TestListSessionsMissingSessionStore(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		return domain.Campaign{ID: "camp-123"}, nil
	}
	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
		},
		clock: time.Now,
	}

	_, err := service.ListSessions(context.Background(), &sessionv1.ListSessionsRequest{
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

func TestListSessionsEmptyCampaignID(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	sessionStore := &fakeSessionStore{}
	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
		},
		clock: time.Now,
	}

	_, err := service.ListSessions(context.Background(), &sessionv1.ListSessionsRequest{
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
