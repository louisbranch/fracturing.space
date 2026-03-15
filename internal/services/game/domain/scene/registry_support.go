package scene

import (
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

type commandContract struct {
	definition command.Definition
}

type eventProjectionContract struct {
	definition event.Definition
	emittable  bool
	projection bool
}

func sceneCommandTypes(contracts []commandContract) []command.Type {
	types := make([]command.Type, 0, len(contracts))
	for _, contract := range contracts {
		types = append(types, contract.definition.Type)
	}
	return types
}

func sceneEventTypes(contracts []eventProjectionContract, include func(eventProjectionContract) bool) []event.Type {
	types := make([]event.Type, 0, len(contracts))
	for _, contract := range contracts {
		if include(contract) {
			types = append(types, contract.definition.Type)
		}
	}
	return types
}

func appendSceneCommandContracts(groups ...[]commandContract) []commandContract {
	total := 0
	for _, group := range groups {
		total += len(group)
	}
	contracts := make([]commandContract, 0, total)
	for _, group := range groups {
		contracts = append(contracts, group...)
	}
	return contracts
}

func appendSceneEventContracts(groups ...[]eventProjectionContract) []eventProjectionContract {
	total := 0
	for _, group := range groups {
		total += len(group)
	}
	contracts := make([]eventProjectionContract, 0, total)
	for _, group := range groups {
		contracts = append(contracts, group...)
	}
	return contracts
}
