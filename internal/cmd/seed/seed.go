// Package seed parses seed command flags and executes fixture / generation workflows.
package seed

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"time"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	campaignv1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	entrypoint "github.com/louisbranch/fracturing.space/internal/platform/cmd"
	"github.com/louisbranch/fracturing.space/internal/platform/discovery"
	grpcmeta "github.com/louisbranch/fracturing.space/internal/services/game/api/grpc/metadata"
	"github.com/louisbranch/fracturing.space/internal/tools/seed"
	"github.com/louisbranch/fracturing.space/internal/tools/seed/generator"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

// Config holds seed command configuration.
type Config struct {
	SeedConfig           seed.Config
	Timeout              time.Duration
	List                 bool
	Generate             bool
	SeedCampaignListings bool
	ListingAddr          string
	Preset               generator.Preset
	Seed                 int64
	Campaigns            int
}

// seedEnv holds env-tagged fields for the seed command.
type seedEnv struct {
	GameAddr    string        `env:"FRACTURING_SPACE_GAME_ADDR"`
	AuthAddr    string        `env:"FRACTURING_SPACE_AUTH_ADDR"`
	ListingAddr string        `env:"FRACTURING_SPACE_LISTING_ADDR"`
	Timeout     time.Duration `env:"FRACTURING_SPACE_SEED_TIMEOUT" envDefault:"10m"`
}

// ParseConfig parses environment and flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	var se seedEnv
	if err := entrypoint.ParseConfig(&se); err != nil {
		return Config{}, err
	}

	seedCfg := seed.DefaultConfig()
	seedCfg.GRPCAddr = discovery.OrDefaultGRPCAddr(se.GameAddr, discovery.ServiceGame)
	seedCfg.AuthAddr = discovery.OrDefaultGRPCAddr(se.AuthAddr, discovery.ServiceAuth)
	listingAddr := discovery.OrDefaultGRPCAddr(se.ListingAddr, discovery.ServiceListing)
	timeout := se.Timeout
	var list bool
	var generate bool
	var seedCampaignListings bool
	var preset string
	var seedVal int64
	var campaigns int

	fs.StringVar(&seedCfg.GRPCAddr, "grpc-addr", seedCfg.GRPCAddr, "game server address")
	fs.StringVar(&seedCfg.AuthAddr, "auth-addr", seedCfg.AuthAddr, "auth server address")
	fs.StringVar(&listingAddr, "listing-addr", listingAddr, "listing server address")
	fs.DurationVar(&timeout, "timeout", timeout, "overall timeout")
	fs.StringVar(&seedCfg.Scenario, "scenario", "", "run specific scenario (default: all)")
	fs.BoolVar(&seedCfg.Verbose, "v", false, "verbose output")
	fs.BoolVar(&list, "list", false, "list available scenarios")
	fs.BoolVar(&generate, "generate", false, "use dynamic generation instead of fixtures")
	fs.BoolVar(&seedCampaignListings, "seed-campaign-listings", false, "seed starter campaign listings into listing service")
	fs.StringVar(&preset, "preset", string(generator.PresetDemo), "generation preset (demo, variety, session-heavy, stress-test)")
	fs.Int64Var(&seedVal, "seed", 0, "random seed for reproducibility (0 = random)")
	fs.IntVar(&campaigns, "campaigns", 0, "number of campaigns to generate (0 = use preset default)")
	if err := entrypoint.ParseArgs(fs, args); err != nil {
		return Config{}, err
	}

	root, err := repoRoot()
	if err != nil {
		return Config{}, err
	}
	seedCfg.RepoRoot = root

	return Config{
		SeedConfig:           seedCfg,
		Timeout:              timeout,
		List:                 list,
		Generate:             generate,
		SeedCampaignListings: seedCampaignListings,
		ListingAddr:          listingAddr,
		Preset:               generator.Preset(preset),
		Seed:                 seedVal,
		Campaigns:            campaigns,
	}, nil
}

// Run executes the seed command across dynamic generation or fixture replay.
func Run(ctx context.Context, cfg Config, out io.Writer, errOut io.Writer) error {
	return entrypoint.RunWithTelemetry(ctx, entrypoint.ServiceSeed, func(runCtx context.Context) error {
		if out == nil {
			out = io.Discard
		}
		if errOut == nil {
			errOut = io.Discard
		}

		if cfg.List {
			scenarios, err := seed.ListScenarios(cfg.SeedConfig)
			if err != nil {
				return err
			}
			fmt.Fprintln(out, "Available scenarios:")
			for _, name := range scenarios {
				fmt.Fprintf(out, "  %s\n", name)
			}
			fmt.Fprintln(out, "\nAvailable presets (for -generate):")
			fmt.Fprintln(out, "  demo         - Rich single campaign with full party")
			fmt.Fprintln(out, "  variety      - 8 campaigns across all statuses/modes")
			fmt.Fprintln(out, "  session-heavy - Few campaigns with many sessions")
			fmt.Fprintln(out, "  stress-test  - 50 minimal campaigns")
			return nil
		}

		beforeCampaignIDs := []string{}
		if cfg.SeedCampaignListings {
			var err error
			beforeCampaignIDs, err = listCampaignIDsByAddr(runCtx, cfg.SeedConfig.GRPCAddr)
			if err != nil {
				return err
			}
		}

		if cfg.Generate {
			if err := validatePreset(cfg.Preset); err != nil {
				return err
			}
			genCfg := generator.Config{
				GRPCAddr:  cfg.SeedConfig.GRPCAddr,
				AuthAddr:  cfg.SeedConfig.AuthAddr,
				Preset:    cfg.Preset,
				Seed:      cfg.Seed,
				Campaigns: cfg.Campaigns,
				Verbose:   cfg.SeedConfig.Verbose,
			}
			gen, err := generator.New(runCtx, genCfg)
			if err != nil {
				return err
			}
			defer gen.Close()

			if err := gen.Run(runCtx); err != nil {
				return err
			}
		} else {
			if err := seed.Run(runCtx, cfg.SeedConfig); err != nil {
				return err
			}
		}

		if cfg.SeedCampaignListings {
			afterCampaignIDs, err := listCampaignIDsByAddr(runCtx, cfg.SeedConfig.GRPCAddr)
			if err != nil {
				return err
			}
			listings, err := prepareStarterCampaignListings(beforeCampaignIDs, afterCampaignIDs)
			if err != nil {
				return err
			}
			if err := seedStarterCampaignListingsByAddr(runCtx, cfg.ListingAddr, listings, out); err != nil {
				return err
			}
		}
		return nil
	})
}

type campaignListingCreator interface {
	CreateCampaignListing(ctx context.Context, in *listingv1.CreateCampaignListingRequest, opts ...grpc.CallOption) (*listingv1.CreateCampaignListingResponse, error)
}

type campaignLister interface {
	ListCampaigns(ctx context.Context, in *campaignv1.ListCampaignsRequest, opts ...grpc.CallOption) (*campaignv1.ListCampaignsResponse, error)
}

type seedCampaignListingResult struct {
	Created int
	Skipped int
}

type starterCampaignListingTemplate struct {
	Title                      string
	Description                string
	RecommendedParticipantsMin int32
	RecommendedParticipantsMax int32
	DifficultyTier             listingv1.CampaignDifficultyTier
	ExpectedDurationLabel      string
	System                     commonv1.GameSystem
}

func listCampaignIDsByAddr(ctx context.Context, addr string) ([]string, error) {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return nil, fmt.Errorf("game server address is required")
	}
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		return nil, fmt.Errorf("connect game server: %w", err)
	}
	defer func() { _ = conn.Close() }()

	return listCampaignIDs(ctx, campaignv1.NewCampaignServiceClient(conn))
}

func listCampaignIDs(ctx context.Context, client campaignLister) ([]string, error) {
	if client == nil {
		return nil, fmt.Errorf("campaign client is required")
	}

	adminCtx := metadata.NewOutgoingContext(ctx, metadata.Pairs(
		grpcmeta.PlatformRoleHeader, grpcmeta.PlatformRoleAdmin,
		grpcmeta.AuthzOverrideReasonHeader, "seed_listing_population",
	))

	pageToken := ""
	uniqueIDs := make(map[string]struct{})
	for {
		resp, err := client.ListCampaigns(adminCtx, &campaignv1.ListCampaignsRequest{
			PageSize:  10,
			PageToken: pageToken,
		})
		if err != nil {
			return nil, fmt.Errorf("list campaigns: %w", err)
		}
		for _, campaign := range resp.GetCampaigns() {
			campaignID := strings.TrimSpace(campaign.GetId())
			if campaignID == "" {
				continue
			}
			uniqueIDs[campaignID] = struct{}{}
		}
		if resp.GetNextPageToken() == "" {
			break
		}
		pageToken = resp.GetNextPageToken()
	}

	campaignIDs := slices.Collect(maps.Keys(uniqueIDs))
	sort.Strings(campaignIDs)
	return campaignIDs, nil
}

func seedStarterCampaignListingsByAddr(ctx context.Context, addr string, listings []*listingv1.CreateCampaignListingRequest, out io.Writer) error {
	addr = strings.TrimSpace(addr)
	if addr == "" {
		return fmt.Errorf("listing server address is required")
	}
	if len(listings) == 0 {
		return fmt.Errorf("starter listings are required")
	}
	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		return fmt.Errorf("connect listing server: %w", err)
	}
	defer func() { _ = conn.Close() }()

	client := listingv1.NewCampaignListingServiceClient(conn)
	result, err := seedStarterCampaignListings(ctx, client, listings)
	if err != nil {
		return err
	}
	if out != nil {
		total := result.Created + result.Skipped
		fmt.Fprintf(out, "Seeded starter campaign listing(s): created=%d skipped=%d total=%d\n", result.Created, result.Skipped, total)
	}
	return nil
}

func seedStarterCampaignListings(ctx context.Context, client campaignListingCreator, listings []*listingv1.CreateCampaignListingRequest) (seedCampaignListingResult, error) {
	if client == nil {
		return seedCampaignListingResult{}, fmt.Errorf("campaign listing client is required")
	}
	result := seedCampaignListingResult{}
	for _, listing := range listings {
		if listing == nil {
			continue
		}
		campaignID := strings.TrimSpace(listing.GetCampaignId())
		if campaignID == "" {
			return seedCampaignListingResult{}, fmt.Errorf("starter listing campaign id is required")
		}
		if _, err := client.CreateCampaignListing(ctx, listing); err != nil {
			if status.Code(err) == codes.AlreadyExists {
				result.Skipped++
				continue
			}
			return seedCampaignListingResult{}, fmt.Errorf("create campaign listing %q: %w", campaignID, err)
		}
		result.Created++
	}
	return result, nil
}

func campaignIDsCreatedSince(before, after []string) []string {
	beforeSet := make(map[string]struct{}, len(before))
	for _, campaignID := range before {
		campaignID = strings.TrimSpace(campaignID)
		if campaignID == "" {
			continue
		}
		beforeSet[campaignID] = struct{}{}
	}

	createdSet := make(map[string]struct{}, len(after))
	for _, campaignID := range after {
		campaignID = strings.TrimSpace(campaignID)
		if campaignID == "" {
			continue
		}
		if _, exists := beforeSet[campaignID]; exists {
			continue
		}
		createdSet[campaignID] = struct{}{}
	}

	created := slices.Collect(maps.Keys(createdSet))
	sort.Strings(created)
	return created
}

func prepareStarterCampaignListings(beforeCampaignIDs, afterCampaignIDs []string) ([]*listingv1.CreateCampaignListingRequest, error) {
	createdCampaignIDs := campaignIDsCreatedSince(beforeCampaignIDs, afterCampaignIDs)
	if len(createdCampaignIDs) == 0 {
		return nil, fmt.Errorf("no newly created campaigns found for starter listings")
	}
	return starterCampaignListingsForCampaignIDs(createdCampaignIDs, defaultStarterCampaignListingTemplates())
}

func starterCampaignListingsForCampaignIDs(campaignIDs []string, templates []starterCampaignListingTemplate) ([]*listingv1.CreateCampaignListingRequest, error) {
	if len(campaignIDs) == 0 {
		return nil, fmt.Errorf("campaign IDs are required")
	}
	if len(templates) == 0 {
		return nil, fmt.Errorf("starter listing templates are required")
	}
	if len(campaignIDs) > len(templates) {
		return nil, fmt.Errorf("not enough starter listing templates: have %d templates for %d campaigns", len(templates), len(campaignIDs))
	}

	limit := len(campaignIDs)
	if limit <= 0 {
		return nil, fmt.Errorf("starter listing template mapping is empty")
	}

	listings := make([]*listingv1.CreateCampaignListingRequest, 0, limit)
	for i := 0; i < limit; i++ {
		template := templates[i]
		campaignID := strings.TrimSpace(campaignIDs[i])
		if campaignID == "" {
			return nil, fmt.Errorf("starter listing campaign id is required")
		}
		listings = append(listings, &listingv1.CreateCampaignListingRequest{
			CampaignId:                 campaignID,
			Title:                      template.Title,
			Description:                template.Description,
			RecommendedParticipantsMin: template.RecommendedParticipantsMin,
			RecommendedParticipantsMax: template.RecommendedParticipantsMax,
			DifficultyTier:             template.DifficultyTier,
			ExpectedDurationLabel:      template.ExpectedDurationLabel,
			System:                     template.System,
		})
	}
	return listings, nil
}

func defaultStarterCampaignListingTemplates() []starterCampaignListingTemplate {
	return []starterCampaignListingTemplate{
		{
			Title:                      "Shadow Over Sunfall",
			Description:                "Small-town mystery with escalating supernatural pressure and clear session beats for first-time groups.",
			RecommendedParticipantsMin: 3,
			RecommendedParticipantsMax: 5,
			DifficultyTier:             listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_BEGINNER,
			ExpectedDurationLabel:      "2-3 sessions",
			System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		},
		{
			Title:                      "Fall of Cinder Vale",
			Description:                "Tactical frontier-defense arc with branching objectives and room for player-led plans.",
			RecommendedParticipantsMin: 4,
			RecommendedParticipantsMax: 6,
			DifficultyTier:             listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_INTERMEDIATE,
			ExpectedDurationLabel:      "4-6 sessions",
			System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		},
		{
			Title:                      "Shards of the Astral Lens",
			Description:                "Long-form relic hunt with travel, faction politics, and escalating encounter complexity.",
			RecommendedParticipantsMin: 4,
			RecommendedParticipantsMax: 6,
			DifficultyTier:             listingv1.CampaignDifficultyTier_CAMPAIGN_DIFFICULTY_TIER_ADVANCED,
			ExpectedDurationLabel:      "8+ sessions",
			System:                     commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART,
		},
	}
}

func validatePreset(preset generator.Preset) error {
	validPresets := []generator.Preset{
		generator.PresetDemo,
		generator.PresetVariety,
		generator.PresetSessionHeavy,
		generator.PresetStressTest,
	}
	for _, p := range validPresets {
		if preset == p {
			return nil
		}
	}
	return fmt.Errorf("unknown preset %q (valid presets: demo, variety, session-heavy, stress-test)", preset)
}

func repoRoot() (string, error) {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "", errors.New("failed to resolve runtime caller")
	}

	dir := filepath.Dir(filename)
	for {
		candidate := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(candidate); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("go.mod not found from %s", filename)
}
