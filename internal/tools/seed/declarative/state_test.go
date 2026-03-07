package declarative

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFileStateStoreSaveUsesInjectedClock(t *testing.T) {
	t.Parallel()

	fixedNow := time.Date(2026, time.March, 6, 15, 4, 5, 123456000, time.UTC)
	store := newFileStateStore(func() time.Time {
		return fixedNow
	})

	path := filepath.Join(t.TempDir(), "seed-state.json")
	state := seedState{
		Entries: map[string]string{
			"user:alice": "user-1",
		},
	}
	if err := store.Save(path, state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read state file: %v", err)
	}

	var saved seedState
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("decode saved state: %v", err)
	}
	if saved.UpdatedAt != fixedNow.Format(time.RFC3339Nano) {
		t.Fatalf("updated_at = %q, want %q", saved.UpdatedAt, fixedNow.Format(time.RFC3339Nano))
	}
	if saved.Version != stateVersion {
		t.Fatalf("version = %d, want %d", saved.Version, stateVersion)
	}
	if saved.Entries["user:alice"] != "user-1" {
		t.Fatalf("saved entry = %q, want %q", saved.Entries["user:alice"], "user-1")
	}
}
