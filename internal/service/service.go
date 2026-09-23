package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/vastimofeev/yandex-tracker-cli/internal/client"
	"github.com/vastimofeev/yandex-tracker-cli/internal/config"
	"github.com/vastimofeev/yandex-tracker-cli/internal/model"
)

var errStatusFieldUpdate = errors.New("issue status cannot be edited directly; use issue transition")

type TrackerService struct {
	client *client.Client
}

func New(c *client.Client) *TrackerService {
	return &TrackerService{client: c}
}

func (s *TrackerService) ValidateAuth(ctx context.Context) (*model.User, error) {
	return s.client.GetMyself(ctx)
}

func (s *TrackerService) GetIssue(ctx context.Context, key string) (*model.Issue, error) {
	return s.client.GetIssue(ctx, key)
}

func (s *TrackerService) SearchIssues(ctx context.Context, query string, filters []string, keys []string, perPage int, scrollType string, perScroll int, scrollTTL int, scrollID string) (*model.SearchResult[model.Issue], error) {
	filterMap, err := parseKVList(filters)
	if err != nil {
		return nil, err
	}
	req := client.SearchIssuesRequest{
		Query:           query,
		Keys:            keys,
		Filter:          filterMap,
		PerPage:         perPage,
		ScrollType:      scrollType,
		PerScroll:       perScroll,
		ScrollTTLMillis: scrollTTL,
		ScrollID:        scrollID,
	}
	return s.client.SearchIssues(ctx, req)
}

func (s *TrackerService) CountIssues(ctx context.Context, query string, filters []string) (map[string]any, error) {
	filterMap, err := parseKVList(filters)
	if err != nil {
		return nil, err
	}
	req := client.IssuesCountRequest{
		Filter: filterMap,
		Query:  query,
	}
	count, err := s.client.CountIssues(ctx, req)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"query":  query,
		"filter": filterMap,
		"count":  count,
	}, nil
}

func (s *TrackerService) CreateIssue(ctx context.Context, queue, issueType, summary, description string, fields []string, bodyJSON string) (*model.Issue, error) {
	payload, err := buildPayload(fields, bodyJSON)
	if err != nil {
		return nil, err
	}
	if queue != "" {
		payload["queue"] = queue
	}
	if issueType != "" {
		payload["type"] = issueType
	}
	if summary != "" {
		payload["summary"] = summary
	}
	if description != "" {
		payload["description"] = description
		payload["markupType"] = "md"
	}
	if err := s.validateCreateIssuePayload(ctx, queue, payload); err != nil {
		return nil, err
	}
	return s.client.CreateIssue(ctx, payload)
}

func (s *TrackerService) EditIssue(ctx context.Context, key, summary, description string, fields []string, bodyJSON string) (*model.Issue, error) {
	payload, err := buildPayload(fields, bodyJSON)
	if err != nil {
		return nil, err
	}
	if _, hasStatus := payload["status"]; hasStatus {
		return nil, errStatusFieldUpdate
	}
	if summary != "" {
		payload["summary"] = summary
	}
	if description != "" {
		payload["description"] = description
		payload["markupType"] = "md"
	}
	if err := s.validateEditIssuePayload(ctx, key, payload); err != nil {
		return nil, err
	}
	return s.client.EditIssue(ctx, key, payload)
}

func (s *TrackerService) ListTransitions(ctx context.Context, key string) ([]model.Transition, error) {
	return s.client.ListTransitions(ctx, key)
}

func (s *TrackerService) ExecuteTransition(ctx context.Context, key, to, id, comment string, fields []string, bodyJSON string) (map[string]any, error) {
	transitions, err := s.client.ListTransitions(ctx, key)
	if err != nil {
		return nil, err
	}
	transitionID, err := resolveTransitionID(transitions, id, to)
	if err != nil {
		return nil, err
	}
	payload, err := buildPayload(fields, bodyJSON)
	if err != nil {
		return nil, err
	}
	if comment != "" {
		payload["comment"] = comment
	}
	if strings.TrimSpace(transitionID) == "" {
		return nil, fmt.Errorf("transition is required; pass --id or --to")
	}
	nextTransitions, err := s.client.ExecuteTransition(ctx, key, transitionID, payload)
	if err != nil {
		return nil, err
	}
	issue, err := s.client.GetIssue(ctx, key)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"issue":       issue,
		"transitions": nextTransitions,
	}, nil
}

func (s *TrackerService) ListComments(ctx context.Context, key string, perPage, page int) (*model.SearchResult[model.Comment], error) {
	return s.client.ListComments(ctx, key, perPage, page)
}

func (s *TrackerService) AddComment(ctx context.Context, key, text string, summonees []string, addToFollowers *bool) (*model.Comment, error) {
	req := client.CommentCreateRequest{
		Text:             text,
		MarkupType:       "md",
		Summonees:        summonees,
		IsAddToFollowers: addToFollowers,
	}
	return s.client.AddComment(ctx, key, req)
}

func (s *TrackerService) EditComment(ctx context.Context, key string, commentID int64, text string, summonees []string, version int) (*model.Comment, error) {
	req := client.CommentEditRequest{
		Text:       text,
		MarkupType: "md",
		Summonees:  summonees,
	}
	_ = version
	return s.client.EditComment(ctx, key, commentID, req)
}

func (s *TrackerService) ListWorklog(ctx context.Context, key string, perPage, page int) (*model.SearchResult[model.Worklog], error) {
	return s.client.ListWorklog(ctx, key, perPage, page)
}

func (s *TrackerService) AddWorklog(ctx context.Context, key, duration, comment, start string) (*model.Worklog, error) {
	if strings.TrimSpace(start) == "" {
		start = time.Now().Format(time.RFC3339)
	}
	return s.client.AddWorklog(ctx, key, client.WorklogCreateRequest{
		Start:    start,
		Duration: duration,
		Comment:  comment,
	})
}

func (s *TrackerService) ListLinks(ctx context.Context, key string) ([]model.IssueLink, error) {
	return s.client.ListLinks(ctx, key)
}

func (s *TrackerService) AddLink(ctx context.Context, key, relationship, issue string) (*model.IssueLink, error) {
	if strings.TrimSpace(relationship) == "" {
		return nil, fmt.Errorf("relationship is required")
	}
	if strings.TrimSpace(issue) == "" {
		return nil, fmt.Errorf("issue is required")
	}

	resolved, err := ResolveLinkRelationship(relationship)
	if err != nil {
		return nil, err
	}

	// Direction-swapping aliases ("blocks") post the link from the blocked
	// side, because the API has no direct "source blocks target" value.
	from, target := key, issue
	if resolved.Swap {
		from, target = issue, key
	}
	return s.client.AddLink(ctx, from, client.LinkCreateRequest{
		Relationship: resolved.Value,
		Issue:        target,
	})
}

func (s *TrackerService) ListChecklist(ctx context.Context, key string) ([]model.ChecklistItem, error) {
	return s.client.ListChecklist(ctx, key)
}

func (s *TrackerService) AddChecklistItem(ctx context.Context, key, text, assignee, deadline string, checked bool) ([]model.ChecklistItem, error) {
	if strings.TrimSpace(text) == "" {
		return nil, fmt.Errorf("text is required")
	}
	payload := client.ChecklistItemRequest{
		Text:     text,
		Checked:  boolPtr(checked),
		Assignee: assignee,
		Deadline: buildChecklistDeadline(deadline),
	}
	issue, err := s.client.AddChecklistItem(ctx, key, payload)
	if err != nil {
		return nil, err
	}
	return issue.ChecklistItems, nil
}

func (s *TrackerService) UpdateChecklistItem(ctx context.Context, key, itemID, text, assignee, deadline string, checked *bool) (*model.ChecklistItem, error) {
	current, err := s.resolveChecklistItem(ctx, key, itemID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(text) == "" {
		text = current.Text
	}
	payload := client.ChecklistItemRequest{
		Text:     text,
		Checked:  checked,
		Assignee: assignee,
		Deadline: buildChecklistDeadline(deadline),
	}
	issue, err := s.client.UpdateChecklistItem(ctx, key, itemID, payload)
	if err != nil {
		return nil, err
	}
	return checklistItemByID(issue.ChecklistItems, itemID)
}

func (s *TrackerService) SetChecklistItemChecked(ctx context.Context, key, itemID string, checked bool) (*model.ChecklistItem, error) {
	current, err := s.resolveChecklistItem(ctx, key, itemID)
	if err != nil {
		return nil, err
	}
	issue, err := s.client.UpdateChecklistItem(ctx, key, itemID, client.ChecklistItemRequest{
		Text:    current.Text,
		Checked: boolPtr(checked),
	})
	if err != nil {
		return nil, err
	}
	return checklistItemByID(issue.ChecklistItems, itemID)
}

func (s *TrackerService) DeleteChecklistItem(ctx context.Context, key, itemID string) (map[string]string, error) {
	if err := s.client.DeleteChecklistItem(ctx, key, itemID); err != nil {
		return nil, err
	}
	return map[string]string{
		"status": "deleted",
		"issue":  key,
		"itemID": itemID,
	}, nil
}

func (s *TrackerService) ListQueues(ctx context.Context, perPage, page int) ([]model.Queue, error) {
	return s.client.ListQueues(ctx, perPage, page)
}

func (s *TrackerService) GetQueue(ctx context.Context, key string, expand []string) (*model.Queue, error) {
	return s.client.GetQueue(ctx, key, expand)
}

func (s *TrackerService) GetQueueFields(ctx context.Context, key string) ([]model.Field, error) {
	return s.client.GetQueueFields(ctx, key)
}

func (s *TrackerService) GetQueueLocalFields(ctx context.Context, key string) ([]model.Field, error) {
	return s.client.GetQueueLocalFields(ctx, key)
}

func (s *TrackerService) ListGlobalFields(ctx context.Context) ([]model.Field, error) {
	return s.client.ListGlobalFields(ctx)
}

func (s *TrackerService) ListIssueTypes(ctx context.Context) ([]model.Reference, error) {
	return s.client.ListIssueTypes(ctx)
}

func (s *TrackerService) ListStatuses(ctx context.Context) ([]model.Reference, error) {
	return s.client.ListStatuses(ctx)
}

func (s *TrackerService) ListPriorities(ctx context.Context) ([]model.Reference, error) {
	return s.client.ListPriorities(ctx)
}

func (s *TrackerService) ListEntities(ctx context.Context, entityType, query string, fields []string, perPage, page int) (*model.SearchResult[model.Entity], error) {
	payload, err := buildPayload(fields, "")
	if err != nil {
		return nil, err
	}
	if query != "" {
		payload["input"] = query
	}
	return s.client.ListEntities(ctx, entityType, payload, perPage, page)
}

func (s *TrackerService) GetEntity(ctx context.Context, entityType, id string) (*model.Entity, error) {
	return s.client.GetEntity(ctx, entityType, id)
}

func buildPayload(fields []string, bodyJSON string) (map[string]any, error) {
	payload := map[string]any{}
	if strings.TrimSpace(bodyJSON) != "" {
		if err := json.Unmarshal([]byte(bodyJSON), &payload); err != nil {
			return nil, fmt.Errorf("parse body json: %w", err)
		}
	}
	fieldMap, err := parseKVList(fields)
	if err != nil {
		return nil, err
	}
	for key, value := range fieldMap {
		payload[key] = value
	}
	return payload, nil
}

func parseKVList(items []string) (map[string]any, error) {
	result := map[string]any{}
	for _, item := range items {
		key, value, ok := strings.Cut(item, "=")
		if !ok || strings.TrimSpace(key) == "" {
			return nil, fmt.Errorf("expected key=value, got %q", item)
		}
		result[strings.TrimSpace(key)] = parseScalar(strings.TrimSpace(value))
	}
	return result, nil
}

func parseScalar(value string) any {
	if value == "" {
		return ""
	}
	switch strings.ToLower(value) {
	case "true":
		return true
	case "false":
		return false
	case "null":
		return nil
	}
	if n, err := strconv.Atoi(value); err == nil {
		return n
	}
	if strings.Contains(value, ",") {
		parts := strings.Split(value, ",")
		out := make([]any, 0, len(parts))
		for _, part := range parts {
			out = append(out, parseScalar(strings.TrimSpace(part)))
		}
		return out
	}
	return value
}

func resolveTransitionID(transitions []model.Transition, id, to string) (string, error) {
	if id != "" {
		return id, nil
	}
	for _, transition := range transitions {
		if strings.EqualFold(transition.ID, to) ||
			strings.EqualFold(transition.Display, to) ||
			(transition.To != nil && (strings.EqualFold(transition.To.Key, to) || strings.EqualFold(string(transition.To.ID), to) || strings.EqualFold(transition.To.Display, to))) {
			return transition.ID, nil
		}
	}
	return "", fmt.Errorf("transition %q not found", to)
}

func boolPtr(v bool) *bool {
	return &v
}

func buildChecklistDeadline(deadline string) *client.ChecklistDeadlineRequest {
	if strings.TrimSpace(deadline) == "" {
		return nil
	}
	return &client.ChecklistDeadlineRequest{
		Date:         deadline,
		DeadlineType: "date",
	}
}

func (s *TrackerService) resolveChecklistItem(ctx context.Context, key, itemID string) (*model.ChecklistItem, error) {
	items, err := s.client.ListChecklist(ctx, key)
	if err != nil {
		return nil, err
	}
	return checklistItemByID(items, itemID)
}

func checklistItemByID(items []model.ChecklistItem, itemID string) (*model.ChecklistItem, error) {
	for i := range items {
		if items[i].ID == itemID {
			return &items[i], nil
		}
	}
	return nil, fmt.Errorf("checklist item %q not found", itemID)
}

func (s *TrackerService) validateCreateIssuePayload(ctx context.Context, queue string, payload map[string]any) error {
	if strings.TrimSpace(queue) == "" {
		return fmt.Errorf("queue is required")
	}
	if err := s.validateIssueTypeForQueue(ctx, queue, payload["type"]); err != nil {
		return err
	}
	fields, err := s.client.GetQueueFields(ctx, queue)
	if err != nil {
		return err
	}
	queueMeta, err := s.client.GetQueue(ctx, queue, []string{"types"})
	if err != nil {
		return err
	}
	for _, field := range fields {
		if !isRequiredField(field) || field.ReadOnly {
			continue
		}
		if hasUsableValue(payload[field.Key]) {
			continue
		}
		if field.Key == "type" && queueMeta.DefaultType != nil {
			continue
		}
		if field.Key == "priority" && queueMeta.DefaultPriority != nil {
			continue
		}
		return fmt.Errorf("queue %s requires field %q", queue, field.Key)
	}
	return nil
}

func (s *TrackerService) validateEditIssuePayload(ctx context.Context, key string, payload map[string]any) error {
	typeValue, ok := payload["type"]
	if !ok {
		return nil
	}
	queue := ""
	if queueValue, hasQueue := payload["queue"]; hasQueue {
		queue = stringifyScalar(queueValue)
	}
	if queue == "" {
		issue, err := s.client.GetIssue(ctx, key)
		if err != nil {
			return err
		}
		if issue.Queue != nil {
			queue = issue.Queue.Key
		}
	}
	if queue == "" {
		return fmt.Errorf("cannot resolve queue for issue %s", key)
	}
	return s.validateIssueTypeForQueue(ctx, queue, typeValue)
}

func (s *TrackerService) validateIssueTypeForQueue(ctx context.Context, queue string, typeValue any) error {
	value := stringifyScalar(typeValue)
	if value == "" {
		return nil
	}
	queueMeta, err := s.client.GetQueue(ctx, queue, []string{"types"})
	if err != nil {
		return err
	}
	if len(queueMeta.IssueTypes) == 0 {
		return nil
	}
	for _, item := range queueMeta.IssueTypes {
		if strings.EqualFold(item.Key, value) || strings.EqualFold(string(item.ID), value) || strings.EqualFold(item.Display, value) {
			return nil
		}
	}
	return fmt.Errorf("issue type %q is not allowed in queue %s", value, queue)
}

func isRequiredField(field model.Field) bool {
	if field.Schema == nil {
		return false
	}
	required, ok := field.Schema["required"].(bool)
	return ok && required
}

func hasUsableValue(v any) bool {
	if v == nil {
		return false
	}
	switch value := v.(type) {
	case string:
		return strings.TrimSpace(value) != ""
	case []any:
		return len(value) > 0
	case []string:
		return len(value) > 0
	default:
		return true
	}
}

func stringifyScalar(v any) string {
	switch value := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(value)
	case fmt.Stringer:
		return strings.TrimSpace(value.String())
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", value))
	}
}

func BuildStoredConfig(runtime config.RuntimeConfig) config.StoredConfig {
	return config.StoredConfig{
		BaseURL:   runtime.BaseURL,
		Auth:      runtime.Auth,
		AuthStore: runtime.AuthStore,
		OAuth:     runtime.OAuth,
	}
}
