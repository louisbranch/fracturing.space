package workflowwrite

import "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/systems/daggerheart/workflowruntime"

// CommandInput re-exports the unified struct from workflowruntime so callers
// in the write path do not need to import the runtime package directly.
type CommandInput = workflowruntime.CommandInput

// DomainCommandInput is a type alias kept for call-site compatibility.
type DomainCommandInput = CommandInput

// CoreCommandInput is a type alias kept for call-site compatibility.
type CoreCommandInput = CommandInput
