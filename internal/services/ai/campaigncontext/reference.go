package campaigncontext

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const (
	defaultReferenceSearchResults = 5
	maxReferenceSearchResults     = 10
)

// ReferenceDocument contains one readable rules/reference document.
type ReferenceDocument struct {
	System     string
	DocumentID string
	Title      string
	Kind       string
	Path       string
	Aliases    []string
	Content    string
}

// ReferenceSearchResult contains one search hit.
type ReferenceSearchResult struct {
	System     string
	DocumentID string
	Title      string
	Kind       string
	Path       string
	Aliases    []string
	Snippet    string
}

type referenceIndexEntry struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Kind    string   `json:"kind"`
	Path    string   `json:"path"`
	Aliases []string `json:"aliases"`
}

// ReferenceCorpus serves a read-only filesystem-backed system corpus.
type ReferenceCorpus struct {
	root    string
	loadErr error
	once    sync.Once
	entries []referenceIndexEntry
}

// NewReferenceCorpus builds a read-only reference corpus rooted at one local directory.
func NewReferenceCorpus(root string) *ReferenceCorpus {
	return &ReferenceCorpus{root: strings.TrimSpace(root)}
}

// Search returns ranked system-reference hits for a query.
func (c *ReferenceCorpus) Search(ctx context.Context, system string, query string, maxResults int) ([]ReferenceSearchResult, error) {
	if err := validateReferenceSystem(system); err != nil {
		return nil, err
	}
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("query is required")
	}
	entries, err := c.indexEntries()
	if err != nil {
		return nil, err
	}
	if maxResults <= 0 {
		maxResults = defaultReferenceSearchResults
	}
	if maxResults > maxReferenceSearchResults {
		maxResults = maxReferenceSearchResults
	}

	queryLower := strings.ToLower(query)
	queryTerms := strings.Fields(queryLower)
	type scoredResult struct {
		score  int
		result ReferenceSearchResult
	}
	scored := make([]scoredResult, 0, len(entries))
	seen := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		score := metadataScore(entry, queryLower, queryTerms)
		if score == 0 {
			continue
		}
		scored = append(scored, scoredResult{
			score: score,
			result: ReferenceSearchResult{
				System:     DaggerheartSystem,
				DocumentID: strings.TrimSpace(entry.ID),
				Title:      strings.TrimSpace(entry.Title),
				Kind:       strings.TrimSpace(entry.Kind),
				Path:       strings.TrimSpace(entry.Path),
				Aliases:    append([]string(nil), entry.Aliases...),
				Snippet:    metadataSnippet(entry),
			},
		})
		seen[strings.TrimSpace(entry.ID)] = struct{}{}
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].result.Title < scored[j].result.Title
	})

	results := make([]ReferenceSearchResult, 0, maxResults)
	for _, item := range scored {
		if len(results) == maxResults {
			return results, nil
		}
		results = append(results, item.result)
	}
	if len(results) == maxResults {
		return results, nil
	}

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if _, ok := seen[strings.TrimSpace(entry.ID)]; ok {
			continue
		}
		content, err := c.readEntryContent(entry)
		if err != nil {
			return nil, err
		}
		contentLower := strings.ToLower(content)
		if !strings.Contains(contentLower, queryLower) && !containsAllTerms(contentLower, queryTerms) {
			continue
		}
		results = append(results, ReferenceSearchResult{
			System:     DaggerheartSystem,
			DocumentID: strings.TrimSpace(entry.ID),
			Title:      strings.TrimSpace(entry.Title),
			Kind:       strings.TrimSpace(entry.Kind),
			Path:       strings.TrimSpace(entry.Path),
			Aliases:    append([]string(nil), entry.Aliases...),
			Snippet:    contentSnippet(content, queryLower),
		})
		if len(results) == maxResults {
			break
		}
	}
	return results, nil
}

// Read returns one full reference document by document id.
func (c *ReferenceCorpus) Read(ctx context.Context, system string, documentID string) (ReferenceDocument, error) {
	if err := validateReferenceSystem(system); err != nil {
		return ReferenceDocument{}, err
	}
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return ReferenceDocument{}, fmt.Errorf("document id is required")
	}
	entries, err := c.indexEntries()
	if err != nil {
		return ReferenceDocument{}, err
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return ReferenceDocument{}, err
		}
		if strings.TrimSpace(entry.ID) != documentID && strings.TrimSpace(entry.Path) != documentID {
			continue
		}
		content, err := c.readEntryContent(entry)
		if err != nil {
			return ReferenceDocument{}, err
		}
		return ReferenceDocument{
			System:     DaggerheartSystem,
			DocumentID: strings.TrimSpace(entry.ID),
			Title:      strings.TrimSpace(entry.Title),
			Kind:       strings.TrimSpace(entry.Kind),
			Path:       strings.TrimSpace(entry.Path),
			Aliases:    append([]string(nil), entry.Aliases...),
			Content:    content,
		}, nil
	}
	return ReferenceDocument{}, fmt.Errorf("reference document %q not found", documentID)
}

func (c *ReferenceCorpus) indexEntries() ([]referenceIndexEntry, error) {
	if c == nil {
		return nil, fmt.Errorf("reference corpus is not configured")
	}
	c.once.Do(func() {
		root := strings.TrimSpace(c.root)
		if root == "" {
			c.loadErr = fmt.Errorf("reference root is not configured")
			return
		}
		data, err := os.ReadFile(filepath.Join(root, "index.json"))
		if err != nil {
			c.loadErr = fmt.Errorf("read reference index: %w", err)
			return
		}
		if err := json.Unmarshal(data, &c.entries); err != nil {
			c.loadErr = fmt.Errorf("parse reference index: %w", err)
			return
		}
	})
	if c.loadErr != nil {
		return nil, c.loadErr
	}
	return append([]referenceIndexEntry(nil), c.entries...), nil
}

func (c *ReferenceCorpus) readEntryContent(entry referenceIndexEntry) (string, error) {
	root := strings.TrimSpace(c.root)
	if root == "" {
		return "", fmt.Errorf("reference root is not configured")
	}
	cleanPath := filepath.Clean(filepath.Join(root, filepath.FromSlash(strings.TrimSpace(entry.Path))))
	if !strings.HasPrefix(cleanPath, filepath.Clean(root)+string(os.PathSeparator)) && cleanPath != filepath.Clean(root) {
		return "", fmt.Errorf("reference path %q escapes root", entry.Path)
	}
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("read reference document %q: %w", entry.ID, err)
	}
	return string(data), nil
}

func validateReferenceSystem(system string) error {
	if strings.EqualFold(strings.TrimSpace(system), "") || strings.EqualFold(strings.TrimSpace(system), DaggerheartSystem) {
		return nil
	}
	return fmt.Errorf("system %q is not supported", system)
}

func metadataScore(entry referenceIndexEntry, queryLower string, queryTerms []string) int {
	score := 0
	idLower := strings.ToLower(strings.TrimSpace(entry.ID))
	titleLower := strings.ToLower(strings.TrimSpace(entry.Title))
	kindLower := strings.ToLower(strings.TrimSpace(entry.Kind))
	pathLower := strings.ToLower(strings.TrimSpace(entry.Path))
	switch {
	case titleLower == queryLower:
		score += 120
	case strings.Contains(titleLower, queryLower):
		score += 100
	}
	if idLower == queryLower {
		score += 110
	} else if strings.Contains(idLower, queryLower) {
		score += 90
	}
	if kindLower == queryLower {
		score += 60
	} else if strings.Contains(kindLower, queryLower) {
		score += 40
	}
	if strings.Contains(pathLower, queryLower) {
		score += 35
	}
	for _, alias := range entry.Aliases {
		aliasLower := strings.ToLower(strings.TrimSpace(alias))
		if aliasLower == queryLower {
			score += 105
			continue
		}
		if strings.Contains(aliasLower, queryLower) {
			score += 80
		}
	}
	if score == 0 {
		return 0
	}
	for _, term := range queryTerms {
		if term == "" {
			continue
		}
		if strings.Contains(titleLower, term) || strings.Contains(idLower, term) || strings.Contains(kindLower, term) || strings.Contains(pathLower, term) {
			score += 10
			continue
		}
		for _, alias := range entry.Aliases {
			if strings.Contains(strings.ToLower(strings.TrimSpace(alias)), term) {
				score += 10
				break
			}
		}
	}
	return score
}

func metadataSnippet(entry referenceIndexEntry) string {
	parts := make([]string, 0, 3)
	if title := strings.TrimSpace(entry.Title); title != "" {
		parts = append(parts, title)
	}
	if kind := strings.TrimSpace(entry.Kind); kind != "" {
		parts = append(parts, "kind: "+kind)
	}
	if len(entry.Aliases) != 0 {
		parts = append(parts, "aliases: "+strings.Join(entry.Aliases, ", "))
	}
	return strings.Join(parts, " | ")
}

func containsAllTerms(text string, terms []string) bool {
	for _, term := range terms {
		if term == "" {
			continue
		}
		if !strings.Contains(text, term) {
			return false
		}
	}
	return true
}

func contentSnippet(content string, queryLower string) string {
	if strings.TrimSpace(content) == "" {
		return ""
	}
	content = strings.ReplaceAll(content, "\n", " ")
	content = strings.ReplaceAll(content, "\r", " ")
	lower := strings.ToLower(content)
	index := strings.Index(lower, queryLower)
	if index == -1 {
		if len(content) <= 200 {
			return content
		}
		return strings.TrimSpace(content[:200])
	}
	start := index - 80
	if start < 0 {
		start = 0
	}
	end := index + len(queryLower) + 80
	if end > len(content) {
		end = len(content)
	}
	return strings.TrimSpace(content[start:end])
}
