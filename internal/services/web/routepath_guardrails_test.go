package web

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGoSourcesDoNotHardcodeInternalRoutes(t *testing.T) {
	t.Parallel()

	forbiddenFragments := []string{
		`"/app/`,
		`"/discover/`,
		`"/invite/`,
		`"/auth/`,
		`"/passkeys/`,
		`"/u/`,
	}

	walkErr := filepath.WalkDir(".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if path == "routepath" {
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Ext(path) != ".go" || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer func() {
			_ = file.Close()
		}()

		scanner := bufio.NewScanner(file)
		lineNo := 1
		for scanner.Scan() {
			line := scanner.Text()
			for _, fragment := range forbiddenFragments {
				if strings.Contains(line, fragment) {
					t.Errorf("%s:%d contains hardcoded route fragment %q; use routepath constants", path, lineNo, fragment)
				}
			}
			lineNo++
		}
		if err := scanner.Err(); err != nil {
			return err
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("scan web package: %v", walkErr)
	}
}

func TestRoutepathPackageStaysSplitByOwnedSurface(t *testing.T) {
	t.Parallel()

	for _, path := range []string{
		"routepath/doc.go",
		"routepath/helpers.go",
		"routepath/core.go",
		"routepath/publicauth.go",
		"routepath/discovery.go",
		"routepath/invite.go",
		"routepath/profile.go",
		"routepath/dashboard.go",
		"routepath/notifications.go",
		"routepath/settings.go",
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("routepath ownership file %s missing: %v", path, err)
		}
	}

	for _, path := range []string{"routepath/routepath.go", "routepath/campaigns.go"} {
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Fatalf("%s returned; keep route ownership split by owned surface", path)
		}
	}

	entries, err := os.ReadDir("routepath")
	if err != nil {
		t.Fatalf("read routepath dir: %v", err)
	}

	campaignSurfaceFiles := map[string]struct{}{}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		if strings.HasPrefix(entry.Name(), "campaigns_") {
			campaignSurfaceFiles[entry.Name()] = struct{}{}
		}
	}

	for _, required := range []string{
		"campaigns_core.go",
		"campaigns_overview.go",
		"campaigns_participants.go",
		"campaigns_characters.go",
		"campaigns_sessions.go",
		"campaigns_invites.go",
		"campaigns_starters.go",
	} {
		if _, ok := campaignSurfaceFiles[required]; !ok {
			t.Fatalf("routepath missing campaign surface file %s", required)
		}
	}
	if len(campaignSurfaceFiles) < 7 {
		t.Fatalf("campaign routepath surfaces = %d, want at least 7 split files", len(campaignSurfaceFiles))
	}
}
