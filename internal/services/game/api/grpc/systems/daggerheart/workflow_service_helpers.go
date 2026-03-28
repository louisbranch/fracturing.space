package daggerheart

import (
	"context"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/internal/grpcerror"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/charactermutationtransport"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowruntime"
	"github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowwrite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
)

// workflowRuntime returns the shared Daggerheart workflow runtime bound to this
// service's write path and stores.
func (s *DaggerheartService) workflowRuntime() *workflowruntime.Runtime {
	return workflowwrite.NewRuntime(s.stores.Write, s.stores.Event, s.stores.Daggerheart)
}

// executeWorkflowDomainCommand executes one Daggerheart system-domain command
// using the service's configured workflow write path.
func (s *DaggerheartService) executeWorkflowDomainCommand(ctx context.Context, in workflowwrite.DomainCommandInput) error {
	return workflowwrite.ExecuteDomainCommand(ctx, s.stores.Write, s.stores.Daggerheart, in)
}

// executeCharacterMutationCommand adapts the character-mutation transport input
// into the shared Daggerheart workflow domain-command shape.
func (s *DaggerheartService) executeCharacterMutationCommand(ctx context.Context, in charactermutationtransport.CharacterCommandInput) error {
	return s.executeWorkflowDomainCommand(ctx, workflowwrite.DomainCommandInput{
		CampaignID:      in.CampaignID,
		CommandType:     in.CommandType,
		SessionID:       strings.TrimSpace(in.SessionID),
		RequestID:       in.RequestID,
		InvocationID:    in.InvocationID,
		EntityType:      "character",
		EntityID:        in.CharacterID,
		PayloadJSON:     in.PayloadJSON,
		MissingEventMsg: in.MissingEventMsg,
		ApplyErrMessage: in.ApplyErrMessage,
	})
}

// executeWorkflowSystemCommand executes one Daggerheart workflow system command
// using the service's configured runtime dependencies.
func (s *DaggerheartService) executeWorkflowSystemCommand(ctx context.Context, in workflowruntime.SystemCommandInput) error {
	return workflowwrite.ExecuteSystemCommand(ctx, s.stores.Write, s.stores.Event, s.stores.Daggerheart, in)
}

// executeWorkflowCoreCommand executes one core-domain workflow command after
// resolving the projection applier owned by this service.
func (s *DaggerheartService) executeWorkflowCoreCommand(ctx context.Context, in workflowwrite.CoreCommandInput) (engine.Result, error) {
	applier, err := s.resolvedApplier()
	if err != nil {
		return engine.Result{}, grpcerror.Internal("build projection applier", err)
	}
	return workflowwrite.ExecuteCoreCommand(ctx, s.stores.Write, applier, in)
}

// applyWorkflowCoreCommand executes one core-domain workflow command and
// discards the domain result when callers only care about transport success.
func (s *DaggerheartService) applyWorkflowCoreCommand(ctx context.Context, in workflowwrite.CoreCommandInput) error {
	_, err := s.executeWorkflowCoreCommand(ctx, in)
	return err
}
