package daggerheart

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	contentfilter "github.com/louisbranch/fracturing.space/internal/services/game/domain/bridge/daggerheart/content/filter"
)

type testContentItem struct {
	ID   string
	Name string
}

func testContentConfig() contentListConfig[testContentItem] {
	return contentListConfig[testContentItem]{
		PageSizeConfig: pagination.PageSizeConfig{Default: 2, Max: 5},
		OrderByConfig: pagination.OrderByConfig{
			Default: "name",
			Allowed: []string{"name", "name desc"},
		},
		FilterFields: contentfilter.Fields{
			"id":   contentfilter.FieldString,
			"name": contentfilter.FieldString,
		},
		KeySpec: []contentKeySpec{{Name: "name", Kind: pagination.CursorValueString}, {Name: "id", Kind: pagination.CursorValueString}},
		KeyFunc: func(item testContentItem) []pagination.CursorValue {
			return []pagination.CursorValue{
				pagination.StringValue("name", item.Name),
				pagination.StringValue("id", item.ID),
			}
		},
		Resolver: func(item testContentItem, field string) (any, bool) {
			switch field {
			case "id":
				return item.ID, true
			case "name":
				return item.Name, true
			default:
				return nil, false
			}
		},
	}
}

func TestListContentPagePagination(t *testing.T) {
	items := []testContentItem{
		{ID: "a", Name: "Alpha"},
		{ID: "b", Name: "Beta"},
		{ID: "c", Name: "Gamma"},
	}

	page1, err := listContentPage(items, contentListRequest{PageSize: 2}, testContentConfig())
	if err != nil {
		t.Fatalf("page1 error: %v", err)
	}
	if len(page1.Items) != 2 {
		t.Fatalf("page1 expected 2 items, got %d", len(page1.Items))
	}
	if page1.Items[0].ID != "a" || page1.Items[1].ID != "b" {
		t.Fatalf("page1 order mismatch: %+v", page1.Items)
	}
	if page1.NextPageToken == "" {
		t.Fatal("expected next page token")
	}

	page2, err := listContentPage(items, contentListRequest{PageSize: 2, PageToken: page1.NextPageToken}, testContentConfig())
	if err != nil {
		t.Fatalf("page2 error: %v", err)
	}
	if len(page2.Items) != 1 || page2.Items[0].ID != "c" {
		t.Fatalf("page2 items mismatch: %+v", page2.Items)
	}
	if page2.PreviousPageToken == "" {
		t.Fatal("expected previous page token")
	}

	pageBack, err := listContentPage(items, contentListRequest{PageSize: 2, PageToken: page2.PreviousPageToken}, testContentConfig())
	if err != nil {
		t.Fatalf("pageBack error: %v", err)
	}
	if len(pageBack.Items) != 2 || pageBack.Items[0].ID != "a" {
		t.Fatalf("pageBack items mismatch: %+v", pageBack.Items)
	}
}

func TestListContentPageDescending(t *testing.T) {
	items := []testContentItem{
		{ID: "a", Name: "Alpha"},
		{ID: "b", Name: "Beta"},
		{ID: "c", Name: "Gamma"},
	}

	page1, err := listContentPage(items, contentListRequest{PageSize: 2, OrderBy: "name desc"}, testContentConfig())
	if err != nil {
		t.Fatalf("page1 error: %v", err)
	}
	if len(page1.Items) != 2 {
		t.Fatalf("page1 expected 2 items, got %d", len(page1.Items))
	}
	if page1.Items[0].ID != "c" || page1.Items[1].ID != "b" {
		t.Fatalf("page1 order mismatch: %+v", page1.Items)
	}
	if page1.NextPageToken == "" {
		t.Fatal("expected next page token")
	}

	page2, err := listContentPage(items, contentListRequest{PageSize: 2, OrderBy: "name desc", PageToken: page1.NextPageToken}, testContentConfig())
	if err != nil {
		t.Fatalf("page2 error: %v", err)
	}
	if len(page2.Items) != 1 || page2.Items[0].ID != "a" {
		t.Fatalf("page2 items mismatch: %+v", page2.Items)
	}
}

func TestCompareInts(t *testing.T) {
	tests := []struct {
		left, right int64
		want        int
	}{
		{1, 2, -1},
		{2, 2, 0},
		{3, 2, 1},
		{-5, 5, -1},
		{0, 0, 0},
	}
	for _, tc := range tests {
		if got := compareInts(tc.left, tc.right); got != tc.want {
			t.Errorf("compareInts(%d, %d) = %d, want %d", tc.left, tc.right, got, tc.want)
		}
	}
}

func TestCompareUints(t *testing.T) {
	tests := []struct {
		left, right uint64
		want        int
	}{
		{1, 2, -1},
		{2, 2, 0},
		{3, 2, 1},
		{0, 0, 0},
		{0, 100, -1},
	}
	for _, tc := range tests {
		if got := compareUints(tc.left, tc.right); got != tc.want {
			t.Errorf("compareUints(%d, %d) = %d, want %d", tc.left, tc.right, got, tc.want)
		}
	}
}

func TestCompareCursorValue(t *testing.T) {
	t.Run("string comparison", func(t *testing.T) {
		left := pagination.StringValue("name", "alpha")
		right := pagination.StringValue("name", "beta")
		got, err := compareCursorValue(left, right)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != -1 {
			t.Errorf("expected -1, got %d", got)
		}
	})

	t.Run("int comparison", func(t *testing.T) {
		left := pagination.IntValue("seq", 10)
		right := pagination.IntValue("seq", 5)
		got, err := compareCursorValue(left, right)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 1 {
			t.Errorf("expected 1, got %d", got)
		}
	})

	t.Run("uint comparison", func(t *testing.T) {
		left := pagination.UintValue("id", 7)
		right := pagination.UintValue("id", 7)
		got, err := compareCursorValue(left, right)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 0 {
			t.Errorf("expected 0, got %d", got)
		}
	})

	t.Run("kind mismatch", func(t *testing.T) {
		left := pagination.StringValue("name", "a")
		right := pagination.IntValue("name", 1)
		_, err := compareCursorValue(left, right)
		if err == nil {
			t.Fatal("expected error for kind mismatch")
		}
	})

	t.Run("unsupported kind", func(t *testing.T) {
		left := pagination.CursorValue{Name: "x", Kind: "unknown"}
		right := pagination.CursorValue{Name: "x", Kind: "unknown"}
		_, err := compareCursorValue(left, right)
		if err == nil {
			t.Fatal("expected error for unsupported kind")
		}
	})
}

func TestCompareCursorValues(t *testing.T) {
	t.Run("equal multi-key", func(t *testing.T) {
		left := []pagination.CursorValue{
			pagination.StringValue("name", "alpha"),
			pagination.StringValue("id", "1"),
		}
		right := []pagination.CursorValue{
			pagination.StringValue("name", "alpha"),
			pagination.StringValue("id", "1"),
		}
		got, err := compareCursorValues(left, right)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != 0 {
			t.Errorf("expected 0, got %d", got)
		}
	})

	t.Run("first key differs", func(t *testing.T) {
		left := []pagination.CursorValue{
			pagination.StringValue("name", "alpha"),
			pagination.StringValue("id", "2"),
		}
		right := []pagination.CursorValue{
			pagination.StringValue("name", "beta"),
			pagination.StringValue("id", "1"),
		}
		got, err := compareCursorValues(left, right)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got != -1 {
			t.Errorf("expected -1, got %d", got)
		}
	})

	t.Run("length mismatch", func(t *testing.T) {
		left := []pagination.CursorValue{pagination.StringValue("a", "1")}
		right := []pagination.CursorValue{
			pagination.StringValue("a", "1"),
			pagination.StringValue("b", "2"),
		}
		_, err := compareCursorValues(left, right)
		if err == nil {
			t.Fatal("expected error for length mismatch")
		}
	})
}

func TestCursorKeysFromToken(t *testing.T) {
	t.Run("string key", func(t *testing.T) {
		cursor := pagination.Cursor{
			Values: []pagination.CursorValue{pagination.StringValue("name", "test")},
		}
		specs := []contentKeySpec{{Name: "name", Kind: pagination.CursorValueString}}
		keys, err := cursorKeysFromToken(cursor, specs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 1 || keys[0].StringValue != "test" {
			t.Errorf("unexpected keys: %+v", keys)
		}
	})

	t.Run("int key", func(t *testing.T) {
		cursor := pagination.Cursor{
			Values: []pagination.CursorValue{pagination.IntValue("seq", 42)},
		}
		specs := []contentKeySpec{{Name: "seq", Kind: pagination.CursorValueInt}}
		keys, err := cursorKeysFromToken(cursor, specs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 1 || keys[0].IntValue != 42 {
			t.Errorf("unexpected keys: %+v", keys)
		}
	})

	t.Run("uint key", func(t *testing.T) {
		cursor := pagination.Cursor{
			Values: []pagination.CursorValue{pagination.UintValue("id", 99)},
		}
		specs := []contentKeySpec{{Name: "id", Kind: pagination.CursorValueUint}}
		keys, err := cursorKeysFromToken(cursor, specs)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(keys) != 1 || keys[0].UintValue != 99 {
			t.Errorf("unexpected keys: %+v", keys)
		}
	})

	t.Run("missing key", func(t *testing.T) {
		cursor := pagination.Cursor{}
		specs := []contentKeySpec{{Name: "missing", Kind: pagination.CursorValueString}}
		_, err := cursorKeysFromToken(cursor, specs)
		if err == nil {
			t.Fatal("expected error for missing key")
		}
	})

	t.Run("unsupported kind", func(t *testing.T) {
		cursor := pagination.Cursor{}
		specs := []contentKeySpec{{Name: "x", Kind: "bogus"}}
		_, err := cursorKeysFromToken(cursor, specs)
		if err == nil {
			t.Fatal("expected error for unsupported kind")
		}
	})
}

func TestValidateKeySpec(t *testing.T) {
	t.Run("matching specs", func(t *testing.T) {
		base := []pagination.CursorValue{
			pagination.StringValue("name", "a"),
			pagination.StringValue("id", "1"),
		}
		candidate := []pagination.CursorValue{
			pagination.StringValue("name", "b"),
			pagination.StringValue("id", "2"),
		}
		if err := validateKeySpec(base, candidate); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})

	t.Run("length mismatch", func(t *testing.T) {
		base := []pagination.CursorValue{pagination.StringValue("name", "a")}
		candidate := []pagination.CursorValue{
			pagination.StringValue("name", "a"),
			pagination.StringValue("id", "1"),
		}
		if err := validateKeySpec(base, candidate); err == nil {
			t.Fatal("expected error for length mismatch")
		}
	})

	t.Run("name mismatch", func(t *testing.T) {
		base := []pagination.CursorValue{pagination.StringValue("name", "a")}
		candidate := []pagination.CursorValue{pagination.StringValue("id", "a")}
		if err := validateKeySpec(base, candidate); err == nil {
			t.Fatal("expected error for name mismatch")
		}
	})

	t.Run("kind mismatch", func(t *testing.T) {
		base := []pagination.CursorValue{pagination.StringValue("name", "a")}
		candidate := []pagination.CursorValue{pagination.IntValue("name", 1)}
		if err := validateKeySpec(base, candidate); err == nil {
			t.Fatal("expected error for kind mismatch")
		}
	})
}

func TestListContentPageInvalidFilter(t *testing.T) {
	items := []testContentItem{{ID: "a", Name: "Alpha"}}
	_, err := listContentPage(items, contentListRequest{Filter: "!!!invalid"}, testContentConfig())
	if err == nil {
		t.Fatal("expected error for invalid filter syntax")
	}
}

func TestListContentPageInvalidOrderBy(t *testing.T) {
	items := []testContentItem{{ID: "a", Name: "Alpha"}}
	_, err := listContentPage(items, contentListRequest{OrderBy: "nonexistent"}, testContentConfig())
	if err == nil {
		t.Fatal("expected error for invalid order_by")
	}
}

func TestListContentPageInvalidPageToken(t *testing.T) {
	items := []testContentItem{{ID: "a", Name: "Alpha"}}
	_, err := listContentPage(items, contentListRequest{PageToken: "not-valid-base64!!"}, testContentConfig())
	if err == nil {
		t.Fatal("expected error for invalid page token")
	}
}

func TestListContentPageEmptyResult(t *testing.T) {
	items := []testContentItem{}
	page, err := listContentPage(items, contentListRequest{}, testContentConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(page.Items))
	}
	if page.TotalSize != 0 {
		t.Fatalf("expected total size 0, got %d", page.TotalSize)
	}
}

func TestListContentPageFilterNoMatch(t *testing.T) {
	items := []testContentItem{
		{ID: "a", Name: "Alpha"},
		{ID: "b", Name: "Beta"},
	}
	page, err := listContentPage(items, contentListRequest{Filter: `name = "Nonexistent"`}, testContentConfig())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(page.Items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(page.Items))
	}
}

func TestListContentPageFilter(t *testing.T) {
	items := []testContentItem{
		{ID: "a", Name: "Alpha"},
		{ID: "b", Name: "Beta"},
	}

	page, err := listContentPage(items, contentListRequest{Filter: "name = \"Beta\""}, testContentConfig())
	if err != nil {
		t.Fatalf("filter error: %v", err)
	}
	if len(page.Items) != 1 || page.Items[0].ID != "b" {
		t.Fatalf("filtered items mismatch: %+v", page.Items)
	}
	if page.TotalSize != 1 {
		t.Fatalf("expected total size 1, got %d", page.TotalSize)
	}
}
