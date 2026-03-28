package openviking

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

const (
	defaultMaxResults      = 4
	defaultMaxSections     = 2
	defaultResourceTimeout = 2 * time.Second
)

type resourceSearchClient interface {
	AddResource(ctx context.Context, input AddResourceInput) (AddResourceResult, error)
	Find(ctx context.Context, input SearchInput) (SearchResult, error)
	Search(ctx context.Context, input SearchInput) (SearchResult, error)
	GetSession(ctx context.Context, sessionID string) (SessionInfo, error)
	Overview(ctx context.Context, uri string) (string, error)
	Read(ctx context.Context, uri string) (string, error)
	Tree(ctx context.Context, uri string) ([]FilesystemEntry, error)
}

// PromptAugmenterConfig declares the OpenViking-backed prompt augmentation
// policy.
type PromptAugmenterConfig struct {
	Client            resourceSearchClient
	Mode              IntegrationMode
	MirrorRoot        string
	VisibleMirrorRoot string
	MaxResults        int
	MaxSections       int
	ResourceTimeout   time.Duration
}

// PromptAugmenter mirrors campaign artifacts into OpenViking and retrieves a
// compact supplemental context slice for the current turn.
type PromptAugmenter struct {
	client            resourceSearchClient
	mode              IntegrationMode
	mirrorRoot        string
	visibleMirrorRoot string
	maxResults        int
	maxSections       int
	resourceTimeout   time.Duration
}

// NewPromptAugmenter builds one prompt augmenter from explicit dependencies.
func NewPromptAugmenter(cfg PromptAugmenterConfig) (*PromptAugmenter, error) {
	if cfg.Client == nil {
		return nil, fmt.Errorf("openviking client is required")
	}
	mode, err := ParseIntegrationMode(string(cfg.Mode))
	if err != nil {
		return nil, err
	}
	mirrorRoot := strings.TrimSpace(cfg.MirrorRoot)
	if mirrorRoot == "" {
		mirrorRoot = filepath.Join(os.TempDir(), "fracturing-space-openviking")
	}
	absMirrorRoot, err := filepath.Abs(mirrorRoot)
	if err != nil {
		return nil, fmt.Errorf("normalize openviking mirror root: %w", err)
	}
	maxResults := cfg.MaxResults
	if maxResults <= 0 {
		maxResults = defaultMaxResults
	}
	maxSections := cfg.MaxSections
	if maxSections <= 0 {
		maxSections = defaultMaxSections
	}
	resourceTimeout := cfg.ResourceTimeout
	if resourceTimeout <= 0 {
		resourceTimeout = defaultResourceTimeout
	}
	return &PromptAugmenter{
		client:            cfg.Client,
		mode:              mode,
		mirrorRoot:        absMirrorRoot,
		visibleMirrorRoot: strings.TrimSpace(cfg.VisibleMirrorRoot),
		maxResults:        maxResults,
		maxSections:       maxSections,
		resourceTimeout:   resourceTimeout,
	}, nil
}

// Augment implements orchestration.PromptAugmenter.
func (a *PromptAugmenter) Augment(ctx context.Context, sess orchestration.Session, brief orchestration.SessionBrief, input orchestration.PromptInput) (orchestration.BriefContribution, error) {
	if a == nil || a.client == nil {
		return orchestration.BriefContribution{}, nil
	}
	mirrored, err := a.syncArtifacts(ctx, sess, input.CampaignID)
	mirroredTargets := mirroredArtifactRoots(mirrored)
	orchestration.RecordPromptAugmentation(ctx, orchestration.PromptAugmentationDiagnostics{
		Attempted:       true,
		Mode:            string(a.mode),
		MirroredTargets: mirroredTargets,
	})
	if err != nil {
		orchestration.RecordPromptAugmentation(ctx, orchestration.PromptAugmentationDiagnostics{
			Attempted:         true,
			Mode:              string(a.mode),
			MirroredTargets:   mirroredTargets,
			Degraded:          true,
			DegradationReason: err.Error(),
		})
		return orchestration.BriefContribution{}, err
	}

	result, err := a.searchContexts(ctx, brief, input, mirrored)
	if err != nil {
		orchestration.RecordPromptAugmentation(ctx, orchestration.PromptAugmentationDiagnostics{
			Attempted:         true,
			Mode:              string(a.mode),
			MirroredTargets:   mirroredTargets,
			SearchAttempted:   true,
			Degraded:          true,
			DegradationReason: err.Error(),
		})
		return orchestration.BriefContribution{}, err
	}
	orchestration.RecordPromptAugmentation(ctx, orchestration.PromptAugmentationDiagnostics{
		Attempted:       true,
		Mode:            string(a.mode),
		MirroredTargets: mirroredTargets,
		SearchAttempted: true,
		ResourceHits:    len(result.Resources),
		MemoryHits:      len(result.Memories),
	})

	sections, traces := a.buildRetrievedSections(ctx, result, a.maxSections)
	orchestration.RecordRetrievedContexts(ctx, traces)
	if len(sections) == 0 {
		return orchestration.BriefContribution{}, nil
	}
	return orchestration.BriefContribution{Sections: sections}, nil
}

type mirroredArtifact struct {
	LogicalPath string
	RootURI     string
}

func (a *PromptAugmenter) syncArtifacts(ctx context.Context, sess orchestration.Session, campaignID string) ([]mirroredArtifact, error) {
	type artifactSpec struct {
		path   string
		to     string
		reason string
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, fmt.Errorf("campaign ID is required")
	}
	specs := []artifactSpec{}
	if a.mode.MirrorsStory() {
		specs = append(specs, artifactSpec{
			path:   "story.md",
			to:     fmt.Sprintf("viking://resources/fracturing-space/campaigns/%s/story.md", campaignID),
			reason: "campaign story context",
		})
	}
	if a.mode.MirrorsMemory() {
		specs = append(specs, artifactSpec{
			path:   "memory.md",
			to:     fmt.Sprintf("viking://resources/fracturing-space/campaigns/%s/memory.md", campaignID),
			reason: "campaign GM memory context",
		})
	}
	root := filepath.Join(a.mirrorRoot, campaignID)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create openviking mirror root: %w", err)
	}
	targets := make([]mirroredArtifact, 0, len(specs))
	for _, spec := range specs {
		raw, err := optionalSessionResource(ctx, sess, fmt.Sprintf("campaign://%s/artifacts/%s", campaignID, spec.path))
		if err != nil {
			return targets, err
		}
		localPath := filepath.Join(root, spec.path)
		if err := os.WriteFile(localPath, []byte(raw), 0o600); err != nil {
			return targets, fmt.Errorf("write mirrored artifact %s: %w", spec.path, err)
		}
		visiblePath, err := a.visiblePath(localPath)
		if err != nil {
			return targets, err
		}
		result, err := a.client.AddResource(ctx, AddResourceInput{
			Path:    visiblePath,
			To:      spec.to,
			Reason:  spec.reason,
			Wait:    true,
			Timeout: a.resourceTimeout,
		})
		if err != nil {
			return targets, fmt.Errorf("mirror %s into openviking: %w", spec.path, err)
		}
		rootURI := strings.TrimSpace(result.RootURI)
		if rootURI == "" {
			rootURI = spec.to
		}
		targets = append(targets, mirroredArtifact{
			LogicalPath: spec.path,
			RootURI:     rootURI,
		})
	}
	return targets, nil
}

func optionalSessionResource(ctx context.Context, sess orchestration.Session, uri string) (string, error) {
	value, err := sess.ReadResource(ctx, uri)
	if err != nil {
		if isResourceNotFound(err) {
			return "", nil
		}
		return "", fmt.Errorf("read %s: %w", uri, err)
	}
	return value, nil
}

func isResourceNotFound(err error) bool {
	if err == nil {
		return false
	}
	text := strings.ToLower(err.Error())
	return strings.Contains(text, "not found") || strings.Contains(text, "missing resource")
}

func buildSearchQuery(brief orchestration.SessionBrief, input orchestration.PromptInput) string {
	var b strings.Builder
	b.WriteString("Campaign GM turn context")
	if mode := strings.TrimSpace(string(brief.TurnMode())); mode != "" {
		b.WriteString("\nMode: ")
		b.WriteString(mode)
	}
	if turnInput := strings.TrimSpace(input.TurnInput); turnInput != "" {
		b.WriteString("\nTurn input: ")
		b.WriteString(turnInput)
	}
	return b.String()
}

func (a *PromptAugmenter) searchContexts(ctx context.Context, brief orchestration.SessionBrief, input orchestration.PromptInput, mirrored []mirroredArtifact) (SearchResult, error) {
	query := buildSearchQuery(brief, input)
	sessionID := StableSessionID(input.CampaignID, input.SessionID, input.ParticipantID)
	if !a.mode.UsesScopedRetrieval() {
		return a.client.Search(ctx, SearchInput{
			Query:     query,
			SessionID: sessionID,
			Limit:     a.maxResults,
		})
	}

	combined := SearchResult{}
	if storyRoot := mirroredArtifactRoot(mirrored, "story.md"); storyRoot != "" {
		resourceResult, err := a.client.Find(ctx, SearchInput{
			Query:     query,
			TargetURI: storyRoot,
			Limit:     a.maxResults,
		})
		if err != nil {
			return SearchResult{}, err
		}
		combined.Resources = append(combined.Resources, resourceResult.Resources...)
	}

	if a.mode.UsesSessionMemorySupplement() {
		session, err := a.client.GetSession(ctx, sessionID)
		if err != nil {
			if !isNotFoundError(err) {
				return SearchResult{}, err
			}
		} else if memoryRoot := userMemoryRoot(session.User); memoryRoot != "" {
			memoryResult, err := a.client.Find(ctx, SearchInput{
				Query:     query,
				TargetURI: memoryRoot,
				Limit:     a.maxResults,
			})
			if err != nil {
				return SearchResult{}, err
			}
			combined.Memories = append(combined.Memories, memoryResult.Memories...)
		}
	}
	return combined, nil
}

func (a *PromptAugmenter) buildRetrievedSections(ctx context.Context, result SearchResult, maxSections int) ([]orchestration.BriefSection, []orchestration.RetrievedContext) {
	type group struct {
		id      string
		label   string
		context *MatchedContext
	}

	groups := []group{
		{id: "openviking_resource", label: "Retrieved resources", context: firstMatchedContext(result.Resources)},
		{id: "openviking_memory", label: "Retrieved memory", context: firstMatchedContext(result.Memories)},
	}
	sort.SliceStable(groups, func(i, j int) bool {
		return scoreOf(groups[i].context) > scoreOf(groups[j].context)
	})

	sections := make([]orchestration.BriefSection, 0, len(groups))
	traces := make([]orchestration.RetrievedContext, 0, len(groups))
	for _, item := range groups {
		if item.context == nil {
			continue
		}
		if maxSections > 0 && len(sections) >= maxSections {
			break
		}
		match := *item.context
		rendered := a.renderMatchedContent(ctx, item.context)
		content := strings.TrimSpace(rendered.Content)
		if content == "" {
			continue
		}
		section := orchestration.BriefSection{
			ID:       item.id,
			Priority: 350,
			Label:    item.label,
			Content: strings.Join([]string{
				fmt.Sprintf("URI: %s", strings.TrimSpace(match.URI)),
				fmt.Sprintf("Match reason: %s", strings.TrimSpace(match.MatchReason)),
				fmt.Sprintf("Summary: %s", content),
			}, "\n"),
		}
		sections = append(sections, section)
		traces = append(traces, orchestration.RetrievedContext{
			URI:           strings.TrimSpace(match.URI),
			RenderedURI:   strings.TrimSpace(rendered.RenderedURI),
			ContextType:   strings.TrimSpace(match.ContextType),
			Abstract:      content,
			MatchReason:   strings.TrimSpace(match.MatchReason),
			Score:         match.Score,
			ContentSource: strings.TrimSpace(rendered.Source),
			ContentError:  strings.TrimSpace(rendered.Error),
		})
	}
	return sections, traces
}

type renderedMatchedContent struct {
	Content     string
	RenderedURI string
	Source      string
	Error       string
}

func (a *PromptAugmenter) renderMatchedContent(ctx context.Context, item *MatchedContext) renderedMatchedContent {
	if item == nil {
		return renderedMatchedContent{}
	}
	content := strings.TrimSpace(item.Abstract)
	if !a.mode.UsesScopedRetrieval() {
		return renderedMatchedContent{
			Content:     content,
			RenderedURI: strings.TrimSpace(item.URI),
			Source:      "abstract",
		}
	}
	if !item.isLeaf() {
		var readErrText string
		if backingURI := backingReadURI(item.URI); backingURI != "" {
			full, err := a.client.Read(ctx, backingURI)
			if err == nil && strings.TrimSpace(full) != "" {
				return renderedMatchedContent{
					Content:     strings.TrimSpace(full),
					RenderedURI: backingURI,
					Source:      "backing_read",
				}
			}
			readErrText = formatRenderError("backing_read", backingURI, err, full)
			if childURI, childErr := a.backingTreeLeafURI(ctx, backingURI); childURI != "" {
				full, err := a.client.Read(ctx, childURI)
				if err == nil && strings.TrimSpace(full) != "" {
					return renderedMatchedContent{
						Content:     strings.TrimSpace(full),
						RenderedURI: childURI,
						Source:      "backing_tree_read",
						Error:       readErrText,
					}
				}
				readErrText = joinRenderErrors(readErrText, formatRenderError("backing_tree_read", childURI, err, full))
			} else if strings.TrimSpace(childErr) != "" {
				readErrText = joinRenderErrors(readErrText, childErr)
			}
		}
		overview, err := a.client.Overview(ctx, item.URI)
		if err == nil && strings.TrimSpace(overview) != "" {
			return renderedMatchedContent{
				Content:     strings.TrimSpace(overview),
				RenderedURI: strings.TrimSpace(item.URI),
				Source:      "overview",
				Error:       readErrText,
			}
		}
		return renderedMatchedContent{
			Content:     content,
			RenderedURI: strings.TrimSpace(item.URI),
			Source:      "abstract",
			Error:       joinRenderErrors(readErrText, formatRenderError("overview", item.URI, err, overview)),
		}
	}
	full, err := a.client.Read(ctx, item.URI)
	if err == nil && strings.TrimSpace(full) != "" {
		return renderedMatchedContent{
			Content:     strings.TrimSpace(full),
			RenderedURI: strings.TrimSpace(item.URI),
			Source:      "leaf_read",
		}
	}
	return renderedMatchedContent{
		Content:     content,
		RenderedURI: strings.TrimSpace(item.URI),
		Source:      "abstract",
		Error:       formatRenderError("leaf_read", item.URI, err, full),
	}
}

func firstMatchedContext(items []MatchedContext) *MatchedContext {
	if len(items) == 0 {
		return nil
	}
	best := items[0]
	for _, item := range items[1:] {
		if item.Score > best.Score {
			best = item
		}
	}
	return &best
}

func scoreOf(item *MatchedContext) float64 {
	if item == nil {
		return -1
	}
	return item.Score
}

func (a *PromptAugmenter) visiblePath(localPath string) (string, error) {
	if strings.TrimSpace(a.visibleMirrorRoot) == "" {
		return localPath, nil
	}
	rel, err := filepath.Rel(a.mirrorRoot, localPath)
	if err != nil {
		return "", fmt.Errorf("rewrite openviking mirror path: %w", err)
	}
	if rel == "." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || rel == ".." {
		return "", fmt.Errorf("openviking mirrored file %s is outside mirror root %s", localPath, a.mirrorRoot)
	}
	return filepath.Join(a.visibleMirrorRoot, rel), nil
}

func mirroredArtifactRoots(items []mirroredArtifact) []string {
	roots := make([]string, 0, len(items))
	for _, item := range items {
		if root := strings.TrimSpace(item.RootURI); root != "" {
			roots = append(roots, root)
		}
	}
	return roots
}

func mirroredArtifactRoot(items []mirroredArtifact, logicalPath string) string {
	for _, item := range items {
		if item.LogicalPath == logicalPath {
			return strings.TrimSpace(item.RootURI)
		}
	}
	return ""
}

func userMemoryRoot(user SessionUser) string {
	userID := strings.TrimSpace(user.UserID)
	if userID == "" {
		return ""
	}
	return fmt.Sprintf("viking://user/%s/memories/", userID)
}

func (m MatchedContext) isLeaf() bool {
	return m.IsLeaf || m.Level >= 2
}

func backingReadURI(uri string) string {
	uri = strings.TrimSpace(uri)
	if strings.HasSuffix(uri, "/.overview.md") {
		return strings.TrimSpace(strings.TrimSuffix(uri, "/.overview.md"))
	}
	return ""
}

func (a *PromptAugmenter) backingTreeLeafURI(ctx context.Context, uri string) (string, string) {
	entries, err := a.client.Tree(ctx, uri)
	if err != nil {
		return "", formatRenderError("backing_tree", uri, err, "")
	}
	fileURIs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir {
			continue
		}
		item := strings.TrimSpace(entry.URI)
		if item == "" {
			continue
		}
		fileURIs = append(fileURIs, item)
	}
	if len(fileURIs) == 1 {
		return fileURIs[0], ""
	}
	if len(fileURIs) == 0 {
		return "", formatRenderError("backing_tree", uri, nil, "")
	}
	return "", fmt.Sprintf("backing_tree: %s: ambiguous leaf candidates: %s", strings.TrimSpace(uri), strings.Join(fileURIs, ", "))
}

func formatRenderError(source, uri string, err error, content string) string {
	if err == nil && strings.TrimSpace(content) != "" {
		return ""
	}
	details := []string{}
	if source = strings.TrimSpace(source); source != "" {
		details = append(details, source)
	}
	if uri = strings.TrimSpace(uri); uri != "" {
		details = append(details, uri)
	}
	if err != nil {
		details = append(details, err.Error())
	} else {
		details = append(details, "empty content")
	}
	return strings.Join(details, ": ")
}

func joinRenderErrors(parts ...string) string {
	items := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		items = append(items, part)
	}
	return strings.Join(items, " | ")
}
