package interceptors

import (
	"context"
	"errors"
	"log"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// blockedMethodCommandTypes maps mutating RPCs to their domain command type so
// transport lock behavior stays aligned with centralized domain policy.
var blockedMethodCommandTypes = map[string]command.Type{
	campaignv1.CampaignService_UpdateCampaign_FullMethodName:         commandids.CampaignUpdate,
	campaignv1.CampaignService_EndCampaign_FullMethodName:            commandids.CampaignEnd,
	campaignv1.CampaignService_ArchiveCampaign_FullMethodName:        commandids.CampaignArchive,
	campaignv1.CampaignService_RestoreCampaign_FullMethodName:        commandids.CampaignRestore,
	campaignv1.CampaignService_SetCampaignCover_FullMethodName:       commandids.CampaignUpdate,
	campaignv1.CampaignService_SetCampaignAIBinding_FullMethodName:   commandids.CampaignAIBind,
	campaignv1.CampaignService_ClearCampaignAIBinding_FullMethodName: commandids.CampaignAIUnbind,

	campaignv1.ParticipantService_CreateParticipant_FullMethodName: commandids.ParticipantJoin,
	campaignv1.ParticipantService_UpdateParticipant_FullMethodName: commandids.ParticipantUpdate,
	campaignv1.ParticipantService_DeleteParticipant_FullMethodName: commandids.ParticipantLeave,

	campaignv1.InviteService_CreateInvite_FullMethodName: commandids.InviteCreate,
	campaignv1.InviteService_ClaimInvite_FullMethodName:  commandids.InviteClaim,

	campaignv1.CharacterService_CreateCharacter_FullMethodName:                commandids.CharacterCreate,
	campaignv1.CharacterService_UpdateCharacter_FullMethodName:                commandids.CharacterUpdate,
	campaignv1.CharacterService_DeleteCharacter_FullMethodName:                commandids.CharacterDelete,
	campaignv1.CharacterService_SetDefaultControl_FullMethodName:              commandids.CharacterUpdate,
	campaignv1.CharacterService_PatchCharacterProfile_FullMethodName:          commandids.CharacterProfileUpdate,
	campaignv1.CharacterService_ApplyCharacterCreationStep_FullMethodName:     commandids.CharacterProfileUpdate,
	campaignv1.CharacterService_ApplyCharacterCreationWorkflow_FullMethodName: commandids.CharacterProfileUpdate,
	campaignv1.CharacterService_ResetCharacterCreationWorkflow_FullMethodName: commandids.CharacterProfileUpdate,

	campaignv1.ForkService_ForkCampaign_FullMethodName: commandids.CampaignFork,
}

// campaignIDGetter extracts campaign_id from gRPC request messages.
type campaignIDGetter interface {
	GetCampaignId() string
}

// sourceCampaignIDGetter extracts source_campaign_id from gRPC request messages.
type sourceCampaignIDGetter interface {
	GetSourceCampaignId() string
}

// SessionLockInterceptor blocks campaign service mutators when a campaign has
// an active session. This enforces turn/state coherence across direct gRPC
// writes that might otherwise bypass session-aware tooling.
func SessionLockInterceptor(sessionStore storage.SessionStore) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		if !isBlockedMethod(info.FullMethod) {
			return handler(ctx, req)
		}

		campaignID := campaignIDFromRequest(req)
		if campaignID == "" {
			return nil, status.Errorf(codes.InvalidArgument, "%s is required", requiredCampaignIDField(info.FullMethod))
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
	cmdType, ok := blockedMethodCommandTypes[fullMethod]
	if !ok {
		return false
	}
	policy, classified := engine.ActiveSessionPolicyForCommandType(cmdType)
	return classified && policy == engine.ActiveSessionCommandPolicyBlocked
}

// campaignIDFromRequest extracts the campaign_id field from a request if present.
func campaignIDFromRequest(req any) string {
	if getter, ok := req.(campaignIDGetter); ok {
		return strings.TrimSpace(getter.GetCampaignId())
	}
	if getter, ok := req.(sourceCampaignIDGetter); ok {
		return strings.TrimSpace(getter.GetSourceCampaignId())
	}
	return ""
}

func requiredCampaignIDField(fullMethod string) string {
	if fullMethod == campaignv1.ForkService_ForkCampaign_FullMethodName {
		return "source_campaign_id"
	}
	return "campaign_id"
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
