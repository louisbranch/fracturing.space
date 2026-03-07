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
