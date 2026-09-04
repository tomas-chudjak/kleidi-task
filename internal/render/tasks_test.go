package render

import (
	"bytes"
	"strings"
	"testing"

	"github.com/tomas-chudjak/kleidi-task/internal/core"
)

func sampleTasks() []core.Task {
	return []core.Task{
		{ID: 60, Type: core.TypeBug, Status: core.StatusTodo, Priority: 7, Category: "backend", Title: "Task templates not applied"},
		{ID: 68, Type: core.TypeFeature, Status: core.StatusTodo, Title: "Task dependencies"},
	}
}

func TestMarkdownIsCanonical(t *testing.T) {
	got := Markdown(sampleTasks(), ListMeta{Project: "kleidi-task", Open: 5, Done: 42, HasCounts: true})

	want := strings.Join([]string{
		"**kleidi-task** — 5 open, 42 done",
		"",
		"| # | type | status | pri | category | title |",
		"|---|----|------|---|--------|-----|",
		"| 60 | bug | todo | 7 | backend | Task templates not applied |",
		"| 68 | feature | todo | – | – | Task dependencies |",
	}, "\n")

	if got != want {
		t.Errorf("markdown mismatch\ngot:\n%s\n\nwant:\n%s", got, want)
	}
}

func TestMarkdownEscapesPipes(t *testing.T) {
	tasks := []core.Task{{ID: 1, Type: core.TypeTask, Status: core.StatusTodo, Title: "a | b\nc"}}
	got := Markdown(tasks, ListMeta{})

	if strings.Contains(got, "a | b") {
		t.Errorf("unescaped pipe leaked into table: %s", got)
	}
	if !strings.Contains(got, `a \| b c`) {
		t.Errorf("expected escaped, newline-flattened title, got: %s", got)
	}
	if n := strings.Count(got, "\n"); n != 2 {
		t.Errorf("expected header + rule + one row, got %d newlines in:\n%s", n, got)
	}
}

func TestMarkdownEmpty(t *testing.T) {
	got := Markdown(nil, ListMeta{Project: "kleidi-task", HasCounts: true})
	want := "**kleidi-task** — 0 open, 0 done\n\n_No tasks found._"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}

	if got := Markdown(nil, ListMeta{}); got != "_No tasks found._" {
		t.Errorf("without project meta got %q", got)
	}
}

func TestMarkdownPagination(t *testing.T) {
	meta := ListMeta{Page: 2, TotalPages: 3, Total: 68}
	got := Markdown(sampleTasks(), meta)
	if !strings.HasSuffix(got, "\nPage 2/3 (total: 68)") {
		t.Errorf("missing pagination footer: %s", got)
	}

	if got := Markdown(sampleTasks(), ListMeta{Page: 1, TotalPages: 1, Total: 2}); strings.Contains(got, "Page ") {
		t.Errorf("single page should not render a footer: %s", got)
	}
}

func TestTableUsesSameColumns(t *testing.T) {
	var buf bytes.Buffer
	Table(&buf, sampleTasks())

	lines := strings.Split(strings.TrimRight(buf.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected header + 2 rows, got %d lines: %q", len(lines), buf.String())
	}

	for _, c := range TaskColumns {
		if !strings.Contains(lines[0], strings.ToUpper(c.Header)) {
			t.Errorf("header %q missing from terminal table: %q", c.Header, lines[0])
		}
	}
	if !strings.Contains(lines[2], EmptyCell) {
		t.Errorf("expected %q placeholder for unset priority/category: %q", EmptyCell, lines[2])
	}
}

func TestMetaForDerivesOpenCount(t *testing.T) {
	stats := core.ProjectStats{Todo: 4, Doing: 1, Done: 42}
	result := core.ListResult{Page: 1, TotalPages: 2, Total: 68}

	meta := MetaFor("kleidi-task", stats, result)

	if meta.Open != 5 {
		t.Errorf("Open = %d, want 5 (todo + doing)", meta.Open)
	}
	if meta.Done != 42 || !meta.HasCounts {
		t.Errorf("unexpected meta: %+v", meta)
	}
	if meta.TotalPages != 2 || meta.Total != 68 {
		t.Errorf("pagination not carried over: %+v", meta)
	}
}

func TestJSONRendersEmptyArray(t *testing.T) {
	var buf bytes.Buffer
	if err := JSON(&buf, nil); err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(buf.String()) != "[]" {
		t.Errorf("nil tasks should render [], got %q", buf.String())
	}
}
