package domainwriteexec

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
)

// WritePath groups the dependencies needed to execute domain commands and
// apply their results at the transport boundary. Services embed or accept
// a WritePath instead of carrying the executor, runtime, and audit store
// as separate fields.
//
// WritePath satisfies [Deps] so it can be passed directly to
// [ExecuteAndApply] and [ExecuteWithoutInlineApply].
type WritePath struct {
	// Executor executes domain commands.
	Executor domainwrite.Executor

	// Runtime owns request-path write execution flags (inline apply,
	// intent filtering).
	Runtime *domainwrite.Runtime

	// Audit is the optional audit event store for domain rejection
	// telemetry. When set, rejections are persisted as durable audit
	// events; when nil, they are logged via structured logging.
	Audit storage.AuditEventStore
}

// DomainExecutor implements [Deps].
func (w WritePath) DomainExecutor() domainwrite.Executor { return w.Executor }

// DomainWriteRuntime implements [Deps].
func (w WritePath) DomainWriteRuntime() *domainwrite.Runtime { return w.Runtime }

// AuditEventStore implements the optional [auditStoreDeps] interface
// consumed by [setDefaultOnRejection].
func (w WritePath) AuditEventStore() storage.AuditEventStore { return w.Audit }
