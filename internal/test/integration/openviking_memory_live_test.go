//go:build integration && liveai

package integration

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/ai/openviking"
)

func TestOpenVikingSessionMemoryLive(t *testing.T) {
	baseURL := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_AI_OPENVIKING_BASE_URL"))
	if baseURL == "" {
		t.Skip("FRACTURING_SPACE_AI_OPENVIKING_BASE_URL is required")
	}

	client, err := openviking.New(openviking.Config{
		BaseURL: baseURL,
		APIKey:  strings.TrimSpace(os.Getenv("FRACTURING_SPACE_AI_OPENVIKING_API_KEY")),
		Timeout: 60 * time.Second,
	})
	if err != nil {
		t.Fatalf("new openviking client: %v", err)
	}

	sessionID := fmt.Sprintf("integration-openviking-memory-%d", time.Now().UTC().UnixNano())
	ctx := context.Background()
	if err := client.AddMessage(ctx, sessionID, openviking.AddMessageInput{
		Role:    "user",
		Content: "Aria studies the flooded ledger vault and asks who on the docks can still be trusted.",
	}); err != nil {
		t.Fatalf("add user message: %v", err)
	}
	if err := client.AddMessage(ctx, sessionID, openviking.AddMessageInput{
		Role: "assistant",
		Parts: []openviking.MessagePart{
			{Type: "text", Text: "Dockmaster Harl distrusts magic but still owes Aria a favor from the harbor debt."},
		},
	}); err != nil {
		t.Fatalf("add assistant message: %v", err)
	}

	_, err = client.GetSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("load session after first message: %v", err)
	}

	commit, err := client.Commit(ctx, sessionID, true)
	if err != nil {
		t.Fatalf("commit session: %v", err)
	}
	deadline := time.Now().Add(45 * time.Second)
	if strings.TrimSpace(commit.Status) == "" {
		t.Fatalf("commit result = %#v, want non-empty status", commit)
	}

	session, err := client.GetSession(ctx, sessionID)
	if err != nil {
		t.Fatalf("get session: %v", err)
	}
	memoryRoot := fmt.Sprintf("viking://user/%s/memories/", strings.TrimSpace(session.User.UserID))
	if strings.TrimSpace(session.User.UserID) == "" {
		t.Fatalf("session user = %#v, want concrete user id", session.User)
	}

	var result openviking.SearchResult
	for {
		result, err = client.Find(ctx, openviking.SearchInput{
			Query:     "Who distrusts magic at the harbor and what do they owe Aria?",
			TargetURI: memoryRoot,
			Limit:     5,
		})
		if err == nil && len(result.Memories) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("memory search did not return hits: result=%#v err=%v", result, err)
		}
		time.Sleep(1500 * time.Millisecond)
	}
}

func TestOpenVikingResourceSearchLive(t *testing.T) {
	baseURL := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_AI_OPENVIKING_BASE_URL"))
	if baseURL == "" {
		t.Skip("FRACTURING_SPACE_AI_OPENVIKING_BASE_URL is required")
	}

	client, err := openviking.New(openviking.Config{
		BaseURL: baseURL,
		APIKey:  strings.TrimSpace(os.Getenv("FRACTURING_SPACE_AI_OPENVIKING_API_KEY")),
		Timeout: 60 * time.Second,
	})
	if err != nil {
		t.Fatalf("new openviking client: %v", err)
	}

	hostMirrorRoot, visibleMirrorRoot := openVikingMirrorRoots(t)
	resourceDir := filepath.Join(hostMirrorRoot, "integration")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		t.Fatalf("create resource dir: %v", err)
	}
	path := filepath.Join(resourceDir, fmt.Sprintf("story-%d.md", time.Now().UTC().UnixNano()))
	const needle = "The harbor debt is tied to the Black Lantern and only Dockmaster Harl still knows the dawn collector."
	if err := os.WriteFile(path, []byte(needle), 0o600); err != nil {
		t.Fatalf("write resource file: %v", err)
	}

	targetURI := fmt.Sprintf("viking://resources/fracturing-space/integration/story-%d.md", time.Now().UTC().UnixNano())
	visiblePath, err := rewriteVisibleMirrorPath(path, hostMirrorRoot, visibleMirrorRoot)
	if err != nil {
		t.Fatalf("rewrite visible path: %v", err)
	}
	ctx := context.Background()
	added, err := client.AddResource(ctx, openviking.AddResourceInput{
		Path:    visiblePath,
		To:      targetURI,
		Reason:  "integration resource search smoke",
		Wait:    true,
		Timeout: 45 * time.Second,
	})
	if err != nil {
		t.Fatalf("add resource: %v", err)
	}
	searchRoot := strings.TrimSpace(added.RootURI)
	if searchRoot == "" {
		searchRoot = targetURI
	}

	deadline := time.Now().Add(45 * time.Second)
	var result openviking.SearchResult
	for {
		result, err = client.Find(ctx, openviking.SearchInput{
			Query:     "Who knows the dawn collector tied to the Black Lantern harbor debt?",
			TargetURI: searchRoot,
			Limit:     5,
		})
		if err == nil && len(result.Resources) > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("resource search did not return hits: result=%#v err=%v", result, err)
		}
		time.Sleep(1500 * time.Millisecond)
	}

	hit := result.Resources[0]
	if hit.IsLeaf || hit.Level >= 2 {
		content, err := client.Read(ctx, hit.URI)
		if err != nil {
			t.Fatalf("read resource %q: %v", hit.URI, err)
		}
		if !strings.Contains(content, "Black Lantern") {
			t.Fatalf("read content = %q, want Black Lantern context", content)
		}
		return
	}

	if backingURI := strings.TrimSpace(strings.TrimSuffix(hit.URI, "/.overview.md")); backingURI != "" && strings.HasSuffix(hit.URI, "/.overview.md") {
		content, err := client.Read(ctx, backingURI)
		if err != nil {
			t.Fatalf("read backing resource %q: %v", backingURI, err)
		}
		if !strings.Contains(content, "Black Lantern") {
			t.Fatalf("backing read content = %q, want Black Lantern context", content)
		}
		return
	}

	overview, err := client.Overview(ctx, hit.URI)
	if err != nil {
		t.Fatalf("overview resource %q: %v", hit.URI, err)
	}
	if !strings.Contains(overview, "Black Lantern") && !strings.Contains(overview, "harbor debt") {
		t.Fatalf("overview = %q, want resource summary", overview)
	}
}

func openVikingMirrorRoots(t *testing.T) (string, string) {
	t.Helper()

	hostRoot := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_AI_OPENVIKING_MIRROR_ROOT"))
	if hostRoot == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("resolve home dir: %v", err)
		}
		hostRoot = filepath.Join(homeDir, ".openviking", "data", "fracturing-space")
	}
	visibleRoot := strings.TrimSpace(os.Getenv("FRACTURING_SPACE_AI_OPENVIKING_VISIBLE_MIRROR_ROOT"))
	if visibleRoot == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("resolve home dir: %v", err)
		}
		defaultDataRoot := filepath.Join(homeDir, ".openviking", "data")
		rel, err := filepath.Rel(defaultDataRoot, hostRoot)
		if err == nil && rel != "." && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			visibleRoot = filepath.Join("/app/data", rel)
		}
	}
	if visibleRoot == "" {
		visibleRoot = hostRoot
	}
	return hostRoot, visibleRoot
}

func rewriteVisibleMirrorPath(localPath, hostRoot, visibleRoot string) (string, error) {
	rel, err := filepath.Rel(hostRoot, localPath)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("path %s is outside mirror root %s", localPath, hostRoot)
	}
	return filepath.Join(visibleRoot, rel), nil
}
