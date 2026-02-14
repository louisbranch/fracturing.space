package maintenance

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/campaign/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite"
)

const adminReplayPageSize = 200

// Config holds maintenance command configuration.
type Config struct {
	CampaignID        string
	CampaignIDs       string
	EventsDBPath      string        `env:"FRACTURING_SPACE_GAME_EVENTS_DB_PATH"`
	ProjectionsDBPath string        `env:"FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH"`
	Timeout           time.Duration `env:"FRACTURING_SPACE_MAINTENANCE_TIMEOUT" envDefault:"10m"`
	UntilSeq          uint64
	AfterSeq          uint64
	DryRun            bool
	Validate          bool
	Integrity         bool
	WarningsCap       int
	JSONOutput        bool
}

// ParseConfig parses flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse env: %w", err)
	}
	if cfg.EventsDBPath == "" {
		cfg.EventsDBPath = filepath.Join("data", "game-events.db")
	}
	if cfg.ProjectionsDBPath == "" {
		cfg.ProjectionsDBPath = filepath.Join("data", "game-projections.db")
	}
	cfg.WarningsCap = 25

	fs.StringVar(&cfg.CampaignID, "campaign-id", "", "campaign ID to replay snapshot-related events")
	fs.StringVar(&cfg.CampaignIDs, "campaign-ids", "", "comma-separated campaign IDs to replay snapshot-related events")
	fs.StringVar(&cfg.EventsDBPath, "events-db-path", cfg.EventsDBPath, "path to events sqlite database (default: FRACTURING_SPACE_GAME_EVENTS_DB_PATH or data/game-events.db)")
	fs.StringVar(&cfg.ProjectionsDBPath, "projections-db-path", cfg.ProjectionsDBPath, "path to projections sqlite database (default: FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH or data/game-projections.db)")
	fs.Uint64Var(&cfg.UntilSeq, "until-seq", 0, "replay up to this event sequence (0 = latest)")
	fs.Uint64Var(&cfg.AfterSeq, "after-seq", 0, "start replay after this event sequence")
	fs.BoolVar(&cfg.DryRun, "dry-run", false, "scan snapshot-related events without applying projections")
	fs.BoolVar(&cfg.Validate, "validate", false, "validate snapshot event payloads without applying projections (implies -dry-run)")
	fs.BoolVar(&cfg.Integrity, "integrity", false, "replay snapshot-related events into a scratch store and compare against stored projections")
	fs.IntVar(&cfg.WarningsCap, "warnings-cap", cfg.WarningsCap, "max warnings to print (0 = no limit)")
	fs.BoolVar(&cfg.JSONOutput, "json", false, "output JSON reports")
	fs.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "overall timeout")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

// Run executes the maintenance command.
func Run(ctx context.Context, cfg Config, out io.Writer, errOut io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	if cfg.Validate {
		cfg.DryRun = true
	}
	if cfg.Integrity && (cfg.DryRun || cfg.Validate) {
		return errors.New("-integrity cannot be combined with -dry-run or -validate")
	}
	if cfg.Integrity && cfg.AfterSeq > 0 {
		return errors.New("-integrity does not support -after-seq; replay must start at the beginning")
	}

	if _, err := resolveCampaignIDs(cfg.CampaignID, cfg.CampaignIDs); err != nil {
		return err
	}
	if cfg.WarningsCap < 0 {
		return errors.New("-warnings-cap must be >= 0")
	}

	eventStore, projStore, err := openStores(ctx, cfg.EventsDBPath, cfg.ProjectionsDBPath)
	if err != nil {
		return err
	}

	return runWithDeps(ctx, cfg, eventStore, projStore, out, errOut)
}

// runWithDeps contains the core maintenance logic with injectable dependencies.
// It owns the lifecycle of the stores (closing them on return).
func runWithDeps(ctx context.Context, cfg Config, eventStore closableEventStore, projStore closableProjectionStore, out io.Writer, errOut io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	defer func() {
		if err := eventStore.Close(); err != nil {
			fmt.Fprintf(errOut, "Error: close event store: %v\n", err)
		}
		if err := projStore.Close(); err != nil {
			fmt.Fprintf(errOut, "Error: close projection store: %v\n", err)
		}
	}()

	if cfg.Validate {
		cfg.DryRun = true
	}

	ids, err := resolveCampaignIDs(cfg.CampaignID, cfg.CampaignIDs)
	if err != nil {
		return err
	}

	options := runOptions{
		AfterSeq:    cfg.AfterSeq,
		UntilSeq:    cfg.UntilSeq,
		DryRun:      cfg.DryRun,
		Validate:    cfg.Validate,
		Integrity:   cfg.Integrity,
		WarningsCap: cfg.WarningsCap,
		JSONOutput:  cfg.JSONOutput,
	}

	failed := false
	for _, id := range ids {
		result := runCampaign(ctx, eventStore, projStore, id, options, errOut)
		if options.JSONOutput {
			outputJSON(out, errOut, result)
		} else {
			prefix := ""
			if len(ids) > 1 {
				prefix = fmt.Sprintf("[%s] ", id)
			}
			printResult(out, errOut, result, prefix)
		}
		if result.ExitCode != 0 {
			failed = true
		}
	}
	if failed {
		return errors.New("maintenance failed")
	}
	return nil
}

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

func runCampaign(ctx context.Context, eventStore storage.EventStore, projStore storage.ProjectionStore, campaignID string, options runOptions, errOut io.Writer) runResult {
	result := runResult{CampaignID: campaignID}
	if options.Integrity {
		result.Mode = "integrity"
		report, warnings, err := checkSnapshotIntegrity(ctx, eventStore, projStore, campaignID, options.UntilSeq, errOut)
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
		report, warnings, err := scanSnapshotEvents(ctx, eventStore, campaignID, options.AfterSeq, options.UntilSeq, options.Validate)
		result.Warnings, result.WarningsTotal = capWarnings(warnings, options.WarningsCap)
		if err != nil {
			result.Error = fmt.Sprintf("scan snapshot-related events: %v", err)
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
	if projStore == nil {
		result.Error = "projection store is not configured"
		result.ExitCode = 1
		return result
	}
	applier := projection.Applier{Campaign: projStore, Daggerheart: projStore}

	var lastSeq uint64
	var err error
	if options.AfterSeq > 0 {
		lastSeq, err = projection.ReplayCampaignWith(ctx, eventStore, applier, campaignID, projection.ReplayOptions{
			AfterSeq: options.AfterSeq,
			UntilSeq: options.UntilSeq,
			Filter: func(evt event.Event) bool {
				return strings.TrimSpace(evt.SystemID) != ""
			},
		})
	} else {
		lastSeq, err = projection.ReplaySnapshot(ctx, eventStore, applier, campaignID, options.UntilSeq)
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

func capWarnings(warnings []string, limit int) ([]string, int) {
	total := len(warnings)
	if limit == 0 || total <= limit {
		return warnings, total
	}
	return warnings[:limit], total
}

func outputJSON(out io.Writer, errOut io.Writer, result runResult) {
	encoded, err := json.Marshal(result)
	if err != nil {
		fmt.Fprintf(errOut, "Error: encode report: %v\n", err)
		return
	}
	fmt.Fprintln(out, string(encoded))
}

func printResult(out io.Writer, errOut io.Writer, result runResult, prefix string) {
	if result.Error != "" {
		fmt.Fprintf(errOut, "%sError: %s\n", prefix, result.Error)
	}
	if len(result.Warnings) > 0 {
		for _, warning := range result.Warnings {
			fmt.Fprintf(errOut, "%sWarning: %s\n", prefix, warning)
		}
	}
	if result.WarningsTotal > len(result.Warnings) {
		fmt.Fprintf(errOut, "%sWarning: %d more warnings suppressed\n", prefix, result.WarningsTotal-len(result.Warnings))
	}
	if len(result.Report) == 0 {
		return
	}
	if result.Mode == "integrity" {
		var report integrityReport
		if err := json.Unmarshal(result.Report, &report); err != nil {
			fmt.Fprintf(errOut, "%sError: decode report: %v\n", prefix, err)
			return
		}
		fmt.Fprintf(out, "%sIntegrity check for campaign %s through seq %d\n", prefix, result.CampaignID, report.LastSeq)
		fmt.Fprintf(out, "%sGM fear match: %t (source=%d replay=%d)\n", prefix, report.GmFearMatch, report.GmFearSource, report.GmFearReplay)
		fmt.Fprintf(out, "%sCharacter state mismatches: %d (missing states: %d)\n", prefix, report.CharacterMismatches, report.MissingStates)
		return
	}

	var report snapshotScanReport
	if err := json.Unmarshal(result.Report, &report); err != nil {
		fmt.Fprintf(errOut, "%sError: decode report: %v\n", prefix, err)
		return
	}
	if result.Mode == "validate" {
		fmt.Fprintf(out, "%sValidated snapshot-related events for campaign %s through seq %d (%d snapshot-related events, %d invalid, %d total)\n", prefix, result.CampaignID, report.LastSeq, report.SnapshotEvents, report.InvalidEvents, report.TotalEvents)
		return
	}
	if result.Mode == "scan" {
		fmt.Fprintf(out, "%sScanned snapshot-related events for campaign %s through seq %d (%d snapshot-related events, %d total)\n", prefix, result.CampaignID, report.LastSeq, report.SnapshotEvents, report.TotalEvents)
		return
	}
	fmt.Fprintf(out, "%sReplayed snapshot-related events for campaign %s through seq %d\n", prefix, result.CampaignID, report.LastSeq)
}

func openStores(ctx context.Context, eventsPath, projectionsPath string) (*sqlite.Store, *sqlite.Store, error) {
	eventStore, err := openEventStore(ctx, eventsPath)
	if err != nil {
		return nil, nil, err
	}
	projStore, err := openProjectionStore(projectionsPath)
	if err != nil {
		_ = eventStore.Close()
		return nil, nil, err
	}
	return eventStore, projStore, nil
}

func openEventStore(ctx context.Context, path string) (*sqlite.Store, error) {
	cleanPath := filepath.Clean(path)
	if cleanPath == "." || cleanPath == "" {
		return nil, fmt.Errorf("events db path is required")
	}
	if dir := filepath.Dir(cleanPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create storage dir: %w", err)
		}
	}
	keyring, err := integrity.KeyringFromEnv()
	if err != nil {
		return nil, err
	}
	store, err := sqlite.OpenEvents(cleanPath, keyring)
	if err != nil {
		return nil, fmt.Errorf("open events store: %w", err)
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if err := store.VerifyEventIntegrity(ctx); err != nil {
		_ = store.Close()
		return nil, fmt.Errorf("verify event integrity: %w", err)
	}
	return store, nil
}

func openProjectionStore(path string) (*sqlite.Store, error) {
	cleanPath := filepath.Clean(path)
	if cleanPath == "." || cleanPath == "" {
		return nil, fmt.Errorf("projections db path is required")
	}
	if dir := filepath.Dir(cleanPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create storage dir: %w", err)
		}
	}
	store, err := sqlite.OpenProjections(cleanPath)
	if err != nil {
		return nil, fmt.Errorf("open projections store: %w", err)
	}
	return store, nil
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
	return strings.TrimSpace(evt.SystemID) != ""
}

func validateSnapshotEvent(evt event.Event) error {
	switch evt.Type {
	case daggerheart.EventTypeCharacterStatePatched:
		var payload daggerheart.CharacterStatePatchedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return fmt.Errorf("decode character state payload: %w", err)
		}
		if payload.CharacterID == "" {
			return fmt.Errorf("character id is required")
		}
		if payload.HpAfter != nil && (*payload.HpAfter < daggerheart.HPMin || *payload.HpAfter > daggerheart.HPMaxCap) {
			return fmt.Errorf("hp %d exceeds range %d..%d", *payload.HpAfter, daggerheart.HPMin, daggerheart.HPMaxCap)
		}
		if payload.HopeMaxAfter != nil && (*payload.HopeMaxAfter < daggerheart.HopeMin || *payload.HopeMaxAfter > daggerheart.HopeMax) {
			return fmt.Errorf("hope_max %d exceeds range %d..%d", *payload.HopeMaxAfter, daggerheart.HopeMin, daggerheart.HopeMax)
		}
		if payload.HopeAfter != nil {
			maxHope := daggerheart.HopeMax
			if payload.HopeMaxAfter != nil {
				maxHope = *payload.HopeMaxAfter
			}
			if *payload.HopeAfter < daggerheart.HopeMin || *payload.HopeAfter > maxHope {
				return fmt.Errorf("hope %d exceeds range %d..%d", *payload.HopeAfter, daggerheart.HopeMin, maxHope)
			}
		}
		if payload.StressAfter != nil && (*payload.StressAfter < daggerheart.StressMin || *payload.StressAfter > daggerheart.StressMaxCap) {
			return fmt.Errorf("stress %d exceeds range %d..%d", *payload.StressAfter, daggerheart.StressMin, daggerheart.StressMaxCap)
		}
		if payload.ArmorAfter != nil && (*payload.ArmorAfter < daggerheart.ArmorMin || *payload.ArmorAfter > daggerheart.ArmorMaxCap) {
			return fmt.Errorf("armor %d exceeds range %d..%d", *payload.ArmorAfter, daggerheart.ArmorMin, daggerheart.ArmorMaxCap)
		}
		if payload.LifeStateAfter != nil {
			if _, err := daggerheart.NormalizeLifeState(*payload.LifeStateAfter); err != nil {
				return fmt.Errorf("life_state %v is invalid", *payload.LifeStateAfter)
			}
		}
	case daggerheart.EventTypeDeathMoveResolved:
		var payload daggerheart.DeathMoveResolvedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return fmt.Errorf("decode death move payload: %w", err)
		}
		if payload.CharacterID == "" {
			return fmt.Errorf("character id is required")
		}
		if _, err := daggerheart.NormalizeDeathMove(payload.Move); err != nil {
			return fmt.Errorf("death move %v is invalid", payload.Move)
		}
		if payload.LifeStateAfter == "" {
			return fmt.Errorf("life_state_after is required")
		}
		if _, err := daggerheart.NormalizeLifeState(payload.LifeStateAfter); err != nil {
			return fmt.Errorf("life_state_after %v is invalid", payload.LifeStateAfter)
		}
		if payload.HopeDie != nil && (*payload.HopeDie < 1 || *payload.HopeDie > 12) {
			return fmt.Errorf("hope_die %d exceeds range 1..12", *payload.HopeDie)
		}
		if payload.FearDie != nil && (*payload.FearDie < 1 || *payload.FearDie > 12) {
			return fmt.Errorf("fear_die %d exceeds range 1..12", *payload.FearDie)
		}
	case daggerheart.EventTypeBlazeOfGloryResolved:
		var payload daggerheart.BlazeOfGloryResolvedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return fmt.Errorf("decode blaze of glory payload: %w", err)
		}
		if payload.CharacterID == "" {
			return fmt.Errorf("character id is required")
		}
		if payload.LifeStateAfter == "" {
			return fmt.Errorf("life_state_after is required")
		}
		if _, err := daggerheart.NormalizeLifeState(payload.LifeStateAfter); err != nil {
			return fmt.Errorf("life_state_after %v is invalid", payload.LifeStateAfter)
		}
	case daggerheart.EventTypeAttackResolved:
		var payload daggerheart.AttackResolvedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return fmt.Errorf("decode attack payload: %w", err)
		}
		if payload.CharacterID == "" {
			return fmt.Errorf("character id is required")
		}
		if payload.RollSeq == 0 {
			return fmt.Errorf("roll_seq is required")
		}
		if len(payload.Targets) == 0 {
			return fmt.Errorf("targets are required")
		}
		for _, target := range payload.Targets {
			if strings.TrimSpace(target) == "" {
				return fmt.Errorf("targets must not contain empty values")
			}
		}
		if payload.Outcome == "" {
			return fmt.Errorf("outcome is required")
		}
	case daggerheart.EventTypeReactionResolved:
		var payload daggerheart.ReactionResolvedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return fmt.Errorf("decode reaction payload: %w", err)
		}
		if payload.CharacterID == "" {
			return fmt.Errorf("character id is required")
		}
		if payload.RollSeq == 0 {
			return fmt.Errorf("roll_seq is required")
		}
		if payload.Outcome == "" {
			return fmt.Errorf("outcome is required")
		}
	case daggerheart.EventTypeDamageRollResolved:
		var payload daggerheart.DamageRollResolvedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return fmt.Errorf("decode damage roll payload: %w", err)
		}
		if payload.CharacterID == "" {
			return fmt.Errorf("character id is required")
		}
		if payload.RollSeq == 0 {
			return fmt.Errorf("roll_seq is required")
		}
		if len(payload.Rolls) == 0 {
			return fmt.Errorf("rolls are required")
		}
	case daggerheart.EventTypeGMFearChanged:
		var payload daggerheart.GMFearChangedPayload
		if err := json.Unmarshal(evt.PayloadJSON, &payload); err != nil {
			return fmt.Errorf("decode gm fear payload: %w", err)
		}
		if payload.After < daggerheart.GMFearMin || payload.After > daggerheart.GMFearMax {
			return fmt.Errorf("gm fear %d exceeds range %d..%d", payload.After, daggerheart.GMFearMin, daggerheart.GMFearMax)
		}
	}
	return nil
}

func checkSnapshotIntegrity(ctx context.Context, eventStore storage.EventStore, projStore storage.ProjectionStore, campaignID string, untilSeq uint64, errOut io.Writer) (integrityReport, []string, error) {
	report := integrityReport{}
	warnings := []string{}
	if eventStore == nil {
		return report, warnings, fmt.Errorf("event store is not configured")
	}
	if projStore == nil {
		return report, warnings, fmt.Errorf("projection store is not configured")
	}
	if campaignID == "" {
		return report, warnings, fmt.Errorf("campaign id is required")
	}

	tmpFile, err := os.CreateTemp("", "fracturing-space-integrity-*.db")
	if err != nil {
		return report, warnings, fmt.Errorf("create temp db: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return report, warnings, fmt.Errorf("close temp db: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	scratch, err := sqlite.OpenProjections(tmpFile.Name())
	if err != nil {
		return report, warnings, fmt.Errorf("open scratch store: %w", err)
	}
	defer func() {
		if err := scratch.Close(); err != nil {
			if errOut != nil {
				fmt.Fprintf(errOut, "Error: close scratch store: %v\n", err)
			}
		}
	}()

	return checkIntegrityWithStores(ctx, eventStore, projStore, scratch, campaignID, untilSeq, errOut)
}

// checkIntegrityWithStores contains the testable integrity logic. It seeds the
// scratch store, replays events, and compares projections between source and
// scratch.
func checkIntegrityWithStores(ctx context.Context, eventStore storage.EventStore, source storage.ProjectionStore, scratch storage.ProjectionStore, campaignID string, untilSeq uint64, errOut io.Writer) (integrityReport, []string, error) {
	report := integrityReport{}
	warnings := []string{}

	campaignRecord, err := source.Get(ctx, campaignID)
	if err != nil {
		return report, warnings, fmt.Errorf("load campaign: %w", err)
	}
	if err := scratch.Put(ctx, campaignRecord); err != nil {
		return report, warnings, fmt.Errorf("seed campaign: %w", err)
	}

	applier := projection.Applier{Campaign: scratch, Daggerheart: scratch}
	lastSeq, err := projection.ReplaySnapshot(ctx, eventStore, applier, campaignID, untilSeq)
	if err != nil {
		return report, warnings, fmt.Errorf("replay snapshot: %w", err)
	}
	report.LastSeq = lastSeq

	sourceSnapshot, err := source.GetDaggerheartSnapshot(ctx, campaignID)
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
		page, err := source.ListCharacters(ctx, campaignID, adminReplayPageSize, pageToken)
		if err != nil {
			return report, warnings, fmt.Errorf("list characters: %w", err)
		}
		for _, ch := range page.Characters {
			sourceState, err := source.GetDaggerheartCharacterState(ctx, campaignID, ch.ID)
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
