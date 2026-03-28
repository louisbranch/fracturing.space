// Package aieval parses CLI flags and runs one live AI GM evaluation lane for Promptfoo.
package aieval

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	evalsupport "github.com/louisbranch/fracturing.space/internal/test/aieval"
)

const (
	integrationEvalOutputPathEnv = "INTEGRATION_AI_EVAL_OUTPUT_PATH"
	integrationEvalCaseIDEnv     = "INTEGRATION_AI_EVAL_CASE_ID"
	integrationEvalRunIDEnv      = "INTEGRATION_AI_EVAL_RUN_ID"
	integrationPromptProfileEnv  = "INTEGRATION_AI_PROMPT_PROFILE"
)

var fileLinePrefixPattern = regexp.MustCompile(`^\s*\S+:\d+:\s*`)

// Config holds AI GM evaluation CLI configuration.
type Config struct {
	Scenario         string `env:"INTEGRATION_AI_EVAL_SCENARIO"`
	Model            string `env:"INTEGRATION_AI_MODEL"               envDefault:"gpt-5.4"`
	ReasoningEffort  string `env:"INTEGRATION_AI_REASONING_EFFORT"    envDefault:"medium"`
	PromptProfile    string `env:"INTEGRATION_AI_PROMPT_PROFILE"      envDefault:"baseline"`
	InstructionsRoot string `env:"FRACTURING_SPACE_AI_INSTRUCTIONS_ROOT"`
	ResponsesURL     string `env:"INTEGRATION_OPENAI_RESPONSES_URL"`
	CaseID           string `env:"INTEGRATION_AI_EVAL_CASE_ID"`
	RunID            string `env:"INTEGRATION_AI_EVAL_RUN_ID"`
	JSONPath         string `env:"INTEGRATION_AI_EVAL_JSON_PATH"`
}

// ParseConfig parses environment and flags into Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := entrypoint.ParseConfig(&cfg); err != nil {
		return Config{}, err
	}

	fs.StringVar(&cfg.Scenario, "scenario", cfg.Scenario, "AI GM scenario id")
	fs.StringVar(&cfg.Model, "model", cfg.Model, "model name for the live eval run")
	fs.StringVar(&cfg.ReasoningEffort, "reasoning-effort", cfg.ReasoningEffort, "reasoning effort for the live eval run")
	fs.StringVar(&cfg.PromptProfile, "prompt-profile", cfg.PromptProfile, "prompt profile label to include in the eval output")
	fs.StringVar(&cfg.InstructionsRoot, "instructions-root", cfg.InstructionsRoot, "override instruction root for the live eval run")
	fs.StringVar(&cfg.ResponsesURL, "responses-url", cfg.ResponsesURL, "alternate OpenAI-compatible Responses URL")
	fs.StringVar(&cfg.CaseID, "case-id", cfg.CaseID, "stable eval case identifier used for isolated artifacts")
	fs.StringVar(&cfg.RunID, "run-id", cfg.RunID, "optional parent eval run identifier")
	fs.StringVar(&cfg.JSONPath, "json", cfg.JSONPath, "optional path to also write the final JSON output")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}

	if strings.TrimSpace(cfg.Scenario) == "" {
		return Config{}, errors.New("scenario is required")
	}
	if _, ok := evalsupport.ScenarioByID(cfg.Scenario); !ok {
		return Config{}, fmt.Errorf("unknown scenario %q", cfg.Scenario)
	}
	return cfg, nil
}

// Run executes one live eval scenario through go test and writes the resulting JSON.
func Run(ctx context.Context, cfg Config, out io.Writer, errOut io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	scenario, ok := evalsupport.ScenarioByID(cfg.Scenario)
	if !ok {
		return fmt.Errorf("unknown scenario %q", cfg.Scenario)
	}

	repoRoot, err := findRepoRoot()
	if err != nil {
		return err
	}
	caseID := resolvedCaseID(cfg, scenario)
	outputPath := evalOutputPath(repoRoot, caseID)

	cmdArgs := []string{
		"test",
		"-tags=integration liveai",
		"./internal/test/integration",
		"-run",
		"^" + scenario.LiveTestName + "$",
		"-count=1",
	}
	cmd := exec.CommandContext(ctx, "go", cmdArgs...)
	cmd.Dir = repoRoot
	cmd.Env = buildCommandEnv(cfg, caseID, outputPath)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		combined := combineCommandOutput(stdout.Bytes(), stderr.Bytes())
		logPath := writeHarnessLog(repoRoot, scenario, cfg, combined)
		result, readErr := readEvalOutput(outputPath)
		if readErr == nil {
			failureKind, failureSummary, failureReason, metricStatus := classifyHarnessFailure(result, combined)
			enrichFailedResult(&result, cfg, caseID, logPath, failureKind, failureSummary, failureReason, metricStatus)
			return writeEvalOutput(result, out, cfg.JSONPath)
		}
		if logPath != "" {
			_, _ = fmt.Fprintf(errOut, "harness log written to %s\n", logPath)
		}
		return fmt.Errorf("%s", compactRunFailureMessage(scenario, cfg, combined, logPath, err))
	}

	result, err := readEvalOutput(outputPath)
	if err != nil {
		return err
	}
	applyOutputDefaults(&result, cfg, caseID)
	if strings.TrimSpace(result.RunStatus) == "" {
		result.RunStatus = evalsupport.RunStatusPassed
	}
	if strings.TrimSpace(result.MetricStatus) == "" {
		result.MetricStatus = evalsupport.MetricStatusPass
	}
	return writeEvalOutput(result, out, cfg.JSONPath)
}

// buildCommandEnv injects only the eval-specific overrides while preserving the caller environment.
func buildCommandEnv(cfg Config, caseID string, outputPath string) []string {
	env := os.Environ()
	env = appendOrReplaceEnv(env, integrationEvalOutputPathEnv, outputPath)
	env = appendOrReplaceEnv(env, integrationEvalCaseIDEnv, strings.TrimSpace(caseID))
	env = appendOrReplaceEnv(env, integrationEvalRunIDEnv, strings.TrimSpace(cfg.RunID))
	env = appendOrReplaceEnv(env, integrationPromptProfileEnv, strings.TrimSpace(cfg.PromptProfile))
	env = appendOrReplaceEnv(env, "INTEGRATION_AI_MODEL", strings.TrimSpace(cfg.Model))
	env = appendOrReplaceEnv(env, "INTEGRATION_AI_REASONING_EFFORT", strings.TrimSpace(cfg.ReasoningEffort))
	if root := strings.TrimSpace(cfg.InstructionsRoot); root != "" {
		env = appendOrReplaceEnv(env, "FRACTURING_SPACE_AI_INSTRUCTIONS_ROOT", root)
	} else {
		env = removeEnv(env, "FRACTURING_SPACE_AI_INSTRUCTIONS_ROOT")
	}
	if url := strings.TrimSpace(cfg.ResponsesURL); url != "" {
		env = appendOrReplaceEnv(env, "INTEGRATION_OPENAI_RESPONSES_URL", url)
	}
	return env
}

// appendOrReplaceEnv keeps child-process environment overrides deterministic.
func appendOrReplaceEnv(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

// removeEnv clears optional overrides so the child process falls back to its defaults.
func removeEnv(env []string, key string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env))
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			continue
		}
		out = append(out, entry)
	}
	return out
}

func readEvalOutput(path string) (evalsupport.Output, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return evalsupport.Output{}, fmt.Errorf("read eval output: %w", err)
	}
	var result evalsupport.Output
	if err := json.Unmarshal(data, &result); err != nil {
		return evalsupport.Output{}, fmt.Errorf("parse eval output: %w", err)
	}
	return result, nil
}

func writeEvalOutput(result evalsupport.Output, out io.Writer, jsonPath string) error {
	encoded, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal eval output: %w", err)
	}
	encoded = append(encoded, '\n')
	if _, err := out.Write(encoded); err != nil {
		return fmt.Errorf("write stdout json: %w", err)
	}
	if path := strings.TrimSpace(jsonPath); path != "" {
		if err := os.WriteFile(path, encoded, 0o644); err != nil {
			return fmt.Errorf("write json file: %w", err)
		}
	}
	return nil
}

func enrichFailedResult(result *evalsupport.Output, cfg Config, caseID string, logPath string, failureKind string, failureSummary string, failureReason string, metricStatus string) {
	if result == nil {
		return
	}
	applyOutputDefaults(result, cfg, caseID)
	result.RunStatus = evalsupport.RunStatusFailed
	if strings.TrimSpace(metricStatus) == "" {
		metricStatus = evalsupport.MetricStatusInvalid
	}
	result.MetricStatus = strings.TrimSpace(metricStatus)
	result.FailureKind = strings.TrimSpace(failureKind)
	result.FailureSummary = strings.TrimSpace(failureSummary)
	result.FailureReason = strings.TrimSpace(failureReason)
	if logPath != "" {
		result.Artifacts.HarnessLog = logPath
	}
}

func combineCommandOutput(stdout, stderr []byte) string {
	parts := make([]string, 0, 2)
	if len(stderr) > 0 {
		parts = append(parts, strings.TrimSpace(string(stderr)))
	}
	if len(stdout) > 0 {
		parts = append(parts, strings.TrimSpace(string(stdout)))
	}
	return strings.TrimSpace(strings.Join(parts, "\n\n"))
}

func writeHarnessLog(repoRoot string, scenario evalsupport.Scenario, cfg Config, combined string) string {
	if strings.TrimSpace(combined) == "" {
		return ""
	}
	dir := filepath.Join(repoRoot, ".tmp", "promptfoo", "logs")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return ""
	}
	filename := fmt.Sprintf(
		"%s-%s-%s-%s.log",
		sanitizeFileToken(scenario.ID),
		sanitizeFileToken(cfg.Model),
		sanitizeFileToken(cfg.PromptProfile),
		time.Now().UTC().Format("20060102T150405Z"),
	)
	path := filepath.Join(dir, filename)
	if err := os.WriteFile(path, []byte(strings.TrimSpace(combined)+"\n"), 0o644); err != nil {
		return ""
	}
	return path
}

func classifyHarnessFailure(result evalsupport.Output, combined string) (string, string, string, string) {
	if kind := strings.TrimSpace(result.FailureKind); kind != "" {
		summary := strings.TrimSpace(result.FailureSummary)
		if summary == "" {
			summary = strings.TrimSpace(result.FailureReason)
		}
		if summary == "" {
			summary = kind
		}
		reason := strings.TrimSpace(result.FailureReason)
		if reason == "" {
			reason = summary
		}
		metricStatus := strings.TrimSpace(result.MetricStatus)
		if metricStatus == "" {
			metricStatus = defaultMetricStatusForFailure(kind)
		}
		return kind, summary, reason, metricStatus
	}

	line := firstActionableFailureLine(combined)
	lower := strings.ToLower(combined)

	switch {
	case strings.Contains(lower, `missing required tool "daggerheart_action_roll_resolve"`):
		reason := `missing required tool "daggerheart_action_roll_resolve"`
		return "missing_authoritative_roll", reason, reason, evalsupport.MetricStatusFail
	case strings.Contains(lower, "system_reference_search calls =") || strings.Contains(lower, "system_reference_read calls ="):
		reason := fallbackFailureReason(line, "reference lookup budget exceeded")
		return "over_research", reason, reason, evalsupport.MetricStatusFail
	case strings.Contains(lower, "fixture should not call"):
		reason := fallbackFailureReason(line, "forbidden tool path used")
		return "forbidden_tool_path", reason, reason, evalsupport.MetricStatusFail
	case strings.Contains(lower, "current interaction is missing beat type") || strings.Contains(lower, "current interaction is missing all beat types"):
		if !containsString(result.ToolNames, "daggerheart_action_roll_resolve") {
			reason := fallbackFailureReason(line, "missing authoritative roll before committed beats")
			return "missing_authoritative_roll", reason, reason, evalsupport.MetricStatusFail
		}
		reason := fallbackFailureReason(line, "interaction beat contract failed")
		return "turn_control_error", reason, reason, evalsupport.MetricStatusFail
	case strings.Contains(lower, "live recorder:"):
		reason := fallbackFailureReason(line, "live recorder failed")
		return "recorder_error", reason, reason, evalsupport.MetricStatusInvalid
	default:
		reason := fallbackFailureReason(line, "live eval failed")
		return "harness_error", reason, reason, evalsupport.MetricStatusInvalid
	}
}

func compactRunFailureMessage(scenario evalsupport.Scenario, cfg Config, combined string, logPath string, runErr error) string {
	line := fallbackFailureReason(firstActionableFailureLine(combined), fmt.Sprintf("run %s: %v", scenario.LiveTestName, runErr))
	header := fmt.Sprintf(
		"aieval failed for scenario=%s model=%s prompt_profile=%s",
		scenario.ID,
		strings.TrimSpace(cfg.Model),
		strings.TrimSpace(cfg.PromptProfile),
	)
	if logPath == "" {
		return header + ": " + line
	}
	return fmt.Sprintf("%s: %s (log: %s)", header, line, logPath)
}

func firstActionableFailureLine(combined string) string {
	for _, raw := range strings.Split(combined, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if isNoiseLine(line) {
			continue
		}
		line = fileLinePrefixPattern.ReplaceAllString(line, "")
		if line == "" {
			continue
		}
		return line
	}
	return ""
}

func isNoiseLine(line string) bool {
	switch {
	case strings.HasPrefix(line, "202"):
		return true
	case strings.HasPrefix(line, "--- FAIL:"):
		return true
	case strings.HasPrefix(line, "FAIL"):
		return true
	case strings.HasPrefix(line, "step="):
		return true
	case strings.Contains(line, "written to "):
		return true
	case strings.Contains(line, "server listening"):
		return true
	case strings.Contains(line, "managed conn"):
		return true
	case strings.Contains(line, "status reporter:"):
		return true
	case strings.Contains(line, "projection apply mode resolved"):
		return true
	case strings.Contains(line, "requests:"):
		return true
	default:
		return false
	}
}

func fallbackFailureReason(line string, fallback string) string {
	if strings.TrimSpace(line) != "" {
		return strings.TrimSpace(line)
	}
	return fallback
}

func applyOutputDefaults(result *evalsupport.Output, cfg Config, caseID string) {
	if result == nil {
		return
	}
	if strings.TrimSpace(result.CaseID) == "" {
		result.CaseID = strings.TrimSpace(caseID)
	}
	if strings.TrimSpace(result.PromptProfile) == "" {
		result.PromptProfile = strings.TrimSpace(cfg.PromptProfile)
	}
	if strings.TrimSpace(result.PromptContext.Profile) == "" {
		result.PromptContext = evalsupport.BuildPromptContext(strings.TrimSpace(cfg.PromptProfile), strings.TrimSpace(cfg.InstructionsRoot))
	}
}

func defaultMetricStatusForFailure(kind string) string {
	switch strings.TrimSpace(kind) {
	case "missing_authoritative_roll", "resource_accounting", "over_research", "narrator_authority", "phase_reopen", "forbidden_tool_path", "turn_control_error":
		return evalsupport.MetricStatusFail
	default:
		return evalsupport.MetricStatusInvalid
	}
}

func resolvedCaseID(cfg Config, scenario evalsupport.Scenario) string {
	if caseID := strings.TrimSpace(cfg.CaseID); caseID != "" {
		return caseID
	}
	parts := []string{
		sanitizeFileToken(scenario.ID),
		sanitizeFileToken(cfg.Model),
		sanitizeFileToken(cfg.PromptProfile),
	}
	if runID := strings.TrimSpace(cfg.RunID); runID != "" {
		parts = append(parts, sanitizeFileToken(runID))
	}
	parts = append(parts, time.Now().UTC().Format("20060102T150405.000000000Z"))
	return strings.Join(parts, "__")
}

func evalOutputPath(repoRoot string, caseID string) string {
	dir := filepath.Join(repoRoot, ".tmp", "aieval")
	_ = os.MkdirAll(dir, 0o755)
	return filepath.Join(dir, sanitizeFileToken(caseID)+".json")
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if strings.TrimSpace(value) == target {
			return true
		}
	}
	return false
}

// findRepoRoot locates the repository root so the CLI can be launched from nested directories.
func findRepoRoot() (string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getwd: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd, nil
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return "", errors.New("could not find repo root from current working directory")
}

// sanitizeFileToken keeps temp filenames readable and shell-safe.
func sanitizeFileToken(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, "/", "-")
	value = strings.ReplaceAll(value, " ", "-")
	if value == "" {
		return "eval"
	}
	return value
}
