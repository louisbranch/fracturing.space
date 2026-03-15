package coreprojection

import (
	"errors"
	"testing"

	"github.com/louisbranch/fracturing.space/internal/platform/storage/sqliteutil"
)

func TestMapPageRows_TruncatesAndReturnsNextToken(t *testing.T) {
	rows := []struct {
		id   string
		name string
	}{
		{id: "r1", name: "one"},
		{id: "r2", name: "two"},
		{id: "r3", name: "three"},
	}

	items, nextToken, err := sqliteutil.MapPageRows(rows, 2, func(row struct {
		id   string
		name string
	}) string {
		return row.id
	}, func(row struct {
		id   string
		name string
	}) (string, error) {
		return row.name, nil
	})
	if err != nil {
		t.Fatalf("sqliteutil.MapPageRows() error = %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("len(items) = %d, want 2", len(items))
	}
	if items[0] != "one" || items[1] != "two" {
		t.Fatalf("items = %#v, want [one two]", items)
	}
	if nextToken != "r2" {
		t.Fatalf("next token = %q, want %q", nextToken, "r2")
	}
}

func TestMapPageRows_NoNextTokenWhenUnderLimit(t *testing.T) {
	rows := []struct {
		id string
	}{
		{id: "r1"},
	}

	items, nextToken, err := sqliteutil.MapPageRows(rows, 2, func(row struct{ id string }) string {
		return row.id
	}, func(row struct{ id string }) (string, error) {
		return row.id, nil
	})
	if err != nil {
		t.Fatalf("sqliteutil.MapPageRows() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0] != "r1" {
		t.Fatalf("items[0] = %q, want %q", items[0], "r1")
	}
	if nextToken != "" {
		t.Fatalf("next token = %q, want empty", nextToken)
	}
}

func TestMapPageRows_MapperError(t *testing.T) {
	boom := errors.New("boom")
	rows := []struct {
		id string
	}{
		{id: "r1"},
	}

	items, nextToken, err := sqliteutil.MapPageRows(rows, 1, func(row struct{ id string }) string {
		return row.id
	}, func(row struct{ id string }) (string, error) {
		return "", boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("sqliteutil.MapPageRows() error = %v, want %v", err, boom)
	}
	if items != nil {
		t.Fatalf("items = %#v, want nil on mapper error", items)
	}
	if nextToken != "" {
		t.Fatalf("next token = %q, want empty on mapper error", nextToken)
	}
}
