package service

import (
	"context"
	"testing"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/domain"
	"google.golang.org/grpc"
)

type fakeCampaignArtifactClient struct{}

func (fakeCampaignArtifactClient) EnsureCampaignArtifacts(context.Context, *aiv1.EnsureCampaignArtifactsRequest, ...grpc.CallOption) (*aiv1.EnsureCampaignArtifactsResponse, error) {
	return nil, nil
}
func (fakeCampaignArtifactClient) ListCampaignArtifacts(context.Context, *aiv1.ListCampaignArtifactsRequest, ...grpc.CallOption) (*aiv1.ListCampaignArtifactsResponse, error) {
	return nil, nil
}
func (fakeCampaignArtifactClient) GetCampaignArtifact(context.Context, *aiv1.GetCampaignArtifactRequest, ...grpc.CallOption) (*aiv1.GetCampaignArtifactResponse, error) {
	return nil, nil
}
func (fakeCampaignArtifactClient) UpsertCampaignArtifact(context.Context, *aiv1.UpsertCampaignArtifactRequest, ...grpc.CallOption) (*aiv1.UpsertCampaignArtifactResponse, error) {
	return nil, nil
}

type fakeSystemReferenceClient struct{}

func (fakeSystemReferenceClient) SearchSystemReference(context.Context, *aiv1.SearchSystemReferenceRequest, ...grpc.CallOption) (*aiv1.SearchSystemReferenceResponse, error) {
	return nil, nil
}
func (fakeSystemReferenceClient) ReadSystemReferenceDocument(context.Context, *aiv1.ReadSystemReferenceDocumentRequest, ...grpc.CallOption) (*aiv1.ReadSystemReferenceDocumentResponse, error) {
	return nil, nil
}

func TestRegisterCampaignContextToolsRegistersArtifactAndReferenceTools(t *testing.T) {
	target := &fakeMCPRegistrationTarget{}

	err := registerCampaignContextTools(
		target,
		fakeCampaignArtifactClient{},
		fakeSystemReferenceClient{},
		func() domain.Context { return domain.Context{} },
		nil,
	)
	if err != nil {
		t.Fatalf("registerCampaignContextTools() error = %v", err)
	}

	want := []string{
		"campaign_artifact_list",
		"campaign_artifact_get",
		"campaign_artifact_upsert",
		"system_reference_search",
		"system_reference_read",
	}
	if len(target.tools) != len(want) {
		t.Fatalf("tool count = %d, want %d (%v)", len(target.tools), len(want), target.tools)
	}
	for index, name := range want {
		if target.tools[index] != name {
			t.Fatalf("tool[%d] = %q, want %q", index, target.tools[index], name)
		}
	}
}

func TestRegisterCampaignContextResourcesRegistersArtifactTemplates(t *testing.T) {
	target := &fakeMCPRegistrationTarget{}

	registerCampaignContextResources(target, fakeCampaignArtifactClient{})

	want := []string{
		"campaign://{campaign_id}/artifacts",
		"campaign://{campaign_id}/artifacts/{path}",
	}
	if len(target.resourceTemplates) != len(want) {
		t.Fatalf("resource template count = %d, want %d (%v)", len(target.resourceTemplates), len(want), target.resourceTemplates)
	}
	for index, uri := range want {
		if target.resourceTemplates[index] != uri {
			t.Fatalf("resource template[%d] = %q, want %q", index, target.resourceTemplates[index], uri)
		}
	}
}
