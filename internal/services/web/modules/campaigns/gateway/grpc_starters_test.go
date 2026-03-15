package gateway

import (
	"context"
	"net/http"
	"testing"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	discoveryv1 "github.com/louisbranch/fracturing.space/api/gen/go/discovery/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	campaignapp "github.com/louisbranch/fracturing.space/internal/services/web/modules/campaigns/app"
	apperrors "github.com/louisbranch/fracturing.space/internal/services/web/platform/errors"
	"google.golang.org/grpc"
)

type contractDiscoveryClient struct {
	resp    *discoveryv1.GetDiscoveryEntryResponse
	err     error
	lastReq *discoveryv1.GetDiscoveryEntryRequest
}

func (c *contractDiscoveryClient) GetDiscoveryEntry(_ context.Context, req *discoveryv1.GetDiscoveryEntryRequest, _ ...grpc.CallOption) (*discoveryv1.GetDiscoveryEntryResponse, error) {
	c.lastReq = req
	if c.err != nil {
		return nil, c.err
	}
	if c.resp != nil {
		return c.resp, nil
	}
	return &discoveryv1.GetDiscoveryEntryResponse{}, nil
}

type contractForkClient struct {
	resp    *statev1.ForkCampaignResponse
	err     error
	lastReq *statev1.ForkCampaignRequest
}

func (c *contractForkClient) ForkCampaign(_ context.Context, req *statev1.ForkCampaignRequest, _ ...grpc.CallOption) (*statev1.ForkCampaignResponse, error) {
	c.lastReq = req
	if c.err != nil {
		return nil, c.err
	}
	if c.resp != nil {
		return c.resp, nil
	}
	return &statev1.ForkCampaignResponse{}, nil
}

func TestNewStarterGatewayRequiresExplicitDependencies(t *testing.T) {
	t.Parallel()

	if got := NewStarterGateway(StarterDeps{}); got != nil {
		t.Fatalf("expected nil starter gateway for missing deps")
	}

	ready := NewStarterGateway(StarterDeps{
		Discovery: &contractDiscoveryClient{},
		Agent:     &contractAgentClient{},
		Campaign:  &contractCampaignClient{},
		Fork:      &contractForkClient{},
	})
	if ready == nil {
		t.Fatal("expected starter gateway when all deps are present")
	}
}

func TestStarterPreviewMapsEntryAndLaunchableAgents(t *testing.T) {
	t.Parallel()

	discoveryClient := &contractDiscoveryClient{resp: &discoveryv1.GetDiscoveryEntryResponse{
		Entry: &discoveryv1.DiscoveryEntry{
			EntryId:                    " starter:lantern-in-the-dark ",
			SourceId:                   " tmpl-1 ",
			Title:                      " The Lantern in the Dark ",
			Description:                " A tight mystery for one session. ",
			PreviewHook:                " A lantern appears on the black tide. ",
			PreviewPlaystyleLabel:      " Investigation ",
			PreviewCharacterName:       " Seren Vale ",
			PreviewCharacterSummary:    " A steadfast guardian chasing a vanished light. ",
			Storyline:                  " Follow the lantern into the wreck. ",
			System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
			DifficultyTier:             discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_BEGINNER,
			ExpectedDurationLabel:      " 1 session ",
			GmMode:                     discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_AI,
			RecommendedParticipantsMin: 1,
			RecommendedParticipantsMax: 1,
			Tags:                       []string{"mystery", "solo"},
		},
	}}
	agentClient := &contractAgentClient{listResp: &aiv1.ListAgentsResponse{Agents: []*aiv1.Agent{
		nil,
		{Id: "agent-ready", Label: "Ready GM", Status: aiv1.AgentStatus_AGENT_STATUS_ACTIVE, AuthState: aiv1.AgentAuthState_AGENT_AUTH_STATE_READY},
		{Id: "agent-unavailable", Label: "Unavailable GM", Status: aiv1.AgentStatus_AGENT_STATUS_ACTIVE, AuthState: aiv1.AgentAuthState_AGENT_AUTH_STATE_AUTH_REFERENCE_UNAVAILABLE},
		{Id: "", Label: "Missing ID", Status: aiv1.AgentStatus_AGENT_STATUS_ACTIVE, AuthState: aiv1.AgentAuthState_AGENT_AUTH_STATE_READY},
	}}}

	gateway := NewStarterGateway(StarterDeps{
		Discovery: discoveryClient,
		Agent:     agentClient,
		Campaign:  &contractCampaignClient{},
		Fork:      &contractForkClient{},
	})

	preview, err := gateway.StarterPreview(context.Background(), " starter:lantern-in-the-dark ")
	if err != nil {
		t.Fatalf("StarterPreview() error = %v", err)
	}
	if discoveryClient.lastReq == nil || discoveryClient.lastReq.GetEntryId() != "starter:lantern-in-the-dark" {
		t.Fatalf("GetDiscoveryEntry request = %#v", discoveryClient.lastReq)
	}
	if agentClient.lastListReq == nil || agentClient.lastListReq.GetPageSize() != 50 {
		t.Fatalf("ListAgents request = %#v", agentClient.lastListReq)
	}
	if preview.EntryID != "starter:lantern-in-the-dark" || preview.TemplateCampaignID != "tmpl-1" {
		t.Fatalf("preview ids = %#v", preview)
	}
	if preview.Title != "The Lantern in the Dark" || preview.Description != "A tight mystery for one session." {
		t.Fatalf("preview titles = %#v", preview)
	}
	if preview.System != "Daggerheart" || preview.Difficulty != "Beginner" || preview.GmMode != "AI" {
		t.Fatalf("preview labels = %#v", preview)
	}
	if preview.Players != "1" || preview.Duration != "1 session" {
		t.Fatalf("players/duration = %#v", preview)
	}
	if !preview.HasAvailableAIAgents {
		t.Fatal("expected available AI agents")
	}
	if len(preview.AIAgentOptions) != 2 {
		t.Fatalf("len(AIAgentOptions) = %d, want 2", len(preview.AIAgentOptions))
	}
	if !preview.AIAgentOptions[0].Enabled || preview.AIAgentOptions[1].Enabled {
		t.Fatalf("agent options = %#v", preview.AIAgentOptions)
	}
}

func TestLaunchStarterForksTemplateAndBindsSelectedAgent(t *testing.T) {
	t.Parallel()

	discoveryClient := &contractDiscoveryClient{resp: &discoveryv1.GetDiscoveryEntryResponse{
		Entry: &discoveryv1.DiscoveryEntry{
			EntryId:  "starter:lantern-in-the-dark",
			SourceId: "tmpl-1",
			Title:    "The Lantern in the Dark",
		},
	}}
	agentClient := &contractAgentClient{listResp: &aiv1.ListAgentsResponse{Agents: []*aiv1.Agent{
		{Id: "agent-ready", Label: "Ready GM", Status: aiv1.AgentStatus_AGENT_STATUS_ACTIVE, AuthState: aiv1.AgentAuthState_AGENT_AUTH_STATE_READY},
		{Id: "agent-disabled", Label: "Disabled GM", Status: aiv1.AgentStatus_AGENT_STATUS_UNSPECIFIED, AuthState: aiv1.AgentAuthState_AGENT_AUTH_STATE_READY},
	}}}
	forkClient := &contractForkClient{resp: &statev1.ForkCampaignResponse{Campaign: &statev1.Campaign{Id: "camp-777"}}}
	campaignClient := &contractCampaignClient{}

	gateway := NewStarterGateway(StarterDeps{
		Discovery: discoveryClient,
		Agent:     agentClient,
		Campaign:  campaignClient,
		Fork:      forkClient,
	})

	result, err := gateway.LaunchStarter(context.Background(), "starter:lantern-in-the-dark", campaignapp.LaunchStarterInput{AIAgentID: "agent-ready"})
	if err != nil {
		t.Fatalf("LaunchStarter() error = %v", err)
	}
	if result.CampaignID != "camp-777" {
		t.Fatalf("result.CampaignID = %q, want %q", result.CampaignID, "camp-777")
	}
	if forkClient.lastReq == nil {
		t.Fatal("expected fork request")
	}
	if forkClient.lastReq.GetSourceCampaignId() != "tmpl-1" || !forkClient.lastReq.GetCopyParticipants() {
		t.Fatalf("fork request = %#v", forkClient.lastReq)
	}
	if forkClient.lastReq.GetNewCampaignName() != "The Lantern in the Dark" {
		t.Fatalf("new campaign name = %q, want %q", forkClient.lastReq.GetNewCampaignName(), "The Lantern in the Dark")
	}
	if campaignClient.lastSetAIBindingReq == nil {
		t.Fatal("expected AI binding request")
	}
	if campaignClient.lastSetAIBindingReq.GetCampaignId() != "camp-777" || campaignClient.lastSetAIBindingReq.GetAiAgentId() != "agent-ready" {
		t.Fatalf("AI binding request = %#v", campaignClient.lastSetAIBindingReq)
	}
}

func TestLaunchStarterRejectsUnavailableTemplate(t *testing.T) {
	t.Parallel()

	gateway := NewStarterGateway(StarterDeps{
		Discovery: &contractDiscoveryClient{resp: &discoveryv1.GetDiscoveryEntryResponse{
			Entry: &discoveryv1.DiscoveryEntry{EntryId: "starter:lantern"},
		}},
		Agent:    &contractAgentClient{},
		Campaign: &contractCampaignClient{},
		Fork:     &contractForkClient{},
	})

	_, err := gateway.LaunchStarter(context.Background(), "starter:lantern", campaignapp.LaunchStarterInput{AIAgentID: "agent-ready"})
	if err == nil {
		t.Fatal("expected LaunchStarter() error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusServiceUnavailable {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusServiceUnavailable)
	}
}

func TestLaunchStarterRejectsUnavailableAgentSelection(t *testing.T) {
	t.Parallel()

	gateway := NewStarterGateway(StarterDeps{
		Discovery: &contractDiscoveryClient{resp: &discoveryv1.GetDiscoveryEntryResponse{
			Entry: &discoveryv1.DiscoveryEntry{EntryId: "starter:lantern", SourceId: "tmpl-1", Title: "The Lantern"},
		}},
		Agent: &contractAgentClient{listResp: &aiv1.ListAgentsResponse{Agents: []*aiv1.Agent{
			{Id: "agent-disabled", Label: "Disabled GM", Status: aiv1.AgentStatus_AGENT_STATUS_UNSPECIFIED, AuthState: aiv1.AgentAuthState_AGENT_AUTH_STATE_READY},
		}}},
		Campaign: &contractCampaignClient{},
		Fork:     &contractForkClient{},
	})

	_, err := gateway.LaunchStarter(context.Background(), "starter:lantern", campaignapp.LaunchStarterInput{AIAgentID: "agent-disabled"})
	if err == nil {
		t.Fatal("expected LaunchStarter() error")
	}
	if got := apperrors.HTTPStatus(err); got != http.StatusBadRequest {
		t.Fatalf("HTTPStatus(err) = %d, want %d", got, http.StatusBadRequest)
	}
}

func TestStarterGatewayLabelsAndPlayerRanges(t *testing.T) {
	t.Parallel()

	if got := starterDifficultyLabel(discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_INTERMEDIATE); got != "Intermediate" {
		t.Fatalf("starterDifficultyLabel(intermediate) = %q", got)
	}
	if got := starterDifficultyLabel(discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_ADVANCED); got != "Advanced" {
		t.Fatalf("starterDifficultyLabel(advanced) = %q", got)
	}
	if got := starterDifficultyLabel(discoveryv1.DiscoveryDifficultyTier_DISCOVERY_DIFFICULTY_TIER_UNSPECIFIED); got != "" {
		t.Fatalf("starterDifficultyLabel(unspecified) = %q, want empty", got)
	}
	if got := starterGMModeLabel(discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_HUMAN); got != "Human" {
		t.Fatalf("starterGMModeLabel(human) = %q", got)
	}
	if got := starterGMModeLabel(discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_HYBRID); got != "Hybrid" {
		t.Fatalf("starterGMModeLabel(hybrid) = %q", got)
	}
	if got := starterGMModeLabel(discoveryv1.DiscoveryGmMode_DISCOVERY_GM_MODE_UNSPECIFIED); got != "" {
		t.Fatalf("starterGMModeLabel(unspecified) = %q, want empty", got)
	}
	if got := starterGameSystemLabel(commonv1.GameSystem_GAME_SYSTEM_UNSPECIFIED); got != "" {
		t.Fatalf("starterGameSystemLabel(unspecified) = %q, want empty", got)
	}
	if got := starterPlayersLabel(1, 3); got != "1-3" {
		t.Fatalf("starterPlayersLabel(1,3) = %q", got)
	}
	if got := starterPlayersLabel(2, 0); got != "2+" {
		t.Fatalf("starterPlayersLabel(2,0) = %q", got)
	}
	if got := starterPlayersLabel(0, 4); got != "up to 4" {
		t.Fatalf("starterPlayersLabel(0,4) = %q", got)
	}
	if got := starterPlayersLabel(0, 0); got != "" {
		t.Fatalf("starterPlayersLabel(0,0) = %q, want empty", got)
	}
	if containsEnabledAIAgent(nil, "agent-ready") {
		t.Fatal("expected nil options to reject AI agent")
	}
	if containsEnabledAIAgent([]campaignapp.CampaignAIAgentOption{{ID: "agent-ready", Enabled: false}}, "agent-ready") {
		t.Fatal("expected disabled AI agent to be rejected")
	}
}
