package service

import (
	"context"
	"errors"
	"log"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
	"github.com/louisbranch/fracturing.space/internal/grpcmeta"
	"github.com/louisbranch/fracturing.space/internal/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// campaignServicePrefix scopes session locks to CampaignService RPCs.
const campaignServicePrefix = "/campaign.v1.CampaignService/"

// blockedCampaignMethods lists CampaignService mutators that are blocked during an active session.
var blockedCampaignMethods = map[string]struct{}{
	campaignv1.CampaignService_CreateParticipant_FullMethodName:     {},
	campaignv1.CampaignService_CreateCharacter_FullMethodName:       {},
	campaignv1.CampaignService_SetDefaultControl_FullMethodName:     {},
	campaignv1.CampaignService_PatchCharacterProfile_FullMethodName: {},
	campaignv1.CampaignService_PatchCharacterState_FullMethodName:   {},
}

// campaignIDGetter extracts campaign_id from gRPC request messages.
type campaignIDGetter interface {
	GetCampaignId() string
}

// SessionLockInterceptor blocks CampaignService mutators when a campaign has an active session.
func SessionLockInterceptor(sessionStore storage.SessionStore) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !isCampaignServiceMethod(info.FullMethod) {
			return handler(ctx, req)
		}
		if !isBlockedCampaignMethod(info.FullMethod) {
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
			// TODO: Append REQUEST_REJECTED session event when session events are implemented.
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

// isCampaignServiceMethod reports whether a method belongs to CampaignService.
func isCampaignServiceMethod(fullMethod string) bool {
	return strings.HasPrefix(fullMethod, campaignServicePrefix)
}

// isBlockedCampaignMethod reports whether a CampaignService method is a mutator.
func isBlockedCampaignMethod(fullMethod string) bool {
	_, blocked := blockedCampaignMethods[fullMethod]
	return blocked
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
