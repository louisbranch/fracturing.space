package referencecorpus

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const repoPlaybookRelativeDir = "docs/reference/daggerheart-playbooks"

func loadRepoPlaybookEntries() ([]indexEntry, error) {
	dir, err := repoPlaybookDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read repo playbook dir: %w", err)
	}
	var result []indexEntry
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(entry.Name()), ".md") {
			continue
		}
		fullPath := filepath.Join(dir, entry.Name())
		index, err := loadRepoPlaybookEntry(fullPath)
		if err != nil {
			return nil, err
		}
		result = append(result, index)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].ID < result[j].ID
	})
	return result, nil
}

func loadRepoPlaybookEntry(fullPath string) (indexEntry, error) {
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return indexEntry{}, fmt.Errorf("read repo playbook %q: %w", fullPath, err)
	}
	meta, err := parseSimpleFrontMatter(string(data))
	if err != nil {
		return indexEntry{}, fmt.Errorf("parse repo playbook %q: %w", fullPath, err)
	}
	id := strings.TrimSpace(meta["id"])
	title := strings.TrimSpace(meta["title"])
	kind := strings.TrimSpace(meta["kind"])
	if id == "" || title == "" || kind == "" {
		return indexEntry{}, fmt.Errorf("repo playbook %q is missing id, title, or kind front matter", fullPath)
	}
	aliases, err := frontMatterStringSlice(meta["aliases"])
	if err != nil {
		return indexEntry{}, fmt.Errorf("parse aliases for %q: %w", fullPath, err)
	}
	return indexEntry{
		ID:      id,
		Title:   title,
		Kind:    kind,
		Path:    filepath.ToSlash(filepath.Join("playbooks", filepath.Base(fullPath))),
		Aliases: aliases,
		absPath: fullPath,
	}, nil
}

func repoPlaybookDir() (string, error) {
	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("resolve repo playbook dir: runtime caller unavailable")
	}
	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "..", ".."))
	return filepath.Join(repoRoot, filepath.FromSlash(repoPlaybookRelativeDir)), nil
}

func parseSimpleFrontMatter(content string) (map[string]string, error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return nil, fmt.Errorf("front matter opening delimiter is required")
	}
	rest := strings.TrimPrefix(content, "---\n")
	idx := strings.Index(rest, "\n---\n")
	if idx < 0 {
		return nil, fmt.Errorf("front matter closing delimiter is required")
	}
	block := rest[:idx]
	lines := strings.Split(block, "\n")
	meta := make(map[string]string, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			return nil, fmt.Errorf("invalid front matter line %q", line)
		}
		meta[strings.TrimSpace(key)] = normalizeFrontMatterValue(strings.TrimSpace(value))
	}
	return meta, nil
}

func normalizeFrontMatterValue(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "[") {
		return value
	}
	var decoded string
	if err := json.Unmarshal([]byte(value), &decoded); err == nil {
		return strings.TrimSpace(decoded)
	}
	return strings.Trim(value, `"`)
}

func frontMatterStringSlice(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	var items []string
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out, nil
}

func mergeIndexEntries(rootEntries, repoEntries []indexEntry) []indexEntry {
	if len(repoEntries) == 0 {
		return rootEntries
	}
	merged := make(map[string]indexEntry, len(rootEntries)+len(repoEntries))
	order := make([]string, 0, len(rootEntries)+len(repoEntries))
	for _, entry := range rootEntries {
		if _, ok := merged[entry.ID]; !ok {
			order = append(order, entry.ID)
		}
		merged[entry.ID] = entry
	}
	for _, entry := range repoEntries {
		if _, ok := merged[entry.ID]; !ok {
			order = append(order, entry.ID)
		}
		merged[entry.ID] = entry
	}
	result := make([]indexEntry, 0, len(order))
	for _, id := range order {
		result = append(result, merged[id])
	}
	return result
}
