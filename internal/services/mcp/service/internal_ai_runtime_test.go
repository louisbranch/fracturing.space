package service

import (
	"context"
	"errors"
	"net/http"
	"reflect"
	"strings"
	"testing"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	platformgrpc "github.com/louisbranch/fracturing.space/internal/platform/grpc"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/httptransport"
	"github.com/louisbranch/fracturing.space/internal/services/mcp/sessionctx"
	"github.com/louisbranch/fracturing.space/internal/services/shared/mcpbridge"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestMCPInternalAIRegistrationModules(t *testing.T) {
	server := &Server{}
	modules := newInternalAIRegistrationModules(
		server,
		mcpRegistrationClients{
			campaignArtifactClient: aiv1.NewCampaignArtifactServiceClient(nil),
			systemReferenceClient:  aiv1.NewSystemReferenceServiceClient(nil),
		},
		nil,
	)

	wantNames := []string{
		mcpDaggerheartToolsModuleName,
		mcpSceneToolsModuleName,
		mcpInteractionToolsModuleName,
		mcpCampaignResourceModuleName,
		mcpSessionResourceModuleName,
		mcpSceneResourceModuleName,
		mcpInteractionResourceModuleName,
		mcpContextResourceModuleName,
		"campaign-context-tools",
		"campaign-context-resources",
	}
	gotNames := make([]string, 0, len(modules))
	for _, module := range modules {
		gotNames = append(gotNames, module.name)
	}
	if !reflect.DeepEqual(gotNames, wantNames) {
		t.Fatalf("internal AI registration modules = %v, want %v", gotNames, wantNames)
	}
}

func TestMCPInternalAIRegistrationModulesAreIsolated(t *testing.T) {
	server := &Server{}
	modules := newInternalAIRegistrationModules(
		server,
		mcpRegistrationClients{
			campaignArtifactClient: aiv1.NewCampaignArtifactServiceClient(nil),
			systemReferenceClient:  aiv1.NewSystemReferenceServiceClient(nil),
		},
		nil,
	)

	expectedCounts := map[string]int{
		mcpDaggerheartToolsModuleName:    6,
		mcpSceneToolsModuleName:          1,
		mcpInteractionToolsModuleName:    11,
		mcpCampaignResourceModuleName:    3,
		mcpSessionResourceModuleName:     1,
		mcpSceneResourceModuleName:       1,
		mcpInteractionResourceModuleName: 1,
		mcpContextResourceModuleName:     1,
		"campaign-context-tools":         5,
		"campaign-context-resources":     2,
	}

	for _, module := range modules {
		module := module
		t.Run(module.name, func(t *testing.T) {
			target := &fakeMCPRegistrationTarget{}
			if err := module.register(target); err != nil {
				t.Fatalf("module registration failed for %q: %v", module.name, err)
			}

			if len(target.tools)+len(target.resourceTemplates)+len(target.resources) == 0 {
				t.Fatalf("module %q did not register anything", module.name)
			}

			switch module.kind {
			case mcpRegistrationKindTools:
				if len(target.tools) == 0 {
					t.Fatalf("tool module %q has no tools", module.name)
				}
				if len(target.resourceTemplates) != 0 || len(target.resources) != 0 {
					t.Fatalf("tool module %q also registered resources", module.name)
				}
			case mcpRegistrationKindResources:
				if len(target.tools) != 0 {
					t.Fatalf("resource module %q also registered tools", module.name)
				}
			default:
				t.Fatalf("unexpected module kind %v", module.kind)
			}

			if got, want := len(target.tools)+len(target.resourceTemplates)+len(target.resources), expectedCounts[module.name]; got != want {
				t.Fatalf("module %q registered %d items, want %d", module.name, got, want)
			}
		})
	}
}

func TestRegisterInternalAIInteractionToolsRegistersRestrictedToolNames(t *testing.T) {
	target := &fakeMCPRegistrationTarget{}

	if err := registerInternalAIInteractionTools(target, nil, func() sessionctx.Context { return sessionctx.Context{} }, nil); err != nil {
		t.Fatalf("registerInternalAIInteractionTools() error = %v", err)
	}

	want := []string{
		"interaction_active_scene_set",
		"interaction_scene_player_phase_start",
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
		t.Fatalf("internal AI interaction tools = %v, want %v", target.tools, want)
	}
}

func TestRegisterInternalAICampaignResourcesAddsScopedTemplates(t *testing.T) {
	target := &fakeMCPRegistrationTarget{}

	registerInternalAICampaignResources(target, nil, nil, nil, func() sessionctx.Context { return sessionctx.Context{} })

	want := []string{
		"campaign://{campaign_id}",
		"campaign://{campaign_id}/participants",
		"campaign://{campaign_id}/characters",
	}
	if !reflect.DeepEqual(target.resourceTemplates, want) {
		t.Fatalf("internal AI campaign resources = %v, want %v", target.resourceTemplates, want)
	}
}

func TestNewHarnessConfiguresHarnessProfile(t *testing.T) {
	server, err := NewHarness("localhost:8080")
	if err != nil {
		t.Fatalf("NewHarness() error = %v", err)
	}
	t.Cleanup(func() {
		_ = server.Close()
	})

	if server.profile != mcpRegistrationProfileHarness {
		t.Fatalf("server profile = %v, want %v", server.profile, mcpRegistrationProfileHarness)
	}
}

func TestNewServerWithAIConnBuildsStandardProfile(t *testing.T) {
	server, err := newServerWithAIConn(nil, nil)
	if err != nil {
		t.Fatalf("newServerWithAIConn() error = %v", err)
	}

	if server.profile != mcpRegistrationProfileStandard {
		t.Fatalf("server profile = %v, want %v", server.profile, mcpRegistrationProfileStandard)
	}
}

func TestNewInternalAISessionServerSeedsFixedContext(t *testing.T) {
	server, err := newInternalAISessionServer(nil, nil, mcpbridge.SessionContext{
		CampaignID:    "camp-1",
		SessionID:     "session-1",
		ParticipantID: "participant-1",
	})
	if err != nil {
		t.Fatalf("newInternalAISessionServer() error = %v", err)
	}

	if server.profile != mcpRegistrationProfileInternalAI {
		t.Fatalf("server profile = %v, want %v", server.profile, mcpRegistrationProfileInternalAI)
	}
	want := sessionctx.Context{
		CampaignID:    "camp-1",
		SessionID:     "session-1",
		ParticipantID: "participant-1",
	}
	if got := server.getContext(); got != want {
		t.Fatalf("server context = %+v, want %+v", got, want)
	}
}

func TestHTTPTransportRuntimeFactoryRejectsStandardBootstrapWithoutBridgeHeaders(t *testing.T) {
	factory := httpTransportRuntimeFactory{
		runtime: &Server{
			profile: mcpRegistrationProfileStandard,
			gameMc:  &platformgrpc.ManagedConn{},
		},
	}

	_, err := factory.NewSessionRuntime(http.Header{})
	if !errors.Is(err, httptransport.ErrSessionBootstrapRejected) {
		t.Fatalf("NewSessionRuntime() error = %v, want %v", err, httptransport.ErrSessionBootstrapRejected)
	}
}

func TestHTTPTransportRuntimeFactoryAllowsHarnessBootstrapWithoutBridgeHeaders(t *testing.T) {
	factory := httpTransportRuntimeFactory{
		runtime: &Server{
			profile: mcpRegistrationProfileHarness,
			gameMc:  &platformgrpc.ManagedConn{},
		},
	}

	runtime, err := factory.NewSessionRuntime(http.Header{})
	if err != nil {
		t.Fatalf("NewSessionRuntime() error = %v", err)
	}
	if runtime == nil {
		t.Fatal("expected session runtime")
	}
}

func TestServeWithTransportRequiresTransport(t *testing.T) {
	server := &Server{
		mcpServer: mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1.0.0"}, nil),
	}

	err := server.ServeWithTransport(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "transport is required") {
		t.Fatalf("ServeWithTransport() error = %v, want transport required", err)
	}
}

func TestRunRejectsInvalidRegistrationProfile(t *testing.T) {
	err := Run(context.Background(), Config{RegistrationProfile: RegistrationProfile("invalid")})
	if err == nil || !strings.Contains(err.Error(), "unsupported MCP registration profile") {
		t.Fatalf("Run() error = %v, want unsupported profile", err)
	}
}

func TestAIAddress(t *testing.T) {
	t.Run("fallback wins", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_AI_ADDR", "env:9090")
		if got := aiAddress(" fallback:8080 "); got != "fallback:8080" {
			t.Fatalf("aiAddress() = %q, want %q", got, "fallback:8080")
		}
	})

	t.Run("env used when fallback empty", func(t *testing.T) {
		t.Setenv("FRACTURING_SPACE_AI_ADDR", "env:9090")
		if got := aiAddress(""); got != "env:9090" {
			t.Fatalf("aiAddress() = %q, want %q", got, "env:9090")
		}
	})
}
