package core

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/tomas-chudjak/kleidi-task/internal/db"
)

func TestParseSections(t *testing.T) {
	tests := []struct {
		name string
		body string
		want []string
	}{
		{"empty", "", nil},
		{"no headings", "just some prose\nand more", nil},
		{"basic", "## Problem\n\ntext\n\n## Notes\n", []string{"Problem", "Notes"}},
		{"ignores h3", "## Design\n\n### Option A\n\n### Option B\n", []string{"Design"}},
		{"ignores h1", "# Title\n\n## Problem\n", []string{"Problem"}},
		{"trims trailing space", "## Problem   \n", []string{"Problem"}},
		{"dedupes case-insensitively", "## Notes\n\n## notes\n", []string{"Notes"}},
		{"requires space after hashes", "##NoSpace\n\n## Real\n", []string{"Real"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseSections(tt.body)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("section %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestMissingSections(t *testing.T) {
	required := []string{"Problem", "Acceptance criteria"}

	if got := MissingSections("## Problem\n\n## Acceptance criteria\n", required); got != nil {
		t.Errorf("expected none missing, got %v", got)
	}
	if got := MissingSections("## problem\n\n## ACCEPTANCE CRITERIA\n", required); got != nil {
		t.Errorf("expected case-insensitive match, got %v", got)
	}
	got := MissingSections("## Problem\n", required)
	if len(got) != 1 || got[0] != "Acceptance criteria" {
		t.Errorf("expected [Acceptance criteria], got %v", got)
	}
	if got := MissingSections("anything", nil); got != nil {
		t.Errorf("no required sections means nothing missing, got %v", got)
	}
}

// enforcementService builds a TaskService wired with templates and the given
// enforcement mode.
func enforcementService(t *testing.T, mode EnforcementMode) (*TaskService, context.Context) {
	t.Helper()
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	svc.SetTemplates(NewTemplateService(testDB))
	ctx := context.Background()

	if err := NewConfigService(testDB).Set(ctx, "template_enforcement", string(mode)); err != nil {
		t.Fatalf("setting enforcement mode: %v", err)
	}
	return svc, ctx
}

const completeBug = "## Steps to reproduce\n1. x\n\n## Expected behavior\ny\n\n## Actual behavior\nz\n\n## Environment\n- OS: linux\n"

func TestCreateStrictRejectsIncompleteDescription(t *testing.T) {
	svc, ctx := enforcementService(t, EnforcementStrict)

	_, err := svc.Create(ctx, CreateTaskInput{
		Title:       "Login crashes",
		Type:        TypeBug,
		Description: "it broke",
		Source:      SourceMCP,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	// The error is what the agent reads — it must name the missing sections
	// and the tool to call.
	for _, want := range []string{"Steps to reproduce", "Expected behavior", "template_get"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %q", err, want)
		}
	}
}

func TestCreateStrictAcceptsCompleteDescription(t *testing.T) {
	svc, ctx := enforcementService(t, EnforcementStrict)

	task, err := svc.Create(ctx, CreateTaskInput{
		Title:       "Login crashes",
		Type:        TypeBug,
		Description: completeBug,
		Source:      SourceMCP,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if task.Description != completeBug {
		t.Error("description was modified")
	}
}

func TestCreateStrictAllowsInteractiveSources(t *testing.T) {
	// A human at the terminal is not skipping the template flow — they are
	// typing. Strict mode must not make `klt add` unusable.
	for _, src := range []Source{SourceCLI, SourceUI} {
		svc, ctx := enforcementService(t, EnforcementStrict)

		task, err := svc.Create(ctx, CreateTaskInput{
			Title:       "quick note",
			Type:        TypeBug,
			Description: "just a line",
			Source:      src,
		})
		if err != nil {
			t.Fatalf("source %s: unexpected error: %v", src, err)
		}
		if task.Description != "just a line" {
			t.Errorf("source %s: description was modified to %q", src, task.Description)
		}
	}
}

func TestCreateFillsSkeletonForInteractiveSources(t *testing.T) {
	svc, ctx := enforcementService(t, EnforcementStrict)

	task, err := svc.Create(ctx, CreateTaskInput{
		Title:  "Login crashes",
		Type:   TypeBug,
		Source: SourceCLI,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(task.Description, "## Steps to reproduce") {
		t.Errorf("expected template skeleton, got %q", task.Description)
	}
}

func TestCreateDoesNotFillSkeletonForMCP(t *testing.T) {
	// The agent must fill the template itself; silently scaffolding an empty
	// description would let it skip the flow and still pass validation.
	svc, ctx := enforcementService(t, EnforcementStrict)

	_, err := svc.Create(ctx, CreateTaskInput{
		Title:  "Login crashes",
		Type:   TypeBug,
		Source: SourceMCP,
	})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput for empty MCP description, got %v", err)
	}
}

func TestCreateWarnModeCreatesAnyway(t *testing.T) {
	svc, ctx := enforcementService(t, EnforcementWarn)

	task, err := svc.Create(ctx, CreateTaskInput{
		Title:       "Login crashes",
		Type:        TypeBug,
		Description: "it broke",
		Source:      SourceMCP,
	})
	if err != nil {
		t.Fatalf("warn mode must not block: %v", err)
	}
	if task.Description != "it broke" {
		t.Errorf("warn mode must not modify the description, got %q", task.Description)
	}
}

func TestCreateOffModeSkipsValidation(t *testing.T) {
	svc, ctx := enforcementService(t, EnforcementOff)

	if _, err := svc.Create(ctx, CreateTaskInput{
		Title:       "Login crashes",
		Type:        TypeBug,
		Description: "it broke",
		Source:      SourceMCP,
	}); err != nil {
		t.Fatalf("off mode must not block: %v", err)
	}
}

func TestCreateDefaultsToWarn(t *testing.T) {
	// Migration 011 seeds 'warn' — an upgrade must not start rejecting tasks.
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB)
	svc.SetTemplates(NewTemplateService(testDB))

	if mode := NewConfigService(testDB).EnforcementMode(context.Background()); mode != EnforcementWarn {
		t.Errorf("expected default warn, got %q", mode)
	}
	if _, err := svc.Create(context.Background(), CreateTaskInput{
		Title:  "bare task",
		Type:   TypeBug,
		Source: SourceMCP,
	}); err != nil {
		t.Fatalf("default mode must not block: %v", err)
	}
}

func TestCreateTypeWithoutTemplateIsNeverBlocked(t *testing.T) {
	svc, ctx := enforcementService(t, EnforcementStrict)

	if _, err := svc.Create(ctx, CreateTaskInput{
		Title:       "Spike something",
		Type:        TaskType("spike"),
		Description: "no template exists for this type",
		Source:      SourceMCP,
	}); err != nil {
		t.Fatalf("type without a template must not be constrained: %v", err)
	}
}

func TestCreateWithoutTemplateServiceIsNeverBlocked(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTaskService(testDB) // no SetTemplates
	ctx := context.Background()

	if err := NewConfigService(testDB).Set(ctx, "template_enforcement", string(EnforcementStrict)); err != nil {
		t.Fatalf("setting mode: %v", err)
	}
	if _, err := svc.Create(ctx, CreateTaskInput{
		Title:  "bare task",
		Type:   TypeBug,
		Source: SourceMCP,
	}); err != nil {
		t.Fatalf("unconfigured templates must not block: %v", err)
	}
}

func TestParseEnforcementMode(t *testing.T) {
	tests := map[string]EnforcementMode{
		"off":      EnforcementOff,
		"warn":     EnforcementWarn,
		"strict":   EnforcementStrict,
		"":         EnforcementWarn,
		"nonsense": EnforcementWarn,
	}
	for in, want := range tests {
		if got := ParseEnforcementMode(in); got != want {
			t.Errorf("ParseEnforcementMode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestOpenQuestionsSectionSeeded(t *testing.T) {
	// Migration 012 adds the section to the types that carry design decisions.
	testDB := db.NewTestProjectDB(t)
	svc := NewTemplateService(testDB)
	ctx := context.Background()

	for _, typ := range []string{"task", "feature"} {
		tmpl, err := svc.GetByType(ctx, typ)
		if err != nil {
			t.Fatalf("getting %s template: %v", typ, err)
		}
		sections := ParseSections(tmpl.Description)
		found := false
		for _, s := range sections {
			if s == "Open questions" {
				found = true
			}
		}
		if !found {
			t.Errorf("%s template has no Open questions section: %v", typ, sections)
		}
	}

	// Bug and hotfix are diagnostic, not design — they keep their structure.
	for _, typ := range []string{"bug", "hotfix"} {
		tmpl, err := svc.GetByType(ctx, typ)
		if err != nil {
			t.Fatalf("getting %s template: %v", typ, err)
		}
		if strings.Contains(tmpl.Description, "Open questions") {
			t.Errorf("%s template should not have Open questions", typ)
		}
	}
}

func TestAgentRulesSeededAndSeparate(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	tplSvc := NewTemplateService(testDB)
	taskSvc := NewTaskService(testDB)
	taskSvc.SetTemplates(tplSvc)
	ctx := context.Background()

	tmpl, err := tplSvc.GetByType(ctx, "feature")
	if err != nil {
		t.Fatalf("getting feature template: %v", err)
	}
	if tmpl.AgentRules == "" {
		t.Fatal("expected seeded agent rules")
	}
	if strings.Contains(tmpl.Description, tmpl.AgentRules) {
		t.Error("agent rules must not be part of the description skeleton")
	}

	// A scaffolded description must carry the skeleton, never the rules.
	task, err := taskSvc.Create(ctx, CreateTaskInput{
		Title:  "scaffolded",
		Type:   TypeFeature,
		Source: SourceCLI,
	})
	if err != nil {
		t.Fatalf("creating task: %v", err)
	}
	if strings.Contains(task.Description, "Final designs only") {
		t.Error("agent rules leaked into the task description")
	}
	if !strings.Contains(task.Description, "## Open questions") {
		t.Error("scaffolded description is missing Open questions")
	}
}

func TestUpdateTemplatePreservesAgentRules(t *testing.T) {
	testDB := db.NewTestProjectDB(t)
	svc := NewTemplateService(testDB)
	ctx := context.Background()

	tmpl, err := svc.GetByType(ctx, "feature")
	if err != nil {
		t.Fatalf("getting template: %v", err)
	}
	rules := tmpl.AgentRules

	// The rules-unaware Update must not silently wipe them.
	updated, err := svc.Update(ctx, tmpl.ID, tmpl.Name, tmpl.Type, tmpl.Priority, "## Changed\n")
	if err != nil {
		t.Fatalf("updating template: %v", err)
	}
	if updated.AgentRules != rules {
		t.Errorf("agent rules lost on update: got %q", updated.AgentRules)
	}
}
