package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mcpDialer struct {
	url  string
	http *http.Client
}

type mcpSession struct {
	sess *mcp.ClientSession
}

// NewMCPDialer builds one streamable-HTTP MCP dialer for AI orchestration.
func NewMCPDialer(url string, client *http.Client) Dialer {
	if client == nil {
		client = http.DefaultClient
	}
	return &mcpDialer{
		url:  strings.TrimSpace(url),
		http: client,
	}
}

func (d *mcpDialer) Dial(ctx context.Context) (Session, error) {
	if d == nil || strings.TrimSpace(d.url) == "" {
		return nil, fmt.Errorf("mcp url is required")
	}
	cli := mcp.NewClient(&mcp.Implementation{Name: "fracturing-space-ai", Version: "0.1.0"}, nil)
	sess, err := cli.Connect(ctx, &mcp.StreamableClientTransport{
		Endpoint:   d.url,
		HTTPClient: d.http,
	}, nil)
	if err != nil {
		return nil, err
	}
	return &mcpSession{sess: sess}, nil
}

func (s *mcpSession) ListTools(ctx context.Context) ([]Tool, error) {
	cursor := ""
	items := make([]Tool, 0, 16)
	for {
		res, err := s.sess.ListTools(ctx, &mcp.ListToolsParams{Cursor: cursor})
		if err != nil {
			return nil, err
		}
		for _, item := range res.Tools {
			if item == nil {
				continue
			}
			items = append(items, Tool{
				Name:        strings.TrimSpace(item.Name),
				Description: strings.TrimSpace(item.Description),
				InputSchema: item.InputSchema,
			})
		}
		cursor = strings.TrimSpace(res.NextCursor)
		if cursor == "" {
			return items, nil
		}
	}
}

func (s *mcpSession) CallTool(ctx context.Context, name string, args any) (ToolResult, error) {
	res, err := s.sess.CallTool(ctx, &mcp.CallToolParams{
		Name:      strings.TrimSpace(name),
		Arguments: args,
	})
	if err != nil {
		return ToolResult{}, err
	}
	return ToolResult{
		Output:  toolOutput(res),
		IsError: res.IsError,
	}, nil
}

func (s *mcpSession) ReadResource(ctx context.Context, uri string) (string, error) {
	res, err := s.sess.ReadResource(ctx, &mcp.ReadResourceParams{URI: strings.TrimSpace(uri)})
	if err != nil {
		return "", err
	}
	if len(res.Contents) == 0 || res.Contents[0] == nil {
		return "", fmt.Errorf("resource %q returned no contents", uri)
	}
	item := res.Contents[0]
	if strings.TrimSpace(item.Text) != "" {
		return item.Text, nil
	}
	if len(item.Blob) != 0 {
		return string(item.Blob), nil
	}
	return "", fmt.Errorf("resource %q returned empty contents", uri)
}

func (s *mcpSession) Close() error {
	if s == nil || s.sess == nil {
		return nil
	}
	return s.sess.Close()
}

func toolOutput(res *mcp.CallToolResult) string {
	if res == nil {
		return "{}"
	}
	if res.StructuredContent != nil {
		data, err := json.Marshal(res.StructuredContent)
		if err == nil {
			return string(data)
		}
	}
	texts := make([]string, 0, len(res.Content))
	for _, item := range res.Content {
		switch value := item.(type) {
		case *mcp.TextContent:
			if strings.TrimSpace(value.Text) != "" {
				texts = append(texts, value.Text)
			}
		case *mcp.EmbeddedResource:
			if value.Resource != nil && strings.TrimSpace(value.Resource.Text) != "" {
				texts = append(texts, value.Resource.Text)
			}
		}
	}
	if len(texts) != 0 {
		return strings.Join(texts, "\n")
	}
	data, err := json.Marshal(res)
	if err != nil {
		return "{}"
	}
	return string(data)
}
