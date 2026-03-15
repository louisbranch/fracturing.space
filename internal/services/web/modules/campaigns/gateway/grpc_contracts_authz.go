package gateway

import (
	"context"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"google.golang.org/grpc"
)

// AuthorizationClient exposes campaign authorization checks.
type AuthorizationClient interface {
	Can(context.Context, *statev1.CanRequest, ...grpc.CallOption) (*statev1.CanResponse, error)
	BatchCan(context.Context, *statev1.BatchCanRequest, ...grpc.CallOption) (*statev1.BatchCanResponse, error)
}
