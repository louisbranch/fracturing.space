package commandbuild

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	daggerheart "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
)

// CoreInput describes a command envelope for core domain commands.
type CoreInput struct {
	CampaignID   string
	Type         command.Type
	ActorType    command.ActorType
	ActorID      string
	SessionID    string
	RequestID    string
	InvocationID string
	EntityType   string
	EntityID     string
	PayloadJSON  []byte
}

// Core builds a core-domain command envelope.
func Core(in CoreInput) command.Command {
	return command.Command{
		CampaignID:   in.CampaignID,
		Type:         in.Type,
		ActorType:    in.ActorType,
		ActorID:      in.ActorID,
		SessionID:    in.SessionID,
		RequestID:    in.RequestID,
		InvocationID: in.InvocationID,
		EntityType:   in.EntityType,
		EntityID:     in.EntityID,
		PayloadJSON:  in.PayloadJSON,
	}
}

// DaggerheartSystemInput describes a system command envelope for Daggerheart.
type DaggerheartSystemInput struct {
	CoreInput
}

// DaggerheartSystem builds a Daggerheart system-domain command envelope.
func DaggerheartSystem(in DaggerheartSystemInput) command.Command {
	cmd := Core(in.CoreInput)
	cmd.SystemID = daggerheart.SystemID
	cmd.SystemVersion = daggerheart.SystemVersion
	return cmd
}
