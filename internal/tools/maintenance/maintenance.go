// Package maintenance contains event/replay tooling entrypoints for projection and
// outbox recovery operations.
//
// These commands are intentionally operational (not business-critical runtime)
// and are invoked when read model state must be reconstructed or validated.
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
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/engine"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/event"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart"
	systemmanifest "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/manifest"
	"github.com/louisbranch/fracturing.space/internal/services/game/projection"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage"
	"github.com/louisbranch/fracturing.space/internal/services/game/storage/integrity"
	sqlitecoreprojection "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/coreprojection"
	sqliteeventjournal "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/eventjournal"
)

const (
	adminReplayPageSize = 200
	defaultOutboxLimit  = 50
)

// Command identifies which maintenance operation to execute.
type Command string

const (
	commandReplay           Command = "replay"
	commandOutboxReport     Command = "outbox-report"
	commandOutboxRequeue    Command = "outbox-requeue"
	commandOutboxRequeueAll Command = "outbox-requeue-dead"
	commandGapDetect        Command = "gap-detect"
	commandGapRepair        Command = "gap-repair"
)

// Config holds maintenance command configuration.
type Config struct {
	Command                 Command
	CampaignID              string
	CampaignIDs             string
	EventsDBPath            string        `env:"FRACTURING_SPACE_GAME_EVENTS_DB_PATH"`
	ProjectionsDBPath       string        `env:"FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH"`
	Timeout                 time.Duration `env:"FRACTURING_SPACE_MAINTENANCE_TIMEOUT" envDefault:"10m"`
	UntilSeq                uint64
	AfterSeq                uint64
	DryRun                  bool
	Validate                bool
	Integrity               bool
	WarningsCap             int
	JSONOutput              bool
	OutboxStatus            string
	OutboxLimit             int
	OutboxRequeueDeadLimit  int
	OutboxRequeueCampaignID string
	OutboxRequeueSeq        uint64
}

type envConfig struct {
	EventsDBPath      string        `env:"FRACTURING_SPACE_GAME_EVENTS_DB_PATH"`
	ProjectionsDBPath string        `env:"FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH"`
	Timeout           time.Duration `env:"FRACTURING_SPACE_MAINTENANCE_TIMEOUT" envDefault:"10m"`
}

// ParseConfig parses flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	_ = fs
	cfg, err := defaultConfig()
	if err != nil {
		return Config{}, err
	}
	if len(args) == 0 {
		return Config{}, fmt.Errorf("maintenance subcommand is required\n\n%s", maintenanceUsage())
	}

	command := Command(strings.TrimSpace(args[0]))
	commandArgs := args[1:]
	switch command {
	case commandReplay:
		flags := flag.NewFlagSet(string(commandReplay), flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		bindReplayFlags(flags, &cfg)
		if err := flags.Parse(commandArgs); err != nil {
			return Config{}, err
		}
	case commandOutboxReport:
		flags := flag.NewFlagSet(string(commandOutboxReport), flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		bindOutboxReportFlags(flags, &cfg)
		if err := flags.Parse(commandArgs); err != nil {
			return Config{}, err
		}
	case commandOutboxRequeue:
		flags := flag.NewFlagSet(string(commandOutboxRequeue), flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		bindOutboxRequeueFlags(flags, &cfg)
		if err := flags.Parse(commandArgs); err != nil {
			return Config{}, err
		}
	case commandOutboxRequeueAll:
		flags := flag.NewFlagSet(string(commandOutboxRequeueAll), flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		bindOutboxRequeueDeadFlags(flags, &cfg)
		if err := flags.Parse(commandArgs); err != nil {
			return Config{}, err
		}
	case commandGapDetect:
		flags := flag.NewFlagSet(string(commandGapDetect), flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		bindGapFlags(flags, &cfg)
		if err := flags.Parse(commandArgs); err != nil {
			return Config{}, err
		}
	case commandGapRepair:
		flags := flag.NewFlagSet(string(commandGapRepair), flag.ContinueOnError)
		flags.SetOutput(io.Discard)
		bindGapFlags(flags, &cfg)
		if err := flags.Parse(commandArgs); err != nil {
			return Config{}, err
		}
	default:
		return Config{}, fmt.Errorf("unknown maintenance subcommand %q\n\n%s", command, maintenanceUsage())
	}
	cfg.Command = command
	return cfg, nil
}

func defaultConfig() (Config, error) {
	var envCfg envConfig
	if err := env.Parse(&envCfg); err != nil {
		return Config{}, fmt.Errorf("parse env: %w", err)
	}

	cfg := Config{
		EventsDBPath:      envCfg.EventsDBPath,
		ProjectionsDBPath: envCfg.ProjectionsDBPath,
		Timeout:           envCfg.Timeout,
		WarningsCap:       25,
	}
	if cfg.EventsDBPath == "" {
		cfg.EventsDBPath = filepath.Join("data", "game-events.db")
	}
	if cfg.ProjectionsDBPath == "" {
		cfg.ProjectionsDBPath = filepath.Join("data", "game-projections.db")
	}
	return cfg, nil
}

func bindReplayFlags(fs *flag.FlagSet, cfg *Config) {
	bindAllStoreFlags(fs, cfg)
	fs.StringVar(&cfg.CampaignID, "campaign-id", "", "campaign ID to replay snapshot-related events")
	fs.StringVar(&cfg.CampaignIDs, "campaign-ids", "", "comma-separated campaign IDs to replay snapshot-related events")
	fs.Uint64Var(&cfg.UntilSeq, "until-seq", 0, "replay up to this event sequence (0 = latest)")
	fs.Uint64Var(&cfg.AfterSeq, "after-seq", 0, "start replay after this event sequence")
	fs.BoolVar(&cfg.DryRun, "dry-run", false, "scan snapshot-related events without applying projections")
	fs.BoolVar(&cfg.Validate, "validate", false, "validate snapshot event payloads without applying projections (implies -dry-run)")
	fs.BoolVar(&cfg.Integrity, "integrity", false, "replay snapshot-related events into a scratch store and compare against stored projections")
	fs.IntVar(&cfg.WarningsCap, "warnings-cap", cfg.WarningsCap, "max warnings to print (0 = no limit)")
	fs.BoolVar(&cfg.JSONOutput, "json", false, "output JSON reports")
}

func bindOutboxReportFlags(fs *flag.FlagSet, cfg *Config) {
	bindEventStoreFlags(fs, cfg)
	if cfg.OutboxLimit <= 0 {
		cfg.OutboxLimit = defaultOutboxLimit
	}
	fs.BoolVar(&cfg.JSONOutput, "json", false, "output JSON reports")
	fs.StringVar(&cfg.OutboxStatus, "outbox-status", "", "optional outbox status filter (pending|processing|failed|dead)")
	fs.IntVar(&cfg.OutboxLimit, "outbox-limit", cfg.OutboxLimit, "max outbox rows to print/list")
}

func bindOutboxRequeueFlags(fs *flag.FlagSet, cfg *Config) {
	bindEventStoreFlags(fs, cfg)
	fs.BoolVar(&cfg.JSONOutput, "json", false, "output JSON reports")
	fs.StringVar(&cfg.OutboxRequeueCampaignID, "outbox-requeue-campaign-id", "", "campaign id for outbox requeue")
	fs.Uint64Var(&cfg.OutboxRequeueSeq, "outbox-requeue-seq", 0, "event sequence for outbox requeue")
}

func bindOutboxRequeueDeadFlags(fs *flag.FlagSet, cfg *Config) {
	bindEventStoreFlags(fs, cfg)
	fs.BoolVar(&cfg.JSONOutput, "json", false, "output JSON reports")
	fs.IntVar(&cfg.OutboxRequeueDeadLimit, "outbox-requeue-dead-limit", 0, "max dead outbox rows to requeue (required with -outbox-requeue-dead)")
}

func bindGapFlags(fs *flag.FlagSet, cfg *Config) {
	bindAllStoreFlags(fs, cfg)
	fs.BoolVar(&cfg.JSONOutput, "json", false, "output JSON reports")
}

func bindEventStoreFlags(fs *flag.FlagSet, cfg *Config) {
	fs.StringVar(&cfg.EventsDBPath, "events-db-path", cfg.EventsDBPath, "path to events sqlite database (default: FRACTURING_SPACE_GAME_EVENTS_DB_PATH or data/game-events.db)")
	fs.DurationVar(&cfg.Timeout, "timeout", cfg.Timeout, "overall timeout")
}

func bindAllStoreFlags(fs *flag.FlagSet, cfg *Config) {
	bindEventStoreFlags(fs, cfg)
	fs.StringVar(&cfg.ProjectionsDBPath, "projections-db-path", cfg.ProjectionsDBPath, "path to projections sqlite database (default: FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH or data/game-projections.db)")
}

func maintenanceUsage() string {
	return strings.TrimSpace(`
Usage:
  maintenance replay [flags]
  maintenance outbox-report [flags]
  maintenance outbox-requeue [flags]
  maintenance outbox-requeue-dead [flags]
  maintenance gap-detect [flags]
  maintenance gap-repair [flags]

Examples:
  maintenance replay -campaign-id <id> -validate
  maintenance outbox-report -outbox-status failed -outbox-limit 50
  maintenance gap-repair -json
`)
}

// Run executes the maintenance command.
func Run(ctx context.Context, cfg Config, out io.Writer, errOut io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	if err := validateCommandConfig(cfg); err != nil {
		return err
	}

	switch cfg.Command {
	case commandReplay:
		return runReplayCommand(ctx, cfg, out, errOut)
	case commandOutboxReport:
		return runOutboxReportCommand(ctx, cfg, out, errOut)
	case commandOutboxRequeue:
		return runOutboxRequeueCommand(ctx, cfg, out, errOut)
	case commandOutboxRequeueAll:
		return runOutboxRequeueDeadCommand(ctx, cfg, out, errOut)
	case commandGapDetect:
		return runGapCommand(ctx, cfg, false, out, errOut)
	case commandGapRepair:
		return runGapCommand(ctx, cfg, true, out, errOut)
	default:
		return fmt.Errorf("unknown maintenance subcommand %q\n\n%s", cfg.Command, maintenanceUsage())
	}
}

func validateCommandConfig(cfg Config) error {
	switch cfg.Command {
	case commandReplay:
		return validateReplayConfig(cfg)
	case commandOutboxReport:
		return validateOutboxReportConfig(cfg)
	case commandOutboxRequeue:
		return validateOutboxRequeueConfig(cfg)
	case commandOutboxRequeueAll:
		return validateOutboxRequeueDeadConfig(cfg)
	case commandGapDetect, commandGapRepair:
		return validateGapConfig(cfg)
	case "":
		return fmt.Errorf("maintenance subcommand is required\n\n%s", maintenanceUsage())
	default:
		return nil
	}
}

func validateReplayConfig(cfg Config) error {
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
	return nil
}

func validateOutboxReportConfig(cfg Config) error {
	if cfg.CampaignID != "" || cfg.CampaignIDs != "" {
		return errors.New("-outbox-report cannot be combined with -campaign-id or -campaign-ids")
	}
	if cfg.DryRun || cfg.Validate || cfg.Integrity || cfg.AfterSeq > 0 || cfg.UntilSeq > 0 {
		return errors.New("-outbox-report cannot be combined with replay/scan flags")
	}
	if cfg.OutboxLimit <= 0 {
		return errors.New("-outbox-limit must be > 0")
	}
	if cfg.OutboxRequeueDeadLimit > 0 || strings.TrimSpace(cfg.OutboxRequeueCampaignID) != "" || cfg.OutboxRequeueSeq > 0 {
		return errors.New("-outbox-report cannot be combined with outbox requeue flags")
	}
	return nil
}

func validateOutboxRequeueConfig(cfg Config) error {
	if cfg.CampaignID != "" || cfg.CampaignIDs != "" {
		return errors.New("-outbox-requeue cannot be combined with -campaign-id or -campaign-ids")
	}
	if cfg.DryRun || cfg.Validate || cfg.Integrity || cfg.AfterSeq > 0 || cfg.UntilSeq > 0 {
		return errors.New("-outbox-requeue cannot be combined with replay/scan flags")
	}
	if strings.TrimSpace(cfg.OutboxStatus) != "" || cfg.OutboxLimit > 0 {
		return errors.New("-outbox-requeue cannot be combined with -outbox-status or -outbox-limit")
	}
	if strings.TrimSpace(cfg.OutboxRequeueCampaignID) == "" {
		return errors.New("-outbox-requeue-campaign-id is required")
	}
	if cfg.OutboxRequeueSeq == 0 {
		return errors.New("-outbox-requeue-seq must be > 0")
	}
	if cfg.OutboxRequeueDeadLimit > 0 {
		return errors.New("-outbox-requeue cannot be combined with -outbox-requeue-dead-limit")
	}
	return nil
}

func validateOutboxRequeueDeadConfig(cfg Config) error {
	if cfg.CampaignID != "" || cfg.CampaignIDs != "" {
		return errors.New("-outbox-requeue-dead cannot be combined with -campaign-id or -campaign-ids")
	}
	if cfg.DryRun || cfg.Validate || cfg.Integrity || cfg.AfterSeq > 0 || cfg.UntilSeq > 0 {
		return errors.New("-outbox-requeue-dead cannot be combined with replay/scan flags")
	}
	if strings.TrimSpace(cfg.OutboxStatus) != "" || cfg.OutboxLimit > 0 {
		return errors.New("-outbox-requeue-dead cannot be combined with -outbox-status or -outbox-limit")
	}
	if cfg.OutboxRequeueDeadLimit <= 0 {
		return errors.New("-outbox-requeue-dead-limit must be > 0")
	}
	if strings.TrimSpace(cfg.OutboxRequeueCampaignID) != "" || cfg.OutboxRequeueSeq > 0 {
		return errors.New("-outbox-requeue-dead cannot be combined with -outbox-requeue-campaign-id or -outbox-requeue-seq")
	}
	return nil
}

func validateGapConfig(cfg Config) error {
	if cfg.CampaignID != "" || cfg.CampaignIDs != "" {
		return errors.New("-gap-detect/-gap-repair cannot be combined with -campaign-id or -campaign-ids")
	}
	if cfg.DryRun || cfg.Validate || cfg.Integrity || cfg.AfterSeq > 0 || cfg.UntilSeq > 0 {
		return errors.New("-gap-detect/-gap-repair cannot be combined with replay/scan flags")
	}
	if cfg.OutboxLimit > 0 || cfg.OutboxRequeueDeadLimit > 0 || strings.TrimSpace(cfg.OutboxStatus) != "" || strings.TrimSpace(cfg.OutboxRequeueCampaignID) != "" || cfg.OutboxRequeueSeq != 0 {
		return errors.New("-gap-detect/-gap-repair cannot be combined with outbox flags")
	}
	return nil
}

func runReplayCommand(ctx context.Context, cfg Config, out io.Writer, errOut io.Writer) error {
	eventStore, projStore, err := openStores(ctx, cfg.EventsDBPath, cfg.ProjectionsDBPath)
	if err != nil {
		return err
	}
	return runWithDeps(ctx, cfg, eventStore, projStore, out, errOut)
}

func runOutboxReportCommand(ctx context.Context, cfg Config, out io.Writer, errOut io.Writer) error {
	eventStore, err := openEventStore(ctx, cfg.EventsDBPath)
	if err != nil {
		return err
	}
	defer closeStore(errOut, "event store", eventStore)
	return runOutboxReport(ctx, eventStore.ProjectionApplyOutboxStore(), cfg.OutboxStatus, cfg.OutboxLimit, cfg.JSONOutput, out, errOut)
}

func runOutboxRequeueCommand(ctx context.Context, cfg Config, out io.Writer, errOut io.Writer) error {
	eventStore, err := openEventStore(ctx, cfg.EventsDBPath)
	if err != nil {
		return err
	}
	defer closeStore(errOut, "event store", eventStore)
	return runOutboxRequeue(
		ctx,
		eventStore.ProjectionApplyOutboxStore(),
		cfg.OutboxRequeueCampaignID,
		cfg.OutboxRequeueSeq,
		time.Now().UTC(),
		cfg.JSONOutput,
		out,
		errOut,
	)
}

func runOutboxRequeueDeadCommand(ctx context.Context, cfg Config, out io.Writer, errOut io.Writer) error {
	eventStore, err := openEventStore(ctx, cfg.EventsDBPath)
	if err != nil {
		return err
	}
	defer closeStore(errOut, "event store", eventStore)
	return runOutboxRequeueDeadRows(
		ctx,
		eventStore.ProjectionApplyOutboxStore(),
		cfg.OutboxRequeueDeadLimit,
		time.Now().UTC(),
		cfg.JSONOutput,
		out,
		errOut,
	)
}

func runGapCommand(ctx context.Context, cfg Config, repair bool, out io.Writer, errOut io.Writer) error {
	eventStore, projStore, err := openStores(ctx, cfg.EventsDBPath, cfg.ProjectionsDBPath)
	if err != nil {
		return err
	}
	defer closeStore(errOut, "event store", eventStore)
	defer closeStore(errOut, "projection store", projStore)

	if repair {
		return runGapRepair(ctx, eventStore, projStore, cfg.JSONOutput, out, errOut)
	}
	return runGapDetect(ctx, eventStore, projStore, cfg.JSONOutput, out, errOut)
}

func closeStore(errOut io.Writer, name string, store interface{ Close() error }) {
	if err := store.Close(); err != nil {
		fmt.Fprintf(errOut, "Error: close %s: %v\n", name, err)
	}
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
	systemAdapters, err := systemmanifest.AdapterRegistry(systemmanifest.ProjectionStoresFromSource(projStore))
	if err != nil {
		result.Error = fmt.Sprintf("build projection adapters: %v", err)
		result.ExitCode = 1
		return result
	}
	applier := projection.Applier{
		Campaign: projStore,
		Adapters: systemAdapters,
	}

	var lastSeq uint64
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

func openStores(ctx context.Context, eventsPath, projectionsPath string) (*sqliteeventjournal.Store, *sqlitecoreprojection.Store, error) {
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

type outboxInspector interface {
	GetProjectionApplyOutboxSummary(context.Context) (storage.ProjectionApplyOutboxSummary, error)
	ListProjectionApplyOutboxRows(context.Context, string, int) ([]storage.ProjectionApplyOutboxEntry, error)
}

type outboxRequeuer interface {
	RequeueProjectionApplyOutboxRow(context.Context, string, uint64, time.Time) (bool, error)
	RequeueProjectionApplyOutboxDeadRows(context.Context, int, time.Time) (int, error)
}

type outboxReport struct {
	Mode    string                               `json:"mode"`
	Status  string                               `json:"status,omitempty"`
	Limit   int                                  `json:"limit"`
	Summary storage.ProjectionApplyOutboxSummary `json:"summary"`
	Rows    []storage.ProjectionApplyOutboxEntry `json:"rows"`
}

type outboxRequeueResult struct {
	Mode       string `json:"mode"`
	CampaignID string `json:"campaign_id"`
	Seq        uint64 `json:"seq"`
	Requeued   bool   `json:"requeued"`
}

type outboxRequeueDeadResult struct {
	Mode     string `json:"mode"`
	Limit    int    `json:"limit"`
	Requeued int    `json:"requeued"`
}

func runOutboxReport(
	ctx context.Context,
	inspector outboxInspector,
	status string,
	limit int,
	jsonOutput bool,
	out io.Writer,
	errOut io.Writer,
) error {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}
	if inspector == nil {
		return fmt.Errorf("outbox inspector is not configured")
	}
	if limit <= 0 {
		return fmt.Errorf("outbox limit must be > 0")
	}

	summary, err := inspector.GetProjectionApplyOutboxSummary(ctx)
	if err != nil {
		return fmt.Errorf("read outbox summary: %w", err)
	}
	rows, err := inspector.ListProjectionApplyOutboxRows(ctx, status, limit)
	if err != nil {
		return fmt.Errorf("list outbox rows: %w", err)
	}

	if jsonOutput {
		report := outboxReport{
			Mode:    "outbox",
			Status:  strings.TrimSpace(status),
			Limit:   limit,
			Summary: summary,
			Rows:    rows,
		}
		encoded, err := json.Marshal(report)
		if err != nil {
			return fmt.Errorf("encode outbox report: %w", err)
		}
		fmt.Fprintln(out, string(encoded))
		return nil
	}

	fmt.Fprintf(
		out,
		"Outbox summary: pending=%d processing=%d failed=%d dead=%d\n",
		summary.PendingCount,
		summary.ProcessingCount,
		summary.FailedCount,
		summary.DeadCount,
	)
	if summary.OldestPendingCampaignID == "" || summary.OldestPendingSeq == 0 || summary.OldestPendingAt.IsZero() {
		fmt.Fprintln(out, "Oldest pending/failed row: none")
	} else {
		fmt.Fprintf(
			out,
			"Oldest pending/failed row: %s/%d next_attempt_at=%s\n",
			summary.OldestPendingCampaignID,
			summary.OldestPendingSeq,
			summary.OldestPendingAt.Format(time.RFC3339),
		)
	}
	filter := strings.TrimSpace(status)
	if filter == "" {
		fmt.Fprintf(out, "Rows (all statuses, limit=%d):\n", limit)
	} else {
		fmt.Fprintf(out, "Rows (status=%s, limit=%d):\n", filter, limit)
	}
	for _, row := range rows {
		fmt.Fprintf(
			out,
			"- %s/%d status=%s attempts=%d next_attempt_at=%s type=%s\n",
			row.CampaignID,
			row.Seq,
			row.Status,
			row.AttemptCount,
			row.NextAttemptAt.Format(time.RFC3339),
			row.EventType,
		)
		if strings.TrimSpace(row.LastError) != "" {
			fmt.Fprintf(out, "  last_error=%s\n", row.LastError)
		}
	}
	return nil
}

func runOutboxRequeue(
	ctx context.Context,
	requeuer outboxRequeuer,
	campaignID string,
	seq uint64,
	now time.Time,
	jsonOutput bool,
	out io.Writer,
	errOut io.Writer,
) error {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}
	if requeuer == nil {
		return fmt.Errorf("outbox requeuer is not configured")
	}
	campaignID = strings.TrimSpace(campaignID)
	if campaignID == "" {
		return fmt.Errorf("campaign id is required")
	}
	if seq == 0 {
		return fmt.Errorf("event sequence must be greater than zero")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	requeued, err := requeuer.RequeueProjectionApplyOutboxRow(ctx, campaignID, seq, now)
	if err != nil {
		return fmt.Errorf("requeue outbox row: %w", err)
	}
	if !requeued {
		return fmt.Errorf("dead outbox row not found for %s/%d", campaignID, seq)
	}

	if jsonOutput {
		payload, err := json.Marshal(outboxRequeueResult{
			Mode:       "outbox-requeue",
			CampaignID: campaignID,
			Seq:        seq,
			Requeued:   true,
		})
		if err != nil {
			return fmt.Errorf("encode outbox requeue report: %w", err)
		}
		fmt.Fprintln(out, string(payload))
		return nil
	}

	fmt.Fprintf(out, "Requeued outbox row: %s/%d\n", campaignID, seq)
	return nil
}

func runOutboxRequeueDeadRows(
	ctx context.Context,
	requeuer outboxRequeuer,
	limit int,
	now time.Time,
	jsonOutput bool,
	out io.Writer,
	errOut io.Writer,
) error {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}
	if requeuer == nil {
		return fmt.Errorf("outbox requeuer is not configured")
	}
	if limit <= 0 {
		return fmt.Errorf("outbox requeue limit must be > 0")
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	requeued, err := requeuer.RequeueProjectionApplyOutboxDeadRows(ctx, limit, now)
	if err != nil {
		return fmt.Errorf("requeue dead outbox rows: %w", err)
	}

	if jsonOutput {
		payload, err := json.Marshal(outboxRequeueDeadResult{
			Mode:     "outbox-requeue-dead",
			Limit:    limit,
			Requeued: requeued,
		})
		if err != nil {
			return fmt.Errorf("encode outbox dead requeue report: %w", err)
		}
		fmt.Fprintln(out, string(payload))
		return nil
	}

	fmt.Fprintf(out, "Requeued dead outbox rows: %d (limit=%d)\n", requeued, limit)
	return nil
}

// buildEventRegistry constructs the v2 event registry for validation.
func buildEventRegistry() (*event.Registry, error) {
	registries, err := engine.BuildRegistries(daggerheart.NewModule())
	if err != nil {
		return nil, err
	}
	return registries.Events, nil
}

func openEventStore(ctx context.Context, path string) (*sqliteeventjournal.Store, error) {
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
	registry, err := buildEventRegistry()
	if err != nil {
		return nil, fmt.Errorf("build registries: %w", err)
	}
	store, err := sqliteeventjournal.Open(cleanPath, keyring, registry)
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

func openProjectionStore(path string) (*sqlitecoreprojection.Store, error) {
	cleanPath := filepath.Clean(path)
	if cleanPath == "." || cleanPath == "" {
		return nil, fmt.Errorf("projections db path is required")
	}
	if dir := filepath.Dir(cleanPath); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create storage dir: %w", err)
		}
	}
	store, err := sqlitecoreprojection.Open(cleanPath)
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
	var registry *event.Registry
	if validate {
		var err error
		registry, err = buildEventRegistry()
		if err != nil {
			return report, warnings, fmt.Errorf("build event registry: %w", err)
		}
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
				if err := validateSnapshotEvent(registry, evt); err != nil {
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

func validateSnapshotEvent(registry *event.Registry, evt event.Event) error {
	if registry == nil {
		return fmt.Errorf("event registry is required")
	}
	validated := evt
	validated.Seq = 0
	validated.Hash = ""
	validated.PrevHash = ""
	validated.ChainHash = ""
	validated.Signature = ""
	validated.SignatureKeyID = ""
	_, err := registry.ValidateForAppend(validated)
	if err != nil {
		if errors.Is(err, event.ErrTypeUnknown) {
			return nil
		}
		return err
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

	scratch, err := sqlitecoreprojection.Open(tmpFile.Name())
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

	systemAdapters, err := systemmanifest.AdapterRegistry(systemmanifest.ProjectionStoresFromSource(scratch))
	if err != nil {
		return report, warnings, fmt.Errorf("build projection adapters: %w", err)
	}
	applier := projection.Applier{
		Campaign: scratch,
		Adapters: systemAdapters,
	}
	lastSeq, err := projection.ReplaySnapshot(ctx, eventStore, applier, campaignID, untilSeq)
	if err != nil {
		return report, warnings, fmt.Errorf("replay snapshot: %w", err)
	}
	report.LastSeq = lastSeq

	sourceDH := systemmanifest.ProjectionStoresFromSource(source).Daggerheart
	scratchDH := systemmanifest.ProjectionStoresFromSource(scratch).Daggerheart
	if sourceDH == nil || scratchDH == nil {
		return report, warnings, nil
	}

	sourceSnapshot, err := sourceDH.GetDaggerheartSnapshot(ctx, campaignID)
	if err != nil {
		return report, warnings, fmt.Errorf("get source snapshot: %w", err)
	}
	replaySnapshot, err := scratchDH.GetDaggerheartSnapshot(ctx, campaignID)
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
			sourceState, err := sourceDH.GetDaggerheartCharacterState(ctx, campaignID, ch.ID)
			if err != nil {
				if errors.Is(err, storage.ErrNotFound) {
					report.MissingStates++
					warnings = append(warnings, fmt.Sprintf("missing source state for character %s", ch.ID))
					continue
				}
				return report, warnings, fmt.Errorf("get source state: %w", err)
			}
			replayState, err := scratchDH.GetDaggerheartCharacterState(ctx, campaignID, ch.ID)
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

// gapDetectReport holds the result of a projection gap detection scan.
type gapDetectReport struct {
	Mode string                     `json:"mode"`
	Gaps []projection.ProjectionGap `json:"gaps"`
}

// gapRepairReport holds the result of a projection gap repair operation.
type gapRepairReport struct {
	Mode    string                       `json:"mode"`
	Results []projection.GapRepairResult `json:"results"`
}

// runGapDetect compares projection watermarks against the event journal and
// reports campaigns where projections are behind.
func runGapDetect(ctx context.Context, eventStore storage.EventStore, projStore storage.ProjectionWatermarkStore, jsonOutput bool, out io.Writer, errOut io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	gaps, err := projection.DetectProjectionGaps(ctx, projStore, eventStore)
	if err != nil {
		return fmt.Errorf("detect projection gaps: %w", err)
	}

	if jsonOutput {
		report := gapDetectReport{Mode: "gap-detect", Gaps: gaps}
		encoded, encErr := json.Marshal(report)
		if encErr != nil {
			return fmt.Errorf("encode gap detect report: %w", encErr)
		}
		fmt.Fprintln(out, string(encoded))
		return nil
	}

	if len(gaps) == 0 {
		fmt.Fprintln(out, "No projection gaps detected.")
		return nil
	}
	fmt.Fprintf(out, "Detected %d projection gap(s):\n", len(gaps))
	for _, gap := range gaps {
		fmt.Fprintf(out, "  %s: watermark=%d journal=%d (behind by %d)\n",
			gap.CampaignID, gap.WatermarkSeq, gap.JournalSeq, gap.JournalSeq-gap.WatermarkSeq)
	}
	return nil
}

// runGapRepair detects projection gaps and replays missing events to close them.
func runGapRepair(ctx context.Context, eventStore storage.EventStore, projStore storage.ProjectionStore, jsonOutput bool, out io.Writer, errOut io.Writer) error {
	if out == nil {
		out = io.Discard
	}
	if errOut == nil {
		errOut = io.Discard
	}

	systemAdapters, err := systemmanifest.AdapterRegistry(systemmanifest.ProjectionStoresFromSource(projStore))
	if err != nil {
		return fmt.Errorf("build projection adapters: %w", err)
	}
	applier := projection.Applier{
		Campaign:   projStore,
		Adapters:   systemAdapters,
		Watermarks: projStore,
	}

	results, err := projection.RepairProjectionGaps(ctx, projStore, eventStore, applier)
	if err != nil {
		return fmt.Errorf("repair projection gaps: %w", err)
	}

	if jsonOutput {
		report := gapRepairReport{Mode: "gap-repair", Results: results}
		encoded, encErr := json.Marshal(report)
		if encErr != nil {
			return fmt.Errorf("encode gap repair report: %w", encErr)
		}
		fmt.Fprintln(out, string(encoded))
		return nil
	}

	if len(results) == 0 {
		fmt.Fprintln(out, "No projection gaps found.")
		return nil
	}
	fmt.Fprintf(out, "Repaired %d campaign(s):\n", len(results))
	for _, r := range results {
		fmt.Fprintf(out, "  %s: replayed %d event(s)\n", r.CampaignID, r.EventsReplayed)
	}
	return nil
}
