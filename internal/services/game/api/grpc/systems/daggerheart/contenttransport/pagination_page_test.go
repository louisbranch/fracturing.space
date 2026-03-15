package contenttransport

import "testing"

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
