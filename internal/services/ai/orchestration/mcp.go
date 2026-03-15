package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/shared/mcpbridge"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.opentelemetry.io/otel/attribute"
)

type mcpDialer struct {
	url         string
	http        *http.Client
	dialTimeout time.Duration
}

type mcpSession struct {
	sess *mcp.ClientSession
}

// NewMCPDialer builds one streamable-HTTP MCP dialer for AI orchestration.
func NewMCPDialer(cfg MCPDialerConfig) Dialer {
	client := cfg.HTTPClient
	if client == nil {
		client = http.DefaultClient
	}
	return &mcpDialer{
		url:         strings.TrimSpace(cfg.URL),
		http:        client,
		dialTimeout: cfg.DialTimeout,
	}
}

func (d *mcpDialer) Dial(ctx context.Context) (Session, error) {
	ctx, span := tracer.Start(ctx, "ai.orchestration.mcp.dial")
	defer span.End()
	if d == nil || strings.TrimSpace(d.url) == "" {
		err := fmt.Errorf("mcp url is required")
		recordSpanError(span, err)
		return nil, err
	}
	httpClient := d.http
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	httpClient = withBridgeContextHTTPClient(httpClient)
	dialCtx := ctx
	if d.dialTimeout > 0 {
		var cancel context.CancelFunc
		dialCtx, cancel = context.WithTimeout(ctx, d.dialTimeout)
		defer cancel()
		span.SetAttributes(attribute.Int64("ai.orchestration.mcp.dial_timeout_ms", d.dialTimeout.Milliseconds()))
	}
	cli := mcp.NewClient(&mcp.Implementation{Name: "fracturing-space-ai", Version: "0.1.0"}, nil)
	sess, err := cli.Connect(dialCtx, &mcp.StreamableClientTransport{
		Endpoint:   d.url,
		HTTPClient: httpClient,
	}, nil)
	if err != nil {
		recordSpanError(span, err)
		return nil, err
	}
	return &mcpSession{sess: sess}, nil
}

func (s *mcpSession) ListTools(ctx context.Context) ([]Tool, error) {
	ctx, span := tracer.Start(ctx, "ai.orchestration.mcp.list_tools")
	defer span.End()
	cursor := ""
	items := make([]Tool, 0, 16)
	for {
		res, err := s.sess.ListTools(ctx, &mcp.ListToolsParams{Cursor: cursor})
		if err != nil {
			recordSpanError(span, err)
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
			span.SetAttributes(attribute.Int("ai.orchestration.mcp.tool_count", len(items)))
			return items, nil
		}
	}
}

func (s *mcpSession) CallTool(ctx context.Context, name string, args any) (ToolResult, error) {
	ctx, span := tracer.Start(ctx, "ai.orchestration.mcp.call_tool")
	defer span.End()
	span.SetAttributes(attribute.String("ai.orchestration.tool_name", strings.TrimSpace(name)))
	res, err := s.sess.CallTool(ctx, &mcp.CallToolParams{
		Name:      strings.TrimSpace(name),
		Arguments: args,
	})
	if err != nil {
		recordSpanError(span, err)
		return ToolResult{}, err
	}
	result := ToolResult{
		Output:  toolOutput(res),
		IsError: res.IsError,
	}
	span.SetAttributes(attribute.Bool("ai.orchestration.tool_error", result.IsError))
	return result, nil
}

func (s *mcpSession) ReadResource(ctx context.Context, uri string) (string, error) {
	ctx, span := tracer.Start(ctx, "ai.orchestration.mcp.read_resource")
	defer span.End()
	span.SetAttributes(attribute.String("ai.orchestration.resource_uri", strings.TrimSpace(uri)))
	res, err := s.sess.ReadResource(ctx, &mcp.ReadResourceParams{URI: strings.TrimSpace(uri)})
	if err != nil {
		recordSpanError(span, err)
		return "", err
	}
	if len(res.Contents) == 0 || res.Contents[0] == nil {
		err := fmt.Errorf("resource %q returned no contents", uri)
		recordSpanError(span, err)
		return "", err
	}
	item := res.Contents[0]
	if strings.TrimSpace(item.Text) != "" {
		span.SetAttributes(attribute.Int("ai.orchestration.resource_bytes", len(item.Text)))
		return item.Text, nil
	}
	if len(item.Blob) != 0 {
		span.SetAttributes(attribute.Int("ai.orchestration.resource_bytes", len(item.Blob)))
		return string(item.Blob), nil
	}
	err = fmt.Errorf("resource %q returned empty contents", uri)
	recordSpanError(span, err)
	return "", err
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

func withBridgeContextHTTPClient(base *http.Client) *http.Client {
	if base == nil {
		base = http.DefaultClient
	}
	clone := *base
	transport := clone.Transport
	if transport == nil {
		transport = http.DefaultTransport
	}
	clone.Transport = bridgeContextRoundTripper{base: transport}
	return &clone
}

type bridgeContextRoundTripper struct {
	base http.RoundTripper
}

func (rt bridgeContextRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	base := rt.base
	if base == nil {
		base = http.DefaultTransport
	}
	if req == nil {
		return base.RoundTrip(req)
	}
	sessionCtx := mcpbridge.SessionContextFromContext(req.Context())
	if sessionCtx.Valid() {
		req = req.Clone(req.Context())
		sessionCtx.ApplyToHeader(req.Header)
	}
	return base.RoundTrip(req)
}
