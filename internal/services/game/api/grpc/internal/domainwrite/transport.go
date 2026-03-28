package domainwrite

import (
	"context"
	"log/slog"

	errori18n "github.com/louisbranch/fracturing.space/internal/platform/errors/i18n"
	grpcstatus "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcstatus"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/observability/audit"
	auditevents "github.com/louisbranch/fracturing.space/internal/services/game/observability/audit/events"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"go.opentelemetry.io/otel/trace"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Deps provides the domain execution/runtime dependencies consumed by
// transport write helpers.
type Deps interface {
	DomainExecutor() Executor
	DomainWriteRuntime() *Runtime
}

// WritePath groups the dependencies needed to execute domain commands and
// apply their results at the transport boundary. Services embed or accept
// a WritePath instead of carrying the executor, runtime, and audit store
// as separate fields.
//
// WritePath satisfies [Deps] so it can be passed directly to
// [TransportExecuteAndApply] and [TransportExecuteWithoutInlineApply].
type WritePath struct {
	// Executor executes domain commands.
	Executor Executor

	// Runtime owns request-path write execution flags (inline apply,
	// intent filtering).
	Runtime *Runtime

	// Audit is the optional audit event store for domain rejection
	// telemetry. When set, rejections are persisted as durable audit
	// events; when nil, they are logged via structured logging.
	Audit storage.AuditEventStore
}

// DomainExecutor implements [Deps].
func (w WritePath) DomainExecutor() Executor { return w.Executor }

// DomainWriteRuntime implements [Deps].
func (w WritePath) DomainWriteRuntime() *Runtime { return w.Runtime }

// AuditEventStore implements the optional [auditStoreDeps] interface
// consumed by [setDefaultOnRejection].
func (w WritePath) AuditEventStore() storage.AuditEventStore { return w.Audit }

// NormalizeDomainWriteOptionsConfig controls default gRPC mapping behavior for
// domain write helper options.
type NormalizeDomainWriteOptionsConfig struct {
	// PreserveDomainCodeOnApply keeps structured domain errors intact in apply
	// callbacks instead of flattening them to codes.Internal. This is used by
	// game-system extensions (e.g. Daggerheart) that return system-specific
	// domain error codes from projection apply paths.
	PreserveDomainCodeOnApply bool
}

// TransportExecuteAndApply normalizes transport write options, wires audit
// callbacks, and executes one command using runtime-controlled inline apply
// behavior. If no OnRejection callback is set, a default audit-emitting
// callback is wired automatically.
func TransportExecuteAndApply(
	ctx context.Context,
	deps Deps,
	applier EventApplier,
	cmd command.Command,
	options Options,
	normalizeConfig NormalizeDomainWriteOptionsConfig,
) (engine.Result, error) {
	NormalizeDomainWriteOptions(ctx, &options, normalizeConfig)
	setDefaultOnRejection(&options, deps)
	runtime := deps.DomainWriteRuntime()
	if runtime == nil {
		runtime = NewRuntime()
	}
	return runtime.ExecuteAndApply(ctx, deps.DomainExecutor(), applier, cmd, options)
}

// TransportExecuteWithoutInlineApply normalizes transport write options and
// executes one command while forcing projection apply to happen out-of-band.
func TransportExecuteWithoutInlineApply(
	ctx context.Context,
	deps Deps,
	cmd command.Command,
	options Options,
	normalizeConfig NormalizeDomainWriteOptionsConfig,
) (engine.Result, error) {
	NormalizeDomainWriteOptions(ctx, &options, normalizeConfig)
	setDefaultOnRejection(&options, deps)
	runtime := deps.DomainWriteRuntime()
	if runtime == nil {
		runtime = NewRuntime()
	}
	return runtime.ExecuteWithoutInlineApply(ctx, deps.DomainExecutor(), cmd, options)
}

// NormalizeDomainWriteOptions applies gRPC-aware defaults to Options while
// allowing callers to override callbacks explicitly. The context is used to
// derive the caller's locale for rejection message formatting.
func NormalizeDomainWriteOptions(ctx context.Context, options *Options, config NormalizeDomainWriteOptionsConfig) {
	if options == nil {
		return
	}
	if options.ExecuteErr == nil {
		message := options.ExecuteErrMessage
		if message == "" {
			message = "execute domain command"
		}
		options.ExecuteErr = func(err error) error {
			if engine.IsNonRetryable(err) {
				return status.Errorf(codes.FailedPrecondition, "%s: %v", message, err)
			}
			return grpcstatus.Internal(message, err)
		}
	}
	if options.ApplyErr == nil {
		message := options.ApplyErrMessage
		if message == "" {
			message = "apply event"
		}
		if config.PreserveDomainCodeOnApply {
			options.ApplyErr = grpcstatus.ApplyErrorWithDomainCodePreserve(message)
		} else {
			options.ApplyErr = func(err error) error {
				return grpcstatus.Internal(message, err)
			}
		}
	}
	if options.RejectErr == nil {
		locale := grpcmeta.LocaleFromContext(ctx)
		options.RejectErr = func(code, message string) error {
			cat := errori18n.GetCatalog(locale)
			if localized := cat.Format(code, nil); localized != code {
				return status.Error(codes.FailedPrecondition, localized)
			}
			return status.Error(codes.FailedPrecondition, message)
		}
	}
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
func setDefaultOnRejection(options *Options, deps Deps) {
	if options.OnRejection != nil {
		return
	}
	emitter := audit.NewEmitter(audit.DisabledPolicy())
	if ad, ok := deps.(auditStoreDeps); ok {
		if store := ad.AuditEventStore(); store != nil {
			emitter = audit.NewEmitter(audit.EnabledPolicy(store))
		}
	}
	options.OnRejection = func(ctx context.Context, info OnRejectionInfo) {
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
