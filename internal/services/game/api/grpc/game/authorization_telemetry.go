package game

import (
	"context"
	"log/slog"
	"strings"

	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	domainauthz "github.com/louisbranch/fracturing.space/internal/services/game/domain/authz"
	"github.com/louisbranch/fracturing.space/internal/services/game/observability/audit"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// authzDecisionEvent groups the parameters for an authorization decision
// audit log entry, replacing the previous 9-argument positional call.
type authzDecisionEvent struct {
	Store           storage.AuditEventStore
	CampaignID      string
	Capability      domainauthz.Capability
	Decision        string
	ReasonCode      string
	Actor           storage.ParticipantRecord
	Err             error
	ExtraAttributes map[string]any
}

func emitAuthzDecisionTelemetry(ctx context.Context, evt authzDecisionEvent) {
	severity := audit.SeverityInfo
	code := codes.OK
	if evt.Err != nil {
		severity = audit.SeverityWarn
		if st, ok := status.FromError(evt.Err); ok {
			code = st.Code()
		}
		if code == codes.Internal {
			severity = audit.SeverityError
		}
	}

	actorID := strings.TrimSpace(evt.Actor.ID)
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
		"decision":      evt.Decision,
		"reason_code":   evt.ReasonCode,
		"policy_action": policyCapabilityLabel(evt.Capability),
		"grpc_code":     code.String(),
	}
	if access := strings.TrimSpace(string(evt.Actor.CampaignAccess)); access != "" {
		attributes["campaign_access"] = access
	}
	if userID := strings.TrimSpace(evt.Actor.UserID); userID != "" {
		attributes["actor_user_id"] = userID
	}
	for key, value := range evt.ExtraAttributes {
		attributes[key] = value
	}

	emitter := audit.NewEmitter(evt.Store)
	if err := emitter.Emit(ctx, storage.AuditEvent{
		EventName:    authzEventDecisionName,
		Severity:     string(severity),
		CampaignID:   strings.TrimSpace(evt.CampaignID),
		ActorType:    actorType,
		ActorID:      actorID,
		RequestID:    grpcmeta.RequestIDFromContext(ctx),
		InvocationID: grpcmeta.InvocationIDFromContext(ctx),
		TraceID:      traceID,
		SpanID:       spanID,
		Attributes:   attributes,
	}); err != nil {
		slog.Error("audit emit failed", "event", authzEventDecisionName, "error", err)
	}
}
