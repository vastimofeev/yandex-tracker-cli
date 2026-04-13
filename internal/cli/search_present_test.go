package cli

import (
	"testing"

	"github.com/vasti/yandex-tracker-cli/internal/model"
)

func TestPresentIssueSearchSortLimitSelect(t *testing.T) {
	t.Parallel()

	result := &model.SearchResult[model.Issue]{
		TotalCount: 3,
		Items: []model.Issue{
			{Key: "DV-3", Summary: "Third", Start: "2026-04-11", Status: &model.Reference{Display: "Open"}},
			{Key: "DV-1", Summary: "First", Start: "2026-04-09", Status: &model.Reference{Display: "In Progress"}},
			{Key: "DV-2", Summary: "Second", Start: "2026-04-10", Status: &model.Reference{Display: "Open"}},
		},
	}

	got, err := presentIssueSearch(result, "start", "asc", 1, []string{"key", "start", "status.display"})
	if err != nil {
		t.Fatalf("presentIssueSearch() error = %v", err)
	}

	out, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("result type = %T, want map[string]any", got)
	}
	items, ok := out["items"].([]map[string]any)
	if ok {
		if len(items) != 1 {
			t.Fatalf("items len = %d, want 1", len(items))
		}
		if items[0]["key"] != "DV-1" {
			t.Fatalf("key = %#v, want DV-1", items[0]["key"])
		}
		return
	}

	rawItems, ok := out["items"].([]any)
	if !ok || len(rawItems) != 1 {
		t.Fatalf("items = %#v, want one selected item", out["items"])
	}
	row, ok := rawItems[0].(map[string]any)
	if !ok {
		t.Fatalf("row = %#v, want map[string]any", rawItems[0])
	}
	if row["key"] != "DV-1" {
		t.Fatalf("key = %#v, want DV-1", row["key"])
	}
	if row["start"] != "2026-04-09" {
		t.Fatalf("start = %#v, want 2026-04-09", row["start"])
	}
	if row["status.display"] != "In Progress" {
		t.Fatalf("status.display = %#v, want In Progress", row["status.display"])
	}
}

func TestPresentIssueSearchRejectsBadOrder(t *testing.T) {
	t.Parallel()

	_, err := presentIssueSearch(&model.SearchResult[model.Issue]{}, "start", "sideways", 0, nil)
	if err == nil {
		t.Fatal("expected error for invalid order")
	}
}
