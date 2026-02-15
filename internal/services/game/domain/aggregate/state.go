package aggregate

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/character"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/invite"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/participant"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/session"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/system"
)

// State captures aggregate core domain state.
type State struct {
	Campaign     campaign.State
	Session      session.State
	Participants map[string]participant.State
	Characters   map[string]character.State
	Invites      map[string]invite.State
	Systems      map[system.Key]any
}
