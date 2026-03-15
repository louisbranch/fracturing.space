package service

import (
	"context"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/mcp/domain"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/sessionctx"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestCompletionHandler(t *testing.T) {
	result, err := completionHandler(context.Background(), &mcp.CompleteRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if len(result.Completion.Values) != 0 {
		t.Errorf("expected empty values, got %v", result.Completion.Values)
	}
}

func TestResourceSubscribeHandler(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		if err := resourceSubscribeHandler(context.Background(), nil); err == nil {
			t.Fatal("expected error for nil request")
		}
	})

	t.Run("nil params", func(t *testing.T) {
		if err := resourceSubscribeHandler(context.Background(), &mcp.SubscribeRequest{}); err == nil {
			t.Fatal("expected error for nil params")
		}
	})

	t.Run("empty URI", func(t *testing.T) {
		if err := resourceSubscribeHandler(context.Background(), &mcp.SubscribeRequest{
			Params: &mcp.SubscribeParams{URI: ""},
		}); err == nil {
			t.Fatal("expected error for empty URI")
		}
	})

	t.Run("valid URI", func(t *testing.T) {
		if err := resourceSubscribeHandler(context.Background(), &mcp.SubscribeRequest{
			Params: &mcp.SubscribeParams{URI: "campaigns://list"},
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestResourceUnsubscribeHandler(t *testing.T) {
	t.Run("nil request", func(t *testing.T) {
		if err := resourceUnsubscribeHandler(context.Background(), nil); err == nil {
			t.Fatal("expected error for nil request")
		}
	})

	t.Run("valid URI", func(t *testing.T) {
		if err := resourceUnsubscribeHandler(context.Background(), &mcp.UnsubscribeRequest{
			Params: &mcp.UnsubscribeParams{URI: "campaigns://list"},
		}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestGrpcAddress(t *testing.T) {
	tests := []struct {
		name     string
		fallback string
		want     string
	}{
		{"uses fallback when provided", "localhost:50051", "localhost:50051"},
		{"uses fallback for whitespace", "  ", "  "},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := grpcAddress(tt.fallback); got != tt.want {
				t.Errorf("grpcAddress(%q) = %q, want %q", tt.fallback, got, tt.want)
			}
		})
	}
}

func TestServerContext(t *testing.T) {
	s := &Server{}

	t.Run("default context is empty", func(t *testing.T) {
		ctx := s.getContext()
		if ctx.CampaignID != "" || ctx.SessionID != "" || ctx.ParticipantID != "" {
			t.Errorf("expected empty context, got %+v", ctx)
		}
	})

	t.Run("set and get context", func(t *testing.T) {
		s.setContext(sessionctx.Context{CampaignID: "c1", SessionID: "s1"})
		ctx := s.getContext()
		if ctx.CampaignID != "c1" {
			t.Errorf("expected campaign_id %q, got %q", "c1", ctx.CampaignID)
		}
		if ctx.SessionID != "s1" {
			t.Errorf("expected session_id %q, got %q", "s1", ctx.SessionID)
		}
	})

	t.Run("nil server is safe", func(t *testing.T) {
		var nilServer *Server
		nilServer.setContext(sessionctx.Context{CampaignID: "x"})
		ctx := nilServer.getContext()
		if ctx.CampaignID != "" {
			t.Errorf("expected empty context from nil server, got %+v", ctx)
		}
	})
}

func TestServerClose(t *testing.T) {
	t.Run("nil server is safe", func(t *testing.T) {
		var s *Server
		if err := s.Close(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("nil conn is safe", func(t *testing.T) {
		s := &Server{}
		if err := s.Close(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestRegisterToolsNoPanic(t *testing.T) {
	// Verify all registration functions can be called without panic
	// when given a real MCP server and nil gRPC clients.
	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
	server := &Server{}

	t.Run("registerDaggerheartTools", func(t *testing.T) {
		err := registerDaggerheartTools(mcpServerRegistrationAdapter{server: mcpServer}, nil)
		if err != nil {
			t.Fatalf("registerDaggerheartTools: %v", err)
		}
	})

	t.Run("registerCampaignTools", func(t *testing.T) {
		err := registerCampaignTools(mcpServerRegistrationAdapter{server: mcpServer}, nil, nil, nil, nil, server.getContext, nil)
		if err != nil {
			t.Fatalf("registerCampaignTools: %v", err)
		}
	})

	t.Run("registerSessionTools", func(t *testing.T) {
		err := registerSessionTools(mcpServerRegistrationAdapter{server: mcpServer}, nil, server.getContext, nil)
		if err != nil {
			t.Fatalf("registerSessionTools: %v", err)
		}
	})

	t.Run("registerInteractionTools", func(t *testing.T) {
		err := registerInteractionTools(mcpServerRegistrationAdapter{server: mcpServer}, nil, server.getContext, nil)
		if err != nil {
			t.Fatalf("registerInteractionTools: %v", err)
		}
	})

	t.Run("registerEventTools", func(t *testing.T) {
		err := registerEventTools(mcpServerRegistrationAdapter{server: mcpServer}, nil, server.getContext)
		if err != nil {
			t.Fatalf("registerEventTools: %v", err)
		}
	})

	t.Run("registerForkTools", func(t *testing.T) {
		err := registerForkTools(mcpServerRegistrationAdapter{server: mcpServer}, nil, server.getContext, nil)
		if err != nil {
			t.Fatalf("registerForkTools: %v", err)
		}
	})

	t.Run("registerHarnessContextTools", func(t *testing.T) {
		err := registerHarnessContextTools(mcpServerRegistrationAdapter{server: mcpServer}, nil, nil, nil, server, nil)
		if err != nil {
			t.Fatalf("registerHarnessContextTools: %v", err)
		}
	})

	t.Run("registerCampaignResources", func(t *testing.T) {
		registerCampaignResources(mcpServerRegistrationAdapter{server: mcpServer}, nil, nil, nil)
	})

	t.Run("registerSessionResources", func(t *testing.T) {
		registerSessionResources(mcpServerRegistrationAdapter{server: mcpServer}, nil)
	})

	t.Run("registerInteractionResources", func(t *testing.T) {
		registerInteractionResources(mcpServerRegistrationAdapter{server: mcpServer}, nil, server.getContext)
	})

	t.Run("registerEventResources", func(t *testing.T) {
		registerEventResources(mcpServerRegistrationAdapter{server: mcpServer}, nil)
	})

	t.Run("registerContextResources", func(t *testing.T) {
		registerContextResources(mcpServerRegistrationAdapter{server: mcpServer}, server)
	})
}

type fakeMCPRegistrationTarget struct {
	tools             []string
	resourceTemplates []string
	resources         []string
}

func (f *fakeMCPRegistrationTarget) AddTool(tool *mcp.Tool, _ any) error {
	if tool != nil {
		f.tools = append(f.tools, tool.Name)
	}
	return nil
}

func (f *fakeMCPRegistrationTarget) AddResourceTemplate(resourceTemplate *mcp.ResourceTemplate, _ mcp.ResourceHandler) {
	if resourceTemplate != nil {
		f.resourceTemplates = append(f.resourceTemplates, resourceTemplate.URITemplate)
	}
}

func (f *fakeMCPRegistrationTarget) AddResource(resource *mcp.Resource, _ mcp.ResourceHandler) {
	if resource != nil {
		f.resources = append(f.resources, resource.URI)
	}
}

func TestMCPRegistrationModules(t *testing.T) {
	server := &Server{}
	modules := newMCPRegistrationModules(
		server,
		mcpRegistrationClients{},
		nil,
	)

	expectNames := []string{
		mcpDaggerheartToolsModuleName,
		mcpCampaignToolsModuleName,
		mcpSessionToolsModuleName,
		mcpSceneToolsModuleName,
		mcpInteractionToolsModuleName,
		mcpForkToolsModuleName,
		mcpEventToolsModuleName,
		mcpCampaignResourceModuleName,
		mcpSessionResourceModuleName,
		mcpSceneResourceModuleName,
		mcpInteractionResourceModuleName,
		mcpEventResourceModuleName,
		mcpContextResourceModuleName,
	}
	gotNames := make([]string, 0, len(modules))
	for _, module := range modules {
		gotNames = append(gotNames, module.name)
	}
	if !reflect.DeepEqual(gotNames, expectNames) {
		t.Fatalf("expected registration modules %v, got %v", expectNames, gotNames)
	}
}

func TestMCPHarnessRegistrationModules(t *testing.T) {
	server := &Server{}
	modules := newMCPRegistrationModules(
		server,
		mcpRegistrationClients{},
		mcpRegistrationProfileHarness,
	)

	expectNames := []string{
		mcpDaggerheartToolsModuleName,
		mcpCampaignToolsModuleName,
		mcpSessionToolsModuleName,
		mcpSceneToolsModuleName,
		mcpInteractionToolsModuleName,
		mcpForkToolsModuleName,
		mcpEventToolsModuleName,
		mcpHarnessContextToolsModuleName,
		mcpCampaignResourceModuleName,
		mcpSessionResourceModuleName,
		mcpSceneResourceModuleName,
		mcpInteractionResourceModuleName,
		mcpEventResourceModuleName,
		mcpContextResourceModuleName,
	}
	gotNames := make([]string, 0, len(modules))
	for _, module := range modules {
		gotNames = append(gotNames, module.name)
	}
	if !reflect.DeepEqual(gotNames, expectNames) {
		t.Fatalf("expected harness registration modules %v, got %v", expectNames, gotNames)
	}
}

func TestMCPRegistrationModulesAreIsolated(t *testing.T) {
	server := &Server{}
	modules := newMCPRegistrationModules(
		server,
		mcpRegistrationClients{},
		nil,
	)

	expectedCounts := map[string]int{
		mcpDaggerheartToolsModuleName:    6,
		mcpCampaignToolsModuleName:       15,
		mcpSessionToolsModuleName:        2,
		mcpSceneToolsModuleName:          1,
		mcpInteractionToolsModuleName:    14,
		mcpForkToolsModuleName:           2,
		mcpEventToolsModuleName:          1,
		mcpCampaignResourceModuleName:    4,
		mcpSessionResourceModuleName:     1,
		mcpSceneResourceModuleName:       1,
		mcpInteractionResourceModuleName: 1,
		mcpEventResourceModuleName:       1,
		mcpContextResourceModuleName:     1,
	}

	for _, module := range modules {
		module := module
		t.Run(module.name, func(t *testing.T) {
			fake := &fakeMCPRegistrationTarget{}
			if err := module.register(fake); err != nil {
				t.Fatalf("module registration failed for %q: %v", module.name, err)
			}

			if len(fake.tools)+len(fake.resourceTemplates)+len(fake.resources) == 0 {
				t.Fatalf("module %q did not register anything", module.name)
			}

			switch module.kind {
			case mcpRegistrationKindTools:
				if len(fake.tools) == 0 {
					t.Fatalf("tool module %q has no tools", module.name)
				}
				if len(fake.resourceTemplates) != 0 || len(fake.resources) != 0 {
					t.Fatalf("tool module %q also registered resources", module.name)
				}
			case mcpRegistrationKindResources:
				if len(fake.resourceTemplates) == 0 && len(fake.resources) == 0 {
					t.Fatalf("resource module %q has no resources", module.name)
				}
				if len(fake.tools) != 0 {
					t.Fatalf("resource module %q also registered tools", module.name)
				}
			default:
				t.Fatalf("unexpected module kind %v for %q", module.kind, module.name)
			}

			want := expectedCounts[module.name]
			if want == 0 {
				t.Fatalf("module %q missing from expectedCounts", module.name)
			}
			if got := len(fake.tools) + len(fake.resourceTemplates) + len(fake.resources); got != want {
				t.Fatalf("module %q registered %d items, expected %d", module.name, got, want)
			}
		})
	}
}

func TestMCPHarnessRegistrationModulesAreIsolated(t *testing.T) {
	server := &Server{}
	modules := newMCPRegistrationModules(
		server,
		mcpRegistrationClients{},
		mcpRegistrationProfileHarness,
	)

	expectedCounts := map[string]int{
		mcpDaggerheartToolsModuleName:    6,
		mcpCampaignToolsModuleName:       15,
		mcpSessionToolsModuleName:        2,
		mcpSceneToolsModuleName:          1,
		mcpInteractionToolsModuleName:    14,
		mcpForkToolsModuleName:           2,
		mcpEventToolsModuleName:          1,
		mcpHarnessContextToolsModuleName: 1,
		mcpCampaignResourceModuleName:    4,
		mcpSessionResourceModuleName:     1,
		mcpSceneResourceModuleName:       1,
		mcpInteractionResourceModuleName: 1,
		mcpEventResourceModuleName:       1,
		mcpContextResourceModuleName:     1,
	}

	for _, module := range modules {
		module := module
		t.Run(module.name, func(t *testing.T) {
			fake := &fakeMCPRegistrationTarget{}
			if err := module.register(fake); err != nil {
				t.Fatalf("module registration failed for %q: %v", module.name, err)
			}

			if len(fake.tools)+len(fake.resourceTemplates)+len(fake.resources) == 0 {
				t.Fatalf("module %q did not register anything", module.name)
			}

			switch module.kind {
			case mcpRegistrationKindTools:
				if len(fake.tools) == 0 {
					t.Fatalf("tool module %q has no tools", module.name)
				}
				if len(fake.resourceTemplates) != 0 || len(fake.resources) != 0 {
					t.Fatalf("tool module %q also registered resources", module.name)
				}
			case mcpRegistrationKindResources:
				if len(fake.resourceTemplates) == 0 && len(fake.resources) == 0 {
					t.Fatalf("resource module %q has no resources", module.name)
				}
				if len(fake.tools) != 0 {
					t.Fatalf("resource module %q also registered tools", module.name)
				}
			default:
				t.Fatalf("unexpected module kind %v for %q", module.kind, module.name)
			}

			want := expectedCounts[module.name]
			if want == 0 {
				t.Fatalf("module %q missing from expectedCounts", module.name)
			}
			if got := len(fake.tools) + len(fake.resourceTemplates) + len(fake.resources); got != want {
				t.Fatalf("module %q registered %d items, expected %d", module.name, got, want)
			}
		})
	}
}

type failingMCPRegistrationTarget struct{}

func (failingMCPRegistrationTarget) AddTool(*mcp.Tool, any) error {
	return fmt.Errorf("boom")
}

func (failingMCPRegistrationTarget) AddResourceTemplate(*mcp.ResourceTemplate, mcp.ResourceHandler) {}

func (failingMCPRegistrationTarget) AddResource(*mcp.Resource, mcp.ResourceHandler) {}

func TestRegisterToolRejectsNilTool(t *testing.T) {
	if err := registerTool(&fakeMCPRegistrationTarget{}, nil, struct{}{}); err == nil {
		t.Fatal("expected nil tool error")
	}
}

func TestRegisterInteractionToolsRegistersAllInteractionToolNames(t *testing.T) {
	target := &fakeMCPRegistrationTarget{}

	if err := registerInteractionTools(target, nil, func() sessionctx.Context { return sessionctx.Context{} }, nil); err != nil {
		t.Fatalf("registerInteractionTools() error = %v", err)
	}

	want := []string{
		"interaction_active_scene_set",
		"interaction_scene_player_phase_start",
		"interaction_scene_player_post_submit",
		"interaction_scene_player_phase_yield",
		"interaction_scene_player_phase_unyield",
		"interaction_scene_gm_output_commit",
		"interaction_scene_player_phase_accept",
		"interaction_scene_player_revisions_request",
		"interaction_scene_player_phase_end",
		"interaction_ooc_pause",
		"interaction_ooc_post",
		"interaction_ooc_ready_mark",
		"interaction_ooc_ready_clear",
		"interaction_ooc_resume",
	}
	if !reflect.DeepEqual(target.tools, want) {
		t.Fatalf("interaction tools = %v, want %v", target.tools, want)
	}
}

func TestRegisterInteractionResourcesAddsInteractionResourceTemplate(t *testing.T) {
	target := &fakeMCPRegistrationTarget{}

	registerInteractionResources(target, nil, func() sessionctx.Context { return sessionctx.Context{} })

	if !reflect.DeepEqual(target.resourceTemplates, []string{"campaign://{campaign_id}/interaction"}) {
		t.Fatalf("resource templates = %v", target.resourceTemplates)
	}
}

func TestRegisterSceneToolsAddsSceneCreateTool(t *testing.T) {
	target := &fakeMCPRegistrationTarget{}

	if err := registerSceneTools(target, nil, func() sessionctx.Context { return sessionctx.Context{} }, nil); err != nil {
		t.Fatalf("registerSceneTools() error = %v", err)
	}
	if !reflect.DeepEqual(target.tools, []string{"scene_create"}) {
		t.Fatalf("scene tools = %v", target.tools)
	}
}

func TestRegisterSceneResourcesAddsSceneResourceTemplate(t *testing.T) {
	target := &fakeMCPRegistrationTarget{}

	registerSceneResources(target, nil)

	if !reflect.DeepEqual(target.resourceTemplates, []string{"campaign://{campaign_id}/sessions/{session_id}/scenes"}) {
		t.Fatalf("resource templates = %v", target.resourceTemplates)
	}
}

func TestRegisterToolPropagatesRegistrarError(t *testing.T) {
	if err := registerTool(failingMCPRegistrationTarget{}, domain.InteractionSetActiveSceneTool(), struct{}{}); err == nil {
		t.Fatal("expected registrar error")
	}
}

func TestAddMCPToolErrorsOnUnsupportedHandler(t *testing.T) {
	t.Run("unsupported handler type returns an error", func(t *testing.T) {
		srv := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0"}, nil)
		err := addMCPTool(srv, domain.CampaignCreateTool(), func() {})
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "mcp registration adapter does not support handler type") {
			t.Fatalf("unexpected error message: %q", err.Error())
		}
	})
}

func TestGrpcAddressFromEnv(t *testing.T) {
	t.Run("reads from env when fallback is empty", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_GAME_ADDR", "custom:9999")
		got := grpcAddress("")
		if got != "custom:9999" {
			t.Errorf("expected %q, got %q", "custom:9999", got)
		}
	})

	t.Run("fallback takes precedence over env", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_GAME_ADDR", "custom:9999")
		got := grpcAddress("default:50051")
		if got != "default:50051" {
			t.Errorf("expected %q, got %q", "default:50051", got)
		}
	})

	t.Run("empty fallback and empty env returns fallback", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_GAME_ADDR", "")
		got := grpcAddress("")
		if got != "" {
			t.Errorf("expected empty string, got %q", got)
		}
	})
}
