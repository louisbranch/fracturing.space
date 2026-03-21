package gametools

import (
	"context"
	"encoding/json"
	"fmt"

	aiv1 "github.com/louisbranch/fracturing.space/api/gen/go/ai/v1"
	"github.com/louisbranch/fracturing.space/internal/services/ai/orchestration"
)

// --- Input types ---

type referenceSearchInput struct {
	System     string `json:"system,omitempty"`
	Query      string `json:"query"`
	MaxResults int    `json:"max_results,omitempty"`
}

type referenceReadInput struct {
	System     string `json:"system,omitempty"`
	DocumentID string `json:"document_id"`
}

// --- Result types ---

type referenceSearchEntry struct {
	System     string   `json:"system"`
	DocumentID string   `json:"document_id"`
	Title      string   `json:"title"`
	Kind       string   `json:"kind,omitempty"`
	Path       string   `json:"path,omitempty"`
	Aliases    []string `json:"aliases,omitempty"`
	Snippet    string   `json:"snippet,omitempty"`
}

type referenceSearchResult struct {
	Results []referenceSearchEntry `json:"results"`
}

type referenceDocumentResult struct {
	System     string   `json:"system"`
	DocumentID string   `json:"document_id"`
	Title      string   `json:"title"`
	Kind       string   `json:"kind,omitempty"`
	Path       string   `json:"path,omitempty"`
	Aliases    []string `json:"aliases,omitempty"`
	Content    string   `json:"content"`
}

// --- Handlers ---

func (s *DirectSession) referenceSearch(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input referenceSearchInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Reference.SearchSystemReference(callCtx, &aiv1.SearchSystemReferenceRequest{
		System:     input.System,
		Query:      input.Query,
		MaxResults: int32(input.MaxResults),
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("system reference search failed: %w", err)
	}
	result := referenceSearchResult{Results: make([]referenceSearchEntry, 0, len(resp.GetResults()))}
	for _, item := range resp.GetResults() {
		result.Results = append(result.Results, referenceSearchEntry{
			System:     item.GetSystem(),
			DocumentID: item.GetDocumentId(),
			Title:      item.GetTitle(),
			Kind:       item.GetKind(),
			Path:       item.GetPath(),
			Aliases:    append([]string(nil), item.GetAliases()...),
			Snippet:    item.GetSnippet(),
		})
	}
	return toolResultJSON(result)
}

func (s *DirectSession) referenceRead(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input referenceReadInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	callCtx, cancel := outgoingContext(ctx, s.sc)
	defer cancel()

	resp, err := s.clients.Reference.ReadSystemReferenceDocument(callCtx, &aiv1.ReadSystemReferenceDocumentRequest{
		System:     input.System,
		DocumentId: input.DocumentID,
	})
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("system reference read failed: %w", err)
	}
	doc := resp.GetDocument()
	return toolResultJSON(referenceDocumentResult{
		System:     doc.GetSystem(),
		DocumentID: doc.GetDocumentId(),
		Title:      doc.GetTitle(),
		Kind:       doc.GetKind(),
		Path:       doc.GetPath(),
		Aliases:    append([]string(nil), doc.GetAliases()...),
		Content:    doc.GetContent(),
	})
}
