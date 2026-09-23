package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/vastimofeev/yandex-tracker-cli/internal/config"
	"github.com/vastimofeev/yandex-tracker-cli/internal/model"
)

type Client struct {
	baseURL    string
	auth       config.AuthContext
	httpClient *http.Client
	retries    int
}

type SearchIssuesRequest struct {
	Queue           string
	Keys            []string
	Filter          map[string]any
	Query           string
	Order           string
	PerPage         int
	ScrollType      string
	PerScroll       int
	ScrollTTLMillis int
	ScrollID        string
	Expand          []string
}

type CommentCreateRequest struct {
	Text             string   `json:"text"`
	MarkupType       string   `json:"markupType,omitempty"`
	Summonees        []string `json:"summonees,omitempty"`
	IsAddToFollowers *bool    `json:"isAddToFollowers,omitempty"`
}

type CommentEditRequest struct {
	Text       string   `json:"text,omitempty"`
	MarkupType string   `json:"markupType,omitempty"`
	Summonees  []string `json:"summonees,omitempty"`
}

type WorklogCreateRequest struct {
	Start    string `json:"start,omitempty"`
	Duration string `json:"duration"`
	Comment  string `json:"comment,omitempty"`
}

type IssuesCountRequest struct {
	Filter map[string]any `json:"filter,omitempty"`
	Query  string         `json:"query,omitempty"`
}

type LinkCreateRequest struct {
	Relationship string `json:"relationship"`
	Issue        string `json:"issue"`
}

type ChecklistDeadlineRequest struct {
	Date         string `json:"date,omitempty"`
	DeadlineType string `json:"deadlineType,omitempty"`
}

type ChecklistItemRequest struct {
	Text     string                    `json:"text,omitempty"`
	Checked  *bool                     `json:"checked,omitempty"`
	Assignee string                    `json:"assignee,omitempty"`
	Deadline *ChecklistDeadlineRequest `json:"deadline,omitempty"`
}

func New(baseURL string, auth config.AuthContext, httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		auth:       auth,
		httpClient: httpClient,
		retries:    2,
	}
}

func (c *Client) GetMyself(ctx context.Context) (*model.User, error) {
	var user model.User
	if err := c.doJSON(ctx, http.MethodGet, "/myself", nil, nil, &user); err != nil {
		return nil, err
	}
	return &user, nil
}

func (c *Client) GetIssue(ctx context.Context, key string) (*model.Issue, error) {
	var issue model.Issue
	if err := c.doJSON(ctx, http.MethodGet, "/issues/"+url.PathEscape(key), nil, nil, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

func (c *Client) SearchIssues(ctx context.Context, req SearchIssuesRequest) (*model.SearchResult[model.Issue], error) {
	query := url.Values{}
	if req.PerPage > 0 {
		query.Set("perPage", strconv.Itoa(req.PerPage))
	}
	if len(req.Expand) > 0 {
		query.Set("expand", strings.Join(req.Expand, ","))
	}
	if req.ScrollType != "" {
		query.Set("scrollType", req.ScrollType)
	}
	if req.PerScroll > 0 {
		query.Set("perScroll", strconv.Itoa(req.PerScroll))
	}
	if req.ScrollTTLMillis > 0 {
		query.Set("scrollTTLMillis", strconv.Itoa(req.ScrollTTLMillis))
	}
	if req.ScrollID != "" {
		query.Set("scrollId", req.ScrollID)
	}

	body := map[string]any{}
	switch {
	case req.Queue != "":
		body["queue"] = req.Queue
	case len(req.Keys) > 0:
		body["keys"] = req.Keys
	case len(req.Filter) > 0:
		body["filter"] = req.Filter
	case req.Query != "":
		body["query"] = req.Query
	default:
		return nil, &ValidationError{Message: "search requires one of queue, keys, filter, or query"}
	}
	if req.Order != "" {
		body["order"] = req.Order
	}

	var issues []model.Issue
	headers, err := c.doJSONWithHeaders(ctx, http.MethodPost, "/issues/_search", query, body, &issues)
	if err != nil {
		return nil, err
	}
	result := &model.SearchResult[model.Issue]{Items: issues, PerPage: req.PerPage}
	if total := headers.Get("X-Total-Count"); total != "" {
		if parsed, parseErr := strconv.Atoi(total); parseErr == nil {
			result.TotalCount = parsed
		}
	}
	result.ScrollID = firstNonEmpty(headers.Get("X-Scroll-Id"), headers.Get("X-inScroll-Id"))
	return result, nil
}

func (c *Client) CountIssues(ctx context.Context, req IssuesCountRequest) (int, error) {
	var count int
	if err := c.doJSON(ctx, http.MethodPost, "/issues/_count", nil, req, &count); err != nil {
		return 0, err
	}
	return count, nil
}

func (c *Client) CreateIssue(ctx context.Context, payload map[string]any) (*model.Issue, error) {
	var issue model.Issue
	if err := c.doJSON(ctx, http.MethodPost, "/issues/", nil, payload, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

func (c *Client) EditIssue(ctx context.Context, key string, payload map[string]any) (*model.Issue, error) {
	var issue model.Issue
	if err := c.doJSON(ctx, http.MethodPatch, "/issues/"+url.PathEscape(key), nil, payload, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

func (c *Client) ListTransitions(ctx context.Context, key string) ([]model.Transition, error) {
	var transitions []model.Transition
	if err := c.doJSON(ctx, http.MethodGet, "/issues/"+url.PathEscape(key)+"/transitions", nil, nil, &transitions); err != nil {
		return nil, err
	}
	return transitions, nil
}

func (c *Client) ExecuteTransition(ctx context.Context, key, transitionID string, payload map[string]any) ([]model.Transition, error) {
	var transitions []model.Transition
	path := fmt.Sprintf("/issues/%s/transitions/%s/_execute", url.PathEscape(key), url.PathEscape(transitionID))
	if err := c.doJSON(ctx, http.MethodPost, path, nil, payload, &transitions); err != nil {
		return nil, err
	}
	return transitions, nil
}

func (c *Client) ListComments(ctx context.Context, key string, perPage, page int) (*model.SearchResult[model.Comment], error) {
	query := url.Values{}
	if perPage > 0 {
		query.Set("perPage", strconv.Itoa(perPage))
	}
	if page > 0 {
		query.Set("page", strconv.Itoa(page))
	}
	var comments []model.Comment
	headers, err := c.doJSONWithHeaders(ctx, http.MethodGet, "/issues/"+url.PathEscape(key)+"/comments", query, nil, &comments)
	if err != nil {
		return nil, err
	}
	result := &model.SearchResult[model.Comment]{Items: comments, Page: page, PerPage: perPage}
	if total := headers.Get("X-Total-Count"); total != "" {
		if parsed, parseErr := strconv.Atoi(total); parseErr == nil {
			result.TotalCount = parsed
		}
	}
	return result, nil
}

func (c *Client) AddComment(ctx context.Context, key string, payload CommentCreateRequest) (*model.Comment, error) {
	var comment model.Comment
	if err := c.doJSON(ctx, http.MethodPost, "/issues/"+url.PathEscape(key)+"/comments", nil, payload, &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}

func (c *Client) EditComment(ctx context.Context, key string, commentID int64, payload CommentEditRequest) (*model.Comment, error) {
	var comment model.Comment
	path := fmt.Sprintf("/issues/%s/comments/%d", url.PathEscape(key), commentID)
	if err := c.doJSON(ctx, http.MethodPatch, path, nil, payload, &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}

func (c *Client) ListWorklog(ctx context.Context, key string, perPage, page int) (*model.SearchResult[model.Worklog], error) {
	query := url.Values{}
	if perPage > 0 {
		query.Set("perPage", strconv.Itoa(perPage))
	}
	if page > 0 {
		query.Set("page", strconv.Itoa(page))
	}
	var entries []model.Worklog
	headers, err := c.doJSONWithHeaders(ctx, http.MethodGet, "/issues/"+url.PathEscape(key)+"/worklog", query, nil, &entries)
	if err != nil {
		return nil, err
	}
	result := &model.SearchResult[model.Worklog]{Items: entries, Page: page, PerPage: perPage}
	if total := headers.Get("X-Total-Count"); total != "" {
		if parsed, parseErr := strconv.Atoi(total); parseErr == nil {
			result.TotalCount = parsed
		}
	}
	return result, nil
}

func (c *Client) AddWorklog(ctx context.Context, key string, payload WorklogCreateRequest) (*model.Worklog, error) {
	var entry model.Worklog
	if err := c.doJSON(ctx, http.MethodPost, "/issues/"+url.PathEscape(key)+"/worklog", nil, payload, &entry); err != nil {
		return nil, err
	}
	return &entry, nil
}

func (c *Client) ListLinks(ctx context.Context, key string) ([]model.IssueLink, error) {
	var links []model.IssueLink
	if err := c.doJSON(ctx, http.MethodGet, "/issues/"+url.PathEscape(key)+"/links", nil, nil, &links); err != nil {
		return nil, err
	}
	return links, nil
}

func (c *Client) AddLink(ctx context.Context, key string, payload LinkCreateRequest) (*model.IssueLink, error) {
	var link model.IssueLink
	if err := c.doJSON(ctx, http.MethodPost, "/issues/"+url.PathEscape(key)+"/links", nil, payload, &link); err != nil {
		return nil, err
	}
	return &link, nil
}

func (c *Client) DeleteLink(ctx context.Context, key, linkID string) error {
	return c.doJSON(ctx, http.MethodDelete, "/issues/"+url.PathEscape(key)+"/links/"+url.PathEscape(linkID), nil, nil, nil)
}

func (c *Client) ListChecklist(ctx context.Context, key string) ([]model.ChecklistItem, error) {
	var items []model.ChecklistItem
	if err := c.doJSON(ctx, http.MethodGet, "/issues/"+url.PathEscape(key)+"/checklistItems", nil, nil, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *Client) AddChecklistItem(ctx context.Context, key string, payload ChecklistItemRequest) (*model.Issue, error) {
	var issue model.Issue
	if err := c.doJSON(ctx, http.MethodPost, "/issues/"+url.PathEscape(key)+"/checklistItems", nil, []ChecklistItemRequest{payload}, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

func (c *Client) UpdateChecklistItem(ctx context.Context, key, itemID string, payload ChecklistItemRequest) (*model.Issue, error) {
	var issue model.Issue
	path := fmt.Sprintf("/issues/%s/checklistItems/%s", url.PathEscape(key), url.PathEscape(itemID))
	if err := c.doJSON(ctx, http.MethodPatch, path, nil, payload, &issue); err != nil {
		return nil, err
	}
	return &issue, nil
}

func (c *Client) DeleteChecklistItem(ctx context.Context, key, itemID string) error {
	path := fmt.Sprintf("/issues/%s/checklistItems/%s", url.PathEscape(key), url.PathEscape(itemID))
	return c.doJSON(ctx, http.MethodDelete, path, nil, nil, nil)
}

func (c *Client) ListQueues(ctx context.Context, perPage, page int) ([]model.Queue, error) {
	query := url.Values{}
	if perPage > 0 {
		query.Set("perPage", strconv.Itoa(perPage))
	}
	if page > 0 {
		query.Set("page", strconv.Itoa(page))
	}
	var queues []model.Queue
	if err := c.doJSON(ctx, http.MethodGet, "/queues/", query, nil, &queues); err != nil {
		return nil, err
	}
	return queues, nil
}

func (c *Client) GetQueue(ctx context.Context, key string, expand []string) (*model.Queue, error) {
	query := url.Values{}
	if len(expand) > 0 {
		query.Set("expand", strings.Join(expand, ","))
	}
	var queue model.Queue
	if err := c.doJSON(ctx, http.MethodGet, "/queues/"+url.PathEscape(key), query, nil, &queue); err != nil {
		return nil, err
	}
	return &queue, nil
}

func (c *Client) GetQueueFields(ctx context.Context, key string) ([]model.Field, error) {
	var fields []model.Field
	if err := c.doJSON(ctx, http.MethodGet, "/queues/"+url.PathEscape(key)+"/fields", nil, nil, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func (c *Client) GetQueueLocalFields(ctx context.Context, key string) ([]model.Field, error) {
	var fields []model.Field
	if err := c.doJSON(ctx, http.MethodGet, "/queues/"+url.PathEscape(key)+"/localFields", nil, nil, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func (c *Client) ListGlobalFields(ctx context.Context) ([]model.Field, error) {
	var fields []model.Field
	if err := c.doJSON(ctx, http.MethodGet, "/fields", nil, nil, &fields); err != nil {
		return nil, err
	}
	return fields, nil
}

func (c *Client) ListIssueTypes(ctx context.Context) ([]model.Reference, error) {
	var items []model.Reference
	if err := c.doJSON(ctx, http.MethodGet, "/issuetypes", nil, nil, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *Client) ListStatuses(ctx context.Context) ([]model.Reference, error) {
	var items []model.Reference
	if err := c.doJSON(ctx, http.MethodGet, "/statuses", nil, nil, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *Client) ListPriorities(ctx context.Context) ([]model.Reference, error) {
	var items []model.Reference
	if err := c.doJSON(ctx, http.MethodGet, "/priorities", nil, nil, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (c *Client) ListEntities(ctx context.Context, entityType string, payload map[string]any, perPage, page int) (*model.SearchResult[model.Entity], error) {
	query := url.Values{}
	if perPage > 0 {
		query.Set("perPage", strconv.Itoa(perPage))
	}
	if page > 0 {
		query.Set("page", strconv.Itoa(page))
	}
	var response struct {
		Hits   int            `json:"hits"`
		Pages  int            `json:"pages"`
		Values []model.Entity `json:"values"`
	}
	headers, err := c.doJSONWithHeaders(ctx, http.MethodPost, "/entities/"+url.PathEscape(entityType)+"/_search", query, payload, &response)
	if err != nil {
		return nil, err
	}
	result := &model.SearchResult[model.Entity]{
		Items:      response.Values,
		TotalCount: response.Hits,
		Pages:      response.Pages,
		Page:       page,
		PerPage:    perPage,
	}
	if total := headers.Get("X-Total-Count"); total != "" {
		if parsed, parseErr := strconv.Atoi(total); parseErr == nil {
			result.TotalCount = parsed
		}
	}
	return result, nil
}

func (c *Client) GetEntity(ctx context.Context, entityType, id string) (*model.Entity, error) {
	var entity model.Entity
	path := fmt.Sprintf("/entities/%s/%s", url.PathEscape(entityType), url.PathEscape(id))
	if err := c.doJSON(ctx, http.MethodGet, path, nil, nil, &entity); err != nil {
		return nil, err
	}
	return &entity, nil
}

func (c *Client) doJSON(ctx context.Context, method, path string, query url.Values, body any, out any) error {
	_, err := c.doJSONWithHeaders(ctx, method, path, query, body, out)
	return err
}

func (c *Client) doJSONWithHeaders(ctx context.Context, method, path string, query url.Values, body any, out any) (http.Header, error) {
	var payload []byte
	var err error
	if body != nil {
		payload, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	var headers http.Header
	for attempt := 0; attempt <= c.retries; attempt++ {
		req, err := c.newRequest(ctx, method, path, query, payload)
		if err != nil {
			return nil, err
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if isRetryable(err) && attempt < c.retries {
				continue
			}
			return nil, &NetworkError{Err: err}
		}

		headers = resp.Header.Clone()
		respErr := decodeResponse(resp, out)
		resp.Body.Close()
		if apiErr, ok := respErr.(*APIError); ok && apiErr.StatusCode >= 500 && attempt < c.retries {
			continue
		}
		return headers, respErr
	}

	return headers, nil
}

func (c *Client) newRequest(ctx context.Context, method, path string, query url.Values, payload []byte) (*http.Request, error) {
	target := c.baseURL + path
	if len(query) > 0 {
		target += "?" + query.Encode()
	}
	body := io.Reader(http.NoBody)
	if payload != nil {
		body = bytes.NewReader(payload)
	}
	req, err := http.NewRequestWithContext(ctx, method, target, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.auth.Token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("%s %s", config.NormalizeTokenType(c.auth.TokenType), c.auth.Token))
	}
	if c.auth.OrgID != "" && c.auth.OrgHeader != "" {
		req.Header.Set(config.NormalizeOrgHeader(c.auth.OrgHeader), c.auth.OrgID)
	}
	return req, nil
}

func decodeResponse(resp *http.Response, out any) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode >= 400 {
		message := strings.TrimSpace(string(body))
		switch resp.StatusCode {
		case http.StatusBadRequest:
			return &ValidationError{Message: firstNonEmpty(message, "request validation failed")}
		case http.StatusUnauthorized, http.StatusForbidden:
			return &AuthError{Message: firstNonEmpty(message, "authorization failed")}
		case http.StatusNotFound:
			return &NotFoundError{Message: firstNonEmpty(message, "resource not found")}
		default:
			return &APIError{StatusCode: resp.StatusCode, Message: message}
		}
	}
	if out == nil || len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	return json.Unmarshal(body, out)
}

func isRetryable(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	return false
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
