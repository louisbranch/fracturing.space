package server

import (
	"context"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"google.golang.org/grpc"
)

// artifactClientAdapter wraps a CampaignArtifactServiceServer to satisfy the
// CampaignArtifactServiceClient interface, enabling in-process calls without
// a self-dial loopback connection.
type artifactClientAdapter struct {
	server aiv1.CampaignArtifactServiceServer
}

func (a artifactClientAdapter) EnsureCampaignArtifacts(ctx context.Context, in *aiv1.EnsureCampaignArtifactsRequest, _ ...grpc.CallOption) (*aiv1.EnsureCampaignArtifactsResponse, error) {
	return a.server.EnsureCampaignArtifacts(ctx, in)
}

func (a artifactClientAdapter) ListCampaignArtifacts(ctx context.Context, in *aiv1.ListCampaignArtifactsRequest, _ ...grpc.CallOption) (*aiv1.ListCampaignArtifactsResponse, error) {
	return a.server.ListCampaignArtifacts(ctx, in)
}

func (a artifactClientAdapter) GetCampaignArtifact(ctx context.Context, in *aiv1.GetCampaignArtifactRequest, _ ...grpc.CallOption) (*aiv1.GetCampaignArtifactResponse, error) {
	return a.server.GetCampaignArtifact(ctx, in)
}

func (a artifactClientAdapter) UpsertCampaignArtifact(ctx context.Context, in *aiv1.UpsertCampaignArtifactRequest, _ ...grpc.CallOption) (*aiv1.UpsertCampaignArtifactResponse, error) {
	return a.server.UpsertCampaignArtifact(ctx, in)
}

// referenceClientAdapter wraps a SystemReferenceServiceServer to satisfy the
// SystemReferenceServiceClient interface, enabling in-process calls without
// a self-dial loopback connection.
type referenceClientAdapter struct {
	server aiv1.SystemReferenceServiceServer
}

func (a referenceClientAdapter) SearchSystemReference(ctx context.Context, in *aiv1.SearchSystemReferenceRequest, _ ...grpc.CallOption) (*aiv1.SearchSystemReferenceResponse, error) {
	return a.server.SearchSystemReference(ctx, in)
}

func (a referenceClientAdapter) ReadSystemReferenceDocument(ctx context.Context, in *aiv1.ReadSystemReferenceDocumentRequest, _ ...grpc.CallOption) (*aiv1.ReadSystemReferenceDocumentResponse, error) {
	return a.server.ReadSystemReferenceDocument(ctx, in)
}
