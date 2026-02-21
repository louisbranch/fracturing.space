//go:build integration

package integration

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

func loadMarkedScenarioFiles(scenarioDir, marker string) (map[string]struct{}, error) {
	out := make(map[string]struct{})
	if strings.TrimSpace(scenarioDir) == "" {
		return nil, fmt.Errorf("scenario directory is required")
	}
	if strings.TrimSpace(marker) == "" {
		return nil, fmt.Errorf("marker is required")
	}

	err := filepath.WalkDir(scenarioDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".lua" {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !strings.Contains(string(content), marker) {
			return nil
		}
		scenario := strings.TrimSuffix(filepath.Base(path), ".lua")
		out[scenario] = struct{}{}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return out, nil
}

func loadScenarioTimelineIndex(docPath string) (map[string]string, error) {
	file, err := os.Open(docPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	rows := make(map[string]string)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		cells := markdownCells(line)
		if len(cells) < 2 {
			continue
		}
		scenarioPath := strings.Trim(strings.TrimSpace(cells[0]), "`")
		if !strings.HasPrefix(scenarioPath, "internal/test/game/scenarios/") || !strings.HasSuffix(scenarioPath, ".lua") {
			continue
		}
		scenario := strings.TrimSuffix(filepath.Base(scenarioPath), ".lua")
		rowID := strings.Trim(strings.TrimSpace(cells[1]), "`")
		if scenario == "" || rowID == "" {
			continue
		}
		if existing, ok := rows[scenario]; ok && existing != rowID {
			return nil, fmt.Errorf("scenario %s has conflicting timeline row IDs: %s and %s", scenario, existing, rowID)
		}
		rows[scenario] = rowID
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return rows, nil
}

func loadTimelineRowIDs(docPath string) (map[string]struct{}, error) {
	file, err := os.Open(docPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	rows := make(map[string]struct{})
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		cells := markdownCells(line)
		if len(cells) == 0 {
			continue
		}
		rowID := strings.Trim(strings.TrimSpace(cells[0]), "`")
		if len(rowID) < 2 || rowID[0] != 'P' {
			continue
		}
		allDigits := true
		for _, r := range rowID[1:] {
			if r < '0' || r > '9' {
				allDigits = false
				break
			}
		}
		if !allDigits {
			continue
		}
		rows[rowID] = struct{}{}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return rows, nil
}

func markdownCells(line string) []string {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "|") || !strings.HasSuffix(trimmed, "|") {
		return nil
	}
	raw := strings.Split(trimmed, "|")
	if len(raw) < 3 {
		return nil
	}
	cells := make([]string, 0, len(raw)-2)
	for i := 1; i < len(raw)-1; i++ {
		cells = append(cells, strings.TrimSpace(raw[i]))
	}
	return cells
}
