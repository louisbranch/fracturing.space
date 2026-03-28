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
	mirrored, err := a.syncArtifacts(ctx, sess, brief, input)
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

func (a *PromptAugmenter) syncArtifacts(ctx context.Context, sess orchestration.Session, brief orchestration.SessionBrief, input orchestration.PromptInput) ([]mirroredArtifact, error) {
	type artifactSpec struct {
		logicalPath string
		to          string
		reason      string
		content     string
	}
	campaignID := strings.TrimSpace(input.CampaignID)
	if campaignID == "" {
		return nil, fmt.Errorf("campaign ID is required")
	}
	storyRaw, err := optionalSessionResource(ctx, sess, fmt.Sprintf("campaign://%s/artifacts/story.md", campaignID))
	if err != nil {
		return nil, err
	}
	memoryRaw := ""
	if a.mode.MirrorsMemory() {
		memoryRaw, err = optionalSessionResource(ctx, sess, fmt.Sprintf("campaign://%s/artifacts/memory.md", campaignID))
		if err != nil {
			return nil, err
		}
	}
	specs := []artifactSpec{}
	if a.mode.UsesScopedRetrieval() {
		phaseContent := strings.TrimSpace(strings.Join([]string{
			orchestration.BuildPhaseGuide(brief.TurnMode(), input),
			orchestration.BuildContextAccessMap(brief.TurnMode(), input),
		}, "\n\n"))
		if phaseContent != "" {
			logicalPath := phaseResourceLogicalPath(brief.TurnMode())
			specs = append(specs, artifactSpec{
				logicalPath: logicalPath,
				to:          campaignResourceURI(campaignID, logicalPath),
				reason:      "campaign turn phase guide",
				content:     phaseContent,
			})
		}
		if storyIndex := strings.TrimSpace(orchestration.BuildStoryContextIndex(campaignID, storyRaw)); storyIndex != "" {
			specs = append(specs, artifactSpec{
				logicalPath: storyIndexLogicalPath,
				to:          campaignResourceURI(campaignID, storyIndexLogicalPath),
				reason:      "campaign story index",
				content:     storyIndex,
			})
		}
	}
	if a.mode.MirrorsStory() {
		specs = append(specs, artifactSpec{
			logicalPath: rawStoryLogicalPath,
			to:          campaignResourceURI(campaignID, rawStoryLogicalPath),
			reason:      "campaign story context",
			content:     storyRaw,
		})
	}
	if a.mode.MirrorsMemory() {
		specs = append(specs, artifactSpec{
			logicalPath: rawMemoryLogicalPath,
			to:          campaignResourceURI(campaignID, rawMemoryLogicalPath),
			reason:      "campaign GM memory context",
			content:     memoryRaw,
		})
	}
	root := filepath.Join(a.mirrorRoot, campaignID)
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, fmt.Errorf("create openviking mirror root: %w", err)
	}
	targets := make([]mirroredArtifact, 0, len(specs))
	for _, spec := range specs {
		localPath := filepath.Join(root, filepath.FromSlash(spec.logicalPath))
		if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
			return targets, fmt.Errorf("create mirrored artifact parent %s: %w", spec.logicalPath, err)
		}
		if err := os.WriteFile(localPath, []byte(spec.content), 0o600); err != nil {
			return targets, fmt.Errorf("write mirrored artifact %s: %w", spec.logicalPath, err)
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
			return targets, fmt.Errorf("mirror %s into openviking: %w", spec.logicalPath, err)
		}
		rootURI := strings.TrimSpace(result.RootURI)
		if rootURI == "" {
			rootURI = spec.to
		}
		targets = append(targets, mirroredArtifact{
			LogicalPath: spec.logicalPath,
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
	for _, target := range []string{
		mirroredArtifactRoot(mirrored, phaseResourceLogicalPath(brief.TurnMode())),
		mirroredArtifactRoot(mirrored, storyIndexLogicalPath),
	} {
		target = strings.TrimSpace(target)
		if target == "" {
			continue
		}
		remaining := remainingSearchSlots(a.maxResults, combined.Resources)
		if remaining == 0 {
			break
		}
		resourceResult, err := a.client.Find(ctx, SearchInput{
			Query:     query,
			TargetURI: target,
			Limit:     remaining,
		})
		if err != nil {
			return SearchResult{}, err
		}
		combined.Resources = appendDistinctMatches(combined.Resources, resourceResult.Resources)
	}
	if len(combined.Resources) == 0 {
		if storyRoot := mirroredArtifactRoot(mirrored, rawStoryLogicalPath); storyRoot != "" {
			resourceResult, err := a.client.Find(ctx, SearchInput{
				Query:     query,
				TargetURI: storyRoot,
				Limit:     a.maxResults,
			})
			if err != nil {
				return SearchResult{}, err
			}
			combined.Resources = appendDistinctMatches(combined.Resources, resourceResult.Resources)
		}
	}

	if a.mode.UsesSessionMemorySupplement() {
		session, err := a.client.GetSession(ctx, sessionID)
		if err != nil {
			if !isNotFoundError(err) {
				return SearchResult{}, err
			}
		} else if memoryRoot := userMemoryRoot(session.User); memoryRoot != "" {
			memoryResult, err := a.client.Search(ctx, SearchInput{
				Query:     query,
				SessionID: sessionID,
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
	candidates := rankedMatchedContexts(result)
	sections := make([]orchestration.BriefSection, 0, len(candidates))
	traces := make([]orchestration.RetrievedContext, 0, len(candidates))
	seenMatches := map[string]struct{}{}
	seenRenderedTargets := map[string]struct{}{}
	seenFamilies := map[string]struct{}{}
	selectedByType := map[string]int{}
	for _, match := range candidates {
		if maxSections > 0 && len(sections) >= maxSections {
			break
		}
		uri := strings.TrimSpace(match.URI)
		if uri == "" {
			continue
		}
		if _, ok := seenMatches[uri]; ok {
			continue
		}
		seenMatches[uri] = struct{}{}
		typeKey := retrievedContextTypeKey(match.ContextType)
		rendered := a.renderMatchedContent(ctx, &match)
		content := strings.TrimSpace(rendered.Content)
		if content == "" {
			continue
		}
		if dedupeKey := renderedContextDedupKey(match, rendered); dedupeKey != "" {
			if _, ok := seenRenderedTargets[dedupeKey]; ok {
				continue
			}
			seenRenderedTargets[dedupeKey] = struct{}{}
		}
		if familyKey := logicalDocumentFamily(match, rendered); familyKey != "" {
			if _, ok := seenFamilies[familyKey]; ok {
				continue
			}
			seenFamilies[familyKey] = struct{}{}
		}
		selectedByType[typeKey]++
		section := orchestration.BriefSection{
			ID:       fmt.Sprintf("openviking_%s_%d", typeKey, selectedByType[typeKey]),
			Priority: 350,
			Label:    retrievedContextLabel(typeKey, selectedByType[typeKey]),
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

// renderedContextDedupKey collapses overview/backing-file variants onto the
// final rendered target so prompt augmentation does not pay twice for the same
// story content.
func renderedContextDedupKey(match MatchedContext, rendered renderedMatchedContent) string {
	if uri := strings.TrimSpace(rendered.RenderedURI); uri != "" {
		return uri
	}
	return strings.TrimSpace(match.URI)
}

func logicalDocumentFamily(match MatchedContext, rendered renderedMatchedContent) string {
	for _, uri := range []string{strings.TrimSpace(rendered.RenderedURI), strings.TrimSpace(match.URI)} {
		if uri == "" {
			continue
		}
		switch {
		case strings.Contains(uri, "/plan/story-index.md"),
			strings.HasSuffix(uri, "/story.md"),
			strings.Contains(uri, "/story.md/"):
			return "story"
		case strings.Contains(uri, "/phase/scene-bootstrap.md"):
			return "phase:scene-bootstrap"
		case strings.Contains(uri, "/phase/scene-play.md"):
			return "phase:scene-play"
		case strings.Contains(uri, "/phase/action-review.md"):
			return "phase:action-review"
		}
	}
	return ""
}

func rankedMatchedContexts(result SearchResult) []MatchedContext {
	candidates := make([]MatchedContext, 0, len(result.Resources)+len(result.Memories))
	candidates = append(candidates, result.Resources...)
	candidates = append(candidates, result.Memories...)
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].Score > candidates[j].Score
	})
	return candidates
}

func retrievedContextTypeKey(contextType string) string {
	switch strings.TrimSpace(contextType) {
	case "memory":
		return "memory"
	default:
		return "resource"
	}
}

func retrievedContextLabel(typeKey string, selected int) string {
	base := "Retrieved resource"
	if typeKey == "memory" {
		base = "Retrieved memory"
	}
	if selected <= 1 {
		return base
	}
	return fmt.Sprintf("%s %d", base, selected)
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

func appendDistinctMatches(existing []MatchedContext, incoming []MatchedContext) []MatchedContext {
	if len(incoming) == 0 {
		return existing
	}
	seen := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		if uri := strings.TrimSpace(item.URI); uri != "" {
			seen[uri] = struct{}{}
		}
	}
	for _, item := range incoming {
		uri := strings.TrimSpace(item.URI)
		if uri == "" {
			continue
		}
		if _, ok := seen[uri]; ok {
			continue
		}
		seen[uri] = struct{}{}
		existing = append(existing, item)
	}
	return existing
}

func remainingSearchSlots(limit int, existing []MatchedContext) int {
	if limit <= 0 {
		return 0
	}
	if len(existing) >= limit {
		return 0
	}
	return limit - len(existing)
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

const (
	rawStoryLogicalPath   = "story.md"
	rawMemoryLogicalPath  = "memory.md"
	storyIndexLogicalPath = "plan/story-index.md"
)

func phaseResourceLogicalPath(mode orchestration.InteractionTurnMode) string {
	return fmt.Sprintf("phase/%s.md", orchestration.PhaseResourceName(mode))
}

func campaignResourceURI(campaignID string, logicalPath string) string {
	return fmt.Sprintf("viking://resources/fracturing-space/campaigns/%s/%s", strings.TrimSpace(campaignID), strings.TrimSpace(logicalPath))
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
