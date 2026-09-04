// Package render holds the canonical presentation of kleidi-task domain objects.
//
// Every entry point (CLI, MCP, UI) renders task lists through the column
// definition in this package, so a task list looks the same no matter which
// surface produced it. The canonical column order is:
//
//	# | type | status | pri | category | title
//
// Renderers never re-sort: they preserve the order the service layer returned
// (priority DESC, created_at DESC), so pagination and filters stay meaningful.
package render

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	"github.com/tomas-chudjak/kleidi-task/internal/core"
)

// EmptyCell is the placeholder for an unset optional value. Used by every
// renderer so an empty priority looks identical in the terminal, in MCP
// output and in the UI.
const EmptyCell = "–"

// Column is one column of the canonical task list.
type Column struct {
	// Header is the column label, lowercase — used verbatim in markdown and
	// uppercased for the terminal table.
	Header string
	// Cell returns the rendered value for a task, already substituted with
	// EmptyCell when unset.
	Cell func(core.Task) string
}

// TaskColumns is the canonical column set and order. UI templates mirror this
// order; changing it here means changing it everywhere.
var TaskColumns = []Column{
	{"#", func(t core.Task) string { return fmt.Sprintf("%d", t.ID) }},
	{"type", func(t core.Task) string { return string(t.Type) }},
	{"status", func(t core.Task) string { return string(t.Status) }},
	{"pri", func(t core.Task) string {
		if t.Priority > 0 {
			return fmt.Sprintf("%d", t.Priority)
		}
		return EmptyCell
	}},
	{"category", func(t core.Task) string { return orEmpty(t.Category) }},
	{"title", func(t core.Task) string { return t.Title }},
}

// ListMeta is the optional context line rendered above a task list.
type ListMeta struct {
	// Project is the project name or slug. Empty means no header line.
	Project string
	// Open (todo + doing) and Done are shown next to the project name when
	// HasCounts is set.
	Open      int64
	Done      int64
	HasCounts bool
	// Pagination — rendered only when TotalPages > 1.
	Page       int64
	TotalPages int64
	Total      int64
}

// MetaFor assembles the header/pagination context from what the service layer
// already returned, so CLI and MCP produce byte-identical header lines.
func MetaFor(projectName string, stats core.ProjectStats, result core.ListResult) ListMeta {
	return ListMeta{
		Project:    projectName,
		Open:       stats.Todo + stats.Doing,
		Done:       stats.Done,
		HasCounts:  true,
		Page:       result.Page,
		TotalPages: result.TotalPages,
		Total:      result.Total,
	}
}

// Markdown renders the canonical markdown table. This is the format MCP
// clients receive and are expected to print verbatim.
func Markdown(tasks []core.Task, meta ListMeta) string {
	var b strings.Builder

	if header := meta.headerLine(); header != "" {
		b.WriteString(header)
		b.WriteString("\n\n")
	}

	if len(tasks) == 0 {
		b.WriteString("_No tasks found._")
		return b.String()
	}

	labels := make([]string, len(TaskColumns))
	rules := make([]string, len(TaskColumns))
	for i, c := range TaskColumns {
		labels[i] = c.Header
		rules[i] = strings.Repeat("-", max(3, len(c.Header)))
	}
	writeRow(&b, labels)
	fmt.Fprintf(&b, "|%s|\n", strings.Join(rules, "|"))

	cells := make([]string, len(TaskColumns))
	for _, t := range tasks {
		for i, c := range TaskColumns {
			cells[i] = escapeMarkdownCell(c.Cell(t))
		}
		writeRow(&b, cells)
	}

	if meta.TotalPages > 1 {
		fmt.Fprintf(&b, "\nPage %d/%d (total: %d)", meta.Page, meta.TotalPages, meta.Total)
	}

	return strings.TrimRight(b.String(), "\n")
}

// Table renders the canonical column set as an aligned plain-text table for
// the terminal.
func Table(w io.Writer, tasks []core.Task) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	labels := make([]string, len(TaskColumns))
	for i, c := range TaskColumns {
		labels[i] = strings.ToUpper(c.Header)
	}
	fmt.Fprintln(tw, strings.Join(labels, "\t"))

	cells := make([]string, len(TaskColumns))
	for _, t := range tasks {
		for i, c := range TaskColumns {
			cells[i] = c.Cell(t)
		}
		fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}

	tw.Flush()
}

// JSON renders the raw task list for scripts and other machine consumers.
func JSON(w io.Writer, tasks []core.Task) error {
	if tasks == nil {
		tasks = []core.Task{}
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(tasks)
}

func writeRow(b *strings.Builder, cells []string) {
	fmt.Fprintf(b, "| %s |\n", strings.Join(cells, " | "))
}

func (m ListMeta) headerLine() string {
	if m.Project == "" {
		return ""
	}
	if !m.HasCounts {
		return fmt.Sprintf("**%s**", m.Project)
	}
	return fmt.Sprintf("**%s** — %d open, %d done", m.Project, m.Open, m.Done)
}

// escapeMarkdownCell keeps a cell from breaking out of its table column.
func escapeMarkdownCell(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "|", `\|`)
	return strings.TrimSpace(s)
}

func orEmpty(s string) string {
	if s == "" {
		return EmptyCell
	}
	return s
}
