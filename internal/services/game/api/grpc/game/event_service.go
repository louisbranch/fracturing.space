package game

import (
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

// EventService implements the game.v1.EventService gRPC API.
type EventService struct {
	campaignv1.UnimplementedEventServiceServer
	stores Stores
}

// NewEventService creates an EventService with the provided stores.
func NewEventService(stores Stores) *EventService {
	return &EventService{
		stores: stores,
	}
}
