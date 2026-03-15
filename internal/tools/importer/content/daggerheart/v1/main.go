package catalogimporter

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteconn"
	"github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/contentstore"
	sqlitedaggerheartcontent "github.com/louisbranch/fracturing.space/internal/services/game/storage/sqlite/daggerheartcontent"
)

const (
	defaultBaseLocale = "en-US"
	defaultSystemID   = "daggerheart"
	defaultSystemVer  = "v1"
)

var (
	defaultOpenRetryMaxAttempts = 5
	defaultOpenRetryBaseDelay   = 250 * time.Millisecond
	defaultOpenRetryMaxDelay    = 2 * time.Second
)

type contentStore interface {
	contentstore.DaggerheartContentStore
	contentstore.DaggerheartCatalogReadinessStore
	Close() error
}

type openContentStoreFunc func(path string) (contentStore, error)

type runDeps struct {
	openStore openContentStoreFunc
	nowUTC    func() time.Time
}

func defaultRunDeps() runDeps {
	return runDeps{
		openStore: openSQLiteContentStore,
		nowUTC: func() time.Time {
			return time.Now().UTC()
		},
	}
}

func openSQLiteContentStore(path string) (contentStore, error) {
	return sqlitedaggerheartcontent.Open(path)
}

// Config holds configuration for the catalog importer.
type Config struct {
	Dir         string
	DBPath      string
	BaseLocale  string
	DryRun      bool
	SkipIfReady bool
}

// ParseConfig parses CLI flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	cfg := Config{
		DBPath:     filepath.Join("data", "game-content.db"),
		BaseLocale: defaultBaseLocale,
	}

	fs.StringVar(&cfg.Dir, "dir", "", "directory containing locale subfolders")
	fs.StringVar(&cfg.DBPath, "db-path", cfg.DBPath, "content database path")
	fs.StringVar(&cfg.BaseLocale, "base-locale", cfg.BaseLocale, "base locale used for catalog data")
	fs.BoolVar(&cfg.DryRun, "dry-run", false, "validate without writing to the database")
	fs.BoolVar(&cfg.SkipIfReady, "skip-if-ready", false, "skip import when required catalog sections are already populated")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	if strings.TrimSpace(cfg.Dir) == "" {
		return Config{}, errors.New("dir is required")
	}
	if strings.TrimSpace(cfg.BaseLocale) == "" {
		return Config{}, errors.New("base-locale is required")
	}

	return cfg, nil
}

// Run executes the importer using the provided Config.
func Run(ctx context.Context, cfg Config, out io.Writer) error {
	return runWithDeps(ctx, cfg, out, defaultRunDeps())
}

func runWithDeps(ctx context.Context, cfg Config, out io.Writer, deps runDeps) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if out == nil {
		out = io.Discard
	}
	if !cfg.DryRun && deps.openStore == nil {
		return errors.New("content store opener is required")
	}
	if deps.nowUTC == nil {
		deps.nowUTC = func() time.Time {
			return time.Now().UTC()
		}
	}

	dir := strings.TrimSpace(cfg.Dir)
	if dir == "" {
		return errors.New("dir is required")
	}
	baseLocale := strings.TrimSpace(cfg.BaseLocale)
	if baseLocale == "" {
		return errors.New("base-locale is required")
	}

	locales, err := listLocaleDirs(dir)
	if err != nil {
		return err
	}
	if len(locales) == 0 {
		return errors.New("no locale directories found")
	}
	if !contains(locales, baseLocale) {
		return fmt.Errorf("base-locale %s not found in %s", baseLocale, dir)
	}

	var store contentStore
	if !cfg.DryRun {
		contentStore, err := openContentStoreWithRetry(ctx, cfg.DBPath, out, deps.openStore)
		if err != nil {
			return fmt.Errorf("open content store: %w", err)
		}
		defer contentStore.Close()
		store = contentStore

		if cfg.SkipIfReady {
			readiness, err := contentstore.EvaluateDaggerheartCatalogReadiness(ctx, contentStore)
			if err != nil {
				return fmt.Errorf("check catalog readiness: %w", err)
			}
			if readiness.Ready {
				_, err = fmt.Fprintf(out, "catalog already ready in %s; skipping import\n", cfg.DBPath)
				return err
			}
		}
	}

	for _, locale := range locales {
		if err := processLocale(ctx, store, dir, locale, baseLocale, cfg.DryRun, deps.nowUTC()); err != nil {
			return err
		}
	}

	if cfg.DryRun {
		_, err = fmt.Fprintf(out, "validated %d locale(s)\n", len(locales))
		return err
	}
	_, err = fmt.Fprintf(out, "imported %d locale(s) into %s\n", len(locales), cfg.DBPath)
	return err
}

func processLocale(ctx context.Context, store contentstore.DaggerheartContentStore, rootDir, locale, baseLocale string, dryRun bool, now time.Time) error {
	localeDir := filepath.Join(rootDir, locale)
	payloads, err := readLocalePayloads(localeDir)
	if err != nil {
		return fmt.Errorf("read %s: %w", locale, err)
	}
	if err := validateLocalePayloads(locale, payloads); err != nil {
		return fmt.Errorf("validate %s: %w", locale, err)
	}
	if dryRun {
		return nil
	}

	isBase := locale == baseLocale
	if err := upsertLocale(ctx, store, locale, isBase, payloads, now); err != nil {
		return fmt.Errorf("import %s: %w", locale, err)
	}
	return nil
}

func openContentStoreWithRetry(ctx context.Context, dbPath string, out io.Writer, openStore openContentStoreFunc) (contentStore, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if out == nil {
		out = io.Discard
	}
	if openStore == nil {
		return nil, errors.New("content store opener is required")
	}

	delay := defaultOpenRetryBaseDelay
	for attempt := 1; attempt <= defaultOpenRetryMaxAttempts; attempt++ {
		store, err := openStore(dbPath)
		if err == nil {
			return store, nil
		}
		if !sqliteconn.IsBusyOrLockedError(err) || attempt == defaultOpenRetryMaxAttempts {
			return nil, err
		}
		if _, writeErr := fmt.Fprintf(
			out,
			"content store busy/locked; retrying open (%d/%d)\n",
			attempt+1,
			defaultOpenRetryMaxAttempts,
		); writeErr != nil {
			return nil, writeErr
		}
		if err := sleepContext(ctx, delay); err != nil {
			return nil, err
		}
		delay *= 2
		if delay > defaultOpenRetryMaxDelay {
			delay = defaultOpenRetryMaxDelay
		}
	}

	return nil, fmt.Errorf("open content store retry budget exhausted")
}

func sleepContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func listLocaleDirs(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}

	var locales []string
	for _, entry := range entries {
		if entry.IsDir() {
			locales = append(locales, entry.Name())
		}
	}
	sort.Strings(locales)
	return locales, nil
}

func contains(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

type localePayloads struct {
	Classes              *classPayload
	Subclasses           *subclassPayload
	Heritages            *heritagePayload
	Experiences          *experiencePayload
	Adversaries          *adversaryPayload
	Beastforms           *beastformPayload
	CompanionExperiences *companionExperiencePayload
	LootEntries          *lootEntryPayload
	DamageTypes          *damageTypePayload
	Domains              *domainPayload
	DomainCards          *domainCardPayload
	Weapons              *weaponPayload
	Armor                *armorPayload
	Items                *itemPayload
	Environments         *environmentPayload
}

func readLocalePayloads(dir string) (localePayloads, error) {
	var payloads localePayloads
	var err error
	payloads.Classes, err = readJSON[classPayload](dir, "classes.json")
	if err != nil {
		return payloads, err
	}
	payloads.Subclasses, err = readJSON[subclassPayload](dir, "subclasses.json")
	if err != nil {
		return payloads, err
	}
	payloads.Heritages, err = readJSON[heritagePayload](dir, "heritages.json")
	if err != nil {
		return payloads, err
	}
	payloads.Experiences, err = readJSON[experiencePayload](dir, "experiences.json")
	if err != nil {
		return payloads, err
	}
	payloads.Adversaries, err = readJSON[adversaryPayload](dir, "adversaries.json")
	if err != nil {
		return payloads, err
	}
	payloads.Beastforms, err = readJSON[beastformPayload](dir, "beastforms.json")
	if err != nil {
		return payloads, err
	}
	payloads.CompanionExperiences, err = readJSON[companionExperiencePayload](dir, "companion_experiences.json")
	if err != nil {
		return payloads, err
	}
	payloads.LootEntries, err = readJSON[lootEntryPayload](dir, "loot_entries.json")
	if err != nil {
		return payloads, err
	}
	payloads.DamageTypes, err = readJSON[damageTypePayload](dir, "damage_types.json")
	if err != nil {
		return payloads, err
	}
	payloads.Domains, err = readJSON[domainPayload](dir, "domains.json")
	if err != nil {
		return payloads, err
	}
	payloads.DomainCards, err = readJSON[domainCardPayload](dir, "domain_cards.json")
	if err != nil {
		return payloads, err
	}
	payloads.Weapons, err = readJSON[weaponPayload](dir, "weapons.json")
	if err != nil {
		return payloads, err
	}
	payloads.Armor, err = readJSON[armorPayload](dir, "armor.json")
	if err != nil {
		return payloads, err
	}
	payloads.Items, err = readJSON[itemPayload](dir, "items.json")
	if err != nil {
		return payloads, err
	}
	payloads.Environments, err = readJSON[environmentPayload](dir, "environments.json")
	if err != nil {
		return payloads, err
	}

	return payloads, nil
}

func readJSON[T any](dir string, name string) (*T, error) {
	path := filepath.Join(dir, name)
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, fmt.Errorf("decode %s: %w", name, err)
	}
	return &value, nil
}

func validateLocalePayloads(locale string, payloads localePayloads) error {
	validate := func(systemID, systemVersion, source, payloadLocale string) error {
		if systemID != defaultSystemID {
			return fmt.Errorf("unsupported system id %s", systemID)
		}
		if systemVersion != defaultSystemVer {
			return fmt.Errorf("unsupported system version %s", systemVersion)
		}
		if strings.TrimSpace(source) == "" {
			return fmt.Errorf("source is required")
		}
		if payloadLocale != locale {
			return fmt.Errorf("locale mismatch: %s", payloadLocale)
		}
		return nil
	}

	if payloads.Classes != nil {
		if err := validate(payloads.Classes.SystemID, payloads.Classes.SystemVersion, payloads.Classes.Source, payloads.Classes.Locale); err != nil {
			return err
		}
	}
	if payloads.Subclasses != nil {
		if err := validate(payloads.Subclasses.SystemID, payloads.Subclasses.SystemVersion, payloads.Subclasses.Source, payloads.Subclasses.Locale); err != nil {
			return err
		}
	}
	if payloads.Heritages != nil {
		if err := validate(payloads.Heritages.SystemID, payloads.Heritages.SystemVersion, payloads.Heritages.Source, payloads.Heritages.Locale); err != nil {
			return err
		}
	}
	if payloads.Experiences != nil {
		if err := validate(payloads.Experiences.SystemID, payloads.Experiences.SystemVersion, payloads.Experiences.Source, payloads.Experiences.Locale); err != nil {
			return err
		}
	}
	if payloads.Adversaries != nil {
		if err := validate(payloads.Adversaries.SystemID, payloads.Adversaries.SystemVersion, payloads.Adversaries.Source, payloads.Adversaries.Locale); err != nil {
			return err
		}
	}
	if payloads.Beastforms != nil {
		if err := validate(payloads.Beastforms.SystemID, payloads.Beastforms.SystemVersion, payloads.Beastforms.Source, payloads.Beastforms.Locale); err != nil {
			return err
		}
	}
	if payloads.CompanionExperiences != nil {
		if err := validate(payloads.CompanionExperiences.SystemID, payloads.CompanionExperiences.SystemVersion, payloads.CompanionExperiences.Source, payloads.CompanionExperiences.Locale); err != nil {
			return err
		}
	}
	if payloads.LootEntries != nil {
		if err := validate(payloads.LootEntries.SystemID, payloads.LootEntries.SystemVersion, payloads.LootEntries.Source, payloads.LootEntries.Locale); err != nil {
			return err
		}
	}
	if payloads.DamageTypes != nil {
		if err := validate(payloads.DamageTypes.SystemID, payloads.DamageTypes.SystemVersion, payloads.DamageTypes.Source, payloads.DamageTypes.Locale); err != nil {
			return err
		}
	}
	if payloads.Domains != nil {
		if err := validate(payloads.Domains.SystemID, payloads.Domains.SystemVersion, payloads.Domains.Source, payloads.Domains.Locale); err != nil {
			return err
		}
	}
	if payloads.DomainCards != nil {
		if err := validate(payloads.DomainCards.SystemID, payloads.DomainCards.SystemVersion, payloads.DomainCards.Source, payloads.DomainCards.Locale); err != nil {
			return err
		}
	}
	if payloads.Weapons != nil {
		if err := validate(payloads.Weapons.SystemID, payloads.Weapons.SystemVersion, payloads.Weapons.Source, payloads.Weapons.Locale); err != nil {
			return err
		}
	}
	if payloads.Armor != nil {
		if err := validate(payloads.Armor.SystemID, payloads.Armor.SystemVersion, payloads.Armor.Source, payloads.Armor.Locale); err != nil {
			return err
		}
	}
	if payloads.Items != nil {
		if err := validate(payloads.Items.SystemID, payloads.Items.SystemVersion, payloads.Items.Source, payloads.Items.Locale); err != nil {
			return err
		}
	}
	if payloads.Environments != nil {
		if err := validate(payloads.Environments.SystemID, payloads.Environments.SystemVersion, payloads.Environments.Source, payloads.Environments.Locale); err != nil {
			return err
		}
	}
	return nil
}

func upsertLocale(ctx context.Context, store contentstore.DaggerheartContentStore, locale string, isBase bool, payloads localePayloads, now time.Time) error {
	if store == nil {
		return fmt.Errorf("content store is required")
	}

	if payloads.Domains != nil {
		if err := upsertDomains(ctx, store, payloads.Domains.Items, locale, isBase, now); err != nil {
			return err
		}
	}
	if payloads.DomainCards != nil {
		if err := upsertDomainCards(ctx, store, payloads.DomainCards.Items, locale, isBase, now); err != nil {
			return err
		}
	}
	if payloads.Classes != nil {
		if err := upsertClasses(ctx, store, payloads.Classes.Items, locale, isBase, now); err != nil {
			return err
		}
	}
	if payloads.Subclasses != nil {
		if err := upsertSubclasses(ctx, store, payloads.Subclasses.Items, locale, isBase, now); err != nil {
			return err
		}
	}
	if payloads.Heritages != nil {
		if err := upsertHeritages(ctx, store, payloads.Heritages.Items, locale, isBase, now); err != nil {
			return err
		}
	}
	if payloads.Experiences != nil {
		if err := upsertExperiences(ctx, store, payloads.Experiences.Items, locale, isBase, now); err != nil {
			return err
		}
	}
	if payloads.Adversaries != nil {
		if err := upsertAdversaries(ctx, store, payloads.Adversaries.Items, locale, isBase, now); err != nil {
			return err
		}
	}
	if payloads.Beastforms != nil {
		if err := upsertBeastforms(ctx, store, payloads.Beastforms.Items, locale, isBase, now); err != nil {
			return err
		}
	}
	if payloads.CompanionExperiences != nil {
		if err := upsertCompanionExperiences(ctx, store, payloads.CompanionExperiences.Items, locale, isBase, now); err != nil {
			return err
		}
	}
	if payloads.LootEntries != nil {
		if err := upsertLootEntries(ctx, store, payloads.LootEntries.Items, locale, isBase, now); err != nil {
			return err
		}
	}
	if payloads.DamageTypes != nil {
		if err := upsertDamageTypes(ctx, store, payloads.DamageTypes.Items, locale, isBase, now); err != nil {
			return err
		}
	}
	if payloads.Weapons != nil {
		if err := upsertWeapons(ctx, store, payloads.Weapons.Items, locale, isBase, now); err != nil {
			return err
		}
	}
	if payloads.Armor != nil {
		if err := upsertArmor(ctx, store, payloads.Armor.Items, locale, isBase, now); err != nil {
			return err
		}
	}
	if payloads.Items != nil {
		if err := upsertItems(ctx, store, payloads.Items.Items, locale, isBase, now); err != nil {
			return err
		}
	}
	if payloads.Environments != nil {
		if err := upsertEnvironments(ctx, store, payloads.Environments.Items, locale, isBase, now); err != nil {
			return err
		}
	}

	return nil
}
