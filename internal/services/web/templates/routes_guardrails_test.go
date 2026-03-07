package templates

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTemplateSourcesDoNotHardcodeInternalRoutes(t *testing.T) {
	t.Parallel()

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("read templates directory: %v", err)
	}

	forbiddenFragments := []string{
		`"/app/`,
		`"/discover/`,
		`"/auth/`,
		`"/profile/`,
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".templ" {
			continue
		}

		path := entry.Name()
		file, err := os.Open(path)
		if err != nil {
			t.Fatalf("open %s: %v", path, err)
		}

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
			_ = file.Close()
			t.Fatalf("scan %s: %v", path, err)
		}
		if err := file.Close(); err != nil {
			t.Fatalf("close %s: %v", path, err)
		}
	}
}
