package cli

import (
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/vasti/yandex-tracker-cli/internal/model"
)

func presentIssueSearch(result *model.SearchResult[model.Issue], sortField, order string, limit int, selectFields []string) (any, error) {
	if result == nil {
		return result, nil
	}

	if sortField != "" {
		desc := strings.EqualFold(strings.TrimSpace(order), "desc")
		if order != "" && !desc && !strings.EqualFold(strings.TrimSpace(order), "asc") {
			return nil, fmt.Errorf("invalid order %q, expected asc or desc", order)
		}
		slices.SortStableFunc(result.Items, func(a, b model.Issue) int {
			left := issueFieldValue(a, sortField)
			right := issueFieldValue(b, sortField)
			cmp := compareFieldValues(left, right)
			if desc {
				return -cmp
			}
			return cmp
		})
	}

	if limit > 0 && len(result.Items) > limit {
		result.Items = result.Items[:limit]
	}

	if len(selectFields) == 0 {
		return result, nil
	}

	rows := make([]map[string]any, 0, len(result.Items))
	for _, item := range result.Items {
		row := map[string]any{}
		for _, field := range selectFields {
			field = strings.TrimSpace(field)
			if field == "" {
				continue
			}
			row[field] = issueFieldValue(item, field)
		}
		rows = append(rows, row)
	}

	return map[string]any{
		"items":         rows,
		"selectedFields": selectFields,
		"totalCount":    result.TotalCount,
		"pages":         result.Pages,
		"page":          result.Page,
		"perPage":       result.PerPage,
		"scrollId":      result.ScrollID,
	}, nil
}

func issueFieldValue(issue model.Issue, path string) any {
	data := map[string]any{}
	raw, err := json.Marshal(issue)
	if err != nil {
		return nil
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil
	}
	return fieldValueByPath(data, path)
}

func fieldValueByPath(data map[string]any, path string) any {
	current := any(data)
	for _, part := range strings.Split(path, ".") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil
		}
		asMap, ok := current.(map[string]any)
		if !ok {
			return nil
		}
		value, ok := asMap[part]
		if !ok {
			return nil
		}
		current = value
	}
	return current
}

func compareFieldValues(left, right any) int {
	if left == nil && right == nil {
		return 0
	}
	if left == nil {
		return 1
	}
	if right == nil {
		return -1
	}

	if lf, lok := numericValue(left); lok {
		if rf, rok := numericValue(right); rok {
			switch {
			case lf < rf:
				return -1
			case lf > rf:
				return 1
			default:
				return 0
			}
		}
	}

	ls := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", left)))
	rs := strings.ToLower(strings.TrimSpace(fmt.Sprintf("%v", right)))
	switch {
	case ls < rs:
		return -1
	case ls > rs:
		return 1
	default:
		return 0
	}
}

func numericValue(v any) (float64, bool) {
	switch value := v.(type) {
	case float64:
		return value, true
	case float32:
		return float64(value), true
	case int:
		return float64(value), true
	case int64:
		return float64(value), true
	case json.Number:
		f, err := value.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(value), 64)
		return f, err == nil
	default:
		return 0, false
	}
}
