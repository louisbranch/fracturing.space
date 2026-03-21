// Package main checks and ratchets package-level coverage floors.
package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"os"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const usage = `coveragefloors checks and ratchets package-level coverage floors.

Usage:
  go run ./internal/tools/coveragefloors check \
    -profile=coverage.out \
    -floors=docs/reference/coverage-floors.json

  go run ./internal/tools/coveragefloors ratchet \
    -profile=coverage.out \
    -seed=docs/reference/coverage-floors.json \
    -existing=coverage-package-floors.json \
    -out=coverage-package-floors.json
`

type floorFile struct {
	Version   int            `json:"version"`
	AllowDrop float64        `json:"allow_drop"`
	UpdatedAt string         `json:"updated_at,omitempty"`
	Packages  []packageFloor `json:"packages"`
}

type packageFloor struct {
	Package     string  `json:"package"`
	Floor       float64 `json:"floor"`
	Description string  `json:"description,omitempty"`
}

type packageStat struct {
	Statements int
	Covered    int
}

func (s packageStat) Percent() float64 {
	if s.Statements == 0 {
		return 0
	}
	return float64(s.Covered) * 100 / float64(s.Statements)
}

func main() {
	if len(os.Args) < 2 {
		fatalf("%s", usage)
	}
	switch os.Args[1] {
	case "check":
		if err := runCheck(os.Args[2:]); err != nil {
			fatalf("%v", err)
		}
	case "ratchet":
		if err := runRatchet(os.Args[2:]); err != nil {
			fatalf("%v", err)
		}
	default:
		fatalf("unknown subcommand %q\n\n%s", os.Args[1], usage)
	}
}

func runCheck(args []string) error {
	fs := flag.NewFlagSet("check", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profilePath := fs.String("profile", "", "coverage profile path")
	floorsPath := fs.String("floors", "", "coverage floors JSON path")
	seedPath := fs.String("seed", "", "seed floors JSON path (caps ratcheted floors for intentional reductions)")
	excludePattern := fs.String("exclude", "", "regex to skip floor entries for excluded packages")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if strings.TrimSpace(*profilePath) == "" || strings.TrimSpace(*floorsPath) == "" {
		return errors.New("check requires -profile and -floors")
	}

	var excludeRe *regexp.Regexp
	if p := strings.TrimSpace(*excludePattern); p != "" {
		var err error
		excludeRe, err = regexp.Compile(p)
		if err != nil {
			return fmt.Errorf("compile -exclude regex: %w", err)
		}
	}

	floors, err := loadFloors(*floorsPath)
	if err != nil {
		return fmt.Errorf("load floors %s: %w", *floorsPath, err)
	}

	// When -seed is provided, cap each ratcheted floor at the seed value.
	// This allows engineers to intentionally lower a floor in the seed file
	// (e.g. after a structural extraction) and have the check respect it.
	seedCap := map[string]float64{}
	if p := strings.TrimSpace(*seedPath); p != "" {
		seed, seedErr := tryLoadFloors(p)
		if seedErr != nil {
			return fmt.Errorf("load seed floors %s: %w", p, seedErr)
		}
		for _, pkg := range seed.Packages {
			seedCap[pkg.Package] = pkg.Floor
		}
	}

	stats, err := parseCoverageProfile(*profilePath)
	if err != nil {
		return fmt.Errorf("parse coverage profile %s: %w", *profilePath, err)
	}

	sort.Slice(floors.Packages, func(i, j int) bool {
		return floors.Packages[i].Package < floors.Packages[j].Package
	})

	var failed bool
	fmt.Println("Package coverage floors:")
	fmt.Println("package,current,floor,threshold,status")
	for _, pkg := range floors.Packages {
		effectiveFloor := pkg.Floor
		if cap, ok := seedCap[pkg.Package]; ok && cap < effectiveFloor {
			effectiveFloor = cap
		}

		stat, ok := stats[pkg.Package]
		if !ok {
			// When seed is provided and the package is absent from seed,
			// treat it as intentionally removed (e.g. renamed package).
			if len(seedCap) > 0 {
				if _, inSeed := seedCap[pkg.Package]; !inSeed {
					fmt.Printf("%s,removed,%.1f,%.1f,SKIP\n", pkg.Package, effectiveFloor, effectiveFloor-floors.AllowDrop)
					continue
				}
			}
			if excludeRe != nil && excludeRe.MatchString(pkg.Package) {
				fmt.Printf("%s,excluded,%.1f,%.1f,SKIP\n", pkg.Package, effectiveFloor, effectiveFloor-floors.AllowDrop)
				continue
			}
			failed = true
			fmt.Printf("%s,missing,%.1f,%.1f,FAIL\n", pkg.Package, effectiveFloor, effectiveFloor-floors.AllowDrop)
			continue
		}
		current := round1(stat.Percent())
		threshold := round1(effectiveFloor - floors.AllowDrop)
		status := "OK"
		if current+1e-9 < threshold {
			failed = true
			status = "FAIL"
		}
		fmt.Printf("%s,%.1f,%.1f,%.1f,%s\n", pkg.Package, current, effectiveFloor, threshold, status)
	}
	if failed {
		return fmt.Errorf("package coverage floor regression detected (allow_drop=%.1f)", floors.AllowDrop)
	}
	fmt.Printf("package coverage floors passed (allow_drop=%.1f)\n", floors.AllowDrop)
	return nil
}

func runRatchet(args []string) error {
	fs := flag.NewFlagSet("ratchet", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	profilePath := fs.String("profile", "", "coverage profile path")
	seedPath := fs.String("seed", "", "seed coverage floors JSON path")
	existingPath := fs.String("existing", "", "existing ratcheted floors JSON path (optional)")
	outPath := fs.String("out", "", "output floors JSON path")
	if err := fs.Parse(args); err != nil {
		return fmt.Errorf("parse flags: %w", err)
	}
	if strings.TrimSpace(*profilePath) == "" || strings.TrimSpace(*seedPath) == "" || strings.TrimSpace(*outPath) == "" {
		return errors.New("ratchet requires -profile, -seed, and -out")
	}

	seed, err := loadFloors(*seedPath)
	if err != nil {
		return fmt.Errorf("load seed floors %s: %w", *seedPath, err)
	}
	stats, err := parseCoverageProfile(*profilePath)
	if err != nil {
		return fmt.Errorf("parse coverage profile %s: %w", *profilePath, err)
	}

	existingByPkg := map[string]packageFloor{}
	if p := strings.TrimSpace(*existingPath); p != "" {
		existing, err := tryLoadFloors(p)
		if err != nil {
			return fmt.Errorf("load existing floors %s: %w", p, err)
		}
		for _, pkg := range existing.Packages {
			existingByPkg[pkg.Package] = pkg
		}
	}

	out := floorFile{
		Version:   seed.Version,
		AllowDrop: seed.AllowDrop,
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Packages:  make([]packageFloor, 0, len(seed.Packages)),
	}
	for _, pkg := range seed.Packages {
		if existing, ok := existingByPkg[pkg.Package]; ok && existing.Floor > pkg.Floor {
			pkg.Floor = existing.Floor
		}
		if stat, ok := stats[pkg.Package]; ok {
			current := round1(stat.Percent())
			if current > pkg.Floor {
				pkg.Floor = current
			}
		}
		pkg.Floor = round1(pkg.Floor)
		out.Packages = append(out.Packages, pkg)
	}
	sort.Slice(out.Packages, func(i, j int) bool {
		return out.Packages[i].Package < out.Packages[j].Package
	})

	if err := writeJSON(*outPath, out); err != nil {
		return fmt.Errorf("write output %s: %w", *outPath, err)
	}
	return nil
}

func parseCoverageProfile(profilePath string) (map[string]packageStat, error) {
	file, err := os.Open(profilePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	stats := map[string]packageStat{}
	scanner := bufio.NewScanner(file)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if lineNum == 1 {
			if !strings.HasPrefix(line, "mode:") {
				return nil, fmt.Errorf("line 1: expected mode header")
			}
			continue
		}
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 3 {
			return nil, fmt.Errorf("line %d: expected 3 fields, got %d", lineNum, len(fields))
		}
		fileAndRange := fields[0]
		statements, err := strconv.Atoi(fields[1])
		if err != nil {
			return nil, fmt.Errorf("line %d: parse statements: %w", lineNum, err)
		}
		count, err := strconv.Atoi(fields[2])
		if err != nil {
			return nil, fmt.Errorf("line %d: parse count: %w", lineNum, err)
		}
		filePath := fileAndRange
		if idx := strings.Index(fileAndRange, ":"); idx >= 0 {
			filePath = fileAndRange[:idx]
		}
		pkg := path.Dir(filePath)
		stat := stats[pkg]
		stat.Statements += statements
		if count > 0 {
			stat.Covered += statements
		}
		stats[pkg] = stat
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return stats, nil
}

func loadFloors(filePath string) (floorFile, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return floorFile{}, err
	}
	defer file.Close()
	var floors floorFile
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&floors); err != nil {
		return floorFile{}, err
	}
	if floors.Version == 0 {
		floors.Version = 1
	}
	if floors.AllowDrop < 0 {
		return floorFile{}, fmt.Errorf("allow_drop must be non-negative")
	}
	if len(floors.Packages) == 0 {
		return floorFile{}, fmt.Errorf("packages must not be empty")
	}
	for _, pkg := range floors.Packages {
		if strings.TrimSpace(pkg.Package) == "" {
			return floorFile{}, fmt.Errorf("package must not be empty")
		}
		if pkg.Floor < 0 || pkg.Floor > 100 {
			return floorFile{}, fmt.Errorf("package floor must be in range 0..100: %s", pkg.Package)
		}
	}
	return floors, nil
}

func tryLoadFloors(filePath string) (floorFile, error) {
	if strings.TrimSpace(filePath) == "" {
		return floorFile{}, nil
	}
	info, err := os.Stat(filePath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return floorFile{}, nil
		}
		return floorFile{}, err
	}
	if info.IsDir() {
		return floorFile{}, fmt.Errorf("%s is a directory", filePath)
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return floorFile{}, err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return floorFile{}, nil
	}
	return loadFloors(filePath)
}

func writeJSON(filePath string, v any) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(filePath, data, 0o644)
}

func round1(v float64) float64 {
	return math.Round(v*10) / 10
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
