package campaigncontext

import (
	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/sessionctx"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Registrar is the narrow MCP registration surface owned by this feature package.
type Registrar interface {
	AddTool(*mcp.Tool, any) error
	AddResourceTemplate(*mcp.ResourceTemplate, mcp.ResourceHandler)
}

// RegisterTools adds the AI-backed campaign-context tools for the given clients.
func RegisterTools(
	registrar Registrar,
	campaignArtifactClient aiv1.CampaignArtifactServiceClient,
	systemReferenceClient aiv1.SystemReferenceServiceClient,
	getContext func() sessionctx.Context,
	notify sessionctx.ResourceUpdateNotifier,
) error {
	if campaignArtifactClient != nil {
		registrations := []struct {
			tool    *mcp.Tool
			handler any
		}{
			{tool: ArtifactListTool(), handler: ArtifactListHandler(campaignArtifactClient, getContext)},
			{tool: ArtifactGetTool(), handler: ArtifactGetHandler(campaignArtifactClient, getContext)},
			{tool: ArtifactUpsertTool(), handler: ArtifactUpsertHandler(campaignArtifactClient, getContext, notify)},
		}
		for _, registration := range registrations {
			if err := registrar.AddTool(registration.tool, registration.handler); err != nil {
				return err
			}
		}
	}
	if systemReferenceClient != nil {
		registrations := []struct {
			tool    *mcp.Tool
			handler any
		}{
			{tool: ReferenceSearchTool(), handler: ReferenceSearchHandler(systemReferenceClient)},
			{tool: ReferenceReadTool(), handler: ReferenceReadHandler(systemReferenceClient)},
		}
		for _, registration := range registrations {
			if err := registrar.AddTool(registration.tool, registration.handler); err != nil {
				return err
			}
		}
	}
	return nil
}

// RegisterResources adds the readable AI-backed campaign artifact resources.
func RegisterResources(registrar Registrar, client aiv1.CampaignArtifactServiceClient) {
	if client == nil {
		return
	}
	registrar.AddResourceTemplate(ArtifactListResourceTemplate(), ArtifactListResourceHandler(client))
	registrar.AddResourceTemplate(ArtifactResourceTemplate(), ArtifactResourceHandler(client))
}
