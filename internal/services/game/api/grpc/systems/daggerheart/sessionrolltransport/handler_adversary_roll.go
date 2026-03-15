package sessionrolltransport

import (
	"context"

	pb "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func (h *Handler) SessionAdversaryAttackRoll(ctx context.Context, in *pb.SessionAdversaryAttackRollRequest) (*pb.SessionAdversaryAttackRollResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session adversary attack roll request is required")
	}
	return h.sessionAdversaryAttackRoll(ctx, in)
}

func (h *Handler) SessionAdversaryActionCheck(ctx context.Context, in *pb.SessionAdversaryActionCheckRequest) (*pb.SessionAdversaryActionCheckResponse, error) {
	if in == nil {
		return nil, status.Error(codes.InvalidArgument, "session adversary action check request is required")
	}
	return h.sessionAdversaryActionCheck(ctx, in)
}
