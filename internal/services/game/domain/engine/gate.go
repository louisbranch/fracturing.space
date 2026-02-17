package engine

import (
	"fmt"
	"strings"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
)

const rejectionCodeSessionGateOpen = "SESSION_GATE_OPEN"

// DecisionGate enforces session gate policy before command decisions run.
type DecisionGate struct {
	Registry *command.Registry
}

// Check returns a rejection when a session gate blocks the command.
//
// Gate evaluation is intentionally centralized so each domain package can expose a
// simple session command shape while policy enforcement remains consistent.
func (g DecisionGate) Check(state session.State, cmd command.Command) command.Decision {
	if g.Registry == nil {
		return command.Decision{}
	}
	def, ok := g.Registry.Definition(cmd.Type)
	if !ok {
		return command.Decision{}
	}
	if def.Gate.Scope != command.GateScopeSession {
		return command.Decision{}
	}
	if def.Gate.AllowWhenOpen || !state.GateOpen {
		return command.Decision{}
	}
	message := "session gate is open"
	if gateID := strings.TrimSpace(state.GateID); gateID != "" {
		message = fmt.Sprintf("session gate is open: %s", gateID)
	}
	return command.Reject(command.Rejection{
		Code:    rejectionCodeSessionGateOpen,
		Message: message,
	})
}
