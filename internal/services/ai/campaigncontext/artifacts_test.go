package campaigncontext

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/storage"
	"github.com/louisbranch/fracturing.space/internal/test/mock/aifakes"
)

func TestManagerEnsureDefaultArtifactsSeedsDefaults(t *testing.T) {
	store := aifakes.NewCampaignArtifactStore()
	now := time.Date(2026, 3, 14, 1, 30, 0, 0, time.UTC)
	manager := NewManager(ManagerConfig{
		Store: store,
		Clock: func() time.Time { return now },
	})

	records, err := manager.EnsureDefaultArtifacts(context.Background(), "campaign-1", " Starter story ")
	if err != nil {
		t.Fatalf("EnsureDefaultArtifacts() error = %v", err)
	}
	if len(records) != 3 {
		t.Fatalf("artifact count = %d, want 3", len(records))
	}

	skills, err := store.GetCampaignArtifact(context.Background(), "campaign-1", SkillsArtifactPath)
	if err != nil {
		t.Fatalf("get skills artifact: %v", err)
	}
	if !skills.ReadOnly {
		t.Fatal("skills.md should be read-only")
	}
	if !strings.Contains(skills.Content, "GM Skills") {
		t.Fatalf("skills content missing GM contract: %q", skills.Content)
	}

	memory, err := store.GetCampaignArtifact(context.Background(), "campaign-1", MemoryArtifactPath)
	if err != nil {
		t.Fatalf("get memory artifact: %v", err)
	}
	if memory.ReadOnly {
		t.Fatal("memory.md should be writable")
	}
	if memory.Content != "" {
		t.Fatalf("memory content = %q, want empty", memory.Content)
	}

	story, err := store.GetCampaignArtifact(context.Background(), "campaign-1", StoryArtifactPath)
	if err != nil {
		t.Fatalf("get story artifact: %v", err)
	}
	if story.Content != "Starter story" {
		t.Fatalf("story content = %q, want %q", story.Content, "Starter story")
	}
	if !story.CreatedAt.Equal(now) || !story.UpdatedAt.Equal(now) {
		t.Fatalf("story timestamps = (%s, %s), want %s", story.CreatedAt, story.UpdatedAt, now)
	}
}

func TestManagerUpsertArtifactValidatesWritablePaths(t *testing.T) {
	store := aifakes.NewCampaignArtifactStore()
	now := time.Date(2026, 3, 14, 1, 31, 0, 0, time.UTC)
	manager := NewManager(ManagerConfig{
		Store: store,
		Clock: func() time.Time { return now },
	})

	if _, err := manager.UpsertArtifact(context.Background(), "campaign-1", SkillsArtifactPath, "nope"); err == nil || !strings.Contains(err.Error(), "read-only") {
		t.Fatalf("UpsertArtifact(skills.md) error = %v, want read-only", err)
	}

	record, err := manager.UpsertArtifact(context.Background(), "campaign-1", "working/session-notes.md", "Notes")
	if err != nil {
		t.Fatalf("UpsertArtifact() error = %v", err)
	}
	if record.Path != "working/session-notes.md" {
		t.Fatalf("path = %q, want %q", record.Path, "working/session-notes.md")
	}
	if record.Content != "Notes" {
		t.Fatalf("content = %q, want %q", record.Content, "Notes")
	}

	now = now.Add(5 * time.Minute)
	manager.clock = func() time.Time { return now }
	record, err = manager.UpsertArtifact(context.Background(), "campaign-1", "./working/session-notes.md", "Updated")
	if err != nil {
		t.Fatalf("UpsertArtifact(update) error = %v", err)
	}
	if record.Content != "Updated" {
		t.Fatalf("updated content = %q, want %q", record.Content, "Updated")
	}
	if !record.UpdatedAt.Equal(now) {
		t.Fatalf("updated_at = %s, want %s", record.UpdatedAt, now)
	}
}

func TestNormalizeArtifactPath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "skills", path: "skills.md", want: "skills.md"},
		{name: "story", path: " ./story.md ", want: "story.md"},
		{name: "working note", path: "working/gm_notes-1.md", want: "working/gm_notes-1.md"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := NormalizeArtifactPath(tt.path)
			if err != nil {
				t.Fatalf("NormalizeArtifactPath() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("NormalizeArtifactPath() = %q, want %q", got, tt.want)
			}
		})
	}

	for _, path := range []string{"", "../story.md", "working/nested/path.md", "notes.txt"} {
		if _, err := NormalizeArtifactPath(path); err == nil {
			t.Fatalf("NormalizeArtifactPath(%q) expected error", path)
		}
	}
}

func TestManagerGetArtifactRequiresStoreAndCampaignID(t *testing.T) {
	manager := NewManager(ManagerConfig{})
	if _, err := manager.GetArtifact(context.Background(), "campaign-1", StoryArtifactPath); err == nil {
		t.Fatal("expected missing store error")
	}

	store := aifakes.NewCampaignArtifactStore()
	manager = NewManager(ManagerConfig{Store: store})
	if _, err := manager.ListArtifacts(context.Background(), " "); err == nil {
		t.Fatal("expected campaign id validation error")
	}
}

func TestManagerEnsureDefaultArtifactsUsesConfiguredSkillsLoader(t *testing.T) {
	store := aifakes.NewCampaignArtifactStore()
	manager := NewManager(ManagerConfig{
		Store: store,
		SkillsLoader: stubSkillsLoader{
			content: "# Custom Skills\nUse the loaded instruction set.",
		},
		DefaultSystem: "custom-system",
	})

	_, err := manager.EnsureDefaultArtifacts(context.Background(), "campaign-1", "")
	if err != nil {
		t.Fatalf("EnsureDefaultArtifacts() error = %v", err)
	}

	skills, err := store.GetCampaignArtifact(context.Background(), "campaign-1", SkillsArtifactPath)
	if err != nil {
		t.Fatalf("get skills artifact: %v", err)
	}
	if !strings.Contains(skills.Content, "Custom Skills") {
		t.Fatalf("skills content = %q, want loaded skills", skills.Content)
	}
}

func TestEnsureArtifactIfMissingPropagatesStoreErrors(t *testing.T) {
	store := &artifactStoreStub{getErr: context.DeadlineExceeded}
	err := ensureArtifactIfMissing(context.Background(), store, time.Now, "campaign-1", storage.CampaignArtifactRecord{Path: StoryArtifactPath})
	if err != context.DeadlineExceeded {
		t.Fatalf("ensureArtifactIfMissing() error = %v, want %v", err, context.DeadlineExceeded)
	}
}

type artifactStoreStub struct {
	getErr error
}

type stubSkillsLoader struct {
	content string
	err     error
}

func (s stubSkillsLoader) LoadSkills(string) (string, error) {
	return s.content, s.err
}

func (s *artifactStoreStub) PutCampaignArtifact(context.Context, storage.CampaignArtifactRecord) error {
	return nil
}

func (s *artifactStoreStub) GetCampaignArtifact(context.Context, string, string) (storage.CampaignArtifactRecord, error) {
	return storage.CampaignArtifactRecord{}, s.getErr
}

func (s *artifactStoreStub) ListCampaignArtifacts(context.Context, string) ([]storage.CampaignArtifactRecord, error) {
	return nil, nil
}
