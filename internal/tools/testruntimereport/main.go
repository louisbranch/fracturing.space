package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
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

func main() {
	var inputDir string
	var outJSON string
	var outCSV string
	var budgetPath string
	var enforceBudget bool

	flag.StringVar(&inputDir, "input-dir", "", "Directory containing go test -json output files (*.jsonl)")
	flag.StringVar(&outJSON, "out-json", "", "Write report JSON to this file path")
	flag.StringVar(&outCSV, "out-csv", "", "Write report CSV to this file path")
	flag.StringVar(&budgetPath, "budget-file", "", "Optional runtime budget JSON file")
	flag.BoolVar(&enforceBudget, "enforce-budget", false, "Fail when a budget regression is detected")
	flag.Parse()

	if strings.TrimSpace(inputDir) == "" {
		fatalf("input-dir is required")
	}

	pattern := filepath.Join(inputDir, "*.jsonl")
	files, err := filepath.Glob(pattern)
	if err != nil {
		fatalf("glob %s: %v", pattern, err)
	}
	if len(files) == 0 {
		fatalf("no jsonl files found in %s", inputDir)
	}
	sort.Strings(files)

	runs := make([]runSummary, 0, len(files))
	for _, path := range files {
		label := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		summary, err := parseRunFile(path, label)
		if err != nil {
			fatalf("parse %s: %v", path, err)
		}
		runs = append(runs, summary)
	}

	totalElapsed := 0.0
	for _, run := range runs {
		totalElapsed += run.ElapsedSeconds
	}
	output := report{
		GeneratedAtUTC: time.Now().UTC().Format(time.RFC3339),
		Runs:           runs,
		TotalElapsed:   totalElapsed,
	}

	if outJSON != "" {
		if err := writeJSON(outJSON, output); err != nil {
			fatalf("write json report: %v", err)
		}
	}
	if outCSV != "" {
		if err := writeCSV(outCSV, runs); err != nil {
			fatalf("write csv report: %v", err)
		}
	}

	printConsoleSummary(output)

	if budgetPath != "" {
		violations, err := checkBudgets(budgetPath, runs)
		if err != nil {
			fatalf("check budgets: %v", err)
		}
		for _, violation := range violations {
			fmt.Fprintf(os.Stderr, "RUNTIME_BUDGET_WARNING: %s\n", violation)
		}
		if enforceBudget && len(violations) > 0 {
			os.Exit(1)
		}
	}
}

func parseRunFile(path, label string) (runSummary, error) {
	file, err := os.Open(path)
	if err != nil {
		return runSummary{}, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)

	testElapsed := make(map[string]float64)
	suiteStatus := "unknown"
	suiteElapsed := 0.0
	packageName := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var event goTestEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			// Ignore non-JSON or malformed lines to keep report generation resilient.
			continue
		}
		if strings.TrimSpace(event.Package) != "" {
			packageName = event.Package
		}

		switch event.Action {
		case "pass", "fail":
			if event.Test == "" {
				suiteStatus = event.Action
				if event.Elapsed > 0 {
					suiteElapsed = event.Elapsed
				}
				continue
			}
			if event.Elapsed > 0 {
				testElapsed[event.Test] = event.Elapsed
			}
		}
	}
	if err := scanner.Err(); err != nil {
		return runSummary{}, err
	}

	tests := make([]testTiming, 0, len(testElapsed))
	for name, elapsed := range testElapsed {
		tests = append(tests, testTiming{Name: name, ElapsedSeconds: elapsed})
	}
	sort.Slice(tests, func(i, j int) bool {
		if tests[i].ElapsedSeconds == tests[j].ElapsedSeconds {
			return tests[i].Name < tests[j].Name
		}
		return tests[i].ElapsedSeconds > tests[j].ElapsedSeconds
	})

	return runSummary{
		Label:          label,
		Package:        packageName,
		Status:         suiteStatus,
		ElapsedSeconds: suiteElapsed,
		Tests:          tests,
	}, nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}

func writeCSV(path string, runs []runSummary) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	if err := writer.Write([]string{
		"label",
		"status",
		"package",
		"suite_elapsed_seconds",
		"test_name",
		"test_elapsed_seconds",
	}); err != nil {
		return err
	}

	for _, run := range runs {
		if len(run.Tests) == 0 {
			if err := writer.Write([]string{
				run.Label,
				run.Status,
				run.Package,
				fmt.Sprintf("%.3f", run.ElapsedSeconds),
				"",
				"",
			}); err != nil {
				return err
			}
			continue
		}
		for _, test := range run.Tests {
			if err := writer.Write([]string{
				run.Label,
				run.Status,
				run.Package,
				fmt.Sprintf("%.3f", run.ElapsedSeconds),
				test.Name,
				fmt.Sprintf("%.3f", test.ElapsedSeconds),
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func printConsoleSummary(output report) {
	fmt.Printf("Runtime report generated at %s\n", output.GeneratedAtUTC)
	for _, run := range output.Runs {
		fmt.Printf("- %s: status=%s elapsed=%.3fs package=%s\n", run.Label, run.Status, run.ElapsedSeconds, run.Package)
		limit := len(run.Tests)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			fmt.Printf("  %d. %s %.3fs\n", i+1, run.Tests[i].Name, run.Tests[i].ElapsedSeconds)
		}
	}
	fmt.Printf("Total elapsed seconds across runs: %.3f\n", output.TotalElapsed)
}

func checkBudgets(path string, runs []runSummary) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var budgets budgetFile
	if err := json.Unmarshal(data, &budgets); err != nil {
		return nil, err
	}
	if len(budgets.Runs) == 0 {
		return nil, nil
	}

	violations := make([]string, 0)
	for _, run := range runs {
		budget, ok := budgets.Runs[run.Label]
		if !ok || budget.BaselineSeconds <= 0 {
			continue
		}
		allowedPct := budget.AllowedRegressionPct
		if allowedPct < 0 {
			allowedPct = 0
		}
		threshold := budget.BaselineSeconds * (1 + (allowedPct / 100.0))
		if run.ElapsedSeconds <= threshold {
			continue
		}
		violations = append(violations, fmt.Sprintf(
			"%s elapsed %.3fs exceeds threshold %.3fs (baseline %.3fs, allowed +%.1f%%)",
			run.Label,
			run.ElapsedSeconds,
			threshold,
			budget.BaselineSeconds,
			allowedPct,
		))
	}
	sort.Strings(violations)
	return violations, nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "ERROR: "+format+"\n", args...)
	os.Exit(1)
}
