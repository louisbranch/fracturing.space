package gametools

import (
	"context"
	"encoding/json"
	"fmt"

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
	if s.clients.Reference == nil {
		return orchestration.ToolResult{}, fmt.Errorf("reference corpus is not configured")
	}

	results, err := s.clients.Reference.Search(ctx, input.System, input.Query, input.MaxResults)
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("system reference search failed: %w", err)
	}
	result := referenceSearchResult{Results: make([]referenceSearchEntry, 0, len(results))}
	for _, item := range results {
		result.Results = append(result.Results, referenceSearchEntry{
			System:     item.System,
			DocumentID: item.DocumentID,
			Title:      item.Title,
			Kind:       item.Kind,
			Path:       item.Path,
			Aliases:    append([]string(nil), item.Aliases...),
			Snippet:    item.Snippet,
		})
	}
	return toolResultJSON(result)
}

func (s *DirectSession) referenceRead(ctx context.Context, argsJSON []byte) (orchestration.ToolResult, error) {
	var input referenceReadInput
	if err := json.Unmarshal(argsJSON, &input); err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("unmarshal args: %w", err)
	}
	if s.clients.Reference == nil {
		return orchestration.ToolResult{}, fmt.Errorf("reference corpus is not configured")
	}

	doc, err := s.clients.Reference.Read(ctx, input.System, input.DocumentID)
	if err != nil {
		return orchestration.ToolResult{}, fmt.Errorf("system reference read failed: %w", err)
	}
	return toolResultJSON(referenceDocumentResult{
		System:     doc.System,
		DocumentID: doc.DocumentID,
		Title:      doc.Title,
		Kind:       doc.Kind,
		Path:       doc.Path,
		Aliases:    append([]string(nil), doc.Aliases...),
		Content:    doc.Content,
	})
}
