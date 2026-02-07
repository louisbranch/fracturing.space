package seed

// ScenarioFixture defines an action-focused scenario that expands into JSON-RPC steps.
type ScenarioFixture struct {
	Name      string                    `json:"name"`
	ExpectSSE bool                      `json:"expect_sse"`
	Blocks    map[string][]ScenarioStep `json:"blocks"`
	Steps     []ScenarioStep            `json:"steps"`
}

// ScenarioStep defines a single human-oriented action step.
type ScenarioStep struct {
	Name         string         `json:"name"`
	Use          string         `json:"use"`
	With         map[string]any `json:"with"`
	Action       string         `json:"action"`
	Tool         string         `json:"tool"`
	Args         map[string]any `json:"args"`
	URI          any            `json:"uri"`
	Expect       string         `json:"expect"`
	ExpectStatus int            `json:"expect_status"`
	ExpectPaths  map[string]any `json:"expect_paths"`
	// ExpectContains asserts that a JSON path array includes matching entries.
	ExpectContains  map[string]any `json:"expect_contains"`
	Capture         map[string]any `json:"capture"`
	Request         map[string]any `json:"request"`
	Method          string         `json:"method"`
	Params          map[string]any `json:"params"`
	Client          map[string]any `json:"client"`
	ProtocolVersion string         `json:"protocol_version"`
}

// BlackboxStep defines a single JSON-RPC request/expectation pair.
type BlackboxStep struct {
	Name         string         `json:"name"`
	ExpectStatus int            `json:"expect_status"`
	Request      map[string]any `json:"request"`
	ExpectPaths  map[string]any `json:"expect_paths"`
	// ExpectContains asserts that a JSON path array includes matching entries.
	ExpectContains map[string]any      `json:"expect_contains"`
	Captures       map[string][]string `json:"captures"`
}

// BlackboxFixture represents an expanded scenario ready for execution.
type BlackboxFixture struct {
	Name      string         `json:"name"`
	ExpectSSE bool           `json:"expect_sse"`
	Steps     []BlackboxStep `json:"steps"`
}

// CaptureDefaults defines common capture shortcuts to structuredContent ID paths.
var CaptureDefaults = map[string][]string{
	"campaign":    {"result.structuredContent.id", "result.structured_content.id"},
	"participant": {"result.structuredContent.id", "result.structured_content.id"},
	"character":   {"result.structuredContent.id", "result.structured_content.id"},
	"session":     {"result.structuredContent.id", "result.structured_content.id"},
}
