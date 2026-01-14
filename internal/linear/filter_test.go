package linear

import "testing"

func TestBuildIssueFilter(t *testing.T) {
	priority := 2
	filter := IssueFilter{
		TeamID:     "team",
		AssigneeID: "assignee",
		StateID:    "state",
		LabelIDs:   []string{"l1", "l2"},
		ProjectID:  "project",
		CycleID:    "cycle",
		Search:     "bug",
		Priority:   &priority,
	}

	out := buildIssueFilter(filter)
	if out == nil {
		t.Fatalf("expected filter map")
	}
	if out["team"] == nil || out["assignee"] == nil || out["state"] == nil {
		t.Fatalf("missing expected keys")
	}
}
