package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	sessionv1 "github.com/louisbranch/duality-engine/api/gen/go/session/v1"
	"github.com/louisbranch/duality-engine/internal/campaign/domain"
	"github.com/louisbranch/duality-engine/internal/grpcmeta"
	sessiondomain "github.com/louisbranch/duality-engine/internal/session/domain"
	"github.com/louisbranch/duality-engine/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
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
	putSession          sessiondomain.Session
	putErr              error
	endSession          sessiondomain.Session
	endSessionErr       error
	endSessionEndedNow  bool
	endSessionCampaign  string
	endSessionID        string
	endSessionEndedAt   time.Time
	getSession          sessiondomain.Session
	getSessionErr       error
	getActiveSession    sessiondomain.Session
	getActiveSessionErr error
	listPage            storage.SessionPage
	listErr             error
	listPageSize        int
	listPageToken       string
	listPageCampaignID  string
}

// fakeParticipantStore implements ParticipantStore for tests.
type fakeParticipantStore struct {
	putParticipant domain.Participant
	putErr         error
	getParticipant domain.Participant
	getErr         error
	listItems      []domain.Participant
	listErr        error
	listPage       storage.ParticipantPage
	listPageErr    error
	listCampaignID string
	listPageSize   int
	listPageToken  string
}

type fakeControlDefaultStore struct {
	putCampaignID  string
	putCharacterID string
	putController  domain.CharacterController
	putErr         error
	getController  domain.CharacterController
	getErr         error
}

// PutParticipant records the participant to store.
func (f *fakeParticipantStore) PutParticipant(ctx context.Context, participant domain.Participant) error {
	f.putParticipant = participant
	return f.putErr
}

// GetParticipant returns the configured participant.
func (f *fakeParticipantStore) GetParticipant(ctx context.Context, campaignID, participantID string) (domain.Participant, error) {
	return f.getParticipant, f.getErr
}

// ListParticipantsByCampaign returns the configured participant list.
func (f *fakeParticipantStore) ListParticipantsByCampaign(ctx context.Context, campaignID string) ([]domain.Participant, error) {
	return f.listItems, f.listErr
}

// ListParticipants returns the configured participant page.
func (f *fakeParticipantStore) ListParticipants(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.ParticipantPage, error) {
	f.listCampaignID = campaignID
	f.listPageSize = pageSize
	f.listPageToken = pageToken
	return f.listPage, f.listPageErr
}

func (f *fakeControlDefaultStore) PutControlDefault(ctx context.Context, campaignID, characterID string, controller domain.CharacterController) error {
	f.putCampaignID = campaignID
	f.putCharacterID = characterID
	f.putController = controller
	return f.putErr
}

func (f *fakeControlDefaultStore) GetControlDefault(ctx context.Context, campaignID, characterID string) (domain.CharacterController, error) {
	return f.getController, f.getErr
}

// fakeRollOutcomeStore implements RollOutcomeStore for tests.
type fakeRollOutcomeStore struct {
	applyInput  storage.RollOutcomeApplyInput
	applyResult storage.RollOutcomeApplyResult
	applyErr    error
}

// ApplyRollOutcome records the apply input and returns the configured result.
func (f *fakeRollOutcomeStore) ApplyRollOutcome(ctx context.Context, input storage.RollOutcomeApplyInput) (storage.RollOutcomeApplyResult, error) {
	f.applyInput = input
	return f.applyResult, f.applyErr
}

func (f *fakeSessionStore) PutSession(ctx context.Context, session sessiondomain.Session) error {
	f.putSession = session
	return f.putErr
}

func (f *fakeSessionStore) EndSession(ctx context.Context, campaignID, sessionID string, endedAt time.Time) (sessiondomain.Session, bool, error) {
	f.endSessionCampaign = campaignID
	f.endSessionID = sessionID
	f.endSessionEndedAt = endedAt
	return f.endSession, f.endSessionEndedNow, f.endSessionErr
}

func (f *fakeSessionStore) GetSession(ctx context.Context, campaignID, sessionID string) (sessiondomain.Session, error) {
	return f.getSession, f.getSessionErr
}

func (f *fakeSessionStore) GetActiveSession(ctx context.Context, campaignID string) (sessiondomain.Session, error) {
	return f.getActiveSession, f.getActiveSessionErr
}

func (f *fakeSessionStore) ListSessions(ctx context.Context, campaignID string, pageSize int, pageToken string) (storage.SessionPage, error) {
	f.listPageCampaignID = campaignID
	f.listPageSize = pageSize
	f.listPageToken = pageToken
	return f.listPage, f.listErr
}

type fakeSessionEventStore struct {
	appendInputs []sessiondomain.SessionEvent
	appendErr    error
	listEvents   []sessiondomain.SessionEvent
	listErr      error
	listSession  string
	listAfterSeq uint64
	listLimit    int
}

func (f *fakeSessionEventStore) AppendSessionEvent(ctx context.Context, event sessiondomain.SessionEvent) (sessiondomain.SessionEvent, error) {
	if f.appendErr != nil {
		return sessiondomain.SessionEvent{}, f.appendErr
	}
	if event.Seq == 0 {
		event.Seq = uint64(len(f.appendInputs) + 1)
	}
	f.appendInputs = append(f.appendInputs, event)
	return event, nil
}

func (f *fakeSessionEventStore) ListSessionEvents(ctx context.Context, sessionID string, afterSeq uint64, limit int) ([]sessiondomain.SessionEvent, error) {
	f.listSession = sessionID
	f.listAfterSeq = afterSeq
	f.listLimit = limit
	if f.listErr != nil {
		return nil, f.listErr
	}
	filtered := make([]sessiondomain.SessionEvent, 0, len(f.listEvents))
	for _, event := range f.listEvents {
		if event.Seq > afterSeq {
			filtered = append(filtered, event)
		}
		if limit > 0 && len(filtered) >= limit {
			break
		}
	}
	return filtered, nil
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
		Name:       "  First Session ",
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
	if sessionStore.putSession.ID != "sess-456" {
		t.Fatalf("expected stored id sess-456, got %q", sessionStore.putSession.ID)
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
		Name:       "",
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
		putErr:              errors.New("boom"),
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
		putErr:              storage.ErrActiveSessionExists,
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
					Status:     sessiondomain.SessionStatusEnded,
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

func TestGetSessionSuccess(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	endedTime := time.Date(2026, 1, 23, 14, 0, 0, 0, time.UTC)
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}
	sessionStore := &fakeSessionStore{
		getSession: sessiondomain.Session{
			ID:         "sess-456",
			CampaignID: "camp-123",
			Name:       "Test Session",
			Status:     sessiondomain.SessionStatusEnded,
			StartedAt:  fixedTime,
			UpdatedAt:  fixedTime,
			EndedAt:    &endedTime,
		},
	}
	service := NewSessionService(Stores{
		Campaign: campaignStore,
		Session:  sessionStore,
	})

	response, err := service.GetSession(context.Background(), &sessionv1.GetSessionRequest{
		CampaignId: "camp-123",
		SessionId:  "sess-456",
	})
	if err != nil {
		t.Fatalf("get session: %v", err)
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
	if response.Session.Name != "Test Session" {
		t.Fatalf("expected name Test Session, got %q", response.Session.Name)
	}
	if response.Session.Status != sessionv1.SessionStatus_ENDED {
		t.Fatalf("expected status ENDED, got %v", response.Session.Status)
	}
	if response.Session.StartedAt.AsTime() != fixedTime {
		t.Fatalf("expected started_at %v, got %v", fixedTime, response.Session.StartedAt.AsTime())
	}
	if response.Session.UpdatedAt.AsTime() != fixedTime {
		t.Fatalf("expected updated_at %v, got %v", fixedTime, response.Session.UpdatedAt.AsTime())
	}
	if response.Session.EndedAt == nil {
		t.Fatal("expected ended_at to be set")
	}
	if response.Session.EndedAt.AsTime() != endedTime {
		t.Fatalf("expected ended_at %v, got %v", endedTime, response.Session.EndedAt.AsTime())
	}
}

func TestGetSessionSuccessWithoutEndedAt(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}
	sessionStore := &fakeSessionStore{
		getSession: sessiondomain.Session{
			ID:         "sess-456",
			CampaignID: "camp-123",
			Name:       "Active Session",
			Status:     sessiondomain.SessionStatusActive,
			StartedAt:  fixedTime,
			UpdatedAt:  fixedTime,
			EndedAt:    nil,
		},
	}
	service := NewSessionService(Stores{
		Campaign: campaignStore,
		Session:  sessionStore,
	})

	response, err := service.GetSession(context.Background(), &sessionv1.GetSessionRequest{
		CampaignId: "camp-123",
		SessionId:  "sess-456",
	})
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	if response == nil || response.Session == nil {
		t.Fatal("expected session response")
	}
	if response.Session.EndedAt != nil {
		t.Fatal("expected ended_at to be nil for active session")
	}
}

func TestGetSessionNilRequest(t *testing.T) {
	service := NewSessionService(Stores{
		Campaign: &fakeCampaignStore{},
		Session:  &fakeSessionStore{},
	})

	_, err := service.GetSession(context.Background(), nil)
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

func TestGetSessionMissingCampaignStore(t *testing.T) {
	service := &SessionService{
		stores: Stores{
			Session: &fakeSessionStore{},
		},
		clock: time.Now,
	}

	_, err := service.GetSession(context.Background(), &sessionv1.GetSessionRequest{
		CampaignId: "camp-123",
		SessionId:  "sess-456",
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

func TestGetSessionMissingSessionStore(t *testing.T) {
	service := &SessionService{
		stores: Stores{
			Campaign: &fakeCampaignStore{},
		},
		clock: time.Now,
	}

	_, err := service.GetSession(context.Background(), &sessionv1.GetSessionRequest{
		CampaignId: "camp-123",
		SessionId:  "sess-456",
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

func TestGetSessionEmptyCampaignID(t *testing.T) {
	service := NewSessionService(Stores{
		Campaign: &fakeCampaignStore{},
		Session:  &fakeSessionStore{},
	})

	_, err := service.GetSession(context.Background(), &sessionv1.GetSessionRequest{
		CampaignId: "  ",
		SessionId:  "sess-456",
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

func TestGetSessionEmptySessionID(t *testing.T) {
	service := NewSessionService(Stores{
		Campaign: &fakeCampaignStore{},
		Session:  &fakeSessionStore{},
	})

	_, err := service.GetSession(context.Background(), &sessionv1.GetSessionRequest{
		CampaignId: "camp-123",
		SessionId:  "  ",
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

func TestGetSessionCampaignNotFound(t *testing.T) {
	campaignStore := &fakeCampaignStore{
		getErr: storage.ErrNotFound,
	}
	service := NewSessionService(Stores{
		Campaign: campaignStore,
		Session:  &fakeSessionStore{},
	})

	_, err := service.GetSession(context.Background(), &sessionv1.GetSessionRequest{
		CampaignId: "camp-999",
		SessionId:  "sess-456",
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
		t.Fatalf("expected message 'campaign not found', got %q", st.Message())
	}
}

func TestGetSessionNotFound(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}
	sessionStore := &fakeSessionStore{
		getSessionErr: storage.ErrNotFound,
	}
	service := NewSessionService(Stores{
		Campaign: campaignStore,
		Session:  sessionStore,
	})

	_, err := service.GetSession(context.Background(), &sessionv1.GetSessionRequest{
		CampaignId: "camp-123",
		SessionId:  "sess-999",
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
	if st.Message() != "session not found" {
		t.Fatalf("expected message 'session not found', got %q", st.Message())
	}
}

func TestGetSessionStoreError(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}
	sessionStore := &fakeSessionStore{
		getSessionErr: errors.New("database error"),
	}
	service := NewSessionService(Stores{
		Campaign: campaignStore,
		Session:  sessionStore,
	})

	_, err := service.GetSession(context.Background(), &sessionv1.GetSessionRequest{
		CampaignId: "camp-123",
		SessionId:  "sess-456",
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

func TestEndSessionSuccess(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}
	sessionStore := &fakeSessionStore{
		endSession: sessiondomain.Session{
			ID:         "sess-456",
			CampaignID: "camp-123",
			Name:       "Test Session",
			Status:     sessiondomain.SessionStatusEnded,
			StartedAt:  fixedTime.Add(-time.Hour),
			UpdatedAt:  fixedTime,
			EndedAt:    &fixedTime,
		},
		endSessionEndedNow: true,
	}
	eventStore := &fakeSessionEventStore{}
	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
			Event:    eventStore,
		},
		clock: func() time.Time { return fixedTime },
	}

	response, err := service.EndSession(context.Background(), &sessionv1.EndSessionRequest{
		CampaignId: "camp-123",
		SessionId:  "sess-456",
	})
	if err != nil {
		t.Fatalf("end session: %v", err)
	}
	if response == nil || response.Session == nil {
		t.Fatal("expected session response")
	}
	if sessionStore.endSessionCampaign != "camp-123" {
		t.Fatalf("expected end session campaign camp-123, got %q", sessionStore.endSessionCampaign)
	}
	if sessionStore.endSessionID != "sess-456" {
		t.Fatalf("expected end session id sess-456, got %q", sessionStore.endSessionID)
	}
	if !sessionStore.endSessionEndedAt.Equal(fixedTime) {
		t.Fatalf("expected end session time %v, got %v", fixedTime, sessionStore.endSessionEndedAt)
	}
	if response.Session.Status != sessionv1.SessionStatus_ENDED {
		t.Fatalf("expected status ENDED, got %v", response.Session.Status)
	}
	if response.Session.EndedAt == nil {
		t.Fatal("expected ended_at to be set")
	}
	if response.Session.EndedAt.AsTime() != fixedTime {
		t.Fatalf("expected ended_at %v, got %v", fixedTime, response.Session.EndedAt.AsTime())
	}
	if len(eventStore.appendInputs) != 1 {
		t.Fatalf("expected 1 event, got %d", len(eventStore.appendInputs))
	}
	if eventStore.appendInputs[0].Type != sessiondomain.SessionEventTypeSessionEnded {
		t.Fatalf("expected session ended event, got %v", eventStore.appendInputs[0].Type)
	}
	var payload sessionEndedPayload
	if err := json.Unmarshal(eventStore.appendInputs[0].PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal session ended payload: %v", err)
	}
	if payload.CampaignID != "camp-123" {
		t.Fatalf("expected campaign_id camp-123, got %q", payload.CampaignID)
	}
}

func TestEndSessionIdempotent(t *testing.T) {
	fixedTime := time.Date(2026, 1, 23, 12, 0, 0, 0, time.UTC)
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}
	sessionStore := &fakeSessionStore{
		endSession: sessiondomain.Session{
			ID:         "sess-456",
			CampaignID: "camp-123",
			Name:       "Test Session",
			Status:     sessiondomain.SessionStatusEnded,
			StartedAt:  fixedTime.Add(-time.Hour),
			UpdatedAt:  fixedTime,
			EndedAt:    &fixedTime,
		},
		endSessionEndedNow: false,
	}
	eventStore := &fakeSessionEventStore{}
	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
			Event:    eventStore,
		},
		clock: func() time.Time { return fixedTime },
	}

	response, err := service.EndSession(context.Background(), &sessionv1.EndSessionRequest{
		CampaignId: "camp-123",
		SessionId:  "sess-456",
	})
	if err != nil {
		t.Fatalf("end session: %v", err)
	}
	if response == nil || response.Session == nil {
		t.Fatal("expected session response")
	}
	if len(eventStore.appendInputs) != 0 {
		t.Fatalf("expected no events, got %d", len(eventStore.appendInputs))
	}
}

func TestEndSessionNotFound(t *testing.T) {
	campaignStore := &fakeCampaignStore{}
	campaignStore.getFunc = func(ctx context.Context, id string) (domain.Campaign, error) {
		if id == "camp-123" {
			return domain.Campaign{ID: "camp-123", Name: "Test Campaign"}, nil
		}
		return domain.Campaign{}, storage.ErrNotFound
	}
	sessionStore := &fakeSessionStore{
		endSessionErr: storage.ErrNotFound,
	}
	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
			Event:    &fakeSessionEventStore{},
		},
		clock: time.Now,
	}

	_, err := service.EndSession(context.Background(), &sessionv1.EndSessionRequest{
		CampaignId: "camp-123",
		SessionId:  "sess-999",
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

// TestSessionEventAppendUsesMetadata ensures metadata fills in missing fields.
func TestSessionEventAppendUsesMetadata(t *testing.T) {
	fixedTime := time.Date(2026, 1, 26, 9, 0, 0, 0, time.UTC)
	eventStore := &fakeSessionEventStore{}
	service := &SessionService{
		stores: Stores{
			Event: eventStore,
		},
		clock: func() time.Time { return fixedTime },
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "part-123"))
	ctx = grpcmeta.WithRequestID(ctx, "req-1")
	ctx = grpcmeta.WithInvocationID(ctx, "inv-1")

	response, err := service.SessionEventAppend(ctx, &sessionv1.SessionEventAppendRequest{
		SessionId:   "sess-123",
		Type:        sessionv1.SessionEventType_NOTE_ADDED,
		CharacterId: " char-9 ",
		PayloadJson: []byte(`{"note":"hello"}`),
	})
	if err != nil {
		t.Fatalf("session event append: %v", err)
	}
	if response == nil || response.Event == nil {
		t.Fatal("expected event response")
	}
	if len(eventStore.appendInputs) != 1 {
		t.Fatalf("expected 1 event stored, got %d", len(eventStore.appendInputs))
	}

	stored := eventStore.appendInputs[0]
	if stored.SessionID != "sess-123" {
		t.Fatalf("expected session id sess-123, got %q", stored.SessionID)
	}
	if stored.Type != sessiondomain.SessionEventTypeNoteAdded {
		t.Fatalf("expected event type NOTE_ADDED, got %s", stored.Type)
	}
	if stored.ParticipantID != "part-123" {
		t.Fatalf("expected participant id part-123, got %q", stored.ParticipantID)
	}
	if stored.CharacterID != "char-9" {
		t.Fatalf("expected character id char-9, got %q", stored.CharacterID)
	}
	if stored.RequestID != "req-1" {
		t.Fatalf("expected request id req-1, got %q", stored.RequestID)
	}
	if stored.InvocationID != "inv-1" {
		t.Fatalf("expected invocation id inv-1, got %q", stored.InvocationID)
	}
	if string(stored.PayloadJSON) != `{"note":"hello"}` {
		t.Fatalf("unexpected payload json %q", string(stored.PayloadJSON))
	}
	if stored.Timestamp != fixedTime {
		t.Fatalf("expected timestamp %v, got %v", fixedTime, stored.Timestamp)
	}

	if response.Event.SessionId != "sess-123" {
		t.Fatalf("expected response session id sess-123, got %q", response.Event.SessionId)
	}
	if response.Event.Type != sessionv1.SessionEventType_NOTE_ADDED {
		t.Fatalf("expected response event type NOTE_ADDED, got %v", response.Event.Type)
	}
	if response.Event.ParticipantId != "part-123" {
		t.Fatalf("expected response participant id part-123, got %q", response.Event.ParticipantId)
	}
	if response.Event.CharacterId != "char-9" {
		t.Fatalf("expected response character id char-9, got %q", response.Event.CharacterId)
	}
	if response.Event.RequestId != "req-1" {
		t.Fatalf("expected response request id req-1, got %q", response.Event.RequestId)
	}
	if response.Event.InvocationId != "inv-1" {
		t.Fatalf("expected response invocation id inv-1, got %q", response.Event.InvocationId)
	}
	if string(response.Event.PayloadJson) != `{"note":"hello"}` {
		t.Fatalf("unexpected response payload json %q", string(response.Event.PayloadJson))
	}
	if response.Event.Ts.AsTime() != fixedTime {
		t.Fatalf("expected response timestamp %v, got %v", fixedTime, response.Event.Ts.AsTime())
	}
}

// TestSessionEventAppendRejectsInvalidType verifies invalid type handling.
func TestSessionEventAppendRejectsInvalidType(t *testing.T) {
	fixedTime := time.Date(2026, 1, 26, 10, 0, 0, 0, time.UTC)
	eventStore := &fakeSessionEventStore{}
	service := &SessionService{
		stores: Stores{
			Event: eventStore,
		},
		clock: func() time.Time { return fixedTime },
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "part-9"))
	ctx = grpcmeta.WithRequestID(ctx, "req-9")
	ctx = grpcmeta.WithInvocationID(ctx, "inv-9")

	_, err := service.SessionEventAppend(ctx, &sessionv1.SessionEventAppendRequest{
		SessionId: "sess-999",
		Type:      sessionv1.SessionEventType_SESSION_EVENT_TYPE_UNSPECIFIED,
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
	if st.Message() != "event type is required" {
		t.Fatalf("expected message 'event type is required', got %q", st.Message())
	}
	if len(eventStore.appendInputs) != 1 {
		t.Fatalf("expected 1 rejected event, got %d", len(eventStore.appendInputs))
	}
	stored := eventStore.appendInputs[0]
	if stored.Type != sessiondomain.SessionEventTypeRequestRejected {
		t.Fatalf("expected REQUEST_REJECTED event, got %s", stored.Type)
	}
	if stored.ParticipantID != "part-9" {
		t.Fatalf("expected participant id part-9, got %q", stored.ParticipantID)
	}
	if stored.RequestID != "req-9" {
		t.Fatalf("expected request id req-9, got %q", stored.RequestID)
	}
	if stored.InvocationID != "inv-9" {
		t.Fatalf("expected invocation id inv-9, got %q", stored.InvocationID)
	}

	var payload requestRejectedPayload
	if err := json.Unmarshal(stored.PayloadJSON, &payload); err != nil {
		t.Fatalf("unmarshal request rejected payload: %v", err)
	}
	if payload.RPC != "session.v1.SessionService/SessionEventAppend" {
		t.Fatalf("expected rpc SessionEventAppend, got %q", payload.RPC)
	}
	if payload.ReasonCode != "INVALID_ARGUMENT" {
		t.Fatalf("expected reason code INVALID_ARGUMENT, got %q", payload.ReasonCode)
	}
	if payload.Message != "event type is required" {
		t.Fatalf("expected message 'event type is required', got %q", payload.Message)
	}
}

// TestSessionEventAppendReturnsInternalOnStoreError checks store errors.
func TestSessionEventAppendReturnsInternalOnStoreError(t *testing.T) {
	eventStore := &fakeSessionEventStore{appendErr: errors.New("store error")}
	service := &SessionService{
		stores: Stores{
			Event: eventStore,
		},
		clock: time.Now,
	}

	_, err := service.SessionEventAppend(context.Background(), &sessionv1.SessionEventAppendRequest{
		SessionId: "sess-1",
		Type:      sessionv1.SessionEventType_NOTE_ADDED,
	})
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
	if len(eventStore.appendInputs) != 0 {
		t.Fatalf("expected no events stored, got %d", len(eventStore.appendInputs))
	}
}

// TestSessionEventsListDefaultsLimit verifies the default limit behavior.
func TestSessionEventsListDefaultsLimit(t *testing.T) {
	eventStore := &fakeSessionEventStore{}
	service := &SessionService{
		stores: Stores{
			Event: eventStore,
		},
	}

	_, err := service.SessionEventsList(context.Background(), &sessionv1.SessionEventsListRequest{
		SessionId: "sess-1",
		AfterSeq:  12,
	})
	if err != nil {
		t.Fatalf("session events list: %v", err)
	}
	if eventStore.listSession != "sess-1" {
		t.Fatalf("expected session id sess-1, got %q", eventStore.listSession)
	}
	if eventStore.listAfterSeq != 12 {
		t.Fatalf("expected after seq 12, got %d", eventStore.listAfterSeq)
	}
	if eventStore.listLimit != defaultListSessionEventsLimit {
		t.Fatalf("expected limit %d, got %d", defaultListSessionEventsLimit, eventStore.listLimit)
	}
}

// TestSessionEventsListCapsLimitAndConverts ensures max limit and conversion.
func TestSessionEventsListCapsLimitAndConverts(t *testing.T) {
	fixedTime := time.Date(2026, 1, 26, 11, 0, 0, 0, time.UTC)
	eventStore := &fakeSessionEventStore{
		listEvents: []sessiondomain.SessionEvent{
			{
				SessionID:     "sess-1",
				Seq:           99,
				Timestamp:     fixedTime,
				Type:          sessiondomain.SessionEventTypeNoteAdded,
				RequestID:     "req-7",
				InvocationID:  "inv-7",
				ParticipantID: "part-7",
				CharacterID:   "char-7",
				PayloadJSON:   []byte(`{"note":"hi"}`),
			},
		},
	}
	service := &SessionService{
		stores: Stores{
			Event: eventStore,
		},
	}

	response, err := service.SessionEventsList(context.Background(), &sessionv1.SessionEventsListRequest{
		SessionId: "sess-1",
		Limit:     500,
	})
	if err != nil {
		t.Fatalf("session events list: %v", err)
	}
	if eventStore.listLimit != maxListSessionEventsLimit {
		t.Fatalf("expected limit %d, got %d", maxListSessionEventsLimit, eventStore.listLimit)
	}
	if response == nil || len(response.Events) != 1 {
		t.Fatalf("expected 1 event response, got %+v", response)
	}
	resp := response.Events[0]
	if resp.SessionId != "sess-1" {
		t.Fatalf("expected session id sess-1, got %q", resp.SessionId)
	}
	if resp.Seq != 99 {
		t.Fatalf("expected seq 99, got %d", resp.Seq)
	}
	if resp.Type != sessionv1.SessionEventType_NOTE_ADDED {
		t.Fatalf("expected type NOTE_ADDED, got %v", resp.Type)
	}
	if resp.RequestId != "req-7" {
		t.Fatalf("expected request id req-7, got %q", resp.RequestId)
	}
	if resp.InvocationId != "inv-7" {
		t.Fatalf("expected invocation id inv-7, got %q", resp.InvocationId)
	}
	if resp.ParticipantId != "part-7" {
		t.Fatalf("expected participant id part-7, got %q", resp.ParticipantId)
	}
	if resp.CharacterId != "char-7" {
		t.Fatalf("expected character id char-7, got %q", resp.CharacterId)
	}
	if string(resp.PayloadJson) != `{"note":"hi"}` {
		t.Fatalf("unexpected payload json %q", string(resp.PayloadJson))
	}
	if resp.Ts.AsTime() != fixedTime {
		t.Fatalf("expected timestamp %v, got %v", fixedTime, resp.Ts.AsTime())
	}
}

// TestSessionEventsListRejectsMissingSessionID covers session id validation.
func TestSessionEventsListRejectsMissingSessionID(t *testing.T) {
	eventStore := &fakeSessionEventStore{}
	service := &SessionService{
		stores: Stores{
			Event: eventStore,
		},
	}

	_, err := service.SessionEventsList(context.Background(), &sessionv1.SessionEventsListRequest{
		SessionId: " ",
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

// TestSessionEventsListReturnsInternalOnStoreError checks list errors.
func TestSessionEventsListReturnsInternalOnStoreError(t *testing.T) {
	eventStore := &fakeSessionEventStore{listErr: errors.New("list error")}
	service := &SessionService{
		stores: Stores{
			Event: eventStore,
		},
	}

	_, err := service.SessionEventsList(context.Background(), &sessionv1.SessionEventsListRequest{
		SessionId: "sess-1",
	})
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
	if len(eventStore.appendInputs) != 0 {
		t.Fatalf("expected no events appended, got %d", len(eventStore.appendInputs))
	}
}

func TestSessionActionRollSuccessAppendsEvents(t *testing.T) {
	fixedTime := time.Date(2026, 1, 25, 10, 0, 0, 0, time.UTC)
	campaignStore := &fakeCampaignStore{
		getCampaign: domain.Campaign{ID: "camp-123", Name: "Test Campaign"},
	}
	sessionStore := &fakeSessionStore{
		getSession: sessiondomain.Session{ID: "sess-123", CampaignID: "camp-123", Status: sessiondomain.SessionStatusActive},
	}
	eventStore := &fakeSessionEventStore{}

	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
			Event:    eventStore,
		},
		clock: func() time.Time { return fixedTime },
		seedFunc: func() (int64, error) {
			return 1, nil
		},
	}

	response, err := service.SessionActionRoll(context.Background(), &sessionv1.SessionActionRollRequest{
		CampaignId:  "camp-123",
		SessionId:   "sess-123",
		CharacterId: "char-1",
		Trait:       "bravery",
		Difficulty:  10,
		Modifiers: []*sessionv1.ActionRollModifier{
			{Source: "skill", Value: 2},
		},
	})
	if err != nil {
		t.Fatalf("session action roll: %v", err)
	}
	if response == nil {
		t.Fatal("expected response")
	}
	if response.GetRollSeq() == 0 {
		t.Fatal("expected roll seq")
	}
	if len(eventStore.appendInputs) != 2 {
		t.Fatalf("expected 2 events, got %d", len(eventStore.appendInputs))
	}
	if eventStore.appendInputs[0].Type != sessiondomain.SessionEventTypeActionRollRequested {
		t.Fatalf("expected first event type ACTION_ROLL_REQUESTED, got %s", eventStore.appendInputs[0].Type)
	}
	if eventStore.appendInputs[1].Type != sessiondomain.SessionEventTypeActionRollResolved {
		t.Fatalf("expected second event type ACTION_ROLL_RESOLVED, got %s", eventStore.appendInputs[1].Type)
	}
	if response.GetRollSeq() != eventStore.appendInputs[1].Seq {
		t.Fatalf("expected roll seq %d, got %d", eventStore.appendInputs[1].Seq, response.GetRollSeq())
	}

	var requested actionRollRequestedPayload
	if err := json.Unmarshal(eventStore.appendInputs[0].PayloadJSON, &requested); err != nil {
		t.Fatalf("unmarshal requested payload: %v", err)
	}
	if requested.CharacterID != "char-1" {
		t.Fatalf("expected character id char-1, got %q", requested.CharacterID)
	}
	if requested.Trait != "bravery" {
		t.Fatalf("expected trait bravery, got %q", requested.Trait)
	}
	if requested.Difficulty != 10 {
		t.Fatalf("expected difficulty 10, got %d", requested.Difficulty)
	}
	if len(requested.Modifiers) != 1 || requested.Modifiers[0].Source != "skill" {
		t.Fatalf("expected modifiers to include skill")
	}
}

func TestSessionActionRollRejectsMissingTrait(t *testing.T) {
	campaignStore := &fakeCampaignStore{getCampaign: domain.Campaign{ID: "camp-123"}}
	sessionStore := &fakeSessionStore{
		getSession: sessiondomain.Session{ID: "sess-123", CampaignID: "camp-123", Status: sessiondomain.SessionStatusActive},
	}
	eventStore := &fakeSessionEventStore{}

	service := &SessionService{
		stores: Stores{
			Campaign: campaignStore,
			Session:  sessionStore,
			Event:    eventStore,
		},
		clock:    time.Now,
		seedFunc: func() (int64, error) { return 1, nil },
	}

	_, err := service.SessionActionRoll(context.Background(), &sessionv1.SessionActionRollRequest{
		CampaignId:  "camp-123",
		SessionId:   "sess-123",
		CharacterId: "char-1",
		Trait:       " ",
		Difficulty:  10,
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if len(eventStore.appendInputs) != 1 {
		t.Fatalf("expected 1 event, got %d", len(eventStore.appendInputs))
	}
	if eventStore.appendInputs[0].Type != sessiondomain.SessionEventTypeRequestRejected {
		t.Fatalf("expected REQUEST_REJECTED event, got %s", eventStore.appendInputs[0].Type)
	}
}

// TestApplyRollOutcomeHopeSuccess verifies applying a HOPE outcome updates state.
func TestApplyRollOutcomeHopeSuccess(t *testing.T) {
	fixedTime := time.Date(2026, 1, 26, 12, 0, 0, 0, time.UTC)
	sessionID := "sess-123"
	campaignID := "camp-123"
	rollSeq := uint64(2)

	startedPayload, err := json.Marshal(sessionStartedPayload{CampaignID: campaignID})
	if err != nil {
		t.Fatalf("marshal session started payload: %v", err)
	}
	resolvedPayload, err := json.Marshal(actionRollResolvedPayload{
		RollerCharacterID: "char-1",
		Dice: actionRollResolvedDice{
			HopeDie: 8,
			FearDie: 5,
		},
		Total:      13,
		Difficulty: 10,
		Success:    true,
		Flavor:     "HOPE",
		Crit:       false,
	})
	if err != nil {
		t.Fatalf("marshal action roll resolved payload: %v", err)
	}

	eventStore := &fakeSessionEventStore{
		listEvents: []sessiondomain.SessionEvent{
			{SessionID: sessionID, Seq: 1, Type: sessiondomain.SessionEventTypeSessionStarted, PayloadJSON: startedPayload},
			{SessionID: sessionID, Seq: rollSeq, Type: sessiondomain.SessionEventTypeActionRollResolved, CharacterID: "char-1", PayloadJSON: resolvedPayload},
		},
	}
	sessionStore := &fakeSessionStore{
		getSession: sessiondomain.Session{ID: sessionID, CampaignID: campaignID, Status: sessiondomain.SessionStatusActive},
	}
	participantStore := &fakeParticipantStore{
		getParticipant: domain.Participant{ID: "part-gm", CampaignID: campaignID, Role: domain.ParticipantRoleGM},
	}
	controlStore := &fakeControlDefaultStore{}
	outcomeStore := &fakeRollOutcomeStore{
		applyResult: storage.RollOutcomeApplyResult{
			UpdatedCharacterStates: []domain.CharacterState{{CampaignID: campaignID, CharacterID: "char-1", Hope: 2, Stress: 1, Hp: 5}},
		},
	}

	service := &SessionService{
		stores: Stores{
			Session:        sessionStore,
			Event:          eventStore,
			Outcome:        outcomeStore,
			Participant:    participantStore,
			ControlDefault: controlStore,
		},
		clock: func() time.Time { return fixedTime },
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "part-gm"))
	response, err := service.ApplyRollOutcome(ctx, &sessionv1.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		RollSeq:   rollSeq,
	})
	if err != nil {
		t.Fatalf("apply roll outcome: %v", err)
	}
	if response == nil {
		t.Fatal("expected response")
	}
	if response.GetRequiresComplication() {
		t.Fatal("expected no complication")
	}
	if response.GetRollSeq() != rollSeq {
		t.Fatalf("expected roll seq %d, got %d", rollSeq, response.GetRollSeq())
	}
	if response.Updated == nil || len(response.Updated.GetCharacterStates()) != 1 {
		t.Fatalf("expected updated character state")
	}

	if outcomeStore.applyInput.GMFearDelta != 0 {
		t.Fatalf("expected gm fear delta 0, got %d", outcomeStore.applyInput.GMFearDelta)
	}
	if outcomeStore.applyInput.RequiresComplication {
		t.Fatal("expected requires_complication false")
	}
	if len(outcomeStore.applyInput.CharacterDeltas) != 1 {
		t.Fatalf("expected 1 character delta, got %d", len(outcomeStore.applyInput.CharacterDeltas))
	}
	if outcomeStore.applyInput.CharacterDeltas[0].HopeDelta != 1 {
		t.Fatalf("expected hope delta 1, got %d", outcomeStore.applyInput.CharacterDeltas[0].HopeDelta)
	}

	if len(eventStore.appendInputs) != 1 {
		t.Fatalf("expected 1 event append, got %d", len(eventStore.appendInputs))
	}
	if eventStore.appendInputs[0].Type != sessiondomain.SessionEventTypeOutcomeApplyRequested {
		t.Fatalf("expected OUTCOME_APPLY_REQUESTED event, got %s", eventStore.appendInputs[0].Type)
	}
}

// TestApplyRollOutcomeFearSuccess verifies applying a FEAR outcome updates gm fear.
func TestApplyRollOutcomeFearSuccess(t *testing.T) {
	sessionID := "sess-456"
	campaignID := "camp-456"
	rollSeq := uint64(3)

	startedPayload, err := json.Marshal(sessionStartedPayload{CampaignID: campaignID})
	if err != nil {
		t.Fatalf("marshal session started payload: %v", err)
	}
	resolvedPayload, err := json.Marshal(actionRollResolvedPayload{
		RollerCharacterID: "char-9",
		Dice: actionRollResolvedDice{
			HopeDie: 2,
			FearDie: 9,
		},
		Total:      11,
		Difficulty: 10,
		Success:    true,
		Flavor:     "FEAR",
		Crit:       false,
	})
	if err != nil {
		t.Fatalf("marshal action roll resolved payload: %v", err)
	}

	eventStore := &fakeSessionEventStore{
		listEvents: []sessiondomain.SessionEvent{
			{SessionID: sessionID, Seq: 1, Type: sessiondomain.SessionEventTypeSessionStarted, PayloadJSON: startedPayload},
			{SessionID: sessionID, Seq: rollSeq, Type: sessiondomain.SessionEventTypeActionRollResolved, CharacterID: "char-9", PayloadJSON: resolvedPayload},
		},
	}
	sessionStore := &fakeSessionStore{
		getSession: sessiondomain.Session{ID: sessionID, CampaignID: campaignID, Status: sessiondomain.SessionStatusActive},
	}
	participantStore := &fakeParticipantStore{
		getParticipant: domain.Participant{ID: "part-gm", CampaignID: campaignID, Role: domain.ParticipantRoleGM},
	}
	controlStore := &fakeControlDefaultStore{}
	outcomeStore := &fakeRollOutcomeStore{
		applyResult: storage.RollOutcomeApplyResult{
			UpdatedCharacterStates: []domain.CharacterState{{CampaignID: campaignID, CharacterID: "char-9", Hope: 1, Stress: 2, Hp: 6}},
			GMFearChanged:          true,
			GMFearAfter:            2,
		},
	}

	service := &SessionService{
		stores: Stores{
			Session:        sessionStore,
			Event:          eventStore,
			Outcome:        outcomeStore,
			Participant:    participantStore,
			ControlDefault: controlStore,
		},
		clock: time.Now,
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "part-gm"))
	response, err := service.ApplyRollOutcome(ctx, &sessionv1.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		RollSeq:   rollSeq,
	})
	if err != nil {
		t.Fatalf("apply roll outcome: %v", err)
	}
	if response == nil {
		t.Fatal("expected response")
	}
	if !response.GetRequiresComplication() {
		t.Fatal("expected requires_complication true")
	}
	if response.Updated == nil || response.Updated.GmFear == nil {
		t.Fatal("expected updated gm fear")
	}
	if response.Updated.GetGmFear() != 2 {
		t.Fatalf("expected gm fear 2, got %d", response.Updated.GetGmFear())
	}

	if outcomeStore.applyInput.GMFearDelta != 1 {
		t.Fatalf("expected gm fear delta 1, got %d", outcomeStore.applyInput.GMFearDelta)
	}
	if !outcomeStore.applyInput.RequiresComplication {
		t.Fatal("expected requires_complication true")
	}
	if len(outcomeStore.applyInput.CharacterDeltas) != 0 {
		t.Fatalf("expected no character deltas, got %d", len(outcomeStore.applyInput.CharacterDeltas))
	}
}

// TestApplyRollOutcomeRejectsAlreadyApplied ensures already applied outcomes are rejected.
func TestApplyRollOutcomeRejectsAlreadyApplied(t *testing.T) {
	sessionID := "sess-999"
	campaignID := "camp-999"
	rollSeq := uint64(4)

	startedPayload, err := json.Marshal(sessionStartedPayload{CampaignID: campaignID})
	if err != nil {
		t.Fatalf("marshal session started payload: %v", err)
	}
	resolvedPayload, err := json.Marshal(actionRollResolvedPayload{
		RollerCharacterID: "char-7",
		Dice: actionRollResolvedDice{
			HopeDie: 7,
			FearDie: 7,
		},
		Total:      14,
		Difficulty: 12,
		Success:    true,
		Flavor:     "HOPE",
		Crit:       true,
	})
	if err != nil {
		t.Fatalf("marshal action roll resolved payload: %v", err)
	}

	eventStore := &fakeSessionEventStore{
		listEvents: []sessiondomain.SessionEvent{
			{SessionID: sessionID, Seq: 1, Type: sessiondomain.SessionEventTypeSessionStarted, PayloadJSON: startedPayload},
			{SessionID: sessionID, Seq: rollSeq, Type: sessiondomain.SessionEventTypeActionRollResolved, CharacterID: "char-7", PayloadJSON: resolvedPayload},
		},
	}
	sessionStore := &fakeSessionStore{
		getSession: sessiondomain.Session{ID: sessionID, CampaignID: campaignID, Status: sessiondomain.SessionStatusActive},
	}
	participantStore := &fakeParticipantStore{
		getParticipant: domain.Participant{ID: "part-gm", CampaignID: campaignID, Role: domain.ParticipantRoleGM},
	}
	controlStore := &fakeControlDefaultStore{}
	outcomeStore := &fakeRollOutcomeStore{
		applyErr: sessiondomain.ErrOutcomeAlreadyApplied,
	}

	service := &SessionService{
		stores: Stores{
			Session:        sessionStore,
			Event:          eventStore,
			Outcome:        outcomeStore,
			Participant:    participantStore,
			ControlDefault: controlStore,
		},
		clock: time.Now,
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "part-gm"))
	_, err = service.ApplyRollOutcome(ctx, &sessionv1.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		RollSeq:   rollSeq,
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
	if len(eventStore.appendInputs) != 2 {
		t.Fatalf("expected 2 events, got %d", len(eventStore.appendInputs))
	}
	if eventStore.appendInputs[1].Type != sessiondomain.SessionEventTypeOutcomeRejected {
		t.Fatalf("expected OUTCOME_REJECTED event, got %s", eventStore.appendInputs[1].Type)
	}

	var rejected sessiondomain.OutcomeRejectedPayload
	if err := json.Unmarshal(eventStore.appendInputs[1].PayloadJSON, &rejected); err != nil {
		t.Fatalf("unmarshal outcome rejected payload: %v", err)
	}
	if rejected.ReasonCode != outcomeRejectAlreadyApplied {
		t.Fatalf("expected reason code %s, got %s", outcomeRejectAlreadyApplied, rejected.ReasonCode)
	}
}

// TestApplyRollOutcomeRejectsPlayerTargetMismatch verifies player restrictions.
func TestApplyRollOutcomeRejectsPlayerTargetMismatch(t *testing.T) {
	sessionID := "sess-777"
	campaignID := "camp-777"
	rollSeq := uint64(5)

	startedPayload, err := json.Marshal(sessionStartedPayload{CampaignID: campaignID})
	if err != nil {
		t.Fatalf("marshal session started payload: %v", err)
	}
	resolvedPayload, err := json.Marshal(actionRollResolvedPayload{
		RollerCharacterID: "char-1",
		Dice: actionRollResolvedDice{
			HopeDie: 4,
			FearDie: 2,
		},
		Total:      6,
		Difficulty: 5,
		Success:    true,
		Flavor:     "HOPE",
		Crit:       false,
	})
	if err != nil {
		t.Fatalf("marshal action roll resolved payload: %v", err)
	}

	eventStore := &fakeSessionEventStore{
		listEvents: []sessiondomain.SessionEvent{
			{SessionID: sessionID, Seq: 1, Type: sessiondomain.SessionEventTypeSessionStarted, PayloadJSON: startedPayload},
			{SessionID: sessionID, Seq: rollSeq, Type: sessiondomain.SessionEventTypeActionRollResolved, CharacterID: "char-1", PayloadJSON: resolvedPayload},
		},
	}
	sessionStore := &fakeSessionStore{
		getSession: sessiondomain.Session{ID: sessionID, CampaignID: campaignID, Status: sessiondomain.SessionStatusActive},
	}
	participantStore := &fakeParticipantStore{
		getParticipant: domain.Participant{ID: "part-1", CampaignID: campaignID, Role: domain.ParticipantRolePlayer},
	}
	controlStore := &fakeControlDefaultStore{
		getController: domain.CharacterController{ParticipantID: "part-1"},
	}
	outcomeStore := &fakeRollOutcomeStore{}

	service := &SessionService{
		stores: Stores{
			Session:        sessionStore,
			Event:          eventStore,
			Outcome:        outcomeStore,
			Participant:    participantStore,
			ControlDefault: controlStore,
		},
		clock: time.Now,
	}

	ctx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(grpcmeta.ParticipantIDHeader, "part-1"))
	_, err = service.ApplyRollOutcome(ctx, &sessionv1.ApplyRollOutcomeRequest{
		SessionId: sessionID,
		RollSeq:   rollSeq,
		Targets:   []string{"char-2"},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected grpc status error, got %v", err)
	}
	if st.Code() != codes.PermissionDenied {
		t.Fatalf("expected permission denied, got %v", st.Code())
	}
	if len(eventStore.appendInputs) != 1 {
		t.Fatalf("expected 1 event, got %d", len(eventStore.appendInputs))
	}
	if eventStore.appendInputs[0].Type != sessiondomain.SessionEventTypeOutcomeRejected {
		t.Fatalf("expected OUTCOME_REJECTED event, got %s", eventStore.appendInputs[0].Type)
	}
}
