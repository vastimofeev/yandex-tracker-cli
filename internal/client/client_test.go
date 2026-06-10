package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/vastimofeev/yandex-tracker-cli/internal/config"
)

func TestNewRequestSetsAuthAndOrgHeaders(t *testing.T) {
	t.Parallel()

	c := New("https://api.example.test/v3", config.AuthContext{
		Token:     "abc123",
		TokenType: "bearer",
		OrgID:     "org-id",
		OrgHeader: "x-cloud-org-id",
	}, nil)

	req, err := c.newRequest(context.Background(), http.MethodGet, "/myself", nil, nil)
	if err != nil {
		t.Fatalf("newRequest() error = %v", err)
	}

	if got := req.Header.Get("Authorization"); got != "Bearer abc123" {
		t.Fatalf("Authorization = %q, want %q", got, "Bearer abc123")
	}
	if got := req.Header.Get("X-Cloud-Org-ID"); got != "org-id" {
		t.Fatalf("X-Cloud-Org-ID = %q, want %q", got, "org-id")
	}
}

func TestSearchIssuesBuildsScrollQueryAndBody(t *testing.T) {
	t.Parallel()

	var capturedQuery url.Values
	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query()
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.Header().Set("X-Total-Count", "42")
		w.Header().Set("X-Scroll-Id", "scroll-123")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	c := New(server.URL, config.AuthContext{}, server.Client())
	result, err := c.SearchIssues(context.Background(), SearchIssuesRequest{
		Filter:          map[string]any{"queue": "TREK", "assignee": "alice"},
		PerPage:         50,
		ScrollType:      "sorted",
		PerScroll:       100,
		ScrollTTLMillis: 10000,
		ScrollID:        "next-page",
	})
	if err != nil {
		t.Fatalf("SearchIssues() error = %v", err)
	}

	if capturedQuery.Get("perPage") != "50" {
		t.Fatalf("perPage query = %q, want 50", capturedQuery.Get("perPage"))
	}
	if capturedQuery.Get("scrollType") != "sorted" {
		t.Fatalf("scrollType query = %q, want sorted", capturedQuery.Get("scrollType"))
	}
	if capturedQuery.Get("perScroll") != "100" {
		t.Fatalf("perScroll query = %q, want 100", capturedQuery.Get("perScroll"))
	}
	if capturedQuery.Get("scrollTTLMillis") != "10000" {
		t.Fatalf("scrollTTLMillis query = %q, want 10000", capturedQuery.Get("scrollTTLMillis"))
	}
	if capturedQuery.Get("scrollId") != "next-page" {
		t.Fatalf("scrollId query = %q, want next-page", capturedQuery.Get("scrollId"))
	}
	filter, ok := capturedBody["filter"].(map[string]any)
	if !ok {
		t.Fatalf("filter body missing or wrong type: %#v", capturedBody["filter"])
	}
	if filter["queue"] != "TREK" {
		t.Fatalf("filter.queue = %#v, want TREK", filter["queue"])
	}
	if result.TotalCount != 42 || result.ScrollID != "scroll-123" {
		t.Fatalf("result = %#v, want TotalCount=42 ScrollID=scroll-123", result)
	}
}

func TestDecodeResponseMapsStatusCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantType   any
	}{
		{name: "bad request", statusCode: http.StatusBadRequest, wantType: &ValidationError{}},
		{name: "unauthorized", statusCode: http.StatusUnauthorized, wantType: &AuthError{}},
		{name: "forbidden", statusCode: http.StatusForbidden, wantType: &AuthError{}},
		{name: "not found", statusCode: http.StatusNotFound, wantType: &NotFoundError{}},
		{name: "server error", statusCode: http.StatusInternalServerError, wantType: &APIError{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, tt.name, tt.statusCode)
			}))
			defer server.Close()

			c := New(server.URL, config.AuthContext{}, server.Client())
			err := c.doJSON(context.Background(), http.MethodGet, "/anything", nil, nil, nil)
			if err == nil {
				t.Fatalf("expected error for status %d", tt.statusCode)
			}
			switch tt.wantType.(type) {
			case *ValidationError:
				if _, ok := err.(*ValidationError); !ok {
					t.Fatalf("error type = %T, want ValidationError", err)
				}
			case *AuthError:
				if _, ok := err.(*AuthError); !ok {
					t.Fatalf("error type = %T, want AuthError", err)
				}
			case *NotFoundError:
				if _, ok := err.(*NotFoundError); !ok {
					t.Fatalf("error type = %T, want NotFoundError", err)
				}
			case *APIError:
				if _, ok := err.(*APIError); !ok {
					t.Fatalf("error type = %T, want APIError", err)
				}
			}
		})
	}
}

func TestCountIssuesBuildsRequestBody(t *testing.T) {
	t.Parallel()

	var capturedBody map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/issues/_count" {
			t.Fatalf("path = %s, want /issues/_count", r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`7`))
	}))
	defer server.Close()

	c := New(server.URL, config.AuthContext{}, server.Client())
	count, err := c.CountIssues(context.Background(), IssuesCountRequest{
		Query:  "Queue: DV",
		Filter: map[string]any{"assignee": "alice"},
	})
	if err != nil {
		t.Fatalf("CountIssues() error = %v", err)
	}
	if count != 7 {
		t.Fatalf("count = %d, want 7", count)
	}
	if capturedBody["query"] != "Queue: DV" {
		t.Fatalf("query = %#v, want Queue: DV", capturedBody["query"])
	}
	filter, ok := capturedBody["filter"].(map[string]any)
	if !ok || filter["assignee"] != "alice" {
		t.Fatalf("filter = %#v, want assignee=alice", capturedBody["filter"])
	}
}

func TestChecklistAndLinksPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		run    func(*Client) error
		method string
		path   string
	}{
		{
			name: "list links",
			run: func(c *Client) error {
				_, err := c.ListLinks(context.Background(), "DV-1")
				return err
			},
			method: http.MethodGet,
			path:   "/issues/DV-1/links",
		},
		{
			name: "add link",
			run: func(c *Client) error {
				_, err := c.AddLink(context.Background(), "DV-1", LinkCreateRequest{Relationship: "relates", Issue: "DV-2"})
				return err
			},
			method: http.MethodPost,
			path:   "/issues/DV-1/links",
		},
		{
			name: "list checklist",
			run: func(c *Client) error {
				_, err := c.ListChecklist(context.Background(), "DV-1")
				return err
			},
			method: http.MethodGet,
			path:   "/issues/DV-1/checklistItems",
		},
		{
			name: "add checklist item",
			run: func(c *Client) error {
				_, err := c.AddChecklistItem(context.Background(), "DV-1", ChecklistItemRequest{Text: "todo"})
				return err
			},
			method: http.MethodPost,
			path:   "/issues/DV-1/checklistItems",
		},
		{
			name: "update checklist item",
			run: func(c *Client) error {
				checked := true
				_, err := c.UpdateChecklistItem(context.Background(), "DV-1", "11", ChecklistItemRequest{Checked: &checked})
				return err
			},
			method: http.MethodPatch,
			path:   "/issues/DV-1/checklistItems/11",
		},
		{
			name: "delete checklist item",
			run: func(c *Client) error {
				return c.DeleteChecklistItem(context.Background(), "DV-1", "11")
			},
			method: http.MethodDelete,
			path:   "/issues/DV-1/checklistItems/11",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != tt.method {
					t.Fatalf("method = %s, want %s", r.Method, tt.method)
				}
				if r.URL.Path != tt.path {
					t.Fatalf("path = %s, want %s", r.URL.Path, tt.path)
				}
				w.WriteHeader(http.StatusOK)
				switch tt.method {
				case http.MethodGet:
					_, _ = w.Write([]byte(`[]`))
				case http.MethodDelete:
				case http.MethodPost:
					if tt.path == "/issues/DV-1/checklistItems" {
						_, _ = w.Write([]byte(`{"key":"DV-1","checklistItems":[{"id":"11","text":"todo"}]}`))
					} else {
						_, _ = w.Write([]byte(`{}`))
					}
				default:
					if tt.path == "/issues/DV-1/checklistItems/11" {
						_, _ = w.Write([]byte(`{"key":"DV-1","checklistItems":[{"id":"11","text":"todo","checked":true}]}`))
					} else {
						_, _ = w.Write([]byte(`{}`))
					}
				}
			}))
			defer server.Close()

			c := New(server.URL, config.AuthContext{}, server.Client())
			if err := tt.run(c); err != nil {
				t.Fatalf("%s error = %v", tt.name, err)
			}
		})
	}
}

func TestDictionaryPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path string
		run  func(*Client) error
	}{
		{
			name: "global fields",
			path: "/fields",
			run: func(c *Client) error {
				_, err := c.ListGlobalFields(context.Background())
				return err
			},
		},
		{
			name: "issue types",
			path: "/issuetypes",
			run: func(c *Client) error {
				_, err := c.ListIssueTypes(context.Background())
				return err
			},
		},
		{
			name: "statuses",
			path: "/statuses",
			run: func(c *Client) error {
				_, err := c.ListStatuses(context.Background())
				return err
			},
		},
		{
			name: "priorities",
			path: "/priorities",
			run: func(c *Client) error {
				_, err := c.ListPriorities(context.Background())
				return err
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodGet {
					t.Fatalf("method = %s, want GET", r.Method)
				}
				if r.URL.Path != tt.path {
					t.Fatalf("path = %s, want %s", r.URL.Path, tt.path)
				}
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`[]`))
			}))
			defer server.Close()

			c := New(server.URL, config.AuthContext{}, server.Client())
			if err := tt.run(c); err != nil {
				t.Fatalf("%s error = %v", tt.name, err)
			}
		})
	}
}

func TestListEntitiesUsesQueryPagination(t *testing.T) {
	t.Parallel()

	var (
		capturedQuery url.Values
		capturedBody  map[string]any
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Fatalf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/entities/project/_search" {
			t.Fatalf("path = %s, want /entities/project/_search", r.URL.Path)
		}
		capturedQuery = r.URL.Query()
		if err := json.NewDecoder(r.Body).Decode(&capturedBody); err != nil {
			t.Fatalf("decode body: %v", err)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"hits":3,"pages":2,"values":[]}`))
	}))
	defer server.Close()

	c := New(server.URL, config.AuthContext{}, server.Client())
	result, err := c.ListEntities(context.Background(), "project", map[string]any{"input": "demo"}, 3, 2)
	if err != nil {
		t.Fatalf("ListEntities() error = %v", err)
	}
	if capturedQuery.Get("perPage") != "3" {
		t.Fatalf("perPage = %q, want 3", capturedQuery.Get("perPage"))
	}
	if capturedQuery.Get("page") != "2" {
		t.Fatalf("page = %q, want 2", capturedQuery.Get("page"))
	}
	if capturedBody["input"] != "demo" {
		t.Fatalf("input = %#v, want demo", capturedBody["input"])
	}
	if result.TotalCount != 3 {
		t.Fatalf("TotalCount = %d, want 3", result.TotalCount)
	}
	if result.Pages != 2 {
		t.Fatalf("Pages = %d, want 2", result.Pages)
	}
}
