package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"

	"github.com/vastimofeev/yandex-tracker-cli/internal/config"
	"sync/atomic"
	"testing"
)

func TestPostIsNotRetriedAfterServerError(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	c := New(server.URL, config.AuthContext{}, server.Client())

	_, err := c.AddLink(context.Background(), "TREK-1", LinkCreateRequest{Relationship: "relates", Issue: "TREK-2"})
	if err == nil {
		t.Fatal("expected error")
	}
	if got := calls.Load(); got != 1 {
		t.Fatalf("POST /issues/{key}/links was sent %d times, want 1 (not repeatable)", got)
	}
}

func TestSearchPostIsRetriedAfterServerError(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) >= 2 {
			_ = json.NewEncoder(w).Encode([]map[string]any{{"key": "TREK-1"}})
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c := New(server.URL, config.AuthContext{}, server.Client())

	_, err := c.SearchIssues(context.Background(), SearchIssuesRequest{Query: "Queue: TREK"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("POST /issues/_search was sent %d times, want 2 (retry on 5xx)", got)
	}
}

func TestGetIsRetriedAfterServerError(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) >= 2 {
			_, _ = w.Write([]byte(`{"key":"TREK-1"}`))
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	c := New(server.URL, config.AuthContext{}, server.Client())

	if _, err := c.GetIssue(context.Background(), "TREK-1"); err != nil {
		t.Fatalf("get issue: %v", err)
	}
	if got := calls.Load(); got != 2 {
		t.Fatalf("GET was sent %d times, want 2 (retry on 5xx)", got)
	}
}
