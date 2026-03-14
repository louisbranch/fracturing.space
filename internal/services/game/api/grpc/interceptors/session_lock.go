package interceptors

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
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
	campaignv1.CharacterService_ClaimCharacterControl_FullMethodName:          commandids.CharacterUpdate,
	campaignv1.CharacterService_ReleaseCharacterControl_FullMethodName:        commandids.CharacterUpdate,
	campaignv1.CharacterService_PatchCharacterProfile_FullMethodName:          commandids.DaggerheartCharacterProfileReplace,
	campaignv1.CharacterService_ApplyCharacterCreationStep_FullMethodName:     commandids.DaggerheartCharacterProfileReplace,
	campaignv1.CharacterService_ApplyCharacterCreationWorkflow_FullMethodName: commandids.DaggerheartCharacterProfileReplace,
	campaignv1.CharacterService_ResetCharacterCreationWorkflow_FullMethodName: commandids.DaggerheartCharacterProfileDelete,

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
		return nil, grpcerror.Internal("check active session", err)
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

// ValidateSessionLockPolicyCoverage checks that transport-layer blocking and
// domain-layer policy agree. Every command type mapped from an RPC must be
// classified as "blocked" by the domain policy. Every blocked command namespace
// known to the domain must have at least one transport entry.
//
// Call this at startup to catch drift between transport interceptor and domain
// policy when new commands or namespaces are added.
func ValidateSessionLockPolicyCoverage(blockedNamespaces []string) error {
	// Verify every mapped command type is actually "blocked" per domain policy.
	for method, cmdType := range blockedMethodCommandTypes {
		policy, classified := engine.ActiveSessionPolicyForCommandType(cmdType)
		if !classified {
			return fmt.Errorf("session lock interceptor maps %s to unclassified command %s", method, cmdType)
		}
		if policy != engine.ActiveSessionCommandPolicyBlocked {
			return fmt.Errorf("session lock interceptor maps %s to command %s which domain policy classifies as %q, not blocked", method, cmdType, policy)
		}
	}

	// Verify every blocked namespace has at least one transport entry.
	transportNamespaces := make(map[string]bool)
	for _, cmdType := range blockedMethodCommandTypes {
		transportNamespaces[commandNamespaceFromType(cmdType)] = true
	}
	for _, ns := range blockedNamespaces {
		if !transportNamespaces[ns] {
			return fmt.Errorf("domain policy blocks namespace %q but no RPC method maps to it in session lock interceptor", ns)
		}
	}
	return nil
}

// BlockedCommandNamespaces returns the namespaces the domain policy classifies
// as blocked during active sessions. Used by startup validation to ensure
// transport coverage.
func BlockedCommandNamespaces() []string {
	return []string{"campaign", "participant", "invite", "character"}
}

func commandNamespaceFromType(cmdType command.Type) string {
	s := string(cmdType)
	if idx := strings.Index(s, "."); idx > 0 {
		return s[:idx]
	}
	return s
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
