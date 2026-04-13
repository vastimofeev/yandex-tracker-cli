package output

import (
	"strings"
	"testing"

	"github.com/vasti/yandex-tracker-cli/internal/model"
)

func TestHumanFormatsIssue(t *testing.T) {
	t.Parallel()

	text := Human(&model.Issue{
		Key:         "DV-1",
		Summary:     "Demo issue",
		Version:     3,
		Description: "Line one",
		Status:      &model.Reference{Key: "open", Display: "Open"},
		Type:        &model.Reference{Key: "task", Display: "Task"},
		Priority:    &model.Reference{Key: "normal", Display: "Normal"},
		Queue:       &model.Reference{Key: "DV", Display: "Development"},
		Assignee:    &model.User{Display: "Alice"},
	})

	for _, want := range []string{
		"DV-1  Demo issue",
		"status: Open",
		"type: Task",
		"priority: Normal",
		"queue: Development",
		"assignee: Alice",
		"version: 3",
		"Line one",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("formatted issue missing %q:\n%s", want, text)
		}
	}
}

func TestHumanFormatsIssueSearchResult(t *testing.T) {
	t.Parallel()

	text := Human(&model.SearchResult[model.Issue]{
		TotalCount: 2,
		Page:       2,
		Pages:      4,
		PerPage:    25,
		Items: []model.Issue{
			{Key: "DV-1", Summary: "One", Status: &model.Reference{Display: "Open"}},
			{Key: "DV-2", Summary: "Two", Status: &model.Reference{Display: "Closed"}},
		},
	})

	for _, want := range []string{
		"Issues",
		"count: 2",
		"total: 2",
		"page: 2 of 4",
		"per_page: 25",
		"DV-1  One  [Open]",
		"DV-2  Two  [Closed]",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("formatted search result missing %q:\n%s", want, text)
		}
	}
}

func TestHumanFormatsEntitySearchResult(t *testing.T) {
	t.Parallel()

	text := Human(&model.SearchResult[model.Entity]{
		TotalCount: 12,
		Page:       3,
		Pages:      6,
		PerPage:    2,
		Items: []model.Entity{
			{ID: "101", EntityType: "project", ShortID: 7, Fields: map[string]any{"summary": "Platform migration"}},
		},
	})

	for _, want := range []string{
		"Entities",
		"total: 12",
		"page: 3 of 6",
		"PROJECT #7  101  Platform migration",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("formatted entity result missing %q:\n%s", want, text)
		}
	}
}
