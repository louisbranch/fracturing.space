package app

import "testing"

func TestStartupRollbackCleanup_ReverseOrder(t *testing.T) {
	rollback := startupRollback{}
	order := make([]int, 0, 3)
	rollback.add(func() { order = append(order, 1) })
	rollback.add(func() { order = append(order, 2) })
	rollback.add(func() { order = append(order, 3) })

	rollback.cleanup()

	want := []int{3, 2, 1}
	if len(order) != len(want) {
		t.Fatalf("cleanup order len = %d, want %d", len(order), len(want))
	}
	for idx := range want {
		if order[idx] != want[idx] {
			t.Fatalf("cleanup order[%d] = %d, want %d", idx, order[idx], want[idx])
		}
	}
}

func TestStartupRollbackRelease_SkipsCleanup(t *testing.T) {
	rollback := startupRollback{}
	calls := 0
	rollback.add(func() { calls++ })
	rollback.release()

	rollback.cleanup()

	if calls != 0 {
		t.Fatalf("cleanup calls after release = %d, want 0", calls)
	}
}
