package interceptors

import (
	"context"
	"log"
	"strings"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/observability/audit"
	"github.com/louisbranch/fracturing.space/internal/services/game/observability/audit/events"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuditInterceptor emits an audit event for each unary gRPC call handled by the game service.
//
// All unary calls are captured to make cross-service telemetry coverage explicit
// while preserving existing read/write classification in event attributes.
func AuditInterceptor(store storage.AuditEventStore) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		resp, err := handler(ctx, req)
		if store == nil {
			return resp, err
		}

		methodType := classifyMethodKind(info.FullMethod)
		eventName := events.GRPCWrite
		if methodType == "read" {
			eventName = events.GRPCRead
		}

		severity := audit.SeverityInfo
		code := codes.OK
		if err != nil {
			severity = audit.SeverityError
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

		var traceID, spanID string
		if sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
			traceID = sc.TraceID().String()
			spanID = sc.SpanID().String()
		}

		emitter := audit.NewEmitter(store)
		emitErr := emitter.Emit(ctx, storage.AuditEvent{
			EventName:    eventName,
			Severity:     string(severity),
			CampaignID:   campaignID,
			SessionID:    sessionID,
			ActorType:    actorType,
			ActorID:      actorID,
			RequestID:    grpcmeta.RequestIDFromContext(ctx),
			InvocationID: grpcmeta.InvocationIDFromContext(ctx),
			TraceID:      traceID,
			SpanID:       spanID,
			Attributes: map[string]any{
				"method":      info.FullMethod,
				"method_kind": methodType,
				"code":        code.String(),
			},
		})
		if emitErr != nil {
			log.Printf("audit emit %s: %v", info.FullMethod, emitErr)
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

func classifyMethodKind(fullMethod string) string {
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
		return "read"
	default:
		return "write"
	}
}
