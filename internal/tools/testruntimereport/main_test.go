package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunSummaryFromInputDir(t *testing.T) {
	tempDir := t.TempDir()
	writeFile(
		t,
		filepath.Join(tempDir, "unit.jsonl"),
		`{"Action":"pass","Package":"example/pkg","Test":"TestSlow","Elapsed":0.400}`+"\n"+
			`{"Action":"pass","Package":"example/pkg","Elapsed":1.250}`+"\n",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{"-input-dir", tempDir}, &stdout, &stderr)
	if err != nil {
		t.Fatalf("run returned error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "- unit: status=pass elapsed=1.250s package=example/pkg") {
		t.Fatalf("expected run summary output, got %q", stdout.String())
	}
}

func TestRunBudgetEnforcement(t *testing.T) {
	tempDir := t.TempDir()
	writeFile(
		t,
		filepath.Join(tempDir, "unit.jsonl"),
		`{"Action":"pass","Package":"example/pkg","Elapsed":1.250}`+"\n",
	)
	budgetPath := filepath.Join(tempDir, "budget.json")
	writeFile(
		t,
		budgetPath,
		`{"runs":{"unit":{"baseline_seconds":1.0,"allowed_regression_pct":0}}}`+"\n",
	)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := run([]string{
		"-input-dir", tempDir,
		"-budget-file", budgetPath,
		"-enforce-budget",
	}, &stdout, &stderr)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if code := exitCode(err); code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "RUNTIME_BUDGET_WARNING: unit elapsed 1.250s exceeds threshold 1.000s") {
		t.Fatalf("expected runtime budget warning, got %q", stderr.String())
	}
}

func TestRunStreamWritesFinalStatusAndRawJSONL(t *testing.T) {
	tempDir := t.TempDir()
	statusPath := filepath.Join(tempDir, "status.json")
	rawPath := filepath.Join(tempDir, "raw.jsonl")
	input := strings.Join([]string{
		`{"Action":"start","Package":"example/pkg"}`,
		`{"Action":"run","Package":"example/pkg","Test":"TestSlow"}`,
		`{"Action":"pass","Package":"example/pkg","Test":"TestSlow","Elapsed":0.400}`,
		`{"Action":"pass","Package":"example/pkg","Elapsed":1.250}`,
	}, "\n") + "\n"

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := runStream(
		[]string{
			"-label", "unit",
			"-status-json", statusPath,
			"-raw-jsonl", rawPath,
			"-heartbeat-interval", "1s",
		},
		strings.NewReader(input),
		&stdout,
		&stderr,
	)
	if err != nil {
		t.Fatalf("stream run returned error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "[unit] started") {
		t.Fatalf("expected start output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "[unit] finished state=passed") {
		t.Fatalf("expected final summary output, got %q", stdout.String())
	}

	status := readStreamStatus(t, statusPath)
	if status.State != "passed" {
		t.Fatalf("status state = %q, want passed", status.State)
	}
	if status.PackagesCompleted != 1 {
		t.Fatalf("status packages_completed = %d, want 1", status.PackagesCompleted)
	}
	if got := strings.TrimSpace(readFile(t, rawPath)); got != strings.TrimSpace(input) {
		t.Fatalf("raw jsonl = %q, want %q", got, strings.TrimSpace(input))
	}
}

func TestRunStreamPrintsHeartbeatAndKeepsStatusFresh(t *testing.T) {
	tempDir := t.TempDir()
	statusPath := filepath.Join(tempDir, "status.json")

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	defer reader.Close()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- runStream(
			[]string{
				"-label", "slow",
				"-status-json", statusPath,
				"-heartbeat-interval", "10ms",
			},
			reader,
			&stdout,
			&stderr,
		)
	}()

	writePipeLine(t, writer, `{"Action":"start","Package":"example/pkg"}`)
	writePipeLine(t, writer, `{"Action":"run","Package":"example/pkg","Test":"TestSlow"}`)
	time.Sleep(40 * time.Millisecond)
	writePipeLine(t, writer, `{"Action":"pass","Package":"example/pkg","Test":"TestSlow","Elapsed":0.400}`)
	writePipeLine(t, writer, `{"Action":"pass","Package":"example/pkg","Elapsed":1.250}`)
	if err := writer.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}

	if err := <-done; err != nil {
		t.Fatalf("stream run returned error: %v", err)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "heartbeat elapsed=") {
		t.Fatalf("expected heartbeat output, got %q", stdout.String())
	}

	status := readStreamStatus(t, statusPath)
	if status.State != "passed" {
		t.Fatalf("status state = %q, want passed", status.State)
	}
	if status.LastEventAtUTC == "" {
		t.Fatal("expected last_event_at_utc to be populated")
	}
	if status.ElapsedSeconds <= 0 {
		t.Fatalf("expected positive elapsed seconds, got %.3f", status.ElapsedSeconds)
	}
}

func readStreamStatus(t *testing.T, path string) streamStatus {
	t.Helper()
	data := readFile(t, path)
	var status streamStatus
	if err := json.Unmarshal([]byte(data), &status); err != nil {
		t.Fatalf("unmarshal status %s: %v", path, err)
	}
	return status
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %s: %v", path, err)
	}
	return string(data)
}

func writeFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func writePipeLine(t *testing.T, writer *os.File, line string) {
	t.Helper()
	if _, err := writer.WriteString(line + "\n"); err != nil {
		t.Fatalf("write pipe line: %v", err)
	}
}
