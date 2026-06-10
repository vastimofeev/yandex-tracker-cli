package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vastimofeev/yandex-tracker-cli/internal/client"
	"github.com/vastimofeev/yandex-tracker-cli/internal/config"
)

func TestEditIssueRejectsStatusField(t *testing.T) {
	t.Parallel()

	svc := New(client.New("https://api.example.test/v3", config.AuthContext{}, nil))
	_, err := svc.EditIssue(context.Background(), "TREK-1", "", "", []string{"status=resolved"}, "")
	if err == nil {
		t.Fatal("expected error when editing status directly")
	}
	if err != errStatusFieldUpdate {
		t.Fatalf("error = %v, want %v", err, errStatusFieldUpdate)
	}
}

func TestCreateIssueBuildsMarkupPayload(t *testing.T) {
	t.Parallel()

	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/queues/TREK/fields":
			_, _ = w.Write([]byte(`[]`))
		case "/queues/TREK":
			_, _ = w.Write([]byte(`{"key":"TREK","defaultType":{"key":"task"}}`))
		case "/issues/":
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"key":"TREK-1","summary":"Demo"}`))
		default:
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	svc := New(client.New(server.URL, config.AuthContext{}, server.Client()))
	issue, err := svc.CreateIssue(context.Background(), "TREK", "task", "Demo", "Hello", []string{"tags=one,two", "estimate=3"}, "")
	if err != nil {
		t.Fatalf("CreateIssue() error = %v", err)
	}
	if issue.Key != "TREK-1" {
		t.Fatalf("issue.Key = %q, want TREK-1", issue.Key)
	}
	if payload["markupType"] != "md" {
		t.Fatalf("markupType = %#v, want md", payload["markupType"])
	}
	tags, ok := payload["tags"].([]any)
	if !ok || len(tags) != 2 {
		t.Fatalf("tags payload = %#v, want 2-item array", payload["tags"])
	}
	if payload["estimate"] != float64(3) && payload["estimate"] != 3 {
		t.Fatalf("estimate payload = %#v, want 3", payload["estimate"])
	}
}

func TestAddCommentBuildsExpectedPayload(t *testing.T) {
	t.Parallel()

	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":12,"text":"hello"}`))
	}))
	defer server.Close()

	svc := New(client.New(server.URL, config.AuthContext{}, server.Client()))
	addToFollowers := false
	_, err := svc.AddComment(context.Background(), "TREK-1", "hello", []string{"alice", "1001"}, &addToFollowers)
	if err != nil {
		t.Fatalf("AddComment() error = %v", err)
	}

	if payload["markupType"] != "md" {
		t.Fatalf("markupType = %#v, want md", payload["markupType"])
	}
	if payload["isAddToFollowers"] != false {
		t.Fatalf("isAddToFollowers = %#v, want false", payload["isAddToFollowers"])
	}
	summonees, ok := payload["summonees"].([]any)
	if !ok || len(summonees) != 2 {
		t.Fatalf("summonees payload = %#v, want 2-item array", payload["summonees"])
	}
}

func TestAddLinkBuildsExpectedPayload(t *testing.T) {
	t.Parallel()

	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":1}`))
	}))
	defer server.Close()

	svc := New(client.New(server.URL, config.AuthContext{}, server.Client()))
	_, err := svc.AddLink(context.Background(), "DV-1", "relates", "DV-2")
	if err != nil {
		t.Fatalf("AddLink() error = %v", err)
	}

	if payload["relationship"] != "relates" {
		t.Fatalf("relationship = %#v, want relates", payload["relationship"])
	}
	if payload["issue"] != "DV-2" {
		t.Fatalf("issue = %#v, want DV-2", payload["issue"])
	}
}

func TestAddChecklistItemBuildsExpectedPayload(t *testing.T) {
	t.Parallel()

	var payload []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"key":"DV-1","checklistItems":[{"id":"11","text":"Write docs","checked":true}]}`))
	}))
	defer server.Close()

	svc := New(client.New(server.URL, config.AuthContext{}, server.Client()))
	_, err := svc.AddChecklistItem(context.Background(), "DV-1", "Write docs", "alice", "2026-04-30", true)
	if err != nil {
		t.Fatalf("AddChecklistItem() error = %v", err)
	}

	if len(payload) != 1 {
		t.Fatalf("payload len = %d, want 1", len(payload))
	}
	if payload[0]["text"] != "Write docs" {
		t.Fatalf("text = %#v, want Write docs", payload[0]["text"])
	}
	if payload[0]["assignee"] != "alice" {
		t.Fatalf("assignee = %#v, want alice", payload[0]["assignee"])
	}
	deadline, ok := payload[0]["deadline"].(map[string]any)
	if !ok || deadline["date"] != "2026-04-30" || deadline["deadlineType"] != "date" {
		t.Fatalf("deadline = %#v, want date=2026-04-30 deadlineType=date", payload[0]["deadline"])
	}
	if payload[0]["checked"] != true {
		t.Fatalf("checked = %#v, want true", payload[0]["checked"])
	}
}

func TestSetChecklistItemCheckedBuildsExpectedPayload(t *testing.T) {
	t.Parallel()

	var (
		payload   map[string]any
		listCalls int
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/issues/DV-1/checklistItems":
			listCalls++
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`[{"id":"11","text":"Write docs","checked":false}]`))
		case r.Method == http.MethodPatch && r.URL.Path == "/issues/DV-1/checklistItems/11":
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				t.Fatalf("decode body: %v", err)
			}
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"key":"DV-1","checklistItems":[{"id":"11","text":"Write docs","checked":true}]}`))
		default:
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
	}))
	defer server.Close()

	svc := New(client.New(server.URL, config.AuthContext{}, server.Client()))
	_, err := svc.SetChecklistItemChecked(context.Background(), "DV-1", "11", true)
	if err != nil {
		t.Fatalf("SetChecklistItemChecked() error = %v", err)
	}

	if listCalls != 1 {
		t.Fatalf("listCalls = %d, want 1", listCalls)
	}
	if payload["checked"] != true {
		t.Fatalf("checked = %#v, want true", payload["checked"])
	}
	if payload["text"] != "Write docs" {
		t.Fatalf("text = %#v, want Write docs", payload["text"])
	}
}

func TestAddWorklogDefaultsStartTime(t *testing.T) {
	t.Parallel()

	var payload map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"id":1,"duration":"PT1M"}`))
	}))
	defer server.Close()

	svc := New(client.New(server.URL, config.AuthContext{}, server.Client()))
	_, err := svc.AddWorklog(context.Background(), "DV-1", "PT1M", "validation", "")
	if err != nil {
		t.Fatalf("AddWorklog() error = %v", err)
	}

	if payload["duration"] != "PT1M" {
		t.Fatalf("duration = %#v, want PT1M", payload["duration"])
	}
	start, ok := payload["start"].(string)
	if !ok || start == "" {
		t.Fatalf("start = %#v, want non-empty string", payload["start"])
	}
}

func TestCreateIssueValidatesRequiredQueueFields(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/queues/DV/fields":
			_, _ = w.Write([]byte(`[{"key":"queue","schema":{"required":true},"readonly":true},{"key":"summary","schema":{"required":true}},{"key":"customField","schema":{"required":true}}]`))
		case "/queues/DV":
			_, _ = w.Write([]byte(`{"key":"DV","defaultType":{"key":"task"}}`))
		default:
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	svc := New(client.New(server.URL, config.AuthContext{}, server.Client()))
	_, err := svc.CreateIssue(context.Background(), "DV", "task", "Demo", "", nil, "")
	if err == nil {
		t.Fatal("expected validation error")
	}
	if got := err.Error(); got != `queue DV requires field "customField"` {
		t.Fatalf("error = %q, want missing customField", got)
	}
}

func TestCreateIssueValidatesAllowedQueueType(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/queues/DV":
			_, _ = w.Write([]byte(`{"key":"DV","issueTypes":[{"id":"2","key":"task","display":"Task"}]}`))
		case "/queues/DV/fields":
			_, _ = w.Write([]byte(`[]`))
		default:
			t.Fatalf("unexpected request: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	svc := New(client.New(server.URL, config.AuthContext{}, server.Client()))
	_, err := svc.CreateIssue(context.Background(), "DV", "bug", "Demo", "", nil, "")
	if err == nil {
		t.Fatal("expected validation error")
	}
	if got := err.Error(); got != `issue type "bug" is not allowed in queue DV` {
		t.Fatalf("error = %q, want invalid type error", got)
	}
}
