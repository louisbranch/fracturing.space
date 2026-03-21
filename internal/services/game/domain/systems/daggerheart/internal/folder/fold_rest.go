package folder

import (
	"fmt"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/payload"
	daggerheartstate "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/state"
)

func (f *Folder) foldRestTaken(state *daggerheartstate.SnapshotState, p payload.RestTakenPayload) error {
	state.GMFear = p.GMFear
	if state.GMFear < daggerheartstate.GMFearMin || state.GMFear > daggerheartstate.GMFearMax {
		return fmt.Errorf("rest_taken gm_fear_after must be in range %d..%d", daggerheartstate.GMFearMin, daggerheartstate.GMFearMax)
	}
	for _, participantID := range p.Participants {
		if p.RefreshRest || p.RefreshLongRest {
			clearRestTemporaryArmor(state, participantID.String(), p.RefreshRest, p.RefreshLongRest)
		}
		clearRestStatModifiers(state, participantID, p.RefreshRest, p.RefreshLongRest)
	}
	return nil
}
