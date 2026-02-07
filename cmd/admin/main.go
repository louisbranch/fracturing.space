// Package main provides administrative maintenance utilities.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/louisbranch/fracturing.space/internal/state/event"
	"github.com/louisbranch/fracturing.space/internal/state/projection"
	"github.com/louisbranch/fracturing.space/internal/storage"
	"github.com/louisbranch/fracturing.space/internal/storage/sqlite"
	"github.com/louisbranch/fracturing.space/internal/systems/daggerheart"
)

const adminReplayPageSize = 200

type snapshotScanReport struct {
	LastSeq        uint64
	TotalEvents    int
	SnapshotEvents int
	InvalidEvents  int
}

type integrityReport struct {
	LastSeq             uint64
	CharacterMismatches int
	MissingStates       int
	GmFearMatch         bool
	GmFearSource        int
	GmFearReplay        int
}

func main() {
	var campaignID string
	var campaignIDs string
	var dbPath string
	var untilSeq uint64
	var afterSeq uint64
	var dryRun bool
	var validate bool
	var integrity bool
	var warningsCap int
	var jsonOutput bool

	flag.StringVar(&campaignID, "campaign-id", "", "campaign ID to replay snapshot events")
	flag.StringVar(&campaignIDs, "campaign-ids", "", "comma-separated campaign IDs to replay snapshot events")
	flag.StringVar(&dbPath, "db-path", defaultDBPath(), "path to sqlite database (default: DUALITY_DB_PATH or data/fracturing.space.db)")
	flag.Uint64Var(&untilSeq, "until-seq", 0, "replay up to this event sequence (0 = latest)")
	flag.Uint64Var(&afterSeq, "after-seq", 0, "start replay after this event sequence")
	flag.BoolVar(&dryRun, "dry-run", false, "scan snapshot events without applying projections")
	flag.BoolVar(&validate, "validate", false, "validate snapshot event payloads without applying projections (implies -dry-run)")
	flag.BoolVar(&integrity, "integrity", false, "replay snapshot events into a scratch store and compare against stored projections")
	flag.IntVar(&warningsCap, "warnings-cap", 25, "max warnings to print (0 = no limit)")
	flag.BoolVar(&jsonOutput, "json", false, "output JSON reports")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	store, err := openStore(dbPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := store.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: close store: %v\n", err)
		}
	}()

	if validate {
		dryRun = true
	}
	if integrity && (dryRun || validate) {
		fmt.Fprintln(os.Stderr, "Error: -integrity cannot be combined with -dry-run or -validate")
		os.Exit(1)
	}
	if integrity && afterSeq > 0 {
		fmt.Fprintln(os.Stderr, "Error: -integrity does not support -after-seq; replay must start at the beginning")
		os.Exit(1)
	}

	ids, err := resolveCampaignIDs(campaignID, campaignIDs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	if warningsCap < 0 {
		fmt.Fprintln(os.Stderr, "Error: -warnings-cap must be >= 0")
		os.Exit(1)
	}

	options := runOptions{
		AfterSeq:    afterSeq,
		UntilSeq:    untilSeq,
		DryRun:      dryRun,
		Validate:    validate,
		Integrity:   integrity,
		WarningsCap: warningsCap,
		JSONOutput:  jsonOutput,
	}

	failed := false
	for _, id := range ids {
		result := runCampaign(ctx, store, id, options)
		if options.JSONOutput {
			outputJSON(result)
		} else {
			prefix := ""
			if len(ids) > 1 {
				prefix = fmt.Sprintf("[%s] ", id)
			}
			printResult(result, prefix)
		}
		if result.ExitCode != 0 {
			failed = true
		}
	}
	if failed {
		os.Exit(1)
	}
}

type runOptions struct {
	AfterSeq    uint64
	UntilSeq    uint64
	DryRun      bool
	Validate    bool
	Integrity   bool
	WarningsCap int
	JSONOutput  bool
}

type runResult struct {
	CampaignID    string          `json:"campaign_id"`
	Mode          string          `json:"mode"`
	Report        json.RawMessage `json:"report,omitempty"`
	Warnings      []string        `json:"warnings,omitempty"`
	WarningsTotal int             `json:"warnings_total,omitempty"`
	Error         string          `json:"error,omitempty"`
	ExitCode      int             `json:"-"`
}

func runCampaign(ctx context.Context, store storage.Store, campaignID string, options runOptions) runResult {
	result := runResult{CampaignID: campaignID}
	if options.Integrity {
		result.Mode = "integrity"
		report, warnings, err := checkSnapshotIntegrity(ctx, store, campaignID, options.UntilSeq)
		result.Warnings, result.WarningsTotal = capWarnings(warnings, options.WarningsCap)
		if err != nil {
			result.Error = fmt.Sprintf("integrity check: %v", err)
			result.ExitCode = 1
			return result
		}
		payload, err := json.Marshal(report)
		if err != nil {
			result.Error = fmt.Sprintf("encode report: %v", err)
			result.ExitCode = 1
			return result
		}
		result.Report = payload
		if !report.GmFearMatch || report.CharacterMismatches > 0 {
			result.ExitCode = 1
		}
		return result
	}

	if options.DryRun {
		mode := "scan"
		if options.Validate {
			mode = "validate"
		}
		result.Mode = mode
		report, warnings, err := scanSnapshotEvents(ctx, store, campaignID, options.AfterSeq, options.UntilSeq, options.Validate)
		result.Warnings, result.WarningsTotal = capWarnings(warnings, options.WarningsCap)
		if err != nil {
			result.Error = fmt.Sprintf("scan snapshot events: %v", err)
			result.ExitCode = 1
			return result
		}
		payload, err := json.Marshal(report)
		if err != nil {
			result.Error = fmt.Sprintf("encode report: %v", err)
			result.ExitCode = 1
			return result
		}
		result.Report = payload
		if options.Validate && report.InvalidEvents > 0 {
			result.ExitCode = 1
		}
		return result
	}

	result.Mode = "replay"
	applier := projection.Applier{Campaign: store, Daggerheart: store}

	var lastSeq uint64
	var err error
	if options.AfterSeq > 0 {
		lastSeq, err = projection.ReplayCampaignWith(ctx, store, applier, campaignID, projection.ReplayOptions{
			AfterSeq: options.AfterSeq,
			UntilSeq: options.UntilSeq,
			Filter: func(evt event.Event) bool {
				return evt.Type == event.TypeCharacterStateChanged || evt.Type == event.TypeGMFearChanged
			},
		})
	} else {
		lastSeq, err = projection.ReplaySnapshot(ctx, store, applier, campaignID, options.UntilSeq)
	}
	if err != nil {
		result.Error = fmt.Sprintf("replay snapshot: %v", err)
		result.ExitCode = 1
		return result
	}
	report := snapshotScanReport{LastSeq: lastSeq}
	payload, err := json.Marshal(report)
	if err != nil {
		result.Error = fmt.Sprintf("encode report: %v", err)
		result.ExitCode = 1
		return result
	}
	result.Report = payload
	return result
}

func resolveCampaignIDs(singleID, list string) ([]string, error) {
	if singleID == "" && list == "" {
		return nil, fmt.Errorf("-campaign-id or -campaign-ids is required")
	}
	if singleID != "" && list != "" {
		return nil, fmt.Errorf("-campaign-id cannot be combined with -campaign-ids")
	}
	if singleID != "" {
		return []string{singleID}, nil
	}
	ids := splitCSV(list)
	if len(ids) == 0 {
		return nil, fmt.Errorf("-campaign-ids must contain at least one campaign id")
	}
	return ids, nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	output := make([]string, 0, len(parts))
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed == "" {
			continue
		}
		output = append(output, trimmed)
	}
	return output
}

func capWarnings(warnings []string, cap int) ([]string, int) {
	total := len(warnings)
	if cap == 0 || total <= cap {
		return warnings, total
	}
	return warnings[:cap], total
}

func outputJSON(result runResult) {
	encoded, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: encode report: %v\n", err)
		return
	}
	fmt.Println(string(encoded))
}

func printResult(result runResult, prefix string) {
	if result.Error != "" {
		fmt.Fprintf(os.Stderr, "%sError: %s\n", prefix, result.Error)
	}
	if len(result.Warnings) > 0 {
		for _, warning := range result.Warnings {
			fmt.Fprintf(os.Stderr, "%sWarning: %s\n", prefix, warning)
		}
	}
	if result.WarningsTotal > len(result.Warnings) {
		fmt.Fprintf(os.Stderr, "%sWarning: %d more warnings suppressed\n", prefix, result.WarningsTotal-len(result.Warnings))
	}
	if len(result.Report) == 0 {
		return
	}
	if result.Mode == "integrity" {
		var report integrityReport
		if err := json.Unmarshal(result.Report, &report); err != nil {
			fmt.Fprintf(os.Stderr, "%sError: decode report: %v\n", prefix, err)
			return
		}
		fmt.Printf("%sIntegrity check for campaign %s through seq %d\n", prefix, result.CampaignID, report.LastSeq)
		fmt.Printf("%sGM fear match: %t (source=%d replay=%d)\n", prefix, report.GmFearMatch, report.GmFearSource, report.GmFearReplay)
		fmt.Printf("%sCharacter state mismatches: %d (missing states: %d)\n", prefix, report.CharacterMismatches, report.MissingStates)
		return
	}

	var report snapshotScanReport
	if err := json.Unmarshal(result.Report, &report); err != nil {
		fmt.Fprintf(os.Stderr, "%sError: decode report: %v\n", prefix, err)
		return
	}
	if result.Mode == "validate" {
		fmt.Printf("%sValidated snapshot events for campaign %s through seq %d (%d snapshot events, %d invalid, %d total)\n", prefix, result.CampaignID, report.LastSeq, report.SnapshotEvents, report.InvalidEvents, report.TotalEvents)
		return
	}
	if result.Mode == "scan" {
		fmt.Printf("%sScanned snapshot events for campaign %s through seq %d (%d snapshot events, %d total)\n", prefix, result.CampaignID, report.LastSeq, report.SnapshotEvents, report.TotalEvents)
		return
	}
	fmt.Printf("%sReplayed snapshot events for campaign %s through seq %d\n", prefix, result.CampaignID, report.LastSeq)
}

func openStore(path string) (storage.Store, error) {
	cleanPath := filepath.Clean(path)
	if cleanPath == "." || cleanPath == "" {
		return nil, fmt.Errorf("db path is required")
	}
	if dir := filepath.Dir(cleanPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create storage dir: %w", err)
		}
	}
	store, err := sqlite.Open(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite store: %w", err)
	}
	return store, nil
}

func defaultDBPath() string {
	path := os.Getenv("DUALITY_DB_PATH")
	if path == "" {
		path = filepath.Join("data", "fracturing.space.db")
	}
	return path
}

func scanSnapshotEvents(ctx context.Context, eventStore storage.EventStore, campaignID string, afterSeq, untilSeq uint64, validate bool) (snapshotScanReport, []string, error) {
	report := snapshotScanReport{LastSeq: afterSeq}
	warnings := []string{}
	if eventStore == nil {
		return report, warnings, fmt.Errorf("event store is not configured")
	}
	if campaignID == "" {
		return report, warnings, fmt.Errorf("campaign id is required")
	}

	lastSeq := afterSeq
	for {
		events, err := eventStore.ListEvents(ctx, campaignID, lastSeq, adminReplayPageSize)
		if err != nil {
			return report, warnings, err
		}
		if len(events) == 0 {
			report.LastSeq = lastSeq
			return report, warnings, nil
		}
		for _, evt := range events {
			if untilSeq > 0 && evt.Seq > untilSeq {
				report.LastSeq = lastSeq
				return report, warnings, nil
			}
			lastSeq = evt.Seq
			report.TotalEvents++
			if !isSnapshotEvent(evt) {
				continue
			}
			report.SnapshotEvents++
			if validate {
				if err := validateSnapshotEvent(evt); err != nil {
					report.InvalidEvents++
					warnings = append(warnings, fmt.Sprintf("seq %d %s: %v", evt.Seq, evt.Type, err))
				}
			}
		}
		if len(events) < adminReplayPageSize {
			report.LastSeq = lastSeq
			return report, warnings, nil
		}
	}
}

func isSnapshotEvent(evt event.Event) bool {
	switch evt.Type {
	case event.TypeCharacterStateChanged, event.TypeGMFearChanged:
		return true
	default:
		return false
	}
}

func validateSnapshotEvent(evt event.Event) error {
	switch evt.Type {
	case event.TypeCharacterStateChanged:
		var payload event.CharacterStateChangedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return fmt.Errorf("decode character state payload: %w", err)
		}
		if payload.CharacterID == "" {
			return fmt.Errorf("character id is required")
		}
		if payload.SystemState == nil {
			return nil
		}
		dhState, ok := payload.SystemState["daggerheart"]
		if !ok {
			return nil
		}
		stateMap, ok := dhState.(map[string]any)
		if !ok {
			return fmt.Errorf("daggerheart system state must be object")
		}
		if value, ok := stateMap["hope_after"]; ok {
			if _, err := parseSnapshotNumber(value, "hope_after"); err != nil {
				return err
			}
		}
		if value, ok := stateMap["stress_after"]; ok {
			if _, err := parseSnapshotNumber(value, "stress_after"); err != nil {
				return err
			}
		}
	case event.TypeGMFearChanged:
		var payload event.GMFearChangedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return fmt.Errorf("decode gm fear payload: %w", err)
		}
		if payload.After < daggerheart.GMFearMin || payload.After > daggerheart.GMFearMax {
			return fmt.Errorf("gm fear %d exceeds range %d..%d", payload.After, daggerheart.GMFearMin, daggerheart.GMFearMax)
		}
	}
	return nil
}

func parseSnapshotNumber(value any, field string) (int, error) {
	switch v := value.(type) {
	case float64:
		if v != math.Trunc(v) {
			return 0, fmt.Errorf("%s must be an integer", field)
		}
		return int(v), nil
	case float32:
		if v != float32(math.Trunc(float64(v))) {
			return 0, fmt.Errorf("%s must be an integer", field)
		}
		return int(v), nil
	case int:
		return v, nil
	case int32:
		return int(v), nil
	case int64:
		return int(v), nil
	case json.Number:
		parsed, err := v.Int64()
		if err != nil {
			return 0, fmt.Errorf("%s must be an integer", field)
		}
		return int(parsed), nil
	default:
		return 0, fmt.Errorf("%s must be a number", field)
	}
}

func checkSnapshotIntegrity(ctx context.Context, store storage.Store, campaignID string, untilSeq uint64) (integrityReport, []string, error) {
	report := integrityReport{}
	warnings := []string{}
	if store == nil {
		return report, warnings, fmt.Errorf("store is not configured")
	}
	if campaignID == "" {
		return report, warnings, fmt.Errorf("campaign id is required")
	}

	tmpFile, err := os.CreateTemp("", "duality-integrity-*.db")
	if err != nil {
		return report, warnings, fmt.Errorf("create temp db: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return report, warnings, fmt.Errorf("close temp db: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	scratch, err := sqlite.Open(tmpFile.Name())
	if err != nil {
		return report, warnings, fmt.Errorf("open scratch store: %w", err)
	}
	defer func() {
		if err := scratch.Close(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: close scratch store: %v\n", err)
		}
	}()

	campaignRecord, err := store.Get(ctx, campaignID)
	if err != nil {
		return report, warnings, fmt.Errorf("load campaign: %w", err)
	}
	if err := scratch.Put(ctx, campaignRecord); err != nil {
		return report, warnings, fmt.Errorf("seed campaign: %w", err)
	}

	applier := projection.Applier{Campaign: scratch, Daggerheart: scratch}
	lastSeq, err := projection.ReplaySnapshot(ctx, store, applier, campaignID, untilSeq)
	if err != nil {
		return report, warnings, fmt.Errorf("replay snapshot: %w", err)
	}
	report.LastSeq = lastSeq

	sourceSnapshot, err := store.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil {
		return report, warnings, fmt.Errorf("get source snapshot: %w", err)
	}
	replaySnapshot, err := scratch.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil {
		return report, warnings, fmt.Errorf("get replay snapshot: %w", err)
	}
	report.GmFearSource = sourceSnapshot.GMFear
	report.GmFearReplay = replaySnapshot.GMFear
	report.GmFearMatch = sourceSnapshot.GMFear == replaySnapshot.GMFear

	pageToken := ""
	for {
		page, err := store.ListCharacters(ctx, campaignID, adminReplayPageSize, pageToken)
		if err != nil {
			return report, warnings, fmt.Errorf("list characters: %w", err)
		}
		for _, ch := range page.Characters {
			sourceState, err := store.GetDaggerheartCharacterState(ctx, campaignID, ch.ID)
			if err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					report.MissingStates++
					warnings = append(warnings, fmt.Sprintf("missing source state for character %s", ch.ID))
					continue
				}
				return report, warnings, fmt.Errorf("get source state: %w", err)
			}
			replayState, err := scratch.GetDaggerheartCharacterState(ctx, campaignID, ch.ID)
			if err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					report.MissingStates++
					warnings = append(warnings, fmt.Sprintf("missing replay state for character %s", ch.ID))
					continue
				}
				return report, warnings, fmt.Errorf("get replay state: %w", err)
			}
			if sourceState.Hp != replayState.Hp || sourceState.Hope != replayState.Hope || sourceState.Stress != replayState.Stress {
				report.CharacterMismatches++
				warnings = append(warnings, fmt.Sprintf("state mismatch for character %s (source=%d/%d/%d replay=%d/%d/%d)", ch.ID, sourceState.Hp, sourceState.Hope, sourceState.Stress, replayState.Hp, replayState.Hope, replayState.Stress))
			}
		}
		if page.NextPageToken == "" {
			break
		}
		pageToken = page.NextPageToken
	}

	return report, warnings, nil
}
