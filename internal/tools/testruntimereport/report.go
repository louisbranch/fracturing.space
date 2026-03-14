package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func runReport(args []string, stdout, stderr io.Writer) error {
	var inputDir string
	var outJSON string
	var outCSV string
	var budgetPath string
	var enforceBudget bool

	flags := flag.NewFlagSet("testruntimereport", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&inputDir, "input-dir", "", "Directory containing go test -json output files (*.jsonl)")
	flags.StringVar(&outJSON, "out-json", "", "Write report JSON to this file path")
	flags.StringVar(&outCSV, "out-csv", "", "Write report CSV to this file path")
	flags.StringVar(&budgetPath, "budget-file", "", "Optional runtime budget JSON file")
	flags.BoolVar(&enforceBudget, "enforce-budget", false, "Fail when a budget regression is detected")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return withExitCode(err, 2)
	}

	if strings.TrimSpace(inputDir) == "" {
		err := errors.New("input-dir is required")
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return withExitCode(err, 1)
	}

	pattern := filepath.Join(inputDir, "*.jsonl")
	files, err := filepath.Glob(pattern)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: glob %s: %v\n", pattern, err)
		return withExitCode(err, 1)
	}
	if len(files) == 0 {
		err := fmt.Errorf("no jsonl files found in %s", inputDir)
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return withExitCode(err, 1)
	}
	sort.Strings(files)

	runs := make([]runSummary, 0, len(files))
	for _, path := range files {
		label := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
		summary, err := parseRunFile(path, label)
		if err != nil {
			fmt.Fprintf(stderr, "ERROR: parse %s: %v\n", path, err)
			return withExitCode(err, 1)
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
			fmt.Fprintf(stderr, "ERROR: write json report: %v\n", err)
			return withExitCode(err, 1)
		}
	}
	if outCSV != "" {
		if err := writeCSV(outCSV, runs); err != nil {
			fmt.Fprintf(stderr, "ERROR: write csv report: %v\n", err)
			return withExitCode(err, 1)
		}
	}

	printConsoleSummary(stdout, output)

	if budgetPath != "" {
		violations, err := checkBudgets(budgetPath, runs)
		if err != nil {
			fmt.Fprintf(stderr, "ERROR: check budgets: %v\n", err)
			return withExitCode(err, 1)
		}
		for _, violation := range violations {
			fmt.Fprintf(stderr, "RUNTIME_BUDGET_WARNING: %s\n", violation)
		}
		if enforceBudget && len(violations) > 0 {
			return withExitCode(errors.New("runtime budget regression detected"), 1)
		}
	}
	return nil
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

	return runSummary{
		Label:          label,
		Package:        packageName,
		Status:         suiteStatus,
		ElapsedSeconds: suiteElapsed,
		Tests:          topTestsByElapsed(testElapsed, len(testElapsed)),
	}, nil
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
				formatFloatSeconds(run.ElapsedSeconds),
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
				formatFloatSeconds(run.ElapsedSeconds),
				test.Name,
				formatFloatSeconds(test.ElapsedSeconds),
			}); err != nil {
				return err
			}
		}
	}
	return nil
}

func printConsoleSummary(stdout io.Writer, output report) {
	fmt.Fprintf(stdout, "Runtime report generated at %s\n", output.GeneratedAtUTC)
	for _, run := range output.Runs {
		fmt.Fprintf(stdout, "- %s: status=%s elapsed=%.3fs package=%s\n", run.Label, run.Status, run.ElapsedSeconds, run.Package)
		limit := len(run.Tests)
		if limit > 5 {
			limit = 5
		}
		for i := 0; i < limit; i++ {
			fmt.Fprintf(stdout, "  %d. %s %.3fs\n", i+1, run.Tests[i].Name, run.Tests[i].ElapsedSeconds)
		}
	}
	fmt.Fprintf(stdout, "Total elapsed seconds across runs: %.3f\n", output.TotalElapsed)
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
