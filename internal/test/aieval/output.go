package aieval

const (
	// RunStatusPassed indicates the live eval lane completed and passed its harness checks.
	RunStatusPassed = "passed"
	// RunStatusFailed indicates the live eval lane produced structured output but failed harness checks.
	RunStatusFailed = "failed"
	// MetricStatusPass indicates the run counts as a successful quality signal.
	MetricStatusPass = "pass"
	// MetricStatusFail indicates the run completed and exposed a model-quality failure.
	MetricStatusFail = "fail"
	// MetricStatusInvalid indicates the run did not produce a trustworthy quality signal.
	MetricStatusInvalid = "invalid"
)

// Output is the Promptfoo-facing JSON contract emitted by one live GM eval run.
type Output struct {
	CaseID                     string             `json:"case_id,omitempty"`
	Scenario                   string             `json:"scenario"`
	Label                      string             `json:"label,omitempty"`
	Model                      string             `json:"model"`
	ReasoningEffort            string             `json:"reasoning_effort,omitempty"`
	PromptProfile              string             `json:"prompt_profile,omitempty"`
	PromptContext              PromptContext      `json:"prompt_context,omitempty"`
	RunStatus                  string             `json:"run_status,omitempty"`
	MetricStatus               string             `json:"metric_status,omitempty"`
	FailureKind                string             `json:"failure_kind,omitempty"`
	FailureSummary             string             `json:"failure_summary,omitempty"`
	FailureReason              string             `json:"failure_reason,omitempty"`
	ResultClass                string             `json:"result_class,omitempty"`
	ToolNames                  []string           `json:"tool_names,omitempty"`
	ToolCalls                  []ToolCall         `json:"tool_calls,omitempty"`
	ToolErrorCount             int                `json:"tool_error_count"`
	ReferenceSearchCount       int                `json:"reference_search_count"`
	ReferenceReadCount         int                `json:"reference_read_count"`
	UnexpectedReferenceLookups int                `json:"unexpected_reference_lookup_count"`
	OutputText                 string             `json:"output_text,omitempty"`
	MemoryContent              string             `json:"memory_content,omitempty"`
	SkillsReadOnly             bool               `json:"skills_read_only"`
	TurnCount                  int                `json:"turn_count,omitempty"`
	Interaction                InteractionSummary `json:"interaction"`
	CharacterState             CharacterState     `json:"character_state"`
	Artifacts                  ArtifactPaths      `json:"artifacts"`
}

// PromptContext captures the resolved instruction-bundle metadata behind one prompt profile.
type PromptContext struct {
	Profile            string `json:"profile,omitempty"`
	InstructionsRoot   string `json:"instructions_root,omitempty"`
	InstructionsSource string `json:"instructions_source,omitempty"`
	InstructionsDigest string `json:"instructions_digest,omitempty"`
	Summary            string `json:"summary,omitempty"`
}

// ToolCall records one tool invocation from the tokenized replay fixture.
type ToolCall struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments,omitempty"`
}

// InteractionSummary captures the player-visible state the scenario left behind.
type InteractionSummary struct {
	ActiveSceneID    string   `json:"active_scene_id,omitempty"`
	PlayerPhaseOpen  bool     `json:"player_phase_open"`
	CurrentTitle     string   `json:"current_title,omitempty"`
	CurrentBeatTypes []string `json:"current_beat_types,omitempty"`
	PromptText       string   `json:"prompt_text,omitempty"`
}

// CharacterState captures the acting character's authoritative Daggerheart state after the run.
type CharacterState struct {
	HP     int `json:"hp"`
	Hope   int `json:"hope"`
	Stress int `json:"stress"`
	Armor  int `json:"armor"`
}

// ArtifactPaths points at the locally-written debugging artifacts for the run.
type ArtifactPaths struct {
	RawCapture     string `json:"raw_capture,omitempty"`
	MarkdownReport string `json:"markdown_report,omitempty"`
	Summary        string `json:"summary,omitempty"`
	Diagnostics    string `json:"diagnostics,omitempty"`
	HarnessLog     string `json:"harness_log,omitempty"`
}
