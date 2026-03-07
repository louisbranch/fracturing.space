package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindModuleRoot(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/project\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	nested := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(nested, 0o755); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}

	got, err := FindModuleRoot(nested)
	if err != nil {
		t.Fatalf("FindModuleRoot returned error: %v", err)
	}
	if got != root {
		t.Fatalf("FindModuleRoot = %q, want %q", got, root)
	}
}

func TestFindModuleRootMissing(t *testing.T) {
	root := t.TempDir()

	_, err := FindModuleRoot(root)
	if err == nil {
		t.Fatal("expected error when go.mod is missing")
	}
}

func TestResolveRoot(t *testing.T) {
	t.Run("explicit flag", func(t *testing.T) {
		got, err := ResolveRoot("/some/path")
		if err != nil {
			t.Fatalf("ResolveRoot returned error: %v", err)
		}
		if got != "/some/path" {
			t.Fatalf("ResolveRoot = %q, want /some/path", got)
		}
	})

	t.Run("empty flag uses cwd", func(t *testing.T) {
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/project\n"), 0o644); err != nil {
			t.Fatalf("write go.mod: %v", err)
		}

		oldWD, err := os.Getwd()
		if err != nil {
			t.Fatalf("get working dir: %v", err)
		}
		if err := os.Chdir(root); err != nil {
			t.Fatalf("chdir: %v", err)
		}
		t.Cleanup(func() { _ = os.Chdir(oldWD) })

		got, err := ResolveRoot("")
		if err != nil {
			t.Fatalf("ResolveRoot returned error: %v", err)
		}
		if got != root {
			t.Fatalf("ResolveRoot = %q, want %q", got, root)
		}
	})
}

func TestResolvePath(t *testing.T) {
	root := t.TempDir()
	relative := filepath.Join("docs", "out.md")
	absolute := filepath.Join(t.TempDir(), "out.md")

	if got := ResolvePath(root, relative); got != filepath.Join(root, relative) {
		t.Fatalf("ResolvePath(relative) = %q, want %q", got, filepath.Join(root, relative))
	}
	if got := ResolvePath(root, absolute); got != absolute {
		t.Fatalf("ResolvePath(absolute) = %q, want %q", got, absolute)
	}
}
