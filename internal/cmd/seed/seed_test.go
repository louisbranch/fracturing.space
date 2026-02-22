package seed

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	listingv1 "github.com/louisbranch/fracturing.space/api/gen/go/listing/v1"
	"github.com/louisbranch/fracturing.space/internal/tools/seed/generator"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestValidatePreset(t *testing.T) {
	if err := validatePreset(generator.PresetDemo); err != nil {
		t.Fatalf("expected demo to be valid: %v", err)
	}
	if err := validatePreset("unknown"); err == nil {
		t.Fatal("expected error for unknown preset")
	}
}

func TestParseConfigDefaults(t *testing.T) {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.Preset != generator.PresetDemo {
		t.Fatalf("expected demo preset, got %q", cfg.Preset)
	}
	if cfg.SeedConfig.AuthAddr != "auth:8083" {
		t.Fatalf("expected default auth addr, got %q", cfg.SeedConfig.AuthAddr)
	}
	if cfg.SeedConfig.GRPCAddr != "game:8082" {
		t.Fatalf("expected default game grpc addr, got %q", cfg.SeedConfig.GRPCAddr)
	}
	if cfg.SeedConfig.RepoRoot == "" {
		t.Fatal("expected repo root to be set")
	}
	if _, err := os.Stat(filepath.Join(cfg.SeedConfig.RepoRoot, "go.mod")); err != nil {
		t.Fatalf("expected go.mod in repo root: %v", err)
	}
}

func TestParseConfigReadsServiceAddrEnvOverrides(t *testing.T) {
	t.Setenv("FRACTURING_SPACE_GAME_ADDR", "localhost:18082")
	t.Setenv("FRACTURING_SPACE_AUTH_ADDR", "localhost:18083")

	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, nil)
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if cfg.SeedConfig.GRPCAddr != "localhost:18082" {
		t.Fatalf("expected env game grpc addr, got %q", cfg.SeedConfig.GRPCAddr)
	}
	if cfg.SeedConfig.AuthAddr != "localhost:18083" {
		t.Fatalf("expected env auth grpc addr, got %q", cfg.SeedConfig.AuthAddr)
	}
}

func TestParseConfigListFlag(t *testing.T) {
	fs := flag.NewFlagSet("seed", flag.ContinueOnError)
	cfg, err := ParseConfig(fs, []string{"-list"})
	if err != nil {
		t.Fatalf("parse config: %v", err)
	}
	if !cfg.List {
		t.Fatal("expected list flag to be true")
	}
}

func TestSeedStarterCampaignListings_IdempotentAlreadyExists(t *testing.T) {
	client := &fakeCampaignListingCreator{
		createCampaignListing: func(_ context.Context, in *listingv1.CreateCampaignListingRequest, _ ...grpc.CallOption) (*listingv1.CreateCampaignListingResponse, error) {
			if in.GetCampaignId() == "camp-1" {
				return nil, status.Error(codes.AlreadyExists, "already exists")
			}
			return &listingv1.CreateCampaignListingResponse{
				Listing: &listingv1.CampaignListing{CampaignId: in.GetCampaignId()},
			}, nil
		},
	}
	listings, err := starterCampaignListingsForCampaignIDs([]string{"camp-1", "camp-2", "camp-3"}, defaultStarterCampaignListingTemplates())
	if err != nil {
		t.Fatalf("starterCampaignListingsForCampaignIDs: %v", err)
	}

	result, err := seedStarterCampaignListings(context.Background(), client, listings)
	if err != nil {
		t.Fatalf("seedStarterCampaignListings returned error: %v", err)
	}
	if len(client.createdCampaignIDs) != len(listings) {
		t.Fatalf("created calls = %d, want %d", len(client.createdCampaignIDs), len(listings))
	}
	if result.Created != 2 || result.Skipped != 1 {
		t.Fatalf("result = %+v, want created=2 skipped=1", result)
	}
}

func TestSeedStarterCampaignListings_PropagatesCreateFailure(t *testing.T) {
	client := &fakeCampaignListingCreator{
		createCampaignListing: func(_ context.Context, in *listingv1.CreateCampaignListingRequest, _ ...grpc.CallOption) (*listingv1.CreateCampaignListingResponse, error) {
			if in.GetCampaignId() == "camp-2" {
				return nil, errors.New("listing backend unavailable")
			}
			return &listingv1.CreateCampaignListingResponse{
				Listing: &listingv1.CampaignListing{CampaignId: in.GetCampaignId()},
			}, nil
		},
	}
	listings, err := starterCampaignListingsForCampaignIDs([]string{"camp-1", "camp-2", "camp-3"}, defaultStarterCampaignListingTemplates())
	if err != nil {
		t.Fatalf("starterCampaignListingsForCampaignIDs: %v", err)
	}

	_, err = seedStarterCampaignListings(context.Background(), client, listings)
	if err == nil {
		t.Fatal("expected error")
	}
	if got := err.Error(); got == "" || !containsAll(got, "camp-2", "listing backend unavailable") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestStarterCampaignListingsForCampaignIDs_UsesProvidedCampaignIDs(t *testing.T) {
	campaignIDs := []string{"camp-z", "camp-a", "camp-m"}
	listings, err := starterCampaignListingsForCampaignIDs(campaignIDs, defaultStarterCampaignListingTemplates())
	if err != nil {
		t.Fatalf("starterCampaignListingsForCampaignIDs: %v", err)
	}
	if len(listings) != len(campaignIDs) {
		t.Fatalf("len(listings) = %d, want %d", len(listings), len(campaignIDs))
	}

	gotIDs := make([]string, 0, len(listings))
	for _, listing := range listings {
		gotIDs = append(gotIDs, listing.GetCampaignId())
	}
	if !reflect.DeepEqual(gotIDs, campaignIDs) {
		t.Fatalf("campaign IDs = %v, want %v", gotIDs, campaignIDs)
	}
	if listings[0].GetSystem() != commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART {
		t.Fatalf("system = %v, want %v", listings[0].GetSystem(), commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART)
	}
}

func TestStarterCampaignListingsForCampaignIDs_ReturnsErrorWhenTemplatesInsufficient(t *testing.T) {
	_, err := starterCampaignListingsForCampaignIDs(
		[]string{"camp-1", "camp-2", "camp-3", "camp-4"},
		defaultStarterCampaignListingTemplates(),
	)
	if err == nil {
		t.Fatal("expected error when campaign IDs exceed available templates")
	}
}

func TestPrepareStarterCampaignListings_UsesOnlyNewCampaigns(t *testing.T) {
	before := []string{"camp-1", "camp-2"}
	after := []string{"camp-2", "camp-1", "camp-3", "camp-4"}

	listings, err := prepareStarterCampaignListings(before, after)
	if err != nil {
		t.Fatalf("prepareStarterCampaignListings: %v", err)
	}
	if len(listings) != 2 {
		t.Fatalf("len(listings) = %d, want 2", len(listings))
	}
	gotIDs := []string{listings[0].GetCampaignId(), listings[1].GetCampaignId()}
	if !reflect.DeepEqual(gotIDs, []string{"camp-3", "camp-4"}) {
		t.Fatalf("new campaign IDs = %v, want [camp-3 camp-4]", gotIDs)
	}
}

func TestPrepareStarterCampaignListings_RequiresNewCampaigns(t *testing.T) {
	_, err := prepareStarterCampaignListings([]string{"camp-1"}, []string{"camp-1"})
	if err == nil {
		t.Fatal("expected error when no new campaigns were created")
	}
}

func TestCampaignIDsCreatedSince_ReturnsSortedSetDifference(t *testing.T) {
	before := []string{"camp-3", "camp-1"}
	after := []string{"camp-2", "camp-3", "camp-4", "camp-1", "camp-2"}

	got := campaignIDsCreatedSince(before, after)
	want := []string{"camp-2", "camp-4"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("campaignIDsCreatedSince = %v, want %v", got, want)
	}
	if !slices.IsSorted(got) {
		t.Fatalf("campaignIDsCreatedSince should be sorted, got %v", got)
	}
}

type fakeCampaignListingCreator struct {
	createCampaignListing func(context.Context, *listingv1.CreateCampaignListingRequest, ...grpc.CallOption) (*listingv1.CreateCampaignListingResponse, error)
	createdCampaignIDs    []string
}

func (f *fakeCampaignListingCreator) CreateCampaignListing(ctx context.Context, in *listingv1.CreateCampaignListingRequest, opts ...grpc.CallOption) (*listingv1.CreateCampaignListingResponse, error) {
	f.createdCampaignIDs = append(f.createdCampaignIDs, in.GetCampaignId())
	if f.createCampaignListing != nil {
		return f.createCampaignListing(ctx, in, opts...)
	}
	return nil, fmt.Errorf("CreateCampaignListing: not implemented")
}

func containsAll(value string, substrings ...string) bool {
	for _, needle := range substrings {
		if !strings.Contains(value, needle) {
			return false
		}
	}
	return true
}
