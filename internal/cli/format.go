package cli

import (
	"fmt"
	"os"

	"github.com/tomas-chudjak/kleidi-task/internal/core"
	"github.com/tomas-chudjak/kleidi-task/internal/render"
)

// printTaskTable prints a list of tasks in the canonical column order
// (# | type | status | pri | category | title) shared with MCP and the UI.
func printTaskTable(tasks []core.Task) {
	render.Table(os.Stdout, tasks)
}

// printTaskRow prints a single task as a one-line confirmation.
// Format: "#ID [type] title (status)"
func printTaskRow(action string, t core.Task) {
	cat := ""
	if t.Category != "" {
		cat = fmt.Sprintf(" [%s]", t.Category)
	}
	fmt.Printf("%s #%d %s — %s (%s)%s\n", action, t.ID, t.Type, t.Title, t.Status, cat)
}

// printTaskDetail prints full task details in a structured key-value format.
func printTaskDetail(t core.Task) {
	fmt.Printf("ID:          #%d\n", t.ID)
	fmt.Printf("Title:       %s\n", t.Title)
	fmt.Printf("Type:        %s\n", t.Type)
	fmt.Printf("Status:      %s\n", t.Status)
	fmt.Printf("Priority:    %d\n", t.Priority)
	if t.Category != "" {
		fmt.Printf("Category:    %s\n", t.Category)
	}
	if t.Description != "" {
		fmt.Printf("Description: %s\n", t.Description)
	}
	fmt.Printf("Source:      %s\n", t.Source)
	fmt.Printf("Created:     %s\n", t.CreatedAt.Format("2006-01-02 15:04"))
	fmt.Printf("Updated:     %s\n", t.UpdatedAt.Format("2006-01-02 15:04"))
	if t.CompletedAt != nil {
		fmt.Printf("Completed:   %s\n", t.CompletedAt.Format("2006-01-02 15:04"))
	}
	if t.IsArchived {
		fmt.Printf("Archived:    yes\n")
	}
	if t.Metadata != nil {
		if t.Metadata.ConversationID != "" {
			fmt.Printf("Conversation: %s\n", t.Metadata.ConversationID)
		}
		if t.Metadata.SessionID != "" {
			fmt.Printf("Session:     %s\n", t.Metadata.SessionID)
		}
	}
}
