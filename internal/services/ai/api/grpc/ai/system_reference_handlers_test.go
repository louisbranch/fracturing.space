package ai

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext"
	"github.com/louisbranch/fracturing.space/internal/services/ai/campaigncontext/referencecorpus"
)

func TestSystemReferenceHandlersRoundTrip(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "index.json"), []byte(`[
		{
			"id": "combat-basics",
			"title": "Combat Basics",
			"kind": "chapter",
			"path": "combat-basics.md",
			"aliases": ["combat"]
		}
	]`), 0o600); err != nil {
		t.Fatalf("write reference index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "combat-basics.md"), []byte("Combat resolves with duality rolls."), 0o600); err != nil {
		t.Fatalf("write reference document: %v", err)
	}

	svc := NewSystemReferenceHandlers(referencecorpus.New(root))

	searchResp, err := svc.SearchSystemReference(context.Background(), &aiv1.SearchSystemReferenceRequest{
		System:     campaigncontext.DaggerheartSystem,
		Query:      "combat",
		MaxResults: 5,
	})
	if err != nil {
		t.Fatalf("SearchSystemReference() error = %v", err)
	}
	if len(searchResp.GetResults()) != 1 {
		t.Fatalf("SearchSystemReference() results = %d, want 1", len(searchResp.GetResults()))
	}
	if got := searchResp.GetResults()[0].GetDocumentId(); got != "combat-basics" {
		t.Fatalf("SearchSystemReference() document id = %q, want %q", got, "combat-basics")
	}

	readResp, err := svc.ReadSystemReferenceDocument(context.Background(), &aiv1.ReadSystemReferenceDocumentRequest{
		System:     campaigncontext.DaggerheartSystem,
		DocumentId: "combat-basics",
	})
	if err != nil {
		t.Fatalf("ReadSystemReferenceDocument() error = %v", err)
	}
	if got := readResp.GetDocument().GetContent(); got != "Combat resolves with duality rolls." {
		t.Fatalf("ReadSystemReferenceDocument() content = %q", got)
	}
}
