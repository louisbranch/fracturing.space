package script

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Shopify/go-lua"
)

// Scenario describes a Lua-scripted scenario and its steps.
type Scenario struct {
	Name  string
	Steps []Step
}

// Step represents a single scenario action and arguments.
type Step struct {
	System string
	Kind   string
	Args   map[string]any
}

// LoadFromFile loads a scenario from a Lua script file.
func LoadFromFile(path string, registerLuaTypes func(*lua.State)) (*Scenario, error) {
	return LoadFromFileWithOptions(path, true, registerLuaTypes)
}

// LoadFromFileWithOptions loads a scenario from a Lua script file.
// Comment validation can be disabled when the caller intentionally trusts
// script content (for example admin-authored scenario scripts).
func LoadFromFileWithOptions(path string, validateComments bool, registerLuaTypes func(*lua.State)) (*Scenario, error) {
	if registerLuaTypes == nil {
		return nil, fmt.Errorf("lua type registrar is required")
	}
	if validateComments {
		if err := validateScenarioComments(path); err != nil {
			return nil, err
		}
	}
	return loadScenarioFromFile(path, registerLuaTypes)
}

func loadScenarioFromFile(path string, registerLuaTypes func(*lua.State)) (*Scenario, error) {
	state := lua.NewState()
	lua.OpenLibraries(state)

	registerLuaTypes(state)

	if err := lua.LoadFile(state, path, ""); err != nil {
		return nil, fmt.Errorf("load lua: %w", err)
	}
	if err := state.ProtectedCall(0, 1, 0); err != nil {
		return nil, fmt.Errorf("run lua: %w", err)
	}

	if state.TypeOf(-1) != lua.TypeUserData {
		state.Pop(1)
		return nil, fmt.Errorf("scenario script must return Scenario")
	}
	ud := state.ToUserData(-1)
	state.Pop(1)
	scenario, ok := ud.(*Scenario)
	if !ok || scenario == nil {
		return nil, fmt.Errorf("scenario script returned invalid Scenario")
	}
	if strings.TrimSpace(scenario.Name) == "" {
		scenario.Name = strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
	}
	return scenario, nil
}

// validateScenarioComments fails fast so scenarios always ship with block intent.
func validateScenarioComments(path string) error {
	contents, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read scenario: %w", err)
	}
	lines := strings.Split(string(contents), "\n")
	blockStart := 0
	// Iterate one past the end to validate the final block without a trailing blank line.
	for i := 0; i <= len(lines); i++ {
		if i < len(lines) && strings.TrimSpace(lines[i]) != "" {
			continue
		}
		if err := validateScenarioBlock(path, lines, blockStart, i); err != nil {
			return err
		}
		blockStart = i + 1
	}
	return nil
}

// validateScenarioBlock enforces comment-first blocks for scenario steps.
func validateScenarioBlock(path string, lines []string, start int, end int) error {
	firstLineIndex := -1
	firstLineContent := ""
	hasStepCall := false
	for i := start; i < end; i++ {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "" {
			continue
		}
		if firstLineIndex == -1 {
			firstLineIndex = i
			firstLineContent = trimmed
		}
		if isScenarioStepCallLine(trimmed) {
			hasStepCall = true
		}
	}
	if hasStepCall && firstLineIndex != -1 && !strings.HasPrefix(firstLineContent, "--") {
		return fmt.Errorf("scenario block missing comment at %s:%d", path, firstLineIndex+1)
	}
	return nil
}

func isScenarioStepCallLine(line string) bool {
	colon := strings.Index(line, ":")
	if colon <= 0 {
		return false
	}
	receiver := strings.TrimSpace(line[:colon])
	if receiver == "" {
		return false
	}
	return isLuaIdentifier(receiver)
}

func isLuaIdentifier(value string) bool {
	for i, r := range value {
		switch {
		case i == 0 && ((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r == '_'):
		case i > 0 && ((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_'):
		default:
			return false
		}
	}
	return true
}
