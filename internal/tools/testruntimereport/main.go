// Package main summarizes go test JSON output into runtime reports and live status artifacts.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type goTestEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Elapsed float64 `json:"Elapsed"`
}

type testTiming struct {
	Name           string  `json:"name"`
	ElapsedSeconds float64 `json:"elapsed_seconds"`
}

type runSummary struct {
	Label          string       `json:"label"`
	Package        string       `json:"package,omitempty"`
	Status         string       `json:"status"`
	ElapsedSeconds float64      `json:"elapsed_seconds"`
	Tests          []testTiming `json:"tests"`
}

type report struct {
	GeneratedAtUTC string       `json:"generated_at_utc"`
	Runs           []runSummary `json:"runs"`
	TotalElapsed   float64      `json:"total_elapsed_seconds"`
}

type runBudget struct {
	BaselineSeconds      float64 `json:"baseline_seconds"`
	AllowedRegressionPct float64 `json:"allowed_regression_pct"`
}

type budgetFile struct {
	Runs map[string]runBudget `json:"runs"`
}

type codedError struct {
	code int
	err  error
}

func (e codedError) Error() string {
	return e.err.Error()
}

func (e codedError) Unwrap() error {
	return e.err
}

func withExitCode(err error, code int) error {
	if err == nil {
		return nil
	}
	return codedError{code: code, err: err}
}

func exitCode(err error) int {
	var codeErr codedError
	if errors.As(err, &codeErr) {
		return codeErr.code
	}
	return 1
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		os.Exit(exitCode(err))
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	if len(args) > 0 && strings.EqualFold(args[0], "stream") {
		return runStream(args[1:], os.Stdin, stdout, stderr)
	}
	return runReport(args, stdout, stderr)
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func writeJSONAtomic(path string, value any) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("json output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmpPath := path + ".tmp"
	if err := writeJSON(tmpPath, value); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func formatUTCTime(ts time.Time) string {
	return ts.UTC().Format(time.RFC3339)
}

func topTestsByElapsed(elapsedByName map[string]float64, limit int) []testTiming {
	tests := make([]testTiming, 0, len(elapsedByName))
	for name, elapsed := range elapsedByName {
		tests = append(tests, testTiming{Name: name, ElapsedSeconds: elapsed})
	}
	sortTestTimings(tests)
	if len(tests) > limit {
		tests = tests[:limit]
	}
	return tests
}

func sortTestTimings(tests []testTiming) {
	sort.Slice(tests, func(i, j int) bool {
		if tests[i].ElapsedSeconds == tests[j].ElapsedSeconds {
			return tests[i].Name < tests[j].Name
		}
		return tests[i].ElapsedSeconds > tests[j].ElapsedSeconds
	})
}

func formatFloatSeconds(seconds float64) string {
	return fmt.Sprintf("%.3f", seconds)
}
