package web

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
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

	requiredFiles := []string{
		"routepath/doc.go",
		"routepath/helpers.go",
		"routepath/core.go",
		"routepath/publicauth.go",
		"routepath/discovery.go",
		"routepath/invite.go",
		"routepath/profile.go",
		"routepath/dashboard.go",
		"routepath/campaigns.go",
		"routepath/notifications.go",
		"routepath/settings.go",
	}

	for _, path := range requiredFiles {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("routepath ownership file %s missing: %v", path, err)
		}
	}
	if _, err := os.Stat("routepath/routepath.go"); !os.IsNotExist(err) {
		t.Fatalf("routepath monolith returned; keep owned surfaces split into area files")
	}

	entries, err := os.ReadDir("routepath")
	if err != nil {
		t.Fatalf("read routepath dir: %v", err)
	}

	var goFiles []string
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".go" || strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		goFiles = append(goFiles, entry.Name())
	}
	slices.Sort(goFiles)

	wantFiles := []string{
		"campaigns.go",
		"core.go",
		"dashboard.go",
		"discovery.go",
		"doc.go",
		"helpers.go",
		"invite.go",
		"notifications.go",
		"profile.go",
		"publicauth.go",
		"settings.go",
	}
	if !slices.Equal(goFiles, wantFiles) {
		t.Fatalf("routepath files = %v, want %v", goFiles, wantFiles)
	}
}
