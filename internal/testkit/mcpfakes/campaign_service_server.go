package mcpfakes

import (
	"context"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
)

// CampaignServiceServer is a fake gRPC server for MCP integration tests.
type CampaignServiceServer struct {
	statev1.UnimplementedCampaignServiceServer
}

// CreateCampaign echoes basic campaign fields for test assertions.
func (f *CampaignServiceServer) CreateCampaign(ctx context.Context, req *statev1.CreateCampaignRequest) (*statev1.CreateCampaignResponse, error) {
	return &statev1.CreateCampaignResponse{
		Campaign: &statev1.Campaign{
			Id:           "camp-123",
			Name:         req.GetName(),
			GmMode:       req.GetGmMode(),
			Intent:       req.GetIntent(),
			AccessPolicy: req.GetAccessPolicy(),
		},
		OwnerParticipant: &statev1.Participant{Id: "part-123"},
	}, nil
}
