package workflowruntime

import (
	"context"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/commandbuild"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/domainwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Runtime owns shared replay-check and Daggerheart system-command support for
// sibling workflow transport packages.
type Runtime struct {
	deps Dependencies
}

// New builds a shared workflow runtime.
func New(deps Dependencies) *Runtime {
	return &Runtime{deps: deps}
}

// SessionRequestEventExists checks whether the session event stream already
// contains the requested event after the given roll sequence.
func (r *Runtime) SessionRequestEventExists(ctx context.Context, in ReplayCheckInput) (bool, error) {
	if r == nil || r.deps.Event == nil {
		return false, status.Error(codes.Internal, "event store is not configured")
	}

	requestID := strings.TrimSpace(in.RequestID)
	entityID := strings.TrimSpace(in.EntityID)
	if in.RollSeq == 0 || requestID == "" {
		return false, nil
	}

	result, err := r.deps.Event.ListEventsPage(ctx, storage.ListEventsPageRequest{
		CampaignID: in.CampaignID,
		AfterSeq:   in.RollSeq - 1,
		PageSize:   1,
		Filter: storage.EventQueryFilter{
			SessionID: in.SessionID,
			RequestID: requestID,
			EventType: string(in.EventType),
			EntityID:  entityID,
		},
	})
	if err != nil {
		return false, err
	}
	return len(result.Events) > 0, nil
}

// ExecuteSystemCommand builds and executes one Daggerheart system command
// through the shared domain-write path.
func (r *Runtime) ExecuteSystemCommand(ctx context.Context, in SystemCommandInput) error {
	if r == nil || r.deps.Daggerheart == nil {
		return status.Error(codes.Internal, "daggerheart store is not configured")
	}
	if r.deps.ExecuteDomainCommand == nil {
		return status.Error(codes.Internal, "domain command executor is not configured")
	}

	cmd := commandbuild.SystemCommand(commandbuild.SystemCommandInput{
		CampaignID:    in.CampaignID,
		Type:          in.CommandType,
		SystemID:      daggerheart.SystemID,
		SystemVersion: daggerheart.SystemVersion,
		SessionID:     in.SessionID,
		SceneID:       in.SceneID,
		RequestID:     in.RequestID,
		InvocationID:  in.InvocationID,
		CorrelationID: in.CorrelationID,
		EntityType:    in.EntityType,
		EntityID:      in.EntityID,
		PayloadJSON:   in.PayloadJSON,
	})
	return r.deps.ExecuteDomainCommand(
		ctx,
		cmd,
		daggerheart.NewAdapter(r.deps.Daggerheart),
		domainwrite.RequireEventsWithDiagnostics(in.MissingEventMsg, in.ApplyErrMessage),
	)
}
