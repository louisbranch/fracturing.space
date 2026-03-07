package scenario

import scenarioscript "github.com/louisbranch/fracturing.space/internal/tools/scenario/script"

// LoadScenarioFromFile loads a scenario from a Lua script file.
func LoadScenarioFromFile(path string) (*Scenario, error) {
	return LoadScenarioFromFileWithOptions(path, true)
}

// LoadScenarioFromFileWithOptions loads a scenario from a Lua script file.
// Comment validation can be disabled when the caller intentionally trusts
// script content (for example admin-authored scenario scripts).
func LoadScenarioFromFileWithOptions(path string, validateComments bool) (*Scenario, error) {
	return scenarioscript.LoadFromFileWithOptions(path, validateComments, registerLuaTypes)
}
