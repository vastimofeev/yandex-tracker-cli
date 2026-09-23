package model

import (
	"encoding/json"
	"testing"
)

func TestIssueParsesStoryPointsResolutionSprintDeadline(t *testing.T) {
	t.Parallel()

	payload := `{
		"key": "LIS-62",
		"summary": "closed issue with estimates",
		"storyPoints": 5,
		"resolution": {"id": "1", "key": "fixed", "display": "Решен"},
		"sprint": [{"id": "296", "display": "Спринт 4"}],
		"deadline": "2026-09-30"
	}`

	var issue Issue
	if err := json.Unmarshal([]byte(payload), &issue); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if issue.StoryPoints == nil || *issue.StoryPoints != 5 {
		t.Fatalf("storyPoints = %v, want 5", issue.StoryPoints)
	}
	if issue.Resolution == nil || issue.Resolution.Key != "fixed" {
		t.Fatalf("resolution = %+v, want fixed", issue.Resolution)
	}
	if len(issue.Sprint) != 1 || issue.Sprint[0].Display != "Спринт 4" {
		t.Fatalf("sprint = %+v, want one item", issue.Sprint)
	}
	if issue.Deadline != "2026-09-30" {
		t.Fatalf("deadline = %q", issue.Deadline)
	}

	// The fields must survive re-marshaling (this is what --json prints).
	out, err := json.Marshal(issue)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	roundTrip := map[string]any{}
	if err := json.Unmarshal(out, &roundTrip); err != nil {
		t.Fatalf("unmarshal round trip: %v", err)
	}
	for _, field := range []string{"storyPoints", "resolution", "sprint", "deadline"} {
		if _, ok := roundTrip[field]; !ok {
			t.Fatalf("field %q missing from round-tripped JSON: %s", field, out)
		}
	}
}

func TestIssueWithoutOptionalsOmitsFields(t *testing.T) {
	t.Parallel()

	var issue Issue
	if err := json.Unmarshal([]byte(`{"key":"X-1"}`), &issue); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if issue.StoryPoints != nil || issue.Resolution != nil || issue.Sprint != nil || issue.Deadline != "" {
		t.Fatalf("optional fields should stay zero-valued, got %+v", issue)
	}
}
