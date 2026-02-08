// Package generator provides dynamic scenario generation for seeding
// the development database with diverse test data.
package generator

import (
	"context"
	"fmt"
	"math/rand"
	"os"

	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/campaign/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	"github.com/louisbranch/fracturing.space/internal/seed/worldbuilder"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// Config holds configuration for the generator.
type Config struct {
	GRPCAddr  string
	Preset    Preset
	Seed      int64
	Campaigns int // Override preset's campaign count (0 = use preset default)
	Verbose   bool
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		GRPCAddr: "localhost:8080",
		Preset:   PresetDemo,
		Seed:     0,
		Verbose:  false,
	}
}

// Generator orchestrates dynamic scenario generation.
type Generator struct {
	config Config
	rng    *rand.Rand
	wb     *worldbuilder.WorldBuilder
	conn   *grpc.ClientConn

	// gRPC service clients (campaign/v1)
	campaigns    statev1.CampaignServiceClient
	participants statev1.ParticipantServiceClient
	characters   statev1.CharacterServiceClient
	sessions     statev1.SessionServiceClient
	events       statev1.EventServiceClient
}

// New creates a new Generator with the given configuration.
func New(ctx context.Context, cfg Config) (*Generator, error) {
	rng := NewSeededRNG(cfg.Seed, cfg.Verbose)

	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "Connecting to gRPC server at %s (waiting for server to be ready)...\n", cfg.GRPCAddr)
	}

	conn, err := grpc.NewClient(
		cfg.GRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to gRPC: %w", err)
	}

	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "Connected to gRPC server\n")
	}

	return &Generator{
		config:       cfg,
		rng:          rng,
		wb:           worldbuilder.New(rng),
		conn:         conn,
		campaigns:    statev1.NewCampaignServiceClient(conn),
		participants: statev1.NewParticipantServiceClient(conn),
		characters:   statev1.NewCharacterServiceClient(conn),
		sessions:     statev1.NewSessionServiceClient(conn),
		events:       statev1.NewEventServiceClient(conn),
	}, nil
}

// Close releases resources held by the generator.
func (g *Generator) Close() error {
	if g.conn != nil {
		return g.conn.Close()
	}
	return nil
}

// Run executes the generation based on the configured preset.
func (g *Generator) Run(ctx context.Context) error {
	presetCfg := GetPresetConfig(g.config.Preset)

	// Override campaign count if specified
	numCampaigns := presetCfg.Campaigns
	if g.config.Campaigns > 0 {
		numCampaigns = g.config.Campaigns
	}

	if g.config.Verbose {
		fmt.Fprintf(os.Stderr, "Running preset %q: %d campaign(s)\n",
			g.config.Preset, numCampaigns)
	}

	for i := 0; i < numCampaigns; i++ {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := g.generateCampaign(ctx, i, presetCfg); err != nil {
			return fmt.Errorf("generate campaign %d: %w", i+1, err)
		}
	}

	if g.config.Verbose {
		fmt.Fprintf(os.Stderr, "Generation complete: %d campaign(s) created\n",
			numCampaigns)
	}
	return nil
}

// generateCampaign creates a single campaign with all its entities.
func (g *Generator) generateCampaign(ctx context.Context, index int, cfg PresetConfig) error {
	// Determine campaign parameters
	gmMode := g.pickGmMode(cfg.VaryGmModes, index)

	// Create the campaign
	campaign, err := g.createCampaign(ctx, gmMode)
	if err != nil {
		return fmt.Errorf("create campaign: %w", err)
	}

	if g.config.Verbose {
		fmt.Fprintf(os.Stderr, "  Created campaign: %s (%s)\n", campaign.Name, campaign.Id)
	}

	// Create participants
	numParticipants := g.randomRange(cfg.ParticipantsMin, cfg.ParticipantsMax)
	participants, err := g.createParticipants(ctx, campaign.Id, numParticipants)
	if err != nil {
		return fmt.Errorf("create participants: %w", err)
	}

	if g.config.Verbose {
		fmt.Fprintf(os.Stderr, "    Created %d participant(s)\n", len(participants))
	}

	// Create characters
	numCharacters := g.randomRange(cfg.CharactersMin, cfg.CharactersMax)
	characters, err := g.createCharacters(ctx, campaign.Id, numCharacters, participants)
	if err != nil {
		return fmt.Errorf("create characters: %w", err)
	}

	if g.config.Verbose {
		fmt.Fprintf(os.Stderr, "    Created %d character(s)\n", len(characters))
	}

	// Create sessions
	numSessions := g.randomRange(cfg.SessionsMin, cfg.SessionsMax)

	// Campaigns that will be COMPLETED or ARCHIVED need at least 1 session
	// to transition from DRAFT -> ACTIVE first
	if cfg.VaryStatuses && numSessions == 0 {
		targetStatus := index % 4
		if targetStatus >= 1 { // ACTIVE, COMPLETED, or ARCHIVED
			numSessions = 1
		}
	}

	if numSessions > 0 {
		if err := g.createSessions(ctx, campaign.Id, numSessions, cfg, characters); err != nil {
			return fmt.Errorf("create sessions: %w", err)
		}
		if g.config.Verbose {
			fmt.Fprintf(os.Stderr, "    Created %d session(s)\n", numSessions)
		}
	}

	// Transition campaign status if needed
	if cfg.VaryStatuses {
		if err := g.transitionCampaignStatus(ctx, campaign.Id, index); err != nil {
			return fmt.Errorf("transition campaign status: %w", err)
		}
	}

	return nil
}

// randomRange returns a random number in [min, max].
func (g *Generator) randomRange(min, max int) int {
	if min >= max {
		return min
	}
	return min + g.rng.Intn(max-min+1)
}

// gameSystem returns the game system to use for campaigns.
// Currently only Daggerheart is supported.
func (g *Generator) gameSystem() commonv1.GameSystem {
	return commonv1.GameSystem_GAME_SYSTEM_DAGGERHEART
}
