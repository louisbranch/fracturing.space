package campaign

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func decideFork(state State, cmd command.Command, now func() time.Time) command.Decision {
	if !state.Created {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCampaignNotCreated,
			Message: "campaign does not exist",
		})
	}
	var payload ForkPayload
	if err := json.Unmarshal(cmd.PayloadJSON, &payload); err != nil {
		return command.Reject(command.Rejection{
			Code:    rejectionCodePayloadDecodeFailed,
			Message: commandDecodeMessage(cmd, err),
		})
	}
	payload.ParentCampaignID = ids.CampaignID(strings.TrimSpace(payload.ParentCampaignID.String()))
	payload.OriginCampaignID = ids.CampaignID(strings.TrimSpace(payload.OriginCampaignID.String()))
	payloadJSON, _ := json.Marshal(payload)
	evt := command.NewEvent(cmd, EventTypeForked, "campaign", string(cmd.CampaignID), payloadJSON, command.NowFunc(now)().UTC())
	return command.Accept(evt)
}
