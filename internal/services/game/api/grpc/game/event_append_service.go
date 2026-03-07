package game

import (
	"context"

	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AppendEvent appends a new event to the campaign journal.
func (s *EventService) AppendEvent(ctx context.Context, in *campaignv1.AppendEventRequest) (*campaignv1.AppendEventResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "append event request is required")
	}

	stored, err := newEventApplication(s).AppendEvent(ctx, in)
	if err != nil {
		return nil, err
	}

	return &campaignv1.AppendEventResponse{Event: eventToProto(stored)}, nil
}
