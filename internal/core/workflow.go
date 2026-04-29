package core

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/ahoylog/kvik-tasks/internal/db/generated"
)

// WorkflowDef defines the phases and triggers for a task type.
type WorkflowDef struct {
	TaskType string              `json:"task_type"`
	Phases   []string            `json:"phases"`
	Triggers map[string]Triggers `json:"triggers"`
}

// Triggers defines before/after skill triggers for a workflow phase.
type Triggers struct {
	Before []string `json:"before,omitempty"`
	After  []string `json:"after,omitempty"`
}

// AdvanceResult holds the result of advancing a task to the next phase.
type AdvanceResult struct {
	Task            Task     `json:"task"`
	PreviousPhase   string   `json:"previous_phase"`
	CurrentPhase    string   `json:"current_phase"`
	SuggestedSkills []string `json:"suggested_skills,omitempty"`
	IsComplete      bool     `json:"is_complete"`
}

// WorkflowContext provides workflow info for a task.
type WorkflowContext struct {
	CurrentPhase    string   `json:"current_phase"`
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
		CurrentPhase: phase,
		Phases:       wf.Phases,
		PhaseIndex:   idx,
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

	// Collect suggested skills for the new phase
	var skills []string
	if triggers, ok := wf.Triggers[nextPhase]; ok {
		skills = append(skills, triggers.Before...)
		skills = append(skills, triggers.After...)
	}

	return AdvanceResult{
		Task:            task,
		PreviousPhase:   prevPhase,
		CurrentPhase:    nextPhase,
		SuggestedSkills: skills,
		IsComplete:      newStatus == StatusDone,
	}, nil
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

func workflowFromRow(row generated.Workflow) WorkflowDef {
	wf := WorkflowDef{
		TaskType: row.TaskType,
		Triggers: map[string]Triggers{},
	}
	json.Unmarshal([]byte(row.Phases), &wf.Phases)
	json.Unmarshal([]byte(row.Triggers), &wf.Triggers)
	return wf
}
