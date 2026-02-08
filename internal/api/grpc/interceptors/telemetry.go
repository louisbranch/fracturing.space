package interceptors

import (
	"context"
	"log"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/storage"
	"github.com/louisbranch/fracturing.space/internal/telemetry"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// TelemetryInterceptor emits telemetry for read-only gRPC methods.
func TelemetryInterceptor(store storage.TelemetryStore) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if store == nil || !isReadMethod(info.FullMethod) {
			return resp, err
		}

		severity := telemetry.SeverityInfo
		code := codes.OK
		if err != nil {
			severity = telemetry.SeverityError
			if st, ok := status.FromError(err); ok {
				code = st.Code()
			}
		}

		campaignID, sessionID := extractScope(req)
		actorID := grpcmeta.ParticipantIDFromContext(ctx)
		actorType := "system"
		if actorID != "" {
			actorType = "participant"
		}

		emitter := telemetry.NewEmitter(store)
		emitErr := emitter.Emit(ctx, storage.TelemetryEvent{
			EventName:    "telemetry.grpc.read",
			Severity:     string(severity),
			CampaignID:   campaignID,
			SessionID:    sessionID,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			Attributes: map[string]any{
				"method": info.FullMethod,
				"code":   code.String(),
			},
		})
		if emitErr != nil {
			log.Printf("telemetry emit %s: %v", info.FullMethod, emitErr)
		}

		return resp, err
	}
}

type telemetryCampaignIDGetter interface {
	GetCampaignId() string
}

type telemetrySessionIDGetter interface {
	GetSessionId() string
}

type telemetrySourceCampaignIDGetter interface {
	GetSourceCampaignId() string
}

func extractScope(req any) (string, string) {
	if req == nil {
		return "", ""
	}
	if getter, ok := req.(telemetryCampaignIDGetter); ok {
		return strings.TrimSpace(getter.GetCampaignId()), sessionIDFromRequest(req)
	}
	if getter, ok := req.(telemetrySourceCampaignIDGetter); ok {
		return strings.TrimSpace(getter.GetSourceCampaignId()), sessionIDFromRequest(req)
	}
	return "", sessionIDFromRequest(req)
}

func sessionIDFromRequest(req any) string {
	getter, ok := req.(telemetrySessionIDGetter)
	if !ok {
		return ""
	}
	return strings.TrimSpace(getter.GetSessionId())
}

func isReadMethod(fullMethod string) bool {
	switch fullMethod {
	case campaignv1.CampaignService_ListCampaigns_FullMethodName,
		campaignv1.CampaignService_GetCampaign_FullMethodName,
		campaignv1.ParticipantService_ListParticipants_FullMethodName,
		campaignv1.ParticipantService_GetParticipant_FullMethodName,
		campaignv1.CharacterService_ListCharacters_FullMethodName,
		campaignv1.CharacterService_GetCharacterSheet_FullMethodName,
		campaignv1.SessionService_ListSessions_FullMethodName,
		campaignv1.SessionService_GetSession_FullMethodName,
		campaignv1.SnapshotService_GetSnapshot_FullMethodName,
		campaignv1.EventService_ListEvents_FullMethodName,
		campaignv1.ForkService_GetLineage_FullMethodName,
		campaignv1.ForkService_ListForks_FullMethodName:
		return true
	default:
		return false
	}
}
