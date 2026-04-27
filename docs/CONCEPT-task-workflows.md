# Concept: Task Workflows with Skill Triggers

> **Status:** Concept (v0.4 candidate)
> **Date:** 2026-04-27

## Problem

Tasks today have a flat lifecycle: `todo → doing → done`. In practice, each task type follows a predictable workflow with distinct phases. AI assistants (Claude) should automatically execute the right skills at each phase transition.

## Core Idea

Each task type defines a **workflow** — a sequence of phases. Each phase transition can trigger **skills** (Claude skills, shell scripts, or MCP tool calls) that run before or after the AI works on that phase.

## Example: Bug Workflow

```
┌─────────┐     ┌────────────┐     ┌──────────┐     ┌──────────┐     ┌──────┐
│ reported │ ──→ │ reproducing│ ──→ │ fixing   │ ──→ │ verifying│ ──→ │ done │
└─────────┘     └────────────┘     └──────────┘     └──────────┘     └──────┘
                  ↓ triggers:        ↓ triggers:       ↓ triggers:
                  - gather-context   - lint             - run-tests
                  - reproduce-bug    - type-check       - regression-test
```

## Example: Feature Workflow

```
┌──────┐     ┌──────────┐     ┌────────────────┐     ┌─────────┐     ┌──────┐
│ todo │ ──→ │ research │ ──→ │ implementation │ ──→ │ review  │ ──→ │ done │
└──────┘     └──────────┘     └────────────────┘     └─────────┘     └──────┘
               ↓ triggers:      ↓ triggers:            ↓ triggers:
               - research        - lint                 - run-tests
               - architecture    - type-check           - code-review
```

## Example: Hotfix Workflow

```
┌──────────┐     ┌─────────┐     ┌──────────┐     ┌──────┐
│ reported │ ──→ │ fixing  │ ──→ │ verifying│ ──→ │ done │
└──────────┘     └─────────┘     └──────────┘     └──────┘
                   ↓ triggers:     ↓ triggers:
                   - root-cause    - run-tests
                   - minimal-fix   - smoke-test
```

## Data Model Extension

### Workflow definition (per-project config)

```json
// .tasks/workflows.json
{
  "bug": {
    "phases": ["reported", "reproducing", "fixing", "verifying", "done"],
    "triggers": {
      "reproducing": {
        "before": ["gather-context"],
        "after": ["reproduce-bug"]
      },
      "fixing": {
        "after": ["lint", "type-check"]
      },
      "verifying": {
        "before": ["run-tests"],
        "after": ["regression-test"]
      }
    }
  },
  "feature": {
    "phases": ["todo", "research", "implementation", "review", "done"],
    "triggers": {
      "research": {
        "before": ["research", "architecture-review"]
      },
      "implementation": {
        "after": ["lint", "type-check"]
      },
      "review": {
        "before": ["run-tests"],
        "after": ["code-review"]
      }
    }
  },
  "task": {
    "phases": ["todo", "doing", "done"],
    "triggers": {}
  },
  "hotfix": {
    "phases": ["reported", "fixing", "verifying", "done"],
    "triggers": {
      "fixing": {
        "before": ["root-cause-analysis"]
      },
      "verifying": {
        "before": ["run-tests", "smoke-test"]
      }
    }
  }
}
```

### Task status change

The current `status` column (`todo`, `doing`, `done`) would be extended to support custom phases per workflow. The `status` field becomes the current phase name.

```sql
-- tasks table gets a phase column (status becomes derived)
-- phase stores the current workflow phase name
-- status is computed: first phase = 'todo', last phase = 'done', others = 'doing'
```

### Skill definition

Skills can be:

1. **Claude skills** — reference a SKILL.md file or inline prompt
2. **Shell commands** — run a script (e.g., `go test ./...`)
3. **MCP tool calls** — invoke another MCP tool

```json
// .tasks/skills.json
{
  "run-tests": {
    "type": "shell",
    "command": "go test ./...",
    "description": "Run the project test suite"
  },
  "lint": {
    "type": "shell",
    "command": "go vet ./...",
    "description": "Run Go linter"
  },
  "code-review": {
    "type": "claude-skill",
    "prompt": "Review the changes made for this task. Check for bugs, security issues, and code quality.",
    "description": "AI-powered code review"
  },
  "gather-context": {
    "type": "claude-skill",
    "prompt": "Analyze the bug report and gather relevant context from the codebase. Identify likely root cause areas.",
    "description": "Gather context for bug investigation"
  },
  "regression-test": {
    "type": "shell",
    "command": "go test -run TestRegression ./...",
    "description": "Run regression tests"
  }
}
```

## MCP Integration

When Claude calls `task_get`, the response includes workflow context:

```json
{
  "id": 42,
  "type": "bug",
  "title": "Login fails on Firefox",
  "phase": "fixing",
  "workflow": {
    "current_phase": "fixing",
    "next_phase": "verifying",
    "phases": ["reported", "reproducing", "fixing", "verifying", "done"],
    "triggers_on_advance": {
      "before": ["run-tests"],
      "after": ["regression-test"]
    }
  }
}
```

New MCP tool:

```typescript
task_advance(id: number): Task
// Moves task to next phase in workflow, triggers associated skills
```

## UI Integration

The task detail page shows:
- Current phase highlighted in a workflow progress bar
- "Advance to next phase" button
- Trigger history (what ran, when, pass/fail)

## Implementation Order

1. **Workflow config file** (`.tasks/workflows.json`) — define phases per type
2. **Phase column in tasks** — migration to add `phase` field
3. **`task_advance` MCP tool** — move to next phase
4. **Shell skill execution** — run commands on phase transition
5. **Claude skill execution** — prompt-based skills
6. **UI workflow progress bar** — visual phase indicator
7. **Trigger history log** — audit trail of skill executions

## Open Questions

- Should workflows be per-project or global?
- Can users define custom task types with custom workflows?
- How to handle skill failures? Block phase advance or just warn?
- Should skills run synchronously or asynchronously?
- How does this interact with `task_complete`? (auto-advance through remaining phases?)
