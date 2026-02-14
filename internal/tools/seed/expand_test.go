package seed

import "testing"

func TestParseCaptureSpecNil(t *testing.T) {
	captures, err := ParseCaptureSpec(nil)
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if captures != nil {
		t.Fatal("expected nil captures")
	}
}

func TestParseCaptureSpecRejectsNonObject(t *testing.T) {
	if _, err := ParseCaptureSpec("bad"); err == nil {
		t.Fatal("expected error for non-object capture")
	}
}

func TestParseCaptureSpecEmptyObject(t *testing.T) {
	captures, err := ParseCaptureSpec(map[string]any{})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if captures != nil {
		t.Fatal("expected nil captures for empty object")
	}
}

func TestParseCaptureSpecDefaultShortcut(t *testing.T) {
	captures, err := ParseCaptureSpec(map[string]any{"camp": "campaign"})
	if err != nil {
		t.Fatalf("parse capture spec: %v", err)
	}
	paths := captures["camp"]
	if len(paths) != 2 {
		t.Fatalf("expected 2 default paths, got %d", len(paths))
	}
	if paths[0] != CaptureDefaults["campaign"][0] {
		t.Fatal("expected default campaign path")
	}
}

func TestParseCaptureSpecStringPath(t *testing.T) {
	captures, err := ParseCaptureSpec(map[string]any{"id": "result.id"})
	if err != nil {
		t.Fatalf("parse capture spec: %v", err)
	}
	paths := captures["id"]
	if len(paths) != 1 || paths[0] != "result.id" {
		t.Fatalf("expected single path, got %v", paths)
	}
}

func TestParseCaptureSpecArrayPaths(t *testing.T) {
	captures, err := ParseCaptureSpec(map[string]any{"id": []any{"result.id", "result.alt"}})
	if err != nil {
		t.Fatalf("parse capture spec: %v", err)
	}
	paths := captures["id"]
	if len(paths) != 2 {
		t.Fatalf("expected 2 paths, got %d", len(paths))
	}
}

func TestParseCaptureSpecRejectsNonStringArrayEntry(t *testing.T) {
	_, err := ParseCaptureSpec(map[string]any{"id": []any{"result.id", 1}})
	if err == nil {
		t.Fatal("expected error for non-string capture path")
	}
}

func TestParseCaptureSpecRejectsUnsupportedType(t *testing.T) {
	_, err := ParseCaptureSpec(map[string]any{"id": 42})
	if err == nil {
		t.Fatal("expected error for unsupported capture type")
	}
}
