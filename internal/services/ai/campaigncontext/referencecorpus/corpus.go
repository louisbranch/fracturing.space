package referencecorpus

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
	defaultSearchResults = 5
	maxSearchResults     = 10
)

// Document contains one readable rules/reference document.
type Document struct {
	System     string
	DocumentID string
	Title      string
	Kind       string
	Path       string
	Aliases    []string
	Content    string
}

// SearchResult contains one search hit.
type SearchResult struct {
	System     string
	DocumentID string
	Title      string
	Kind       string
	Path       string
	Aliases    []string
	Snippet    string
}

type indexEntry struct {
	ID      string   `json:"id"`
	Title   string   `json:"title"`
	Kind    string   `json:"kind"`
	Path    string   `json:"path"`
	Aliases []string `json:"aliases"`
}

// Corpus serves a read-only filesystem-backed system corpus.
type Corpus struct {
	root    string
	loadErr error
	once    sync.Once
	entries []indexEntry
}

// New builds a read-only reference corpus rooted at one local directory.
func New(root string) *Corpus {
	return &Corpus{root: strings.TrimSpace(root)}
}

// Search returns ranked system-reference hits for a query.
func (c *Corpus) Search(ctx context.Context, system, query string, maxResults int) ([]SearchResult, error) {
	if err := validateSystem(system); err != nil {
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
		maxResults = defaultSearchResults
	}
	if maxResults > maxSearchResults {
		maxResults = maxSearchResults
	}

	queryLower := strings.ToLower(query)
	queryTerms := strings.Fields(queryLower)
	type scoredResult struct {
		score  int
		result SearchResult
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
			result: SearchResult{
				System:     supportedSystem,
				DocumentID: entry.ID,
				Title:      entry.Title,
				Kind:       entry.Kind,
				Path:       entry.Path,
				Aliases:    append([]string(nil), entry.Aliases...),
				Snippet:    metadataSnippet(entry),
			},
		})
		seen[entry.ID] = struct{}{}
	}
	sort.SliceStable(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		return scored[i].result.Title < scored[j].result.Title
	})

	results := make([]SearchResult, 0, maxResults)
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
		if _, ok := seen[entry.ID]; ok {
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
		results = append(results, SearchResult{
			System:     supportedSystem,
			DocumentID: entry.ID,
			Title:      entry.Title,
			Kind:       entry.Kind,
			Path:       entry.Path,
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
func (c *Corpus) Read(ctx context.Context, system, documentID string) (Document, error) {
	if err := validateSystem(system); err != nil {
		return Document{}, err
	}
	documentID = strings.TrimSpace(documentID)
	if documentID == "" {
		return Document{}, fmt.Errorf("document id is required")
	}
	entries, err := c.indexEntries()
	if err != nil {
		return Document{}, err
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return Document{}, err
		}
		if entry.ID != documentID && entry.Path != documentID {
			continue
		}
		content, err := c.readEntryContent(entry)
		if err != nil {
			return Document{}, err
		}
		return Document{
			System:     supportedSystem,
			DocumentID: entry.ID,
			Title:      entry.Title,
			Kind:       entry.Kind,
			Path:       entry.Path,
			Aliases:    append([]string(nil), entry.Aliases...),
			Content:    content,
		}, nil
	}
	return Document{}, fmt.Errorf("reference document %q not found", documentID)
}

func (c *Corpus) indexEntries() ([]indexEntry, error) {
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
	return append([]indexEntry(nil), c.entries...), nil
}

func (c *Corpus) readEntryContent(entry indexEntry) (string, error) {
	root := strings.TrimSpace(c.root)
	if root == "" {
		return "", fmt.Errorf("reference root is not configured")
	}
	cleanRoot := filepath.Clean(root)
	cleanPath := filepath.Clean(filepath.Join(cleanRoot, filepath.FromSlash(entry.Path)))
	if !strings.HasPrefix(cleanPath, cleanRoot+string(os.PathSeparator)) && cleanPath != cleanRoot {
		return "", fmt.Errorf("reference path %q escapes root", entry.Path)
	}
	data, err := os.ReadFile(cleanPath)
	if err != nil {
		return "", fmt.Errorf("read reference document %q: %w", entry.ID, err)
	}
	return string(data), nil
}
