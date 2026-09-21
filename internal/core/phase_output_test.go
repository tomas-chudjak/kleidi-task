package core

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tomas-chudjak/kleidi-task/internal/db"
)

// advanceTo moves a task through phases until it reaches target.
func advanceTo(t *testing.T, wf *WorkflowService, id int64, target string) {
	t.Helper()
	for i := 0; i < 10; i++ {
		res, err := wf.Advance(context.Background(), id)
		if err != nil {
			t.Fatalf("advancing to %s: %v", target, err)
		}
		if res.CurrentPhase == target {
			return
		}
	}
	t.Fatalf("never reached phase %s", target)
}

func phaseOutputFixture(t *testing.T) (*TaskService, *WorkflowService, Task) {
	t.Helper()
	testDB := db.NewTestProjectDB(t)
	tasks := NewTaskService(testDB)
	tasks.SetTemplates(NewTemplateService(testDB))
	wf := NewWorkflowService(testDB)

	task, err := tasks.Create(context.Background(), CreateTaskInput{
		Title:       "Add dark mode",
		Type:        TypeFeature,
		Description: "## Problem\n\nno dark mode\n",
		Source:      SourceCLI,
	})
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}
	return tasks, wf, task
}

func TestFeatureWorkflowDeclaresResearchOutput(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	wf := NewWorkflowService(testDB)

	def, err := wf.GetWorkflow(context.Background(), "feature")
	if err != nil {
		t.Fatalf("getting workflow: %v", err)
	}
	if def.PhaseOutputs["research"] != "Design" {
		t.Errorf("expected research → Design, got %q", def.PhaseOutputs["research"])
	}
}

func TestAdvanceBlockedWhileOutputSectionEmpty(t *testing.T) {
	_, wf, task := phaseOutputFixture(t)
	advanceTo(t, wf, task.ID, "research")

	_, err := wf.Advance(context.Background(), task.ID)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	// The error must name the section so the agent knows where to write.
	if !strings.Contains(err.Error(), "## Design") {
		t.Errorf("error should name the section: %v", err)
	}
	if !strings.Contains(err.Error(), "research") {
		t.Errorf("error should name the phase: %v", err)
	}
}

func TestAdvanceAllowedOnceOutputSectionHasContent(t *testing.T) {
	tasks, wf, task := phaseOutputFixture(t)
	ctx := context.Background()
	advanceTo(t, wf, task.ID, "research")

	// The agent records its conclusion where the phase declared it.
	current, _ := tasks.Get(ctx, task.ID)
	updated := UpsertSection(current.Description, "Design", "Use CSS custom properties with a data-theme attribute.")
	if _, err := tasks.Update(ctx, task.ID, UpdateTaskInput{Description: &updated}); err != nil {
		t.Fatalf("updating description: %v", err)
	}

	res, err := wf.Advance(ctx, task.ID)
	if err != nil {
		t.Fatalf("advance should succeed: %v", err)
	}
	if res.CurrentPhase != "implementation" {
		t.Errorf("expected implementation, got %s", res.CurrentPhase)
	}
}

func TestAdvanceIgnoresWhitespaceOnlyOutputSection(t *testing.T) {
	tasks, wf, task := phaseOutputFixture(t)
	ctx := context.Background()
	advanceTo(t, wf, task.ID, "research")

	current, _ := tasks.Get(ctx, task.ID)
	blank := current.Description + "\n## Design\n\n   \n"
	if _, err := tasks.Update(ctx, task.ID, UpdateTaskInput{Description: &blank}); err != nil {
		t.Fatalf("updating description: %v", err)
	}

	if _, err := wf.Advance(ctx, task.ID); !errors.Is(err, ErrInvalidInput) {
		t.Errorf("a whitespace-only section is still empty, got %v", err)
	}
}

func TestPhasesWithoutDeclaredOutputAdvanceFreely(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	tasks := NewTaskService(testDB)
	tasks.SetTemplates(NewTemplateService(testDB))
	wf := NewWorkflowService(testDB)
	ctx := context.Background()

	// 'task' declares no phase outputs — nothing should ever block.
	task, err := tasks.Create(ctx, CreateTaskInput{Title: "plain", Type: TypeTask, Source: SourceCLI})
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}
	for {
		res, err := wf.Advance(ctx, task.ID)
		if err != nil {
			t.Fatalf("unexpected block: %v", err)
		}
		if res.IsComplete {
			break
		}
	}
}

func TestWorkflowContextCarriesCurrentOutput(t *testing.T) {
	tasks, wf, task := phaseOutputFixture(t)
	ctx := context.Background()
	advanceTo(t, wf, task.ID, "research")

	current, _ := tasks.Get(ctx, task.ID)
	wc, err := wf.GetContext(ctx, current)
	if err != nil {
		t.Fatalf("getting context: %v", err)
	}
	if wc.CurrentOutput != "Design" {
		t.Errorf("expected CurrentOutput Design, got %q", wc.CurrentOutput)
	}
}

func TestPhaseOutputsSurviveWorkflowUpdate(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	wf := NewWorkflowService(testDB)
	ctx := context.Background()

	def, _ := wf.GetWorkflow(ctx, "feature")
	def.PhaseOutputs["review"] = "Verification"
	if err := wf.UpdateWorkflow(ctx, def); err != nil {
		t.Fatalf("updating workflow: %v", err)
	}

	reloaded, _ := wf.GetWorkflow(ctx, "feature")
	if reloaded.PhaseOutputs["review"] != "Verification" {
		t.Errorf("phase output not persisted: %v", reloaded.PhaseOutputs)
	}
	if reloaded.PhaseOutputs["research"] != "Design" {
		t.Errorf("existing phase output lost: %v", reloaded.PhaseOutputs)
	}
}
