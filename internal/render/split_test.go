package render

import (
	"strings"
	"testing"

	"github.com/tomas-chudjak/kleidi-task/internal/core"
)

func TestTitleShowsParentProgress(t *testing.T) {
	task := core.Task{Title: "MCP HTTP transport", ChildTotal: 3, ChildDone: 1}
	if got := Title(task); got != "MCP HTTP transport (1/3)" {
		t.Errorf("got %q", got)
	}
}

func TestTitleIndentsChild(t *testing.T) {
	parentID := int64(42)
	task := core.Task{Title: "Service layer", ParentID: &parentID}
	if got := Title(task); got != ChildPrefix+"Service layer" {
		t.Errorf("got %q", got)
	}
}

func TestTitlePlainForStandaloneTask(t *testing.T) {
	if got := Title(core.Task{Title: "Just a task"}); got != "Just a task" {
		t.Errorf("got %q", got)
	}
}

func TestSplittingDoesNotWidenTheTable(t *testing.T) {
	// The canonical column set is fixed; parent/child structure rides in the
	// title cell.
	parentID := int64(1)
	tasks := []core.Task{
		{ID: 1, Type: core.TypeFeature, Status: core.StatusDoing, Title: "Parent", ChildTotal: 2, ChildDone: 1},
		{ID: 2, Type: core.TypeFeature, Status: core.StatusDone, Title: "Child", ParentID: &parentID},
	}

	md := Markdown(tasks, ListMeta{})
	for _, line := range strings.Split(md, "\n") {
		if !strings.HasPrefix(line, "|") {
			continue
		}
		if got := strings.Count(line, "|"); got != len(TaskColumns)+1 {
			t.Errorf("row %q has %d separators, want %d", line, got, len(TaskColumns)+1)
		}
	}
	if !strings.Contains(md, "Parent (1/2)") {
		t.Error("parent progress missing from markdown")
	}
	if !strings.Contains(md, ChildPrefix+"Child") {
		t.Error("child indent missing from markdown")
	}
}
