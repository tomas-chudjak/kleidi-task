package core

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"os/exec"
	"time"

	"github.com/tomas-chudjak/kleidi-task/internal/db/generated"
)

// PhaseAction defines an action to execute when entering a phase.
type PhaseAction struct {
	Type        string `json:"type"`        // "shell" or "prompt"
	Command     string `json:"command"`     // shell command or prompt text
	Description string `json:"description"` // human-readable description
}

// HistoryEntry represents a recorded workflow phase transition.
type HistoryEntry struct {
	ID         int64     `json:"id"`
	TaskID     int64     `json:"task_id"`
	Phase      string    `json:"phase"`
	Action     string    `json:"action"`
	ActionType string    `json:"action_type"`
	Output     string    `json:"output"`
	Success    bool      `json:"success"`
	DurationMs int64     `json:"duration_ms"`
	CreatedAt  time.Time `json:"created_at"`
}

// WorkflowDef defines the phases and triggers for a task type.
type WorkflowDef struct {
	TaskType     string              `json:"task_type"`
	Phases       []string            `json:"phases"`
	Triggers     map[string]Triggers `json:"triggers"`
	PhasePrompts map[string]string   `json:"phase_prompts"`
	Color        string              `json:"color"`
	Prefix       string              `json:"prefix"`
	IsBuiltin    bool                `json:"is_builtin"`
}

// Triggers defines before/after skill triggers for a workflow phase.
type Triggers struct {
	Before []string `json:"before,omitempty"`
	After  []string `json:"after,omitempty"`
}

// AdvanceResult holds the result of advancing a task to the next phase.
type AdvanceResult struct {
	Task            Task           `json:"task"`
	PreviousPhase   string         `json:"previous_phase"`
	CurrentPhase    string         `json:"current_phase"`
	SuggestedSkills []string       `json:"suggested_skills,omitempty"`
	Actions         []PhaseAction  `json:"actions,omitempty"`
	ActionResults   []HistoryEntry `json:"action_results,omitempty"`
	IsComplete      bool           `json:"is_complete"`
}

// WorkflowContext provides workflow info for a task.
type WorkflowContext struct {
	CurrentPhase    string   `json:"current_phase"`
	CurrentPrompt   string   `json:"current_prompt,omitempty"`
	NextPhase       string   `json:"next_phase,omitempty"`
	Phases          []string `json:"phases"`
	PhaseIndex      int      `json:"phase_index"`
	SuggestedSkills []string `json:"suggested_skills,omitempty"`
}

// WorkflowService manages task workflows and phase transitions.
type WorkflowService struct {
	db      *sql.DB
	queries *generated.Queries
}

// NewWorkflowService creates a new WorkflowService.
func NewWorkflowService(db *sql.DB) *WorkflowService {
	return &WorkflowService{db: db, queries: generated.New(db)}
}

// GetWorkflow returns the workflow definition for a task type.
func (s *WorkflowService) GetWorkflow(ctx context.Context, taskType string) (WorkflowDef, error) {
	row, err := s.queries.GetWorkflow(ctx, taskType)
	if err != nil {
		if err == sql.ErrNoRows {
			// Default fallback: simple 3-phase workflow
			return WorkflowDef{
				TaskType: taskType,
				Phases:   []string{"todo", "doing", "done"},
				Triggers: map[string]Triggers{},
			}, nil
		}
		return WorkflowDef{}, fmt.Errorf("getting workflow: %w", err)
	}
	return workflowFromRow(row), nil
}

// GetContext returns workflow context for a task.
func (s *WorkflowService) GetContext(ctx context.Context, task Task) (WorkflowContext, error) {
	wf, err := s.GetWorkflow(ctx, string(task.Type))
	if err != nil {
		return WorkflowContext{}, err
	}

	phase := currentPhase(task)
	idx := phaseIndex(wf.Phases, phase)

	wc := WorkflowContext{
		CurrentPhase:  phase,
		CurrentPrompt: wf.PhasePrompts[phase],
		Phases:        wf.Phases,
		PhaseIndex:    idx,
	}

	if idx < len(wf.Phases)-1 {
		nextPhase := wf.Phases[idx+1]
		wc.NextPhase = nextPhase
		if triggers, ok := wf.Triggers[nextPhase]; ok {
			wc.SuggestedSkills = append(wc.SuggestedSkills, triggers.Before...)
		}
	}

	return wc, nil
}

// Advance moves a task to the next phase in its workflow.
func (s *WorkflowService) Advance(ctx context.Context, taskID int64) (AdvanceResult, error) {
	// Get task
	taskService := NewTaskService(s.db)
	task, err := taskService.Get(ctx, taskID)
	if err != nil {
		return AdvanceResult{}, err
	}

	wf, err := s.GetWorkflow(ctx, string(task.Type))
	if err != nil {
		return AdvanceResult{}, err
	}

	prevPhase := currentPhase(task)
	idx := phaseIndex(wf.Phases, prevPhase)

	if idx >= len(wf.Phases)-1 {
		return AdvanceResult{}, fmt.Errorf("task #%d is already in final phase %q", taskID, prevPhase)
	}

	nextPhase := wf.Phases[idx+1]
	newStatus := statusFromPhase(wf.Phases, nextPhase)

	// Update phase and derived status
	err = s.queries.SetTaskPhase(ctx, generated.SetTaskPhaseParams{
		Phase:  sql.NullString{String: nextPhase, Valid: true},
		Status: string(newStatus),
		ID:     taskID,
	})
	if err != nil {
		return AdvanceResult{}, fmt.Errorf("advancing task: %w", err)
	}

	// If done, also set completed_at
	if newStatus == StatusDone {
		taskService.Complete(ctx, taskID)
	}

	// Re-fetch updated task
	task, _ = taskService.Get(ctx, taskID)

	// Record phase transition in history
	s.recordHistory(ctx, taskID, nextPhase, "phase-advance", "none", fmt.Sprintf("%s → %s", prevPhase, nextPhase), true, 0)

	// Execute actions for the new phase
	var actions []PhaseAction
	var actionResults []HistoryEntry
	var skills []string

	if triggers, ok := wf.Triggers[nextPhase]; ok {
		skills = append(skills, triggers.Before...)
		skills = append(skills, triggers.After...)

		// Execute shell actions from before triggers
		for _, skillName := range triggers.Before {
			action := s.resolveAction(skillName)
			actions = append(actions, action)
			if action.Type == "shell" {
				result := s.executeShell(ctx, taskID, nextPhase, action)
				actionResults = append(actionResults, result)
			}
		}
	}

	return AdvanceResult{
		Task:            task,
		PreviousPhase:   prevPhase,
		CurrentPhase:    nextPhase,
		SuggestedSkills: skills,
		Actions:         actions,
		ActionResults:   actionResults,
		IsComplete:      newStatus == StatusDone,
	}, nil
}

// executeShell runs a shell command and records the result.
func (s *WorkflowService) executeShell(ctx context.Context, taskID int64, phase string, action PhaseAction) HistoryEntry {
	start := time.Now()
	cmd := exec.CommandContext(ctx, "sh", "-c", action.Command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start).Milliseconds()
	success := err == nil

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += stderr.String()
	}
	// Truncate output if too long
	if len(output) > 4000 {
		output = output[:4000] + "\n... (truncated)"
	}

	entry := s.recordHistory(ctx, taskID, phase, action.Command, "shell", output, success, duration)
	return entry
}

// resolveAction maps a skill name to a PhaseAction.
func (s *WorkflowService) resolveAction(skillName string) PhaseAction {
	// Built-in skill mappings
	builtins := map[string]PhaseAction{
		"run-tests":          {Type: "shell", Command: "go test ./...", Description: "Run test suite"},
		"lint":               {Type: "shell", Command: "go vet ./...", Description: "Run linter"},
		"type-check":         {Type: "shell", Command: "go build ./...", Description: "Type check"},
		"smoke-test":         {Type: "shell", Command: "go test -short ./...", Description: "Quick smoke test"},
		"regression-test":    {Type: "shell", Command: "go test -run TestRegression ./...", Description: "Regression tests"},
	}

	if action, ok := builtins[skillName]; ok {
		return action
	}

	// Unknown skill — return as prompt for AI
	return PhaseAction{
		Type:        "prompt",
		Command:     skillName,
		Description: skillName,
	}
}

// ListWorkflows returns all workflow definitions.
func (s *WorkflowService) ListWorkflows(ctx context.Context) ([]WorkflowDef, error) {
	rows, err := s.queries.ListWorkflows(ctx)
	if err != nil {
		return nil, err
	}
	wfs := make([]WorkflowDef, len(rows))
	for i, r := range rows {
		wfs[i] = workflowFromRow(r)
	}
	return wfs, nil
}

// recordHistory writes a history entry to the database.
func (s *WorkflowService) recordHistory(ctx context.Context, taskID int64, phase, action, actionType, output string, success bool, durationMs int64) HistoryEntry {
	successInt := int64(0)
	if success {
		successInt = 1
	}
	row, err := s.queries.InsertWorkflowHistory(ctx, generated.InsertWorkflowHistoryParams{
		TaskID:     taskID,
		Phase:      phase,
		Action:     action,
		ActionType: actionType,
		Output:     output,
		Success:    successInt,
		DurationMs: durationMs,
	})
	if err != nil {
		return HistoryEntry{Phase: phase, Action: action, ActionType: actionType, Output: output, Success: success}
	}
	return historyFromRow(row)
}

// GetHistory returns the workflow history for a task.
func (s *WorkflowService) GetHistory(ctx context.Context, taskID int64) ([]HistoryEntry, error) {
	rows, err := s.queries.ListWorkflowHistory(ctx, taskID)
	if err != nil {
		return nil, err
	}
	entries := make([]HistoryEntry, len(rows))
	for i, r := range rows {
		entries[i] = historyFromRow(r)
	}
	return entries, nil
}

func historyFromRow(row generated.WorkflowHistory) HistoryEntry {
	return HistoryEntry{
		ID:         row.ID,
		TaskID:     row.TaskID,
		Phase:      row.Phase,
		Action:     row.Action,
		ActionType: row.ActionType,
		Output:     row.Output,
		Success:    row.Success != 0,
		DurationMs: row.DurationMs,
		CreatedAt:  row.CreatedAt,
	}
}

func currentPhase(task Task) string {
	if task.Phase != "" {
		return task.Phase
	}
	// No phase set — derive from status
	return string(task.Status)
}

func phaseIndex(phases []string, phase string) int {
	for i, p := range phases {
		if p == phase {
			return i
		}
	}
	return 0
}

func statusFromPhase(phases []string, phase string) TaskStatus {
	if len(phases) == 0 {
		return StatusTodo
	}
	if phase == phases[len(phases)-1] {
		return StatusDone
	}
	if phase == phases[0] {
		return StatusTodo
	}
	return StatusDoing
}

// UpdateWorkflow updates a workflow definition.
func (s *WorkflowService) UpdateWorkflow(ctx context.Context, wf WorkflowDef) error {
	phasesJSON, _ := json.Marshal(wf.Phases)
	triggersJSON, _ := json.Marshal(wf.Triggers)
	promptsJSON, _ := json.Marshal(wf.PhasePrompts)
	return s.queries.UpdateWorkflow(ctx, generated.UpdateWorkflowParams{
		Phases:       string(phasesJSON),
		Triggers:     string(triggersJSON),
		PhasePrompts: string(promptsJSON),
		TaskType:     wf.TaskType,
	})
}

// CreateWorkflow creates a new custom task type with its workflow.
func (s *WorkflowService) CreateWorkflow(ctx context.Context, wf WorkflowDef) (WorkflowDef, error) {
	if wf.TaskType == "" {
		return WorkflowDef{}, fmt.Errorf("%w: task type name is required", ErrInvalidInput)
	}
	if len(wf.Phases) == 0 {
		wf.Phases = []string{"todo", "doing", "done"}
	}
	if wf.Triggers == nil {
		wf.Triggers = map[string]Triggers{}
	}
	if wf.PhasePrompts == nil {
		wf.PhasePrompts = map[string]string{}
	}
	if wf.Color == "" {
		wf.Color = "#e0e7ef"
	}

	phasesJSON, _ := json.Marshal(wf.Phases)
	triggersJSON, _ := json.Marshal(wf.Triggers)
	promptsJSON, _ := json.Marshal(wf.PhasePrompts)

	row, err := s.queries.CreateWorkflow(ctx, generated.CreateWorkflowParams{
		TaskType:     wf.TaskType,
		Phases:       string(phasesJSON),
		Triggers:     string(triggersJSON),
		PhasePrompts: string(promptsJSON),
		Color:        wf.Color,
		Prefix:       wf.Prefix,
		IsBuiltin:    0,
	})
	if err != nil {
		return WorkflowDef{}, fmt.Errorf("creating workflow: %w", err)
	}
	return workflowFromRow(row), nil
}

// DeleteWorkflow removes a custom (non-builtin) task type and its workflow.
func (s *WorkflowService) DeleteWorkflow(ctx context.Context, taskType string) error {
	return s.queries.DeleteWorkflow(ctx, taskType)
}

// ValidTypeExists checks if a task type exists in the workflows table.
func (s *WorkflowService) ValidTypeExists(ctx context.Context, taskType string) bool {
	_, err := s.queries.GetWorkflow(ctx, taskType)
	return err == nil
}

func workflowFromRow(row generated.Workflow) WorkflowDef {
	wf := WorkflowDef{
		TaskType:     row.TaskType,
		Triggers:     map[string]Triggers{},
		PhasePrompts: map[string]string{},
		Color:        row.Color,
		Prefix:       row.Prefix,
		IsBuiltin:    row.IsBuiltin != 0,
	}
	json.Unmarshal([]byte(row.Phases), &wf.Phases)
	json.Unmarshal([]byte(row.Triggers), &wf.Triggers)
	json.Unmarshal([]byte(row.PhasePrompts), &wf.PhasePrompts)
	return wf
}
