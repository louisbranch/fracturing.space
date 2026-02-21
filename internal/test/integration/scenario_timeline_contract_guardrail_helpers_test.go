//go:build integration

package integration

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/services/game/domain/command"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
)

var timelineTypePattern = regexp.MustCompile(`[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)+`)

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

func loadTimelineCommandAndEventTypes(docPath string) (map[string]struct{}, map[string]struct{}, error) {
	file, err := os.Open(docPath)
	if err != nil {
		return nil, nil, err
	}
	defer file.Close()

	commandTypes := make(map[string]struct{})
	eventTypes := make(map[string]struct{})
	commandColumn := -1
	eventColumn := -1

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		cells := markdownCells(scanner.Text())
		if len(cells) == 0 {
			continue
		}

		if commandIdx, eventIdx, ok := commandEventHeaderColumns(cells); ok {
			commandColumn = commandIdx
			eventColumn = eventIdx
			continue
		}
		if commandColumn < 0 || eventColumn < 0 {
			continue
		}
		if isMarkdownSeparatorRow(cells) {
			continue
		}
		if commandColumn >= len(cells) || eventColumn >= len(cells) {
			continue
		}

		for _, commandType := range timelineTypePattern.FindAllString(cells[commandColumn], -1) {
			commandTypes[commandType] = struct{}{}
		}
		for _, eventType := range timelineTypePattern.FindAllString(cells[eventColumn], -1) {
			eventTypes[eventType] = struct{}{}
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, err
	}

	return commandTypes, eventTypes, nil
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

func commandEventHeaderColumns(cells []string) (int, int, bool) {
	commandColumn := -1
	eventColumn := -1
	for i, cell := range cells {
		header := strings.TrimSpace(strings.Trim(cell, "`"))
		switch header {
		case "Command Type(s)":
			commandColumn = i
		case "Emitted Event Type(s)":
			eventColumn = i
		}
	}
	return commandColumn, eventColumn, commandColumn >= 0 && eventColumn >= 0
}

func isMarkdownSeparatorRow(cells []string) bool {
	if len(cells) == 0 {
		return false
	}
	for _, cell := range cells {
		trimmed := strings.TrimSpace(cell)
		if trimmed == "" {
			return false
		}
		for _, r := range trimmed {
			if r != '-' && r != ':' {
				return false
			}
		}
	}
	return true
}

func validateTimelineCoverageForMarkers(
	markerScenarios map[string]struct{},
	indexRows map[string]string,
	timelineRowIDs map[string]struct{},
) error {
	if len(markerScenarios) == 0 {
		return fmt.Errorf("expected at least one marked scenario")
	}
	for scenario := range markerScenarios {
		rowID, ok := indexRows[scenario]
		if !ok {
			return fmt.Errorf("missing scenario timeline index row for %s", scenario)
		}
		if _, ok := timelineRowIDs[rowID]; !ok {
			return fmt.Errorf("scenario %s maps to unknown timeline row id %s", scenario, rowID)
		}
	}
	return nil
}

func TestValidateTimelineCoverageForMarkers_RequiresMarkedScenarios(t *testing.T) {
	indexRows := map[string]string{
		"any-scenario": "P1",
	}
	timelineRowIDs := map[string]struct{}{
		"P1": {},
	}
	if err := validateTimelineCoverageForMarkers(map[string]struct{}{}, indexRows, timelineRowIDs); err == nil {
		t.Fatal("expected error for empty marker set")
	}
}

func missingDaggerheartTimelineCommandTypes(documented map[string]struct{}, definitions []command.Definition) []string {
	missing := make([]string, 0)
	for _, definition := range definitions {
		commandType := strings.TrimSpace(string(definition.Type))
		if !isDaggerheartTimelineTrackedCommandType(commandType) {
			continue
		}
		if _, ok := documented[commandType]; ok {
			continue
		}
		missing = append(missing, commandType)
	}
	sort.Strings(missing)
	return missing
}

func missingDaggerheartTimelineEventTypes(documented map[string]struct{}, definitions []event.Definition) []string {
	missing := make([]string, 0)
	for _, definition := range definitions {
		eventType := strings.TrimSpace(string(definition.Type))
		if !isDaggerheartTimelineTrackedEventType(eventType) {
			continue
		}
		if _, ok := documented[eventType]; ok {
			continue
		}
		missing = append(missing, eventType)
	}
	sort.Strings(missing)
	return missing
}

func isDaggerheartTimelineTrackedCommandType(commandType string) bool {
	if strings.HasPrefix(commandType, "sys.daggerheart.") {
		return true
	}
	switch commandType {
	case
		"action.roll.resolve",
		"action.outcome.apply",
		"action.outcome.reject",
		"session.gate_open",
		"session.spotlight_set",
		"story.note.add":
		return true
	default:
		return false
	}
}

func isDaggerheartTimelineTrackedEventType(eventType string) bool {
	if strings.HasPrefix(eventType, "sys.daggerheart.") {
		return true
	}
	switch eventType {
	case
		"action.roll_resolved",
		"action.outcome_applied",
		"action.outcome_rejected",
		"session.gate_opened",
		"session.spotlight_set",
		"story.note_added":
		return true
	default:
		return false
	}
}
