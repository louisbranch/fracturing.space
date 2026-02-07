// Package main provides a CLI for seeding the local development database
// with demo data by exercising the full MCP→gRPC stack, or by generating
// dynamic scenarios directly via gRPC.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/louisbranch/fracturing.space/internal/seed"
	"github.com/louisbranch/fracturing.space/internal/seed/generator"
)

func main() {
	// Static fixture flags
	seedCfg := seed.DefaultConfig()
	var list bool

	// Dynamic generation flags
	var generate bool
	var preset string
	var seedVal int64
	var campaigns int

	flag.StringVar(&seedCfg.GRPCAddr, "grpc-addr", seedCfg.GRPCAddr, "gRPC server address")
	flag.StringVar(&seedCfg.Scenario, "scenario", "", "run specific scenario (default: all)")
	flag.BoolVar(&seedCfg.Verbose, "v", false, "verbose output")
	flag.BoolVar(&list, "list", false, "list available scenarios")

	// Generation flags
	flag.BoolVar(&generate, "generate", false, "use dynamic generation instead of fixtures")
	flag.StringVar(&preset, "preset", string(generator.PresetDemo), "generation preset (demo, variety, session-heavy, stress-test)")
	flag.Int64Var(&seedVal, "seed", 0, "random seed for reproducibility (0 = random)")
	flag.IntVar(&campaigns, "campaigns", 0, "number of campaigns to generate (0 = use preset default)")

	flag.Parse()

	seedCfg.RepoRoot = repoRoot()

	if list {
		scenarios, err := seed.ListScenarios(seedCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Available scenarios:")
		for _, name := range scenarios {
			fmt.Printf("  %s\n", name)
		}
		fmt.Println("\nAvailable presets (for -generate):")
		fmt.Println("  demo         - Rich single campaign with full party")
		fmt.Println("  variety      - 8 campaigns across all statuses/modes")
		fmt.Println("  session-heavy - Few campaigns with many sessions")
		fmt.Println("  stress-test  - 50 minimal campaigns")
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	if generate {
		// Validate preset
		validPresets := []generator.Preset{
			generator.PresetDemo,
			generator.PresetVariety,
			generator.PresetSessionHeavy,
			generator.PresetStressTest,
		}
		presetVal := generator.Preset(preset)
		valid := false
		for _, p := range validPresets {
			if presetVal == p {
				valid = true
				break
			}
		}
		if !valid {
			fmt.Fprintf(os.Stderr, "Error: unknown preset %q\n", preset)
			fmt.Fprintf(os.Stderr, "Valid presets: demo, variety, session-heavy, stress-test\n")
			os.Exit(1)
		}

		// Dynamic generation mode
		genCfg := generator.Config{
			GRPCAddr:  seedCfg.GRPCAddr,
			Preset:    presetVal,
			Seed:      seedVal,
			Campaigns: campaigns,
			Verbose:   seedCfg.Verbose,
		}

		gen, err := generator.New(ctx, genCfg)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer gen.Close()

		if err := gen.Run(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	} else {
		// Static fixture mode (original behavior)
		if err := seed.Run(ctx, seedCfg); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

// repoRoot returns the repository root by walking up to go.mod.
func repoRoot() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		fmt.Fprintln(os.Stderr, "failed to resolve runtime caller")
		os.Exit(1)
	}

	dir := filepath.Dir(filename)
	for {
		candidate := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(candidate); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	fmt.Fprintf(os.Stderr, "go.mod not found from %s\n", filename)
	os.Exit(1)
	return ""
}
