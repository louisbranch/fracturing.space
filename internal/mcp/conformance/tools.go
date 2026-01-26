//go:build conformance

package conformance

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const (
	simpleTextResponse        = "This is a simple text response for testing."
	errorTextResponse         = "This is an error response for testing."
	errorHandlingResponse     = "This tool intentionally returns an error for testing"
	staticTextResourceContent = "This is the content of the static text resource."
	staticTextResourceName    = "test_static_text"
	staticTextResourceURI     = "test://static-text"
)

// Register adds conformance-only MCP fixtures (tools, prompts, resources).
func Register(mcpServer *mcp.Server) {
	if mcpServer == nil {
		return
	}

	mcp.AddTool(mcpServer, simpleTextTool(), simpleTextHandler())
	mcp.AddTool(mcpServer, errorContentTool(), errorContentHandler())
	mcp.AddTool(mcpServer, errorHandlingTool(), errorHandlingHandler())
	mcpServer.AddPrompt(simplePrompt(), simplePromptHandler())
	mcpServer.AddResource(staticTextResource(), staticTextResourceHandler())
}

// simpleTextTool defines the MCP conformance tool schema for simple text output.
func simpleTextTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "test_simple_text",
		Description: "Conformance tool that returns a simple text response.",
	}
}

// simpleTextHandler returns a fixed text payload for conformance validation.
// TODO: Provide project-aware completion/tool examples once conformance fixtures map to Duality features.
func simpleTextHandler() mcp.ToolHandlerFor[struct{}, any] {
	return func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: simpleTextResponse},
			},
		}, nil, nil
	}
}

// errorContentTool defines the MCP conformance tool schema for error responses.
func errorContentTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "test_error_content",
		Description: "Conformance tool that returns an error response.",
	}
}

// errorContentHandler returns a fixed tool error payload for conformance validation.
func errorContentHandler() mcp.ToolHandlerFor[struct{}, any] {
	return func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: errorTextResponse},
			},
		}, nil, nil
	}
}

// errorHandlingTool defines the MCP conformance tool schema for tool error handling.
func errorHandlingTool() *mcp.Tool {
	return &mcp.Tool{
		Name:        "test_error_handling",
		Description: "Conformance tool that always returns a tool error.",
	}
}

// errorHandlingHandler returns a fixed tool error payload for conformance validation.
func errorHandlingHandler() mcp.ToolHandlerFor[struct{}, any] {
	return func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			IsError: true,
			Content: []mcp.Content{
				&mcp.TextContent{Text: errorHandlingResponse},
			},
		}, nil, nil
	}
}

func simplePrompt() *mcp.Prompt {
	return &mcp.Prompt{
		Name:        "test_simple_prompt",
		Description: "Conformance prompt that returns a simple text message.",
	}
}

func simplePromptHandler() mcp.PromptHandler {
	return func(_ context.Context, _ *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		return &mcp.GetPromptResult{
			Messages: []*mcp.PromptMessage{
				{
					Role: "user",
					Content: &mcp.TextContent{
						Text: "This is a simple prompt for testing.",
					},
				},
			},
		}, nil
	}
}

// staticTextResource defines the MCP conformance resource schema for static text content.
func staticTextResource() *mcp.Resource {
	return &mcp.Resource{
		Name:        staticTextResourceName,
		Description: "Conformance resource that returns fixed text content.",
		MIMEType:    "text/plain",
		URI:         staticTextResourceURI,
	}
}

// staticTextResourceHandler returns fixed text content for conformance validation.
func staticTextResourceHandler() mcp.ResourceHandler {
	return func(_ context.Context, _ *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      staticTextResourceURI,
					MIMEType: "text/plain",
					Text:     staticTextResourceContent,
				},
			},
		}, nil
	}
}
