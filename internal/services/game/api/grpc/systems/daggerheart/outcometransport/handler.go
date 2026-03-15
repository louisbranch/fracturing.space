package outcometransport

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/action"
	bridge "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/commandids"
)

const (
	outcomeFlavorHope = "HOPE"
	outcomeFlavorFear = "FEAR"
)

const (
	commandTypeActionOutcomeApply             = commandids.ActionOutcomeApply
	commandTypeSessionGateOpen                = commandids.SessionGateOpen
	commandTypeSessionSpotlightSet            = commandids.SessionSpotlightSet
	commandTypeDaggerheartCharacterStatePatch = commandids.DaggerheartCharacterStatePatch
	commandTypeDaggerheartGMFearSet           = commandids.DaggerheartGMFearSet
)

const (
	eventTypeActionOutcomeApplied           = action.EventTypeOutcomeApplied
	eventTypeActionRollResolved             = action.EventTypeRollResolved
	eventTypeDaggerheartCharacterStatePatch = bridge.EventTypeCharacterStatePatched
	eventTypeDaggerheartGMFearChanged       = bridge.EventTypeGMFearChanged
)

// Handler owns the Daggerheart outcome transport surface behind an explicit
// dependency bundle so the root package can stay a thin facade.
type Handler struct {
	deps Dependencies
}

// NewHandler builds a Daggerheart outcome transport handler.
func NewHandler(deps Dependencies) *Handler {
	return &Handler{deps: deps}
}
