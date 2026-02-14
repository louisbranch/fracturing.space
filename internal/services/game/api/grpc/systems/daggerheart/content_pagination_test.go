package daggerheart

import (
	"testing"

	"github.com/louisbranch/fracturing.space/internal/platform/grpc/pagination"
	contentfilter "github.com/louisbranch/fracturing.space/internal/services/game/domain/systems/daggerheart/content/filter"
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
