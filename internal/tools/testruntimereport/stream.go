package main

import (
	"bufio"
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

const defaultHeartbeatInterval = 10 * time.Second

type streamStatus struct {
	Label             string       `json:"label"`
	State             string       `json:"state"`
	StartedAtUTC      string       `json:"started_at_utc"`
	UpdatedAtUTC      string       `json:"updated_at_utc"`
	LastEventAtUTC    string       `json:"last_event_at_utc,omitempty"`
	ElapsedSeconds    float64      `json:"elapsed_seconds"`
	CurrentPackage    string       `json:"current_package,omitempty"`
	CurrentTest       string       `json:"current_test,omitempty"`
	PackagesCompleted int          `json:"packages_completed"`
	PackagesRunning   int          `json:"packages_running"`
	ActivePackages    []string     `json:"active_packages,omitempty"`
	ActiveTests       []string     `json:"active_tests,omitempty"`
	SlowPackages      []testTiming `json:"slow_packages,omitempty"`
	SlowTests         []testTiming `json:"slow_tests,omitempty"`
	Artifacts         []string     `json:"artifacts,omitempty"`
	Message           string       `json:"message,omitempty"`
}

type streamConfig struct {
	label             string
	statusJSON        string
	rawJSONL          string
	heartbeatInterval time.Duration
	artifacts         []string
}

type streamTracker struct {
	label          string
	startedAt      time.Time
	updatedAt      time.Time
	lastEventAt    time.Time
	currentPackage string
	currentTest    string
	packageStatus  map[string]string
	packageElapsed map[string]float64
	testElapsed    map[string]float64
	activePackages map[string]time.Time
	activeTests    map[string]time.Time
}

func runStream(args []string, stdin io.Reader, stdout, stderr io.Writer) error {
	var cfg streamConfig
	var heartbeat string
	var artifactsCSV string

	flags := flag.NewFlagSet("testruntimereport stream", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&cfg.label, "label", "", "Status label for this streamed test run")
	flags.StringVar(&cfg.statusJSON, "status-json", "", "Write live status JSON to this file path")
	flags.StringVar(&cfg.rawJSONL, "raw-jsonl", "", "Optional path for raw go test JSONL output")
	flags.StringVar(&heartbeat, "heartbeat-interval", defaultHeartbeatInterval.String(), "Heartbeat interval for console/status updates")
	flags.StringVar(&artifactsCSV, "artifacts", "", "Comma-separated artifact paths to include in the final status JSON")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return withExitCode(err, 2)
	}

	if strings.TrimSpace(cfg.label) == "" {
		err := errors.New("label is required for stream mode")
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return withExitCode(err, 1)
	}
	if strings.TrimSpace(cfg.statusJSON) == "" {
		err := errors.New("status-json is required for stream mode")
		fmt.Fprintf(stderr, "ERROR: %v\n", err)
		return withExitCode(err, 1)
	}

	interval, err := time.ParseDuration(heartbeat)
	if err != nil {
		fmt.Fprintf(stderr, "ERROR: parse heartbeat interval: %v\n", err)
		return withExitCode(err, 1)
	}
	if interval <= 0 {
		fmt.Fprintf(stderr, "ERROR: heartbeat interval must be positive\n")
		return withExitCode(errors.New("heartbeat interval must be positive"), 1)
	}
	cfg.heartbeatInterval = interval
	cfg.artifacts = splitArtifacts(artifactsCSV)

	tracker := newStreamTracker(cfg.label, time.Now().UTC())
	if err := writeJSONAtomic(cfg.statusJSON, tracker.status("running", cfg.artifacts, "")); err != nil {
		fmt.Fprintf(stderr, "ERROR: write initial status: %v\n", err)
		return withExitCode(err, 1)
	}

	var rawWriter io.WriteCloser
	if strings.TrimSpace(cfg.rawJSONL) != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.rawJSONL), 0o755); err != nil {
			fmt.Fprintf(stderr, "ERROR: create raw json dir: %v\n", err)
			return withExitCode(err, 1)
		}
		file, err := os.Create(cfg.rawJSONL)
		if err != nil {
			fmt.Fprintf(stderr, "ERROR: create raw json file: %v\n", err)
			return withExitCode(err, 1)
		}
		rawWriter = file
		defer rawWriter.Close()
	}

	fmt.Fprintf(stdout, "[%s] started\n", cfg.label)

	ticker := time.NewTicker(cfg.heartbeatInterval)
	defer ticker.Stop()

	lines := make(chan string)
	readErrs := make(chan error, 1)
	go scanLines(stdin, lines, readErrs)

	for {
		select {
		case line, ok := <-lines:
			if !ok {
				err := <-readErrs
				if err != nil {
					status := tracker.status("failed", cfg.artifacts, err.Error())
					if writeErr := writeJSONAtomic(cfg.statusJSON, status); writeErr != nil {
						fmt.Fprintf(stderr, "ERROR: write failed status: %v\n", writeErr)
					}
					return withExitCode(err, 1)
				}

				finalState := tracker.finalState()
				status := tracker.status(finalState, cfg.artifacts, "")
				if err := writeJSONAtomic(cfg.statusJSON, status); err != nil {
					fmt.Fprintf(stderr, "ERROR: write final status: %v\n", err)
					return withExitCode(err, 1)
				}
				printStreamFinalSummary(stdout, status)
				return nil
			}

			if rawWriter != nil {
				if _, err := io.WriteString(rawWriter, line+"\n"); err != nil {
					fmt.Fprintf(stderr, "ERROR: write raw json line: %v\n", err)
					return withExitCode(err, 1)
				}
			}

			if err := tracker.consumeLine(line); err != nil {
				fmt.Fprintf(stderr, "ERROR: parse stream line: %v\n", err)
				return withExitCode(err, 1)
			}
			if err := writeJSONAtomic(cfg.statusJSON, tracker.status("running", cfg.artifacts, "")); err != nil {
				fmt.Fprintf(stderr, "ERROR: update status: %v\n", err)
				return withExitCode(err, 1)
			}
		case <-ticker.C:
			status := tracker.status("running", cfg.artifacts, "")
			if err := writeJSONAtomic(cfg.statusJSON, status); err != nil {
				fmt.Fprintf(stderr, "ERROR: write heartbeat status: %v\n", err)
				return withExitCode(err, 1)
			}
			printHeartbeat(stdout, status)
		}
	}
}

func scanLines(reader io.Reader, lines chan<- string, errs chan<- error) {
	defer close(lines)

	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), 2*1024*1024)
	for scanner.Scan() {
		lines <- scanner.Text()
	}
	errs <- scanner.Err()
}

func newStreamTracker(label string, startedAt time.Time) *streamTracker {
	return &streamTracker{
		label:          label,
		startedAt:      startedAt,
		updatedAt:      startedAt,
		packageStatus:  make(map[string]string),
		packageElapsed: make(map[string]float64),
		testElapsed:    make(map[string]float64),
		activePackages: make(map[string]time.Time),
		activeTests:    make(map[string]time.Time),
	}
}

func (t *streamTracker) consumeLine(line string) error {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return nil
	}
	var event goTestEvent
	if err := json.Unmarshal([]byte(trimmed), &event); err != nil {
		return nil
	}
	t.consumeEvent(event, time.Now().UTC())
	return nil
}

func (t *streamTracker) consumeEvent(event goTestEvent, now time.Time) {
	t.updatedAt = now
	if strings.TrimSpace(event.Package) != "" {
		t.currentPackage = event.Package
	}

	switch event.Action {
	case "start":
		if event.Package != "" {
			t.activePackages[event.Package] = now
		}
	case "run":
		if event.Test != "" {
			key := testKey(event.Package, event.Test)
			t.activeTests[key] = now
			t.currentTest = event.Test
			if event.Package != "" {
				t.activePackages[event.Package] = now
			}
		}
	case "pass", "fail", "skip":
		t.lastEventAt = now
		if event.Test == "" {
			if event.Package != "" {
				t.packageStatus[event.Package] = event.Action
				if event.Elapsed > 0 {
					t.packageElapsed[event.Package] = event.Elapsed
				}
				delete(t.activePackages, event.Package)
			}
			t.currentTest = ""
			return
		}

		key := testKey(event.Package, event.Test)
		delete(t.activeTests, key)
		if event.Elapsed > 0 {
			t.testElapsed[key] = event.Elapsed
		}
		if event.Action == "fail" {
			t.currentTest = event.Test
		}
	}
}

func (t *streamTracker) finalState() string {
	failed := false
	completed := false
	for _, status := range t.packageStatus {
		if status == "fail" {
			failed = true
		}
		if status == "pass" || status == "skip" {
			completed = true
		}
	}
	if failed {
		return "failed"
	}
	if completed {
		return "passed"
	}
	return "completed"
}

func (t *streamTracker) status(state string, artifacts []string, message string) streamStatus {
	activePackages := sortedKeys(t.activePackages)
	activeTests := sortedKeys(t.activeTests)

	currentTest := t.currentTest
	if currentTest == "" && len(activeTests) > 0 {
		currentTest = activeTests[0]
	}

	currentPackage := t.currentPackage
	if currentPackage == "" && len(activePackages) > 0 {
		currentPackage = activePackages[0]
	}

	status := streamStatus{
		Label:             t.label,
		State:             state,
		StartedAtUTC:      formatUTCTime(t.startedAt),
		UpdatedAtUTC:      formatUTCTime(t.updatedAt),
		ElapsedSeconds:    t.updatedAt.Sub(t.startedAt).Seconds(),
		CurrentPackage:    currentPackage,
		CurrentTest:       currentTest,
		PackagesCompleted: len(t.packageStatus),
		PackagesRunning:   len(activePackages),
		ActivePackages:    activePackages,
		ActiveTests:       activeTests,
		SlowPackages:      topTestsByElapsed(t.packageElapsed, 5),
		SlowTests:         topTestsByElapsed(t.testElapsed, 5),
		Artifacts:         artifacts,
		Message:           message,
	}
	if !t.lastEventAt.IsZero() {
		status.LastEventAtUTC = formatUTCTime(t.lastEventAt)
	}
	return status
}

func sortedKeys(values map[string]time.Time) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func printHeartbeat(stdout io.Writer, status streamStatus) {
	active := "-"
	if len(status.ActiveTests) > 0 {
		active = status.ActiveTests[0]
	} else if len(status.ActivePackages) > 0 {
		active = status.ActivePackages[0]
	}
	fmt.Fprintf(
		stdout,
		"[%s] heartbeat elapsed=%.1fs completed=%d running=%d active=%s\n",
		status.Label,
		status.ElapsedSeconds,
		status.PackagesCompleted,
		status.PackagesRunning,
		active,
	)
}

func printStreamFinalSummary(stdout io.Writer, status streamStatus) {
	fmt.Fprintf(
		stdout,
		"[%s] finished state=%s elapsed=%.1fs completed=%d\n",
		status.Label,
		status.State,
		status.ElapsedSeconds,
		status.PackagesCompleted,
	)
	limit := len(status.SlowTests)
	if limit > 5 {
		limit = 5
	}
	for i := 0; i < limit; i++ {
		fmt.Fprintf(stdout, "  %d. %s %.3fs\n", i+1, status.SlowTests[i].Name, status.SlowTests[i].ElapsedSeconds)
	}
}

func splitArtifacts(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	artifacts := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		artifacts = append(artifacts, part)
	}
	return artifacts
}

func testKey(pkg, test string) string {
	if strings.TrimSpace(pkg) == "" {
		return test
	}
	if strings.TrimSpace(test) == "" {
		return pkg
	}
	return pkg + "::" + test
}
