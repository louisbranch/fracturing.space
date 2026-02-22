package cache

import (
	"path/filepath"
	"testing"
)

func TestOpenStoreReturnsNilWhenPathEmpty(t *testing.T) {
	t.Parallel()

	store, err := OpenStore("  ")
	if err != nil {
		t.Fatalf("OpenStore returned error: %v", err)
	}
	if store != nil {
		t.Fatal("OpenStore returned non-nil store for empty path")
	}
}

func TestOpenStoreCreatesParentDir(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "nested", "web-cache.db")
	store, err := OpenStore(path)
	if err != nil {
		t.Fatalf("OpenStore returned error: %v", err)
	}
	if store == nil {
		t.Fatal("OpenStore returned nil store")
	}
	t.Cleanup(func() {
		_ = store.Close()
	})
}

func TestBuildAuthConsentURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		base      string
		pendingID string
		want      string
	}{
		{
			name:      "blank base uses relative consent path",
			base:      "",
			pendingID: "pending 1",
			want:      "/authorize/consent?pending_id=pending+1",
		},
		{
			name:      "base trims trailing slash",
			base:      "http://auth.local/",
			pendingID: "pending-1",
			want:      "http://auth.local/authorize/consent?pending_id=pending-1",
		},
		{
			name:      "base preserves path prefix",
			base:      "http://auth.local/auth",
			pendingID: "pending-1",
			want:      "http://auth.local/auth/authorize/consent?pending_id=pending-1",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := BuildAuthConsentURL(tc.base, tc.pendingID)
			if got != tc.want {
				t.Fatalf("BuildAuthConsentURL(%q, %q) = %q, want %q", tc.base, tc.pendingID, got, tc.want)
			}
		})
	}
}
