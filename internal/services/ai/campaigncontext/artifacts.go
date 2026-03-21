package campaigncontext

import (
	"context"
	"fmt"
	pathpkg "path"
	"regexp"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
)

const (
	// DaggerheartSystem identifies the mounted Daggerheart reference corpus.
	DaggerheartSystem = "daggerheart"

	// SkillsArtifactPath is the read-only built-in GM operating contract.
	SkillsArtifactPath = "skills.md"
	// StoryArtifactPath is the campaign storyline/context document.
	StoryArtifactPath = "story.md"
	// MemoryArtifactPath is the AI's durable GM working memory.
	MemoryArtifactPath = "memory.md"
)

var workingArtifactSlugPattern = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]*$`)

// ArtifactStore persists campaign-scoped GM artifacts.
type ArtifactStore interface {
	PutCampaignArtifact(ctx context.Context, record storage.CampaignArtifactRecord) error
	GetCampaignArtifact(ctx context.Context, campaignID string, path string) (storage.CampaignArtifactRecord, error)
	ListCampaignArtifacts(ctx context.Context, campaignID string) ([]storage.CampaignArtifactRecord, error)
}

// Manager owns campaign artifact defaults and path policy.
type Manager struct {
	store             ArtifactStore
	clock             func() time.Time
	instructionLoader *InstructionLoader
}

// NewManager builds a campaign artifact manager over one persistent store.
func NewManager(store ArtifactStore, clock func() time.Time) *Manager {
	if clock == nil {
		clock = time.Now
	}
	return &Manager{store: store, clock: clock}
}

// SetInstructionLoader configures the instruction loader used for default
// artifact content. When set, default skills.md content comes from the
// composed instruction files instead of the hardcoded fallback.
func (m *Manager) SetInstructionLoader(loader *InstructionLoader) {
	if m != nil {
		m.instructionLoader = loader
	}
}

// EnsureDefaultArtifacts creates built-in GM artifacts if they are missing.
func (m *Manager) EnsureDefaultArtifacts(ctx context.Context, campaignID string, storySeed string) ([]storage.CampaignArtifactRecord, error) {
	if m == nil || m.store == nil {
		return nil, fmt.Errorf("campaign artifact store is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	if err := ensureArtifactIfMissing(ctx, m.store, m.clock, campaignID, storage.CampaignArtifactRecord{
		CampaignID: campaignID,
		Path:       SkillsArtifactPath,
		Content:    m.resolveSkillsContent(),
		ReadOnly:   true,
	}); err != nil {
		return nil, err
	}
	if err := ensureArtifactIfMissing(ctx, m.store, m.clock, campaignID, storage.CampaignArtifactRecord{
		CampaignID: campaignID,
		Path:       MemoryArtifactPath,
		Content:    "",
		ReadOnly:   false,
	}); err != nil {
		return nil, err
	}
	storySeed = strings.TrimSpace(storySeed)
	if storySeed != "" {
		if err := ensureArtifactIfMissing(ctx, m.store, m.clock, campaignID, storage.CampaignArtifactRecord{
			CampaignID: campaignID,
			Path:       StoryArtifactPath,
			Content:    storySeed,
			ReadOnly:   false,
		}); err != nil {
			return nil, err
		}
	}
	return m.ListArtifacts(ctx, campaignID)
}

// ListArtifacts returns all persisted campaign artifacts.
func (m *Manager) ListArtifacts(ctx context.Context, campaignID string) ([]storage.CampaignArtifactRecord, error) {
	if m == nil || m.store == nil {
		return nil, fmt.Errorf("campaign artifact store is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return nil, fmt.Errorf("campaign id is required")
	}
	return m.store.ListCampaignArtifacts(ctx, campaignID)
}

// GetArtifact returns one persisted campaign artifact.
func (m *Manager) GetArtifact(ctx context.Context, campaignID string, path string) (storage.CampaignArtifactRecord, error) {
	if m == nil || m.store == nil {
		return storage.CampaignArtifactRecord{}, fmt.Errorf("campaign artifact store is not configured")
	}
	normalizedPath, err := NormalizeArtifactPath(path)
	if err != nil {
		return storage.CampaignArtifactRecord{}, err
	}
	return m.store.GetCampaignArtifact(ctx, strings.TrimSpace(campaignID), normalizedPath)
}

// UpsertArtifact validates policy and replaces one mutable artifact body.
func (m *Manager) UpsertArtifact(ctx context.Context, campaignID string, path string, content string) (storage.CampaignArtifactRecord, error) {
	if m == nil || m.store == nil {
		return storage.CampaignArtifactRecord{}, fmt.Errorf("campaign artifact store is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return storage.CampaignArtifactRecord{}, fmt.Errorf("campaign id is required")
	}
	normalizedPath, err := normalizeWritableArtifactPath(path)
	if err != nil {
		return storage.CampaignArtifactRecord{}, err
	}
	now := m.clock().UTC()
	record, err := m.store.GetCampaignArtifact(ctx, campaignID, normalizedPath)
	switch {
	case err == nil:
		record.Content = content
		record.UpdatedAt = now
	case err == storage.ErrNotFound:
		record = storage.CampaignArtifactRecord{
			CampaignID: campaignID,
			Path:       normalizedPath,
			Content:    content,
			ReadOnly:   false,
			CreatedAt:  now,
			UpdatedAt:  now,
		}
	default:
		return storage.CampaignArtifactRecord{}, err
	}
	if err := m.store.PutCampaignArtifact(ctx, record); err != nil {
		return storage.CampaignArtifactRecord{}, err
	}
	return m.store.GetCampaignArtifact(ctx, campaignID, normalizedPath)
}

// NormalizeArtifactPath validates a readable artifact path.
func NormalizeArtifactPath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("artifact path is required")
	}
	cleaned := pathpkg.Clean(strings.TrimPrefix(strings.ReplaceAll(path, `\`, "/"), "./"))
	if cleaned == "." || cleaned == "/" || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("artifact path %q is not allowed", path)
	}
	switch cleaned {
	case SkillsArtifactPath, StoryArtifactPath, MemoryArtifactPath:
		return cleaned, nil
	}
	if !strings.HasPrefix(cleaned, "working/") || !strings.HasSuffix(cleaned, ".md") {
		return "", fmt.Errorf("artifact path %q is not allowed", path)
	}
	slug := strings.TrimSuffix(strings.TrimPrefix(cleaned, "working/"), ".md")
	if slug == "" || strings.Contains(slug, "/") || !workingArtifactSlugPattern.MatchString(slug) {
		return "", fmt.Errorf("artifact path %q is not allowed", path)
	}
	return cleaned, nil
}

func normalizeWritableArtifactPath(path string) (string, error) {
	normalizedPath, err := NormalizeArtifactPath(path)
	if err != nil {
		return "", err
	}
	if normalizedPath == SkillsArtifactPath {
		return "", fmt.Errorf("%s is read-only", SkillsArtifactPath)
	}
	return normalizedPath, nil
}

func ensureArtifactIfMissing(ctx context.Context, store ArtifactStore, clock func() time.Time, campaignID string, record storage.CampaignArtifactRecord) error {
	_, err := store.GetCampaignArtifact(ctx, campaignID, record.Path)
	if err == nil {
		return nil
	}
	if err != storage.ErrNotFound {
		return err
	}
	now := clock().UTC()
	record.CreatedAt = now
	record.UpdatedAt = now
	return store.PutCampaignArtifact(ctx, record)
}

// resolveSkillsContent returns skills content from the instruction loader if
// available, falling back to the hardcoded default.
func (m *Manager) resolveSkillsContent() string {
	if m.instructionLoader != nil {
		content, err := m.instructionLoader.LoadSkills(DaggerheartSystem)
		if err == nil && strings.TrimSpace(content) != "" {
			return strings.TrimSpace(content)
		}
	}
	return defaultSkillsMarkdown()
}

func defaultSkillsMarkdown() string {
	return strings.TrimSpace(`
# GM Skills

You are the AI GM for this campaign. You are responsible for both narration and authoritative game-state changes.

## Operating rules

- Keep in-character narration separate from out-of-character table talk.
- Use interaction_scene_gm_output_commit for authoritative in-character narration.
- Use interaction_ooc_* tools for rules clarifications, pacing, consent checks, and other out-of-character coordination.
- Use tools for authoritative changes to scenes, interaction flow, rolls, and other game state.
- Do not claim a state change happened until the corresponding tool succeeds.

## Rules lookup

- Use system_reference_search and system_reference_read before improvising Daggerheart rulings, mechanics, or terminology.
- If the reference is ambiguous, say that the ruling is an interpretation.

## Campaign documents

- Read story.md for campaign-specific setup, starter context, and ongoing plot notes.
- Read and update memory.md to keep durable GM memory between turns.
- You may create and update additional markdown notes under working/{slug}.md when you need scratch documents.
- Treat skills.md as the operating contract for this GM workflow. Do not attempt to overwrite it.
`)
}
