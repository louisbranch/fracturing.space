package grpcpaging

import (
	"context"
	"errors"
	"testing"
)

func TestCollectPagesSinglePage(t *testing.T) {
	t.Parallel()

	items, err := CollectPages[string, string](
		context.Background(),
		10,
		func(_ context.Context, pageToken string) ([]string, string, error) {
			return []string{"a", "b"}, "", nil
		},
		func(s string) (string, bool) { return s, true },
	)
	if err != nil {
		t.Fatalf("CollectPages() error = %v", err)
	}
	if len(items) != 2 || items[0] != "a" || items[1] != "b" {
		t.Fatalf("items = %v, want [a b]", items)
	}
}

func TestCollectPagesMultiplePages(t *testing.T) {
	t.Parallel()

	call := 0
	items, err := CollectPages[int, int](
		context.Background(),
		2,
		func(_ context.Context, pageToken string) ([]int, string, error) {
			call++
			switch call {
			case 1:
				return []int{1, 2}, "page2", nil
			case 2:
				return []int{3}, "", nil
			default:
				t.Fatal("unexpected extra call")
				return nil, "", nil
			}
		},
		func(i int) (int, bool) { return i, true },
	)
	if err != nil {
		t.Fatalf("CollectPages() error = %v", err)
	}
	if len(items) != 3 {
		t.Fatalf("len(items) = %d, want 3", len(items))
	}
}

func TestCollectPagesSkipsFilteredItems(t *testing.T) {
	t.Parallel()

	items, err := CollectPages[int, int](
		context.Background(),
		10,
		func(_ context.Context, _ string) ([]int, string, error) {
			return []int{1, 2, 3, 4}, "", nil
		},
		func(i int) (int, bool) { return i, i%2 == 0 },
	)
	if err != nil {
		t.Fatalf("CollectPages() error = %v", err)
	}
	if len(items) != 2 || items[0] != 2 || items[1] != 4 {
		t.Fatalf("items = %v, want [2 4]", items)
	}
}

func TestCollectPagesReturnsErrorFromFetch(t *testing.T) {
	t.Parallel()

	fetchErr := errors.New("rpc failed")
	_, err := CollectPages[string, string](
		context.Background(),
		10,
		func(_ context.Context, _ string) ([]string, string, error) {
			return nil, "", fetchErr
		},
		func(s string) (string, bool) { return s, true },
	)
	if !errors.Is(err, fetchErr) {
		t.Fatalf("err = %v, want %v", err, fetchErr)
	}
}

func TestCollectPagesMaxStopsAtLimit(t *testing.T) {
	t.Parallel()

	call := 0
	items, err := CollectPagesMax[int, int](
		context.Background(),
		1,
		2,
		func(_ context.Context, _ string) ([]int, string, error) {
			call++
			return []int{call}, "next", nil
		},
		func(i int) (int, bool) { return i, true },
	)
	if err != nil {
		t.Fatalf("CollectPagesMax() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if call != 2 {
		t.Fatalf("call count = %d, want 2", call)
	}
}

func TestCollectPagesStopsOnDuplicateToken(t *testing.T) {
	t.Parallel()

	call := 0
	items, err := CollectPages[int, int](
		context.Background(),
		10,
		func(_ context.Context, pageToken string) ([]int, string, error) {
			call++
			return []int{call}, "same-token", nil
		},
		func(i int) (int, bool) { return i, true },
	)
	if err != nil {
		t.Fatalf("CollectPages() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2 (should stop when next token equals current token)", len(items))
	}
	if call != 2 {
		t.Fatalf("call count = %d, want 2", call)
	}
}
