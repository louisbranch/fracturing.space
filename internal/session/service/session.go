package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	sessionv1 "github.com/louisbranch/fracturing.space/api/gen/go/session/v1"
	campaigndomain "github.com/louisbranch/fracturing.space/internal/campaign/domain"
	dualitydomain "github.com/louisbranch/fracturing.space/internal/duality/domain"
	"github.com/louisbranch/fracturing.space/internal/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/id"
	"github.com/louisbranch/fracturing.space/internal/random"
	sessiondomain "github.com/louisbranch/fracturing.space/internal/session/domain"
	"github.com/louisbranch/fracturing.space/internal/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Stores groups all session-related storage interfaces.
type Stores struct {
	Campaign       storage.CampaignStore
	Participant    storage.ParticipantStore
	ControlDefault storage.ControlDefaultStore
	Session        storage.SessionStore
	Event          storage.SessionEventStore
	Outcome        storage.RollOutcomeStore
}

// SessionService implements the SessionService gRPC API.
type SessionService struct {
	sessionv1.UnimplementedSessionServiceServer
	stores      Stores
	clock       func() time.Time
	idGenerator func() (string, error)
	seedFunc    func() (int64, error)
}

// NewSessionService creates a SessionService with default dependencies.
func NewSessionService(stores Stores) *SessionService {
	return &SessionService{
		stores:      stores,
		clock:       time.Now,
		idGenerator: id.NewID,
		seedFunc:    random.NewSeed,
	}
}

// StartSession starts a new session for a campaign.
// Enforces at most one ACTIVE session per campaign.
func (s *SessionService) StartSession(ctx context.Context, in *sessionv1.StartSessionRequest) (*sessionv1.StartSessionResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "start session request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}

	// Validate campaign_id
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	// Check campaign exists
	_, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		return nil, status.Errorf(codes.Internal, "check campaign: %v", err)
	}

	// Check for existing active session
	_, err = s.stores.Session.GetActiveSession(ctx, campaignID)
	if err == nil {
		// Active session exists
		return nil, status.Error(codes.FailedPrecondition, "active session exists")
	}
	if !errors.Is(err, storage.ErrNotFound) {
		// Unexpected error
		return nil, status.Errorf(codes.Internal, "check active session: %v", err)
	}

	// Create session domain object
	session, err := sessiondomain.CreateSession(sessiondomain.CreateSessionInput{
		CampaignID: campaignID,
		Name:       in.GetName(),
	}, s.clock, s.idGenerator)
	if err != nil {
		if errors.Is(err, sessiondomain.ErrEmptyCampaignID) {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		return nil, status.Errorf(codes.Internal, "create session: %v", err)
	}

	// Persist session and set as active (atomic operation)
	if err := s.stores.Session.PutSession(ctx, session); err != nil {
		if errors.Is(err, storage.ErrActiveSessionExists) {
			return nil, status.Error(codes.FailedPrecondition, "active session exists")
		}
		return nil, status.Errorf(codes.Internal, "persist session: %v", err)
	}

	if err := s.appendSessionStartedEvent(ctx, session); err != nil {
		log.Printf("append session started event: %v", err)
	}

	response := &sessionv1.StartSessionResponse{
		Session: &sessionv1.Session{
			Id:         session.ID,
			CampaignId: session.CampaignID,
			Name:       session.Name,
			Status:     sessionStatusToProto(session.Status),
			StartedAt:  timestamppb.New(session.StartedAt),
			UpdatedAt:  timestamppb.New(session.UpdatedAt),
		},
	}
	if session.EndedAt != nil {
		response.Session.EndedAt = timestamppb.New(*session.EndedAt)
	}

	return response, nil
}

const (
	defaultListSessionsPageSize = 10
	maxListSessionsPageSize     = 10
)

// ListSessions returns a page of session records for a campaign.
func (s *SessionService) ListSessions(ctx context.Context, in *sessionv1.ListSessionsRequest) (*sessionv1.ListSessionsResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "list sessions request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}

	// Validate campaign exists
	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}
	_, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		return nil, status.Errorf(codes.Internal, "check campaign: %v", err)
	}

	pageSize := int(in.GetPageSize())
	if pageSize <= 0 {
		pageSize = defaultListSessionsPageSize
	}
	if pageSize > maxListSessionsPageSize {
		pageSize = maxListSessionsPageSize
	}

	page, err := s.stores.Session.ListSessions(ctx, campaignID, pageSize, in.GetPageToken())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list sessions: %v", err)
	}

	response := &sessionv1.ListSessionsResponse{
		NextPageToken: page.NextPageToken,
	}
	if len(page.Sessions) == 0 {
		return response, nil
	}

	response.Sessions = make([]*sessionv1.Session, 0, len(page.Sessions))
	for _, session := range page.Sessions {
		sessionProto := &sessionv1.Session{
			Id:         session.ID,
			CampaignId: session.CampaignID,
			Name:       session.Name,
			Status:     sessionStatusToProto(session.Status),
			StartedAt:  timestamppb.New(session.StartedAt),
			UpdatedAt:  timestamppb.New(session.UpdatedAt),
		}
		if session.EndedAt != nil {
			sessionProto.EndedAt = timestamppb.New(*session.EndedAt)
		}
		response.Sessions = append(response.Sessions, sessionProto)
	}

	return response, nil
}

// GetSession returns a session by campaign ID and session ID.
func (s *SessionService) GetSession(ctx context.Context, in *sessionv1.GetSessionRequest) (*sessionv1.GetSessionResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "get session request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}

	// Validate campaign exists
	_, err := s.stores.Campaign.Get(ctx, campaignID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		return nil, status.Errorf(codes.Internal, "check campaign: %v", err)
	}

	session, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		return nil, status.Errorf(codes.Internal, "get session: %v", err)
	}

	sessionProto := &sessionv1.Session{
		Id:         session.ID,
		CampaignId: session.CampaignID,
		Name:       session.Name,
		Status:     sessionStatusToProto(session.Status),
		StartedAt:  timestamppb.New(session.StartedAt),
		UpdatedAt:  timestamppb.New(session.UpdatedAt),
	}
	if session.EndedAt != nil {
		sessionProto.EndedAt = timestamppb.New(*session.EndedAt)
	}

	response := &sessionv1.GetSessionResponse{
		Session: sessionProto,
	}

	return response, nil
}

// EndSession ends a session by campaign ID and session ID.
func (s *SessionService) EndSession(ctx context.Context, in *sessionv1.EndSessionRequest) (*sessionv1.EndSessionResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "end session request is required")
	}

	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}

	if _, err := s.stores.Campaign.Get(ctx, campaignID); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		return nil, status.Errorf(codes.Internal, "check campaign: %v", err)
	}

	endedAt := s.clock().UTC()
	session, endedNow, err := s.stores.Session.EndSession(ctx, campaignID, sessionID, endedAt)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "session not found")
		}
		return nil, status.Errorf(codes.Internal, "end session: %v", err)
	}

	if endedNow {
		if err := s.appendSessionEndedEvent(ctx, session); err != nil {
			log.Printf("append session ended event: %v", err)
		}
	}

	response := &sessionv1.EndSessionResponse{
		Session: &sessionv1.Session{
			Id:         session.ID,
			CampaignId: session.CampaignID,
			Name:       session.Name,
			Status:     sessionStatusToProto(session.Status),
			StartedAt:  timestamppb.New(session.StartedAt),
			UpdatedAt:  timestamppb.New(session.UpdatedAt),
		},
	}
	if session.EndedAt != nil {
		response.Session.EndedAt = timestamppb.New(*session.EndedAt)
	}

	return response, nil
}

const (
	defaultListSessionEventsLimit = 50
	maxListSessionEventsLimit     = 200
)

// SessionEventAppend appends a session event to the session event stream.
func (s *SessionService) SessionEventAppend(ctx context.Context, in *sessionv1.SessionEventAppendRequest) (*sessionv1.SessionEventAppendResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session event append request is required")
	}

	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "session event store is not configured")
	}

	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}

	eventType, err := eventTypeFromProto(in.GetType())
	if err != nil {
		s.appendRequestRejected(ctx, sessionID, "session.v1.SessionService/SessionEventAppend", "INVALID_ARGUMENT", err.Error(), "")
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	participantID := strings.TrimSpace(in.GetParticipantId())
	if participantID == "" {
		participantID = grpcmeta.ParticipantIDFromContext(ctx)
	}

	characterID := strings.TrimSpace(in.GetCharacterId())
	payload := in.GetPayloadJson()

	event := sessiondomain.SessionEvent{
		SessionID:     sessionID,
		Timestamp:     s.clock().UTC(),
		Type:          eventType,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ParticipantID: participantID,
		CharacterID:   characterID,
		PayloadJSON:   payload,
	}

	stored, err := s.stores.Event.AppendSessionEvent(ctx, event)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append session event: %v", err)
	}

	return &sessionv1.SessionEventAppendResponse{
		Event: sessionEventToProto(stored),
	}, nil
}

// SessionEventsList returns session events ordered by sequence.
func (s *SessionService) SessionEventsList(ctx context.Context, in *sessionv1.SessionEventsListRequest) (*sessionv1.SessionEventsListResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session events list request is required")
	}

	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "session event store is not configured")
	}

	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}

	limit := int(in.GetLimit())
	if limit <= 0 {
		limit = defaultListSessionEventsLimit
	}
	if limit > maxListSessionEventsLimit {
		limit = maxListSessionEventsLimit
	}

	items, err := s.stores.Event.ListSessionEvents(ctx, sessionID, in.GetAfterSeq(), limit)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "list session events: %v", err)
	}

	response := &sessionv1.SessionEventsListResponse{
		Events: make([]*sessionv1.SessionEvent, 0, len(items)),
	}
	for _, event := range items {
		response.Events = append(response.Events, sessionEventToProto(event))
	}

	return response, nil
}

// SessionActionRoll rolls duality dice for a session and appends session events.
func (s *SessionService) SessionActionRoll(ctx context.Context, in *sessionv1.SessionActionRollRequest) (*sessionv1.SessionActionRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session action roll request is required")
	}

	if s.seedFunc == nil {
		return nil, status.Error(codes.Internal, "seed generator is not configured")
	}
	if s.stores.Campaign == nil {
		return nil, status.Error(codes.Internal, "campaign store is not configured")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "session event store is not configured")
	}

	campaignID := strings.TrimSpace(in.GetCampaignId())
	if campaignID == "" {
		return nil, status.Error(codes.InvalidArgument, "campaign id is required")
	}

	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}

	characterID := strings.TrimSpace(in.GetCharacterId())
	if characterID == "" {
		s.appendRequestRejected(ctx, sessionID, "session.v1.SessionService/SessionActionRoll", "INVALID_ARGUMENT", "character id is required", "")
		return nil, status.Error(codes.InvalidArgument, "character id is required")
	}

	trait := strings.TrimSpace(in.GetTrait())
	if trait == "" {
		s.appendRequestRejected(ctx, sessionID, "session.v1.SessionService/SessionActionRoll", "INVALID_ARGUMENT", "trait is required", characterID)
		return nil, status.Error(codes.InvalidArgument, "trait is required")
	}

	if _, err := s.stores.Campaign.Get(ctx, campaignID); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			s.appendRequestRejected(ctx, sessionID, "session.v1.SessionService/SessionActionRoll", "NOT_FOUND", "campaign not found", characterID)
			return nil, status.Error(codes.NotFound, "campaign not found")
		}
		s.appendRequestRejected(ctx, sessionID, "session.v1.SessionService/SessionActionRoll", "INTERNAL", err.Error(), characterID)
		return nil, status.Errorf(codes.Internal, "check campaign: %v", err)
	}

	if _, err := s.stores.Session.GetSession(ctx, campaignID, sessionID); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			s.appendRequestRejected(ctx, sessionID, "session.v1.SessionService/SessionActionRoll", "NOT_FOUND", "session not found", characterID)
			return nil, status.Error(codes.NotFound, "session not found")
		}
		s.appendRequestRejected(ctx, sessionID, "session.v1.SessionService/SessionActionRoll", "INTERNAL", err.Error(), characterID)
		return nil, status.Errorf(codes.Internal, "get session: %v", err)
	}

	modifierTotal := 0
	requestedModifiers := make([]actionRollModifierPayload, 0, len(in.GetModifiers()))
	for _, modifier := range in.GetModifiers() {
		source := strings.TrimSpace(modifier.GetSource())
		if source == "" {
			s.appendRequestRejected(ctx, sessionID, "session.v1.SessionService/SessionActionRoll", "INVALID_ARGUMENT", "modifier source is required", characterID)
			return nil, status.Error(codes.InvalidArgument, "modifier source is required")
		}
		value := int(modifier.GetValue())
		modifierTotal += value
		requestedModifiers = append(requestedModifiers, actionRollModifierPayload{
			Source: source,
			Value:  value,
		})
	}
	sort.Slice(requestedModifiers, func(i, j int) bool {
		if requestedModifiers[i].Source == requestedModifiers[j].Source {
			return requestedModifiers[i].Value < requestedModifiers[j].Value
		}
		return requestedModifiers[i].Source < requestedModifiers[j].Source
	})

	difficulty := int(in.GetDifficulty())
	if difficulty < 0 {
		s.appendRequestRejected(ctx, sessionID, "session.v1.SessionService/SessionActionRoll", "INVALID_ARGUMENT", "difficulty must be zero or greater", characterID)
		return nil, status.Error(codes.InvalidArgument, "difficulty must be zero or greater")
	}

	requestedPayload, err := json.Marshal(actionRollRequestedPayload{
		CharacterID: characterID,
		Trait:       trait,
		Difficulty:  difficulty,
		Modifiers:   requestedModifiers,
	})
	if err != nil {
		s.appendRequestRejected(ctx, sessionID, "session.v1.SessionService/SessionActionRoll", "INTERNAL", "marshal action roll payload", characterID)
		return nil, status.Errorf(codes.Internal, "marshal action roll payload: %v", err)
	}

	if err := s.appendEvent(ctx, sessiondomain.SessionEvent{
		SessionID:     sessionID,
		Timestamp:     s.clock().UTC(),
		Type:          sessiondomain.SessionEventTypeActionRollRequested,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ParticipantID: grpcmeta.ParticipantIDFromContext(ctx),
		CharacterID:   characterID,
		PayloadJSON:   requestedPayload,
	}); err != nil {
		return nil, status.Errorf(codes.Internal, "append action roll requested: %v", err)
	}

	seed, seedSource, rollMode, err := random.ResolveSeed(
		in.GetRng(),
		s.seedFunc,
		func(mode commonv1.RollMode) bool {
			if mode == commonv1.RollMode_REPLAY {
				return true
			}
			if s.stores.Participant == nil {
				return false
			}
			participantID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
			if participantID == "" {
				return false
			}
			participant, err := s.stores.Participant.GetParticipant(ctx, campaignID, participantID)
			if err != nil {
				return false
			}
			return participant.Role == campaigndomain.ParticipantRoleGM
		},
	)
	if err != nil {
		if errors.Is(err, random.ErrSeedOutOfRange()) {
			s.appendRequestRejected(ctx, sessionID, "session.v1.SessionService/SessionActionRoll", "INVALID_ARGUMENT", err.Error(), characterID)
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		s.appendRequestRejected(ctx, sessionID, "session.v1.SessionService/SessionActionRoll", "INTERNAL", "failed to generate seed", characterID)
		return nil, status.Errorf(codes.Internal, "failed to generate seed: %v", err)
	}

	result, err := dualitydomain.RollAction(dualitydomain.ActionRequest{
		Modifier:   modifierTotal,
		Difficulty: &difficulty,
		Seed:       seed,
	})
	if err != nil {
		if errors.Is(err, dualitydomain.ErrInvalidDifficulty) {
			s.appendRequestRejected(ctx, sessionID, "session.v1.SessionService/SessionActionRoll", "INVALID_ARGUMENT", err.Error(), characterID)
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		s.appendRequestRejected(ctx, sessionID, "session.v1.SessionService/SessionActionRoll", "INTERNAL", "failed to roll action", characterID)
		return nil, status.Errorf(codes.Internal, "failed to roll action: %v", err)
	}

	flavor := actionRollFlavor(result.Hope, result.Fear)
	rollModeLabel := "LIVE"
	if rollMode == commonv1.RollMode_REPLAY {
		rollModeLabel = "REPLAY"
	}
	resolvedPayload, err := json.Marshal(actionRollResolvedPayload{
		RollerCharacterID: characterID,
		CharacterID:       characterID,
		Trait:             trait,
		Modifiers:         requestedModifiers,
		SeedUsed:          uint64(seed),
		RngAlgo:           random.RngAlgoMathRandV1,
		SeedSource:        seedSource,
		RollMode:          rollModeLabel,
		Dice: actionRollResolvedDice{
			HopeDie: result.Hope,
			FearDie: result.Fear,
		},
		Total:      result.Total,
		Difficulty: difficulty,
		Success:    result.MeetsDifficulty,
		Flavor:     flavor,
		Crit:       result.IsCrit,
	})
	if err != nil {
		s.appendRequestRejected(ctx, sessionID, "session.v1.SessionService/SessionActionRoll", "INTERNAL", "marshal action roll resolved payload", characterID)
		return nil, status.Errorf(codes.Internal, "marshal action roll resolved payload: %v", err)
	}

	resolvedEvent, err := s.stores.Event.AppendSessionEvent(ctx, sessiondomain.SessionEvent{
		SessionID:     sessionID,
		Timestamp:     s.clock().UTC(),
		Type:          sessiondomain.SessionEventTypeActionRollResolved,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ParticipantID: grpcmeta.ParticipantIDFromContext(ctx),
		CharacterID:   characterID,
		PayloadJSON:   resolvedPayload,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "append action roll resolved: %v", err)
	}

	return &sessionv1.SessionActionRollResponse{
		RollSeq:    resolvedEvent.Seq,
		HopeDie:    int32(result.Hope),
		FearDie:    int32(result.Fear),
		Total:      int32(result.Total),
		Difficulty: int32(difficulty),
		Success:    result.MeetsDifficulty,
		Flavor:     flavor,
		Crit:       result.IsCrit,
		Rng: &commonv1.RngResponse{
			SeedUsed:   uint64(seed),
			RngAlgo:    random.RngAlgoMathRandV1,
			SeedSource: seedSource,
			RollMode:   rollMode,
		},
	}, nil
}

const (
	// outcomeRejectSessionNotActive indicates the session is not active.
	outcomeRejectSessionNotActive = "SESSION_NOT_ACTIVE"
	// outcomeRejectRollNotFound indicates the roll event was not found.
	outcomeRejectRollNotFound = "ROLL_NOT_FOUND"
	// outcomeRejectRollWrongType indicates the roll event was not resolved.
	outcomeRejectRollWrongType = "ROLL_WRONG_TYPE"
	// outcomeRejectAlreadyApplied indicates the roll outcome was already applied.
	outcomeRejectAlreadyApplied = "OUTCOME_ALREADY_APPLIED"
	// outcomeRejectPermissionDenied indicates the caller cannot apply the outcome.
	outcomeRejectPermissionDenied = "PERMISSION_DENIED"
	// outcomeRejectCharacterNotFound indicates a target character was not found.
	outcomeRejectCharacterNotFound = "CHARACTER_NOT_FOUND"
	// outcomeRejectMultiTargetUnsupported indicates multiple targets are not supported.
	outcomeRejectMultiTargetUnsupported = "MULTI_TARGET_UNSUPPORTED"
	// outcomeRejectInternalError indicates an unexpected state error occurred.
	outcomeRejectInternalError = "INTERNAL_ERROR"
)

// ApplyRollOutcome applies the mandatory outcome effects for a resolved action roll.
func (s *SessionService) ApplyRollOutcome(ctx context.Context, in *sessionv1.ApplyRollOutcomeRequest) (*sessionv1.ApplyRollOutcomeResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "apply roll outcome request is required")
	}
	if s.stores.Session == nil {
		return nil, status.Error(codes.Internal, "session store is not configured")
	}
	if s.stores.Event == nil {
		return nil, status.Error(codes.Internal, "session event store is not configured")
	}
	if s.stores.Outcome == nil {
		return nil, status.Error(codes.Internal, "roll outcome store is not configured")
	}
	if s.stores.Participant == nil {
		return nil, status.Error(codes.Internal, "participant store is not configured")
	}
	if s.stores.ControlDefault == nil {
		return nil, status.Error(codes.Internal, "control default store is not configured")
	}

	sessionID := strings.TrimSpace(in.GetSessionId())
	if sessionID == "" {
		return nil, status.Error(codes.InvalidArgument, "session id is required")
	}

	rollSeq := in.GetRollSeq()
	if rollSeq == 0 {
		s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectRollNotFound, "roll seq is required", "")
		return nil, status.Error(codes.InvalidArgument, "roll seq is required")
	}

	rollEvent, err := s.sessionEventBySeq(ctx, sessionID, rollSeq)
	if err != nil {
		s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectRollNotFound, err.Error(), "")
		return nil, status.Error(codes.NotFound, "roll event not found")
	}
	if rollEvent.Type != sessiondomain.SessionEventTypeActionRollResolved {
		s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectRollWrongType, "roll event is not resolved", rollEvent.CharacterID)
		return nil, status.Error(codes.FailedPrecondition, "roll event is not resolved")
	}

	var resolved actionRollResolvedPayload
	if err := json.Unmarshal(rollEvent.PayloadJSON, &resolved); err != nil {
		s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectInternalError, "invalid roll payload", rollEvent.CharacterID)
		return nil, status.Error(codes.Internal, "invalid roll payload")
	}
	rollerID := strings.TrimSpace(resolved.RollerCharacterID)
	if rollerID == "" {
		s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectInternalError, "roll payload missing character id", rollEvent.CharacterID)
		return nil, status.Error(codes.Internal, "roll payload missing character id")
	}

	targets := normalizeOutcomeTargets(in.GetTargets(), rollerID)
	if len(targets) == 0 {
		s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectInternalError, "targets are required", rollerID)
		return nil, status.Error(codes.InvalidArgument, "targets are required")
	}
	if len(targets) > 1 {
		s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectMultiTargetUnsupported, "multi target unsupported", rollerID)
		return nil, status.Error(codes.FailedPrecondition, "multi target unsupported")
	}

	campaignID, err := s.sessionCampaignID(ctx, sessionID)
	if err != nil {
		s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectInternalError, "session campaign not found", rollerID)
		return nil, status.Error(codes.Internal, "session campaign not found")
	}

	session, err := s.stores.Session.GetSession(ctx, campaignID, sessionID)
	if err != nil {
		s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectSessionNotActive, "session not found", rollerID)
		return nil, status.Error(codes.NotFound, "session not found")
	}
	if session.Status != sessiondomain.SessionStatusActive {
		s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectSessionNotActive, "session is not active", rollerID)
		return nil, status.Error(codes.FailedPrecondition, "session is not active")
	}

	participantID := strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	if participantID == "" {
		s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectPermissionDenied, "participant id is required", rollerID)
		return nil, status.Error(codes.PermissionDenied, "participant id is required")
	}
	participant, err := s.stores.Participant.GetParticipant(ctx, campaignID, participantID)
	if err != nil {
		s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectPermissionDenied, "participant not found", rollerID)
		return nil, status.Error(codes.PermissionDenied, "participant not found")
	}
	if participant.Role != campaigndomain.ParticipantRoleGM {
		if targets[0] != rollerID {
			s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectPermissionDenied, "player may only apply to roller", rollerID)
			return nil, status.Error(codes.PermissionDenied, "player may only apply to roller")
		}
		controller, err := s.stores.ControlDefault.GetControlDefault(ctx, campaignID, rollerID)
		if err != nil {
			s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectPermissionDenied, "character controller not found", rollerID)
			return nil, status.Error(codes.PermissionDenied, "character controller not found")
		}
		if controller.IsGM || controller.ParticipantID != participantID {
			s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectPermissionDenied, "participant does not control roller", rollerID)
			return nil, status.Error(codes.PermissionDenied, "participant does not control roller")
		}
	}

	if err := s.appendOutcomeApplyRequested(ctx, sessionID, rollSeq, targets, rollerID); err != nil {
		return nil, status.Errorf(codes.Internal, "append outcome apply requested: %v", err)
	}

	characterDeltas := make([]storage.RollOutcomeDelta, 0, len(targets))
	gmFearDelta := 0
	requiresComplication := false

	switch {
	case resolved.Crit:
		for _, target := range targets {
			characterDeltas = append(characterDeltas, storage.RollOutcomeDelta{
				CharacterID: target,
				HopeDelta:   1,
				StressDelta: -1,
			})
		}
	case strings.EqualFold(resolved.Flavor, "HOPE"):
		for _, target := range targets {
			characterDeltas = append(characterDeltas, storage.RollOutcomeDelta{
				CharacterID: target,
				HopeDelta:   1,
			})
		}
	case strings.EqualFold(resolved.Flavor, "FEAR"):
		gmFearDelta = 1
		requiresComplication = true
	default:
		s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectInternalError, "invalid roll flavor", rollerID)
		return nil, status.Error(codes.Internal, "invalid roll flavor")
	}

	applyResult, err := s.stores.Outcome.ApplyRollOutcome(ctx, storage.RollOutcomeApplyInput{
		CampaignID:           campaignID,
		SessionID:            sessionID,
		RollSeq:              rollSeq,
		Targets:              targets,
		RequiresComplication: requiresComplication,
		RequestID:            grpcmeta.RequestIDFromContext(ctx),
		InvocationID:         grpcmeta.InvocationIDFromContext(ctx),
		ParticipantID:        participantID,
		CharacterID:          rollerID,
		EventTimestamp:       s.clock().UTC(),
		CharacterDeltas:      characterDeltas,
		GMFearDelta:          gmFearDelta,
	})
	if err != nil {
		switch {
		case errors.Is(err, sessiondomain.ErrOutcomeAlreadyApplied):
			s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectAlreadyApplied, "outcome already applied", rollerID)
			return nil, status.Error(codes.FailedPrecondition, "outcome already applied")
		case errors.Is(err, sessiondomain.ErrOutcomeCharacterNotFound):
			s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectCharacterNotFound, "character not found", rollerID)
			return nil, status.Error(codes.NotFound, "character not found")
		case errors.Is(err, sessiondomain.ErrOutcomeGMFearInvalid):
			s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectInternalError, "gm fear update invalid", rollerID)
			return nil, status.Error(codes.FailedPrecondition, "gm fear update invalid")
		default:
			s.appendOutcomeRejected(ctx, sessionID, rollSeq, outcomeRejectInternalError, err.Error(), rollerID)
			return nil, status.Errorf(codes.Internal, "apply roll outcome: %v", err)
		}
	}

	updated := &sessionv1.OutcomeUpdated{
		CharacterStates: make([]*sessionv1.OutcomeCharacterState, 0, len(applyResult.UpdatedCharacterStates)),
	}
	for _, state := range applyResult.UpdatedCharacterStates {
		updated.CharacterStates = append(updated.CharacterStates, &sessionv1.OutcomeCharacterState{
			CharacterId: state.CharacterID,
			Hope:        int32(state.Hope),
			Stress:      int32(state.Stress),
			Hp:          int32(state.Hp),
		})
	}
	if applyResult.GMFearChanged {
		gmFear := int32(applyResult.GMFearAfter)
		updated.GmFear = &gmFear
	}

	return &sessionv1.ApplyRollOutcomeResponse{
		RollSeq:              rollSeq,
		RequiresComplication: requiresComplication,
		Updated:              updated,
	}, nil
}

// sessionStatusToProto maps a domain session status to the protobuf representation.
func sessionStatusToProto(status sessiondomain.SessionStatus) sessionv1.SessionStatus {
	switch status {
	case sessiondomain.SessionStatusActive:
		return sessionv1.SessionStatus_ACTIVE
	case sessiondomain.SessionStatusEnded:
		return sessionv1.SessionStatus_ENDED
	default:
		return sessionv1.SessionStatus_STATUS_UNSPECIFIED
	}
}

type actionRollModifierPayload struct {
	Source string `json:"source"`
	Value  int    `json:"value"`
}

type actionRollRequestedPayload struct {
	CharacterID string                      `json:"character_id"`
	Trait       string                      `json:"trait"`
	Difficulty  int                         `json:"difficulty"`
	Modifiers   []actionRollModifierPayload `json:"modifiers,omitempty"`
}

// actionRollResolvedPayload captures the payload for resolved action roll events.
type actionRollResolvedPayload struct {
	// RollerCharacterID identifies the character that rolled.
	RollerCharacterID string `json:"roller_character_id"`
	// CharacterID echoes the request character identifier.
	CharacterID string `json:"character_id"`
	// Trait echoes the canonicalized trait name.
	Trait string `json:"trait"`
	// Modifiers echoes canonicalized modifiers used for the roll.
	Modifiers []actionRollModifierPayload `json:"modifiers"`
	// SeedUsed is the RNG seed used for this roll.
	SeedUsed uint64 `json:"seed_used"`
	// RngAlgo identifies the RNG algorithm used.
	RngAlgo string `json:"rng_algo"`
	// SeedSource indicates whether the seed was client- or server-supplied.
	SeedSource string `json:"seed_source"`
	// RollMode records whether the roll was live or replayed.
	RollMode string `json:"roll_mode"`
	// Dice captures the raw dice results.
	Dice actionRollResolvedDice `json:"dice"`
	// Total is the sum of dice and modifiers.
	Total int `json:"total"`
	// Difficulty is the target threshold for success.
	Difficulty int `json:"difficulty"`
	// Success indicates whether the roll met the difficulty.
	Success bool `json:"success"`
	// Flavor indicates whether the roll favored hope or fear.
	Flavor string `json:"flavor"`
	// Crit indicates whether the roll is a critical success.
	Crit bool `json:"crit"`
}

type actionRollResolvedDice struct {
	HopeDie int `json:"hope_die"`
	FearDie int `json:"fear_die"`
}

type sessionStartedPayload struct {
	CampaignID string `json:"campaign_id"`
}

// sessionEndedPayload captures the event payload for ended sessions.
type sessionEndedPayload struct {
	CampaignID string `json:"campaign_id"`
	EndedAt    string `json:"ended_at"`
}

type requestRejectedPayload struct {
	RPC        string `json:"rpc"`
	ReasonCode string `json:"reason_code"`
	Message    string `json:"message,omitempty"`
}

func (s *SessionService) appendSessionStartedEvent(ctx context.Context, session sessiondomain.Session) error {
	if s.stores.Event == nil {
		return fmt.Errorf("session event store is not configured")
	}

	payload, err := json.Marshal(sessionStartedPayload{CampaignID: session.CampaignID})
	if err != nil {
		return fmt.Errorf("marshal session started payload: %w", err)
	}

	_, err = s.stores.Event.AppendSessionEvent(ctx, sessiondomain.SessionEvent{
		SessionID:     session.ID,
		Timestamp:     s.clock().UTC(),
		Type:          sessiondomain.SessionEventTypeSessionStarted,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ParticipantID: grpcmeta.ParticipantIDFromContext(ctx),
		PayloadJSON:   payload,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *SessionService) appendSessionEndedEvent(ctx context.Context, session sessiondomain.Session) error {
	if s.stores.Event == nil {
		return fmt.Errorf("session event store is not configured")
	}
	if session.EndedAt == nil {
		return fmt.Errorf("session ended_at is required")
	}

	payload, err := json.Marshal(sessionEndedPayload{
		CampaignID: session.CampaignID,
		EndedAt:    session.EndedAt.UTC().Format(time.RFC3339Nano),
	})
	if err != nil {
		return fmt.Errorf("marshal session ended payload: %w", err)
	}

	_, err = s.stores.Event.AppendSessionEvent(ctx, sessiondomain.SessionEvent{
		SessionID:     session.ID,
		Timestamp:     s.clock().UTC(),
		Type:          sessiondomain.SessionEventTypeSessionEnded,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ParticipantID: grpcmeta.ParticipantIDFromContext(ctx),
		PayloadJSON:   payload,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *SessionService) appendRequestRejected(ctx context.Context, sessionID, rpc, reasonCode, message, characterID string) {
	if s == nil || s.stores.Event == nil {
		return
	}

	payload, err := json.Marshal(requestRejectedPayload{
		RPC:        rpc,
		ReasonCode: reasonCode,
		Message:    message,
	})
	if err != nil {
		return
	}

	_ = s.appendEvent(ctx, sessiondomain.SessionEvent{
		SessionID:     sessionID,
		Timestamp:     s.clock().UTC(),
		Type:          sessiondomain.SessionEventTypeRequestRejected,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ParticipantID: grpcmeta.ParticipantIDFromContext(ctx),
		CharacterID:   characterID,
		PayloadJSON:   payload,
	})
}

func (s *SessionService) appendEvent(ctx context.Context, event sessiondomain.SessionEvent) error {
	if s.stores.Event == nil {
		return fmt.Errorf("session event store is not configured")
	}
	_, err := s.stores.Event.AppendSessionEvent(ctx, event)
	return err
}

func eventTypeFromProto(eventType sessionv1.SessionEventType) (sessiondomain.SessionEventType, error) {
	switch eventType {
	case sessionv1.SessionEventType_SESSION_STARTED:
		return sessiondomain.SessionEventTypeSessionStarted, nil
	case sessionv1.SessionEventType_SESSION_ENDED:
		return sessiondomain.SessionEventTypeSessionEnded, nil
	case sessionv1.SessionEventType_NOTE_ADDED:
		return sessiondomain.SessionEventTypeNoteAdded, nil
	case sessionv1.SessionEventType_ACTION_ROLL_REQUESTED:
		return sessiondomain.SessionEventTypeActionRollRequested, nil
	case sessionv1.SessionEventType_ACTION_ROLL_RESOLVED:
		return sessiondomain.SessionEventTypeActionRollResolved, nil
	case sessionv1.SessionEventType_OUTCOME_APPLY_REQUESTED:
		return sessiondomain.SessionEventTypeOutcomeApplyRequested, nil
	case sessionv1.SessionEventType_OUTCOME_APPLIED:
		return sessiondomain.SessionEventTypeOutcomeApplied, nil
	case sessionv1.SessionEventType_OUTCOME_REJECTED:
		return sessiondomain.SessionEventTypeOutcomeRejected, nil
	case sessionv1.SessionEventType_REQUEST_REJECTED:
		return sessiondomain.SessionEventTypeRequestRejected, nil
	default:
		return "", fmt.Errorf("event type is required")
	}
}

func eventTypeToProto(eventType sessiondomain.SessionEventType) sessionv1.SessionEventType {
	switch eventType {
	case sessiondomain.SessionEventTypeSessionStarted:
		return sessionv1.SessionEventType_SESSION_STARTED
	case sessiondomain.SessionEventTypeSessionEnded:
		return sessionv1.SessionEventType_SESSION_ENDED
	case sessiondomain.SessionEventTypeNoteAdded:
		return sessionv1.SessionEventType_NOTE_ADDED
	case sessiondomain.SessionEventTypeActionRollRequested:
		return sessionv1.SessionEventType_ACTION_ROLL_REQUESTED
	case sessiondomain.SessionEventTypeActionRollResolved:
		return sessionv1.SessionEventType_ACTION_ROLL_RESOLVED
	case sessiondomain.SessionEventTypeOutcomeApplyRequested:
		return sessionv1.SessionEventType_OUTCOME_APPLY_REQUESTED
	case sessiondomain.SessionEventTypeOutcomeApplied:
		return sessionv1.SessionEventType_OUTCOME_APPLIED
	case sessiondomain.SessionEventTypeOutcomeRejected:
		return sessionv1.SessionEventType_OUTCOME_REJECTED
	case sessiondomain.SessionEventTypeRequestRejected:
		return sessionv1.SessionEventType_REQUEST_REJECTED
	default:
		return sessionv1.SessionEventType_SESSION_EVENT_TYPE_UNSPECIFIED
	}
}

func sessionEventToProto(event sessiondomain.SessionEvent) *sessionv1.SessionEvent {
	return &sessionv1.SessionEvent{
		SessionId:     event.SessionID,
		Seq:           event.Seq,
		Ts:            timestamppb.New(event.Timestamp),
		Type:          eventTypeToProto(event.Type),
		RequestId:     event.RequestID,
		InvocationId:  event.InvocationID,
		ParticipantId: event.ParticipantID,
		CharacterId:   event.CharacterID,
		PayloadJson:   event.PayloadJSON,
	}
}

func actionRollFlavor(hope, fear int) string {
	if fear > hope {
		return "FEAR"
	}
	return "HOPE"
}

// normalizeOutcomeTargets resolves targets for an outcome apply request.
func normalizeOutcomeTargets(targets []string, defaultTarget string) []string {
	trimmed := make([]string, 0, len(targets))
	seen := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		value := strings.TrimSpace(target)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		trimmed = append(trimmed, value)
	}
	if len(trimmed) == 0 && strings.TrimSpace(defaultTarget) != "" {
		return []string{strings.TrimSpace(defaultTarget)}
	}
	return trimmed
}

// sessionEventBySeq loads a single session event by sequence.
func (s *SessionService) sessionEventBySeq(ctx context.Context, sessionID string, seq uint64) (sessiondomain.SessionEvent, error) {
	if s == nil || s.stores.Event == nil {
		return sessiondomain.SessionEvent{}, fmt.Errorf("session event store is not configured")
	}
	items, err := s.stores.Event.ListSessionEvents(ctx, sessionID, seq-1, 1)
	if err != nil {
		return sessiondomain.SessionEvent{}, err
	}
	if len(items) == 0 || items[0].Seq != seq {
		return sessiondomain.SessionEvent{}, storage.ErrNotFound
	}
	return items[0], nil
}

// sessionCampaignID resolves the campaign ID for a session via the session started event.
func (s *SessionService) sessionCampaignID(ctx context.Context, sessionID string) (string, error) {
	if s == nil || s.stores.Event == nil {
		return "", fmt.Errorf("session event store is not configured")
	}
	events, err := s.stores.Event.ListSessionEvents(ctx, sessionID, 0, 1)
	if err != nil {
		return "", err
	}
	if len(events) == 0 {
		return "", storage.ErrNotFound
	}
	if events[0].Type != sessiondomain.SessionEventTypeSessionStarted {
		return "", fmt.Errorf("session started event not found")
	}
	var payload sessionStartedPayload
	if err := json.Unmarshal(events[0].PayloadJSON, &payload); err != nil {
		return "", fmt.Errorf("unmarshal session started payload: %w", err)
	}
	campaignID := strings.TrimSpace(payload.CampaignID)
	if campaignID == "" {
		return "", fmt.Errorf("session campaign id is required")
	}
	return campaignID, nil
}

// appendOutcomeApplyRequested appends an outcome apply requested event.
func (s *SessionService) appendOutcomeApplyRequested(ctx context.Context, sessionID string, rollSeq uint64, targets []string, characterID string) error {
	payload, err := json.Marshal(sessiondomain.OutcomeApplyRequestedPayload{
		RollSeq: rollSeq,
		Targets: targets,
	})
	if err != nil {
		return fmt.Errorf("marshal outcome apply requested payload: %w", err)
	}

	return s.appendEvent(ctx, sessiondomain.SessionEvent{
		SessionID:     sessionID,
		Timestamp:     s.clock().UTC(),
		Type:          sessiondomain.SessionEventTypeOutcomeApplyRequested,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ParticipantID: grpcmeta.ParticipantIDFromContext(ctx),
		CharacterID:   characterID,
		PayloadJSON:   payload,
	})
}

// appendOutcomeRejected appends an outcome rejected event.
func (s *SessionService) appendOutcomeRejected(ctx context.Context, sessionID string, rollSeq uint64, reasonCode, message, characterID string) {
	if s == nil || s.stores.Event == nil {
		return
	}
	if strings.TrimSpace(sessionID) == "" {
		return
	}

	payload, err := json.Marshal(sessiondomain.OutcomeRejectedPayload{
		RollSeq:    rollSeq,
		ReasonCode: reasonCode,
		Message:    message,
	})
	if err != nil {
		return
	}

	_ = s.appendEvent(ctx, sessiondomain.SessionEvent{
		SessionID:     sessionID,
		Timestamp:     s.clock().UTC(),
		Type:          sessiondomain.SessionEventTypeOutcomeRejected,
		RequestID:     grpcmeta.RequestIDFromContext(ctx),
		InvocationID:  grpcmeta.InvocationIDFromContext(ctx),
		ParticipantID: grpcmeta.ParticipantIDFromContext(ctx),
		CharacterID:   characterID,
		PayloadJSON:   payload,
	})
}
