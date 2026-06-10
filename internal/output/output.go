package output

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
	"slices"
	"strings"

	"github.com/vastimofeev/yandex-tracker-cli/internal/model"
	"github.com/vastimofeev/yandex-tracker-cli/internal/version"
)

type Printer struct {
	out  io.Writer
	json bool
}

func NewPrinter(out io.Writer, jsonOutput bool) Printer {
	return Printer{out: out, json: jsonOutput}
}

func (p Printer) JSONEnabled() bool {
	return p.json
}

func (p Printer) Print(v any) error {
	if p.json {
		data, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(p.out, string(data))
		return err
	}
	_, err := fmt.Fprintln(p.out, Human(v))
	return err
}

func Human(v any) string {
	switch value := v.(type) {
	case string:
		return value
	case fmt.Stringer:
		return value.String()
	case *model.User:
		return formatUser(value)
	case *model.Issue:
		return formatIssue(value)
	case *model.Comment:
		return formatComment(value)
	case *model.Worklog:
		return formatWorklog(value)
	case *model.Queue:
		return formatQueue(value)
	case *model.Entity:
		return formatEntity(value)
	case *model.ChecklistItem:
		return formatChecklistItem(value)
	case *model.IssueLink:
		return formatIssueLink(value)
	case []model.Transition:
		return formatTransitions(value)
	case []model.Field:
		return formatFields(value)
	case []model.Queue:
		return formatQueues(value)
	case []model.Reference:
		return formatReferences(value)
	case []model.ChecklistItem:
		return formatChecklistItems(value)
	case []model.IssueLink:
		return formatIssueLinks(value)
	case *model.SearchResult[model.Issue]:
		return formatIssueSearchResult(value)
	case *model.SearchResult[model.Comment]:
		return formatCommentSearchResult(value)
	case *model.SearchResult[model.Worklog]:
		return formatWorklogSearchResult(value)
	case *model.SearchResult[model.Entity]:
		return formatEntitySearchResult(value)
	case version.Info:
		return formatVersionInfo(value)
	case *version.Info:
		if value == nil {
			return ""
		}
		return formatVersionInfo(*value)
	case map[string]any:
		if text, ok := formatSpecialMap(value); ok {
			return text
		}
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return strings.TrimSpace(string(data))
}

func formatUser(user *model.User) string {
	if user == nil {
		return ""
	}
	lines := []string{fmt.Sprintf("%s (%s)", firstNonEmpty(user.Display, user.Login, string(user.ID)), firstNonEmpty(user.Login, string(user.ID)))}
	if user.CloudUID != "" {
		lines = append(lines, "cloud_uid: "+user.CloudUID)
	}
	return strings.Join(lines, "\n")
}

func formatIssue(issue *model.Issue) string {
	if issue == nil {
		return ""
	}
	lines := []string{fmt.Sprintf("%s  %s", firstNonEmpty(issue.Key, string(issue.ID)), issue.Summary)}
	lines = appendIf(lines, "status: "+referenceLabel(issue.Status))
	lines = appendIf(lines, "type: "+referenceLabel(issue.Type))
	lines = appendIf(lines, "priority: "+referenceLabel(issue.Priority))
	lines = appendIf(lines, "queue: "+referenceLabel(issue.Queue))
	lines = appendIf(lines, "assignee: "+userLabel(issue.Assignee))
	lines = appendIf(lines, fmt.Sprintf("version: %d", issue.Version))
	lines = appendIf(lines, "created: "+issue.CreatedAt)
	lines = appendIf(lines, "updated: "+issue.UpdatedAt)
	if issue.Description != "" {
		lines = append(lines, "")
		lines = append(lines, issue.Description)
	}
	return strings.Join(lines, "\n")
}

func formatComment(comment *model.Comment) string {
	if comment == nil {
		return ""
	}
	header := fmt.Sprintf("#%d by %s", comment.ID, userLabel(comment.CreatedBy))
	lines := []string{header, comment.Text}
	lines = appendIf(lines, "updated: "+comment.UpdatedAt)
	return strings.Join(lines, "\n")
}

func formatWorklog(entry *model.Worklog) string {
	if entry == nil {
		return ""
	}
	lines := []string{fmt.Sprintf("#%d  %s", entry.ID, entry.Duration)}
	lines = appendIf(lines, "start: "+entry.Start)
	lines = appendIf(lines, "author: "+userLabel(entry.CreatedBy))
	lines = appendIf(lines, "comment: "+entry.Comment)
	return strings.Join(lines, "\n")
}

func formatQueue(queue *model.Queue) string {
	if queue == nil {
		return ""
	}
	lines := []string{fmt.Sprintf("%s  %s", queue.Key, queue.Name)}
	lines = appendIf(lines, "lead: "+userLabel(queue.Lead))
	lines = appendIf(lines, "default type: "+referenceLabel(queue.DefaultType))
	lines = appendIf(lines, "default priority: "+referenceLabel(queue.DefaultPriority))
	if len(queue.IssueTypes) > 0 {
		lines = append(lines, "issue types: "+joinReferences(queue.IssueTypes))
	}
	if queue.Description != "" {
		lines = append(lines, "")
		lines = append(lines, queue.Description)
	}
	return strings.Join(lines, "\n")
}

func formatEntity(entity *model.Entity) string {
	if entity == nil {
		return ""
	}
	lines := []string{fmt.Sprintf("%s #%d", strings.ToUpper(entity.EntityType), entity.ShortID)}
	lines = appendIf(lines, "id: "+string(entity.ID))
	lines = appendIf(lines, "version: "+fmt.Sprintf("%d", entity.Version))
	lines = appendIf(lines, "created: "+entity.CreatedAt)
	lines = appendIf(lines, "updated: "+entity.UpdatedAt)
	if len(entity.Fields) > 0 {
		keys := make([]string, 0, len(entity.Fields))
		for key := range entity.Fields {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		lines = append(lines, "fields: "+strings.Join(keys, ", "))
	}
	return strings.Join(lines, "\n")
}

func formatChecklistItem(item *model.ChecklistItem) string {
	if item == nil {
		return ""
	}
	check := "[ ]"
	if item.Checked {
		check = "[x]"
	}
	lines := []string{fmt.Sprintf("%s %s (%s)", check, item.Text, item.ID)}
	lines = appendIf(lines, "assignee: "+userLabel(item.Assignee))
	if item.Deadline != nil {
		lines = appendIf(lines, "deadline: "+item.Deadline.Date)
	}
	return strings.Join(lines, "\n")
}

func formatIssueLink(link *model.IssueLink) string {
	if link == nil {
		return ""
	}
	target := ""
	if link.Object != nil {
		target = firstNonEmpty(link.Object.Key, link.Object.Display, string(link.Object.ID))
	}
	return strings.TrimSpace(fmt.Sprintf("%s  %s  %s", firstNonEmpty(linkTypeID(link), link.Direction), firstNonEmpty(linkTypeLabel(link), ""), target))
}

func formatTransitions(items []model.Transition) string {
	if len(items) == 0 {
		return "no transitions"
	}
	lines := listHeader("Transitions", len(items))
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("%s -> %s", item.ID, referenceLabel(item.To)))
	}
	return strings.Join(lines, "\n")
}

func formatFields(items []model.Field) string {
	if len(items) == 0 {
		return "no fields"
	}
	lines := listHeader("Fields", len(items))
	for _, item := range items {
		required := ""
		if isRequired(item.Schema) {
			required = " required"
		}
		readonly := ""
		if item.ReadOnly {
			readonly = " readonly"
		}
		lines = append(lines, fmt.Sprintf("%s  %s  (%s%s%s)", item.Key, item.Name, item.Type, required, readonly))
	}
	return strings.Join(lines, "\n")
}

func formatQueues(items []model.Queue) string {
	if len(items) == 0 {
		return "no queues"
	}
	lines := listHeader("Queues", len(items))
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("%s  %s", item.Key, item.Name))
	}
	return strings.Join(lines, "\n")
}

func formatReferences(items []model.Reference) string {
	if len(items) == 0 {
		return "no items"
	}
	lines := listHeader("Items", len(items))
	for _, item := range items {
		lines = append(lines, fmt.Sprintf("%s  %s", firstNonEmpty(item.Key, string(item.ID)), firstNonEmpty(item.Display, item.Key, string(item.ID))))
	}
	return strings.Join(lines, "\n")
}

func formatChecklistItems(items []model.ChecklistItem) string {
	if len(items) == 0 {
		return "no checklist items"
	}
	lines := listHeader("Checklist Items", len(items))
	for _, item := range items {
		lines = append(lines, formatChecklistItem(&item))
	}
	return strings.Join(lines, "\n")
}

func formatIssueLinks(items []model.IssueLink) string {
	if len(items) == 0 {
		return "no links"
	}
	lines := listHeader("Issue Links", len(items))
	for _, item := range items {
		lines = append(lines, formatIssueLink(&item))
	}
	return strings.Join(lines, "\n")
}

func formatIssueSearchResult(result *model.SearchResult[model.Issue]) string {
	if result == nil || len(result.Items) == 0 {
		return "no issues"
	}
	lines := pagedHeader("Issues", result)
	for _, item := range result.Items {
		lines = append(lines, fmt.Sprintf("%s  %s  [%s]", item.Key, item.Summary, referenceLabel(item.Status)))
	}
	return strings.Join(lines, "\n")
}

func formatCommentSearchResult(result *model.SearchResult[model.Comment]) string {
	if result == nil || len(result.Items) == 0 {
		return "no comments"
	}
	lines := pagedHeader("Comments", result)
	for _, item := range result.Items {
		lines = append(lines, fmt.Sprintf("#%d  %s: %s", item.ID, userLabel(item.CreatedBy), trimLine(item.Text)))
	}
	return strings.Join(lines, "\n")
}

func formatWorklogSearchResult(result *model.SearchResult[model.Worklog]) string {
	if result == nil || len(result.Items) == 0 {
		return "no worklog entries"
	}
	lines := pagedHeader("Worklog Entries", result)
	for _, item := range result.Items {
		lines = append(lines, fmt.Sprintf("#%d  %s  %s", item.ID, item.Duration, trimLine(item.Comment)))
	}
	return strings.Join(lines, "\n")
}

func formatEntitySearchResult(result *model.SearchResult[model.Entity]) string {
	if result == nil || len(result.Items) == 0 {
		return "no entities"
	}
	lines := pagedHeader("Entities", result)
	for _, item := range result.Items {
		title := entityTitle(item)
		if title != "" {
			lines = append(lines, fmt.Sprintf("%s #%d  %s  %s", strings.ToUpper(item.EntityType), item.ShortID, string(item.ID), title))
			continue
		}
		lines = append(lines, fmt.Sprintf("%s #%d  %s", strings.ToUpper(item.EntityType), item.ShortID, string(item.ID)))
	}
	return strings.Join(lines, "\n")
}

func formatSpecialMap(value map[string]any) (string, bool) {
	keys := sortedKeys(value)
	switch {
	case reflect.DeepEqual(keys, []string{"count", "filter", "query"}):
		return fmt.Sprintf("count: %v\nquery: %v", value["count"], value["query"]), true
	case reflect.DeepEqual(keys, []string{"issue", "itemID", "status"}):
		return fmt.Sprintf("%v checklist item %v", value["status"], value["itemID"]), true
	case reflect.DeepEqual(keys, []string{"authStore", "baseURL", "orgHeader", "orgID", "status", "user"}):
		return fmt.Sprintf("status: %v\nbase_url: %v\norg: %v (%v)\nauth_store: %v", value["status"], value["baseURL"], value["orgID"], value["orgHeader"], value["authStore"]), true
	case reflect.DeepEqual(keys, []string{"authStore", "baseURL", "configured", "orgHeader", "orgID", "tokenType"}):
		return fmt.Sprintf("configured: %v\nbase_url: %v\norg: %v (%v)\ntoken_type: %v\nauth_store: %v", value["configured"], value["baseURL"], value["orgID"], value["orgHeader"], value["tokenType"], value["authStore"]), true
	case reflect.DeepEqual(keys, []string{"authStore", "baseURL", "configured", "orgHeader", "orgID", "orgPresent", "savedConfigured", "savedOrgHeader", "savedOrgID", "savedOrgPresent", "savedTokenPresent", "savedTokenType", "tokenPresent", "tokenType"}):
		return fmt.Sprintf(
			"current_configured: %v\ncurrent_token_present: %v\ncurrent_org: %v (%v)\ncurrent_org_present: %v\ncurrent_token_type: %v\nsaved_configured: %v\nsaved_token_present: %v\nsaved_org: %v (%v)\nsaved_org_present: %v\nsaved_token_type: %v\nbase_url: %v\nauth_store: %v",
			value["configured"], value["tokenPresent"], value["orgID"], value["orgHeader"], value["orgPresent"], value["tokenType"],
			value["savedConfigured"], value["savedTokenPresent"], value["savedOrgID"], value["savedOrgHeader"], value["savedOrgPresent"], value["savedTokenType"],
			value["baseURL"], value["authStore"],
		), true
	case reflect.DeepEqual(keys, []string{"issue", "transitions"}):
		issueText := Human(value["issue"])
		transitionsText := Human(value["transitions"])
		return strings.TrimSpace(issueText + "\n\nnext transitions:\n" + transitionsText), true
	default:
		return "", false
	}
}

func referenceLabel(ref *model.Reference) string {
	if ref == nil {
		return ""
	}
	return firstNonEmpty(ref.Display, ref.Key, string(ref.ID))
}

func userLabel(user *model.User) string {
	if user == nil {
		return ""
	}
	return firstNonEmpty(user.Display, user.Login, string(user.ID))
}

func joinReferences(items []model.Reference) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		parts = append(parts, firstNonEmpty(item.Display, item.Key, string(item.ID)))
	}
	return strings.Join(parts, ", ")
}

func appendIf(lines []string, line string) []string {
	if strings.TrimSpace(line) == "" {
		return lines
	}
	return append(lines, line)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func trimLine(text string) string {
	text = strings.TrimSpace(text)
	text = strings.ReplaceAll(text, "\n", " ")
	if len(text) > 80 {
		return text[:77] + "..."
	}
	return text
}

func displayCount(total, length int) int {
	if total > 0 {
		return total
	}
	return length
}

func sortedKeys(value map[string]any) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	slices.Sort(keys)
	return keys
}

func isRequired(schema map[string]any) bool {
	if schema == nil {
		return false
	}
	required, ok := schema["required"].(bool)
	return ok && required
}

func linkTypeID(link *model.IssueLink) string {
	if link == nil || link.Type == nil {
		return ""
	}
	return link.Type.ID
}

func linkTypeLabel(link *model.IssueLink) string {
	if link == nil || link.Type == nil {
		return ""
	}
	return firstNonEmpty(link.Type.Outward, link.Type.Inward, link.Type.ID)
}

func formatVersionInfo(info version.Info) string {
	lines := []string{
		"version: " + info.Version,
		"commit: " + info.Commit,
		"build_date: " + info.BuildDate,
		"go: " + info.GoVersion,
		"platform: " + info.Platform,
	}
	return strings.Join(lines, "\n")
}

func listHeader(title string, count int) []string {
	return []string{
		title,
		fmt.Sprintf("count: %d", count),
		"",
	}
}

func pagedHeader[T any](title string, result *model.SearchResult[T]) []string {
	lines := []string{
		title,
		fmt.Sprintf("count: %d", len(result.Items)),
		fmt.Sprintf("total: %d", displayCount(result.TotalCount, len(result.Items))),
	}
	if result.Page > 0 {
		if result.Pages > 0 {
			lines = append(lines, fmt.Sprintf("page: %d of %d", result.Page, result.Pages))
		} else {
			lines = append(lines, fmt.Sprintf("page: %d", result.Page))
		}
	} else if result.Pages > 0 {
		lines = append(lines, fmt.Sprintf("pages: %d", result.Pages))
	}
	if result.PerPage > 0 {
		lines = append(lines, fmt.Sprintf("per_page: %d", result.PerPage))
	}
	if strings.TrimSpace(result.ScrollID) != "" {
		lines = append(lines, "scroll_id: "+result.ScrollID)
	}
	lines = append(lines, "")
	return lines
}

func entityTitle(entity model.Entity) string {
	if len(entity.Fields) == 0 {
		return ""
	}
	for _, key := range []string{"summary", "name", "title", "display", "key"} {
		if value, ok := entity.Fields[key]; ok {
			if text := stringifyValue(value); text != "" {
				return text
			}
		}
	}
	return ""
}

func stringifyValue(v any) string {
	switch value := v.(type) {
	case string:
		return strings.TrimSpace(value)
	case fmt.Stringer:
		return strings.TrimSpace(value.String())
	default:
		text := strings.TrimSpace(fmt.Sprintf("%v", value))
		if text == "<nil>" {
			return ""
		}
		return text
	}
}
