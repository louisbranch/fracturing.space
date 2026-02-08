package interceptors

import (
	"context"
	"errors"
	"log"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service prefixes for session lock scope.
const (
	participantServicePrefix = "/campaign.v1.ParticipantService/"
	characterServicePrefix   = "/campaign.v1.CharacterService/"
	snapshotServicePrefix    = "/campaign.v1.SnapshotService/"
)

// blockedParticipantMethods lists ParticipantService mutators blocked during an active session.
var blockedParticipantMethods = map[string]struct{}{
	campaignv1.ParticipantService_CreateParticipant_FullMethodName: {},
}

// blockedCharacterMethods lists CharacterService mutators blocked during an active session.
var blockedCharacterMethods = map[string]struct{}{
	campaignv1.CharacterService_CreateCharacter_FullMethodName:       {},
	campaignv1.CharacterService_SetDefaultControl_FullMethodName:     {},
	campaignv1.CharacterService_PatchCharacterProfile_FullMethodName: {},
}

// blockedSnapshotMethods lists SnapshotService mutators blocked during an active session.
var blockedSnapshotMethods = map[string]struct{}{
	campaignv1.SnapshotService_PatchCharacterState_FullMethodName: {},
}

// campaignIDGetter extracts campaign_id from gRPC request messages.
type campaignIDGetter interface {
	GetCampaignId() string
}

// SessionLockInterceptor blocks campaign service mutators when a campaign has an active session.
func SessionLockInterceptor(sessionStore storage.SessionStore) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !isBlockedMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		campaignID := campaignIDFromRequest(req)
		if campaignID == "" {
			return nil, status.Error(codes.InvalidArgument, "campaign id is required")
		}
		if sessionStore == nil {
			return nil, status.Error(codes.Internal, "session store is not configured")
		}

		activeSession, err := sessionStore.GetActiveSession(ctx, campaignID)
		if err == nil {
			logCampaignWriteBlocked(ctx, campaignID, activeSession.ID, info.FullMethod)
			return nil, status.Errorf(
				codes.FailedPrecondition,
				"campaign has an active session: active_session_id=%s",
				activeSession.ID,
			)
		}
		if errors.Is(err, storage.ErrNotFound) {
			return handler(ctx, req)
		}
		return nil, status.Errorf(codes.Internal, "check active session: %v", err)
	}
}

// isBlockedMethod reports whether a method is a mutator blocked during active sessions.
func isBlockedMethod(fullMethod string) bool {
	if strings.HasPrefix(fullMethod, participantServicePrefix) {
		_, blocked := blockedParticipantMethods[fullMethod]
		return blocked
	}
	if strings.HasPrefix(fullMethod, characterServicePrefix) {
		_, blocked := blockedCharacterMethods[fullMethod]
		return blocked
	}
	if strings.HasPrefix(fullMethod, snapshotServicePrefix) {
		_, blocked := blockedSnapshotMethods[fullMethod]
		return blocked
	}
	return false
}

// campaignIDFromRequest extracts the campaign_id field from a request if present.
func campaignIDFromRequest(req any) string {
	getter, ok := req.(campaignIDGetter)
	if !ok {
		return ""
	}
	return strings.TrimSpace(getter.GetCampaignId())
}

// logCampaignWriteBlocked emits a structured log for blocked campaign writes.
func logCampaignWriteBlocked(ctx context.Context, campaignID, activeSessionID, fullMethod string) {
	requestID := grpcmeta.RequestIDFromContext(ctx)
	invocationID := grpcmeta.InvocationIDFromContext(ctx)
	log.Printf(
		"campaign write blocked campaign_id=%s active_session_id=%s rpc_name=%s request_id=%s invocation_id=%s",
		campaignID,
		activeSessionID,
		fullMethod,
		requestID,
		invocationID,
	)
}
