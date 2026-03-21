package participant

import (
	"fmt"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/ids"
)

func decideJoin(state State, cmd command.Command, now func() time.Time) command.Decision {
	now = command.NowFunc(now)

	if state.Joined {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantAlreadyJoined,
			Message: "participant already joined",
		})
	}
	payload, err := decodeCommandPayload[JoinPayload](cmd)
	if err != nil {
		return command.Reject(command.Rejection{
			Code:    command.RejectionCodePayloadDecodeFailed,
			Message: fmt.Sprintf("decode %s payload: %v", cmd.Type, err),
		})
	}

	participantID := strings.TrimSpace(payload.ParticipantID.String())
	if participantID == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantIDRequired,
			Message: "participant id is required",
		})
	}
	userID := strings.TrimSpace(payload.UserID.String())
	name := strings.TrimSpace(payload.Name)
	if name == "" {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantNameEmpty,
			Message: "name is required",
		})
	}
	role, ok := normalizeRoleLabel(payload.Role)
	if !ok {
		return command.Reject(command.Rejection{
			Code:    rejectionCodeParticipantRoleInvalid,
			Message: "participant role is required",
		})
	}
	controller, ok := normalizeControllerLabel(payload.Controller)
	if !ok {
		if strings.TrimSpace(payload.Controller) != "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantControllerInvalid,
				Message: "participant controller is invalid",
			})
		}
		controller = "human"
	}
	access, ok := normalizeCampaignAccessLabel(payload.CampaignAccess)
	if !ok {
		if strings.TrimSpace(payload.CampaignAccess) != "" {
			return command.Reject(command.Rejection{
				Code:    rejectionCodeParticipantAccessInvalid,
				Message: "campaign access is invalid",
			})
		}
		access = "member"
	}
	if rejection, ok := validateAISeatInvariant(userID, role, controller, access); !ok {
		return command.Reject(rejection)
	}
	avatarSetID, avatarAssetID, err := resolveParticipantAvatarSelection(
		participantID,
		userID,
		payload.AvatarSetID,
		payload.AvatarAssetID,
	)
	if err != nil {
		return command.Reject(participantAvatarRejection(err))
	}
	pronouns := strings.TrimSpace(payload.Pronouns)

	normalizedPayload := JoinPayload{
		ParticipantID:  ids.ParticipantID(participantID),
		UserID:         ids.UserID(userID),
		Name:           name,
		Role:           role,
		Controller:     controller,
		CampaignAccess: access,
		AvatarSetID:    avatarSetID,
		AvatarAssetID:  avatarAssetID,
		Pronouns:       pronouns,
	}
	return acceptParticipantEvent(cmd, now, EventTypeJoined, participantID, normalizedPayload)
}
