package game

import (
	"context"
	"log"
	"strings"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/observability/audit"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func emitAuthzDecisionTelemetry(
	ctx context.Context,
	store storage.AuditEventStore,
	campaignID string,
	capability domainauthz.Capability,
	decision string,
	reasonCode string,
	actor storage.ParticipantRecord,
	authErr error,
	extraAttributes map[string]any,
) {
	severity := audit.SeverityInfo
	code := codes.OK
	if authErr != nil {
		severity = audit.SeverityWarn
		if st, ok := status.FromError(authErr); ok {
			code = st.Code()
		}
		if code == codes.Internal {
			severity = audit.SeverityError
		}
	}

	actorID := strings.TrimSpace(actor.ID)
	if actorID == "" {
		actorID = strings.TrimSpace(grpcmeta.ParticipantIDFromContext(ctx))
	}
	actorType := "system"
	if actorID != "" {
		actorType = "participant"
	}

	var traceID, spanID string
	if sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
		traceID = sc.TraceID().String()
		spanID = sc.SpanID().String()
	}

	attributes := map[string]any{
		"decision":      decision,
		"reason_code":   reasonCode,
		"policy_action": policyCapabilityLabel(capability),
		"grpc_code":     code.String(),
	}
	if access := strings.TrimSpace(string(actor.CampaignAccess)); access != "" {
		attributes["campaign_access"] = access
	}
	if userID := strings.TrimSpace(actor.UserID); userID != "" {
		attributes["actor_user_id"] = userID
	}
	for key, value := range extraAttributes {
		attributes[key] = value
	}

	emitter := audit.NewEmitter(store)
	if err := emitter.Emit(ctx, storage.AuditEvent{
		EventName:    authzEventDecisionName,
		Severity:     string(severity),
		CampaignID:   strings.TrimSpace(campaignID),
		ActorType:    actorType,
		ActorID:      actorID,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		TraceID:      traceID,
		SpanID:       spanID,
		Attributes:   attributes,
	}); err != nil {
		log.Printf("audit emit %s: %v", authzEventDecisionName, err)
	}
}
