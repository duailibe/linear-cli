package linear

type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

type Team struct {
	ID   string `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

type WorkflowState struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

type IssueSummary struct {
	ID         string `json:"id"`
	Identifier string `json:"identifier"`
	Title      string `json:"title"`
	URL        string `json:"url"`
	State      string `json:"state"`
	Assignee   string `json:"assignee"`
	TeamKey    string `json:"team_key"`
	Cycle      string `json:"cycle"`
	Priority   int    `json:"priority"`
}

type IssueRelation struct {
	ID             string `json:"id"`
	IssueID        string `json:"issue_id"`
	RelatedIssueID string `json:"related_issue_id"`
	Type           string `json:"type"`
}

type IssueRelationSet struct {
	Relations        []IssueRelation `json:"relations"`
	InverseRelations []IssueRelation `json:"inverse_relations"`
}

type IssueDetail struct {
	ID          string       `json:"id"`
	Identifier  string       `json:"identifier"`
	Title       string       `json:"title"`
	URL         string       `json:"url"`
	Description string       `json:"description"`
	Priority    int          `json:"priority"`
	State       string       `json:"state"`
	Assignee    string       `json:"assignee"`
	TeamID      string       `json:"team_id"`
	TeamKey     string       `json:"team_key"`
	Cycle       string       `json:"cycle"`
	Project     string       `json:"project"`
	Labels      []string     `json:"labels"`
	Comments    []Comment    `json:"comments,omitempty"`
	Uploads     []Attachment `json:"uploads,omitempty"`
	CreatedAt   string       `json:"created_at"`
	UpdatedAt   string       `json:"updated_at"`
}

type Comment struct {
	ID        string `json:"id"`
	Body      string `json:"body,omitempty"`
	BodyData  string `json:"body_data,omitempty"`
	CreatedAt string `json:"created_at,omitempty"`
	UserName  string `json:"user_name,omitempty"`
	UserEmail string `json:"user_email,omitempty"`
}

type Attachment struct {
	ID          string `json:"id"`
	Title       string `json:"title,omitempty"`
	URL         string `json:"url,omitempty"`
	FileName    string `json:"file_name,omitempty"`
	ContentType string `json:"content_type,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	CommentID   string `json:"comment_id,omitempty"`
}

type IssueFilter struct {
	TeamID     string
	AssigneeID string
	StateID    string
	LabelIDs   []string
	ProjectID  string
	CycleID    string
	Search     string
	Priority   *int
}

type IssuePage struct {
	Nodes    []IssueSummary `json:"nodes"`
	PageInfo PageInfo       `json:"page_info"`
}

type Cycle struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Number   string `json:"number"`
	StartsAt string `json:"starts_at"`
	EndsAt   string `json:"ends_at"`
	IsActive string `json:"is_active"`
}

type CyclePage struct {
	Nodes    []Cycle  `json:"nodes"`
	PageInfo PageInfo `json:"page_info"`
}

type PageInfo struct {
	HasNextPage bool   `json:"has_next_page"`
	EndCursor   string `json:"end_cursor"`
}
