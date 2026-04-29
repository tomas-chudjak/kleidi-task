package core

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ExportData represents the full export envelope.
type ExportData struct {
	Project    string    `json:"project"`
	ExportedAt time.Time `json:"exported_at"`
	Count      int       `json:"count"`
	Tasks      []Task    `json:"tasks"`
}

// ExportJSON exports tasks as JSON.
func ExportJSON(project string, tasks []Task) ([]byte, error) {
	data := ExportData{
		Project:    project,
		ExportedAt: time.Now().UTC(),
		Count:      len(tasks),
		Tasks:      tasks,
	}
	return json.MarshalIndent(data, "", "  ")
}

// ExportMarkdown exports tasks as readable Markdown.
func ExportMarkdown(project string, tasks []Task) []byte {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("# %s — Task Export\n\n", project))
	b.WriteString(fmt.Sprintf("*Exported: %s · %d tasks*\n\n", time.Now().UTC().Format("2006-01-02 15:04"), len(tasks)))

	// Group by status
	groups := map[TaskStatus][]Task{}
	order := []TaskStatus{StatusTodo, StatusDoing, StatusDone}
	for _, t := range tasks {
		groups[t.Status] = append(groups[t.Status], t)
	}

	for _, status := range order {
		group := groups[status]
		if len(group) == 0 {
			continue
		}
		heading := string(status)
		if len(heading) > 0 {
			heading = strings.ToUpper(heading[:1]) + heading[1:]
		}
		b.WriteString(fmt.Sprintf("## %s\n\n", heading))
		for _, t := range group {
			checkbox := "- [ ]"
			if t.Status == StatusDone {
				checkbox = "- [x]"
			}
			pri := ""
			if t.Priority > 0 {
				pri = fmt.Sprintf(" P%d", t.Priority)
			}
			cat := ""
			if t.Category != "" {
				cat = fmt.Sprintf(" `%s`", t.Category)
			}
			b.WriteString(fmt.Sprintf("%s **#%d** %s (%s%s%s)\n", checkbox, t.ID, t.Title, t.Type, pri, cat))
			if t.Description != "" {
				for _, line := range strings.Split(t.Description, "\n") {
					b.WriteString(fmt.Sprintf("  > %s\n", line))
				}
			}
			b.WriteString("\n")
		}
	}

	return []byte(b.String())
}

// ImportJSON parses an exported JSON file and returns tasks to create.
func ImportJSON(data []byte) (ExportData, error) {
	var export ExportData
	if err := json.Unmarshal(data, &export); err != nil {
		return ExportData{}, fmt.Errorf("parsing JSON: %w", err)
	}
	return export, nil
}

// ImportTasks creates tasks from import data, skipping duplicates by title.
func ImportTasks(ctx context.Context, taskService *TaskService, tasks []Task, source Source) (created, skipped int, err error) {
	// Load existing tasks for duplicate detection
	existing, err := taskService.List(ctx, ListTasksFilter{Limit: 10000})
	if err != nil {
		return 0, 0, fmt.Errorf("listing existing tasks: %w", err)
	}
	titleSet := make(map[string]bool, len(existing))
	for _, t := range existing {
		titleSet[strings.ToLower(t.Title)] = true
	}

	for _, t := range tasks {
		if titleSet[strings.ToLower(t.Title)] {
			skipped++
			continue
		}
		_, err := taskService.Create(ctx, CreateTaskInput{
			Title:       t.Title,
			Description: t.Description,
			Type:        t.Type,
			Priority:    t.Priority,
			Category:    t.Category,
			Source:      source,
		})
		if err != nil {
			return created, skipped, fmt.Errorf("creating task %q: %w", t.Title, err)
		}
		created++
		titleSet[strings.ToLower(t.Title)] = true
	}
	return created, skipped, nil
}
