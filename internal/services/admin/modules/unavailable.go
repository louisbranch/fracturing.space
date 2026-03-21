package modules

import (
	"context"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	invitev1 "github.com/louisbranch/fracturing.space/api/gen/go/invite/v1"
	statusv1 "github.com/louisbranch/fracturing.space/api/gen/go/status/v1"
	daggerheartv1 "github.com/louisbranch/fracturing.space/api/gen/go/systems/daggerheart/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// unavailableConn implements grpc.ClientConnInterface, returning
// codes.Unavailable for every RPC. This allows modules to receive
// non-nil client stubs that fail gracefully through normal gRPC
// error handling paths, eliminating per-handler nil-client checks.
type unavailableConn struct{}

func (unavailableConn) Invoke(context.Context, string, any, any, ...grpc.CallOption) error {
	return status.Error(codes.Unavailable, "service not connected")
}

func (unavailableConn) NewStream(context.Context, *grpc.StreamDesc, string, ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, status.Error(codes.Unavailable, "service not connected")
}

// ensureClients replaces nil client fields with unavailable stubs
// so that modules never need to check for nil clients.
func (input *BuildInput) ensureClients() {
	var conn unavailableConn
	if input.AuthClient == nil {
		input.AuthClient = authv1.NewAuthServiceClient(conn)
	}
	if input.CampaignClient == nil {
		input.CampaignClient = statev1.NewCampaignServiceClient(conn)
	}
	if input.CharacterClient == nil {
		input.CharacterClient = statev1.NewCharacterServiceClient(conn)
	}
	if input.ParticipantClient == nil {
		input.ParticipantClient = statev1.NewParticipantServiceClient(conn)
	}
	if input.InviteClient == nil {
		input.InviteClient = invitev1.NewInviteServiceClient(conn)
	}
	if input.SessionClient == nil {
		input.SessionClient = statev1.NewSessionServiceClient(conn)
	}
	if input.EventClient == nil {
		input.EventClient = statev1.NewEventServiceClient(conn)
	}
	if input.StatisticsClient == nil {
		input.StatisticsClient = statev1.NewStatisticsServiceClient(conn)
	}
	if input.SystemClient == nil {
		input.SystemClient = statev1.NewSystemServiceClient(conn)
	}
	if input.DaggerheartContentClient == nil {
		input.DaggerheartContentClient = daggerheartv1.NewDaggerheartContentServiceClient(conn)
	}
	if input.StatusClient == nil {
		input.StatusClient = statusv1.NewStatusServiceClient(conn)
	}
}
