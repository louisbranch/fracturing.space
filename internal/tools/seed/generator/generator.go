// Package generator provides dynamic scenario generation for seeding
// the development database with diverse test data.
package generator

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strings"

	authv1 "github.com/louisbranch/fracturing.space/api/gen/go/auth/v1"
	commonv1 "github.com/louisbranch/fracturing.space/api/gen/go/common/v1"
	statev1 "github.com/louisbranch/fracturing.space/api/gen/go/game/v1"
	"github.com/louisbranch/fracturing.space/internal/tools/seed/worldbuilder"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// newGenerator constructs a Generator from pre-built dependencies.
// Used by tests to inject fakes without establishing gRPC connections.
func newGenerator(cfg Config, rng *rand.Rand, wb *worldbuilder.WorldBuilder, deps generatorDeps) *Generator {
	return &Generator{
		config:       cfg,
		rng:          rng,
		wb:           wb,
		nameRegistry: newNameRegistry(),
		campaigns:    deps.campaigns,
		participants: deps.participants,
		invites:      deps.invites,
		characters:   deps.characters,
		sessions:     deps.sessions,
		events:       deps.events,
		authClient:   deps.authClient,
	}
}

// generatorDeps bundles the service dependencies for newGenerator.
type generatorDeps struct {
	campaigns    campaignCreator
	participants participantCreator
	invites      inviteManager
	characters   characterCreator
	sessions     sessionManager
	events       eventAppender
	authClient   authProvider
}

// Config holds configuration for the generator.
type Config struct {
	GRPCAddr  string
	AuthAddr  string
	Preset    Preset
	Seed      int64
	Campaigns int // Override preset's campaign count (0 = use preset default)
	Verbose   bool
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() Config {
	return Config{
		GRPCAddr: "localhost:8080",
		AuthAddr: "localhost:8083",
		Preset:   PresetDemo,
		Seed:     0,
		Verbose:  false,
	}
}

// Generator orchestrates dynamic scenario generation.
type Generator struct {
	config       Config
	rng          *rand.Rand
	wb           *worldbuilder.WorldBuilder
	nameRegistry *nameRegistry
	conn         *grpc.ClientConn
	authConn     *grpc.ClientConn

	// Service dependencies (satisfied by gRPC clients in production,
	// fakes in tests).
	campaigns    campaignCreator
	participants participantCreator
	invites      inviteManager
	characters   characterCreator
	sessions     sessionManager
	events       eventAppender
	authClient   authProvider
}

// New creates a new Generator with the given configuration.
func New(ctx context.Context, cfg Config) (*Generator, error) {
	rng := NewSeededRNG(cfg.Seed, cfg.Verbose)

	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "Connecting to game server at %s (waiting for server to be ready)...\n", cfg.GRPCAddr)
	}

	conn, err := grpc.NewClient(
		cfg.GRPCAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to game server: %w", err)
	}

	if cfg.Verbose {
		fmt.Fprintf(os.Stderr, "Connected to game server\n")
	}

	authAddr := strings.TrimSpace(cfg.AuthAddr)
	if authAddr == "" {
		_ = conn.Close()
		return nil, fmt.Errorf("auth server address is required")
	}
	authConn, err := grpc.NewClient(
		authAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.WaitForReady(true)),
	)
	if err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("connect to auth server: %w", err)
	}

	return &Generator{
		config:       cfg,
		rng:          rng,
		wb:           worldbuilder.New(rng),
		nameRegistry: newNameRegistry(),
		conn:         conn,
		authConn:     authConn,
		campaigns:    statev1.NewCampaignServiceClient(conn),
		participants: statev1.NewParticipantServiceClient(conn),
		invites:      statev1.NewInviteServiceClient(conn),
		characters:   statev1.NewCharacterServiceClient(conn),
		sessions:     statev1.NewSessionServiceClient(conn),
		events:       statev1.NewEventServiceClient(conn),
		authClient:   authv1.NewAuthServiceClient(authConn),
	}, nil
}

// Close releases resources held by the generator.
func (g *Generator) Close() error {
	if g.conn != nil {
		if err := g.conn.Close(); err != nil {
			return err
		}
	}
	if g.authConn != nil {
		return g.authConn.Close()
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
	campaign, ownerParticipantID, err := g.createCampaign(ctx, gmMode)
	if err != nil {
		return fmt.Errorf("create campaign: %w", err)
	}

	if g.config.Verbose {
		fmt.Fprintf(os.Stderr, "  Created campaign: %s (%s)\n", campaign.Name, campaign.Id)
	}

	// Create participants
	numParticipants := g.randomRange(cfg.ParticipantsMin, cfg.ParticipantsMax)
	participants, err := g.createParticipants(ctx, campaign.Id, ownerParticipantID, numParticipants)
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
