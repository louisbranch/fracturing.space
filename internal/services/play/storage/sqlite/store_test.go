package sqlite

import (
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/louisbranch/fracturing.space/internal/services/play/transcript/transcripttest"
)

func TestStoreContracts(t *testing.T) {
	t.Parallel()

	transcripttest.RunStoreContract(t, func(t *testing.T) transcripttest.Store {
		t.Helper()

		store, err := Open(filepath.Join(t.TempDir(), "play.sqlite"))
		if err != nil {
			t.Fatalf("Open() error = %v", err)
		}
		baseTime := time.Date(2026, time.March, 13, 12, 0, 0, 0, time.UTC)
		var callCount int64
		store.now = func() time.Time {
			next := atomic.AddInt64(&callCount, 1)
			return baseTime.Add(time.Duration(next) * time.Second)
		}
		return store
	})
}
