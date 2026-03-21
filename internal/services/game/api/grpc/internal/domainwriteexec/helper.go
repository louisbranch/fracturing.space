package domainwriteexec

import (
	"context"
	"log/slog"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/observability/audit"
	auditevents "github.com/louisbranch/fracturing.space/internal/services/game/observability/audit/events"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"go.opentelemetry.io/otel/trace"
)

// Deps provides the domain execution/runtime dependencies consumed by
// transport write helpers.
type Deps interface {
	DomainExecutor() domainwrite.Executor
	DomainWriteRuntime() *domainwrite.Runtime
}

// ExecuteAndApply normalizes transport write options and executes one command
// using runtime-controlled inline apply behavior. If no OnRejection callback
// is set, a default structured-logging callback is wired automatically.
func ExecuteAndApply(
	ctx context.Context,
	deps Deps,
	applier domainwrite.EventApplier,
	cmd command.Command,
	options domainwrite.Options,
	normalizeConfig grpcerror.NormalizeDomainWriteOptionsConfig,
) (engine.Result, error) {
	grpcerror.NormalizeDomainWriteOptions(&options, normalizeConfig)
	setDefaultOnRejection(&options, deps)
	runtime := deps.DomainWriteRuntime()
	if runtime == nil {
		runtime = domainwrite.NewRuntime()
	}
	return runtime.ExecuteAndApply(ctx, deps.DomainExecutor(), applier, cmd, options)
}

// ExecuteWithoutInlineApply normalizes transport write options and executes one
// command while forcing projection apply to happen out-of-band.
func ExecuteWithoutInlineApply(
	ctx context.Context,
	deps Deps,
	cmd command.Command,
	options domainwrite.Options,
	normalizeConfig grpcerror.NormalizeDomainWriteOptionsConfig,
) (engine.Result, error) {
	grpcerror.NormalizeDomainWriteOptions(&options, normalizeConfig)
	setDefaultOnRejection(&options, deps)
	runtime := deps.DomainWriteRuntime()
	if runtime == nil {
		runtime = domainwrite.NewRuntime()
	}
	return runtime.ExecuteWithoutInlineApply(ctx, deps.DomainExecutor(), cmd, options)
}

// auditStoreDeps is an optional interface that Deps implementors can satisfy
// to enable audit event emission for domain rejections. Both game.Stores and
// daggerheart.Stores already carry AuditEventStore and satisfy this implicitly.
type auditStoreDeps interface {
	AuditEventStore() storage.AuditEventStore
}

// setDefaultOnRejection wires an audit-emitting callback when no OnRejection
// is configured. If deps implements auditStoreDeps, rejections are persisted
// as durable audit events; otherwise falls back to structured logging.
func setDefaultOnRejection(options *domainwrite.Options, deps Deps) {
	if options.OnRejection != nil {
		return
	}
	emitter := audit.NewEmitter(audit.DisabledPolicy())
	if ad, ok := deps.(auditStoreDeps); ok {
		if store := ad.AuditEventStore(); store != nil {
			emitter = audit.NewEmitter(audit.EnabledPolicy(store))
		}
	}
	options.OnRejection = func(ctx context.Context, info domainwrite.OnRejectionInfo) {
		if emitter != nil {
			var traceID, spanID string
			if sc := trace.SpanFromContext(ctx).SpanContext(); sc.IsValid() {
				traceID = sc.TraceID().String()
				spanID = sc.SpanID().String()
			}
			if err := emitter.Emit(ctx, storage.AuditEvent{
				EventName:  auditevents.DomainRejection,
				Severity:   string(audit.SeverityWarn),
				CampaignID: info.CampaignID,
				RequestID:  grpcmeta.RequestIDFromContext(ctx),
				TraceID:    traceID,
				SpanID:     spanID,
				Attributes: map[string]any{
					"command_type":   string(info.CommandType),
					"rejection_code": info.Code,
					"message":        info.Message,
				},
			}); err != nil {
				slog.Error("audit emit domain rejection", "error", err)
			}
			return
		}
		slog.Warn("domain rejection",
			"campaign_id", info.CampaignID,
			"command_type", info.CommandType,
			"rejection_code", info.Code,
		)
	}
}
