//go:build scenario

package game

import (
	"context"
	"hash/fnv"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	scenariotool "github.com/louisbranch/fracturing.space/internal/tools/scenario"
)

const (
	scenarioLuaRoot           = "internal/test/game/scenarios"
	scenarioSmokeManifestPath = "internal/test/game/scenarios/manifests/smoke.txt"
	scenarioSystemRootSegment = "systems"
	scenarioSystemDaggerheart = "DAGGERHEART"
)

type scenarioPath struct {
	absolute string
	relative string
	subpath  string
}

type scenarioRuntime struct {
	grpcAddr string
	stop     func()
}

func TestScenarioScripts(t *testing.T) {
	paths := scenarioLuaPaths(t)
	parallelism := scenarioParallelism(t)

	runtimes := make([]scenarioRuntime, 0, parallelism)
	for worker := 0; worker < parallelism; worker++ {
		runtimes = append(runtimes, newScenarioRuntime(t))
	}
	t.Cleanup(func() {
		for _, runtime := range runtimes {
			runtime.stop()
		}
	})

	runtimePool := make(chan scenarioRuntime, len(runtimes))
	for _, runtime := range runtimes {
		runtimePool <- runtime
	}

	for _, path := range paths {
		path := path
		t.Run(strings.TrimSuffix(path.subpath, ".lua"), func(t *testing.T) {
			if parallelism > 1 {
				t.Parallel()
			}
			runtime := <-runtimePool
			defer func() {
				runtimePool <- runtime
			}()
			runScenarioScript(t, runtime.grpcAddr, path.absolute)
		})
	}
}

func TestScenarioShardCoverage(t *testing.T) {
	raw := strings.TrimSpace(os.Getenv("SCENARIO_VERIFY_SHARDS_TOTAL"))
	if raw == "" {
		t.Skip("set SCENARIO_VERIFY_SHARDS_TOTAL to run shard coverage check")
	}
	total, err := strconv.Atoi(raw)
	if err != nil || total <= 0 {
		t.Fatalf("invalid SCENARIO_VERIFY_SHARDS_TOTAL %q", raw)
	}

	_, allPaths := discoverScenarioLuaPaths(t)
	seen := make(map[string]int, len(allPaths))
	for index := 0; index < total; index++ {
		for _, path := range allPaths {
			if scenarioShardForPath(path.subpath, total) != index {
				continue
			}
			seen[path.subpath]++
		}
	}
	for _, path := range allPaths {
		count := seen[path.subpath]
		if count == 0 {
			t.Fatalf("scenario %s was not assigned to any shard", path.subpath)
		}
		if count > 1 {
			t.Fatalf("scenario %s was assigned to %d shards", path.subpath, count)
		}
	}
}

func TestScenarioPathSystemAlignment(t *testing.T) {
	_, paths := discoverScenarioLuaPaths(t)
	for _, path := range paths {
		path := path
		t.Run(path.subpath, func(t *testing.T) {
			expectedSystem, err := expectedSystemFromSubpath(path.subpath)
			if err != nil {
				t.Fatalf("derive expected system: %v", err)
			}
			scn, err := scenariotool.LoadScenarioFromFileWithOptions(path.absolute, false)
			if err != nil {
				t.Fatalf("load scenario %s: %v", path.subpath, err)
			}
			campaignSystem := ""
			for _, step := range scn.Steps {
				if step.Kind != "campaign" {
					continue
				}
				if raw, ok := step.Args["system"].(string); ok {
					campaignSystem = strings.ToUpper(strings.TrimSpace(raw))
				}
				break
			}
			if campaignSystem == "" {
				t.Fatalf("scenario %s campaign step must declare system", path.subpath)
			}
			if campaignSystem != expectedSystem {
				t.Fatalf("scenario %s campaign system = %s, want %s", path.subpath, campaignSystem, expectedSystem)
			}
		})
	}
}

func TestScenarioSmokeManifestEntriesResolveByPath(t *testing.T) {
	repo, allPaths := discoverScenarioLuaPaths(t)
	manifestPath := filepath.Join(repo, scenarioSmokeManifestPath)
	entries, err := readScenarioManifestEntries(manifestPath)
	if err != nil {
		t.Fatalf("read smoke manifest: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("smoke manifest %s is empty", manifestPath)
	}

	known := make(map[string]struct{}, len(allPaths))
	for _, path := range allPaths {
		known[path.subpath] = struct{}{}
	}

	seen := map[string]struct{}{}
	for _, entry := range entries {
		if !strings.Contains(entry, "/") {
			t.Fatalf("manifest entry %q must be a path under %s", entry, scenarioLuaRoot)
		}
		if _, ok := known[entry]; !ok {
			t.Fatalf("manifest entry %q does not map to any scenario file", entry)
		}
		if _, exists := seen[entry]; exists {
			t.Fatalf("manifest entry %q is duplicated", entry)
		}
		seen[entry] = struct{}{}
	}
}

func runScenarioScript(t *testing.T, grpcAddr string, absolutePath string) {
	t.Helper()

	cfg := scenariotool.Config{
		GRPCAddr:         grpcAddr,
		Timeout:          scenarioTimeout(),
		Assertions:       scenariotool.AssertionStrict,
		Verbose:          false,
		Logger:           log.New(io.Discard, "", 0),
		ValidateComments: false,
	}
	if err := scenariotool.RunFile(context.Background(), cfg, absolutePath); err != nil {
		t.Fatalf("run scenario %s: %v", absolutePath, err)
	}
}

func newScenarioRuntime(t *testing.T) scenarioRuntime {
	t.Helper()

	grpcAddr, _, stopServer := startGRPCServer(t)
	return scenarioRuntime{
		grpcAddr: grpcAddr,
		stop:     stopServer,
	}
}

func scenarioLuaPaths(t *testing.T) []scenarioPath {
	t.Helper()

	repo, allPaths := discoverScenarioLuaPaths(t)
	selected := allPaths

	manifest := scenarioManifestEntries(t, repo)
	if len(manifest) > 0 {
		selected = filterScenarioPaths(selected, func(path scenarioPath) bool {
			_, ok := manifest[path.subpath]
			return ok
		})
	}

	if only := scenarioOnlyEntries(); len(only) > 0 {
		selected = filterScenarioPaths(selected, func(path scenarioPath) bool {
			if _, ok := only[path.subpath]; ok {
				return true
			}
			if _, ok := only[filepath.Base(path.subpath)]; ok {
				return true
			}
			return false
		})
	}

	if filterExpr := strings.TrimSpace(os.Getenv("SCENARIO_FILTER")); filterExpr != "" {
		matcher, err := regexp.Compile(filterExpr)
		if err != nil {
			t.Fatalf("compile SCENARIO_FILTER %q: %v", filterExpr, err)
		}
		selected = filterScenarioPaths(selected, func(path scenarioPath) bool {
			return matcher.MatchString(path.subpath) || matcher.MatchString(filepath.Base(path.subpath))
		})
	}

	total, index := scenarioShardConfig(t)
	if total > 1 {
		selected = filterScenarioPaths(selected, func(path scenarioPath) bool {
			return scenarioShardForPath(path.subpath, total) == index
		})
	}

	if len(selected) == 0 {
		t.Skip("no scenario files selected")
	}

	return selected
}

func discoverScenarioLuaPaths(t *testing.T) (string, []scenarioPath) {
	t.Helper()

	repo := repoRoot(t)
	base := filepath.Join(repo, scenarioLuaRoot)
	var paths []scenarioPath
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() || filepath.Ext(path) != ".lua" {
			return nil
		}
		relative, err := filepath.Rel(repo, path)
		if err != nil {
			return err
		}
		subpath, err := filepath.Rel(base, path)
		if err != nil {
			return err
		}
		paths = append(paths, scenarioPath{
			absolute: path,
			relative: filepath.ToSlash(relative),
			subpath:  filepath.ToSlash(subpath),
		})
		return nil
	})
	if err != nil {
		t.Fatalf("walk scenarios under %s: %v", base, err)
	}
	if len(paths) == 0 {
		t.Fatalf("no scenarios found under %s", base)
	}
	sort.Slice(paths, func(i, j int) bool {
		return paths[i].subpath < paths[j].subpath
	})
	return repo, paths
}

func scenarioParallelism(t *testing.T) int {
	t.Helper()
	raw := strings.TrimSpace(os.Getenv("SCENARIO_PARALLELISM"))
	if raw == "" {
		return 1
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		t.Fatalf("invalid SCENARIO_PARALLELISM %q", raw)
	}
	return value
}

func scenarioShardConfig(t *testing.T) (int, int) {
	t.Helper()

	total := 1
	if raw := strings.TrimSpace(os.Getenv("SCENARIO_SHARD_TOTAL")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value <= 0 {
			t.Fatalf("invalid SCENARIO_SHARD_TOTAL %q", raw)
		}
		total = value
	}

	index := 0
	if raw := strings.TrimSpace(os.Getenv("SCENARIO_SHARD_INDEX")); raw != "" {
		value, err := strconv.Atoi(raw)
		if err != nil || value < 0 {
			t.Fatalf("invalid SCENARIO_SHARD_INDEX %q", raw)
		}
		index = value
	}
	if index >= total {
		t.Fatalf("SCENARIO_SHARD_INDEX=%d out of range for SCENARIO_SHARD_TOTAL=%d", index, total)
	}
	return total, index
}

func scenarioShardForPath(relativePath string, total int) int {
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(filepath.ToSlash(relativePath)))
	return int(hasher.Sum32() % uint32(total))
}

func scenarioManifestEntries(t *testing.T, repo string) map[string]struct{} {
	t.Helper()

	manifestPath := strings.TrimSpace(os.Getenv("SCENARIO_MANIFEST"))
	if manifestPath == "" {
		return nil
	}
	if !filepath.IsAbs(manifestPath) {
		manifestPath = filepath.Join(repo, manifestPath)
	}
	entries, err := readScenarioManifestEntries(manifestPath)
	if err != nil {
		t.Fatalf("read SCENARIO_MANIFEST %s: %v", manifestPath, err)
	}
	result := make(map[string]struct{}, len(entries))
	for _, entry := range entries {
		result[entry] = struct{}{}
	}
	return result
}

func readScenarioManifestEntries(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	entries := make([]string, 0)
	for _, line := range strings.Split(string(content), "\n") {
		value := strings.TrimSpace(line)
		if value == "" || strings.HasPrefix(value, "#") {
			continue
		}
		entries = append(entries, filepath.ToSlash(filepath.Clean(value)))
	}
	return entries, nil
}

func scenarioOnlyEntries() map[string]struct{} {
	raw := strings.TrimSpace(os.Getenv("SCENARIO_ONLY"))
	if raw == "" {
		return nil
	}
	entries := make(map[string]struct{})
	for _, value := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ',' || r == '\n' || r == '\t' || r == ' '
	}) {
		candidate := strings.TrimSpace(value)
		if candidate == "" {
			continue
		}
		candidate = filepath.ToSlash(filepath.Clean(candidate))
		entries[candidate] = struct{}{}
		entries[filepath.Base(candidate)] = struct{}{}
	}
	return entries
}

func filterScenarioPaths(paths []scenarioPath, keep func(path scenarioPath) bool) []scenarioPath {
	selected := make([]scenarioPath, 0, len(paths))
	for _, path := range paths {
		if keep(path) {
			selected = append(selected, path)
		}
	}
	return selected
}

func expectedSystemFromSubpath(subpath string) (string, error) {
	parts := strings.Split(filepath.ToSlash(strings.TrimSpace(subpath)), "/")
	if len(parts) < 3 || parts[0] != scenarioSystemRootSegment {
		return "", os.ErrInvalid
	}
	system := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(parts[1]), "-", "_"))
	if system == "" {
		return "", os.ErrInvalid
	}
	return system, nil
}
