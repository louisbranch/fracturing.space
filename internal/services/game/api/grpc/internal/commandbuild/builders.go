package commandbuild

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
)

// CoreInput describes a command envelope for core domain commands.
type CoreInput struct {
	CampaignID    string
	Type          command.Type
	ActorType     command.ActorType
	ActorID       string
	SessionID     string
	SceneID       string
	RequestID     string
	InvocationID  string
	CorrelationID string
	EntityType    string
	EntityID      string
	PayloadJSON   []byte
}

// Core builds a core-domain command envelope.
func Core(in CoreInput) command.Command {
	return command.Command{
		CampaignID:    in.CampaignID,
		Type:          in.Type,
		ActorType:     in.ActorType,
		ActorID:       in.ActorID,
		SessionID:     in.SessionID,
		SceneID:       in.SceneID,
		RequestID:     in.RequestID,
		InvocationID:  in.InvocationID,
		CorrelationID: in.CorrelationID,
		EntityType:    in.EntityType,
		EntityID:      in.EntityID,
		PayloadJSON:   in.PayloadJSON,
	}
}

// CoreSystemInput describes a system-actor command envelope for core domains.
type CoreSystemInput struct {
	CampaignID    string
	Type          command.Type
	SessionID     string
	SceneID       string
	RequestID     string
	InvocationID  string
	CorrelationID string
	EntityType    string
	EntityID      string
	PayloadJSON   []byte
}

// CoreSystem builds a system-actor core-domain command envelope.
func CoreSystem(in CoreSystemInput) command.Command {
	return Core(CoreInput{
		CampaignID:    in.CampaignID,
		Type:          in.Type,
		ActorType:     command.ActorTypeSystem,
		SessionID:     in.SessionID,
		SceneID:       in.SceneID,
		RequestID:     in.RequestID,
		InvocationID:  in.InvocationID,
		CorrelationID: in.CorrelationID,
		EntityType:    in.EntityType,
		EntityID:      in.EntityID,
		PayloadJSON:   in.PayloadJSON,
	})
}

// SystemInput describes a command envelope for system-owned commands.
type SystemInput struct {
	CoreInput
	SystemID      string
	SystemVersion string
}

// System builds a system-domain command envelope.
func System(in SystemInput) command.Command {
	cmd := Core(in.CoreInput)
	cmd.SystemID = in.SystemID
	cmd.SystemVersion = in.SystemVersion
	return cmd
}

// SystemCommandInput describes a system-owned command emitted by the system
// actor.
type SystemCommandInput struct {
	CampaignID    string
	Type          command.Type
	SystemID      string
	SystemVersion string
	SessionID     string
	SceneID       string
	RequestID     string
	InvocationID  string
	CorrelationID string
	EntityType    string
	EntityID      string
	PayloadJSON   []byte
}

// SystemCommand builds a system-domain command envelope with ActorType pre-set
// to system.
func SystemCommand(in SystemCommandInput) command.Command {
	return System(SystemInput{
		CoreInput: CoreInput{
			CampaignID:    in.CampaignID,
			Type:          in.Type,
			ActorType:     command.ActorTypeSystem,
			SessionID:     in.SessionID,
			SceneID:       in.SceneID,
			RequestID:     in.RequestID,
			InvocationID:  in.InvocationID,
			CorrelationID: in.CorrelationID,
			EntityType:    in.EntityType,
			EntityID:      in.EntityID,
			PayloadJSON:   in.PayloadJSON,
		},
		SystemID:      in.SystemID,
		SystemVersion: in.SystemVersion,
	})
}
