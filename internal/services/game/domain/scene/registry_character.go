package scene

import (
	"encoding/json"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

var sceneCharacterCommandContracts = []commandContract{
	{
		definition: command.Definition{
			Type:            CommandTypeCharacterAdd,
			Owner:           command.OwnerCore,
			ValidatePayload: validateCharacterAddedPayload,
			ActiveSession:   command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeCharacterRemove,
			Owner:           command.OwnerCore,
			ValidatePayload: validateCharacterRemovedPayload,
			ActiveSession:   command.AllowedDuringActiveSession(),
		},
	},
	{
		definition: command.Definition{
			Type:            CommandTypeCharacterTransfer,
			Owner:           command.OwnerCore,
			ValidatePayload: validateCharacterTransferPayload,
			ActiveSession:   command.AllowedDuringActiveSession(),
		},
	},
}

var sceneCharacterEventContracts = []eventProjectionContract{
	{
		definition: event.Definition{
			Type:            EventTypeCharacterAdded,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateCharacterAddedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
	{
		definition: event.Definition{
			Type:            EventTypeCharacterRemoved,
			Owner:           event.OwnerCore,
			Addressing:      event.AddressingPolicyEntityTarget,
			ValidatePayload: validateCharacterRemovedPayload,
			Intent:          event.IntentProjectionAndReplay,
		},
		emittable:  true,
		projection: true,
	},
}

func validateCharacterAddedPayload(raw json.RawMessage) error {
	var payload CharacterAddedPayload
	return json.Unmarshal(raw, &payload)
}

func validateCharacterRemovedPayload(raw json.RawMessage) error {
	var payload CharacterRemovedPayload
	return json.Unmarshal(raw, &payload)
}

func validateCharacterTransferPayload(raw json.RawMessage) error {
	var payload CharacterTransferPayload
	return json.Unmarshal(raw, &payload)
}
