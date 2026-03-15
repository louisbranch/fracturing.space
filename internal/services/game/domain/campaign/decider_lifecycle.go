package campaign

import (
	"encoding/json"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
)

var lifecycleCommandTargets = map[command.Type]Status{
	CommandTypeEnd:     StatusCompleted,
	CommandTypeArchive: StatusArchived,
	CommandTypeRestore: StatusDraft,
}

func decideLifecycleStatus(state State, cmd command.Command, now func() time.Time) command.Decision {
	targetStatus, _ := statusCommandTarget(cmd.Type)
	if !state.Created {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCampaignNotCreated,
			Message: "campaign does not exist",
		})
	}
	currentStatus := state.Status
	if currentStatus == "" {
		currentStatus = StatusDraft
	}
	if !isStatusTransitionAllowed(currentStatus, targetStatus) {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeCampaignStatusTransition,
			Message: "campaign status transition is not allowed",
		})
	}

	normalizedPayload := UpdatePayload{Fields: map[string]string{"status": string(targetStatus)}}
	payloadJSON, _ := json.Marshal(normalizedPayload)
	evt := command.NewEvent(cmd, EventTypeUpdated, "campaign", string(cmd.CampaignID), payloadJSON, command.NowFunc(now)().UTC())
	return command.Accept(evt)
}

// statusCommandTarget maps lifecycle command names to their destination status.
//
// Centralizing lifecycle transition targets prevents duplicate status-mapping logic
// in every handler and keeps command intent readable.
func statusCommandTarget(cmdType command.Type) (Status, bool) {
	target, ok := lifecycleCommandTargets[cmdType]
	return target, ok
}
