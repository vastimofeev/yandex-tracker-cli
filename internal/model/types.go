package model

type Reference struct {
	Self    string   `json:"self,omitempty"`
	ID      StringID `json:"id,omitempty"`
	Key     string   `json:"key,omitempty"`
	Display string   `json:"display,omitempty"`
}

type User struct {
	Self       string   `json:"self,omitempty"`
	ID         StringID `json:"id,omitempty"`
	Display    string   `json:"display,omitempty"`
	Login      string   `json:"login,omitempty"`
	PassportID int64    `json:"passportUid,omitempty"`
	CloudUID   string   `json:"cloudUid,omitempty"`
}

type ProjectRef struct {
	Primary   *Reference  `json:"primary,omitempty"`
	Secondary []Reference `json:"secondary,omitempty"`
}

type Issue struct {
	Self           string          `json:"self,omitempty"`
	ID             StringID        `json:"id,omitempty"`
	Key            string          `json:"key,omitempty"`
	Version        int             `json:"version,omitempty"`
	Summary        string          `json:"summary,omitempty"`
	Description    string          `json:"description,omitempty"`
	CreatedAt      string          `json:"createdAt,omitempty"`
	UpdatedAt      string          `json:"updatedAt,omitempty"`
	Queue          *Reference      `json:"queue,omitempty"`
	Status         *Reference      `json:"status,omitempty"`
	PreviousStatus *Reference      `json:"previousStatus,omitempty"`
	Type           *Reference      `json:"type,omitempty"`
	Priority       *Reference      `json:"priority,omitempty"`
	Parent         *Reference      `json:"parent,omitempty"`
	Assignee       *User           `json:"assignee,omitempty"`
	CreatedBy      *User           `json:"createdBy,omitempty"`
	UpdatedBy      *User           `json:"updatedBy,omitempty"`
	Followers      []User          `json:"followers,omitempty"`
	Tags           []string        `json:"tags,omitempty"`
	Project        *ProjectRef     `json:"project,omitempty"`
	ChecklistItems []ChecklistItem `json:"checklistItems,omitempty"`
	ChecklistTotal int             `json:"checklistTotal,omitempty"`
	ChecklistDone  int             `json:"checklistDone,omitempty"`
	Raw            map[string]any  `json:"-"`
}

type Transition struct {
	ID      string     `json:"id,omitempty"`
	Display string     `json:"display,omitempty"`
	To      *Reference `json:"to,omitempty"`
	Screen  any        `json:"screen,omitempty"`
}

type Comment struct {
	Self             string `json:"self,omitempty"`
	ID               int64  `json:"id,omitempty"`
	LongID           string `json:"longId,omitempty"`
	Text             string `json:"text,omitempty"`
	Version          int    `json:"version,omitempty"`
	CreatedAt        string `json:"createdAt,omitempty"`
	UpdatedAt        string `json:"updatedAt,omitempty"`
	CreatedBy        *User  `json:"createdBy,omitempty"`
	UpdatedBy        *User  `json:"updatedBy,omitempty"`
	Summonees        []User `json:"summonees,omitempty"`
	IsAddToFollowers bool   `json:"isAddToFollowers,omitempty"`
}

type ChecklistDeadline struct {
	Date         string `json:"date,omitempty"`
	DeadlineType string `json:"deadlineType,omitempty"`
	IsExceeded   bool   `json:"isExceeded,omitempty"`
}

type ChecklistItem struct {
	ID                string             `json:"id,omitempty"`
	Text              string             `json:"text,omitempty"`
	TextHTML          string             `json:"textHtml,omitempty"`
	Checked           bool               `json:"checked,omitempty"`
	Assignee          *User              `json:"assignee,omitempty"`
	Deadline          *ChecklistDeadline `json:"deadline,omitempty"`
	ChecklistItemType string             `json:"checklistItemType,omitempty"`
}

type LinkType struct {
	Self    string `json:"self,omitempty"`
	ID      string `json:"id,omitempty"`
	Inward  string `json:"inward,omitempty"`
	Outward string `json:"outward,omitempty"`
}

type IssueLink struct {
	Self      string     `json:"self,omitempty"`
	ID        int64      `json:"id,omitempty"`
	Type      *LinkType  `json:"type,omitempty"`
	Direction string     `json:"direction,omitempty"`
	Object    *Reference `json:"object,omitempty"`
	CreatedBy *User      `json:"createdBy,omitempty"`
	UpdatedBy *User      `json:"updatedBy,omitempty"`
	CreatedAt string     `json:"createdAt,omitempty"`
	UpdatedAt string     `json:"updatedAt,omitempty"`
	Assignee  *User      `json:"assignee,omitempty"`
	Status    *Reference `json:"status,omitempty"`
}

type Worklog struct {
	ID        int64  `json:"id,omitempty"`
	Comment   string `json:"comment,omitempty"`
	Start     string `json:"start,omitempty"`
	Duration  string `json:"duration,omitempty"`
	CreatedAt string `json:"createdAt,omitempty"`
	CreatedBy *User  `json:"createdBy,omitempty"`
}

type Queue struct {
	Self            string      `json:"self,omitempty"`
	ID              StringID    `json:"id,omitempty"`
	Key             string      `json:"key,omitempty"`
	Name            string      `json:"name,omitempty"`
	Description     string      `json:"description,omitempty"`
	Version         int         `json:"version,omitempty"`
	Lead            *User       `json:"lead,omitempty"`
	DefaultType     *Reference  `json:"defaultType,omitempty"`
	DefaultPriority *Reference  `json:"defaultPriority,omitempty"`
	IssueTypes      []Reference `json:"issueTypes,omitempty"`
}

type Field struct {
	Self        string         `json:"self,omitempty"`
	ID          StringID       `json:"id,omitempty"`
	Key         string         `json:"key,omitempty"`
	Name        string         `json:"name,omitempty"`
	Description string         `json:"description,omitempty"`
	Type        string         `json:"type,omitempty"`
	Version     int            `json:"version,omitempty"`
	Schema      map[string]any `json:"schema,omitempty"`
	ReadOnly    bool           `json:"readonly,omitempty"`
}

type Entity struct {
	Self       string         `json:"self,omitempty"`
	ID         StringID       `json:"id,omitempty"`
	ShortID    int64          `json:"shortId,omitempty"`
	EntityType string         `json:"entityType,omitempty"`
	Version    int            `json:"version,omitempty"`
	Fields     map[string]any `json:"fields,omitempty"`
	CreatedAt  string         `json:"createdAt,omitempty"`
	UpdatedAt  string         `json:"updatedAt,omitempty"`
	CreatedBy  *User          `json:"createdBy,omitempty"`
	UpdatedBy  *User          `json:"updatedBy,omitempty"`
}

type SearchResult[T any] struct {
	Items      []T    `json:"items"`
	TotalCount int    `json:"totalCount,omitempty"`
	Pages      int    `json:"pages,omitempty"`
	Page       int    `json:"page,omitempty"`
	PerPage    int    `json:"perPage,omitempty"`
	ScrollID   string `json:"scrollId,omitempty"`
}
