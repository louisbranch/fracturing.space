package openviking

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

type resourceSearchClientStub struct {
	added       []AddResourceInput
	finds       []SearchInput
	searches    []SearchInput
	result      SearchResult
	findResults map[string]SearchResult
	addErr      error
	findErr     error
	session     SessionInfo
	sessionErr  error
	overview    map[string]string
	read        map[string]string
	tree        map[string][]FilesystemEntry
}

func (s *resourceSearchClientStub) AddResource(_ context.Context, input AddResourceInput) (AddResourceResult, error) {
	s.added = append(s.added, input)
	if s.addErr != nil {
		return AddResourceResult{}, s.addErr
	}
	return AddResourceResult{
		Status:  "success",
		RootURI: strings.TrimSpace(input.To),
	}, nil
}

func (s *resourceSearchClientStub) Find(_ context.Context, input SearchInput) (SearchResult, error) {
	s.finds = append(s.finds, input)
	if s.findErr != nil {
		return SearchResult{}, s.findErr
	}
	if s.findResults != nil {
		if result, ok := s.findResults[input.TargetURI]; ok {
			return result, nil
		}
	}
	return s.result, nil
}

func (s *resourceSearchClientStub) Search(_ context.Context, input SearchInput) (SearchResult, error) {
	s.searches = append(s.searches, input)
	if s.findErr != nil {
		return SearchResult{}, s.findErr
	}
	return s.result, nil
}

func (s *resourceSearchClientStub) GetSession(context.Context, string) (SessionInfo, error) {
	if s.sessionErr != nil {
		return SessionInfo{}, s.sessionErr
	}
	return s.session, nil
}

func (s *resourceSearchClientStub) Overview(_ context.Context, uri string) (string, error) {
	if s.overview == nil {
		return "", nil
	}
	return s.overview[uri], nil
}

func (s *resourceSearchClientStub) Read(_ context.Context, uri string) (string, error) {
	if s.read == nil {
		return "", nil
	}
	return s.read[uri], nil
}

func (s *resourceSearchClientStub) Tree(_ context.Context, uri string) ([]FilesystemEntry, error) {
	if s.tree == nil {
		return nil, nil
	}
	return s.tree[uri], nil
}

type sessionStub struct {
	resources map[string]string
}

func (s sessionStub) ListTools(context.Context) ([]orchestration.Tool, error) { return nil, nil }
func (s sessionStub) CallTool(context.Context, string, any) (orchestration.ToolResult, error) {
	return orchestration.ToolResult{}, nil
}
func (s sessionStub) ReadResource(_ context.Context, uri string) (string, error) {
	return s.resources[uri], nil
}
func (s sessionStub) Close() error { return nil }

type traceRecorderStub struct {
	contexts     []orchestration.RetrievedContext
	policy       orchestration.PromptContextPolicy
	augmentation orchestration.PromptAugmentationDiagnostics
}

func (s *traceRecorderStub) RecordRetrievedContexts(contexts []orchestration.RetrievedContext) {
	s.contexts = append(s.contexts, contexts...)
}

func (s *traceRecorderStub) RecordPromptContextPolicy(policy orchestration.PromptContextPolicy) {
	s.policy = policy
}

func (s *traceRecorderStub) RecordPromptAugmentation(diagnostics orchestration.PromptAugmentationDiagnostics) {
	s.augmentation = diagnostics
}

func TestPromptAugmenterMirrorsArtifactsAndReturnsRetrievedSections(t *testing.T) {
	client := &resourceSearchClientStub{
		result: SearchResult{
			Resources: []MatchedContext{{
				URI:         "viking://resources/fracturing-space/campaigns/camp-1/story.md",
				ContextType: "resource",
				Abstract:    "The harbor bells warn of a storm.",
				Score:       0.93,
				MatchReason: "story relevance",
			}},
			Memories: []MatchedContext{{
				URI:         "viking://user/memories/session-1",
				ContextType: "memory",
				Abstract:    "The dockmaster distrusts magic.",
				Score:       0.88,
				MatchReason: "recent turn memory",
			}},
		},
	}
	augmenter, err := NewPromptAugmenter(PromptAugmenterConfig{
		Client:          client,
		Mode:            ModeLegacy,
		MirrorRoot:      t.TempDir(),
		ResourceTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewPromptAugmenter() error = %v", err)
	}

	trace := &traceRecorderStub{}
	ctx := orchestration.WithPromptBuildTraceRecorder(context.Background(), trace)
	contribution, err := augmenter.Augment(ctx, sessionStub{resources: map[string]string{
		"campaign://camp-1/artifacts/story.md":  "A storm gathers offshore.",
		"campaign://camp-1/artifacts/memory.md": "## NPCs\nDockmaster Harl is suspicious.",
	}}, orchestration.SessionBrief{}, orchestration.PromptInput{
		CampaignID:    "camp-1",
		SessionID:     "sess-1",
		ParticipantID: "gm-1",
		TurnInput:     "Open the harbor scene.",
	})
	if err != nil {
		t.Fatalf("Augment() error = %v", err)
	}
	if len(client.added) != 2 {
		t.Fatalf("mirrored resources = %d, want 2", len(client.added))
	}
	if client.added[0].To == "" || client.added[1].To == "" {
		t.Fatalf("mirrored resources = %#v", client.added)
	}
	if len(client.searches) != 1 || client.searches[0].SessionID != StableSessionID("camp-1", "sess-1", "gm-1") {
		t.Fatalf("searches = %#v", client.searches)
	}
	if len(contribution.Sections) != 2 {
		t.Fatalf("sections = %#v", contribution.Sections)
	}
	if len(trace.contexts) != 2 {
		t.Fatalf("recorded contexts = %#v", trace.contexts)
	}
	if !trace.augmentation.Attempted || !trace.augmentation.SearchAttempted {
		t.Fatalf("augmentation diagnostics = %#v", trace.augmentation)
	}
}

func TestBuildRetrievedSectionsSelectsTopContextsAcrossResults(t *testing.T) {
	augmenter, err := NewPromptAugmenter(PromptAugmenterConfig{
		Client: &resourceSearchClientStub{},
		Mode:   ModeLegacy,
	})
	if err != nil {
		t.Fatalf("NewPromptAugmenter() error = %v", err)
	}
	sections, traces := augmenter.buildRetrievedSections(context.Background(), SearchResult{
		Resources: []MatchedContext{
			{URI: "resource-1", ContextType: "resource", Abstract: "R1", Score: 0.7, MatchReason: "a"},
			{URI: "resource-2", ContextType: "resource", Abstract: "R2", Score: 0.9, MatchReason: "b"},
			{URI: "resource-3", ContextType: "resource", Abstract: "R3", Score: 0.8, MatchReason: "c"},
		},
		Memories: []MatchedContext{
			{URI: "memory-1", ContextType: "memory", Abstract: "M1", Score: 0.6, MatchReason: "d"},
		},
	}, 2)

	if len(sections) != 2 || len(traces) != 2 {
		t.Fatalf("sections=%d traces=%d, want 2 each", len(sections), len(traces))
	}
	if traces[0].URI != "resource-2" {
		t.Fatalf("top resource URI = %q, want resource-2", traces[0].URI)
	}
	if traces[1].URI != "resource-3" {
		t.Fatalf("second retrieved URI = %q, want resource-3", traces[1].URI)
	}
}

func TestBuildRetrievedSectionsSkipsDuplicateRenderedTargetsAndUsesNextDistinctCandidate(t *testing.T) {
	client := &resourceSearchClientStub{
		tree: map[string][]FilesystemEntry{
			"viking://resources/fracturing-space/campaigns/camp-1/story.md": {
				{
					URI:     "viking://resources/fracturing-space/campaigns/camp-1/story.md/story.md",
					RelPath: "story.md",
					IsDir:   false,
				},
			},
		},
		read: map[string]string{
			"viking://resources/fracturing-space/campaigns/camp-1/story.md/story.md": "Wrapped story file content.",
		},
	}
	augmenter, err := NewPromptAugmenter(PromptAugmenterConfig{
		Client: client,
		Mode:   ModeDocsAlignedSupplement,
	})
	if err != nil {
		t.Fatalf("NewPromptAugmenter() error = %v", err)
	}

	sections, traces := augmenter.buildRetrievedSections(context.Background(), SearchResult{
		Resources: []MatchedContext{
			{
				URI:         "viking://resources/fracturing-space/campaigns/camp-1/story.md/story.md",
				ContextType: "resource",
				Level:       2,
				Abstract:    "Story file abstract.",
				Score:       0.95,
				MatchReason: "story leaf",
			},
			{
				URI:         "viking://resources/fracturing-space/campaigns/camp-1/story.md/.overview.md",
				ContextType: "resource",
				Level:       1,
				Abstract:    "Story overview abstract.",
				Score:       0.90,
				MatchReason: "story overview",
			},
		},
		Memories: []MatchedContext{
			{
				URI:         "viking://user/default/memories/harbor-note",
				ContextType: "memory",
				Level:       2,
				Abstract:    "Dockmaster Harl still knows the debt collector.",
				Score:       0.80,
				MatchReason: "memory fallback",
			},
		},
	}, 2)

	if len(sections) != 2 || len(traces) != 2 {
		t.Fatalf("sections=%d traces=%d, want 2 each", len(sections), len(traces))
	}
	if traces[0].URI != "viking://resources/fracturing-space/campaigns/camp-1/story.md/story.md" {
		t.Fatalf("first retrieved URI = %q", traces[0].URI)
	}
	if traces[1].URI != "viking://user/default/memories/harbor-note" {
		t.Fatalf("second retrieved URI = %q, want memory fallback", traces[1].URI)
	}
	if traces[0].RenderedURI == "" || traces[0].RenderedURI == traces[1].RenderedURI {
		t.Fatalf("rendered URIs = %#v, want distinct rendered targets", traces)
	}
}

func TestBuildRetrievedSectionsSkipsStoryFamilySiblingAndUsesNextDistinctCandidate(t *testing.T) {
	augmenter, err := NewPromptAugmenter(PromptAugmenterConfig{
		Client: &resourceSearchClientStub{
			read: map[string]string{
				"viking://resources/fracturing-space/campaigns/camp-1/plan/story-index.md": "Story index content.",
				"viking://resources/fracturing-space/campaigns/camp-1/story.md/story.md":   "Full story content.",
				"viking://user/default/memories/harbor-note":                               "The dockmaster is waiting at dawn.",
			},
		},
		Mode: ModeDocsAlignedSupplement,
	})
	if err != nil {
		t.Fatalf("NewPromptAugmenter() error = %v", err)
	}

	sections, traces := augmenter.buildRetrievedSections(context.Background(), SearchResult{
		Resources: []MatchedContext{
			{
				URI:         "viking://resources/fracturing-space/campaigns/camp-1/plan/story-index.md",
				ContextType: "resource",
				Level:       2,
				Abstract:    "Story index abstract.",
				Score:       0.95,
				MatchReason: "story index",
			},
			{
				URI:         "viking://resources/fracturing-space/campaigns/camp-1/story.md/story.md",
				ContextType: "resource",
				Level:       2,
				Abstract:    "Story leaf abstract.",
				Score:       0.90,
				MatchReason: "story leaf",
			},
		},
		Memories: []MatchedContext{{
			URI:         "viking://user/default/memories/harbor-note",
			ContextType: "memory",
			Level:       2,
			Abstract:    "Harbor memory abstract.",
			Score:       0.80,
			MatchReason: "memory fallback",
		}},
	}, 2)

	if len(sections) != 2 || len(traces) != 2 {
		t.Fatalf("sections=%d traces=%d, want 2 each", len(sections), len(traces))
	}
	if traces[0].URI != "viking://resources/fracturing-space/campaigns/camp-1/plan/story-index.md" {
		t.Fatalf("first retrieved URI = %q", traces[0].URI)
	}
	if traces[1].URI != "viking://user/default/memories/harbor-note" {
		t.Fatalf("second retrieved URI = %q, want memory fallback", traces[1].URI)
	}
}

func TestPromptAugmenterDocsAlignedModeMirrorsStoryOnlyAndScopesSearches(t *testing.T) {
	client := &resourceSearchClientStub{
		session: SessionInfo{
			SessionID: "sess-1",
			User: SessionUser{
				AccountID: "default",
				UserID:    "default",
				AgentID:   "default",
			},
		},
		findResults: map[string]SearchResult{
			"viking://resources/fracturing-space/campaigns/camp-1/phase/scene-bootstrap.md": {
				Resources: []MatchedContext{{
					URI:         "viking://resources/fracturing-space/campaigns/camp-1/phase/scene-bootstrap.md",
					ContextType: "resource",
					Level:       2,
					Abstract:    "Bootstrap guide.",
					Score:       0.95,
					MatchReason: "phase relevance",
				}},
			},
			"viking://resources/fracturing-space/campaigns/camp-1/plan/story-index.md": {
				Resources: []MatchedContext{{
					URI:         "viking://resources/fracturing-space/campaigns/camp-1/plan/story-index.md",
					ContextType: "resource",
					Level:       2,
					Abstract:    "Storm warning.",
					Score:       0.93,
					MatchReason: "story relevance",
				}},
			},
		},
	}
	augmenter, err := NewPromptAugmenter(PromptAugmenterConfig{
		Client:          client,
		Mode:            ModeDocsAlignedSupplement,
		MirrorRoot:      t.TempDir(),
		ResourceTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewPromptAugmenter() error = %v", err)
	}

	_, err = augmenter.Augment(context.Background(), sessionStub{resources: map[string]string{
		"campaign://camp-1/artifacts/story.md":  "A storm gathers offshore.",
		"campaign://camp-1/artifacts/memory.md": "## NPCs\nDockmaster Harl is suspicious.",
	}}, orchestration.SessionBrief{
		InteractionState: &orchestration.InteractionStateSnapshot{},
	}, orchestration.PromptInput{
		CampaignID:    "camp-1",
		SessionID:     "sess-1",
		ParticipantID: "gm-1",
		TurnInput:     "Open the harbor scene.",
	})
	if err != nil {
		t.Fatalf("Augment() error = %v", err)
	}
	if len(client.added) != 3 {
		t.Fatalf("mirrored resources = %#v", client.added)
	}
	targets := []string{client.added[0].To, client.added[1].To, client.added[2].To}
	if !containsString(targets, "viking://resources/fracturing-space/campaigns/camp-1/phase/scene-bootstrap.md") {
		t.Fatalf("mirrored targets = %#v, want phase guide", targets)
	}
	if !containsString(targets, "viking://resources/fracturing-space/campaigns/camp-1/plan/story-index.md") {
		t.Fatalf("mirrored targets = %#v, want story index", targets)
	}
	if !containsString(targets, "viking://resources/fracturing-space/campaigns/camp-1/story.md") {
		t.Fatalf("mirrored targets = %#v, want raw story fallback source", targets)
	}
	if len(client.finds) != 2 {
		t.Fatalf("finds = %#v, want 2 scoped resource searches", client.finds)
	}
	if client.finds[0].TargetURI != "viking://resources/fracturing-space/campaigns/camp-1/phase/scene-bootstrap.md" {
		t.Fatalf("first resource target = %q", client.finds[0].TargetURI)
	}
	if client.finds[1].TargetURI != "viking://resources/fracturing-space/campaigns/camp-1/plan/story-index.md" {
		t.Fatalf("second resource target = %q", client.finds[1].TargetURI)
	}
	if len(client.searches) != 1 {
		t.Fatalf("searches = %#v, want 1 session-aware memory search", client.searches)
	}
	if client.searches[0].SessionID != StableSessionID("camp-1", "sess-1", "gm-1") {
		t.Fatalf("memory search session = %q", client.searches[0].SessionID)
	}
	if client.searches[0].TargetURI != "viking://user/default/memories/" {
		t.Fatalf("memory target = %q", client.searches[0].TargetURI)
	}
}

func TestPromptAugmenterDocsAlignedModeFallsBackToRawStoryWhenShallowResourcesMiss(t *testing.T) {
	client := &resourceSearchClientStub{
		findResults: map[string]SearchResult{
			"viking://resources/fracturing-space/campaigns/camp-1/story.md": {
				Resources: []MatchedContext{{
					URI:         "viking://resources/fracturing-space/campaigns/camp-1/story.md/story.md",
					ContextType: "resource",
					Level:       2,
					Abstract:    "Fallback story leaf.",
					Score:       0.92,
					MatchReason: "raw story fallback",
				}},
			},
		},
	}
	augmenter, err := NewPromptAugmenter(PromptAugmenterConfig{
		Client:          client,
		Mode:            ModeDocsAlignedSupplement,
		MirrorRoot:      t.TempDir(),
		ResourceTimeout: time.Second,
	})
	if err != nil {
		t.Fatalf("NewPromptAugmenter() error = %v", err)
	}

	_, err = augmenter.Augment(context.Background(), sessionStub{resources: map[string]string{
		"campaign://camp-1/artifacts/story.md": "A storm gathers offshore.",
	}}, orchestration.SessionBrief{
		InteractionState: &orchestration.InteractionStateSnapshot{},
	}, orchestration.PromptInput{
		CampaignID: "camp-1",
		SessionID:  "sess-1",
		TurnInput:  "Open the harbor scene.",
	})
	if err != nil {
		t.Fatalf("Augment() error = %v", err)
	}
	if len(client.finds) != 3 {
		t.Fatalf("finds = %#v, want phase, story index, and raw story fallback", client.finds)
	}
	if client.finds[2].TargetURI != "viking://resources/fracturing-space/campaigns/camp-1/story.md" {
		t.Fatalf("fallback target = %q", client.finds[2].TargetURI)
	}
}

func TestPromptAugmenterRenderMatchedContentPrefersBackingFileReadForOverviewURI(t *testing.T) {
	client := &resourceSearchClientStub{
		overview: map[string]string{
			"viking://resources/fracturing-space/campaigns/camp-1/story.md/.overview.md": "Generic story overview.",
		},
		read: map[string]string{
			"viking://resources/fracturing-space/campaigns/camp-1/story.md": "Actual story body with concrete harbor details.",
		},
	}
	augmenter, err := NewPromptAugmenter(PromptAugmenterConfig{
		Client: client,
		Mode:   ModeDocsAlignedSupplement,
	})
	if err != nil {
		t.Fatalf("NewPromptAugmenter() error = %v", err)
	}

	rendered := augmenter.renderMatchedContent(context.Background(), &MatchedContext{
		URI:         "viking://resources/fracturing-space/campaigns/camp-1/story.md/.overview.md",
		ContextType: "resource",
		Level:       1,
		Abstract:    "Fallback abstract.",
	})

	if rendered.Content != "Actual story body with concrete harbor details." {
		t.Fatalf("rendered content = %q", rendered.Content)
	}
	if rendered.RenderedURI != "viking://resources/fracturing-space/campaigns/camp-1/story.md" {
		t.Fatalf("rendered uri = %q", rendered.RenderedURI)
	}
	if rendered.Source != "backing_read" {
		t.Fatalf("content source = %q", rendered.Source)
	}
}

func containsString(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

func TestBackingReadURITrimsOverviewSuffix(t *testing.T) {
	got := backingReadURI("viking://resources/fracturing-space/campaigns/camp-1/story.md/.overview.md")
	want := "viking://resources/fracturing-space/campaigns/camp-1/story.md"
	if got != want {
		t.Fatalf("backingReadURI() = %q, want %q", got, want)
	}
}

func TestPromptAugmenterRenderMatchedContentFallsBackToOverviewWithError(t *testing.T) {
	client := &resourceSearchClientStub{
		overview: map[string]string{
			"viking://resources/fracturing-space/campaigns/camp-1/story.md/.overview.md": "Generic story overview.",
		},
	}
	augmenter, err := NewPromptAugmenter(PromptAugmenterConfig{
		Client: client,
		Mode:   ModeDocsAlignedSupplement,
	})
	if err != nil {
		t.Fatalf("NewPromptAugmenter() error = %v", err)
	}

	rendered := augmenter.renderMatchedContent(context.Background(), &MatchedContext{
		URI:         "viking://resources/fracturing-space/campaigns/camp-1/story.md/.overview.md",
		ContextType: "resource",
		Level:       1,
		Abstract:    "Fallback abstract.",
	})

	if rendered.Content != "Generic story overview." {
		t.Fatalf("rendered content = %q", rendered.Content)
	}
	if rendered.Source != "overview" {
		t.Fatalf("content source = %q", rendered.Source)
	}
	if !strings.Contains(rendered.Error, "backing_read") {
		t.Fatalf("render error = %q, want backing_read details", rendered.Error)
	}
}

func TestPromptAugmenterRenderMatchedContentPrefersLeafReadOverAbstract(t *testing.T) {
	client := &resourceSearchClientStub{
		read: map[string]string{
			"viking://resources/fracturing-space/campaigns/camp-1/story.md": "Full story text.",
		},
	}
	augmenter, err := NewPromptAugmenter(PromptAugmenterConfig{
		Client: client,
		Mode:   ModeDocsAlignedSupplement,
	})
	if err != nil {
		t.Fatalf("NewPromptAugmenter() error = %v", err)
	}

	rendered := augmenter.renderMatchedContent(context.Background(), &MatchedContext{
		URI:         "viking://resources/fracturing-space/campaigns/camp-1/story.md",
		ContextType: "resource",
		Level:       2,
		Abstract:    "Short abstract.",
	})

	if rendered.Content != "Full story text." {
		t.Fatalf("rendered content = %q", rendered.Content)
	}
	if rendered.Source != "leaf_read" {
		t.Fatalf("content source = %q", rendered.Source)
	}
}

func TestPromptAugmenterRenderMatchedContentUsesBackingTreeLeafWhenRootReadIsEmpty(t *testing.T) {
	client := &resourceSearchClientStub{
		tree: map[string][]FilesystemEntry{
			"viking://resources/fracturing-space/campaigns/camp-1/story.md": {
				{
					URI:     "viking://resources/fracturing-space/campaigns/camp-1/story.md/story.md",
					RelPath: "story.md",
					IsDir:   false,
				},
			},
		},
		read: map[string]string{
			"viking://resources/fracturing-space/campaigns/camp-1/story.md/story.md": "Wrapped story file content.",
		},
	}
	augmenter, err := NewPromptAugmenter(PromptAugmenterConfig{
		Client: client,
		Mode:   ModeDocsAlignedSupplement,
	})
	if err != nil {
		t.Fatalf("NewPromptAugmenter() error = %v", err)
	}

	rendered := augmenter.renderMatchedContent(context.Background(), &MatchedContext{
		URI:         "viking://resources/fracturing-space/campaigns/camp-1/story.md/.overview.md",
		ContextType: "resource",
		Level:       1,
		Abstract:    "Fallback abstract.",
	})

	if rendered.Content != "Wrapped story file content." {
		t.Fatalf("rendered content = %q", rendered.Content)
	}
	if rendered.RenderedURI != "viking://resources/fracturing-space/campaigns/camp-1/story.md/story.md" {
		t.Fatalf("rendered uri = %q", rendered.RenderedURI)
	}
	if rendered.Source != "backing_tree_read" {
		t.Fatalf("content source = %q", rendered.Source)
	}
}
